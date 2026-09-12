#!/usr/bin/env python3
"""Distribution policy for PocketClaw's Android artifacts.

PC-DEF-021. The rule this module exists to enforce is about an artifact's
*purpose*, not its file extension.

A hardened Android App Bundle legitimately carries release-support material
under ``BUNDLE-METADATA/``: the R8 deobfuscation mapping and native debug
symbols. That is AGP working as designed — Google Play consumes those entries to
symbolicate crash reports and does not deliver them to installed clients. So
their presence in a Play-destined bundle is correct and must not be "fixed" by
deleting them, which would only remove Play's ability to read a stack trace.

What is wrong is publishing such a bundle. PocketClaw attached AABs to public
GitHub pre-releases while simultaneously treating the mapping and symbols as
private, and those two facts cannot both hold. The policy therefore is:

    APK  → may be a public release artifact, and must carry no private material
    AAB  → Play upload or private audit only, never a public release asset

An unclassified artifact fails closed. A bundle with no mapping at all is still
forbidden as a public asset, because the prohibition is about what the format is
for, not about what this particular file happens to contain.
"""

from __future__ import annotations

import re
import zipfile
from pathlib import Path

# Distribution classes. These name what an artifact is *for*, which is the only
# thing that can decide whether its contents are acceptable.
PUBLIC_RELEASE = "public-release"
PLAY_UPLOAD = "play-upload"
NON_PUBLISH_AUDIT = "non-publish-audit"
DISTRIBUTION_CLASSES = (PUBLIC_RELEASE, PLAY_UPLOAD, NON_PUBLISH_AUDIT)

DISTRIBUTION_DESCRIPTIONS = {
    PUBLIC_RELEASE: "published to GitHub Releases or served as a direct public download",
    PLAY_UPLOAD: "uploaded to Google Play only; never a public download",
    NON_PUBLISH_AUDIT: "inspection evidence only; not published anywhere",
}

APK = "apk"
AAB = "aab"

# Android binary formats that may be published publicly. The bundle is absent
# deliberately and that absence is the fix for PC-DEF-021.
PUBLIC_RELEASE_BINARY_KINDS = (APK,)


class ArtifactPolicyError(RuntimeError):
    """An artifact cannot be classified, or violates its own classification."""


def detect_artifact_kind(path: Path) -> str:
    """Identify an APK or an AAB from its contents.

    By content rather than by extension: a policy that can be bypassed by
    renaming a file is not a policy. A bundle is identified by ``BundleConfig.pb``
    plus a module-scoped manifest; an APK by a manifest at the archive root.
    """
    if not path.is_file():
        raise ArtifactPolicyError(f"artifact does not exist: {path}")
    try:
        with zipfile.ZipFile(path) as archive:
            names = set(archive.namelist())
    except zipfile.BadZipFile as error:
        raise ArtifactPolicyError(f"{path} is not a zip archive: {error}") from error

    bundle_markers = {"BundleConfig.pb"} & names
    module_manifest = any(
        name.endswith("/manifest/AndroidManifest.xml") for name in names
    )
    if bundle_markers or module_manifest:
        return AAB
    if "AndroidManifest.xml" in names:
        return APK
    raise ArtifactPolicyError(
        f"{path} is neither an APK nor an AAB: no AndroidManifest.xml at the "
        "archive root and no bundle module manifest"
    )


def distribution_verdict(kind: str, distribution: str) -> tuple[bool, str]:
    """Decide whether this artifact format may be used for this purpose.

    Fails closed: an unknown or absent class is a refusal, never a default to
    the permissive answer.
    """
    if distribution not in DISTRIBUTION_CLASSES:
        return False, (
            "artifact distribution class is missing or unrecognised "
            f"({distribution!r}); choose one of {', '.join(DISTRIBUTION_CLASSES)}"
        )
    if distribution != PUBLIC_RELEASE:
        return True, (
            f"{kind.upper()} classified {distribution} — "
            f"{DISTRIBUTION_DESCRIPTIONS[distribution]}"
        )
    if kind in PUBLIC_RELEASE_BINARY_KINDS:
        return True, f"{kind.upper()} is permitted as a public release artifact"
    return False, (
        "AAB is a Play-upload/private-distribution artifact and is forbidden as "
        "a public release asset. A hardened bundle carries the R8 deobfuscation "
        "mapping and native debug symbols under BUNDLE-METADATA/ for Google Play "
        "to consume, and publishing it would publish them. Publish the hardened "
        "APK instead; see docs/RELEASE_PROCESS.md."
    )


