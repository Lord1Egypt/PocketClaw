#!/usr/bin/env python3
"""PC-DEF-021 contract tests: artifact purpose decides what its contents mean.

Every case builds a real zip with the structure it is describing, so the rules
are exercised against archives rather than against a mock of the rules. The
fixtures are small — a bundle here is a handful of entries with the right names,
which is exactly what the policy reads.
"""

from __future__ import annotations

import shutil
import subprocess
import sys
import tempfile
import unittest
import zipfile
from pathlib import Path

TOOL = Path(__file__).resolve().parent
sys.path.insert(0, str(TOOL))

from artifact_policy import (  # noqa: E402
    AAB,
    APK,
    NON_PUBLISH_AUDIT,
    PLAY_UPLOAD,
    PUBLIC_RELEASE,
    ArtifactPolicyError,
    categorize_bundle_metadata,
    detect_artifact_kind,
    distribution_verdict,
    inspect_bundle,
    packaged_r8_mapping_entries,
    private_material_violations,
    public_release_asset_violations,
)

REPO = TOOL.parent
GATE = TOOL / "release_gate.py"

OBFUSCATION = "BUNDLE-METADATA/com.android.tools.build.obfuscation/proguard.map"
DEBUGSYMS = "BUNDLE-METADATA/com.android.tools.build.debugsymbols/arm64-v8a/libapp.so.sym"


def write_zip(path: Path, entries: dict[str, bytes]) -> Path:
    with zipfile.ZipFile(path, "w") as archive:
        for name, data in entries.items():
            archive.writestr(name, data)
    return path


def apk_entries(extra: dict[str, bytes] | None = None) -> dict[str, bytes]:
    entries = {
        "AndroidManifest.xml": b"\x03\x00\x08\x00manifest",
        "classes.dex": b"dex\n035\x00",
        "lib/arm64-v8a/libapp.so": b"\x7fELF" + b"aot",
        "resources.arsc": b"arsc",
        "META-INF/CERT.RSA": b"sig",
    }
    entries.update(extra or {})
    return entries


def bundle_entries(extra: dict[str, bytes] | None = None,
                   metadata: bool = True) -> dict[str, bytes]:
    entries = {
        "BundleConfig.pb": b"\x08\x01",
        "base/manifest/AndroidManifest.xml": b"\x03\x00\x08\x00manifest",
        "base/dex/classes.dex": b"dex\n035\x00",
        "base/lib/arm64-v8a/libapp.so": b"\x7fELF" + b"aot",
        "base/lib/arm64-v8a/libpocketclaw.so": b"\x7fELF" + b"core",
        "base/resources.pb": b"res",
        "META-INF/MANIFEST.MF": b"Manifest-Version: 1.0\n",
    }
    if metadata:
        entries[OBFUSCATION] = b"com.example.Foo -> a:\n"
        entries[DEBUGSYMS] = b"\x7fELF" + b"sym"
        entries["BUNDLE-METADATA/com.android.tools.build.profiles/baseline.prof"] = b"prof"
    entries.update(extra or {})
    return entries


class TempCase(unittest.TestCase):
    def setUp(self) -> None:
        self.tmp = Path(tempfile.mkdtemp(prefix="pocketclaw-policy-"))
        self.addCleanup(shutil.rmtree, self.tmp, ignore_errors=True)


class KindDetection(TempCase):
    def test_apk_detected_from_contents(self):
        path = write_zip(self.tmp / "app.apk", apk_entries())
        self.assertEqual(detect_artifact_kind(path), APK)

    def test_bundle_detected_from_contents(self):
        path = write_zip(self.tmp / "app.aab", bundle_entries())
        self.assertEqual(detect_artifact_kind(path), AAB)

    def test_a_bundle_renamed_to_apk_is_still_a_bundle(self):
        """Renaming must not launder a bundle into a publishable APK."""
        path = write_zip(self.tmp / "sneaky.apk", bundle_entries())
        self.assertEqual(detect_artifact_kind(path), AAB)

    def test_non_android_zip_is_refused(self):
        path = write_zip(self.tmp / "notes.zip", {"README.md": b"hi"})
        with self.assertRaises(ArtifactPolicyError):
            detect_artifact_kind(path)


