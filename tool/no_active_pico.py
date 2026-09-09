#!/usr/bin/env python3
"""Zero Active Pico guard.

PocketClaw is a rename of upstream PicoClaw, so the word "pico" is everywhere
and a repository-wide ban would be wrong by architecture: the Go module is
still github.com/sipeed/picoclaw, the copyright is still PicoClaw contributors,
and several on-disk names must stay readable so an upgraded install keeps
working. What must not exist is an *active* Pico identity: something this build
currently writes, issues, defaults to, emits, registers or advertises.

So this scans a defined set of PocketClaw-owned production files and requires
every Pico-family match to be claimed by an explicit allowlist entry carrying a
category and a reason. Anything unclaimed fails. An entry that stops matching
also fails, because a stale exemption silently re-permits whatever moves back
under it.

Documentation, tests and the vendored upstream tree are out of scope by design;
SCOPE_NOTE below says why for each.

Usage:
    python3 tool/no_active_pico.py [--list] [--json]

Exit status is 0 when clean, 1 when the guard fails.
"""

from __future__ import annotations

import argparse
import fnmatch
import json
import re
import subprocess
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent

# The Pico family, case-insensitive. Deliberately just the stem: "pico" catches
# PicoClaw, picoclaw, PICOCLAW and bare "pico" in one rule, and the allowlist is
# what separates the legitimate ones.
PICO = re.compile(r"pico", re.IGNORECASE)

# Words that merely contain the stem and have nothing to do with the product.
# These are not exemptions, they are non-matches: "picomatch" is an npm package
# and "picometer" is a unit.
# "tópico" is Portuguese for topic and contains the stem after its accent.
NOT_THE_PRODUCT = re.compile(
    r"picomatch|picometer|picosecond|picocolors|t\u00f3picos?", re.IGNORECASE)

VALID_CATEGORIES = {
    "upstream",
    "legal",
    "historical",
    "legacy_migration",
    "external_compatibility",
}

SCOPE_NOTE = """Out of scope, and why:

  core/src (except the paths named below)  vendored upstream PicoClaw. Renaming
                                           it would fork the upstream baseline
                                           this project tracks.
  *.md, docs/                              the migration record. It has to name
                                           the old identities to describe them.
  tests                                    a test proving Pico is gone must
                                           write the word to do it.
  jniLibs/*.so                             build output, not source.
"""


def is_owned_production(path: str) -> bool:
    """Whether one repo-relative path is PocketClaw-owned production source.

    Owned means PocketClaw decides what it says; production means it reaches a
    user, on disk, on a wire or on a screen. Both must hold.
    """
    if path.endswith((".md", ".so", ".png", ".ico", ".svg", ".webmanifest")):
        return False
    if path.endswith(("_test.go", ".test.ts", ".test.tsx", "_test.dart")):
        return False
    if path.startswith("test/") or "/test/" in path or "/testdata/" in path:
        return False

    if path.startswith("lib/") and path.endswith(".dart"):
        return True
    if path.startswith("android/app/src/main/"):
        return True
    if path == "android/app/proguard-rules.pro":
        return True
    if path in ("tool/no_active_pico.py", "tool/test_no_active_pico.py"):
        # This guard and its tests. A rule that lists what is forbidden cannot
        # be scanned by that rule: it would either flag its own allowlist or,
        # worse, silently claim its own patterns and look clean by construction.
        return False
    if path.startswith("tool/") and path.endswith(".py"):
        return True
    if path.startswith("core/") and path.endswith(".sh"):
        return True

    # The dashboard: PocketClaw's own product surface, even though it lives
    # inside the vendored tree.
    if path.startswith("core/src/web/backend/") and path.endswith(".go"):
        return True
    if path.startswith("core/src/web/frontend/src/"):
        return True
    if path in (
        "core/src/web/frontend/package.json",
        "core/src/web/frontend/index.html",
        "core/src/web/frontend/vite.config.ts",
        "core/src/web/Makefile",
    ):
        return True

    # Go packages PocketClaw authored inside the vendored tree.
    for pkg in ("pcruntime", "coresource", "pid", "channels/pocketclaw"):
        if path.startswith(f"core/src/pkg/{pkg}/") and path.endswith(".go"):
            return True
    return False


