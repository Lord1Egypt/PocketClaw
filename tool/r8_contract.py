#!/usr/bin/env python3
"""Evidence checks for PocketClaw's Android R8 release contract."""

from __future__ import annotations

import hashlib
import re
import subprocess
import zipfile
from pathlib import Path


REPO = Path(__file__).resolve().parent.parent
GRADLE = REPO / "android/app/build.gradle.kts"
RULES = REPO / "android/app/proguard-rules.pro"
MAPPING = REPO / "build/app/outputs/mapping/release/mapping.txt"

REQUIRED_COMPONENTS = (
    "com.lord1egypt.pocketclaw.MainActivity",
    "com.lord1egypt.pocketclaw.PocketClawApp",
    "com.lord1egypt.pocketclaw.receiver.BootReceiver",
    "com.lord1egypt.pocketclaw.service.PocketClawService",
)
INTERNAL_CLASSES = (
    "com.lord1egypt.pocketclaw.PocketClawCoreState",
    "com.lord1egypt.pocketclaw.PocketClawMethodChannel",
    "com.lord1egypt.pocketclaw.PocketClawPreferences",
    "com.lord1egypt.pocketclaw.media.ChatImagePicker",
    "com.lord1egypt.pocketclaw.security.GitHubCredentialStore",
    "com.lord1egypt.pocketclaw.util.HealthChecker",
)


class R8ContractError(RuntimeError):
    pass


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1 << 20), b""):
            digest.update(chunk)
    return digest.hexdigest()


def source_contract(gradle: Path = GRADLE, rules: Path = RULES) -> dict[str, object]:
    """Fail if release R8 is disabled or the app restores a blanket keep."""
    gradle_text = gradle.read_text(encoding="utf-8")
    rules_text = rules.read_text(encoding="utf-8")
    required = {
        "minify": "isMinifyEnabled = true",
        "resources": "isShrinkResources = true",
        "optimized_defaults": 'getDefaultProguardFile("proguard-android-optimize.txt")',
        "app_rules": '"proguard-rules.pro"',
    }
    missing = [name for name, marker in required.items() if marker not in gradle_text]
    if "isMinifyEnabled = false" in gradle_text:
        missing.append("minify explicitly disabled")
    if "isShrinkResources = false" in gradle_text:
        missing.append("resource shrinking explicitly disabled")
    active_lines = [
        line.strip()
        for line in rules_text.splitlines()
        if line.strip() and not line.lstrip().startswith("#")
    ]
    active = "\n".join(active_lines)
    forbidden = []
    for label, pattern in (
        ("disable obfuscation", r"(?m)^-dontobfuscate\b"),
        ("disable shrinking", r"(?m)^-dontshrink\b"),
        ("disable optimization", r"(?m)^-dontoptimize\b"),
        ("all PocketClaw classes", r"(?m)^-keep[^\n]*\bcom\.lord1egypt\.pocketclaw\.\*\*"),
        ("all Flutter classes", r"(?m)^-keep[^\n]*\bio\.flutter\.\*\*"),
        ("all plugin classes", r"(?m)^-keep[^\n]*\bio\.flutter\.plugins\.\*\*"),
        ("all classes with all members", r"(?m)^-keep(?:classeswithmembers)?\s+class\s+\*\*?\s*\{\s*\*;\s*\}"),
    ):
        if re.search(pattern, active):
            forbidden.append(label)
    if missing or forbidden:
        details = []
        if missing:
            details.append("missing release settings: " + ", ".join(missing))
        if forbidden:
            details.append("weakening rules: " + ", ".join(forbidden))
        raise R8ContractError("; ".join(details))
    return {
        "minifyEnabled": True,
        "shrinkResources": True,
        "optimizedDefaults": True,
        "activeApplicationRules": len(active_lines),
    }


def parse_class_mappings(mapping: Path) -> dict[str, str]:
    mappings: dict[str, str] = {}
    for line in mapping.read_text(encoding="utf-8", errors="replace").splitlines():
        match = re.fullmatch(r"([^ ]+) -> ([^ ]+):", line)
        if match:
            mappings[match.group(1)] = match.group(2)
    return mappings


