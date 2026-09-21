#!/usr/bin/env python3
"""Focused regression tests for the H4A R8 hardening contract."""

import tempfile
import unittest
import zipfile
from pathlib import Path

import r8_contract


class SourceContractTest(unittest.TestCase):
    def write(self, root: Path, gradle: str, rules: str) -> tuple[Path, Path]:
        gradle_path = root / "build.gradle.kts"
        rules_path = root / "proguard-rules.pro"
        gradle_path.write_text(gradle, encoding="utf-8")
        rules_path.write_text(rules, encoding="utf-8")
        return gradle_path, rules_path

    @property
    def enabled_gradle(self) -> str:
        return '''
            isMinifyEnabled = true
            isShrinkResources = true
            getDefaultProguardFile("proguard-android-optimize.txt")
            "proguard-rules.pro"
        '''

    def test_repository_contract_is_narrow_and_enabled(self):
        evidence = r8_contract.source_contract()
        self.assertTrue(evidence["minifyEnabled"])
        self.assertTrue(evidence["shrinkResources"])
        self.assertEqual(evidence["activeApplicationRules"], 0)

    def test_refuses_disabled_minification_or_resource_shrinking(self):
        with tempfile.TemporaryDirectory() as tmp:
            gradle, rules = self.write(
                Path(tmp),
                self.enabled_gradle.replace("isMinifyEnabled = true", "isMinifyEnabled = false"),
                "",
            )
            with self.assertRaises(r8_contract.R8ContractError):
                r8_contract.source_contract(gradle, rules)
            gradle.write_text(
                self.enabled_gradle.replace("isShrinkResources = true", "isShrinkResources = false"),
                encoding="utf-8",
            )
            with self.assertRaises(r8_contract.R8ContractError):
                r8_contract.source_contract(gradle, rules)

    def test_refuses_blanket_or_disabling_rules(self):
        bad_rules = (
            "-keep class com.lord1egypt.pocketclaw.** { *; }\n"
            "-keep class io.flutter.** { *; }\n"
            "-dontobfuscate\n"
        )
        with tempfile.TemporaryDirectory() as tmp:
            gradle, rules = self.write(Path(tmp), self.enabled_gradle, bad_rules)
            with self.assertRaises(r8_contract.R8ContractError):
                r8_contract.source_contract(gradle, rules)


class OutputContractTest(unittest.TestCase):
    def fixture(self, root: Path, *, package_mapping=False, rename=True, fold_last=False):
        apk = root / "app-release.apk"
        mapping = root / "mapping.txt"
        lines = [f"{name} -> {name}:" for name in r8_contract.REQUIRED_COMPONENTS]
        internal = r8_contract.INTERNAL_CLASSES[:-1] if fold_last else r8_contract.INTERNAL_CLASSES
        lines += [
            f"{name} -> a{index}.a:" if rename else f"{name} -> {name}:"
            for index, name in enumerate(internal)
        ]
        mapping.write_text("\n".join(lines) + "\n", encoding="utf-8")
        usage = "removed.Class\n"
        if fold_last:
            usage += r8_contract.INTERNAL_CLASSES[-1] + "\n"
        mapping.with_name("usage.txt").write_text(usage, encoding="utf-8")
        with zipfile.ZipFile(apk, "w") as archive:
            archive.writestr("classes.dex", b"dex\n035\0obfuscated")
            if package_mapping:
                archive.writestr("assets/mapping.txt", mapping.read_bytes())
        return apk, mapping

    def test_accepts_private_mapping_with_renamed_implementation_classes(self):
        with tempfile.TemporaryDirectory() as tmp:
            apk, mapping = self.fixture(Path(tmp))
            evidence = r8_contract.inspect_outputs(apk, mapping, Path(tmp))
            self.assertEqual(len(evidence["r8RenamedInternalClasses"]), 6)
            self.assertEqual(evidence["dexCount"], 1)
            self.assertTrue(evidence["r8MappingPrivate"])

    def test_accepts_an_internal_class_removed_by_shrinking(self):
        with tempfile.TemporaryDirectory() as tmp:
            apk, mapping = self.fixture(Path(tmp), fold_last=True)
            evidence = r8_contract.inspect_outputs(apk, mapping, Path(tmp))
            self.assertEqual(
                evidence["r8RemovedOrFoldedInternalClasses"],
                [r8_contract.INTERNAL_CLASSES[-1]],
            )

    def test_refuses_missing_mapping_or_shrinking_report(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            apk, mapping = self.fixture(root)
            mapping.unlink()
            with self.assertRaises(r8_contract.R8ContractError):
                r8_contract.inspect_outputs(apk, mapping, root)
            apk, mapping = self.fixture(root)
            mapping.with_name("usage.txt").unlink()
            with self.assertRaises(r8_contract.R8ContractError):
                r8_contract.inspect_outputs(apk, mapping, root)

    def test_refuses_unobfuscated_application_classes(self):
        with tempfile.TemporaryDirectory() as tmp:
            apk, mapping = self.fixture(Path(tmp), rename=False)
            with self.assertRaises(r8_contract.R8ContractError):
                r8_contract.inspect_outputs(apk, mapping, Path(tmp))

    def test_refuses_mapping_packaged_in_apk(self):
        with tempfile.TemporaryDirectory() as tmp:
            apk, mapping = self.fixture(Path(tmp), package_mapping=True)
            with self.assertRaises(r8_contract.R8ContractError):
                r8_contract.inspect_outputs(apk, mapping, Path(tmp))


if __name__ == "__main__":
    unittest.main(verbosity=2)
