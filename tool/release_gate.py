#!/usr/bin/env python3
"""PocketClaw release gate.

One command that decides whether an artifact is releasable, so the answer does
not depend on who is asking or which CI happens to be running. CI, when it
exists, invokes this; it is never the other way round.

The gate delegates wherever an authoritative guard already exists — the staged
Core freshness test, the A1 signing and version contracts, the A2 private
storage guards, the Gradle payload verifiers — rather than restating their logic
in shell. A second implementation of a rule is a second thing to get wrong, and
the two would drift silently.

Modes:

    --verify-source              repository and source contracts only
    --verify-artifact <apk>      inspect a built APK
    --full <apk>                 both
    --verify-bundle <aab>        inspect an Android App Bundle
    --release-assets <name...>   check proposed public release asset names
    --release-class test|production
    --artifact-class public-release|play-upload|non-publish-audit
    --dart-symbols <path>        require H3 Dart hardening evidence from the
                                 private split-debug-info file or directory
    --r8-mapping <path>          require H4 R8 mapping/shrinking evidence

`test` permits the known development signer and can never report a production
release. `production` treats a development signer as an unconditional failure.
The caller chooses; the gate never guesses from context.

`--release-class` is about *signing*; `--artifact-class` is about *purpose*, and
they answer different questions. An artifact phase requires the latter and it
has no default, because the one thing PC-DEF-021 proved is that guessing
"publishable" is the guess that costs something: a hardened AAB carries the R8
mapping and native debug symbols for Google Play to consume, so it is a valid
Play upload and never a valid public download. The gate refuses an unclassified
artifact rather than picking the permissive reading.

Exit 0 when every selected check passes, non-zero otherwise. Every check reports
PASS, FAIL or SKIPPED, and a SKIPPED check is always named.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import shutil
import subprocess
import sys
import zipfile
from dataclasses import dataclass, field
from pathlib import Path

from r8_contract import R8ContractError, inspect_outputs as inspect_r8_outputs
from artifact_policy import (
    AAB,
    APK as ARTIFACT_APK,
    DISTRIBUTION_CLASSES,
    NON_PUBLISH_AUDIT,
    PLAY_UPLOAD,
    PUBLIC_RELEASE,
    ArtifactPolicyError,
    detect_artifact_kind,
    distribution_verdict,
    inspect_bundle,
    packaged_r8_mapping_entries,
    private_material_violations,
    public_release_asset_violations,
)

REPO = Path(__file__).resolve().parent.parent
PACKAGE_ID = "com.lord1egypt.pocketclaw"
EXPECTED_ABI = "arm64-v8a"
EXPECTED_LOCALES = 12

# The permission contract accepted in Release Hardening A1, asserted against the
# packaged APK rather than the source manifest: a dependency can contribute a
# permission the app never wrote.
EXPECTED_PERMISSIONS = {
    "android.permission.ACCESS_NETWORK_STATE",
    "android.permission.FOREGROUND_SERVICE",
    "android.permission.FOREGROUND_SERVICE_SPECIAL_USE",
    "android.permission.INTERNET",
    "android.permission.POST_NOTIFICATIONS",
    "android.permission.WAKE_LOCK",
    f"{PACKAGE_ID}.DYNAMIC_RECEIVER_NOT_EXPORTED_PERMISSION",
}

# Removed in A1 and required to stay gone.
FORBIDDEN_PERMISSIONS = {
    "android.permission.READ_PHONE_STATE",
    "com.google.android.gms.permission.AD_ID",
    "android.permission.ACCESS_ADSERVICES_AD_ID",
    "android.permission.ACCESS_ADSERVICES_ATTRIBUTION",
    "com.google.android.finsky.permission.BIND_GET_INSTALL_REFERRER_SERVICE",
    "freemme.permission.msa",
    # PC-DEF-083: contributed by flutter_background_service and
    # flutter_local_notifications, which were never started or called, and by
    # a boot receiver no UI could enable. Removed with them; must stay gone.
    "android.permission.FOREGROUND_SERVICE_DATA_SYNC",
    "android.permission.RECEIVE_BOOT_COMPLETED",
    "android.permission.VIBRATE",
    # PC-DEF-077: the workspace is app-specific storage, which needs no storage
    # permission; all-files access and the legacy pair are gone for good.
    "android.permission.MANAGE_EXTERNAL_STORAGE",
    "android.permission.READ_EXTERNAL_STORAGE",
    "android.permission.WRITE_EXTERNAL_STORAGE",
}

CORE_LIBS = ("libpocketclaw.so", "libpocketclaw-web.so")
STAGED_CORE_DIR = REPO / "android/app/src/main/jniLibs" / EXPECTED_ABI
DART_GENERATED_REGISTRANT_URI = b"package:pocketclaw_generated/dart_plugin_registrant.dart"
DART_APP_SYMBOL_MARKERS = (
    b"TelegramOnboardingController",
    b"TelegramOnboardingClient",
    b"_MainShellState",
    b"StatusSnapshot",
)

# The Android debug signing certificate this project's local test builds carry.
# Recorded so a development artifact can be *classified*, never so it can be
# accepted as a release.
DEV_SIGNER_SHA256 = "15cf75f9945d5354e75707e0326b7cffc60ac51a68df38156db318ef4578a27c"

# Where the production signing certificate's public fingerprint is enrolled.
# Public metadata, deliberately tracked: it is the certificate, never the key.
PRODUCTION_CERT_FILE = REPO / "android/release-signing-cert.sha256"


def enrolled_production_signer() -> str | None:
    """The enrolled production certificate digest, or None before the ceremony.

    The file is comment-heavy on purpose, so anything that is not a bare
    64-character lowercase hex line is ignored. Before a key exists there is no
    such line, and production verification is supposed to fail — a placeholder
    that could accidentally match is worse than no answer at all.
    """
    if not PRODUCTION_CERT_FILE.is_file():
        return None
    for line in PRODUCTION_CERT_FILE.read_text(encoding="utf-8").splitlines():
        candidate = line.strip()
        if re.fullmatch(r"[0-9a-f]{64}", candidate):
            return candidate
    return None

# Real work that is not done yet. The gate names these rather than implying the
# release is fully hardened; it must not pretend they are solved.
PENDING_FINAL_HARDENING = [
    # Distribution is direct APK + Google Play + official F-Droid. F-Droid will
    # only publish the developer-signed artifact for a build it can reproduce,
    # so reproducibility is what decides whether a user can move between the
    # direct and F-Droid channels without uninstalling. Every hardening step
    # above is a candidate for breaking it; see docs/FDROID_RELEASE.md.
    "APK-level reproducibility not yet proven (required for F-Droid)",
    # The namespace migration was listed here until the sweep finished and
    # namespace.no_active_pico started enforcing it on every run. A standing
    # note that a solved problem is outstanding is as misleading as the reverse.
    "versioned non-destructive bootstrap update strategy outstanding",
]

PASS, FAIL, SKIP = "PASS", "FAIL", "SKIPPED"


@dataclass
class Result:
    name: str
    status: str
    detail: str = ""
    expected: str = ""
    observed: str = ""


@dataclass
class Gate:
    results: list[Result] = field(default_factory=list)
    facts: dict = field(default_factory=dict)

    def record(self, name, status, detail="", expected="", observed=""):
        self.results.append(Result(name, status, detail, expected, observed))
        return status == PASS

    def check(self, name, ok, expected="", observed="", detail=""):
        return self.record(name, PASS if ok else FAIL, detail, expected, observed)

    @property
    def failed(self):
        return [r for r in self.results if r.status == FAIL]


def run(cmd, cwd=None, env=None):
    merged = dict(os.environ)
    if env:
        merged.update(env)
    proc = subprocess.run(
        cmd, cwd=cwd or REPO, env=merged,
        capture_output=True, text=True,
    )
    return proc.returncode, proc.stdout + proc.stderr


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with open(path, "rb") as handle:
        for chunk in iter(lambda: handle.read(1 << 20), b""):
            digest.update(chunk)
    return digest.hexdigest()


def find_jdk() -> Path | None:
    """Locates a JDK for apksigner without hard-coding a machine path."""
    for candidate in (REPO.parent / "PocketCLaw/.tooling/jdk-17",):
        if (candidate / "bin/java").is_file():
            return candidate
    return None


def find_flutter() -> Path | None:
    """Locates Flutter deterministically, preferring the repository toolchain.

    Order matters and PATH is last. The gate decides whether a release is
    acceptable, so it must not depend on which Flutter happens to be first on
    whichever shell invoked it — two machines answering differently is the one
    thing a gate cannot do.
    """
    candidates = []
    pinned = REPO.parent / "PocketCLaw/.tooling/flutter/bin/flutter"
    candidates.append(pinned)
    flutter_root = os.environ.get("FLUTTER_ROOT")
    if flutter_root:
        candidates.append(Path(flutter_root) / "bin/flutter")
    for candidate in candidates:
        if candidate.is_file():
            return candidate
    found = shutil.which("flutter")
    return Path(found) if found else None


def run_flutter_suite(flutter: Path) -> tuple[int, dict[str, object]]:
    """Run the whole Flutter suite once and summarise it.

    Once, not once per named contract: the suite is the expensive part, and
    four invocations of it would be four chances for the gate to disagree with
    itself about the same tree.

    The JSON reporter is used so the summary is parsed rather than scraped, and
    so a failure can name the suite it came from. If the reporter yields nothing
    usable the exit code still decides — a gate that cannot read the output must
    not therefore call it a pass.
    """
    rc, out = run([str(flutter), "test", "--reporter", "json"], cwd=REPO)
    suites: dict[int, str] = {}
    test_suite: dict[int, int] = {}
    passed = failed = 0
    failing_suites: set[str] = set()
    failing_tests: list[str] = []
    names: dict[int, str] = {}
    for line in out.splitlines():
        line = line.strip()
        if not line.startswith("{"):
            continue
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue
        kind = event.get("type")
        if kind == "suite":
            suite = event.get("suite", {})
            suites[suite.get("id")] = suite.get("path") or "<unknown>"
        elif kind == "testStart":
            test = event.get("test", {})
            test_suite[test.get("id")] = test.get("suiteID")
            names[test.get("id")] = test.get("name", "<unnamed>")
        elif kind == "testDone":
            if event.get("hidden"):
                continue
            test_id = event.get("testID")
            if event.get("result") == "success":
                passed += 1
            else:
                failed += 1
                path = suites.get(test_suite.get(test_id), "<unknown>")
                failing_suites.add(path)
                if len(failing_tests) < 10:
                    failing_tests.append(f"{path}: {names.get(test_id, '?')}")
    summary = {
        "passed": passed,
        "failed": failed,
        "failingSuites": sorted(failing_suites),
        "failingTests": failing_tests,
        "parsed": bool(passed or failed),
    }
    return rc, summary


def find_sdk_tool(name: str) -> Path | None:
    """Locates an Android build-tool without hard-coding a machine path."""
    found = shutil.which(name)
    if found:
        return Path(found)
    sdk = os.environ.get("ANDROID_HOME") or os.environ.get("ANDROID_SDK_ROOT")
    roots = [Path(sdk)] if sdk else []
    local_properties = REPO / "android/local.properties"
    if local_properties.is_file():
        for line in local_properties.read_text(encoding="utf-8").splitlines():
            if line.startswith("sdk.dir="):
                roots.append(Path(line.split("=", 1)[1].strip()))
    for root in roots:
        build_tools = root / "build-tools"
        if not build_tools.is_dir():
            continue
        for version in sorted(build_tools.iterdir(), reverse=True):
            candidate = version / name
            if candidate.is_file():
                return candidate
    return None


# --------------------------------------------------------------------------
# Build-time verification
# --------------------------------------------------------------------------

BUILD_TIME_PATTERN = re.compile(rb"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[+-]\d{4}")


def record_expected_build_time(gate: Gate):
    """Records the BuildTime a canonical build of this tree should produce."""
    resolver = REPO / "core/resolve-build-time.sh"
    explicit = os.environ.get("SOURCE_DATE_EPOCH", "").strip()
    rc, out = run([str(resolver)])
    if rc == 0 and out.strip():
        gate.facts["buildTimeExpected"] = out.strip()
    rc, out = run([str(resolver), "--print-commit"])
    if rc == 0 and out.strip():
        gate.facts["coreBuildInputCommit"] = out.strip()
    gate.facts["buildTimeDerivation"] = (
        f"explicit SOURCE_DATE_EPOCH={explicit}" if explicit
        else "canonical Core build-input commit timestamp")


def staged_build_time_gate(gate: Gate, release_class: str):
    """Verifies the BuildTime stamped into the *staged* Core.

    The artifact check needs an APK; this one does not, and it answers the
    question that matters between releases: do the binaries in the tree actually
    correspond to the build inputs currently committed? A stale stamp here means
    the staged Core predates the source it is supposed to have been built from,
    which the fingerprint guard catches for content but not for time.

    Both binaries are checked, because the whole point of resolving the
    timestamp once and passing it to both make invocations is that they agree.
    """
    expected = gate.facts.get("buildTimeExpected")
    if not expected:
        gate.record("core.staged_build_time", SKIP, "expected BuildTime unavailable")
        return

    observed = {}
    for lib in CORE_LIBS:
        path = STAGED_CORE_DIR / lib
        if not path.is_file():
            observed[lib] = "missing"
            continue
        observed[lib] = embedded_build_time(path.read_bytes()) or "unreadable"
    gate.facts["stagedCoreBuildTime"] = observed

    mismatched = {lib: value for lib, value in observed.items() if value != expected}
    if not mismatched:
        gate.check("core.staged_build_time", True,
                   observed=f"both binaries stamped {expected}")
        return

    detail = ", ".join(f"{lib}={value}" for lib, value in mismatched.items())
    if release_class == "production":
        gate.check("core.staged_build_time", False,
                   expected=expected, observed=detail)
    else:
        gate.facts["releasable"] = False
        gate.record("core.staged_build_time", SKIP,
                    f"expected {expected}, got {detail} — NON-RELEASABLE (test class)")


def embedded_build_time(blob: bytes) -> str | None:
    """Reads the BuildTime stamped into a Core binary.

    The linker writes it as a plain string, so the timestamp is recoverable
    from the binary without running it. `dev` is the Makefile's marker for a
    build whose timestamp could not be derived deterministically.
    """
    if b"dev" in blob:
        # Only meaningful alongside the absence of a real stamp; checked after.
        pass
    match = BUILD_TIME_PATTERN.search(blob)
    return match.group(0).decode() if match else None


def build_time_gate(gate: Gate, core_blob: bytes, release_class: str):
    """Verifies what is actually stamped in the binary, not merely what the
    build *would* produce.

    Recording the input and stopping there would have missed the whole class of
    failure this milestone is about: a binary built before the deterministic
    contract, or by a `make` that fell back to `dev`, looks fine from the
    outside.
    """
    expected = gate.facts.get("buildTimeExpected")
    observed = embedded_build_time(core_blob)
    gate.facts["buildTimeObserved"] = observed or "unreadable"

    if observed is None:
        # No parseable timestamp at all: either `dev` or something unreadable.
        is_dev = b"dev" in core_blob
        detail = "BuildTime=dev (non-deterministic developer build)" if is_dev \
            else "no readable BuildTime"
        if release_class == "production":
            gate.check("artifact.build_time", False,
                       expected="a deterministic BuildTime", observed=detail)
        else:
            gate.facts["releasable"] = False
            gate.record("artifact.build_time", SKIP, f"{detail} — NON-RELEASABLE (test class)")
        return

    if expected is None:
        if release_class == "production":
            gate.check("artifact.build_time", False,
                       expected="a resolvable expected BuildTime",
                       observed=f"embedded {observed}, expected unknown")
        else:
            gate.record("artifact.build_time", SKIP,
                        f"embedded {observed}; expected value unavailable")
        return

    if observed == expected:
        gate.check("artifact.build_time", True, observed=observed)
        return

    # A mismatch means the artifact was not built from this tree's build inputs.
    # For a test-class artifact predating the contract that is information, not
    # a defect; for a production artifact it is disqualifying.
    if release_class == "production":
        gate.check("artifact.build_time", False,
                   expected=expected, observed=observed)
    else:
        gate.facts["releasable"] = False
        gate.record("artifact.build_time", SKIP,
                    f"embedded {observed} != expected {expected} — "
                    "LEGACY artifact, predates this tree's build inputs (NON-RELEASABLE)")


# --------------------------------------------------------------------------
# Source-level gates
# --------------------------------------------------------------------------

def tracked_version(gate: Gate):
    pubspec = REPO / "pubspec.yaml"
    match = re.search(
        r"^version:\s*(\d+\.\d+\.\d+)\+(\d+)\s*$",
        pubspec.read_text(encoding="utf-8"), re.M)
    if not match:
        gate.check("version.tracked", False,
                   expected="pubspec.yaml version: <name>+<code>",
                   observed="unparseable")
        return None, None
    name, code = match.group(1), int(match.group(2))
    gate.facts["versionName"] = name
    gate.facts["versionCode"] = code
    gate.check("version.tracked", True, observed=f"{name}+{code}")
    return name, code


def worktree_gate(gate: Gate, release_class: str):
    """A release must be reproducible from a committed state.

    Verifying an artifact built from uncommitted edits proves nothing about
    anything anyone else can obtain, and the build timestamp itself is derived
    from committed history — so a dirty tree can produce bytes whose inputs no
    longer exist. Git's own porcelain status decides what counts as dirty, so
    ignored caches and generated files stay ignored without a second rule here.
    """
    rc, out = run(["git", "status", "--porcelain"])
    if rc != 0:
        # A production verification has to be able to establish provenance:
        # which revision this is, whether it is clean, which commit last touched
        # a build input, and therefore what BuildTime a canonical build would
        # produce. None of that exists outside a git worktree, so "unknown" is
        # a failure rather than something to wave through. Local inspection of
        # an artifact still works — it just cannot claim to be a release.
        if release_class == "production":
            gate.check("repo.clean_worktree", False,
                       expected="a usable git worktree (provenance is required "
                                "for a production release)",
                       observed="not a git checkout")
        else:
            gate.facts["releasable"] = False
            gate.record("repo.clean_worktree", SKIP,
                        "not a git checkout — NON-RELEASABLE (test class)")
        return
    dirty = [line for line in out.splitlines() if line.strip()]
    if not dirty:
        gate.check("repo.clean_worktree", True, observed="clean")
        return

    summary = ", ".join(line[3:] for line in dirty[:5])
    if len(dirty) > 5:
        summary += f", +{len(dirty) - 5} more"
    if release_class == "production":
        gate.check("repo.clean_worktree", False,
                   expected="a clean worktree", observed=f"{len(dirty)} change(s): {summary}")
    else:
        # A local test build from a dirty tree is a normal thing to do; it just
        # can never be a release.
        gate.facts["releasable"] = False
        gate.record("repo.clean_worktree", SKIP,
                    f"{len(dirty)} uncommitted change(s) — NON-RELEASABLE (test class)")


def core_source_fingerprint(gate: Gate) -> str:
    """The fingerprint the current core/src produces, computed once per run.

    Both the source and the artifact paths need it, and only one of them may be
    running: --verify-artifact does not call source_gates at all. Caching it on
    the gate rather than setting it in one path and reading it in the other is
    what keeps artifact.core_provenance_pair able to pass on its own — it could
    not, until this existed.
    """
    cached = gate.facts.get("coreSourceFingerprint")
    if cached:
        return cached
    _, out = run(["go", "run", "./cmd/corefingerprint", "."],
                 cwd=REPO / "core/src", env={"GOOS": "", "GOARCH": ""})
    fingerprint = out.strip().splitlines()[-1] if out.strip() else ""
    if re.fullmatch(r"[0-9a-f]{64}", fingerprint):
        gate.facts["coreSourceFingerprint"] = fingerprint
        return fingerprint
    return ""


def source_gates(gate: Gate, run_tests: bool, release_class: str = "test"):
    worktree_gate(gate, release_class)
    _, code = tracked_version(gate)

    baseline_file = REPO / "android/release-baseline.properties"
    baseline = None
    for line in baseline_file.read_text(encoding="utf-8").splitlines():
        if line.startswith("lastAcceptedVersionCode="):
            baseline = int(line.split("=", 1)[1].strip())
    gate.facts["acceptedBaseline"] = baseline
    gate.check("version.baseline", baseline is not None and code is not None
               and code >= baseline,
               expected=f"versionCode >= {baseline}", observed=str(code))

    # The gitignored file must not carry version identity; that split is what
    # made a clean checkout build versionCode 1 against a released 55.
    local_properties = REPO / "android/local.properties"
    offenders = []
    if local_properties.is_file():
        offenders = [l for l in local_properties.read_text(encoding="utf-8").splitlines()
                     if l.startswith(("flutter.versionCode", "flutter.versionName"))]
    gate.check("version.no_local_override", not offenders,
               expected="local.properties declares no version",
               observed=", ".join(offenders) or "none")

    for lock in ("pubspec.lock", "core/src/go.sum",
                 "core/src/web/frontend/pnpm-lock.yaml"):
        gate.check(f"deps.lockfile:{lock}", (REPO / lock).is_file(),
                   expected="tracked", observed="present" if (REPO / lock).is_file() else "missing")

    # Reproducibility inputs: the resolver must exist and answer deterministically.
    resolver = REPO / "core/resolve-build-time.sh"
    if resolver.is_file():
        rc1, out1 = run([str(resolver)], env={"SOURCE_DATE_EPOCH": "1700000000"})
        rc2, out2 = run([str(resolver)], env={"SOURCE_DATE_EPOCH": "1700000000"})
        stable = rc1 == 0 and rc2 == 0 and out1.strip() == out2.strip()
        gate.check("build.reproducible_timestamp", stable,
                   expected="same epoch resolves identically",
                   observed=out1.strip() or "resolver failed")
        record_expected_build_time(gate)
        staged_build_time_gate(gate, release_class)
    else:
        gate.check("build.reproducible_timestamp", False,
                   expected="core/resolve-build-time.sh", observed="missing")

    # Staged Core must carry zero developer-machine paths.
    leaked = []
    for lib in CORE_LIBS:
        path = STAGED_CORE_DIR / lib
        if not path.is_file():
            leaked.append(f"{lib}: missing")
            continue
        rc, out = run(["sh", "-c",
                       f"strings -a '{path}' | grep -c -E '/home/|/Users/|/root/' || true"])
        if out.strip() not in ("0", ""):
            leaked.append(f"{lib}: {out.strip()}")
    gate.check("core.no_developer_paths", not leaked,
               expected="0 developer paths", observed=", ".join(leaked) or "0")

    # Staged Core must carry no toolchain VCS stamp.
    #
    # Go writes build.vcs.revision, build.vcs.time and build.vcs.modified into
    # the binary automatically, reading the enclosing repository's HEAD. That
    # bypasses core/resolve-build-time.sh completely, so identical build inputs
    # produced different bytes after any unrelated commit — including the commit
    # that stages the binaries, which means a staged Core could never be
    # reproduced from the commit containing it. -buildvcs=false removes it, and
    # this check is how we know the flag is still there: nothing else in the
    # tree fails if it is dropped, and the resulting binary looks correct.
    stamped = []
    for lib in CORE_LIBS:
        path = STAGED_CORE_DIR / lib
        if not path.is_file():
            stamped.append(f"{lib}: missing")
            continue
        # Go encodes build settings as "build\t<key>=<value>", so the tab is
        # what separates "build" from "vcs.revision" — anchoring on a literal
        # "build.vcs." matches nothing and the check passes on a stamped binary.
        rc, out = run(["sh", "-c",
                       f"strings -a '{path}' | grep -c -E 'build.vcs\\.(revision|time|modified)' || true"])
        if out.strip() not in ("0", ""):
            stamped.append(f"{lib}: {out.strip()}")
    gate.check("core.no_vcs_stamp", not stamped,
               expected="0 build.vcs stamps (-buildvcs=false)",
               observed=", ".join(stamped) or "0")

    for lib in CORE_LIBS:
        path = STAGED_CORE_DIR / lib
        if path.is_file():
            gate.facts.setdefault("stagedCore", {})[lib] = sha256(path)

    core_source_fingerprint(gate)

    # Source, not artifact, and deliberately above the --no-tests return: a new
    # active Pico identity has to be caught before it is compiled, because after
    # that the only evidence is a string inside a .so nobody greps. It reads the
    # tree and runs in under a second, so there is no reason to skip it.
    rc, out = run([sys.executable, str(REPO / "tool/no_active_pico.py")], cwd=REPO)
    summary = out.strip().splitlines()[-1] if out.strip() else "FAIL"
    gate.check("namespace.no_active_pico", rc == 0,
               expected="no unclassified Pico identity in owned production source",
               observed="PASS" if rc == 0 else summary)

    # Stray Chinese developer comments were found in PocketClaw's own Kotlin and
    # Dart. Localization, vendored upstream text and CJK test fixtures are
    # classified in the tool; anything else fails here.
    rc, out = run([sys.executable, str(REPO / "tool/cjk_hygiene.py")], cwd=REPO)
    summary = out.strip().splitlines()[-1] if out.strip() else "FAIL"
    gate.check("source.no_stray_cjk", rc == 0,
               expected="every tracked file with CJK text classified and within its count",
               observed="PASS" if rc == 0 else summary)

    # PC-DEF-064. The What's New screen and the published release notes were
    # two independent pieces of prose, so the app could describe a release the
    # notes did not. There is one source now, and this is what stops them
    # drifting apart again: the notes are rendered from the app's own release
    # structure and English strings, and a stale file fails here rather than
    # being noticed by whoever reads the published version.
    rc, out = run([sys.executable, str(REPO / "tool/release_notes.py"), "--check"],
                  cwd=REPO)
    summary = out.strip().splitlines()[-1] if out.strip() else "FAIL"
    gate.check("release.notes_match_whats_new", rc == 0,
               expected="published release notes rendered from the app's What's New source",
               observed=summary.replace("PASS ", "") if rc == 0 else summary)

    # Source side of the same rule. A dependency removed from the artifact but
    # left in pubspec would come back on the next `pub get`.
    proprietary = []
    for name, path in (("firebase_analytics", "pubspec.yaml"),
                       ("firebase_core", "pubspec.yaml"),
                       ("google_app_id", "android/app/src/main/AndroidManifest.xml")):
        target = REPO / path
        if target.is_file() and name in target.read_text(encoding="utf-8"):
            proprietary.append(f"{name} in {path}")
    gate.check("source.fdroid_no_proprietary_sdk", not proprietary,
               expected="no proprietary Google SDK declared in the build",
               observed=", ".join(proprietary) or "clean")

    # Whether a production signer has been enrolled at all. Reported rather than
    # failed: before the key ceremony "none" is the correct state, and a source
    # gate that went red for it would be red for weeks and stop being read.
    enrolled = enrolled_production_signer()
    gate.record("signing.enrolled_signer", PASS,
                detail=f"{enrolled[:16]}…" if enrolled
                else "none yet — production artifacts cannot pass until H2 enrolls one")

    # What the repository *says*, alongside what it does. A reader arriving at
    # the README should meet PocketClaw, not a rename in progress.
    rc, out = run([sys.executable, str(REPO / "tool/no_active_pico.py"), "--public"],
                  cwd=REPO)
    summary = out.strip().splitlines()[-1] if out.strip() else "FAIL"
    gate.check("namespace.no_public_pico", rc == 0,
               expected="no Pico branding on public surfaces outside attribution",
               observed="PASS" if rc == 0 else summary)

    if not run_tests:
        gate.record("tests", SKIP, "not requested (--no-tests)")
        return

    # Delegate to the authoritative guards rather than restating them.
    go_env = {"CGO_ENABLED": "0"}
    rc, out = run(["go", "test", "-tags", "stdjson goolm",
                   "./pkg/coresource/", "-run",
                   "TestStagedCoreWasBuiltFromTheCurrentSource"],
                  cwd=REPO / "core/src", env=go_env)
    gate.check("core.staged_freshness", rc == 0,
               expected="staged Core matches current source fingerprint",
               observed="PASS" if rc == 0 else out.strip().splitlines()[-1] if out.strip() else "FAIL")

    # The whole package, deliberately, rather than a list of test names. The
    # earlier name filter had already gone stale: the path-scoping tests that
    # are the core of this contract did not match it and were never run here.
    rc, out = run(["go", "test", "-tags", "stdjson goolm", "./pkg/coresource/"],
                  cwd=REPO / "core/src", env=go_env)
    gate.check("build.reproducibility_tests", rc == 0,
               expected="deterministic build plumbing",
               observed="PASS" if rc == 0 else out.strip().splitlines()[-1] if out.strip() else "FAIL")

    # PC-DEF-065. The ordered fresh-install path, as its own gate row.
    #
    # This project kept fixing one step and breaking the next while every
    # isolated test still passed: the store was right, the handler's rules were
    # right, the exposure rule was right, and a real first install still could
    # not create its first password -- the claim reconciliation closed the
    # listener carrying the setup response. Only the ordered journey shows that,
    # so it is named here rather than left to be one more test in a package.
    rc, out = run(["go", "test", "-tags", "stdjson goolm", "-run",
                   "TestFreshInstallJourney|TestJourney", "./web/backend/"],
                  cwd=REPO / "core/src", env=go_env)
    summary = out.strip().splitlines()[-1] if out.strip() else "FAIL"
    gate.check("journey.fresh_install", rc == 0,
               expected="the ordered fresh-install path still works end to end",
               observed="PASS" if rc == 0 else summary)

    rc, out = run(["go", "test", "-tags", "stdjson goolm",
                   "./pkg/pid/", "./pkg/logger/", "./pkg/config/",
                   "./pkg/channels/pocketclaw/", "./web/backend/dashboardauth/"],
                  cwd=REPO / "core/src", env=go_env)
    gate.check("a2.private_storage_contracts", rc == 0,
               expected="A2 credential, log and auth guards pass",
               observed="PASS" if rc == 0 else "FAIL")

    rc, out = run([sys.executable, str(REPO / "tool/test_build_hardened_android.py")])
    gate.check("dart.hardening_contract", rc == 0,
               expected="obfuscation, split-info, generated-URI and signing-mode guards",
               observed="PASS" if rc == 0 else out.strip().splitlines()[-1] if out.strip() else "FAIL")

    rc, out = run([sys.executable, str(REPO / "tool/test_r8_hardening.py")])
    gate.check("r8.hardening_contract", rc == 0,
               expected="minify/shrink enabled, narrow rules, fresh private mapping evidence",
               observed="PASS" if rc == 0 else out.strip().splitlines()[-1] if out.strip() else "FAIL")

    rc, out = run([sys.executable, str(REPO / "tool/test_native_elf_audit.py")])
    gate.check("native.elf_audit_contract", rc == 0,
               expected="read-only category-aware ELF audit regression suite",
               observed="PASS" if rc == 0 else out.strip().splitlines()[-1] if out.strip() else "FAIL")

    rc, out = run([sys.executable, str(REPO / "tool/test_native_support.py")])
    gate.check("native.private_support_contract", rc == 0,
               expected="root-independent native builds, private companions and source-level RUNPATH guard",
               observed="PASS" if rc == 0 else out.strip().splitlines()[-1] if out.strip() else "FAIL")


    flutter_suite_gates(gate)


# The named Dart contracts, and the file each is carried by. They remain
# individually reported because a release record that says only "the suite
# passed" loses which guarantee was checked — but they are now derived from the
# full-suite run rather than being the whole of it. Before PC-DEF-025 these
# three files *were* the Flutter gate, so the other 25 files could be red while
# the gate reported green, which is exactly what happened for five milestones.
FLUTTER_NAMED_CONTRACTS = (
    ("a1.contracts",
     ("test/unit/android_release_contract_test.dart",
      "test/unit/android_backup_exclusion_test.dart"),
     "signing fail-closed, version source, backup exclusions"),
    ("signing.production_contract",
     ("test/unit/production_signing_contract_test.dart",),
     "production signing fails closed and pins its signer"),
    ("a2.placement_guards",
     ("test/unit/android_runtime_secret_placement_test.dart",),
     "credential and logs app-private"),
)


def flutter_suite_gates(gate: Gate):
    """The complete Flutter suite is the acceptance criterion.

    One run, and every Flutter gate item is read from it. `flutter.suite` is the
    authoritative one: any failing test fails the gate, whatever file it is in.
    """
    flutter = find_flutter()
    if not flutter:
        gate.record("flutter.suite", SKIP, "flutter toolchain not found")
        for name, _, _ in FLUTTER_NAMED_CONTRACTS:
            gate.record(name, SKIP, "flutter toolchain not found")
        return

    gate.facts["flutterExecutable"] = str(flutter)
    rc, summary = run_flutter_suite(flutter)
    gate.facts["flutterPassed"] = summary["passed"]
    gate.facts["flutterFailed"] = summary["failed"]
    gate.facts["flutterFailingSuites"] = summary["failingSuites"]

    if rc == 0 and not summary["parsed"]:
        # Exit 0 with nothing parsed means the suite did not run. Reporting that
        # as a pass would be the failure mode this gate exists to remove.
        gate.check("flutter.suite", False,
                   expected="the complete Flutter suite runs and passes",
                   observed="flutter test produced no test results")
        for name, _, expected in FLUTTER_NAMED_CONTRACTS:
            gate.check(name, False, expected=expected,
                       observed="no Flutter results")
        return

    # The exit code is authoritative and the parsed counts are evidence. A
    # non-zero exit can never be reported as PASS even if nothing was parsed.
    ok = rc == 0 and summary["failed"] == 0
    failing = summary["failingTests"]
    detail = f"{summary['passed']} passed, {summary['failed']} failed"
    if not ok:
        shown = "; ".join(failing[:3]) if failing else f"flutter test exit {rc}"
        more = len(summary["failingSuites"]) - 3
        if more > 0:
            shown += f" (+{more} more suite(s))"
        detail = f"{detail} — {shown}"
    gate.check("flutter.suite", ok,
               expected="the complete Flutter suite passes",
               observed=detail)

    failing_suites = set(summary["failingSuites"])
    for name, files, expected in FLUTTER_NAMED_CONTRACTS:
        hit = sorted(path for path in failing_suites
                     if any(path.endswith(f) for f in files))
        gate.check(name, not hit, expected=expected,
                   observed="PASS" if not hit else ", ".join(hit))


# --------------------------------------------------------------------------
# Artifact-level gates
# --------------------------------------------------------------------------

def artifact_gates(gate: Gate, apk: Path, release_class: str,
                   dart_symbols: Path | None = None,
                   r8_mapping: Path | None = None):
    if not apk.is_file():
        gate.check("artifact.present", False, expected=str(apk), observed="missing")
        return
    gate.facts["apk"] = apk.name
    gate.facts["apkSha256"] = sha256(apk)
    gate.facts["apkBytes"] = apk.stat().st_size
    gate.check("artifact.present", True, observed=apk.name)

    aapt = find_sdk_tool("aapt2")
    if not aapt:
        gate.record("artifact.manifest", SKIP, "aapt2 not found")
    else:
        rc, badging = run([str(aapt), "dump", "badging", str(apk)])
        header = badging.splitlines()[0] if badging else ""
        package = re.search(r"name='([^']+)'", header)
        version_code = re.search(r"versionCode='(\d+)'", header)
        version_name = re.search(r"versionName='([^']+)'", header)
        gate.facts["packageId"] = package.group(1) if package else None
        gate.check("artifact.package", package and package.group(1) == PACKAGE_ID,
                   expected=PACKAGE_ID, observed=package.group(1) if package else "unknown")
        observed_code = int(version_code.group(1)) if version_code else None
        gate.check("artifact.version",
                   observed_code == gate.facts.get("versionCode")
                   and version_name and version_name.group(1) == gate.facts.get("versionName"),
                   expected=f"{gate.facts.get('versionName')}+{gate.facts.get('versionCode')}",
                   observed=f"{version_name.group(1) if version_name else '?'}+{observed_code}")

        rc, perms_out = run([str(aapt), "dump", "permissions", str(apk)])
        packaged = {line.split("name='")[1].split("'")[0]
                    for line in perms_out.splitlines()
                    if line.startswith("uses-permission") and "name='" in line}
        gate.facts["permissionCount"] = len(packaged)
        gate.facts["permissionContract"] = "release-hardening-a1"
        unexpected = packaged - EXPECTED_PERMISSIONS
        missing = EXPECTED_PERMISSIONS - packaged
        gate.check("artifact.permissions", not unexpected and not missing,
                   expected=f"{len(EXPECTED_PERMISSIONS)} permissions (A1 contract)",
                   observed=f"extra={sorted(unexpected)} missing={sorted(missing)}"
                   if (unexpected or missing) else f"{len(packaged)} as expected")
        present_forbidden = packaged & FORBIDDEN_PERMISSIONS
        gate.check("artifact.forbidden_permissions_absent", not present_forbidden,
                   expected="none of the A1-removed permissions",
                   observed=", ".join(sorted(present_forbidden)) or "none")

    with zipfile.ZipFile(apk) as archive:
        names = set(archive.namelist())

        abis = {n.split("/")[1] for n in names if n.startswith("lib/") and n.count("/") >= 2}
        gate.facts["abi"] = sorted(abis)
        # arm64-v8a is the only ABI PocketClaw can run on: Core, the Managed
        # Runtime and libflutter/libapp exist for it alone. Plugin stubs for
        # other ABIs used to be packaged too, which made the APK advertise
        # armeabi-v7a and x86_64 -- F-Droid would have offered it to devices
        # where it crashes on launch. abiFilters removes them; any other ABI
        # directory is now a failure, not a recorded baseline.
        extra_abis = sorted(abis - {EXPECTED_ABI})
        gate.facts["extraAbis"] = extra_abis
        gate.check("artifact.abi", abis == {EXPECTED_ABI},
                   expected=f"native libraries under {EXPECTED_ABI} only",
                   observed=", ".join(extra_abis) if extra_abis
                   else EXPECTED_ABI if abis else "no native libraries")

        # Packaged Core must be byte-identical to what is staged in the tree.
        mismatched = []
        packaged_core = {}
        for lib in CORE_LIBS:
            entry = f"lib/{EXPECTED_ABI}/{lib}"
            staged = STAGED_CORE_DIR / lib
            if entry not in names or not staged.is_file():
                mismatched.append(f"{lib}: missing")
                continue
            digest = hashlib.sha256(archive.read(entry)).hexdigest()
            packaged_core[lib] = digest
            if digest != sha256(staged):
                mismatched.append(f"{lib}: differs from staged")
        gate.facts["packagedCore"] = packaged_core

        # The fingerprint the artifact actually carries, read from the stamp in
        # the packaged binaries rather than recomputed from a tree that may have
        # moved on since the build.
        #
        # Both shipping binaries, because both are the product. Reading only
        # libpocketclaw.so is the hole N4K-A closed: the dashboard is compiled
        # separately from core/src/web and can be from other source entirely
        # while the Core's stamp looks right.
        core_entry = f"lib/{EXPECTED_ABI}/libpocketclaw.so"
        expected = core_source_fingerprint(gate)
        carried = {}
        unstamped = []
        for lib in CORE_LIBS:
            entry = f"lib/{EXPECTED_ABI}/{lib}"
            if entry not in names:
                continue
            blob = archive.read(entry)
            if expected and expected.encode() in blob:
                carried[lib] = expected
            elif re.search(rb"[0-9a-f]{64}", blob):
                carried[lib] = "present (not matched to source)"
                unstamped.append(lib)
            else:
                carried[lib] = "absent"
                unstamped.append(lib)
        gate.facts["packagedCoreFingerprint"] = carried

        # One provenance unit: both binaries must carry the *same* fingerprint,
        # and it must be the one the current source produces.
        gate.check("artifact.core_provenance_pair",
                   bool(carried) and not unstamped and len(set(carried.values())) == 1,
                   expected="both shipping binaries stamped with the current source fingerprint",
                   observed=", ".join(f"{k}: {v}" for k, v in sorted(carried.items()))
                   or "no Core binaries packaged")

        gate.check("artifact.core_matches_staged", not mismatched,
                   expected="packaged Core byte-identical to staged Core",
                   observed=", ".join(mismatched) or "identical")

        if core_entry in names:
            if "buildTimeExpected" not in gate.facts:
                record_expected_build_time(gate)
            build_time_gate(gate, archive.read(core_entry), release_class)
        else:
            gate.check("artifact.build_time", False,
                       expected="a packaged Core to read BuildTime from",
                       observed="missing")

        # Managed Runtime payload and the Python stdlib survival check are
        # already enforced by the Gradle packaging verifiers; re-assert presence
        # here so an artifact built elsewhere cannot skip them.
        # CORE_LIBS is the exclusion authority rather than a second literal:
        # libpocketclaw-web.so matches the Managed Runtime prefix but is Core's
        # launcher, and counting it inflated this to 9. That mattered because
        # the threshold is a floor — seven real tools plus the launcher would
        # also have reached 8 and the check would have stopped noticing a
        # dropped payload.
        core_names = set(CORE_LIBS)
        runtime_libs = [n for n in names
                        if n.startswith(f"lib/{EXPECTED_ABI}/libpocketclaw-")
                        and Path(n).name not in core_names]
        gate.check("artifact.managed_runtime", len(runtime_libs) >= 8,
                   expected=">=8 managed runtime payloads",
                   observed=str(len(runtime_libs)))

        python_entry = f"lib/{EXPECTED_ABI}/libpocketclaw-python.so"
        if python_entry in names:
            blob = archive.read(python_entry)
            has_eocd = b"PK\x05\x06" in blob[-65557:]
            has_bootstrap = b"pocketclaw_bootstrap.py" in blob
            gate.check("artifact.python_payload", has_eocd and has_bootstrap,
                       expected="appended stdlib + bootstrap entry point",
                       observed=f"eocd={has_eocd} bootstrap={has_bootstrap}")
        else:
            gate.check("artifact.python_payload", False,
                       expected=python_entry, observed="missing")

        # Locale coverage, read from the Dart snapshot the app actually ships.
        libapp = f"lib/{EXPECTED_ABI}/libapp.so"
        if libapp in names:
            blob = archive.read(libapp)
            found = 0
            for arb in sorted((REPO / "lib/l10n").glob("app_*.arb")):
                text = json.loads(arb.read_text(encoding="utf-8"))["aboutDescription"]
                encodings = [text.encode("utf-8"), text.encode("utf-16-le")]
                try:
                    encodings.append(text.encode("latin-1"))
                except UnicodeEncodeError:
                    pass
                found += any(e in blob for e in encodings)
            gate.facts["locales"] = found
            gate.check("artifact.locales", found == EXPECTED_LOCALES,
                       expected=str(EXPECTED_LOCALES), observed=str(found))
        else:
            gate.record("artifact.locales", SKIP, "libapp.so not packaged")

        # Backup exclusions, read from the packaged binary resources.
        if aapt:
            excluded = set()
            for entry in (n for n in names if re.fullmatch(r"res/[A-Za-z0-9_]+\.xml", n)):
                rc, tree = run([str(aapt), "dump", "xmltree", str(apk), "--file", entry])
                if "full-backup-content" in tree or "data-extraction-rules" in tree:
                    excluded.update(re.findall(r'path="([^"]+)"', tree))
            gate.check("artifact.backup_exclusions",
                       {"credentials/", "picoclaw/"} <= excluded,
                       expected="credentials/ and picoclaw/ excluded",
                       observed=", ".join(sorted(excluded)) or "none")
        else:
            gate.record("artifact.backup_exclusions", SKIP, "aapt2 not found")

    native_gates(gate, apk, dart_symbols)
    fdroid_artifact_gate(gate, apk)
    signing_gate(gate, apk, release_class)
    if r8_mapping is not None:
        r8_hardening_gates(gate, apk, r8_mapping)


def deobfuscation_privacy_gate(gate: Gate, artifact: Path):
    """Is anything that could deobfuscate the shipped code inside the artifact?

    This is a property of the artifact alone, so it runs on its own rather than
    behind ``--r8-mapping``: an APK built without mapping evidence to compare
    against is exactly the one nobody would notice shipping a mapping. It also
    runs before the R8 contract, which raises and returns early — this check
    was previously unreachable in the one case it was written for, and reported
    a hardcoded True with the observation "absent from APK". See PC-DEF-021.
    """
    with zipfile.ZipFile(artifact) as archive:
        names = archive.namelist()
    packaged = packaged_r8_mapping_entries(names)
    gate.facts["r8MappingPackagedEntries"] = packaged
    gate.facts["r8MappingScannedEntries"] = len(names)
    gate.check(
        "artifact.r8_mapping_private", not packaged,
        expected="no R8 deobfuscation entry packaged in the artifact",
        observed=(", ".join(packaged) if packaged else
                  f"{len(names)} archive entries scanned, deobfuscation entries = 0"),
    )


def distribution_gates(gate: Gate, artifact: Path, distribution: str | None):
    """Decide whether this artifact format may serve this distribution purpose.

    The first gate any artifact meets, and the one PC-DEF-021 was missing. It is
    about purpose rather than contents: a bundle with no mapping at all is still
    forbidden as a public asset, because what makes the bundle unpublishable is
    what the format is for.
    """
    try:
        kind = detect_artifact_kind(artifact)
    except ArtifactPolicyError as error:
        gate.check("artifact.kind", False,
                   expected="an APK or an AAB", observed=str(error))
        return None
    gate.facts["artifactKind"] = kind
    gate.check("artifact.kind", True, observed=kind.upper())

    gate.facts["distributionClass"] = distribution
    ok, message = distribution_verdict(kind, distribution or "")
    gate.check("artifact.distribution_class", ok,
               expected="a declared distribution class the format may serve",
               observed=message)
    return kind


def bundle_gates(gate: Gate, bundle: Path, distribution: str):
    """Inspect an AAB and hold it to its declared purpose.

    Expected AGP metadata is inventoried deliberately rather than ignored: for a
    Play upload the mapping and native symbols under BUNDLE-METADATA/ are the
    point, and a gate that stayed silent about them would teach a reader that
    the bundle contains no such thing.
    """
    inventory = inspect_bundle(bundle)
    gate.facts["bundle"] = bundle.name
    gate.facts["bundleSha256"] = sha256(bundle)
    gate.facts["bundleBytes"] = bundle.stat().st_size
    gate.facts["bundleModules"] = inventory["modules"]
    gate.facts["bundleAbis"] = inventory["abis"]
    gate.facts["bundleMetadata"] = inventory["bundleMetadata"]
    gate.facts["bundleMetadataBytes"] = inventory["bundleMetadataBytes"]

    gate.check("bundle.modules", bool(inventory["modules"]),
               expected="at least one bundle module",
               observed=", ".join(inventory["modules"]) or "none")
    gate.check("bundle.manifests", bool(inventory["manifests"]),
               expected="a module manifest",
               observed=f"{len(inventory['manifests'])} manifest(s)")

    abis = list(inventory["abis"])
    product = sorted(
        name for name, _ in inventory["nativeEntries"]
        if "/libpocketclaw" in name and "/arm64-v8a/" not in name
    )
    gate.check("bundle.abi", not product,
               expected="the PocketClaw product payload is arm64-v8a only",
               observed=f"{', '.join(abis)}" if not product
               else f"non-arm64 product payload: {', '.join(product)}")
    gate.facts["bundleNativeEntryCount"] = len(inventory["nativeEntries"])
    gate.check("bundle.native_inventory", bool(inventory["nativeEntries"]),
               expected="packaged native entries",
               observed=f"{len(inventory['nativeEntries'])} entries across {len(abis)} ABI(s)")

    metadata = inventory["bundleMetadata"]
    unrecognised = [item["entry"] for item in metadata if not item["allowedForPlayUpload"]]
    mapping_entries = [item for item in metadata
                       if item["category"] == "r8 deobfuscation mapping"]
    symbol_entries = [item for item in metadata if item["category"] == "native debug symbols"]

    if distribution == PLAY_UPLOAD:
        # Expected, verified, and named — not tolerated by silence.
        gate.check("bundle.play_metadata_expected", bool(mapping_entries or symbol_entries),
                   expected="AGP release-support metadata Play consumes",
                   observed=(f"{len(mapping_entries)} mapping + {len(symbol_entries)} "
                             f"debug-symbol entries, {inventory['bundleMetadataBytes']:,} bytes "
                             "— allowed and expected for a Play upload"))
    else:
        gate.record("bundle.play_metadata_expected", SKIP,
                    f"not a Play upload (class: {distribution})")

    gate.check("bundle.metadata_recognised", not unrecognised,
               expected="every BUNDLE-METADATA entry is known AGP output",
               observed=", ".join(unrecognised) or f"{len(metadata)} entries, all recognised")

    leaks = private_material_violations(inventory["entryNames"], kind=AAB)
    gate.check("bundle.no_unrelated_private_material", not leaks,
               expected="no packaged private material beyond expected AGP metadata",
               observed="; ".join(f"{name} ({reason})" for name, reason in leaks)
               or "none")

    if distribution == NON_PUBLISH_AUDIT:
        gate.facts["bundleClassificationNotice"] = (
            "NOT PLAY-READY; NOT PUBLIC-RELEASE-SAFE; NOT A GITHUB RELEASE ASSET"
        )
        gate.check("bundle.non_publish_notice", True,
                   observed="NOT PLAY-READY / NOT PUBLIC-RELEASE-SAFE / NOT A GITHUB RELEASE ASSET")


def release_asset_gates(gate: Gate, assets):
    """Hold a proposed set of public release assets to the allowlist."""
    names = [str(a) for a in assets]
    gate.facts["publicReleaseAssets"] = names
    violations = public_release_asset_violations(names)
    gate.check("release.public_asset_allowlist", not violations,
               expected="only public-safe assets attached to a public release",
               observed="; ".join(f"{name} ({reason})" for name, reason in violations)
               or f"{len(names)} asset(s), all permitted")


def r8_hardening_gates(gate: Gate, apk: Path, mapping: Path):
    """Require effective R8 output and keep its deobfuscation data private."""
    try:
        evidence = inspect_r8_outputs(apk, mapping)
    except R8ContractError as error:
        gate.check("artifact.r8_contract", False,
                   expected="fresh, effective, private R8 output",
                   observed=str(error))
        return
    gate.facts.update(evidence)
    gate.check("artifact.r8_mapping", True,
               observed=f"{mapping.name}: {evidence['r8MappingBytes']} bytes")
    gate.check("artifact.r8_shrinking", True,
               observed=f"usage.txt: {evidence['r8UsageBytes']} bytes")
    gate.check("artifact.r8_obfuscation", True,
               observed=(f"{len(evidence['r8RenamedInternalClasses'])} PocketClaw classes renamed; "
                         f"{len(evidence['r8RemovedOrFoldedInternalClasses'])} removed/folded"))
    gate.check("artifact.r8_entry_points", True,
               observed=f"{len(evidence['r8RequiredEntryPoints'])} manifest components preserved")


def native_gates(gate: Gate, apk: Path, dart_symbols: Path | None = None):
    """ELF hardening for every native library, and build-path privacy for ours.

    The strip / non-executable-stack / alignment rules apply to everything the
    APK ships. The developer-path rule does not, and being precise about that
    matters: `libpocketclaw-gh.so` legitimately carries `/home/runner/work/`
    from upstream's own CI, and Go module paths from its dependencies. Those are
    not this machine's paths and are not ours to fix. What must be zero is Core,
    which the build script already asserts with -trimpath, and which this
    re-asserts against the packaged copy.

    `libapp.so` is scanned separately and reported rather than failed: the Dart
    snapshot embeds a generated-source URI, which is a known, tracked,
    unresolved item. Failing here would block every build on something this
    milestone deliberately does not fix; passing silently would pretend it is
    solved. It is named instead.
    """
    if not shutil.which("readelf"):
        gate.record("artifact.native_hardening", SKIP, "readelf not available")
        return
    import tempfile
    problems = []
    dart_snapshot_paths = 0
    dart_app_blob = None
    archive_names = []
    with tempfile.TemporaryDirectory() as tmp, zipfile.ZipFile(apk) as archive:
        archive_names = archive.namelist()
        for entry in (n for n in archive.namelist()
                      if n.startswith("lib/") and n.endswith(".so")):
            name = Path(entry).name
            path = Path(tmp) / name
            blob = archive.read(entry)
            path.write_bytes(blob)
            if name == "libapp.so" and entry == "lib/arm64-v8a/libapp.so":
                dart_app_blob = blob

            rc, out = run(["readelf", "-lW", str(path)])
            aligns = re.findall(r"LOAD.*?(0x[0-9a-f]+)\s*$", out, re.M)
            max_align = max((int(a, 16) for a in aligns), default=0)
            if max_align and max_align < 0x4000:
                problems.append(f"{name}: align 0x{max_align:x} < 16 KiB")
            if re.search(r"GNU_STACK.*RWE", out):
                problems.append(f"{name}: executable stack")
            rc, out = run(["sh", "-c",
                           f"file '{path}' | grep -c 'not stripped' || true"])
            if out.strip() not in ("0", ""):
                problems.append(f"{name}: not stripped")

            rc, out = run(["sh", "-c",
                           f"strings -a '{path}' | grep -c -E '/home/|/Users/|/root/' || true"])
            count = out.strip()
            if count not in ("0", ""):
                if name in CORE_LIBS:
                    problems.append(f"{name}: {count} developer paths")
                elif name == "libapp.so":
                    dart_snapshot_paths = int(count)

    gate.check("artifact.native_hardening", not problems,
               expected="stripped, nx-stack, >=16 KiB aligned; Core free of developer paths",
               observed="; ".join(problems) or "all native libraries clean")

    if dart_snapshot_paths:
        gate.facts.setdefault("pendingObserved", []).append(
            f"libapp.so embeds {dart_snapshot_paths} generated-source path(s)")
        gate.record("artifact.dart_snapshot_paths", SKIP,
                    f"{dart_snapshot_paths} generated-source path(s) — "
                    "PENDING_FINAL_HARDENING (controlled Dart generated-source URI strategy)")
    else:
        gate.record("artifact.dart_snapshot_paths", PASS, "no generated-source paths")

    if dart_symbols is not None:
        dart_hardening_gates(gate, dart_app_blob, archive_names, dart_symbols)


def dart_hardening_gates(gate: Gate, app_blob: bytes | None,
                         archive_names: list[str], symbols_path: Path):
    """Prove H3 Dart hardening using the APK and its private support artifact."""
    symbols = symbols_path
    if symbols.is_dir():
        symbols = symbols / "app.android-arm64.symbols"
    present = symbols.is_file() and symbols.stat().st_size > 0
    symbol_blob = symbols.read_bytes() if present else b""
    split_shape = (
        present
        and symbol_blob.startswith(b"\x7fELF")
        and b".debug_info" in symbol_blob
        and b".debug_line" in symbol_blob
    )
    retained = [marker for marker in DART_APP_SYMBOL_MARKERS if marker in symbol_blob]
    gate.check("artifact.dart_split_debug_info", split_shape and len(retained) >= 3,
               expected="external arm64 ELF with Dart DWARF and application symbols",
               observed=(f"{symbols.name}: {len(retained)} known private symbols retained"
                         if present else f"missing: {symbols}"))

    generated_uri = app_blob is not None and DART_GENERATED_REGISTRANT_URI in app_blob
    gate.check("artifact.dart_generated_source_uri", generated_uri,
               expected=DART_GENERATED_REGISTRANT_URI.decode(),
               observed="controlled package URI" if generated_uri else "missing")

    exposed = ([marker.decode() for marker in DART_APP_SYMBOL_MARKERS if marker in app_blob]
               if app_blob is not None else ["libapp.so missing"])
    gate.check("artifact.dart_obfuscation", not exposed and len(retained) >= 3,
               expected="application names absent from libapp.so and retained privately",
               observed="obfuscated" if not exposed and len(retained) >= 3
               else ", ".join(exposed) or "private symbol evidence insufficient")

    packaged = [name for name in archive_names
                if name.endswith((".symbols", ".dwarf")) or "private-symbols" in name]
    tracked = False
    if present:
        try:
            relative = symbols.resolve().relative_to(REPO)
        except ValueError:
            relative = None
        if relative is not None:
            rc, _ = run(["git", "ls-files", "--error-unmatch", str(relative)])
            tracked = rc == 0
    gate.check("artifact.dart_symbols_private", not packaged and not tracked,
               expected="split debug info external to APK and untracked",
               observed=(", ".join(packaged) if packaged else
                         "tracked by Git" if tracked else "external and untracked"))
    if present:
        gate.facts["dartSymbols"] = str(symbols)
        gate.facts["dartSymbolsBytes"] = symbols.stat().st_size
        gate.facts["dartSymbolsSha256"] = sha256(symbols)


# Proprietary SDKs that disqualify an app from the official F-Droid repository.
# Matched against DEX class names and packaged entries, because F-Droid judges
# what is in the binary — a runtime feature flag is not an answer to it.
PROPRIETARY_SDK_MARKERS = {
    "firebase": rb"com/google/firebase",
    "gms": rb"com/google/android/gms",
    "admob": rb"com/google/android/gms/ads",
    "measurement": rb"com/google/android/gms/measurement",
}


def fdroid_artifact_gate(gate: Gate, apk: Path):
    """No proprietary Google SDK in the packaged artifact.

    Reported per marker rather than as one verdict, so a regression names the
    thing that came back instead of saying "F-Droid: no".
    """
    with zipfile.ZipFile(apk) as archive:
        names = archive.namelist()
        dex = b"".join(archive.read(n) for n in names if n.endswith(".dex"))

    for label, marker in PROPRIETARY_SDK_MARKERS.items():
        in_dex = marker in dex
        token = marker.decode().rsplit("/", 1)[-1]
        in_entries = [n for n in names if token in n.lower()]
        gate.check(f"artifact.fdroid_no_{label}", not in_dex and not in_entries,
                   expected=f"no {label} SDK packaged",
                   observed="clean" if not in_dex and not in_entries
                   else f"present ({'dex' if in_dex else ''}"
                        f"{' and ' if in_dex and in_entries else ''}"
                        f"{f'{len(in_entries)} entries' if in_entries else ''})")


def signing_gate(gate: Gate, apk: Path, release_class: str):
    apksigner = find_sdk_tool("apksigner")
    if not apksigner:
        gate.record("artifact.signing", SKIP, "apksigner not found")
        return
    # apksigner is a shell wrapper around a JAR, so it needs a JDK on PATH or
    # JAVA_HOME set. Without this the gate reported "unsigned" for a perfectly
    # well-signed APK, which is exactly the kind of false failure that teaches
    # people to ignore a gate.
    env = {}
    if not os.environ.get("JAVA_HOME") and not shutil.which("java"):
        jdk = find_jdk()
        if jdk:
            env["JAVA_HOME"] = str(jdk)
            env["PATH"] = f"{jdk / 'bin'}{os.pathsep}{os.environ.get('PATH', '')}"
    rc, out = run([str(apksigner), "verify", "--print-certs", "--verbose", str(apk)], env=env)

    # Recorded, not enforced. Which schemes AGP emits depends on minSdk and on
    # the signing config, and hard-failing on a scheme here would either
    # duplicate a decision that belongs in the build or invent a requirement
    # nothing has agreed to. The fact is worth having in the report; the
    # judgement belongs to whoever reads it.
    schemes = sorted(
        name for name, ok in re.findall(
            r"Verified using (v[\d.]+) scheme[^:]*:\s*(true|false)", out)
        if ok == "true")
    if schemes:
        gate.facts["signatureSchemes"] = schemes
    match = re.search(r"certificate SHA-256 digest:\s*([0-9a-f]+)", out)
    if rc != 0 or not match:
        first_line = next((l for l in out.splitlines() if l.strip()), "no output")
        gate.check("artifact.signing", False,
                   expected="a verifiable signature",
                   observed=f"apksigner failed: {first_line.strip()[:120]}")
        return
    signers = re.findall(r"certificate SHA-256 digest:\s*([0-9a-f]+)", out)
    signer = match.group(1)
    gate.facts["signerSha256"] = signer
    is_dev = signer == DEV_SIGNER_SHA256

    if release_class == "production":
        enrolled = enrolled_production_signer()
        gate.facts["enrolledProductionSigner"] = enrolled or "none"

        # Four separate ways to be wrong, reported as the one that applies.
        # The digest is authoritative throughout: a certificate subject is
        # attacker-chosen text, so "CN=Android Debug" is a hint and never a
        # check.
        if is_dev:
            reason = "development signer — never a production identity"
        elif enrolled is None:
            reason = ("no production signer enrolled yet: "
                      "android/release-signing-cert.sha256 carries no digest")
        elif len(set(signers)) > 1:
            reason = f"{len(set(signers))} distinct signers; exactly one is expected"
        elif signer != enrolled:
            reason = "signer does not match the enrolled production certificate"
        else:
            reason = None

        gate.check("artifact.signing", reason is None,
                   expected="signed by the enrolled PocketClaw production certificate",
                   observed=reason or "enrolled production signer")
        gate.facts["releasable"] = reason is None
    else:
        gate.record("artifact.signing", PASS,
                    detail="LOCAL TEST / NON-RELEASABLE — development signer"
                    if is_dev else
                    "LOCAL TEST / NON-RELEASABLE — non-development signer")
        gate.facts["releasable"] = False


# --------------------------------------------------------------------------

def main() -> int:
    parser = argparse.ArgumentParser(description="PocketClaw release gate")
    parser.add_argument("--verify-source", action="store_true")
    parser.add_argument("--verify-artifact", metavar="APK")
    parser.add_argument("--full", metavar="APK")
    parser.add_argument("--release-class", choices=("test", "production"),
                        default="test",
                        help="test permits the development signer and can never "
                             "report a production release; production rejects it")
    parser.add_argument("--no-tests", action="store_true",
                        help="skip delegated test suites (source phase)")
    parser.add_argument("--manifest", metavar="PATH",
                        help="write the JSON release manifest here")
    parser.add_argument(
        "--dart-symbols", metavar="PATH",
        help="verify Dart hardening against this private .symbols file or directory",
    )
    parser.add_argument(
        "--r8-mapping", metavar="PATH",
        help="verify R8 hardening against this private mapping.txt",
    )
    parser.add_argument(
        "--verify-bundle", metavar="AAB",
        help="inspect an Android App Bundle and hold it to its distribution class",
    )
    parser.add_argument(
        "--artifact-class", choices=DISTRIBUTION_CLASSES, default=None,
        help="what the artifact is FOR. Required for any artifact phase and "
             "deliberately has no default: an unclassified artifact fails "
             "closed rather than being assumed public-safe. "
             f"{PUBLIC_RELEASE}=published; {PLAY_UPLOAD}=Google Play upload "
             f"only; {NON_PUBLISH_AUDIT}=inspection evidence only",
    )
    parser.add_argument(
        "--release-assets", metavar="NAME", nargs="+", default=None,
        help="check these proposed public release asset names against the allowlist",
    )
    args = parser.parse_args()

    if not (args.verify_source or args.verify_artifact or args.full
            or args.verify_bundle or args.release_assets):
        parser.error("choose --verify-source, --verify-artifact <apk>, --full <apk>, "
                     "--verify-bundle <aab>, or --release-assets <name...>")

    # Fail closed. An artifact whose purpose was not stated cannot be judged,
    # and guessing "public" would be the permissive guess in the one place
    # PC-DEF-021 proved that is unsafe.
    if (args.verify_artifact or args.full or args.verify_bundle) and not args.artifact_class:
        parser.error(
            "--artifact-class is required with an artifact phase and has no default: "
            "an unclassified artifact fails closed. Choose one of "
            + ", ".join(DISTRIBUTION_CLASSES)
        )

    gate = Gate()
    gate.facts["releaseClass"] = args.release_class
    gate.facts["pendingFinalHardening"] = PENDING_FINAL_HARDENING

    if args.verify_source or args.full:
        source_gates(gate, run_tests=not args.no_tests,
                     release_class=args.release_class)
    apk = args.full or args.verify_artifact
    if apk:
        if "versionCode" not in gate.facts:
            # The artifact is compared against the tracked version even when the
            # source phase was not requested, so an APK can never be validated
            # against nothing.
            tracked_version(gate)
        apk_path = Path(apk).resolve()
        if distribution_gates(gate, apk_path, args.artifact_class) is not None:
            deobfuscation_privacy_gate(gate, apk_path)
            artifact_gates(
                gate,
                apk_path,
                args.release_class,
                Path(args.dart_symbols).resolve() if args.dart_symbols else None,
                Path(args.r8_mapping).resolve() if args.r8_mapping else None,
            )

    if args.verify_bundle:
        bundle_path = Path(args.verify_bundle).resolve()
        if distribution_gates(gate, bundle_path, args.artifact_class) is not None:
            bundle_gates(gate, bundle_path, args.artifact_class)

    if args.release_assets:
        release_asset_gates(gate, args.release_assets)

    width = max(len(r.name) for r in gate.results)
    print(f"PocketClaw release gate — class: {args.release_class}")
    print("-" * (width + 40))
    for result in gate.results:
        line = f"{result.status:8} {result.name:{width}}"
        if result.status == FAIL:
            line += f"  expected: {result.expected} | observed: {result.observed}"
        elif result.detail:
            line += f"  {result.detail}"
        elif result.observed:
            line += f"  {result.observed}"
        print(line)
    print("-" * (width + 40))

    gate.facts["result"] = "FAIL" if gate.failed else "PASS"
    if args.manifest:
        Path(args.manifest).write_text(
            json.dumps(gate.facts, indent=2, sort_keys=True) + "\n", encoding="utf-8")
        print(f"release manifest written to {args.manifest}")

    if gate.failed:
        print(f"\n{len(gate.failed)} gate(s) FAILED:")
        for result in gate.failed:
            print(f"  {result.name}: expected {result.expected}, observed {result.observed}")
        return 1

    if args.release_class == "test":
        print("\nPASS — local test candidate. NOT releasable as production.")
    elif args.verify_source:
        # Source mode checks the tree, not a package. Calling this a verified
        # production artifact would claim the artifact gates ran when no
        # artifact was even named — the signing, permission, ABI, packaged-Core
        # and payload checks all live in --verify-artifact.
        print("\nPASS — production source contracts. "
              "NOT a verified production artifact: run --full <apk> for that.")
    else:
        print("\nPASS — production release candidate.")
    print("Pending final hardening: " + "; ".join(PENDING_FINAL_HARDENING))
    return 0


if __name__ == "__main__":
    sys.exit(main())
