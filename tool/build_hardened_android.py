#!/usr/bin/env python3
"""Build and verify PocketClaw's hardened Android release variant.

This is the one supported entry point for a Dart-hardened release build.  It
does not carry signing secrets: production credentials remain environment-only,
and local validation explicitly opts into the development signer.
"""

from __future__ import annotations

import argparse
import base64
import hashlib
import json
import os
import re
import shlex
import shutil
import subprocess
import sys
import urllib.parse
import zipfile
from pathlib import Path

from r8_contract import MAPPING as R8_MAPPING
from r8_contract import R8ContractError, inspect_outputs as inspect_r8_outputs
from r8_contract import source_contract as inspect_r8_source_contract

from artifact_policy import (
    NON_PUBLISH_AUDIT,
    PLAY_UPLOAD,
    PUBLIC_RELEASE,
    detect_artifact_kind,
    distribution_verdict,
    inspect_bundle,
    packaged_r8_mapping_entries,
    private_material_violations,
)


REPO = Path(__file__).resolve().parent.parent
ANDROID = REPO / "android"
PACKAGE_CONFIG = REPO / ".dart_tool/package_config.json"
FLUTTER_BUILD_DIR = REPO / ".dart_tool/flutter_build"
# Gradle's own view of the Flutter AOT step. Clearing the Dart-side cache is not
# enough on its own: Gradle decides separately whether to run the task at all.
FLUTTER_GRADLE_INTERMEDIATES = REPO / "build/app/intermediates/flutter"
APK = REPO / "build/app/outputs/apk/release/app-release.apk"
# AGP names a release built with no signing config this way.
UNSIGNED_APK = REPO / "build/app/outputs/apk/release/app-release-unsigned.apk"
BUNDLE = REPO / "build/app/outputs/bundle/release/app-release.aab"
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
UNSIGNED_CLASSIFICATION = "UNSIGNED / REPOSITORY-SIGNABLE"
CLASSIFICATIONS = {
    "local-test": "LOCAL TEST / NON-RELEASABLE",
    "production": "PRODUCTION",
    "unsigned": UNSIGNED_CLASSIFICATION,
}
OFFICIAL_ONBOARDING_PROPERTIES = ANDROID / "official-onboarding.properties"
ONBOARDING_PROPERTY = "officialOnboardingBaseUrl"
ONBOARDING_DEFINE = "POCKETCLAW_ONBOARDING_BASE_URL"


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
    flutter_gradle_intermediates: Path | None = None,
) -> None:
    """Force Flutter to regenerate AOT and its external DWARF as one pair.

    Flutter 3.47.1's incremental build cache tracks ``app.so`` but not the
    split-debug-info file written beside the build.  Reusing that cache after
    deleting the private DWARF therefore produces a valid AOT library without
    recreating its required symbol companion.  A hardened build clears only
    Flutter's generated build cache before assembly so ``gen_snapshot`` must
    emit both outputs again.

    PC-DEF-048.  That is necessary and was not sufficient.  Gradle decides
    independently whether to run the Flutter AOT task, and its up-to-date check
    watches the Dart sources -- not the private symbol file, which lives outside
    the project tree by design.  A build whose only changes were Go, Kotlin or
    web-console source therefore skipped the task entirely, emitted no symbols,
    and failed the post-build assertion with nothing to say about why.  Dropping
    Gradle's own Flutter intermediates removes the answer it was relying on.
    """
    if flutter_gradle_intermediates is None:
        flutter_gradle_intermediates = FLUTTER_GRADLE_INTERMEDIATES
    if flutter_build_dir.is_symlink():
        raise HardeningError(f"refusing to clear symlinked Flutter build cache: {flutter_build_dir}")
    if flutter_build_dir.exists():
        if not flutter_build_dir.is_dir():
            raise HardeningError(f"Flutter build cache is not a directory: {flutter_build_dir}")
        shutil.rmtree(flutter_build_dir)
    if flutter_gradle_intermediates.is_symlink():
        raise HardeningError(
            "refusing to clear symlinked Flutter intermediates: "
            f"{flutter_gradle_intermediates}")
    if flutter_gradle_intermediates.is_dir():
        shutil.rmtree(flutter_gradle_intermediates)
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