class DistributionClassification(TempCase):
    def test_case_3_aab_classified_public_fails(self):
        ok, message = distribution_verdict(AAB, PUBLIC_RELEASE)
        self.assertFalse(ok)
        self.assertIn("forbidden as a public release asset", message)

    def test_case_3_aab_with_no_mapping_is_still_forbidden_publicly(self):
        """The prohibition is about purpose, not about this file's contents."""
        path = write_zip(self.tmp / "clean.aab", bundle_entries(metadata=False))
        inventory = inspect_bundle(path)
        self.assertEqual(inventory["bundleMetadata"], [])
        ok, message = distribution_verdict(detect_artifact_kind(path), PUBLIC_RELEASE)
        self.assertFalse(ok)
        self.assertIn("forbidden as a public release asset", message)

    def test_case_4_aab_classified_play_upload_passes(self):
        ok, message = distribution_verdict(AAB, PLAY_UPLOAD)
        self.assertTrue(ok)
        self.assertIn("Google Play", message)

    def test_case_5_aab_classified_non_publish_audit_passes(self):
        ok, message = distribution_verdict(AAB, NON_PUBLISH_AUDIT)
        self.assertTrue(ok)
        self.assertIn("not published", message)

    def test_case_6_missing_classification_fails_closed(self):
        for missing in ("", None, "public", "whatever"):
            ok, message = distribution_verdict(AAB, missing or "")
            self.assertFalse(ok, f"{missing!r} must not be accepted")
            self.assertIn("missing or unrecognised", message)

    def test_apk_may_be_public(self):
        ok, _ = distribution_verdict(APK, PUBLIC_RELEASE)
        self.assertTrue(ok)

    def test_apk_missing_classification_also_fails_closed(self):
        ok, _ = distribution_verdict(APK, "")
        self.assertFalse(ok)


class BundleMetadata(TempCase):
    def test_case_7_expected_proguard_map_is_allowed_for_play(self):
        category, allowed = categorize_bundle_metadata(OBFUSCATION)
        self.assertEqual(category, "r8 deobfuscation mapping")
        self.assertTrue(allowed)

    def test_case_8_expected_native_debug_metadata_is_allowed_for_play(self):
        category, allowed = categorize_bundle_metadata(DEBUGSYMS)
        self.assertEqual(category, "native debug symbols")
        self.assertTrue(allowed)

    def test_expected_metadata_is_not_reported_as_leakage(self):
        path = write_zip(self.tmp / "play.aab", bundle_entries())
        leaks = private_material_violations(inspect_bundle(path)["entryNames"], kind=AAB)
        self.assertEqual(leaks, [], f"expected AGP metadata was flagged: {leaks}")

    def test_inventory_reports_modules_abis_and_metadata(self):
        path = write_zip(self.tmp / "play.aab", bundle_entries())
        inventory = inspect_bundle(path)
        self.assertEqual(inventory["modules"], ["base"])
        self.assertEqual(inventory["abis"], ["arm64-v8a"])
        categories = {item["category"] for item in inventory["bundleMetadata"]}
        self.assertIn("r8 deobfuscation mapping", categories)
        self.assertIn("native debug symbols", categories)
        self.assertTrue(inventory["bundleMetadataBytes"] > 0)
        self.assertEqual(len(inventory["nativeEntries"]), 2)

    def test_unrecognised_bundle_metadata_is_not_play_allowed(self):
        category, allowed = categorize_bundle_metadata(
            "BUNDLE-METADATA/com.example.whatever/blob.bin")
        self.assertEqual(category, "unrecognised bundle metadata")
        self.assertFalse(allowed)

    def test_case_9_unrelated_private_artifact_in_a_play_bundle_fails(self):
        for entry, reason in (
            ("base/assets/flutter_assets/release.jks", "signing material"),
            ("base/assets/.env", "environment file"),
            ("BUNDLE-METADATA/com.android.tools.build.obfuscation/release.jks",
             "signing material"),
            ("base/root/private-symbols/native/libpocketclaw.so.debug",
             "private native support archive"),
            ("base/assets/sign-release.sh", "signing helper"),
        ):
            with self.subTest(entry=entry):
                path = write_zip(self.tmp / "leaky.aab", bundle_entries({entry: b"x"}))
                leaks = private_material_violations(
                    inspect_bundle(path)["entryNames"], kind=AAB)
                self.assertTrue(leaks, f"{entry} was not flagged")
                self.assertIn(entry, [name for name, _ in leaks])

    def test_a_sym_file_outside_the_metadata_directory_is_still_private(self):
        path = write_zip(self.tmp / "leaky.aab",
                         bundle_entries({"base/assets/libapp.so.dwarf": b"x"}))
        leaks = private_material_violations(inspect_bundle(path)["entryNames"], kind=AAB)
        self.assertIn("base/assets/libapp.so.dwarf", [name for name, _ in leaks])


