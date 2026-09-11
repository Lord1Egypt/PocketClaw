#!/usr/bin/env python3
"""Focused unit tests for the read-only PocketClaw native ELF audit."""

from __future__ import annotations

import unittest

import native_elf_audit as audit


class NativeElfAuditTest(unittest.TestCase):
    def test_every_expected_elf_has_a_category(self) -> None:
        self.assertEqual(
            {audit.classify_elf(entry) for entry in audit.EXPECTED_ELF_ENTRIES},
            {
                "A: Dart / Flutter AOT", "B: PocketClaw JNI bridge",
                "C: PocketClaw Core executable payload",
                "D: Managed Runtime executable payload",
                "E: plugin ABI stub / dependency-native payload",
            },
        )

    def test_expected_machine_is_abi_specific(self) -> None:
        self.assertEqual(audit.expected_machine("lib/arm64-v8a/libapp.so"), (64, "AArch64"))
        self.assertEqual(audit.expected_machine("lib/armeabi-v7a/libdartjni.so"), (32, "ARM"))
        self.assertEqual(audit.expected_machine("lib/x86_64/libdartjni.so"), (64, "Advanced Micro Devices X86-64"))

    def test_program_headers_parse_stack_relro_and_load_flags(self) -> None:
        parsed = audit.parse_program_headers(
            "  LOAD 0x0 0x0 0x0 0x10 0x10 R E 0x4000\n"
            "  LOAD 0x20 0x20 0x20 0x10 0x10 RW 0x4000\n"
            "  GNU_STACK 0x0 0x0 0x0 0x0 0x0 RW 0x10\n"
            "  GNU_RELRO 0x20 0x20 0x20 0x10 0x10 R 0x1\n"
            "      [Requesting program interpreter: /system/bin/linker64]\n"
        )
        self.assertEqual(parsed["gnuStackFlags"], "RW")
        self.assertTrue(parsed["gnuRelro"])
        self.assertEqual(parsed["interpreter"], "/system/bin/linker64")
        self.assertEqual(parsed["loadSegments"][0], {"flags": "RE", "alignment": 0x4000})

    def test_dynamic_tags_parse_security_and_dependencies(self) -> None:
        parsed = audit.parse_dynamic(
            "0x1 (NEEDED) Shared library: [libc.so]\n"
            "0xe (SONAME) Library soname: [libtest.so]\n"
            "0x1d (RUNPATH) Library runpath: [/tmp/build/lib]\n"
            "0x18 (BIND_NOW)\n0x16 (TEXTREL)\n"
        )
        self.assertEqual(parsed["needed"], ["libc.so"])
        self.assertEqual(parsed["runtimeSearchPaths"], ["/tmp/build/lib"])
        self.assertEqual(parsed["rpath"], [])
        self.assertEqual(parsed["runpath"], ["/tmp/build/lib"])
        self.assertTrue(parsed["bindNow"])
        self.assertTrue(parsed["textRel"])

    def test_debug_section_detection_handles_debug_and_zdebug(self) -> None:
        sections = audit.parse_sections(
            "[ 1] .text PROGBITS 0 0 0\n[ 2] .debug_info PROGBITS 0 0 0\n"
            "[ 3] .zdebug_line PROGBITS 0 0 0\n"
        )
        self.assertEqual(sections, [".text", ".debug_info", ".zdebug_line"])

    def test_build_path_policy_distinguishes_known_build_root(self) -> None:
        strings = audit.collect_interesting_strings(
            "/tmp/pocketclaw-runtime-build/src/file.c\n/tmp/perf-%jd.map\n"
            "/home/runner/work/upstream/project/file.go\n"
        )
        self.assertEqual(strings["prohibitedBuildPaths"], ["/tmp/pocketclaw-runtime-build/src/file.c"])
        self.assertIn("/tmp/perf-%jd.map", strings["absolutePathSamples"])
        self.assertIn("/home/runner/work/upstream/project/file.go", strings["absolutePathSamples"])
        self.assertEqual(strings["hostPathSamples"], ["/home/runner/work/upstream/project/file.go"])

    def test_export_parser_excludes_undefined_and_local_symbols(self) -> None:
        output = (
            "  1: 00000001 4 FUNC GLOBAL DEFAULT 12 Present\n"
            "  2: 00000000 0 FUNC GLOBAL DEFAULT UND Missing\n"
            "  3: 00000002 4 FUNC LOCAL DEFAULT 12 Local\n"
        )
        self.assertEqual(audit.parse_exported_symbols(output), ["Present"])

    def test_policy_flags_runtime_search_path_and_build_path(self) -> None:
        record = self._passing_record("lib/arm64-v8a/libpocketclaw-git-remote-http.so")
        record["runtimeSearchPaths"] = ["/tmp/pocketclaw-runtime-build/deps/lib"]
        record["prohibitedBuildPaths"] = ["/tmp/pocketclaw-runtime-build/deps/lib"]
        failed = {check.name for check in audit.evaluate_record(record) if check.status == "FAIL"}
        self.assertEqual(failed, {
            "lib/arm64-v8a/libpocketclaw-git-remote-http.so.no_runtime_search_path",
            "lib/arm64-v8a/libpocketclaw-git-remote-http.so.build_path_privacy",
        })

    def test_static_pie_does_not_require_bind_now_without_imports(self) -> None:
        record = self._passing_record("lib/arm64-v8a/libpocketclaw.so")
        record.update({"bindNow": False, "needed": [], "hasLazyBindingRelocations": False})
        checks = {check.name: check for check in audit.evaluate_record(record)}
        self.assertEqual(checks[f"{record['entry']}.relocation_hardening"].status, "PASS")

    def test_core_security_regressions_are_independent_failures(self) -> None:
        record = self._passing_record("lib/arm64-v8a/libpocketclaw.so")
        record.update({
            "gnuStackFlags": "RWE", "hasWritableExecutableSegment": True,
            "textRel": True, "debugSections": [".debug_info"],
        })
        failed = {check.name.rsplit(".", 1)[-1] for check in audit.evaluate_record(record) if check.status == "FAIL"}
        self.assertEqual(failed, {"nx_stack", "no_wx_segments", "no_textrel", "no_debug_sections"})

    def test_private_support_suffix_contract_is_explicit(self) -> None:
        for name in ("app.symbols", "lib.debug", "lib.dwarf", "lib.dbg", "lib.sym", "mapping.txt"):
            self.assertTrue(any(name.endswith(suffix) for suffix in audit.PRIVATE_SUPPORT_SUFFIXES))

    @staticmethod
    def _passing_record(entry: str) -> dict[str, object]:
        name = entry.rsplit("/", 1)[-1]
        if name == "libapp.so":
            exports = sorted(audit.LIBAPP_EXPORTS)
        elif name in audit.CORE_EXECUTABLES:
            exports = ["main.main"]
        elif name == "libdartjni.so":
            exports = sorted(audit.DARTJNI_REQUIRED_EXPORTS)
        else:
            exports = []
        return {
            "entry": entry, "name": name, "type": "DYN", "classBits": 64,
            "machine": "AArch64",
            "entryPoint": 1 if name in audit.CORE_EXECUTABLES | audit.RUNTIME_EXECUTABLES else 0,
            "loadSegments": [{"flags": "RE", "alignment": 0x4000}],
            "minimumLoadAlignment": 0x4000, "gnuStackFlags": "RW",
            "hasWritableExecutableSegment": False, "textRel": False,
            "runtimeSearchPaths": [], "debugSections": [], "hasSymtab": False,
            "hasStrtab": False, "prohibitedBuildPaths": [], "gnuRelro": True,
            "bindNow": True, "needed": ["libc.so"],
            "hasLazyBindingRelocations": False, "exportedDynamicSymbols": exports,
        }


if __name__ == "__main__":
    unittest.main(verbosity=2)