# One build contract, two packaging tasks. The hardening properties are the
# contract and are identical for both: an AAB that was not obfuscated, shrunk
# and split-debug-stripped the same way the APK is would not be the same product
# in a different container.
GRADLE_TASKS = {
    "apk": ":app:assembleRelease",
    "bundle": ":app:bundleRelease",
}


def validate_onboarding_base_url(url: str) -> str:
    """Accepts only a public https base URL, and says why when it refuses.

    Plain http would put the pairing poll token -- and, once, the child bot
    token -- on the wire in the clear, so a downgrade is refused rather than
    warned about. Credentials in the authority are refused for the same reason
    the token never ships: a URL is compiled into the APK, and anything inside
    it is published with the app.
    """
    if url != url.strip() or not url:
        raise HardeningError(f"{ONBOARDING_PROPERTY} must be a non-empty, untrimmed-free URL")
    parsed = urllib.parse.urlsplit(url)
    if parsed.scheme != "https":
        raise HardeningError(
            f"{ONBOARDING_PROPERTY} must be https, got {parsed.scheme or 'no scheme'!r}: {url}")
    if not parsed.hostname:
        raise HardeningError(f"{ONBOARDING_PROPERTY} has no host: {url}")
    if parsed.username or parsed.password or "@" in parsed.netloc:
        raise HardeningError(
            f"{ONBOARDING_PROPERTY} must not carry credentials; they would ship in the APK")
    return url


def read_official_onboarding_base_url(path: Path = OFFICIAL_ONBOARDING_PROPERTIES) -> str:
    """The tracked official endpoint, so no release depends on a typed flag."""
    if not path.is_file():
        raise HardeningError(
            f"official onboarding configuration is missing at {path}; a production build "
            "cannot decide on its own where PocketClaw's onboarding service lives")
    value = ""
    for line in path.read_text(encoding="utf-8").splitlines():
        stripped = line.strip()
        if not stripped or stripped.startswith("#"):
            continue
        key, sep, raw = stripped.partition("=")
        if sep and key.strip() == ONBOARDING_PROPERTY:
            value = raw.strip()
    if not value:
        raise HardeningError(f"{path} does not set {ONBOARDING_PROPERTY}")
    return validate_onboarding_base_url(value)


def dart_defines_property(defines: dict[str, str]) -> str:
    """Gradle takes a comma-separated list of base64 KEY=VALUE pairs."""
    encoded = [
        base64.b64encode(f"{key}={value}".encode()).decode()
        for key, value in defines.items()
    ]
    return "-Pdart-defines=" + ",".join(encoded)


def resolve_onboarding_base_url(signing: str, onboarding: str, override: str | None) -> str | None:
    """Decides what this build class compiles in, failing closed for production.

    Production never reaches the "omit" arm: an official build that quietly
    shipped without managed onboarding is the defect this whole contract
    exists to prevent, and it is not a choice a flag gets to make.
    """
    if override is not None:
        return validate_onboarding_base_url(override.strip())
    if onboarding == "omit":
        if signing == "production":
            raise HardeningError(
                "a production build cannot omit the official onboarding URL: managed "
                "Telegram onboarding would be silently absent from the shipped product")
        return None
    return read_official_onboarding_base_url()


def verify_packaged_onboarding_url(app: bytes, expected: str | None) -> None:
    """Holds the finished AOT blob to what the build claimed to compile in.

    The build command is a claim; libapp.so is the evidence. vc51 proved the
    difference matters: the command lost the define, every gate stayed green,
    and the feature vanished from the product for eleven builds.
    """
    present = expected is not None and expected.encode() in app
    if expected is not None and not present:
        raise HardeningError(
            f"the packaged Dart AOT does not contain {ONBOARDING_DEFINE}={expected}; "
            "the build did not carry the define, so managed Telegram onboarding "
            "would be unavailable in this artifact")
    if expected is None:
        try:
            tracked = read_official_onboarding_base_url()
        except HardeningError:
            return
        if tracked.encode() in app:
            raise HardeningError(
                "this build omits the onboarding URL, but the packaged Dart AOT still "
                "contains it -- the artifact is stale and was not rebuilt")