class R8MappingEvidence(TempCase):
    def test_case_1_clean_apk_has_no_packaged_mapping(self):
        path = write_zip(self.tmp / "clean.apk", apk_entries())
        with zipfile.ZipFile(path) as archive:
            self.assertEqual(packaged_r8_mapping_entries(archive.namelist()), [])

    def test_case_2_apk_containing_mapping_is_detected(self):
        for entry in ("mapping.txt", "assets/mapping.txt",
                      "assets/proguard.map", "root/private-mapping/mapping.txt"):
            with self.subTest(entry=entry):
                path = write_zip(self.tmp / "leaky.apk", apk_entries({entry: b"a -> b:\n"}))
                with zipfile.ZipFile(path) as archive:
                    found = packaged_r8_mapping_entries(archive.namelist())
                self.assertIn(entry, found)

    def test_case_12_result_is_derived_from_contents_not_a_constant(self):
        """The same code must answer differently for two different archives."""
        clean = write_zip(self.tmp / "a.apk", apk_entries())
        leaky = write_zip(self.tmp / "b.apk", apk_entries({"mapping.txt": b"a -> b:\n"}))
        with zipfile.ZipFile(clean) as archive:
            clean_result = packaged_r8_mapping_entries(archive.namelist())
        with zipfile.ZipFile(leaky) as archive:
            leaky_result = packaged_r8_mapping_entries(archive.namelist())
        self.assertEqual(clean_result, [])
        self.assertEqual(leaky_result, ["mapping.txt"])
        self.assertNotEqual(clean_result, leaky_result)

    def test_the_gate_reports_real_evidence_not_a_fixed_string(self):
        """artifact.r8_mapping_private must cite what it scanned."""
        source = (TOOL / "release_gate.py").read_text(encoding="utf-8")
        marker = 'gate.check(\n        "artifact.r8_mapping_private"'
        self.assertIn("packaged_r8_mapping_entries", source)
        self.assertNotIn('gate.check("artifact.r8_mapping_private", True,', source,
                         "the vacuous hardcoded check is back")
        self.assertIn("archive entries scanned", source)
        del marker


