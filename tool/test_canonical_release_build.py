#!/usr/bin/env python3
"""Tests for the canonical (F-Droid buildserver) release build: PC-DEF-092."""

import fnmatch
import importlib.util
import re
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

spec = importlib.util.spec_from_file_location(
    "canonical_release_build", Path(__file__).resolve().parent / "canonical_release_build.py"
)
canonical = importlib.util.module_from_spec(spec)
sys.modules["canonical_release_build"] = canonical
spec.loader.exec_module(canonical)

COMMIT = "0123456789abcdef0123456789abcdef01234567"
JNI_DIR = "android/app/src/main/jniLibs/arm64-v8a"
JNI_GLOB = f"{JNI_DIR}/*.so"


class MetadataTest(unittest.TestCase):
    def test_repository_build_carries_no_reference_binary(self):
        text = canonical.render_metadata(COMMIT, "0.2.3", 65, track_b=False)
        self.assertIn(f"commit: {COMMIT}\n", text)
        self.assertIn("versionName: 0.2.3\n", text)
        self.assertIn("versionCode: 65\n", text)
        self.assertIn("CurrentVersionCode: 65\n", text)
        self.assertNotIn("Binaries", text)
        self.assertNotIn("AllowedAPKSigningKeys", text)
        self.assertNotIn("@", text.replace("flutter@", "").replace("pnpm@", ""))

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
        self.assertIn(f"      - {JNI_GLOB}\n", template)
        jni = canonical.REPO / JNI_DIR
        libraries = sorted(p.name for p in jni.glob("lib*.so"))
        self.assertEqual(len(libraries), 10)
        for library in libraries:
            self.assertTrue(fnmatch.fnmatchcase(f"{JNI_DIR}/{library}", JNI_GLOB), library)
        self.assertFalse(fnmatch.fnmatchcase(f"{JNI_DIR}/version.txt", JNI_GLOB))

    def test_node_and_rustup_come_from_debian(self):
        # fdroiddata review of MR !50146: Debian packages, not downloads or a
        # rustup srclib; rustup still installs the pinned rustc that decides
        # the rg payload bytes.
        jni = canonical.REPO / JNI_DIR
        for track_b in (False, True):
            text = canonical.render_metadata(COMMIT, "0.2.3", 65, track_b=track_b)
            with self.subTest(track_b=track_b):
                for obsolete in ("nodejs.org/dist", "node-v25.8.1", "rustup@1.29.1",
                                 "$$rustup$$/rustup-init.sh", ".cargo/bin"):
                    self.assertNotIn(obsolete, text)
                for library in jni.glob("lib*.so"):
                    self.assertNotIn(f"{JNI_DIR}/{library.name}", text)
                self.assertRegex(text, r"apt-get install [^\n]*(\n        [^\n]*)*"
                                       r"\bnodejs npm\b[^\n]*\brustup\b")
                self.assertIn("      - flutter@stable\n    rm:", text)
                self.assertIn("      - rustup toolchain install 1.94.1 --profile minimal"
                              " --target aarch64-linux-android\n", text)
                self.assertIn("pnpm@10.33.0", text)
                self.assertIn(f"      - {JNI_GLOB}\n", text)
                self.assertIn('      - export PATH="$$flutter$$/bin:$PATH"\n', text)
        self.assertEqual(canonical.SRCLIBS, ("flutter",))

    def test_the_current_commit_declares_a_version(self):
        name, code = canonical.version_at("HEAD")
        self.assertRegex(name, r"^\d+\.\d+\.\d+$")
        self.assertGreater(code, 64)


def prebuild_commands(text: str) -> list[str]:
    """The prebuild list as fdroidserver sees it: folded lines rejoined, quotes off."""
    block = text.split("    prebuild:\n", 1)[1].split("    scandelete:\n", 1)[0]
    items = [line[len("      - "):] for line in re.sub(r"\n {8}(?=\S)", " ", block).splitlines()]
    return [item[1:-1].replace("''", "'") if item.startswith("'") else item for item in items]


