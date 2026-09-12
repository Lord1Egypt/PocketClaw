#!/usr/bin/env python3
"""Focused regression tests for H5B native build and support contracts."""

from __future__ import annotations

import importlib
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
import time
import unittest
from pathlib import Path


REPO = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(REPO / "tool"))


class NativeSupportContractTest(unittest.TestCase):
    def test_git_uses_explicit_curl_flags_without_curldir_rpath_trigger(self) -> None:
        script = (REPO / "runtime/build-git-android-arm64.sh").read_text()
        self.assertNotIn('CURLDIR="$DEPS_PREFIX"', script)
        self.assertIn('CURL_CFLAGS="-I$DEPS_PREFIX/include"', script)
        self.assertIn('CURL_LDFLAGS="-lcurl ', script)

    def test_runtime_contract_is_root_independent_and_fails_search_paths(self) -> None:
        script = (REPO / "runtime/android-build-env.sh").read_text()
        self.assertIn('JNI_LIBS="${JNI_LIBS:-', script)
        self.assertIn('NATIVE_SYMBOL_ROOT="${NATIVE_SYMBOL_ROOT:-', script)
        self.assertIn("-ffile-prefix-map=$BUILD_ROOT=/pocketclaw-runtime/build", script)
        self.assertIn('SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-$RUNTIME_EPOCH}"', script)
        self.assertNotIn("show -s --format=%ct HEAD", script)
        self.assertIn("/(RPATH)\\|(RUNPATH)/p", script)
        self.assertIn('grep -c -F "$BUILD_ROOT"', script)

    def test_all_owned_payload_recipes_generate_private_support(self) -> None:
        env = (REPO / "runtime/android-build-env.sh").read_text()
        self.assertIn("tool/native_support.py", env)
        for script in sorted((REPO / "runtime").glob("build-*-android-arm64.sh")):
            text = script.read_text()
            if script.name == "build-all-android-arm64.sh":
                continue
            self.assertTrue("install_payload" in text, script.name)
        core = (REPO / "core/build-android-arm64.sh").read_text()
        self.assertIn("tool/native_support.py", core)
        capture = core.index("tool/native_support.py")
        for guard in ("-trimpath regressed", "does not carry its source fingerprint"):
            self.assertLess(core.index(guard), capture, guard)
        loop = core.rindex("for lib in libpocketclaw.so libpocketclaw-web.so;", 0, capture)
        self.assertIn('--logical-name "$lib"', core[loop:])
        self.assertIn("STRIP_LDFLAGS=", core)
        self.assertIn('if [[ "$CORE_BUILD_DIR" = /* ]]', core)
        makefile = (REPO / "core/src/Makefile").read_text()
        self.assertIn("BUILD_OUTPUT_DIR=$(if $(filter /%,$(BUILD_DIR))", makefile)
        self.assertIn('OUTPUT_ANDROID_ARM64="$(BUILD_OUTPUT_DIR)/picoclaw-launcher-android-arm64"', makefile)

    def test_python_support_comes_from_unstripped_interpreter(self) -> None:
        script = (REPO / "runtime/build-python-android-arm64.sh").read_text()
        self.assertIn('no-strip "$OBJ/python"', script)
        self.assertIn("replacement = \"-DVPATH='\\\"/pocketclaw/cpython\\\"'\"", script)
        self.assertLess(script.index('BUILD_PYTHON="$PY_SRC/cross-build/build/python"'),
                        script.index("replacement = \"-DVPATH="))
        self.assertIn("../cpython/configure", script)
        self.assertNotIn('  "$PY_SRC/configure"', script)
        stdlib = (REPO / "runtime/python-lite-stdlib.py").read_text()
        bootstrap = (REPO / "runtime/install-python-bootstrap.py").read_text()
        self.assertIn("PycInvalidationMode.CHECKED_HASH", stdlib)
        self.assertIn("ZipInfo(ENTRY, date_time=ZIP_EPOCH)", bootstrap)

    def test_jq_does_not_embed_root_bearing_autoconf_arguments(self) -> None:
        script = (REPO / "runtime/build-jq-android-arm64.sh").read_text()
        configure = script.index("./configure")
        stable_config = script.index("cat > src/config_opts.inc")
        build = script.index('make -j"$(nproc)"', stable_config)
        self.assertLess(configure, stable_config)
        self.assertLess(stable_config, build)

    def test_private_root_is_ignored(self) -> None:
        result = subprocess.run(
            ["git", "check-ignore", "build/private-symbols/native/android-arm64/manifest.json"],
            cwd=REPO, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
        )
        self.assertEqual(result.returncode, 0, result.stderr)

    @unittest.skipUnless(shutil.which("cc") and shutil.which("objcopy") and
                         shutil.which("readelf") and shutil.which("nm") and
                         shutil.which("addr2line") and shutil.which("strip"),
                         "host ELF tools are required")
    def test_companion_symbolizes_while_shipped_file_is_stripped(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source = root / "sample.c"
            unstripped, shipped, private = root / "sample", root / "libsample.so", root / "private"
            source.write_text("static int helper(int x) { return x + 7; }\nint main(void) { return helper(3); }\n")
            subprocess.run(["cc", "-g", "-O0", "-o", unstripped, source], check=True)
            shutil.copy2(unstripped, shipped)
            subprocess.run(["strip", "--strip-unneeded", shipped], check=True)
            subprocess.run([
                sys.executable, REPO / "tool/native_support.py",
                "--source", unstripped, "--shipped", shipped,
                "--output-root", private, "--logical-name", "libsample.so",
                "--category", "test", "--toolchain", "host-cc",
                "--source-id", "unit fixture", "--probe-symbol", "main",
                "--objcopy", shutil.which("objcopy"), "--readelf", shutil.which("readelf"),
                "--nm", shutil.which("nm"), "--addr2line", shutil.which("addr2line"),
            ], check=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
            manifest = json.loads((private / "manifest.json").read_text())
            item = manifest["artifacts"][0]
            self.assertEqual(item["logicalName"], "libsample.so")
            self.assertEqual(item["symbolization"]["resolvedFunction"], "main")
            sections = subprocess.run(["readelf", "-SW", shipped], check=True,
                                      text=True, stdout=subprocess.PIPE).stdout
            self.assertNotIn(".debug_info", sections)
            self.assertTrue((private / "libsample.so.debug").is_file())

    def test_pinned_runtime_epoch_does_not_move_with_repository_state(self) -> None:
        """The Python payload embeds this date, so the catalog depends on it.

        A commit that records a payload checksum must not also change the bytes
        that checksum describes, which is what any HEAD-derived epoch would do.
        """
        script = (REPO / "runtime/android-build-env.sh").read_text()
        pinned = re.search(r"^RUNTIME_EPOCH=(\d+)", script, re.MULTILINE)
        self.assertIsNotNone(pinned)
        resolved = subprocess.run(
            ["bash", "-c", 'source "$1" >/dev/null 2>&1; printf %s "$SOURCE_DATE_EPOCH"',
             "_", str(REPO / "runtime/android-build-env.sh")],
            cwd=REPO, text=True, stdout=subprocess.PIPE, env={**os.environ, "SOURCE_DATE_EPOCH": ""},
        )
        self.assertEqual(resolved.stdout, pinned.group(1))
        staged = REPO / "android/app/src/main/jniLibs/arm64-v8a/libpocketclaw-python.so"
        if staged.is_file():
            stamp = time.strftime("%b %e %Y", time.gmtime(int(pinned.group(1)))).encode()
            self.assertIn(stamp, staged.read_bytes())

    def test_private_support_rejects_names_that_escape_the_private_root(self) -> None:
        support = importlib.import_module("native_support")
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            for name in ("../escape", "nested/name.so", "/absolute", "", ".", ".."):
                with self.assertRaises(RuntimeError, msg=name):
                    support.support_path(root, name)
            self.assertEqual(support.support_path(root, "libpocketclaw.so"),
                             root / "libpocketclaw.so.debug")

    def test_malformed_private_manifest_is_reported_not_overwritten(self) -> None:
        support = importlib.import_module("native_support")
        with tempfile.TemporaryDirectory() as temporary:
            manifest = Path(temporary) / "manifest.json"
            for broken in ("{ not json", '{"artifacts": 7}', '{"artifacts": [{"x": 1}]}'):
                manifest.write_text(broken)
                with self.assertRaises(RuntimeError):
                    support.update_manifest(manifest, {"logicalName": "libsample.so"})
                self.assertEqual(manifest.read_text(), broken)

    def test_apk_binding_command_is_private_manifest_only(self) -> None:
        helper = (REPO / "tool/native_support.py").read_text()
        self.assertIn('sys.argv[1] == "bind-apk"', helper)
        self.assertIn('"logicalPath": "build/app/outputs/apk/release/app-release.apk"', helper)


if __name__ == "__main__":
    unittest.main(verbosity=2)
