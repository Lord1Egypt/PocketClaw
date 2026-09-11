#!/usr/bin/env python3
"""Build and verify PocketClaw's hardened Android release variant.

This is the one supported entry point for a Dart-hardened release build.  It
does not carry signing secrets: production credentials remain environment-only,
and local validation explicitly opts into the development signer.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import shlex
import shutil
import subprocess
import sys
import zipfile
from pathlib import Path

from r8_contract import MAPPING as R8_MAPPING
from r8_contract import R8ContractError, inspect_outputs as inspect_r8_outputs
from r8_contract import source_contract as inspect_r8_source_contract


REPO = Path(__file__).resolve().parent.parent
ANDROID = REPO / "android"
PACKAGE_CONFIG = REPO / ".dart_tool/package_config.json"
FLUTTER_BUILD_DIR = REPO / ".dart_tool/flutter_build"
APK = REPO / "build/app/outputs/apk/release/app-release.apk"
DEFAULT_SYMBOLS_DIR = Path("build/private-symbols/dart/android-arm64")
GENERATED_PACKAGE = {
    "name": "pocketclaw_generated",
    "rootUri": "flutter_build/",
    "packageUri": "./",
}
GENERATED_REGISTRANT_URI = b"package:pocketclaw_generated/dart_plugin_registrant.dart"
HOST_PATH_MARKERS = (b"/home/", b"/Users/", b"/root/")
APP_SYMBOL_MARKERS = (
    b"TelegramOnboardingController",
    b"TelegramOnboardingClient",
    b"_MainShellState",
    b"StatusSnapshot",
)
SIGNING_ENV = ("KEYSTORE_PATH", "KEYSTORE_PASSWORD", "KEY_ALIAS", "KEY_PASSWORD")


class HardeningError(RuntimeError):
    pass


def sha256_bytes(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1 << 20), b""):
            digest.update(chunk)
    return digest.hexdigest()


def prepare_generated_source_package(path: Path = PACKAGE_CONFIG) -> bool:
    """Map Flutter's generated registrant to a stable package URI.

    Flutter 3.47.1 converts an additional source to a package URI when that
    source belongs to the active package config.  Pub does not include
    `.dart_tool/flutter_build`, so this deterministic generated-only entry fills
    that gap.  It changes generated metadata, never pubspec or the lockfile.
    """
    if not path.is_file():
        raise HardeningError(
            f"{path} is missing; run the pinned `flutter pub get` before the hardened build."
        )
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        raise HardeningError(f"cannot read valid package config {path}: {error}") from error

    packages = data.get("packages")
    if not isinstance(packages, list):
        raise HardeningError(f"{path} has no package list")
    if not all(isinstance(item, dict) for item in packages):
        raise HardeningError(f"{path} contains a malformed package entry")
    root = next((item for item in packages if item.get("name") == "pocketclaw"), None)
    if not isinstance(root, dict) or not isinstance(root.get("languageVersion"), str):
        raise HardeningError(f"{path} has no PocketClaw language version")

    existing = [item for item in packages if item.get("name") == GENERATED_PACKAGE["name"]]
    if len(existing) > 1:
        raise HardeningError(f"{path} contains duplicate {GENERATED_PACKAGE['name']} entries")
    expected = {**GENERATED_PACKAGE, "languageVersion": root["languageVersion"]}
    if existing:
        if existing[0] != expected:
            raise HardeningError(
                f"{path} contains a conflicting {GENERATED_PACKAGE['name']} mapping"
            )
        return False

    packages.append(expected)
    temporary = path.with_name(f"{path.name}.h3a.tmp")
    try:
        temporary.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")
        temporary.replace(path)
    finally:
        if temporary.exists():
            temporary.unlink()
    return True


def resolve_symbols_dir(raw: str) -> tuple[str, Path]:
    candidate = Path(raw)
    if not raw.strip():
        raise HardeningError("the split-debug-info directory is empty")
    resolved = (candidate if candidate.is_absolute() else REPO / candidate).resolve()
    if resolved == REPO or resolved in REPO.parents:
        raise HardeningError("the split-debug-info directory cannot be the repository or its parent")
    if REPO in resolved.parents:
        relative = resolved.relative_to(REPO)
        allowed = (
            relative.parts[:2] == ("build", "private-symbols")
            or relative.parts[:1] == ("split-debug-info",)
            or relative.parts[:1] == ("symbols",)
        )
        if not allowed:
            raise HardeningError(
                "a repository-local split-debug-info directory must be under "
                "build/private-symbols/, split-debug-info/, or symbols/"
            )
    # Preserve a relative property value so the canonical command contains no
    # checkout-specific absolute path.  Absolute external roots remain useful
    # for private archives and path-sensitivity tests.
    return (raw if not candidate.is_absolute() else str(resolved), resolved)


def reset_generated_build_outputs(
    apk: Path = APK,
    symbols: Path | None = None,
    flutter_build_dir: Path = FLUTTER_BUILD_DIR,
    r8_mapping: Path | None = None,
) -> None:
    """Force Flutter to regenerate AOT and its external DWARF as one pair.

    Flutter 3.47.1's incremental build cache tracks ``app.so`` but not the
    split-debug-info file written beside the build.  Reusing that cache after
    deleting the private DWARF therefore produces a valid AOT library without
    recreating its required symbol companion.  A hardened build clears only
    Flutter's generated build cache before assembly so ``gen_snapshot`` must
    emit both outputs again.
    """
    if flutter_build_dir.is_symlink():
        raise HardeningError(f"refusing to clear symlinked Flutter build cache: {flutter_build_dir}")
    if flutter_build_dir.exists():
        if not flutter_build_dir.is_dir():
            raise HardeningError(f"Flutter build cache is not a directory: {flutter_build_dir}")
        shutil.rmtree(flutter_build_dir)
    if symbols is not None and symbols.exists():
        symbols.unlink()
    if apk.exists():
        apk.unlink()
    # Do not let a previous release satisfy the post-build R8 assertions.
    if r8_mapping is not None and r8_mapping.parent.exists():
        for report in ("mapping.txt", "usage.txt", "seeds.txt", "configuration.txt"):
            candidate = r8_mapping.with_name(report)
            if candidate.exists():
                candidate.unlink()


def gradle_command(signing: str, symbols_property: str) -> list[str]:
    command = [
        str(ANDROID / "gradlew"),
        ":app:assembleRelease",
        "-Ptarget-platform=android-arm64",
        "-PpocketclawDartHardening=true",
        "-Pdart-obfuscation=true",
        f"-Psplit-debug-info={symbols_property}",
    ]
    if signing == "local-test":
        command.append("-PallowDebugSigning=true")
    return command


def validate_signing_environment(signing: str, environ: dict[str, str]) -> None:
    declared = [name for name in SIGNING_ENV if environ.get(name, "")]
    if signing == "local-test" and declared:
        raise HardeningError(
            "LOCAL TEST mode refuses declared production signing variables: "
            + ", ".join(declared)
        )
    if signing == "production" and len(declared) != len(SIGNING_ENV):
        missing = [name for name in SIGNING_ENV if name not in declared]
        raise HardeningError(
            "production signing is incomplete; missing variable names: " + ", ".join(missing)
        )


def inspect_hardened_outputs(
    apk: Path,
    symbols: Path,
    classification: str = "LOCAL TEST / NON-RELEASABLE",
    r8_mapping: Path = R8_MAPPING,
) -> dict[str, object]:
    if not apk.is_file():
        raise HardeningError(f"Gradle completed without producing {apk}")
    if not symbols.is_file() or symbols.stat().st_size == 0:
        raise HardeningError(f"split debug info was not produced at {symbols}")

    symbol_bytes = symbols.read_bytes()
    if not symbol_bytes.startswith(b"\x7fELF"):
        raise HardeningError(f"{symbols} is not an ELF split-debug-info artifact")
    if b".debug_info" not in symbol_bytes or b".debug_line" not in symbol_bytes:
        raise HardeningError(f"{symbols} lacks expected Dart DWARF sections")
    retained = [marker.decode() for marker in APP_SYMBOL_MARKERS if marker in symbol_bytes]
    if len(retained) < 3:
        raise HardeningError(
            "split debug info does not retain enough known application symbols to support deobfuscation"
        )

    with zipfile.ZipFile(apk) as archive:
        names = archive.namelist()
        app_entry = "lib/arm64-v8a/libapp.so"
        if app_entry not in names:
            raise HardeningError(f"{apk} does not package {app_entry}")
        app = archive.read(app_entry)
        packaged_symbols = [
            name for name in names
            if name.endswith((".symbols", ".dwarf")) or "private-symbols" in name
        ]
    if packaged_symbols:
        raise HardeningError("private Dart symbols were packaged: " + ", ".join(packaged_symbols))
    leaked = [marker.decode() for marker in HOST_PATH_MARKERS if marker in app]
    if leaked:
        raise HardeningError("packaged Dart AOT contains host-specific path markers: " + ", ".join(leaked))
    if GENERATED_REGISTRANT_URI not in app:
        raise HardeningError("packaged Dart AOT lacks the controlled generated-source package URI")
    exposed = [marker.decode() for marker in APP_SYMBOL_MARKERS if marker in app]
    if exposed:
        raise HardeningError("Dart obfuscation did not remove application symbols: " + ", ".join(exposed))

    evidence = {
        "apk": str(apk),
        "apkBytes": apk.stat().st_size,
        "apkSha256": sha256_file(apk),
        "dartAotSha256": sha256_bytes(app),
        "symbols": str(symbols),
        "symbolsBytes": symbols.stat().st_size,
        "symbolsSha256": sha256_file(symbols),
        "retainedPrivateMarkers": len(retained),
        "generatedSourceUri": GENERATED_REGISTRANT_URI.decode(),
        "classification": classification,
    }
    evidence.update(inspect_r8_outputs(apk, r8_mapping))
    return evidence


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--signing", required=True, choices=("local-test", "production"))
    parser.add_argument(
        "--symbols-dir",
        default=str(DEFAULT_SYMBOLS_DIR),
        help="private split-debug-info directory (default: %(default)s)",
    )
    parser.add_argument("--clean", action="store_true", help="run :app:clean before assembly")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    try:
        inspect_r8_source_contract()
        validate_signing_environment(args.signing, dict(os.environ))
        symbols_property, symbols_dir = resolve_symbols_dir(args.symbols_dir)
        changed = prepare_generated_source_package()
        symbols = symbols_dir / "app.android-arm64.symbols"
        reset_generated_build_outputs(APK, symbols, r8_mapping=R8_MAPPING)
        symbols_dir.mkdir(parents=True, exist_ok=True)

        environment = dict(os.environ)
        if args.signing == "local-test":
            for name in SIGNING_ENV:
                environment.pop(name, None)

        if changed:
            print(
                "Prepared stable package:pocketclaw_generated mapping in generated package config.",
                flush=True,
            )
        print(
            "Invalidated generated Flutter build cache; AOT and private symbols will be regenerated.",
            flush=True,
        )
        classification = (
            "LOCAL TEST / NON-RELEASABLE" if args.signing == "local-test" else "PRODUCTION"
        )
        print(
            "Signing classification: " + classification,
            flush=True,
        )
        if args.clean:
            clean = [str(ANDROID / "gradlew"), ":app:clean"]
            print("Build step: " + shlex.join(clean), flush=True)
            subprocess.run(clean, cwd=ANDROID, env=environment, check=True)
        command = gradle_command(args.signing, symbols_property)
        print("Build step: " + shlex.join(command), flush=True)
        subprocess.run(command, cwd=ANDROID, env=environment, check=True)
        evidence = inspect_hardened_outputs(APK, symbols, classification)
        print(json.dumps(evidence, indent=2, sort_keys=True))
        return 0
    except (HardeningError, R8ContractError, subprocess.CalledProcessError) as error:
        print(f"Hardened Android build FAILED: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
