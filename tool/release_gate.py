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
    --release-class test|production

`test` permits the known development signer and can never report a production
release. `production` treats a development signer as an unconditional failure.
The caller chooses; the gate never guesses from context.

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
    "android.permission.FOREGROUND_SERVICE_DATA_SYNC",
    "android.permission.FOREGROUND_SERVICE_SPECIAL_USE",
    "android.permission.INTERNET",
    "android.permission.MANAGE_EXTERNAL_STORAGE",
    "android.permission.POST_NOTIFICATIONS",
    "android.permission.READ_EXTERNAL_STORAGE",
    "android.permission.RECEIVE_BOOT_COMPLETED",
    "android.permission.VIBRATE",
    "android.permission.WAKE_LOCK",
    "android.permission.WRITE_EXTERNAL_STORAGE",
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
}

CORE_LIBS = ("libpocketclaw.so", "libpocketclaw-web.so")
STAGED_CORE_DIR = REPO / "android/app/src/main/jniLibs" / EXPECTED_ABI

# The Android debug signing certificate this project's local test builds carry.
# Recorded so a development artifact can be *classified*, never so it can be
# accepted as a release.
DEV_SIGNER_SHA256 = "15cf75f9945d5354e75707e0326b7cffc60ac51a68df38156db318ef4578a27c"

# Real work that is not done yet. The gate names these rather than implying the
# release is fully hardened; it must not pretend they are solved.
PENDING_FINAL_HARDENING = [
    "production signing key not created",
    "Dart obfuscation and split debug info not enabled",
    "R8 keep rules not narrowed",
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

    rc, out = run(["go", "test", "-tags", "stdjson goolm",
                   "./pkg/pid/", "./pkg/logger/", "./pkg/config/",
                   "./pkg/channels/pocketclaw/", "./web/backend/dashboardauth/"],
                  cwd=REPO / "core/src", env=go_env)
    gate.check("a2.private_storage_contracts", rc == 0,
               expected="A2 credential, log and auth guards pass",
               observed="PASS" if rc == 0 else "FAIL")


    flutter = shutil.which("flutter")
    if not flutter:
        gate.record("a1.contracts", SKIP, "flutter not on PATH")
        gate.record("a2.placement_guards", SKIP, "flutter not on PATH")
    else:
        rc, out = run([flutter, "test",
                       "test/unit/android_release_contract_test.dart",
                       "test/unit/android_backup_exclusion_test.dart"])
        gate.check("a1.contracts", rc == 0,
                   expected="signing fail-closed, version source, backup exclusions",
                   observed="PASS" if rc == 0 else "FAIL")
        rc, out = run([flutter, "test",
                       "test/unit/android_runtime_secret_placement_test.dart"])
        gate.check("a2.placement_guards", rc == 0,
                   expected="credential and logs app-private",
                   observed="PASS" if rc == 0 else "FAIL")


# --------------------------------------------------------------------------
# Artifact-level gates
# --------------------------------------------------------------------------

def artifact_gates(gate: Gate, apk: Path, release_class: str):
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
        # The contract is that the canonical *product* payload — Core and the
        # Managed Runtime — is arm64 only. Flutter plugins ship small stubs for
        # other ABIs (libdartjni, libdatastore_shared_counter); those are an
        # accepted baseline and are recorded rather than failed.
        product_prefixes = ("libpocketclaw",)
        misplaced = sorted(
            n for n in names
            if n.startswith("lib/") and n.count("/") >= 2
            and Path(n).name.startswith(product_prefixes)
            and n.split("/")[1] != EXPECTED_ABI
        )
        stub_abis = sorted(abis - {EXPECTED_ABI})
        gate.facts["nonProductStubAbis"] = stub_abis
        gate.check("artifact.abi", EXPECTED_ABI in abis and not misplaced,
                   expected=f"product payload only under {EXPECTED_ABI}",
                   observed=", ".join(misplaced) if misplaced
                   else f"{EXPECTED_ABI} (+ plugin stubs: {', '.join(stub_abis) or 'none'})")

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

    native_gates(gate, apk)
    signing_gate(gate, apk, release_class)


def native_gates(gate: Gate, apk: Path):
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
    with tempfile.TemporaryDirectory() as tmp, zipfile.ZipFile(apk) as archive:
        for entry in (n for n in archive.namelist()
                      if n.startswith("lib/") and n.endswith(".so")):
            name = Path(entry).name
            path = Path(tmp) / name
            path.write_bytes(archive.read(entry))

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
    rc, out = run([str(apksigner), "verify", "--print-certs", str(apk)], env=env)
    match = re.search(r"certificate SHA-256 digest:\s*([0-9a-f]+)", out)
    if rc != 0 or not match:
        first_line = next((l for l in out.splitlines() if l.strip()), "no output")
        gate.check("artifact.signing", False,
                   expected="a verifiable signature",
                   observed=f"apksigner failed: {first_line.strip()[:120]}")
        return
    signer = match.group(1)
    gate.facts["signerSha256"] = signer
    is_dev = signer == DEV_SIGNER_SHA256

    if release_class == "production":
        # No heuristics: a development signer is an unconditional failure, and
        # an artifact signed with a different key cannot update an existing
        # installation in place either.
        gate.check("artifact.signing", not is_dev,
                   expected="a production signing identity",
                   observed="development signer" if is_dev else "non-development signer")
        gate.facts["releasable"] = not is_dev
    else:
        gate.record("artifact.signing", PASS,
                    detail="development signer accepted for a local test artifact"
                    if is_dev else "non-development signer on a test artifact")
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
    args = parser.parse_args()

    if not (args.verify_source or args.verify_artifact or args.full):
        parser.error("choose --verify-source, --verify-artifact <apk>, or --full <apk>")

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
        artifact_gates(gate, Path(apk).resolve(), args.release_class)

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
