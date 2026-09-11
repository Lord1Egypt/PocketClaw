#!/usr/bin/env python3
"""Focused tests for the release gate's decision logic.

Only the parts that decide releasable-or-not are exercised here: the clean
worktree rule, the embedded BuildTime comparison, and the `dev` marker. The
artifact inspection is proven by running the gate against a real APK; what
needs unit coverage is the branching, because a wrong branch silently blesses an
artifact that should have been refused.
"""

import importlib.util
import sys
import tempfile
import unittest
import zipfile
from pathlib import Path, PurePosixPath

import r8_contract

spec = importlib.util.spec_from_file_location(
    "release_gate", Path(__file__).resolve().parent / "release_gate.py")
gate_module = importlib.util.module_from_spec(spec)
# Registered before execution: the module defines dataclasses, and
# dataclasses resolves field types through sys.modules at class-creation time.
sys.modules["release_gate"] = gate_module
spec.loader.exec_module(gate_module)

Gate = gate_module.Gate
PASS, FAIL, SKIP = gate_module.PASS, gate_module.FAIL, gate_module.SKIP

# A Core binary carries its BuildTime as a plain linker-written string.
STAMPED = b"\x00padding\x00" + b"2026-09-08T04:02:42+0000" + b"\x00more\x00"
DEV_ONLY = b"\x00padding\x00" + b"dev" + b"\x00more\x00"


def status_of(gate, name):
    return next(r.status for r in gate.results if r.name == name)


def detail_of(gate, name):
    result = next(r for r in gate.results if r.name == name)
    return f"{result.detail} {result.expected} {result.observed}"


class EmbeddedBuildTime(unittest.TestCase):
    def test_matching_stamp_passes(self):
        gate = Gate()
        gate.facts["buildTimeExpected"] = "2026-09-08T04:02:42+0000"
        gate_module.build_time_gate(gate, STAMPED, "production")
        self.assertEqual(status_of(gate, "artifact.build_time"), PASS)
        self.assertEqual(gate.facts["buildTimeObserved"], "2026-09-08T04:02:42+0000")

    def test_mismatch_fails_production_with_both_values(self):
        gate = Gate()
        gate.facts["buildTimeExpected"] = "2020-01-01T00:00:00+0000"
        gate_module.build_time_gate(gate, STAMPED, "production")
        self.assertEqual(status_of(gate, "artifact.build_time"), FAIL)
        detail = detail_of(gate, "artifact.build_time")
        self.assertIn("2020-01-01T00:00:00+0000", detail)
        self.assertIn("2026-09-08T04:02:42+0000", detail)

    def test_mismatch_is_legacy_and_non_releasable_for_a_test_artifact(self):
        gate = Gate()
        gate.facts["buildTimeExpected"] = "2020-01-01T00:00:00+0000"
        gate_module.build_time_gate(gate, STAMPED, "test")
        self.assertEqual(status_of(gate, "artifact.build_time"), SKIP)
        self.assertIn("LEGACY", detail_of(gate, "artifact.build_time"))
        self.assertFalse(gate.facts["releasable"])

    def test_dev_marker_fails_production(self):
        # The Makefile's fallback for a build whose timestamp could not be
        # derived. It may exist for developer builds; it may never ship.
        gate = Gate()
        gate.facts["buildTimeExpected"] = "2026-09-08T04:02:42+0000"
        gate_module.build_time_gate(gate, DEV_ONLY, "production")
        self.assertEqual(status_of(gate, "artifact.build_time"), FAIL)
        self.assertIn("dev", detail_of(gate, "artifact.build_time"))

    def test_dev_marker_is_non_releasable_for_a_test_artifact(self):
        gate = Gate()
        gate.facts["buildTimeExpected"] = "2026-09-08T04:02:42+0000"
        gate_module.build_time_gate(gate, DEV_ONLY, "test")
        self.assertEqual(status_of(gate, "artifact.build_time"), SKIP)
        self.assertFalse(gate.facts["releasable"])

    def test_unreadable_stamp_fails_production(self):
        gate = Gate()
        gate.facts["buildTimeExpected"] = "2026-09-08T04:02:42+0000"
        gate_module.build_time_gate(gate, b"\x00\x01\x02nothing here", "production")
        self.assertEqual(status_of(gate, "artifact.build_time"), FAIL)

    def test_production_refuses_when_the_expected_value_is_unknown(self):
        # Never pass by default: an expected value we could not compute is not
        # evidence that the observed one is right.
        gate = Gate()
        gate_module.build_time_gate(gate, STAMPED, "production")
        self.assertEqual(status_of(gate, "artifact.build_time"), FAIL)