# Each entry claims the matches it names. `paths` narrows an entry to where the
# reason actually applies, so a rule written for one file cannot quietly cover
# another. Keep this list short: every addition is a place Pico is allowed to
# survive, and it should be obvious from the reason why that is not a bug.
ALLOWLIST: list[dict] = [
    {
        "pattern": r"github\.com/sipeed/picoclaw",
        "category": "upstream",
        "reason": "the Go module path. Renaming it forks the upstream baseline.",
    },
    {
        "pattern": r"\bpicoclaw(\.exe)?\b|picoclaw-launcher|cmd/picoclaw"
                   r"|FindPicoclawBinary|GetPicoclawHome|isPicoclawProcess"
                   r"|executePicoclawVersion|parsePicoclawVersionOutput"
                   r"|PICOCLAW_BINARY|build-dev-picoclaw|picoHome|picotools",
        "category": "upstream",
        "reason": "the upstream Core binary's own name and the helpers that "
                  "locate it. core/src/Makefile still emits `picoclaw`, so "
                  "these are artifact lookups, not this product's identity.",
        "paths": ["core/src/web/**", "core/*.sh", "core/src/scripts/*.sh",
                  "core/src/docker/*.sh", "lib/src/native/*.dart",
                  "core/src/pkg/pid/*.go", "core/src/pkg/coresource/*.go"],
    },
    {
        "pattern": r"picoclaw-launcher-android-arm64|picoclaw-android-arm64"
                   r"|picoclaw-android-universal",
        "category": "upstream",
        "reason": "intermediate build artifact names the upstream Makefile "
                  "writes; the canonical build renames them to libpocketclaw*.so "
                  "when staging.",
        "paths": ["core/*.sh", "core/src/web/Makefile"],
    },
    {
        "pattern": r"\.picoclaw\.pid|picoclaw_prefs|picoclaw_service"
                   r"|picoclaw_foreground|picoclaw_launcher_auth"
                   r"|filesDir, \"picoclaw\"|path=\"picoclaw/\""
                   r"|pocketclaw-core/ and picoclaw/|picoclaw/ is a LEGACY"
                   r"|picoclaw/ line|the picoclaw/ lines",
        "category": "legacy_migration",
        "reason": "pre-Zero-Pico on-disk and OS-registered names. Read, "
                  "migrated, expired or excluded from backup only; never "
                  "written. Semantic guards in the Go and Kotlin suites prove "
                  "the direction.",
    },
    {
        "pattern": r"legacyPicoSessionPrefix|extractLegacyPicoSessionID"
                   r"|findLegacyPicoSessions?"
                   r"|agent:main:pico:direct:pico:|pre-Zero-Pico"
                   r"|legacy Pico|Legacy Pico|LEGACY_NAME|LEGACY_SERVICE_CHANNEL_ID"
                   r"|LEGACY_FOREGROUND_CHANNEL_ID",
        "category": "legacy_migration",
        "reason": "legacy session keys and preference/channel ids, named as "
                  "legacy so they cannot be mistaken for current identities.",
    },
    {
        "pattern": r"PICOCLAW_[A-Z_]+",
        "category": "legacy_migration",
        "reason": "upstream environment variable names. canonicalenv accepts "
                  "them as input for an upgraded install; the Android host "
                  "emits only POCKETCLAW_*, which its own guard proves.",
    },
    {
        "pattern": r"/\.picoclaw/|/pico/|\bpico\b|pico_client|pico-user"
                   r"|picoclaw\.io|\.picoclaw\b",
        "category": "legacy_migration",
        "reason": "legacy serialized channel names, routes and home paths, "
                  "parsed and redacted for installs and log files written "
                  "before the migration. Never minted.",
    },
    {
        "pattern": r"picoclaw:chat-show-thoughts",
        "category": "legacy_migration",
        "reason": "the pre-migration browser preference key, read once so an "
                  "existing dashboard user keeps their setting.",
        "paths": ["core/src/web/frontend/src/features/chat/detail-visibility.ts"],
    },
    {
        "pattern": r"picoclaw-web\.desktop",
        "category": "upstream",
        "reason": "desktop XDG autostart filename. runtime.GOOS is \"android\" "
                  "on the shipped product, so this branch is unreachable there; "
                  "renaming it would orphan existing desktop autostart entries.",
        "paths": ["core/src/web/backend/api/startup.go"],
    },
    {
        "pattern": r"Pico Protocol channel|Pico reasoning publish skipped",
        "category": "legacy_migration",
        "reason": "left-hand sides of the old-log sanitizer maps. These are the "
                  "exact strings a log file written before the migration "
                  "contains; the right-hand side is what the reader is shown. "
                  "Dropping them would make old logs less redacted than they "
                  "are today.",
        "paths": ["core/src/web/backend/api/user_visible_log.go",
                  "core/src/web/frontend/src/lib/plain-text-log.ts"],
    },
    {
        "pattern": r"PicoClawLauncher|picoclaw-web\.desktop",
        "category": "upstream",
        "reason": "desktop launch-at-login identifiers: a Windows Run key name "
                  "and an XDG autostart filename. runtime.GOOS is \"android\" on "
                  "the shipped product, so both branches are unreachable there, "
                  "and renaming them would orphan entries an existing desktop "
                  "install already registered.",
        "paths": ["core/src/web/backend/api/startup.go"],
    },
    {
        "pattern": r"findPicoclawBinaryForInfo|runPicoclawVersionOutput",
        "category": "upstream",
        "reason": "indirection seams for locating and running the upstream Core "
                  "binary, whose filename is still picoclaw.",
        "paths": ["core/src/web/backend/api/version.go"],
    },
    {
        "pattern": r"missing picoclaw_",
        "category": "legacy_migration",
        "reason": "matches an error string an older Core emits, so the host can "
                  "still explain that failure on a device mid-upgrade.",
        "paths": ["lib/src/core/service_manager.dart"],
    },
    {
        "pattern": r"picoclaw/|PicoClaw to PocketClaw namespace migration"
                   r"|PicoClaw lobster|no_active_pico|Pico identity",
        "category": "legacy_migration",
        "reason": "the release gate's own checks and messages about the "
                  "migration: the legacy backup-exclusion path it verifies, the "
                  "wording it reports when the namespace work is outstanding, "
                  "and the name of this guard where the gate invokes it. A "
                  "check has to be able to say what it checks.",
        "paths": ["tool/*.py"],
    },
    {
        "pattern": r"PicoClaw Launcher|To launch PicoClaw|picobot",
        "category": "upstream",
        "reason": "upstream desktop packaging and an upstream IRC smoke-test "
                  "script. Neither is part of what PocketClaw ships for Android.",
        "paths": ["core/src/scripts/*.sh"],
    },
    {
        "pattern": r"upstream PicoClaw|PicoClaw baseline|local PicoClaw checkout"
                   r"|PicoClaw Core v",
        "category": "upstream",
        "reason": "names the upstream project this Core is derived from, in the "
                  "scripts that track that relationship and in the staged "
                  "build's provenance record.",
        "paths": ["core/*.sh", "android/app/src/main/jniLibs/**"],
    },
    {
        "pattern": r"LEGACY_DIR_NAME|Pico identity|Zero-Pico|contains\(\"picoclaw\"\)",
        "category": "legacy_migration",
        "reason": "Android host migration code and the comments explaining why "
                  "the legacy directory and channel ids are still named. The "
                  "semantic Kotlin guards prove none of them is written.",
        "paths": ["android/**"],
    },
    {
        "pattern": r"wecomQRSourceID",
        "category": "external_compatibility",
        "reason": "the source id Tencent's WeCom QR endpoint identifies the "
                  "caller by. It is sent to work.weixin.qq.com as the `source` "
                  "and `sourceID` query parameters, so it is not this project's "
                  "name for itself — it is what a third-party service was "
                  "registered to recognise. Changing it is a claim to someone "
                  "else's system, not a rename, and an unregistered value would "
                  "break WeCom QR login rather than fail loudly. Preserved on "
                  "evidence. The identical constant in the upstream "
                  "cmd/picoclaw CLI is held in step by "
                  "tool/test_no_active_pico.py.",
        "paths": ["core/src/web/backend/api/wecom.go"],
    },
]


