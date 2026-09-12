#!/usr/bin/env python3
"""Create and verify private native debug companions for shipped Android ELFs.

The shipped binary is never modified here. Build recipes first derive their
stripped payload, then call this tool with the matching unstripped input. The
private manifest uses paths relative to its own directory and is intentionally
kept below the ignored build/private-symbols tree.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import subprocess
import tempfile
import sys
from pathlib import Path


SAFE_LOGICAL_NAME = re.compile(r"[A-Za-z0-9][A-Za-z0-9._+-]*\Z")


def support_path(output_root: Path, logical_name: str) -> Path:
    """Resolves the private companion path for one shipped payload.

    The logical name reaches this tool from a build recipe, but it also reaches
    the audit tool back out of the manifest, so it is constrained to a single
    ordinary filename here rather than trusted. A name containing a separator,
    a parent reference or a leading dot would place the companion outside the
    private root and make the manifest's relative path a lie.
    """
    if not SAFE_LOGICAL_NAME.fullmatch(logical_name):
        raise RuntimeError(f"logical name is not a plain file name: {logical_name!r}")
    candidate = (output_root / f"{logical_name}.debug").resolve()
    if candidate.parent != output_root:
        raise RuntimeError(f"private support path escapes {output_root}: {candidate}")
    return candidate


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def run(argv: list[str]) -> str:
    return subprocess.run(argv, check=True, stdout=subprocess.PIPE,
                          stderr=subprocess.STDOUT, text=True,
                          errors="replace").stdout


def build_id(readelf: str, path: Path) -> str | None:
    match = re.search(r"Build ID:\s*([0-9a-fA-F]+)", run([readelf, "-nW", str(path)]))
    return match.group(1).lower() if match else None


def section_names(readelf: str, path: Path) -> set[str]:
    output = run([readelf, "-SW", str(path)])
    return set(re.findall(r"\[\s*\d+\]\s+([^\s]+)", output))


def symbol_address(nm: str, support: Path, requested: str) -> tuple[str, str]:
    candidates: list[tuple[str, str]] = []
    for line in run([nm, "-n", "--defined-only", str(support)]).splitlines():
        fields = line.split(maxsplit=2)
        if len(fields) == 3 and re.fullmatch(r"[0-9a-fA-F]+", fields[0]):
            candidates.append((fields[0], fields[2]))
    for address, name in candidates:
        if name == requested:
            return address, name
    raise RuntimeError(f"representative symbol {requested!r} is absent from {support}")


def update_manifest(path: Path, entry: dict[str, object]) -> None:
    if path.exists():
        try:
            manifest = json.loads(path.read_text(encoding="utf-8"))
            existing = manifest["artifacts"]
            artifacts = {item["logicalName"]: item for item in existing}
        except (json.JSONDecodeError, KeyError, TypeError) as error:
            raise RuntimeError(
                f"existing private manifest is malformed; move or delete it: {path} ({error})"
            ) from error
    else:
        manifest = {"schemaVersion": 1, "architecture": "android-arm64", "artifacts": []}
        artifacts = {}
    artifacts[entry["logicalName"]] = entry
    manifest["artifacts"] = [artifacts[name] for name in sorted(artifacts)]
    path.parent.mkdir(parents=True, exist_ok=True)
    fd, temporary = tempfile.mkstemp(prefix=f".{path.name}.", dir=path.parent)
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as stream:
            json.dump(manifest, stream, indent=2, sort_keys=True)
            stream.write("\n")
        os.replace(temporary, path)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)


def capture(args: argparse.Namespace) -> None:
    source, shipped = args.source.resolve(), args.shipped.resolve()
    if not source.is_file() or not shipped.is_file():
        raise RuntimeError("unstripped source and shipped ELF must both exist")
    output_root = args.output_root.resolve()
    output_root.mkdir(parents=True, exist_ok=True)
    support = support_path(output_root, args.logical_name)
    run([args.objcopy, "--only-keep-debug", str(source), str(support)])

    support_sections = section_names(args.readelf, support)
    useful = any(name.startswith((".debug_", ".zdebug_")) for name in support_sections)
    if not useful:
        raise RuntimeError(f"private support file has no source debug sections: {support}")
    shipped_sections = section_names(args.readelf, shipped)
    leaked = sorted(name for name in shipped_sections
                    if name.startswith((".debug_", ".zdebug_")) or name in {".symtab", ".strtab"})
    if leaked:
        raise RuntimeError(f"shipped ELF retains private debug/static-symbol sections: {leaked}")

    source_id, shipped_id, support_id = (
        build_id(args.readelf, source), build_id(args.readelf, shipped), build_id(args.readelf, support)
    )
    present_ids = {item for item in (source_id, shipped_id, support_id) if item}
    if len(present_ids) > 1:
        raise RuntimeError(f"build ID mismatch across source/shipped/support: {sorted(present_ids)}")

    address, symbol = symbol_address(args.nm, support, args.probe_symbol)
    resolved = run([args.addr2line, "-f", "-C", "-e", str(support), f"0x{address}"]).splitlines()
    if not resolved or resolved[0].strip() in {"", "??"}:
        raise RuntimeError(f"private support file cannot symbolize {symbol} at 0x{address}")

    manifest_path = output_root / "manifest.json"
    update_manifest(manifest_path, {
        "logicalName": args.logical_name,
        "category": args.category,
        "toolchain": args.toolchain,
        "sourceInput": args.source_id,
        "shippedSha256": sha256(shipped),
        "shippedSizeBytes": shipped.stat().st_size,
        "buildId": shipped_id,
        "supportPath": support.relative_to(output_root).as_posix(),
        "supportSha256": sha256(support),
        "supportSizeBytes": support.stat().st_size,
        "symbolization": {
            "address": f"0x{address}", "requestedSymbol": args.probe_symbol,
            "resolvedFunction": resolved[0].strip(),
            "resolvedLocation": resolved[1].strip() if len(resolved) > 1 else "",
        },
    })
    print(f"  private support: {support} ({support.stat().st_size} bytes, {sha256(support)})")
    print(f"  symbolization:   0x{address} -> {resolved[0].strip()} ({resolved[1].strip() if len(resolved) > 1 else ''})")


def bind_apk(argv: list[str]) -> int:
    parser = argparse.ArgumentParser(description="Bind a private native manifest to an exact validation APK")
    parser.add_argument("--output-root", type=Path, required=True)
    parser.add_argument("--apk", type=Path, required=True)
    args = parser.parse_args(argv)
    manifest_path = args.output_root.resolve() / "manifest.json"
    if not manifest_path.is_file() or not args.apk.is_file():
        raise RuntimeError("private native manifest and APK must both exist")
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    manifest["apk"] = {
        "logicalPath": "build/app/outputs/apk/release/app-release.apk",
        "sizeBytes": args.apk.stat().st_size,
        "sha256": sha256(args.apk),
    }
    fd, temporary = tempfile.mkstemp(prefix=f".{manifest_path.name}.", dir=manifest_path.parent)
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as stream:
            json.dump(manifest, stream, indent=2, sort_keys=True)
            stream.write("\n")
        os.replace(temporary, manifest_path)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)
    print(f"  native support manifest bound to APK SHA-256 {manifest['apk']['sha256']}")
    return 0


def main() -> int:
    if len(sys.argv) > 1 and sys.argv[1] == "bind-apk":
        return bind_apk(sys.argv[2:])
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--shipped", type=Path, required=True)
    parser.add_argument("--output-root", type=Path, required=True)
    parser.add_argument("--logical-name", required=True)
    parser.add_argument("--category", required=True)
    parser.add_argument("--toolchain", required=True)
    parser.add_argument("--source-id", required=True)
    parser.add_argument("--probe-symbol", required=True)
    parser.add_argument("--objcopy", required=True)
    parser.add_argument("--readelf", required=True)
    parser.add_argument("--nm", required=True)
    parser.add_argument("--addr2line", required=True)
    capture(parser.parse_args())
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
