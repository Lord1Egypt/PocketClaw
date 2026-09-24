#!/usr/bin/env python3
"""Tests for the CJK source-hygiene check."""

import sys
import unittest
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(REPO / "tool"))

import cjk_hygiene as hygiene  # noqa: E402


class HygieneTest(unittest.TestCase):
    def test_every_entry_is_classified_and_explained(self):
        for glob, category, lines, reason in hygiene.ALLOWLIST:
            with self.subTest(glob=glob):
                self.assertIn(category, hygiene.VALID_CATEGORIES)
                self.assertTrue(reason.strip())
                self.assertTrue(lines is None or lines > 0)

    def test_owned_app_source_has_no_blanket_entry(self):
        # A directory-wide entry over PocketClaw's own code would readmit the
        # developer comments this check exists to keep out.
        for glob, *_ in hygiene.ALLOWLIST:
            for owned in ("android/", "lib/src/core", "lib/src/ui", "tool/"):
                self.assertFalse(
                    glob.startswith(owned) and ("*" in glob),
                    f"{glob} is a blanket exemption over owned source",
                )

    def test_the_detector_sees_cjk_and_ignores_other_scripts(self):
        # Escaped, so this file does not itself need an allowlist entry.
        self.assertTrue(hygiene.CJK.search("/// \u68c0\u67e5\u5b58\u50a8\u6743\u9650"))
        self.assertTrue(hygiene.CJK.search("\u30e2\u30c7\u30eb"))
        self.assertTrue(hygiene.CJK.search("\ubaa8\ub378"))
        self.assertTrue(hygiene.CJK.search("a\uff0cb"))
        self.assertIsNone(hygiene.CJK.search("تسلم café हिन्दी — ✓"))

    def test_an_unlisted_file_is_not_classified(self):
        self.assertIsNone(hygiene.entry_for(
            "android/app/src/main/kotlin/com/lord1egypt/pocketclaw/New.kt"))
        self.assertIsNotNone(hygiene.entry_for("core/src/docs/project/README.zh.md"))

    def test_the_tree_is_currently_clean(self):
        self.assertEqual(hygiene.scan(), [])


if __name__ == "__main__":
    unittest.main()
