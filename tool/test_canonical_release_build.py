#!/usr/bin/env python3
"""Tests for the canonical (F-Droid buildserver) release build: PC-DEF-092."""

import importlib.util
import sys
import unittest
from pathlib import Path

spec = importlib.util.spec_from_file_location(
    "canonical_release_build", Path(__file__).resolve().parent / "canonical_release_build.py"
)
canonical = importlib.util.module_from_spec(spec)
sys.modules["canonical_release_build"] = canonical
spec.loader.exec_module(canonical)

COMMIT = "0123456789abcdef0123456789abcdef01234567"


class MetadataTest(unittest.TestCase):
    def test_repository_build_carries_no_reference_binary(self):
        text = canonical.render_metadata(COMMIT, "0.2.3", 65, track_b=False)
        self.assertIn(f"commit: {COMMIT}\n", text)
        self.assertIn("versionName: 0.2.3\n", text)
        self.assertIn("versionCode: 65\n", text)
        self.assertIn("CurrentVersionCode: 65\n", text)
        self.assertNotIn("Binaries", text)
        self.assertNotIn("AllowedAPKSigningKeys", text)
        self.assertNotIn("@", text.replace("flutter@", "").replace("rustup@", "").replace("pnpm@", ""))

    def test_track_b_pins_the_upstream_apk_and_the_enrolled_signer(self):
        text = canonical.render_metadata(COMMIT, "0.2.3", 65, track_b=True)
        self.assertIn(canonical.BINARIES_URL, text)
        self.assertIn(canonical.enrolled_signer(), text)
        self.assertRegex(canonical.enrolled_signer(), r"^[0-9a-f]{64}$")

    def test_a_short_or_symbolic_commit_is_refused(self):
        for commit in ("0123456", "v0.2.3", "HEAD"):
            with self.subTest(commit=commit), self.assertRaises(canonical.CanonicalBuildError):
                canonical.render_metadata(commit, "0.2.3", 65, track_b=False)

    def test_the_v022_recipe_workarounds_are_gone(self):
        # The v0.2.2 dry run needed all three; each is now fixed upstream
        # (PC-DEF-091, the Core epoch resolver, the builder's gradle fallback).
        template = canonical.TEMPLATE.read_text(encoding="utf-8")
        self.assertNotIn("SOURCE_DATE_EPOCH", template)
        self.assertNotIn("sed -i", template)
        self.assertNotIn("gradlew", template)
        self.assertNotIn("manifest.json", template)

    def test_the_recipe_rebuilds_every_committed_native_payload(self):
        template = canonical.TEMPLATE.read_text(encoding="utf-8")
        jni = canonical.REPO / "android/app/src/main/jniLibs/arm64-v8a"
        for library in sorted(p.name for p in jni.glob("lib*.so")):
            self.assertIn(f"android/app/src/main/jniLibs/arm64-v8a/{library}", template, library)

    def test_the_current_commit_declares_a_version(self):
        name, code = canonical.version_at("HEAD")
        self.assertRegex(name, r"^\d+\.\d+\.\d+$")
        self.assertGreater(code, 64)


class LayoutTest(unittest.TestCase):
    def test_the_container_uses_fdroidservers_server_layout(self):
        command = canonical.docker_run_command(
            Path("/work"), Path("/fdroidserver"), "name", "com.lord1egypt.pocketclaw:65")
        joined = " ".join(command)
        for mount in ("/work/build:/home/vagrant/build", "/work/metadata:/home/vagrant/metadata",
                      "/work/srclibs:/home/vagrant/srclibs", "/work/cache:/home/vagrant/.cache",
                      "/fdroidserver:/home/vagrant/fdroidserver:ro"):
            self.assertIn(mount, joined)
        self.assertIn("-u vagrant", joined)
        self.assertIn("fdroid build --on-server --no-tarball", command[-1])
        self.assertIn("cd /home/vagrant", command[-1])

    def test_the_image_is_pinned_by_digest(self):
        self.assertRegex(canonical.IMAGE, r"@sha256:[0-9a-f]{64}$")
        self.assertRegex(canonical.FDROIDSERVER_COMMIT, r"^[0-9a-f]{40}$")


if __name__ == "__main__":
    unittest.main()