def tracked_files() -> list[str]:
    out = subprocess.run(
        ["git", "-C", str(REPO), "ls-files"],
        capture_output=True, text=True, check=True,
    ).stdout
    return out.split("\n")


def scan() -> tuple[list[dict], set[int]]:
    """Return unclassified findings and the indices of allowlist entries used."""
    findings: list[dict] = []
    used: set[int] = set()
    compiled = [(i, re.compile(e["pattern"]), e) for i, e in enumerate(ALLOWLIST)]

    for rel in tracked_files():
        if not rel or not is_owned_production(rel):
            continue
        full = REPO / rel
        try:
            body = full.read_text(encoding="utf-8")
        except (OSError, UnicodeDecodeError):
            continue
        for lineno, line in enumerate(body.splitlines(), 1):
            if not PICO.search(line):
                continue
            # Strip the words that only coincidentally contain the stem, then
            # ask again: a line that was matching only because of those is not
            # about the product at all.
            residue = NOT_THE_PRODUCT.sub("", line)
            if not PICO.search(residue):
                continue

            # Every entry that could claim this line is marked used, not just
            # the first. Stopping at the first match would report a perfectly
            # good entry as stale simply because a broader one sits above it,
            # and the fix for that false report is to delete a real exemption.
            claimed = False
            for idx, rx, entry in compiled:
                paths = entry.get("paths")
                if paths and not any(fnmatch.fnmatch(rel, p) for p in paths):
                    continue
                if rx.search(residue):
                    used.add(idx)
                    claimed = True
            if not claimed:
                findings.append({"path": rel, "line": lineno, "text": line.strip()[:160]})
    return findings, used


