#!/usr/bin/env python3
"""Read-only native ELF audit for PocketClaw Android artifacts.

The audit classifies every packaged ELF before applying category-aware release
checks. It never rewrites an APK or ELF. Without --enforce-target, findings are
reported for planning; with it, failed release-policy checks return non-zero.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import re
import shutil
import subprocess
import sys
import tempfile
import zipfile
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Iterable


EXPECTED_ELF_ENTRIES = {
    "lib/arm64-v8a/libapp.so", "lib/arm64-v8a/libdartjni.so",
    "lib/arm64-v8a/libdatastore_shared_counter.so", "lib/arm64-v8a/libflutter.so",
    "lib/arm64-v8a/libpocketclaw-curl.so", "lib/arm64-v8a/libpocketclaw-gh.so",
    "lib/arm64-v8a/libpocketclaw-git-remote-http.so", "lib/arm64-v8a/libpocketclaw-git.so",
    "lib/arm64-v8a/libpocketclaw-jq.so", "lib/arm64-v8a/libpocketclaw-python.so",
    "lib/arm64-v8a/libpocketclaw-rg.so", "lib/arm64-v8a/libpocketclaw-sqlite3.so",
    "lib/arm64-v8a/libpocketclaw-web.so", "lib/arm64-v8a/libpocketclaw.so",
    "lib/armeabi-v7a/libdartjni.so", "lib/armeabi-v7a/libdatastore_shared_counter.so",
    "lib/x86_64/libdartjni.so", "lib/x86_64/libdatastore_shared_counter.so",
}

RUNTIME_EXECUTABLES = {
    "libpocketclaw-curl.so", "libpocketclaw-gh.so",
    "libpocketclaw-git-remote-http.so", "libpocketclaw-git.so",
    "libpocketclaw-jq.so", "libpocketclaw-python.so",
    "libpocketclaw-rg.so", "libpocketclaw-sqlite3.so",
}
CORE_EXECUTABLES = {"libpocketclaw.so", "libpocketclaw-web.so"}
LIBAPP_EXPORTS = {"_kDartSnapshotBuildId", "_kDartSnapshotData", "_kDartSnapshotText"}
DARTJNI_REQUIRED_EXPORTS = {
    "GetGlobalEnv", "InitDartApiDL", "JNI_OnUnload",
    "Java_com_github_dart_1lang_jni_JniPlugin_setClassLoader",
    "Java_com_github_dart_1lang_jni_JniUtils_fromReferenceAddress",
    "Java_com_github_dart_1lang_jni_PortCleaner_clean",
    "Java_com_github_dart_1lang_jni_PortContinuation__1resumeWith",
    "Java_com_github_dart_1lang_jni_PortProxyBuilder__1cleanUp",
    "Java_com_github_dart_1lang_jni_PortProxyBuilder__1invoke",
}
PROHIBITED_PATH_MARKERS = (
    "/home/lordegypt", "PocketClaw-App", "/tmp/pocketclaw-runtime-build",
)
PROHIBITED_TEMP_BUILD_RE = re.compile(r"/tmp/pocketclaw-(?:h5b|native|runtime)[A-Za-z0-9_./+-]*")
PRIVATE_SUPPORT_SUFFIXES = (".symbols", ".dwarf", ".debug", ".dbg", ".sym", "mapping.txt")


@dataclass
class Check:
    name: str
    status: str
    detail: str


def sha256_bytes(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def classify_elf(entry: str) -> str:
    name = Path(entry).name
    if name == "libapp.so":
        return "A: Dart / Flutter AOT"
    if name == "libdartjni.so":
        return "B: PocketClaw JNI bridge"
    if name in CORE_EXECUTABLES:
        return "C: PocketClaw Core executable payload"
    if name in RUNTIME_EXECUTABLES:
        return "D: Managed Runtime executable payload"
    return "E: plugin ABI stub / dependency-native payload"


def expected_machine(entry: str) -> tuple[int, str]:
    if "/armeabi-v7a/" in entry:
        return (32, "ARM")
    if "/x86_64/" in entry:
        return (64, "Advanced Micro Devices X86-64")
    return (64, "AArch64")


def parse_elf_header(output: str) -> dict[str, object]:
    def value(label: str) -> str:
        match = re.search(rf"^\s*{re.escape(label)}:\s*(.+)$", output, re.MULTILINE)
        return match.group(1).strip() if match else ""
    class_text = value("Class")
    entry_text = value("Entry point address")
    return {
        "classBits": 64 if class_text == "ELF64" else 32 if class_text == "ELF32" else 0,
        "machine": value("Machine"),
        "type": value("Type").split()[0],
        "entryPoint": int(entry_text, 0) if entry_text else 0,
    }


def parse_program_headers(output: str) -> dict[str, object]:
    load_segments: list[dict[str, object]] = []
    stack_flags = ""
    has_relro = False
    interpreter = ""
    for line in output.splitlines():
        stripped = line.strip()
        if "Requesting program interpreter:" in stripped:
            interpreter = stripped.split(":", 1)[1].rstrip("]").strip()
        fields = stripped.split()
        if not fields:
            continue
        if fields[0] == "GNU_RELRO":
            has_relro = True
        if fields[0] not in {"LOAD", "GNU_STACK"} or len(fields) < 8:
            continue
        try:
            alignment = int(fields[-1], 0)
        except ValueError:
            continue
        flags = "".join(fields[6:-1])
        if fields[0] == "LOAD":
            load_segments.append({"flags": flags, "alignment": alignment})
        else:
            stack_flags = flags
    return {
        "loadSegments": load_segments, "gnuStackFlags": stack_flags,
        "gnuRelro": has_relro, "interpreter": interpreter,
    }


def parse_sections(output: str) -> list[str]:
    return re.findall(r"\[\s*\d+\]\s+([^\s]+)", output)


def parse_dynamic(output: str) -> dict[str, object]:
    needed = re.findall(r"\(NEEDED\).*?\[(.*?)\]", output)
    soname = re.findall(r"\(SONAME\).*?\[(.*?)\]", output)
    rpath = re.findall(r"\(RPATH\).*?\[(.*?)\]", output)
    runpath = re.findall(r"\(RUNPATH\).*?\[(.*?)\]", output)
    return {
        "needed": needed, "soname": soname[0] if soname else None,
        "rpath": rpath, "runpath": runpath,
        "runtimeSearchPaths": rpath + runpath,
        "bindNow": "(BIND_NOW)" in output or bool(re.search(r"\(FLAGS(?:_1)?\).*?\bNOW\b", output)),
        "textRel": "(TEXTREL)" in output,
    }


def parse_exported_symbols(output: str) -> list[str]:
    exports: set[str] = set()
    for line in output.splitlines():
        fields = line.split(maxsplit=7)
        if len(fields) != 8 or not fields[0].rstrip(":").isdigit():
            continue
        _number, _value, _size, _kind, binding, visibility, index, name = fields
        if binding not in {"GLOBAL", "WEAK"} or visibility not in {"DEFAULT", "PROTECTED"} or index == "UND":
            continue
        clean_name = name.split("@", 1)[0]
        if clean_name:
            exports.add(clean_name)
    return sorted(exports)


def parse_build_id(output: str) -> str | None:
    match = re.search(r"Build ID:\s*([0-9a-fA-F]+)", output)
    return match.group(1).lower() if match else None


def collect_interesting_strings(output: str) -> dict[str, list[str]]:
    values = [line.strip() for line in output.splitlines() if line.strip()]
    prohibited = sorted({line for line in values
                         if any(marker in line for marker in PROHIBITED_PATH_MARKERS)
                         or PROHIBITED_TEMP_BUILD_RE.search(line)})
    host_paths = sorted({line for line in values if any(marker in line for marker in ("/home/", "/Users/", "/root/"))})
    absolute = sorted({line for line in values if re.search(r"(?:^|\s)/(?:home|tmp|root|mnt|Users)/[^\s]+", line)})
    source_paths = sorted({
        match.group(0) for line in values
        for match in re.finditer(r"[A-Za-z0-9_./@+~-]+\.(?:c|cc|cpp|cxx|h|hpp|go|rs)(?::\d+)?", line)
        if "/" in match.group(0)
    })
    return {
        "prohibitedBuildPaths": prohibited[:50],
        "hostPathSamples": host_paths[:50],
        "absolutePathSamples": absolute[:50],
        "sourcePathSamples": source_paths[:50],
    }


def find_tool(preferred: Iterable[str]) -> str:
    for candidate in preferred:
        located = shutil.which(candidate)
        if located:
            return located
    raise RuntimeError(f"required ELF tool not found: {' or '.join(preferred)}")


def run_tool(argv: list[str]) -> str:
    completed = subprocess.run(argv, check=True, stdout=subprocess.PIPE,
                               stderr=subprocess.STDOUT, text=True, errors="replace")
    return completed.stdout


def inspect_elf(entry: str, data: bytes, work_root: Path, tools: dict[str, str]) -> dict[str, object]:
    local_path = work_root / entry
    local_path.parent.mkdir(parents=True, exist_ok=True)
    local_path.write_bytes(data)
    local_path.chmod(0o600)
    header = parse_elf_header(run_tool([tools["readelf"], "-hW", str(local_path)]))
    program = parse_program_headers(run_tool([tools["readelf"], "-lW", str(local_path)]))
    sections = parse_sections(run_tool([tools["readelf"], "-SW", str(local_path)]))
    dynamic = parse_dynamic(run_tool([tools["readelf"], "-dW", str(local_path)]))
    relocations = run_tool([tools["readelf"], "-rW", str(local_path)])
    symbols = parse_exported_symbols(run_tool([tools["readelf"], "--dyn-syms", "-W", str(local_path)]))
    notes = run_tool([tools["readelf"], "-nW", str(local_path)])
    strings_info = collect_interesting_strings(run_tool([tools["strings"], "-a", str(local_path)]))
    comments = run_tool([tools["readelf"], "-p", ".comment", str(local_path)]).strip() if ".comment" in sections else ""
    debug_sections = sorted(section for section in sections if section.startswith((".debug_", ".zdebug_")))
    loads = program["loadSegments"]
    return {
        "entry": entry, "name": Path(entry).name, "category": classify_elf(entry),
        "sizeBytes": len(data), "sha256": sha256_bytes(data),
        "fileDescription": run_tool([tools["file"], "-b", str(local_path)]).strip(),
        **header, "interpreter": program["interpreter"], "loadSegments": loads,
        "minimumLoadAlignment": min((int(segment["alignment"]) for segment in loads), default=0),
        "gnuStackFlags": program["gnuStackFlags"],
        "hasWritableExecutableSegment": any("W" in str(segment["flags"]) and "E" in str(segment["flags"]) for segment in loads),
        "gnuRelro": program["gnuRelro"], **dynamic,
        "hasLazyBindingRelocations": bool(re.search(r"(?:JUMP_SLOT|JMP_SLOT)", relocations)),
        "debugSections": debug_sections, "hasSymtab": ".symtab" in sections,
        "hasStrtab": ".strtab" in sections, "hasDynsym": ".dynsym" in sections,
        "hasDynstr": ".dynstr" in sections, "exportedDynamicSymbolCount": len(symbols),
        "exportedDynamicSymbols": symbols, "buildId": parse_build_id(notes),
        "comment": comments, **strings_info,
    }


def evaluate_record(record: dict[str, object]) -> list[Check]:
    entry, name = str(record["entry"]), str(record["name"])
    expected_bits, expected_arch = expected_machine(entry)
    executable = name in CORE_EXECUTABLES | RUNTIME_EXECUTABLES
    identity_ok = (record["type"] == "DYN" and record["classBits"] == expected_bits
                   and record["machine"] == expected_arch
                   and (int(record["entryPoint"]) != 0) == executable)
    loads = list(record["loadSegments"])
    alignment_ok = bool(loads) and all(
        int(segment["alignment"]) >= 0x4000
        and int(segment["alignment"]) % 0x4000 == 0
        for segment in loads
    )
    stack_flags = str(record["gnuStackFlags"])
    bind_required = bool(record["needed"] or record["hasLazyBindingRelocations"])
    if name == "libapp.so":
        relocation_ok = not bind_required
    elif bind_required:
        relocation_ok = bool(record["gnuRelro"]) and bool(record["bindNow"])
    else:
        relocation_ok = bool(record["gnuRelro"])
    checks = [
        Check(f"{entry}.identity", "PASS" if identity_ok else "FAIL", f"{record['classBits']}-bit {record['machine']} {record['type']}; entry={record['entryPoint']:#x}"),
        Check(f"{entry}.load_alignment", "PASS" if alignment_ok else "FAIL", f"minimum PT_LOAD alignment={record['minimumLoadAlignment']:#x}"),
        Check(f"{entry}.nx_stack", "PASS" if stack_flags and "E" not in stack_flags else "FAIL", f"GNU_STACK={stack_flags or 'missing'}"),
        Check(f"{entry}.no_wx_segments", "PASS" if not record["hasWritableExecutableSegment"] else "FAIL", "no writable+executable PT_LOAD" if not record["hasWritableExecutableSegment"] else "writable+executable PT_LOAD present"),
        Check(f"{entry}.no_textrel", "PASS" if not record["textRel"] else "FAIL", "no DT_TEXTREL" if not record["textRel"] else "DT_TEXTREL present"),
        Check(f"{entry}.no_runtime_search_path", "PASS" if not record["runtimeSearchPaths"] else "FAIL", f"RPATH/RUNPATH={record['runtimeSearchPaths']}"),
        Check(f"{entry}.no_debug_sections", "PASS" if not record["debugSections"] else "FAIL", f"debug sections={record['debugSections']}"),
        Check(f"{entry}.no_static_symbol_table", "PASS" if not record["hasSymtab"] and not record["hasStrtab"] else "FAIL", f".symtab={record['hasSymtab']}; .strtab={record['hasStrtab']}"),
        Check(f"{entry}.build_path_privacy", "PASS" if not record["prohibitedBuildPaths"] else "FAIL", f"prohibited path strings={len(record['prohibitedBuildPaths'])}"),
        Check(f"{entry}.relocation_hardening", "PASS" if relocation_ok else "FAIL", f"GNU_RELRO={record['gnuRelro']}; BIND_NOW={record['bindNow']}; imports={len(record['needed'])}; lazy relocations={record['hasLazyBindingRelocations']}"),
    ]
    exports = set(record["exportedDynamicSymbols"])
    if name == "libapp.so":
        checks.append(Check(f"{entry}.required_exports", "PASS" if exports == LIBAPP_EXPORTS else "FAIL", f"exports={sorted(exports)}"))
    elif name in CORE_EXECUTABLES:
        checks.append(Check(f"{entry}.required_exports", "PASS" if exports == {"main.main"} else "FAIL", f"exports={sorted(exports)}"))
    elif name == "libdartjni.so":
        missing = sorted(DARTJNI_REQUIRED_EXPORTS - exports)
        checks.append(Check(f"{entry}.required_exports", "PASS" if not missing else "FAIL", f"required JNI/Dart exports missing={missing}; total exports={len(exports)}"))
    return checks


def support_manifest_checks(manifest_path: Path, apk: Path,
                            records: list[dict[str, object]], tools: dict[str, str]) -> list[Check]:
    expected = CORE_EXECUTABLES | RUNTIME_EXECUTABLES
    try:
        manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
        items = manifest["artifacts"]
        if not isinstance(manifest, dict) or not isinstance(items, list) \
                or not all(isinstance(item, dict) for item in items):
            raise TypeError("artifacts must be a list of objects")
    except (OSError, KeyError, json.JSONDecodeError, TypeError) as error:
        return [Check("native.private_support_manifest", "FAIL", f"cannot read manifest: {error}")]
    names = {str(item.get("logicalName", "")) for item in items}
    checks = [Check("native.private_support_manifest", "PASS" if names == expected else "FAIL",
                    f"entries={len(names)}; missing={sorted(expected - names)}; unexpected={sorted(names - expected)}")]
    record_by_name = {str(record["name"]): record for record in records}
    hashes_ok = binding_ok = symbols_ok = True
    problems: list[str] = []
    root = manifest_path.resolve().parent
    for item in items:
        name = str(item.get("logicalName", ""))
        relative = Path(str(item.get("supportPath", "")))
        support = (root / relative).resolve()
        if relative.is_absolute() or root not in support.parents or not support.is_file():
            hashes_ok = False; problems.append(f"{name}: invalid support path")
            continue
        if sha256_bytes(support.read_bytes()) != item.get("supportSha256") or support.stat().st_size != item.get("supportSizeBytes"):
            hashes_ok = False; problems.append(f"{name}: support hash/size mismatch")
        record = record_by_name.get(name)
        if not record or record["sha256"] != item.get("shippedSha256") or record["sizeBytes"] != item.get("shippedSizeBytes") or record["buildId"] != item.get("buildId"):
            binding_ok = False; problems.append(f"{name}: shipped ELF binding mismatch")
        # A support file the manifest names but readelf cannot parse is a
        # finding, not a crash: this audit must report on whatever the private
        # root actually contains.
        try:
            sections = parse_sections(run_tool([tools["readelf"], "-SW", str(support)]))
        except subprocess.CalledProcessError:
            sections = []
        symbolization = item.get("symbolization")
        if not isinstance(symbolization, dict):
            symbolization = {}
        if not any(section.startswith((".debug_", ".zdebug_")) for section in sections) or not symbolization.get("resolvedFunction") or symbolization.get("resolvedFunction") == "??":
            symbols_ok = False; problems.append(f"{name}: missing debug/symbolization evidence")
    checks.extend([
        Check("native.private_support_hashes", "PASS" if hashes_ok else "FAIL", "; ".join(problems) or "all support hashes and relative paths valid"),
        Check("native.private_support_binding", "PASS" if binding_ok else "FAIL", "all support entries match shipped ELF hash/size/build ID" if binding_ok else "; ".join(problems)),
        Check("native.private_support_symbolization", "PASS" if symbols_ok else "FAIL", "all entries contain debug data and a resolved representative function" if symbols_ok else "; ".join(problems)),
    ])
    apk_info = manifest.get("apk", {})
    checks.append(Check("native.private_support_apk_binding", "PASS" if apk_info.get("sha256") == sha256_bytes(apk.read_bytes()) and apk_info.get("sizeBytes") == apk.stat().st_size else "FAIL", f"manifest APK SHA-256={apk_info.get('sha256')}"))
    repo = Path(__file__).resolve().parents[1]
    candidates = [manifest_path, *(root / str(item.get("supportPath", "")) for item in items)]
    tracked = []
    for candidate in candidates:
        try:
            query = candidate.resolve().relative_to(repo).as_posix()
        except ValueError:
            continue
        result = subprocess.run(["git", "-C", str(repo), "ls-files", "--error-unmatch", query], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        if result.returncode == 0:
            tracked.append(str(candidate))
    checks.append(Check("native.private_support_untracked", "PASS" if not tracked else "FAIL", f"tracked private files={tracked}"))
    return checks


def audit_apk(apk: Path, support_manifest: Path | None = None) -> dict[str, object]:
    tools = {"readelf": find_tool(("llvm-readelf", "readelf")),
             "strings": find_tool(("llvm-strings", "strings")), "file": find_tool(("file",))}
    apk_bytes = apk.read_bytes()
    records: list[dict[str, object]] = []
    checks: list[Check] = []
    with zipfile.ZipFile(apk) as archive:
        elf_entries = {info.filename for info in archive.infolist()
                       if info.filename.startswith("lib/") and archive.read(info)[:4] == b"\x7fELF"}
        unexpected, missing = sorted(elf_entries - EXPECTED_ELF_ENTRIES), sorted(EXPECTED_ELF_ENTRIES - elf_entries)
        checks.append(Check("native.inventory", "PASS" if not unexpected and not missing else "FAIL", f"ELF={len(elf_entries)}; missing={missing}; unexpected={unexpected}"))
        private_entries = sorted(
            info.filename for info in archive.infolist()
            if "private-symbols" in info.filename.lower()
            or any(info.filename.lower().endswith(suffix) for suffix in PRIVATE_SUPPORT_SUFFIXES)
        )
        checks.append(Check("native.private_support_not_packaged", "PASS" if not private_entries else "FAIL", f"private support entries={private_entries}"))
        with tempfile.TemporaryDirectory(prefix="pocketclaw-native-elf-audit-") as work:
            for entry in sorted(elf_entries):
                record = inspect_elf(entry, archive.read(entry), Path(work), tools)
                records.append(record)
                checks.extend(evaluate_record(record))
    if support_manifest is not None:
        checks.extend(support_manifest_checks(support_manifest, apk, records, tools))
    totals = {status: sum(check.status == status for check in checks) for status in ("PASS", "FAIL", "SKIP")}
    return {"schemaVersion": 1, "apk": {"path": str(apk.resolve()), "sizeBytes": len(apk_bytes),
            "sha256": sha256_bytes(apk_bytes)}, "tools": tools, "records": records,
            "checks": [asdict(check) for check in checks], "totals": totals,
            "targetPolicyReady": totals["FAIL"] == 0}


def print_report(manifest: dict[str, object]) -> None:
    print(f"APK: {manifest['apk']['path']}")
    print(f"SHA-256: {manifest['apk']['sha256']}")
    print("ELF inventory:")
    for record in manifest["records"]:
        relro = "full" if record["gnuRelro"] and record["bindNow"] else "static/N/A" if not record["needed"] else "partial"
        print(f"  {record['entry']} | {record['category']} | {record['type']} entry={record['entryPoint']:#x} | NX | RELRO={relro} | debug={len(record['debugSections'])} | exports={record['exportedDynamicSymbolCount']}")
    failed = [check for check in manifest["checks"] if check["status"] == "FAIL"]
    if failed:
        print("Target-policy findings:")
        for check in failed:
            print(f"  FAIL {check['name']}: {check['detail']}")
    totals = manifest["totals"]
    print(f"TOTAL: {totals['PASS']} PASS / {totals['FAIL']} FAIL / {totals['SKIP']} SKIP")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--apk", type=Path, required=True, help="APK to inspect without modifying")
    parser.add_argument("--manifest", type=Path, help="write full JSON evidence to this path")
    parser.add_argument("--enforce-target", action="store_true", help="fail if final native policy is unmet")
    parser.add_argument("--native-support-manifest", type=Path,
                        help="verify the private native support manifest and APK binding")
    args = parser.parse_args()
    if not args.apk.is_file():
        parser.error(f"APK does not exist: {args.apk}")
    manifest = audit_apk(args.apk, args.native_support_manifest)
    if args.manifest:
        args.manifest.parent.mkdir(parents=True, exist_ok=True)
        args.manifest.write_text(json.dumps(manifest, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    print_report(manifest)
    return 1 if args.enforce_target and not manifest["targetPolicyReady"] else 0


if __name__ == "__main__":
    sys.exit(main())