def gradle_command(
    signing: str,
    symbols_property: str,
    package: str = "apk",
    onboarding_base_url: str | None = None,
) -> list[str]:
    try:
        task = GRADLE_TASKS[package]
    except KeyError as error:
        raise HardeningError(f"unknown package type: {package}") from error
    command = [
        str(ANDROID / "gradlew"),
        task,
        "-Ptarget-platform=android-arm64",
        "-PpocketclawDartHardening=true",
        "-Pdart-obfuscation=true",
        f"-Psplit-debug-info={symbols_property}",
    ]
    # The canonical release is the Gradle path, which has no --dart-define.
    # It takes -Pdart-defines instead: FlutterPlugin reads the property and
    # forwards it to `flutter assemble` as --DartDefines.
    if onboarding_base_url is not None:
        command.append(dart_defines_property({ONBOARDING_DEFINE: onboarding_base_url}))
    if signing == "local-test":
        command.append("-PallowDebugSigning=true")
    if signing == "unsigned":
        command.append("-PpocketclawUnsignedRelease=true")
    return command


def validate_signing_environment(signing: str, environ: dict[str, str]) -> None:
    declared = [name for name in SIGNING_ENV if environ.get(name, "")]
    if signing == "local-test" and declared:
        raise HardeningError(
            "LOCAL TEST mode refuses declared production signing variables: "
            + ", ".join(declared)
        )
    if signing == "unsigned" and declared:
        raise HardeningError(
            "UNSIGNED mode refuses declared production signing variables: "
            + ", ".join(declared)
        )
    if signing == "production" and len(declared) != len(SIGNING_ENV):
        missing = [name for name in SIGNING_ENV if name not in declared]
        raise HardeningError(
            "production signing is incomplete; missing variable names: " + ", ".join(missing)
        )


def verify_private_dart_symbols(symbols: Path) -> list[str]:
    """Prove the private split-debug-info is usable for deobfuscation.

    Shared by both packaging paths: the symbols come from one Dart compilation,
    so checking them twice differently would be two chances to be wrong.
    """
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
    return retained


def verify_packaged_dart_aot(app: bytes) -> None:
    """The obfuscation, path-privacy and generated-URI contract for libapp.so."""
    leaked = [marker.decode() for marker in HOST_PATH_MARKERS if marker in app]
    if leaked:
        raise HardeningError(
            "packaged Dart AOT contains host-specific path markers: " + ", ".join(leaked))
    if GENERATED_REGISTRANT_URI not in app:
        raise HardeningError("packaged Dart AOT lacks the controlled generated-source package URI")
    exposed = [marker.decode() for marker in APP_SYMBOL_MARKERS if marker in app]
    if exposed:
        raise HardeningError(
            "Dart obfuscation did not remove application symbols: " + ", ".join(exposed))


