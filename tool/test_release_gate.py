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
import unittest
from pathlib import Path

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

    def test_non_git_checkout_is_skipped_not_failed(self):
        self._with_status("fatal: not a git repository", rc=128)
        gate = Gate()
        gate_module.worktree_gate(gate, "production")
        self.assertEqual(status_of(gate, "repo.clean_worktree"), SKIP)


if __name__ == "__main__":
    unittest.main(verbosity=2)
