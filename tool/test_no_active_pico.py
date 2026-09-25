#!/usr/bin/env python3
"""Tests for the Zero Active Pico guard, and the dispositions it does not cover.

The guard scans PocketClaw-owned production source. A few decisions live
outside that scope on purpose — the WeCom source id is in the upstream CLI, the
upstream module path is the whole vendored tree — so they are pinned here
instead, by path and content, rather than left to prose.
"""

import re
import subprocess
import sys
import unittest
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(REPO / "tool"))

import no_active_pico as guard  # noqa: E402


class GuardShapeTest(unittest.TestCase):
    def test_every_entry_has_a_valid_category_and_a_reason(self):
        self.assertGreater(len(guard.ALLOWLIST), 0)
        for i, entry in enumerate(guard.ALLOWLIST):
            with self.subTest(entry=i):
                self.assertIn(entry.get("category"), guard.VALID_CATEGORIES)
                self.assertTrue(entry.get("reason", "").strip(),
                                "an exemption without a reason cannot be reviewed")
                re.compile(entry["pattern"])

    def test_the_allowlist_stays_small(self):
        # Not a style rule: every entry is a place Pico may survive, and a list
        # long enough to skim is a list nobody reads.
        self.assertLessEqual(len(guard.ALLOWLIST), 25,
                             "the allowlist is growing into a blanket exclusion")

    def test_the_tree_is_currently_clean(self):
        findings, used = guard.scan()
        self.assertEqual(findings, [], "unclassified active Pico occurrences")
        stale = [i for i in range(len(guard.ALLOWLIST)) if i not in used]
        self.assertEqual(stale, [], "allowlist entries matching nothing")


class ScopeTest(unittest.TestCase):
    def test_owned_production_is_in_scope(self):
        for path in (
            "lib/src/core/service_manager.dart",
            "android/app/src/main/kotlin/com/lord1egypt/pocketclaw/"
            "service/PocketClawService.kt",
            "core/src/web/backend/middleware/launcher_dashboard_auth.go",
            "core/src/web/frontend/src/features/chat/protocol.ts",
            "core/src/pkg/channels/pocketclaw/pocketclaw.go",
        ):
            self.assertTrue(guard.is_owned_production(path), path)

    def test_upstream_docs_and_tests_are_not(self):
        for path in (
            "core/src/pkg/agent/agent.go",          # vendored upstream
            "core/src/cmd/picoclaw/main.go",        # vendored upstream
            "PROJECT_STATE.md",                     # the migration record
            "core/src/docs/guides/chat-apps.md",    # upstream documentation
            "test/unit/zero_pico_n4i_test.dart",    # a test naming what it removes
            "core/src/web/backend/api/session_test.go",
            "android/app/src/main/jniLibs/arm64-v8a/libpocketclaw.so",
        ):
            self.assertFalse(guard.is_owned_production(path), path)

    def test_words_that_merely_contain_the_stem_are_not_matches(self):
        for benign in ("picomatch", "a picometer wide", "tópico", "tópicos"):
            self.assertIsNone(guard.PICO.search(guard.NOT_THE_PRODUCT.sub("", benign)),
                              benign)


class MutationTest(unittest.TestCase):
    """The guard has to fail on the things it exists to catch."""

    def _scan_with(self, path: Path, body: str):
        original = path.read_text(encoding="utf-8")
        path.write_text(original + body, encoding="utf-8")
        try:
            return guard.scan()
        finally:
            path.write_text(original, encoding="utf-8")

    def test_a_new_active_identity_in_owned_production_fails(self):
        target = REPO / "core/src/pkg/channels/pocketclaw/protocol.go"
        findings, _ = self._scan_with(target, '\n// const marker = "picoclaw_test_output"\n')
        self.assertTrue(
            any(f["path"].endswith("protocol.go") and "picoclaw_test_output" in f["text"]
                for f in findings),
            "a new picoclaw_* identity in owned production source was not caught")

    def test_the_same_string_in_a_test_file_is_ignored(self):
        target = REPO / "core/src/pkg/channels/pocketclaw/pocketclaw_test.go"
        findings, _ = self._scan_with(target, '\n// picoclaw_test_output\n')
        self.assertFalse(
            any("picoclaw_test_output" in f["text"] for f in findings),
            "tests are out of scope; a test naming the old identity must not fail")

    def test_removing_a_legacy_occurrence_makes_its_entry_stale(self):
        # The WeCom-style case: an entry whose last match disappears must be
        # reported so the exemption is deleted rather than left lying around.
        probe = dict(guard.ALLOWLIST[0])
        probe["pattern"] = r"a_string_that_appears_nowhere_at_all"
        guard.ALLOWLIST.append(probe)
        try:
            _, used = guard.scan()
            self.assertNotIn(len(guard.ALLOWLIST) - 1, used,
                             "an entry matching nothing was reported as in use")
        finally:
            guard.ALLOWLIST.pop()