class FlutterPinTest(unittest.TestCase):
    """fdroiddata review of MR !50146: pin Flutter upstream and extract it."""

    def setUp(self):
        self.text = canonical.render_metadata(COMMIT, "0.2.3", 65, track_b=True)
        self.commands = prebuild_commands(self.text)

    def test_fdroiddata_names_only_the_generic_srclib(self):
        self.assertIn("    srclibs:\n      - flutter@stable\n    rm:", self.text)
        self.assertNotRegex(self.text, r"flutter@\d")

    def test_the_version_is_extracted_then_checked_out_before_flutter_runs(self):
        extract = next(i for i, c in enumerate(self.commands) if c.startswith("flutterVersion="))
        self.assertIn("runtime/toolchains.env", self.commands[extract])
        self.assertTrue(self.commands[extract + 1].startswith("[[ $flutterVersion =~ "))
        self.assertEqual(self.commands[extract + 2], "git -C $$flutter$$ checkout -f $flutterVersion")
        first_flutter = next(i for i, c in enumerate(self.commands) if c.startswith("$$flutter$$/bin/"))
        self.assertLess(extract + 2, first_flutter)

    def run_extraction(self, toolchains: str | None) -> tuple[int, str]:
        """Runs the rendered extraction and checkout the way fdroidserver does."""
        extract = next(i for i, c in enumerate(self.commands) if c.startswith("flutterVersion="))
        with tempfile.TemporaryDirectory() as tmp:
            source, flutter = Path(tmp, "app"), Path(tmp, "flutter")
            (source / "runtime").mkdir(parents=True)
            if toolchains is not None:
                (source / "runtime/toolchains.env").write_text(toolchains, encoding="utf-8")
            git = ["git", "-c", "user.name=t", "-c", "user.email=t@t", "-C", str(flutter)]
            flutter.mkdir()
            subprocess.run(git + ["init", "-q"], check=True)
            for tag in ("3.47.1", "3.99.0"):
                subprocess.run(git + ["commit", "-q", "--allow-empty", "-m", tag], check=True)
                subprocess.run(git + ["tag", tag], check=True)
            script = "; ".join(self.commands[extract:extract + 3]).replace("$$flutter$$", str(flutter))
            result = subprocess.run(["bash", "-e", "-u", "-o", "pipefail", "-c", script],
                                    cwd=source, capture_output=True, text=True)
            head = subprocess.run(git + ["describe", "--tags"], capture_output=True, text=True)
            return result.returncode, head.stdout.strip()

    def test_the_upstream_pin_is_what_gets_checked_out(self):
        pinned = canonical.flutter_pin((canonical.REPO / "runtime/toolchains.env").read_text(encoding="utf-8"))
        self.assertRegex(pinned, r"^\d+\.\d+\.\d+$")
        code, checked_out = self.run_extraction(f"# x\nPOCKETCLAW_FLUTTER_VERSION={pinned}\n")
        self.assertEqual((code, checked_out), (0, pinned))

    def test_an_absent_or_malformed_pin_stops_the_build_before_checkout(self):
        for toolchains in (None, "", "POCKETCLAW_RUST_TOOLCHAIN=1.94.1\n",
                           "POCKETCLAW_FLUTTER_VERSION=3.47\n", "POCKETCLAW_FLUTTER_VERSION=stable\n",
                           "POCKETCLAW_FLUTTER_VERSION=3.47.1\nPOCKETCLAW_FLUTTER_VERSION=3.47.1\n"):
            with self.subTest(toolchains=toolchains):
                code, checked_out = self.run_extraction(toolchains)
                self.assertNotEqual(code, 0)
                self.assertEqual(checked_out, "3.99.0")

    def test_rendering_refuses_a_missing_or_ambiguous_pin(self):
        for toolchains in ("", "POCKETCLAW_FLUTTER_VERSION=3.47\n", " POCKETCLAW_FLUTTER_VERSION=3.47.1\n",
                           "POCKETCLAW_FLUTTER_VERSION=3.47.1\nPOCKETCLAW_FLUTTER_VERSION=3.48.0\n"):
            with self.subTest(toolchains=toolchains), self.assertRaises(canonical.CanonicalBuildError):
                canonical.flutter_pin(toolchains)
        with self.assertRaises(canonical.CanonicalBuildError):
            canonical.flutter_version_at("0000000000000000000000000000000000000000")


class LayoutTest(unittest.TestCase):
    def test_the_container_uses_fdroidservers_server_layout(self):
        command = canonical.docker_run_command(
            Path("/work"), Path("/fdroidserver"), "name", "com.lord1egypt.pocketclaw:65", COMMIT)
        joined = " ".join(command)
        for mount in ("/work/build:/home/vagrant/build", "/work/metadata:/home/vagrant/metadata",
                      "/work/srclibs:/home/vagrant/srclibs", "/work/cache:/home/vagrant/.cache",
                      "/fdroidserver:/home/vagrant/fdroidserver:ro"):
            self.assertIn(mount, joined)
        self.assertIn("-u vagrant", joined)
        self.assertIn("fdroid build --on-server --no-tarball", command[-1])
        self.assertIn("cd /home/vagrant", command[-1])
        # The host half of server mode checks the commit out with fdroidserver's
        # own VCS layer before the server half builds it.
        self.assertIn(f"build/com.lord1egypt.pocketclaw {COMMIT}", command[-1])
        self.assertLess(command[-1].index("gotorevision"), command[-1].index("--on-server"))

    def test_the_image_is_pinned_by_digest(self):
        self.assertRegex(canonical.IMAGE, r"@sha256:[0-9a-f]{64}$")
        self.assertRegex(canonical.FDROIDSERVER_COMMIT, r"^[0-9a-f]{40}$")


if __name__ == "__main__":
    unittest.main()