# ---------------------------------------------------------------------------
# Public release asset allowlist
# ---------------------------------------------------------------------------
#
# Applied to the *names of files attached to a public release*, not to entries
# inside an archive. Ordinary release companions — checksums, notices, the
# source tarballs GitHub generates — stay allowed; anything that carries
# deobfuscation power or signing material does not.

FORBIDDEN_PUBLIC_ASSET_RULES: tuple[tuple[str, re.Pattern[str]], ...] = (
    ("android app bundle", re.compile(r"\.aab$", re.I)),
    ("r8/proguard mapping", re.compile(r"(?:^|[-_.])(?:mapping|seeds|usage|configuration)\.txt$", re.I)),
    ("r8/proguard mapping", re.compile(r"proguard[-_.]?map", re.I)),
    ("native debug companion", re.compile(r"\.(?:debug|dbg|sym|dwarf|dwp)$", re.I)),
    ("dart split debug info", re.compile(r"\.symbols$", re.I)),
    # More specific before more general: "private-symbols.tar.gz" is both, and
    # the private-support label is the one that says why it is forbidden.
    ("private support archive", re.compile(r"private[-_]?(?:symbols?|mapping|support)", re.I)),
    ("symbol archive", re.compile(r"(?:symbols?|debug[-_]?symbols?|dsym)[^/]*\.(?:zip|tar|tar\.gz|tgz|7z)$", re.I)),
    ("signing material", re.compile(r"\.(?:jks|p12|keystore|pfx|pem|ppk|key)$", re.I)),
    ("signing material", re.compile(r"keystore", re.I)),
    ("environment file", re.compile(r"(?:^|/)\.env(?:$|\.)", re.I)),
)


def public_release_asset_violations(names) -> list[tuple[str, str]]:
    """Return (asset, reason) for every name forbidden as a public release asset."""
    violations: list[tuple[str, str]] = []
    for raw in names:
        name = Path(str(raw)).name
        for reason, pattern in FORBIDDEN_PUBLIC_ASSET_RULES:
            if pattern.search(name):
                violations.append((str(raw), reason))
                break
    return violations


# ---------------------------------------------------------------------------
# Packaged private material
# ---------------------------------------------------------------------------
#
# These fail in *every* distribution class, including a Play upload. They are
# not release-support metadata Play asked for; they are material that has no
# business inside a distributed artifact at all.

PRIVATE_MATERIAL_RULES: tuple[tuple[str, re.Pattern[str]], ...] = (
    ("signing material", re.compile(r"\.(?:jks|p12|keystore|pfx|pem|ppk)$", re.I)),
    ("private key", re.compile(r"(?:^|/)id_(?:rsa|dsa|ecdsa|ed25519)(?:$|\.)", re.I)),
    ("environment file", re.compile(r"(?:^|/)\.env(?:$|\.)", re.I)),
    ("private native support archive", re.compile(r"private-symbols/", re.I)),
    ("native debug companion", re.compile(r"\.(?:debug|dbg|dwarf|dwp)$", re.I)),
    ("dart split debug info", re.compile(r"\.symbols$", re.I)),
    ("signing helper", re.compile(r"(?:sign|keystore)[^/]*\.(?:sh|bash|ps1|bat)$", re.I)),
    ("vcs metadata", re.compile(r"(?:^|/)\.git(?:/|$)", re.I)),
    ("credential store", re.compile(r"(?:^|/)credentials?(?:$|\.|/)", re.I)),
)

# One deliberate carve-out, and it is deliberately narrow. AGP names the
# bundle's native debug symbols "<lib>.so.sym", which the "native debug
# companion" rule would otherwise catch by extension, and it writes the R8
# mapping as "proguard.map". Those exact shapes are expected Play input and are
# reported by the metadata inventory instead of failed.
#
# The exemption matches the *entry*, never the directory. Exempting the whole
# directory would mean anything dropped into it — a keystore, an .env — became
# invisible to the privacy rules, which is the opposite of what a carve-out for
# two known filenames should buy.
BUNDLE_DEBUG_SYMBOL_PREFIX = "BUNDLE-METADATA/com.android.tools.build.debugsymbols/"
BUNDLE_OBFUSCATION_PREFIX = "BUNDLE-METADATA/com.android.tools.build.obfuscation/"

EXPECTED_BUNDLE_METADATA_ENTRIES = (
    re.compile(
        r"^BUNDLE-METADATA/com\.android\.tools\.build\.obfuscation/proguard\.map$"),
    re.compile(
        r"^BUNDLE-METADATA/com\.android\.tools\.build\.debugsymbols/"
        r"[A-Za-z0-9_\-]+/[^/]+\.so\.sym$"),
)


def _is_expected_bundle_metadata(name: str) -> bool:
    return any(pattern.match(name) for pattern in EXPECTED_BUNDLE_METADATA_ENTRIES)


