#!/usr/bin/env python3
"""Fail if the resolved Android build graph contains a proprietary SDK.

F-Droid's inclusion policy forbids proprietary tracking, advertising and
analytics libraries in the build, not merely in the APK: a compileOnly
dependency still has to be downloaded and compiled against. Umeng sat in
releaseCompileClasspath for exactly that reason while every DEX scan of the APK
came back clean. So this asks Gradle what it actually resolves, for both the
compile and the runtime classpath of the release variant, rather than
searching build files for names.

Exit status is 0 when both graphs resolve and neither contains a forbidden
group, and 1 otherwise -- including when Gradle cannot resolve the graph,
because an unreadable graph proves nothing.
"""

from __future__ import annotations

import os
import re
import subprocess
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
ANDROID = REPO / "android"
CONFIGURATIONS = ("releaseCompileClasspath", "releaseRuntimeClasspath")

# Maven groups of proprietary analytics, attribution, advertising, crash
# reporting and vendor-push SDKs. A group matches itself and its subgroups.
FORBIDDEN_GROUPS = (
    "com.umeng",
    "com.google.firebase",
    "com.google.android.gms",
    "com.google.android.play",
    "com.android.installreferrer",
    "com.google.android.ads",
    "com.crashlytics",
    "io.fabric",
    "com.facebook.android",
    "com.appsflyer",
    "com.adjust.sdk",
    "com.amplitude",
    "com.mixpanel",
    "io.sentry",
    "com.bugsnag",
    "com.flurry",
    "com.segment",
    "com.newrelic",
    "com.instabug",
    "com.yandex.android",
    "io.branch",
    "com.huawei.hms",
    "com.xiaomi",
    "com.tencent.bugly",
    "com.applovin",
    "com.unity3d.ads",
)

# Libraries that schedule work Android runs later on its own; WorkManager
# merges a BOOT_COMPLETED receiver, which would start PocketClaw after a
# reboot without the owner (PC-DEF-085). PocketClaw schedules nothing.
BACKGROUND_SCHEDULERS = (
    "androidx.work",
    "com.firebase:firebase-jobdispatcher",
    "com.evernote:android-job",
)

# "+--- group:artifact:version" and "\--- group:artifact -> version" lines.
COORDINATE = re.compile(r"^[| +\\-]*--- ([A-Za-z0-9_.\-]+):([A-Za-z0-9_.\-]+)", re.MULTILINE)


def coordinates(tree: str) -> set[tuple[str, str]]:
    return {(match.group(1), match.group(2)) for match in COORDINATE.finditer(tree)}


def forbidden(coords: set[tuple[str, str]]) -> list[str]:
    hits = []
    for group, artifact in sorted(coords):
        coordinate = f"{group}:{artifact}"
        if any(group == banned or group.startswith(banned + ".") for banned in FORBIDDEN_GROUPS):
            hits.append(coordinate)
        elif any(coordinate.startswith(s) if ":" in s else (group == s or group.startswith(s + "."))
                 for s in BACKGROUND_SCHEDULERS):
            hits.append(coordinate + " (background scheduler)")
    return hits


def resolve(configuration: str) -> str:
    env = dict(os.environ)
    jdk = REPO.parent / "PocketCLaw/.tooling/jdk-17"
    if not env.get("JAVA_HOME") and (jdk / "bin/java").is_file():
        env["JAVA_HOME"] = str(jdk)
    proc = subprocess.run(
        [str(ANDROID / "gradlew"), "-q", ":app:dependencies", "--configuration", configuration],
        cwd=ANDROID, env=env, capture_output=True, text=True,
    )
    if proc.returncode != 0:
        raise RuntimeError(f"gradle could not resolve {configuration}: "
                           + (proc.stderr.strip().splitlines() or ["no output"])[-1])
    return proc.stdout


def main() -> int:
    total = 0
    failed = False
    for configuration in CONFIGURATIONS:
        try:
            tree = resolve(configuration)
        except RuntimeError as error:
            print(f"FAIL {error}")
            return 1
        coords = coordinates(tree)
        if not coords:
            print(f"FAIL {configuration} resolved to nothing; the output was not a dependency tree")
            return 1
        total += len(coords)
        hits = forbidden(coords)
        for hit in hits:
            print(f"{configuration}: forbidden {hit}")
        failed = failed or bool(hits)
    if failed:
        print("FAIL a proprietary SDK or background scheduler is in the resolved build graph")
        return 1
    print(f"PASS {total} resolved coordinates across {len(CONFIGURATIONS)} graphs, "
          "no proprietary SDK or background scheduler")
    return 0


if __name__ == "__main__":
    sys.exit(main())