class CleanWorktree(unittest.TestCase):
    def setUp(self):
        self._run = gate_module.run

    def tearDown(self):
        gate_module.run = self._run

    def _with_status(self, porcelain, rc=0):
        gate_module.run = lambda cmd, cwd=None, env=None: (rc, porcelain)

    def test_clean_passes(self):
        self._with_status("")
        gate = Gate()
        gate_module.worktree_gate(gate, "production")
        self.assertEqual(status_of(gate, "repo.clean_worktree"), PASS)

    def test_dirty_fails_production(self):
        self._with_status(" M lib/main.dart\n?? stray.txt\n")
        gate = Gate()
        gate_module.worktree_gate(gate, "production")
        self.assertEqual(status_of(gate, "repo.clean_worktree"), FAIL)
        self.assertIn("lib/main.dart", detail_of(gate, "repo.clean_worktree"))

    def test_dirty_is_non_releasable_for_a_test_build(self):
        self._with_status(" M lib/main.dart\n")
        gate = Gate()
        gate_module.worktree_gate(gate, "test")
        self.assertEqual(status_of(gate, "repo.clean_worktree"), SKIP)
        self.assertFalse(gate.facts["releasable"])

    def test_non_git_checkout_fails_production(self):
        # Provenance is not optional for a release: outside a worktree there is
        # no revision, no cleanliness and no build-input commit, so there is no
        # way to say what a canonical build would have produced.
        self._with_status("fatal: not a git repository", rc=128)
        gate = Gate()
        gate_module.worktree_gate(gate, "production")
        self.assertEqual(status_of(gate, "repo.clean_worktree"), FAIL)
        self.assertIn("provenance", detail_of(gate, "repo.clean_worktree"))

    def test_non_git_checkout_is_non_releasable_for_a_test_inspection(self):
        # Inspecting an artifact outside a checkout stays useful; it just can
        # never be reported as a release.
        self._with_status("fatal: not a git repository", rc=128)
        gate = Gate()
        gate_module.worktree_gate(gate, "test")
        self.assertEqual(status_of(gate, "repo.clean_worktree"), SKIP)
        self.assertFalse(gate.facts["releasable"])




class ManagedRuntimeCountTest(unittest.TestCase):
    """The Managed Runtime count must not include Core's own libraries.

    N3 renamed Core's launcher to libpocketclaw-web.so, which matches the
    Managed Runtime prefix. Counting it reported 9 where 8 tools exist, and
    because the check is a floor, seven real tools plus the launcher would also
    have reached 8 — the guard would have stopped noticing a dropped payload.
    """

    ABI = "arm64-v8a"
    CORE = ("libpocketclaw.so", "libpocketclaw-web.so")
    TOOLS = ("curl", "gh", "git", "git-remote-http", "jq", "python", "rg", "sqlite3")

    def count(self, tool_names):
        names = [f"lib/{self.ABI}/{n}" for n in self.CORE]
        names += [f"lib/{self.ABI}/libpocketclaw-{t}.so" for t in tool_names]
        core_names = set(self.CORE)
        return len([n for n in names
                    if n.startswith(f"lib/{self.ABI}/libpocketclaw-")
                    and PurePosixPath(n).name not in core_names])

    def test_core_pair_plus_eight_tools_reports_eight(self):
        self.assertEqual(self.count(self.TOOLS), 8)

    def test_core_web_is_never_counted_as_a_runtime_tool(self):
        # Without the exclusion this is 9.
        self.assertNotIn("libpocketclaw-web.so",
                         [f"libpocketclaw-{t}.so" for t in self.TOOLS])
        self.assertEqual(self.count(self.TOOLS), len(self.TOOLS))

    def test_a_dropped_payload_now_fails_the_floor(self):
        seven = self.TOOLS[:-1]
        self.assertEqual(self.count(seven), 7)
        self.assertLess(self.count(seven), 8,
                        "seven tools must fail the >=8 floor; the launcher must "
                        "not make up the difference")

    def test_core_libs_are_still_validated_separately(self):
        # CORE_LIBS keeps its own checks; excluding it here removes it from one
        # count, not from the gate.
        source = Path("tool/release_gate.py").read_text(encoding="utf-8")
        self.assertIn("artifact.core_matches_staged", source)
        self.assertIn("core.staged_freshness", source)
        self.assertIn('CORE_LIBS = ("libpocketclaw.so", "libpocketclaw-web.so")', source)