def private_material_violations(names, kind: str = APK) -> list[tuple[str, str]]:
    """Return (entry, reason) for packaged material no distribution class allows.

    ``kind`` only decides whether the bundle's own AGP metadata directories are
    exempt; every other rule applies identically to an APK and an AAB.
    """
    violations: list[tuple[str, str]] = []
    for raw in names:
        name = str(raw)
        if kind == AAB and _is_expected_bundle_metadata(name):
            continue
        for reason, pattern in PRIVATE_MATERIAL_RULES:
            if pattern.search(name):
                violations.append((name, reason))
                break
    return violations


# ---------------------------------------------------------------------------
# R8 mapping, as actual archive evidence
# ---------------------------------------------------------------------------

R8_MAPPING_ENTRY_NAMES = frozenset(
    {"mapping.txt", "usage.txt", "seeds.txt", "configuration.txt", "proguard.map"}
)
R8_MAPPING_PATH_MARKERS = ("private-mapping", "proguard-map", "proguard.map")


def packaged_r8_mapping_entries(names) -> list[str]:
    """Entries that would let a reader deobfuscate the shipped DEX.

    Named entries and path markers, evaluated over the archive's real listing.
    The bundle's ``BUNDLE-METADATA/...obfuscation/proguard.map`` matches here on
    purpose: whether it is acceptable is a question for the distribution class,
    and that decision belongs to the caller rather than to this inventory.
    """
    found = []
    for raw in names:
        name = str(raw)
        base = Path(name).name
        if base in R8_MAPPING_ENTRY_NAMES or any(
            marker in name.lower() for marker in R8_MAPPING_PATH_MARKERS
        ):
            found.append(name)
    return sorted(found)


# ---------------------------------------------------------------------------
# Bundle inventory
# ---------------------------------------------------------------------------

BUNDLE_METADATA_CATEGORIES: tuple[tuple[str, str, bool], ...] = (
    # (prefix, category, allowed for a Play upload)
    (BUNDLE_OBFUSCATION_PREFIX, "r8 deobfuscation mapping", True),
    (BUNDLE_DEBUG_SYMBOL_PREFIX, "native debug symbols", True),
    ("BUNDLE-METADATA/com.android.tools.build.libraries/", "dependency metadata", True),
    ("BUNDLE-METADATA/com.android.tools.build.profiles/", "baseline profile", True),
    ("BUNDLE-METADATA/com.android.tools.build.gradle/", "agp build metadata", True),
    ("BUNDLE-METADATA/com.android.tools/", "agp tooling metadata", True),
)


def categorize_bundle_metadata(name: str) -> tuple[str, bool]:
    """Category and Play-upload acceptability for one BUNDLE-METADATA entry."""
    for prefix, category, allowed in BUNDLE_METADATA_CATEGORIES:
        if name.startswith(prefix):
            return category, allowed
    return "unrecognised bundle metadata", False


def inspect_bundle(path: Path) -> dict[str, object]:
    """Structural inventory of an AAB: modules, ABIs, natives, metadata.

    Read-only, and it classifies nothing — it reports what is there so a caller
    holding the distribution class can decide.
    """
    with zipfile.ZipFile(path) as archive:
        infos = archive.infolist()

    names = [info.filename for info in infos]
    sizes = {info.filename: info.file_size for info in infos}

    modules = sorted({
        name.split("/", 1)[0]
        for name in names
        if "/" in name and not name.startswith(("BUNDLE-METADATA/", "META-INF/"))
    })
    abis = sorted({
        match.group(1)
        for name in names
        if (match := re.search(r"(?:^|/)lib/([^/]+)/", name))
    })
    natives = sorted(
        (name, sizes[name]) for name in names if re.search(r"(?:^|/)lib/[^/]+/[^/]+\.so$", name)
    )
    manifests = sorted(name for name in names if name.endswith("AndroidManifest.xml"))
    metadata = []
    for name in sorted(name for name in names if name.startswith("BUNDLE-METADATA/")):
        category, play_allowed = categorize_bundle_metadata(name)
        metadata.append({
            "entry": name,
            "sizeBytes": sizes[name],
            "category": category,
            "allowedForPlayUpload": play_allowed,
        })
    signatures = sorted(name for name in names if name.startswith("META-INF/"))

    return {
        "kind": AAB,
        "entryCount": len(infos),
        "modules": modules,
        "abis": abis,
        "manifests": manifests,
        "nativeEntries": natives,
        "bundleMetadata": metadata,
        "bundleMetadataBytes": sum(int(item["sizeBytes"]) for item in metadata),
        "signatureEntries": signatures,
        "entryNames": names,
    }