def inspect_hardened_bundle(
    bundle: Path,
    symbols: Path,
    distribution: str,
    classification: str = "LOCAL TEST / NON-PUBLISH",
    r8_mapping: Path = R8_MAPPING,
    onboarding_base_url: str | None = None,
) -> dict[str, object]:
    """Verify a hardened AAB and record what its distribution class permits.

    The bundle is held to the same Dart contract as the APK, and then to a
    different privacy contract — because the two are not the same artifact for
    the same audience. AGP writes the R8 mapping and native debug symbols into
    BUNDLE-METADATA/ for Google Play to consume, so they are expected here and
    are inventoried rather than treated as leakage. What is *not* permitted in
    any class is unrelated private material, and what is never permitted at all
    is calling this thing publishable.
    """
    if not bundle.is_file():
        raise HardeningError(f"Gradle completed without producing {bundle}")
    if not symbols.is_file() or symbols.stat().st_size == 0:
        raise HardeningError(f"split debug info was not produced at {symbols}")

    kind = detect_artifact_kind(bundle)
    allowed, message = distribution_verdict(kind, distribution)
    if not allowed:
        raise HardeningError(message)

    retained = verify_private_dart_symbols(symbols)
    inventory = inspect_bundle(bundle)

    app_entry = "base/lib/arm64-v8a/libapp.so"
    with zipfile.ZipFile(bundle) as archive:
        names = archive.namelist()
        if app_entry not in names:
            raise HardeningError(f"{bundle} does not package {app_entry}")
        app = archive.read(app_entry)
    verify_packaged_dart_aot(app)
    verify_packaged_onboarding_url(app, onboarding_base_url)

    leaks = private_material_violations(names, kind=kind)
    if leaks:
        raise HardeningError(
            "bundle packages private material no distribution class permits: "
            + ", ".join(f"{name} ({reason})" for name, reason in leaks))

    mapping_entries = [item for item in inventory["bundleMetadata"]
                       if item["category"] == "r8 deobfuscation mapping"]
    symbol_entries = [item for item in inventory["bundleMetadata"]
                      if item["category"] == "native debug symbols"]
    unrecognised = [item["entry"] for item in inventory["bundleMetadata"]
                    if not item["allowedForPlayUpload"]]
    if unrecognised:
        raise HardeningError(
            "bundle carries unrecognised BUNDLE-METADATA entries: " + ", ".join(unrecognised))

    evidence = {
        "bundle": str(bundle),
        "bundleBytes": bundle.stat().st_size,
        "bundleSha256": sha256_file(bundle),
        "dartAotSha256": sha256_bytes(app),
        "symbols": str(symbols),
        "symbolsBytes": symbols.stat().st_size,
        "symbolsSha256": sha256_file(symbols),
        "retainedPrivateMarkers": len(retained),
        "generatedSourceUri": GENERATED_REGISTRANT_URI.decode(),
        "classification": classification,
        "distributionClass": distribution,
        "distributionVerdict": message,
        "modules": inventory["modules"],
        "abis": inventory["abis"],
        "nativeEntryCount": len(inventory["nativeEntries"]),
        "bundleMetadata": inventory["bundleMetadata"],
        "bundleMetadataBytes": inventory["bundleMetadataBytes"],
        "r8MappingEntriesInBundle": [item["entry"] for item in mapping_entries],
        "nativeDebugSymbolEntriesInBundle": [item["entry"] for item in symbol_entries],
        "publicReleaseSafe": False,
        "playReady": distribution == PLAY_UPLOAD,
        "notice": (
            "NOT PLAY-READY; NOT PUBLIC-RELEASE-SAFE; NOT A GITHUB RELEASE ASSET"
            if distribution == NON_PUBLISH_AUDIT else
            "PLAY UPLOAD ONLY; NOT PUBLIC-RELEASE-SAFE; NOT A GITHUB RELEASE ASSET"
        ),
    }
    return evidence


APK_SIGNATURE_ENTRY = re.compile(r"^META-INF/[^/]+\.(SF|RSA|DSA|EC)$")
APK_SIGNING_BLOCK_MAGIC = b"APK Sig Block 42"


def verify_unsigned_apk(apk: Path) -> None:
    """Fails unless the APK carries no v1 signature and no APK Signing Block."""
    with zipfile.ZipFile(apk) as archive:
        v1 = [name for name in archive.namelist() if APK_SIGNATURE_ENTRY.match(name)]
    if v1:
        raise HardeningError("the unsigned APK carries JAR signature entries: " + ", ".join(v1))
    data = apk.read_bytes()
    eocd = data.rfind(b"PK\x05\x06")
    if eocd < 0:
        raise HardeningError(f"{apk} has no ZIP end-of-central-directory record")
    central_directory = int.from_bytes(data[eocd + 16:eocd + 20], "little")
    if data[max(0, central_directory - 16):central_directory] == APK_SIGNING_BLOCK_MAGIC:
        raise HardeningError("the unsigned APK carries an APK Signing Block")