class DartHardeningEvidenceTest(unittest.TestCase):
    def make_symbols(self, directory):
        path = Path(directory) / "app.android-arm64.symbols"
        path.write_bytes(
            b"\x7fELF.debug_info.debug_line\0"
            + b"\0".join(gate_module.DART_APP_SYMBOL_MARKERS)
        )
        return path

    def test_hardened_artifact_requires_both_public_and_private_evidence(self):
        with tempfile.TemporaryDirectory() as tmp:
            symbols = self.make_symbols(tmp)
            app = b"\x7fELF" + gate_module.DART_GENERATED_REGISTRANT_URI
            gate = Gate()
            gate_module.dart_hardening_gates(gate, app, ["lib/arm64-v8a/libapp.so"], symbols)
            for name in (
                "artifact.dart_split_debug_info",
                "artifact.dart_generated_source_uri",
                "artifact.dart_obfuscation",
                "artifact.dart_symbols_private",
            ):
                self.assertEqual(status_of(gate, name), PASS)

    def test_host_uri_strategy_cannot_pass_without_controlled_package_uri(self):
        with tempfile.TemporaryDirectory() as tmp:
            symbols = self.make_symbols(tmp)
            app = b"file:///home/person/checkout/dart_plugin_registrant.dart"
            gate = Gate()
            gate_module.dart_hardening_gates(gate, app, ["lib/arm64-v8a/libapp.so"], symbols)
            self.assertEqual(status_of(gate, "artifact.dart_generated_source_uri"), FAIL)

    def test_unobfuscated_name_fails_even_when_split_info_exists(self):
        with tempfile.TemporaryDirectory() as tmp:
            symbols = self.make_symbols(tmp)
            app = gate_module.DART_GENERATED_REGISTRANT_URI + gate_module.DART_APP_SYMBOL_MARKERS[0]
            gate = Gate()
            gate_module.dart_hardening_gates(gate, app, ["lib/arm64-v8a/libapp.so"], symbols)
            self.assertEqual(status_of(gate, "artifact.dart_obfuscation"), FAIL)

    def test_packaged_private_symbols_fail(self):
        with tempfile.TemporaryDirectory() as tmp:
            symbols = self.make_symbols(tmp)
            app = gate_module.DART_GENERATED_REGISTRANT_URI
            gate = Gate()
            gate_module.dart_hardening_gates(
                gate,
                app,
                ["lib/arm64-v8a/libapp.so", "assets/private-symbols/app.android-arm64.symbols"],
                symbols,
            )
            self.assertEqual(status_of(gate, "artifact.dart_symbols_private"), FAIL)


class R8HardeningEvidenceTest(unittest.TestCase):
    def make_outputs(self, directory: Path, packaged_mapping=False):
        apk = directory / "app-release.apk"
        mapping = directory / "mapping.txt"
        lines = [f"{name} -> {name}:" for name in r8_contract.REQUIRED_COMPONENTS]
        lines += [
            f"{name} -> gate.{index}:"
            for index, name in enumerate(r8_contract.INTERNAL_CLASSES)
        ]
        mapping.write_text("\n".join(lines) + "\n", encoding="utf-8")
        mapping.with_name("usage.txt").write_text("removed.Class\n", encoding="utf-8")
        with zipfile.ZipFile(apk, "w") as archive:
            archive.writestr("classes.dex", b"dex\n035\0obfuscated")
            if packaged_mapping:
                archive.writestr("assets/mapping.txt", mapping.read_bytes())
        return apk, mapping

    def test_gate_records_effective_private_r8_evidence(self):
        with tempfile.TemporaryDirectory() as tmp:
            apk, mapping = self.make_outputs(Path(tmp))
            gate = Gate()
            gate_module.r8_hardening_gates(gate, apk, mapping)
            for name in (
                "artifact.r8_mapping",
                "artifact.r8_shrinking",
                "artifact.r8_obfuscation",
                "artifact.r8_entry_points",
                "artifact.r8_mapping_private",
            ):
                self.assertEqual(status_of(gate, name), PASS)

    def test_gate_refuses_packaged_mapping(self):
        with tempfile.TemporaryDirectory() as tmp:
            apk, mapping = self.make_outputs(Path(tmp), packaged_mapping=True)
            gate = Gate()
            gate_module.r8_hardening_gates(gate, apk, mapping)
            self.assertEqual(status_of(gate, "artifact.r8_contract"), FAIL)


if __name__ == "__main__":
    unittest.main(verbosity=2)