class PublicReleaseAssetAllowlist(unittest.TestCase):
    def test_case_10_allowlist_rejects_aab(self):
        violations = public_release_asset_violations(["PocketClaw-v0.2.0-rc1.aab"])
        self.assertEqual(len(violations), 1)
        self.assertEqual(violations[0][1], "android app bundle")

    def test_case_11_allowlist_rejects_private_release_support(self):
        forbidden = {
            "mapping.txt": "r8/proguard mapping",
            "usage.txt": "r8/proguard mapping",
            "proguard.map": "r8/proguard mapping",
            "libpocketclaw.so.debug": "native debug companion",
            "libapp.so.sym": "native debug companion",
            "app.android-arm64.symbols": "dart split debug info",
            "native-symbols.zip": "symbol archive",
            "private-symbols.tar.gz": "private support archive",
            "pocketclaw-release.jks": "signing material",
            "upload.p12": "signing material",
            ".env": "environment file",
        }
        violations = dict(
            (name, reason) for name, reason in
            public_release_asset_violations(list(forbidden))
        )
        for name, expected_reason in forbidden.items():
            with self.subTest(asset=name):
                self.assertIn(name, violations, f"{name} was allowed")
                self.assertEqual(violations[name], expected_reason)

    def test_ordinary_public_assets_remain_allowed(self):
        allowed = [
            "PocketClaw-v0.2.0-arm64.apk",
            "SHA256SUMS.txt",
            "THIRD_PARTY_NOTICES.md",
            "CHANGELOG.md",
            "pocketclaw-0.2.0-source.tar.gz",
            "LICENSE",
        ]
        self.assertEqual(public_release_asset_violations(allowed), [])

    def test_a_path_is_judged_by_its_basename(self):
        violations = public_release_asset_violations(
            ["dist/public/PocketClaw-v0.2.0.aab"])
        self.assertEqual(len(violations), 1)


class GateIntegration(TempCase):
    """The rules as the gate actually applies them, through its real CLI."""

    def run_gate(self, *args: str) -> subprocess.CompletedProcess:
        return subprocess.run(
            [sys.executable, str(GATE), *args],
            cwd=REPO, capture_output=True, text=True, check=False)

    def test_bundle_classified_public_fails_the_gate(self):
        path = write_zip(self.tmp / "app.aab", bundle_entries())
        result = self.run_gate("--verify-bundle", str(path),
                               "--artifact-class", PUBLIC_RELEASE)
        self.assertEqual(result.returncode, 1, result.stdout)
        self.assertIn("artifact.distribution_class", result.stdout)
        self.assertIn("forbidden as a public release asset", result.stdout)

    def test_bundle_classified_play_upload_passes_the_gate(self):
        path = write_zip(self.tmp / "app.aab", bundle_entries())
        result = self.run_gate("--verify-bundle", str(path),
                               "--artifact-class", PLAY_UPLOAD)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("bundle.play_metadata_expected", result.stdout)
        self.assertIn("allowed and expected for a Play upload", result.stdout)

    def test_bundle_classified_non_publish_audit_passes_with_a_notice(self):
        path = write_zip(self.tmp / "app.aab", bundle_entries())
        result = self.run_gate("--verify-bundle", str(path),
                               "--artifact-class", NON_PUBLISH_AUDIT)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("NOT PUBLIC-RELEASE-SAFE", result.stdout)

    def test_bundle_without_a_classification_is_refused(self):
        path = write_zip(self.tmp / "app.aab", bundle_entries())
        result = self.run_gate("--verify-bundle", str(path))
        self.assertEqual(result.returncode, 2)
        self.assertIn("fails closed", result.stderr)

    def test_play_bundle_with_unrelated_private_material_fails(self):
        path = write_zip(self.tmp / "leaky.aab",
                         bundle_entries({"base/assets/release.jks": b"x"}))
        result = self.run_gate("--verify-bundle", str(path),
                               "--artifact-class", PLAY_UPLOAD)
        self.assertEqual(result.returncode, 1, result.stdout)
        self.assertIn("bundle.no_unrelated_private_material", result.stdout)

    def test_release_asset_allowlist_through_the_gate(self):
        bad = self.run_gate("--release-assets", "PocketClaw.apk", "PocketClaw.aab")
        self.assertEqual(bad.returncode, 1, bad.stdout)
        self.assertIn("release.public_asset_allowlist", bad.stdout)

        good = self.run_gate("--release-assets", "PocketClaw.apk", "SHA256SUMS.txt")
        self.assertEqual(good.returncode, 0, good.stdout + good.stderr)


if __name__ == "__main__":
    unittest.main(verbosity=2)