def inspect_hardened_outputs(
    apk: Path,
    symbols: Path,
    classification: str = "LOCAL TEST / NON-RELEASABLE",
    r8_mapping: Path = R8_MAPPING,
    onboarding_base_url: str | None = None,
) -> dict[str, object]:
    if not apk.is_file():
        raise HardeningError(f"Gradle completed without producing {apk}")
    if not symbols.is_file() or symbols.stat().st_size == 0:
        raise HardeningError(f"split debug info was not produced at {symbols}")
    if classification == UNSIGNED_CLASSIFICATION:
        verify_unsigned_apk(apk)

    retained = verify_private_dart_symbols(symbols)

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
    verify_packaged_dart_aot(app)
    verify_packaged_onboarding_url(app, onboarding_base_url)

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
        "onboardingBaseUrl": onboarding_base_url or "omitted",
    }
    evidence.update(inspect_r8_outputs(apk, r8_mapping))
    return evidence


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--signing", required=True, choices=("local-test", "production", "unsigned"),
        help="local-test: development key; production: owner key from the "
             "environment; unsigned: no signature, for a repository to sign "
             "(refuses any declared signing variable)")
    parser.add_argument(
        "--symbols-dir",
        default=str(DEFAULT_SYMBOLS_DIR),
        help="private split-debug-info directory (default: %(default)s)",
    )
    parser.add_argument("--clean", action="store_true", help="run :app:clean before assembly")
    parser.add_argument(
        "--package", choices=tuple(GRADLE_TASKS), default="apk",
        help="what to package (default: %(default)s). Both use the same "
             "hardening contract; only the Gradle task and the privacy contract "
             "of the output differ",
    )
    parser.add_argument(
        "--artifact-class", choices=(PLAY_UPLOAD, NON_PUBLISH_AUDIT), default=None,
        help="required with --package bundle, and has no default: an "
             "unclassified bundle fails closed. A bundle can never be "
             f"{PUBLIC_RELEASE}, so that choice is not offered here",
    )
    parser.add_argument(
        "--onboarding", choices=("official", "omit"), default="official",
        help="which onboarding contract this build class carries (default: "
             "%(default)s). 'official' compiles in the tracked PocketClaw "
             "endpoint; 'omit' ships without managed onboarding and is refused "
             "for --signing production",
    )
    parser.add_argument(
        "--onboarding-url", default=None,
        help="override the onboarding base URL for a downstream or F-Droid "
             "build. Must be https. Only a public base URL belongs here",
    )
    args = parser.parse_args()
    if args.onboarding_url is not None and args.onboarding == "omit":
        parser.error("--onboarding-url cannot be combined with --onboarding omit")
    if args.package == "bundle" and not args.artifact_class:
        parser.error(
            "--artifact-class is required with --package bundle and has no "
            "default: a hardened bundle carries the R8 mapping and native debug "
            "symbols for Google Play, so what it is FOR decides whether that is "
            f"acceptable. Choose {PLAY_UPLOAD} or {NON_PUBLISH_AUDIT}."
        )
    if args.package == "apk" and args.artifact_class:
        parser.error("--artifact-class applies to --package bundle only")
    if args.signing == "unsigned" and args.package != "apk":
        parser.error("--signing unsigned builds a repository APK; an AAB is Play-only")
    return args


def main() -> int:
    args = parse_args()
    try:
        inspect_r8_source_contract()
        validate_signing_environment(args.signing, dict(os.environ))
        symbols_property, symbols_dir = resolve_symbols_dir(args.symbols_dir)
        changed = prepare_generated_source_package()
        symbols = symbols_dir / "app.android-arm64.symbols"
        apk = UNSIGNED_APK if args.signing == "unsigned" else APK
        reset_generated_build_outputs(
            BUNDLE if args.package == "bundle" else apk, symbols, r8_mapping=R8_MAPPING)
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
        classification = CLASSIFICATIONS[args.signing]
        print(
            "Signing classification: " + classification,
            flush=True,
        )
        if args.clean:
            clean = [str(ANDROID / "gradlew"), ":app:clean"]
            print("Build step: " + shlex.join(clean), flush=True)
            subprocess.run(clean, cwd=ANDROID, env=environment, check=True)
        onboarding_base_url = resolve_onboarding_base_url(
            args.signing, args.onboarding, args.onboarding_url)
        print(
            "Onboarding contract: "
            + (onboarding_base_url if onboarding_base_url else "omitted (no managed onboarding)"),
            flush=True,
        )
        command = gradle_command(
            args.signing, symbols_property, args.package, onboarding_base_url)
        print("Build step: " + shlex.join(command), flush=True)
        subprocess.run(command, cwd=ANDROID, env=environment, check=True)
        if args.package == "bundle":
            bundle_classification = (
                "LOCAL TEST / NON-PUBLISH" if args.signing == "local-test"
                else "PRODUCTION / NON-PUBLISH"
            )
            evidence = inspect_hardened_bundle(
                BUNDLE, symbols, args.artifact_class, bundle_classification,
                onboarding_base_url=onboarding_base_url)
            print(json.dumps(evidence, indent=2, sort_keys=True))
            print("\n" + evidence["notice"], flush=True)
            return 0
        evidence = inspect_hardened_outputs(
            apk, symbols, classification, onboarding_base_url=onboarding_base_url)
        print(json.dumps(evidence, indent=2, sort_keys=True))
        return 0
    except (HardeningError, R8ContractError, subprocess.CalledProcessError) as error:
        print(f"Hardened Android build FAILED: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