def inspect_outputs(apk: Path, mapping: Path = MAPPING, repo: Path = REPO) -> dict[str, object]:
    """Prove that fresh R8 output shrank/renamed code and stayed private."""
    if not mapping.is_file() or mapping.stat().st_size == 0:
        raise R8ContractError(f"R8 mapping was not produced at {mapping}")
    usage = mapping.with_name("usage.txt")
    if not usage.is_file() or usage.stat().st_size == 0:
        raise R8ContractError(f"R8 shrinking evidence was not produced at {usage}")
    if not apk.is_file():
        raise R8ContractError(f"APK was not produced at {apk}")

    mappings = parse_class_mappings(mapping)
    usage_text = usage.read_text(encoding="utf-8", errors="replace")
    missing_components = [name for name in REQUIRED_COMPONENTS if name not in mappings]
    changed_components = [name for name in REQUIRED_COMPONENTS if mappings.get(name) not in (None, name)]
    if missing_components or changed_components:
        raise R8ContractError(
            "manifest entry-point mapping is invalid: missing=" + ",".join(missing_components)
            + " changed=" + ",".join(changed_components)
        )

    present_internal = [name for name in INTERNAL_CLASSES if name in mappings]
    renamed_internal = [name for name in present_internal if mappings[name] != name]
    removed_or_folded_internal = [
        name for name in INTERNAL_CLASSES
        if name not in mappings
        and (
            re.search(rf"(?m)^{re.escape(name)}(?:\$|:|$)", usage_text)
            or any(key.startswith(name + "$") for key in mappings)
        )
    ]
    accounted_internal = set(renamed_internal) | set(removed_or_folded_internal)
    if set(INTERNAL_CLASSES) != accounted_internal:
        raise R8ContractError(
            "R8 did not rename, remove, or fold every PocketClaw probe class: "
            + ", ".join(sorted(set(INTERNAL_CLASSES) - accounted_internal))
        )

    with zipfile.ZipFile(apk) as archive:
        names = archive.namelist()
        dex_entries = sorted(name for name in names if re.fullmatch(r"classes(?:\d+)?\.dex", name))
        dex_blobs = [archive.read(name) for name in dex_entries]
        packaged_support = [
            name for name in names
            if Path(name).name in {"mapping.txt", "usage.txt", "seeds.txt", "configuration.txt"}
            or "private-mapping" in name
        ]
    if not dex_entries:
        raise R8ContractError(f"{apk} contains no classes DEX")
    if packaged_support:
        raise R8ContractError("private R8 support files were packaged: " + ", ".join(packaged_support))

    combined_dex = b"".join(dex_blobs)
    clear_descriptors = []
    for name in INTERNAL_CLASSES:
        descriptor = ("L" + name.replace(".", "/") + ";").encode()
        if descriptor in combined_dex:
            clear_descriptors.append(name)
    if clear_descriptors:
        raise R8ContractError(
            "renamed implementation descriptors remain in DEX: " + ", ".join(clear_descriptors)
        )

    tracked = False
    try:
        relative = mapping.resolve().relative_to(repo.resolve())
    except ValueError:
        relative = None
    if relative is not None:
        tracked = subprocess.run(
            ["git", "ls-files", "--error-unmatch", str(relative)], cwd=repo,
            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, check=False,
        ).returncode == 0
    if tracked:
        raise R8ContractError(f"R8 mapping is tracked by Git: {mapping}")

    return {
        "r8Mapping": str(mapping),
        "r8MappingBytes": mapping.stat().st_size,
        "r8MappingSha256": sha256_file(mapping),
        "r8Usage": str(usage),
        "r8UsageBytes": usage.stat().st_size,
        "r8UsageSha256": sha256_file(usage),
        "dexCount": len(dex_entries),
        "dexBytes": sum(len(blob) for blob in dex_blobs),
        "dexEntries": dex_entries,
        "r8RequiredEntryPoints": list(REQUIRED_COMPONENTS),
        "r8RenamedInternalClasses": renamed_internal,
        "r8RemovedOrFoldedInternalClasses": removed_or_folded_internal,
        "r8MappingPrivate": True,
    }