class PublicSurfaceTest(unittest.TestCase):
    """What the repository says, as distinct from what it does."""

    def test_public_surfaces_are_currently_clean(self):
        self.assertEqual(guard.scan_public(), [],
                         "Pico branding on a public surface outside attribution")

    def test_branding_outside_the_attribution_section_is_caught(self):
        readme = REPO / "README.md"
        original = readme.read_text(encoding="utf-8")
        readme.write_text(
            original.replace("## What it is", "## What it is\n\nPicoClaw is the product.", 1),
            encoding="utf-8")
        try:
            findings = guard.scan_public()
            self.assertTrue(any(f["path"] == "README.md" for f in findings),
                            "branding in the product description was not caught")
        finally:
            readme.write_text(original, encoding="utf-8")

    def test_the_attribution_section_may_name_upstream(self):
        # The exception has to actually work, or the only way to pass the guard
        # is to drop the credit — which would be worse than the branding.
        readme = (REPO / "README.md").read_text(encoding="utf-8")
        self.assertRegex(readme, r"(?im)^#{1,6}\s.*\b(attribution|upstream)\b",
                         "no attribution heading for the exception to key on")
        self.assertIn("PicoClaw", readme,
                      "upstream is not credited anywhere in the README")
        self.assertEqual(guard.scan_public(), [])

    def test_the_guard_may_be_named_in_contributor_docs(self):
        # Telling a contributor which tool to run is not branding.
        self.assertFalse(guard.PUBLIC_ALLOWED.sub("", "run tool/no_active_pico.py").count("pico"))


class PinnedDispositionTest(unittest.TestCase):
    """Decisions the guard's scope deliberately does not reach."""

    def test_wecom_source_id_is_preserved_and_confined(self):
        # EXTERNAL COMPATIBILITY. It is sent to Tencent as source/sourceID, so
        # it names this application to a third party; changing it is not a
        # rename, it is a claim about identity to someone else's service.
        # There are two independent copies: the upstream CLI's, and the
        # dashboard's — and the dashboard ships, so this value really does reach
        # Tencent from the product. Both must stay put, and stay identical.
        copies = [
            REPO / "core/src/cmd/picoclaw/internal/auth/wecom.go",
            REPO / "core/src/web/backend/api/wecom.go",
        ]
        for path in copies:
            body = path.read_text(encoding="utf-8")
            self.assertIn('wecomQRSourceID          = "picoclaw"', body,
                          f"{path.name}: the WeCom source id changed. It is sent "
                          f"to work.weixin.qq.com as source/sourceID, so this "
                          f"alters what a third party is told this application "
                          f"is, and an unregistered value breaks QR login.")
            self.assertIn("work.weixin.qq.com", body,
                          f"{path.name}: the evidence for preserving the id is "
                          f"that it is sent to WeCom")

        # Nowhere else may name it, so the exemption cannot spread.
        out = subprocess.run(
            ["git", "-C", str(REPO), "grep", "-l", "wecomQRSourceID"],
            capture_output=True, text=True).stdout.split()
        allowed = {"core/src/cmd/picoclaw/internal/auth/wecom.go",
                   "core/src/cmd/picoclaw/internal/auth/wecom_test.go",
                   "core/src/web/backend/api/wecom.go",
                   "tool/no_active_pico.py",
                   "tool/test_no_active_pico.py"}
        for path in out:
            if path.endswith(".md"):
                continue  # the documented disposition, not a reference
            self.assertIn(path, allowed,
                          f"{path} references the WeCom source id outside the "
                          f"two files that send it")

    def test_upstream_module_identity_is_unchanged(self):
        gomod = (REPO / "core/src/go.mod").read_text(encoding="utf-8")
        self.assertIn("module github.com/sipeed/picoclaw", gomod)

    def test_shipping_binary_names_are_pocketclaw(self):
        jni = REPO / "android/app/src/main/jniLibs/arm64-v8a"
        for lib in ("libpocketclaw.so", "libpocketclaw-web.so"):
            self.assertTrue((jni / lib).is_file(), lib)
        self.assertEqual(
            sorted(p.name for p in jni.glob("libpicoclaw*.so")), [],
            "a Pico-named binary is staged")

    def test_the_historical_managed_runtime_digest_is_untouched(self):
        # vc60/vc61 shipped a section whose digest is evidence. Rewriting it to
        # look canonical would make the history lie.
        digest = "47b63011a55eaa659470f2ab09d05532e9942800848020a1cfe443e6f21aca76"
        out = subprocess.run(["git", "-C", str(REPO), "grep", "-l", digest],
                             capture_output=True, text=True).stdout.split()
        self.assertTrue(out, f"the immutable historical digest {digest} is gone")

    def test_the_irc_default_nick_is_current(self):
        defaults = (REPO / "core/src/pkg/config/defaults.go").read_text(encoding="utf-8")
        self.assertIn('"nick":     "pocketclaw"', defaults,
                      "the shipped IRC default would announce the old identity "
                      "on a real IRC network")
        self.assertNotIn('"nick":     "picoclaw"', defaults)

    def test_the_distribution_channel_define_is_gone(self):
        # Renamed in N4K-B, then removed with the device telemetry it fed in
        # F-Droid Phase B. Neither name may come back.
        dart = (REPO / "lib/src/core/service_manager.dart").read_text(encoding="utf-8")
        self.assertNotIn("POCKETCLAW_DISTRIBUTION_CHANNEL", dart)
        self.assertNotIn("PICOCLAW_DISTRIBUTION_CHANNEL", dart)


if __name__ == "__main__":
    unittest.main(verbosity=2)
