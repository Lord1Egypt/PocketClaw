#!/usr/bin/env python3
"""Focused tests for the H3A hardened Android build contract."""

import importlib.util
import json
import sys
import tempfile
import unittest
import zipfile
from pathlib import Path


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
            if packaged_symbol:
                archive.writestr("assets/private-symbols/app.android-arm64.symbols", symbol_bytes)
        return apk, symbols

    def test_accepts_obfuscated_aot_with_external_split_info(self):
        with tempfile.TemporaryDirectory() as tmp:
            apk, symbols = self.make_outputs(Path(tmp))
            evidence = hardening.inspect_hardened_outputs(apk, symbols)
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
            apk, symbols = self.make_outputs(Path(tmp), app=bad)
            with self.assertRaises(hardening.HardeningError):
                hardening.inspect_hardened_outputs(apk, symbols)

    def test_refuses_private_symbols_packaged_in_apk(self):
        with tempfile.TemporaryDirectory() as tmp:
            apk, symbols = self.make_outputs(Path(tmp), packaged_symbol=True)
            with self.assertRaises(hardening.HardeningError):
                hardening.inspect_hardened_outputs(apk, symbols)


if __name__ == "__main__":
    unittest.main(verbosity=2)