def main() -> int:
    parser = argparse.ArgumentParser(description="Zero Active Pico guard")
    parser.add_argument("--list", action="store_true",
                        help="print the allowlist and the scope note")
    parser.add_argument("--json", action="store_true", help="machine-readable output")
    args = parser.parse_args()

    if args.list:
        print(SCOPE_NOTE)
        for entry in ALLOWLIST:
            where = ", ".join(entry.get("paths", ["(anywhere in scope)"]))
            print(f"[{entry['category']}] {entry['pattern'][:70]}\n"
                  f"    where:  {where}\n    reason: {entry['reason']}\n")
        return 0

    problems: list[str] = []

    # A malformed entry is a guard that cannot be reviewed.
    for i, entry in enumerate(ALLOWLIST):
        if entry.get("category") not in VALID_CATEGORIES:
            problems.append(f"allowlist[{i}] has category "
                            f"{entry.get('category')!r}, want one of "
                            f"{sorted(VALID_CATEGORIES)}")
        if not entry.get("reason", "").strip():
            problems.append(f"allowlist[{i}] has no reason")
        try:
            re.compile(entry["pattern"])
        except re.error as exc:
            problems.append(f"allowlist[{i}] pattern does not compile: {exc}")

    findings, used = scan()

    stale = [i for i in range(len(ALLOWLIST)) if i not in used]
    for i in stale:
        problems.append(
            f"allowlist[{i}] ({ALLOWLIST[i]['category']}) matches nothing any "
            f"more — remove it: {ALLOWLIST[i]['pattern'][:70]}")

    if args.json:
        print(json.dumps({"findings": findings, "problems": problems,
                          "allowlist": len(ALLOWLIST)}, indent=2))
    else:
        for f in findings:
            print(f"ACTIVE PICO  {f['path']}:{f['line']}: {f['text']}")
        for p in problems:
            print(f"GUARD        {p}")
        total = len(findings) + len(problems)
        if total == 0:
            print(f"no active Pico identity in PocketClaw-owned production source "
                  f"({len(ALLOWLIST)} allowlist entries, all in use)")
        else:
            print(f"\n{len(findings)} unclassified occurrence(s), "
                  f"{len(problems)} allowlist problem(s)")
    return 1 if (findings or problems) else 0


if __name__ == "__main__":
    sys.exit(main())
