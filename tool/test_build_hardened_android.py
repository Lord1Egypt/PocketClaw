#!/usr/bin/env python3
"""Focused tests for the H3A hardened Android build contract."""

import importlib.util
import json
import os
import sys
import base64
import tempfile
import unittest
import zipfile
from pathlib import Path

import r8_contract


spec = importlib.util.spec_from_file_location(
    "build_hardened_android", Path(__file__).resolve().parent / "build_hardened_android.py"
)
hardening = importlib.util.module_from_spec(spec)
sys.modules["build_hardened_android"] = hardening
spec.loader.exec_module(hardening)


class GeneratedSourcePackageTest(unittest.TestCase):
    def write_config(self, directory: Path, extra=None) -> Path:
        path = directory / "package_config.json"
        packages = [
            {
                "name": "pocketclaw",
                "rootUri": "../",
                "packageUri": "lib/",
                "languageVersion": "3.11",
            }
        ]
        if extra:
            packages.extend(extra)
        path.write_text(json.dumps({"configVersion": 2, "packages": packages}), encoding="utf-8")
        return path

    def test_adds_exact_stable_mapping_and_is_idempotent(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = self.write_config(Path(tmp))
            self.assertTrue(hardening.prepare_generated_source_package(path))
            self.assertFalse(hardening.prepare_generated_source_package(path))
            mappings = [
                item
                for item in json.loads(path.read_text(encoding="utf-8"))["packages"]
                if item["name"] == "pocketclaw_generated"
            ]
            self.assertEqual(
                mappings,
                [{
                    "name": "pocketclaw_generated",
                    "rootUri": "flutter_build/",
                    "packageUri": "./",
                    "languageVersion": "3.11",
                }],
            )

    def test_refuses_a_conflicting_generated_mapping(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = self.write_config(
                Path(tmp),
                [{
                    "name": "pocketclaw_generated",
                    "rootUri": "file:///developer/checkout/",
                    "packageUri": "./",
                    "languageVersion": "3.11",
                }],
            )
            with self.assertRaises(hardening.HardeningError):
                hardening.prepare_generated_source_package(path)


class SigningModeTest(unittest.TestCase):
    def test_local_test_explicitly_selects_debug_signing(self):
        command = hardening.gradle_command("local-test", "build/private-symbols/dart/android-arm64")
        self.assertIn("-PallowDebugSigning=true", command)
        self.assertIn("-Pdart-obfuscation=true", command)
        self.assertIn("-PpocketclawDartHardening=true", command)

    def test_production_command_never_selects_debug_signing(self):
        command = hardening.gradle_command("production", "build/private-symbols/dart/android-arm64")
        self.assertNotIn("-PallowDebugSigning=true", command)

    def test_local_test_refuses_accidentally_declared_production_material(self):
        with self.assertRaises(hardening.HardeningError):
            hardening.validate_signing_environment("local-test", {"KEYSTORE_PATH": "/outside"})

    def test_production_requires_all_variable_names_without_reading_values(self):
        with self.assertRaises(hardening.HardeningError):
            hardening.validate_signing_environment("production", {"KEYSTORE_PATH": "/outside"})


class GeneratedOutputResetTest(unittest.TestCase):
    def test_clears_cached_aot_and_stale_outputs_but_preserves_package_config(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            flutter_build = root / ".dart_tool/flutter_build"
            cached_aot = flutter_build / "cache-key/arm64-v8a/app.so"
            cached_aot.parent.mkdir(parents=True)
            cached_aot.write_bytes(b"cached aot")
            package_config = root / ".dart_tool/package_config.json"
            package_config.write_text("{}", encoding="utf-8")
            apk = root / "build/app-release.apk"
            symbols = root / "build/private-symbols/app.android-arm64.symbols"
            mapping = root / "build/app/outputs/mapping/release/mapping.txt"
            apk.parent.mkdir(parents=True)
            symbols.parent.mkdir(parents=True)
            mapping.parent.mkdir(parents=True)
            apk.write_bytes(b"old apk")
            symbols.write_bytes(b"old symbols")
            mapping.write_bytes(b"old mapping")
            mapping.with_name("usage.txt").write_bytes(b"old usage")

            hardening.reset_generated_build_outputs(apk, symbols, flutter_build, mapping)

            self.assertFalse(flutter_build.exists())
            self.assertFalse(apk.exists())
            self.assertFalse(symbols.exists())
            self.assertFalse(mapping.exists())
            self.assertFalse(mapping.with_name("usage.txt").exists())
            self.assertTrue(package_config.is_file())

    def test_refuses_a_symlinked_flutter_build_cache(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            target = root / "outside"
            target.mkdir()
            flutter_build = root / "flutter_build"
            flutter_build.symlink_to(target, target_is_directory=True)
            with self.assertRaises(hardening.HardeningError):
                hardening.reset_generated_build_outputs(
                    root / "app.apk", root / "app.symbols", flutter_build
                )
            self.assertTrue(target.is_dir())

    def test_relative_symbol_contract_does_not_depend_on_callers_cwd(self):
        original = Path.cwd()
        try:
            with tempfile.TemporaryDirectory() as tmp:
                os.chdir(tmp)
                property_value, resolved = hardening.resolve_symbols_dir(
                    "build/private-symbols/dart/android-arm64"
                )
        finally:
            os.chdir(original)
        self.assertEqual(property_value, "build/private-symbols/dart/android-arm64")
        self.assertEqual(
            resolved,
            (hardening.REPO / "build/private-symbols/dart/android-arm64").resolve(),
        )


class OutputInspectionTest(unittest.TestCase):
    def make_outputs(self, directory: Path, app: bytes | None = None, packaged_symbol=False):
        apk = directory / "app-release.apk"
        symbols = directory / "app.android-arm64.symbols"
        symbol_bytes = (
            b"\x7fELF.debug_info.debug_line"
            + b"\0".join(hardening.APP_SYMBOL_MARKERS)
        )
        symbols.write_bytes(symbol_bytes)
        app_bytes = app if app is not None else b"\x7fELF" + hardening.GENERATED_REGISTRANT_URI
        with zipfile.ZipFile(apk, "w") as archive:
            archive.writestr("lib/arm64-v8a/libapp.so", app_bytes)
            archive.writestr("classes.dex", b"dex\n035\0obfuscated")
            if packaged_symbol:
                archive.writestr("assets/private-symbols/app.android-arm64.symbols", symbol_bytes)
        mapping = directory / "mapping.txt"
        lines = [f"{name} -> {name}:" for name in r8_contract.REQUIRED_COMPONENTS]
        lines += [
            f"{name} -> h3.{index}:"
            for index, name in enumerate(r8_contract.INTERNAL_CLASSES)
        ]
        mapping.write_text("\n".join(lines) + "\n", encoding="utf-8")
        mapping.with_name("usage.txt").write_text("removed.Class\n", encoding="utf-8")
        return apk, symbols, mapping

    def test_accepts_obfuscated_aot_with_external_split_info(self):
        with tempfile.TemporaryDirectory() as tmp:
            apk, symbols, mapping = self.make_outputs(Path(tmp))
            evidence = hardening.inspect_hardened_outputs(apk, symbols, r8_mapping=mapping)
            self.assertEqual(
                evidence["generatedSourceUri"],
                "package:pocketclaw_generated/dart_plugin_registrant.dart",
            )
            self.assertEqual(evidence["classification"], "LOCAL TEST / NON-RELEASABLE")

    def test_refuses_host_path_or_unobfuscated_application_name(self):
        with tempfile.TemporaryDirectory() as tmp:
            bad = (
                b"file:///home/person/project/dart_plugin_registrant.dart"
                + hardening.GENERATED_REGISTRANT_URI
                + hardening.APP_SYMBOL_MARKERS[0]
            )
            apk, symbols, mapping = self.make_outputs(Path(tmp), app=bad)
            with self.assertRaises(hardening.HardeningError):
                hardening.inspect_hardened_outputs(apk, symbols, r8_mapping=mapping)

    def test_refuses_private_symbols_packaged_in_apk(self):
        with tempfile.TemporaryDirectory() as tmp:
            apk, symbols, mapping = self.make_outputs(Path(tmp), packaged_symbol=True)
            with self.assertRaises(hardening.HardeningError):
                hardening.inspect_hardened_outputs(apk, symbols, r8_mapping=mapping)


OFFICIAL_URL = "https://pocketclaw-telegram-setup-bot-83ai.vercel.app"


class OfficialOnboardingContract(unittest.TestCase):
    """The build must carry the official endpoint without being reminded to.

    vc51 through vc62 shipped without managed Telegram onboarding because the
    define lived in a hand-typed command. Every assertion here exists because
    that failure produced no error, no red test and no changed gate.
    """

    def write_properties(self, directory: Path, value: str | None) -> Path:
        path = directory / "official-onboarding.properties"
        body = "# tracked official endpoint\n"
        if value is not None:
            body += f"{hardening.ONBOARDING_PROPERTY}={value}\n"
        path.write_text(body, encoding="utf-8")
        return path

    def test_tracked_official_url_is_https_and_parses(self):
        url = hardening.read_official_onboarding_base_url()
        self.assertEqual(url, OFFICIAL_URL)
        self.assertTrue(url.startswith("https://"))

    def test_reads_the_value_from_a_properties_file(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = self.write_properties(Path(tmp), OFFICIAL_URL)
            self.assertEqual(hardening.read_official_onboarding_base_url(path), OFFICIAL_URL)

    def test_missing_file_or_missing_key_fails_closed(self):
        with tempfile.TemporaryDirectory() as tmp:
            absent = Path(tmp) / "nope.properties"
            with self.assertRaises(hardening.HardeningError):
                hardening.read_official_onboarding_base_url(absent)
            empty = self.write_properties(Path(tmp), None)
            with self.assertRaises(hardening.HardeningError):
                hardening.read_official_onboarding_base_url(empty)

    def test_refuses_plain_http_and_hostless_and_credentialed_urls(self):
        for bad in (
            "http://pocketclaw-telegram-setup-bot-83ai.vercel.app",
            "https://",
            "ftp://example.test",
            "not-a-url",
            "https://user:secret@example.test",
        ):
            with self.subTest(url=bad):
                with self.assertRaises(hardening.HardeningError):
                    hardening.validate_onboarding_base_url(bad)

    def test_gradle_command_carries_the_define_as_base64(self):
        command = hardening.gradle_command("production", "sym", "apk", OFFICIAL_URL)
        defines = [arg for arg in command if arg.startswith("-Pdart-defines=")]
        self.assertEqual(len(defines), 1)
        payload = defines[0].split("=", 1)[1]
        decoded = base64.b64decode(payload).decode()
        self.assertEqual(decoded, f"{hardening.ONBOARDING_DEFINE}={OFFICIAL_URL}")

    def test_gradle_command_omits_the_property_when_no_url(self):
        command = hardening.gradle_command("local-test", "sym", "apk", None)
        self.assertFalse([arg for arg in command if arg.startswith("-Pdart-defines=")])

    def test_production_resolves_the_official_url_without_a_flag(self):
        self.assertEqual(
            hardening.resolve_onboarding_base_url("production", "official", None),
            OFFICIAL_URL,
        )

    def test_production_cannot_omit_managed_onboarding(self):
        with self.assertRaises(hardening.HardeningError):
            hardening.resolve_onboarding_base_url("production", "omit", None)

    def test_local_test_may_omit_and_downstream_may_override(self):
        self.assertIsNone(hardening.resolve_onboarding_base_url("local-test", "omit", None))
        self.assertEqual(
            hardening.resolve_onboarding_base_url("local-test", "official", "https://fork.test"),
            "https://fork.test",
        )

    def test_override_must_still_be_https(self):
        with self.assertRaises(hardening.HardeningError):
            hardening.resolve_onboarding_base_url("production", "official", "http://fork.test")

    def test_finished_apk_without_the_url_is_refused(self):
        """The build command is a claim; libapp.so is the evidence."""
        with tempfile.TemporaryDirectory() as tmp:
            apk, symbols, mapping = self.make_apk(Path(tmp), carries=False)
            with self.assertRaises(hardening.HardeningError):
                hardening.inspect_hardened_outputs(
                    apk, symbols, r8_mapping=mapping, onboarding_base_url=OFFICIAL_URL)

    def test_finished_apk_with_the_url_is_accepted_and_recorded(self):
        with tempfile.TemporaryDirectory() as tmp:
            apk, symbols, mapping = self.make_apk(Path(tmp), carries=True)
            evidence = hardening.inspect_hardened_outputs(
                apk, symbols, r8_mapping=mapping, onboarding_base_url=OFFICIAL_URL)
            self.assertEqual(evidence["onboardingBaseUrl"], OFFICIAL_URL)

    def test_stale_artifact_is_caught_when_the_build_omits_the_url(self):
        with tempfile.TemporaryDirectory() as tmp:
            apk, symbols, mapping = self.make_apk(Path(tmp), carries=True)
            with self.assertRaises(hardening.HardeningError):
                hardening.inspect_hardened_outputs(
                    apk, symbols, r8_mapping=mapping, onboarding_base_url=None)

    def make_apk(self, directory: Path, carries: bool):
        app = b"\x7fELF" + hardening.GENERATED_REGISTRANT_URI
        if carries:
            app += OFFICIAL_URL.encode()
        return OutputInspectionTest.make_outputs(self, directory, app=app)



if __name__ == "__main__":
    unittest.main(verbosity=2)
