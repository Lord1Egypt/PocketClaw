#!/usr/bin/env python3
"""Builds PocketClaw exactly the way F-Droid's buildserver does, at its paths.

PC-DEF-092. The Dart AOT snapshot (libapp.so) depends on where the checkout,
the Flutter SDK and the pub cache live, and no supported Flutter option removes
that dependence for a pub-based Android build. So the release that PocketClaw
publishes is built in the environment F-Droid rebuilds it in: F-Droid's
pinned buildserver image, running `fdroid build --on-server` from
/home/vagrant, which puts the checkout at /home/vagrant/build/<appid>, Flutter
at /home/vagrant/build/srclib/flutter and the pub cache inside the checkout --
the layout fdroidserver's own server mode creates. The recipe is
fdroid/metadata.yml.in; the metadata submitted to fdroiddata is rendered from
the same file, so the two builds cannot drift apart.

    metadata   render the fdroiddata metadata for a commit
    build      run the canonical unsigned build and collect its outputs
    verify     check a signed APK against an unsigned one with fdroidserver's
               own verify_apks (apksigcopier), inside the same image

Needs docker, a checkout of fdroidserver at the pinned commit and an fdroiddata
checkout for the flutter and rustup srclib definitions. The container runs as
the image's vagrant user (uid 1000), so the work directory must be writable by
that uid. Nothing here signs anything.
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
import time
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
SOURCE_REPO = "https://github.com/Lord1Egypt/PocketClaw.git"
TEMPLATE = REPO / "fdroid/metadata.yml.in"
SIGNER_FILE = REPO / "android/release-signing-cert.sha256"
APP_ID = "com.lord1egypt.pocketclaw"
IMAGE = ("registry.gitlab.com/fdroid/fdroidserver@sha256:"
         "9cb68105642ca4e7b295f0ceab10f069f5b3247dc18fa7c36046e9d81aa469a8")
FDROIDSERVER_COMMIT = "a35fdfddd9c66823987a410566a6101186e39c84"
SRCLIBS = ("flutter", "rustup")
BINARIES_URL = ("https://github.com/Lord1Egypt/PocketClaw/releases/download/"
                "v%v/PocketClaw-v%v-arm64-v8a.apk")
HOME = "/home/vagrant"
LAYOUT = ("metadata", "srclibs", "build", "unsigned", "tmp", "logs", "cache")


class CanonicalBuildError(RuntimeError):
    pass


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for block in iter(lambda: handle.read(1 << 20), b""):
            digest.update(block)
    return digest.hexdigest()


def enrolled_signer() -> str:
    for line in SIGNER_FILE.read_text(encoding="utf-8").splitlines():
        if re.fullmatch(r"[0-9a-f]{64}", line.strip()):
            return line.strip()
    raise CanonicalBuildError(f"{SIGNER_FILE} has no SHA-256 certificate digest")


def version_at(commit: str) -> tuple[str, int]:
    """versionName and versionCode as pubspec.yaml declares them at commit."""
    try:
        pubspec = subprocess.run(
            ["git", "-C", str(REPO), "show", f"{commit}:pubspec.yaml"],
            check=True, capture_output=True, text=True).stdout
    except subprocess.CalledProcessError as error:
        raise CanonicalBuildError(f"cannot read pubspec.yaml at {commit}: {error.stderr}") from error
    match = re.search(r"^version:\s*(\d+\.\d+\.\d+)\+(\d+)\s*$", pubspec, re.MULTILINE)
    if not match:
        raise CanonicalBuildError(f"pubspec.yaml at {commit} has no <name>+<code> version")
    return match.group(1), int(match.group(2))


def full_commit(commit: str) -> str:
    try:
        resolved = subprocess.run(
            ["git", "-C", str(REPO), "rev-parse", "--verify", f"{commit}^{{commit}}"],
            check=True, capture_output=True, text=True).stdout.strip()
    except subprocess.CalledProcessError as error:
        raise CanonicalBuildError(f"unknown commit {commit}") from error
    return resolved


def render_metadata(commit: str, version_name: str, version_code: int, track_b: bool,
                    template: str | None = None) -> str:
    """The fdroiddata metadata for one release.

    Track B adds the upstream APK as the reference binary and pins the enrolled
    production certificate, so F-Droid publishes the upstream signature only
    when its rebuild matches byte for byte.
    """
    if not re.fullmatch(r"[0-9a-f]{40}", commit):
        raise CanonicalBuildError(f"metadata needs a full commit hash, got {commit!r}")
    text = TEMPLATE.read_text(encoding="utf-8") if template is None else template
    replacements = {
        "@VERSION_NAME@": version_name,
        "@VERSION_CODE@": str(version_code),
        "@COMMIT@": commit,
        # Laid out exactly as fdroiddata CI's `fdroid rewritemeta` writes them
        # (Debian's fdroidserver with ruamel.yaml 0.18; 0.19 folds lines
        # differently), so the rendered file passes that job unchanged.
        "@BINARIES@": f"Binaries: \n  {BINARIES_URL}\n" if track_b else "",
        "@SIGNING_KEYS@": f"\nAllowedAPKSigningKeys: {enrolled_signer()}\n" if track_b else "",
    }
    for placeholder, value in replacements.items():
        if placeholder not in text:
            raise CanonicalBuildError(f"{TEMPLATE} lacks {placeholder}")
        text = text.replace(placeholder, value)
    return text


def fdroidserver_commit(path: Path) -> str:
    try:
        return subprocess.run(["git", "-C", str(path), "rev-parse", "HEAD"],
                              check=True, capture_output=True, text=True).stdout.strip()
    except (OSError, subprocess.CalledProcessError) as error:
        raise CanonicalBuildError(f"{path} is not a git checkout of fdroidserver") from error


# The host half of `fdroid build --server`: fdroidserver's own VCS layer clones
# the app into build/<appid> and checks out the commit, and that directory is
# what the server half builds from.
HOST_CHECKOUT = (
    "import sys; from fdroidserver import common; "
    "common.config = common.read_config(); "
    "common.getvcs('git', sys.argv[1], sys.argv[2]).gotorevision(sys.argv[3])"
)


def container_command(app_ref: str, commit: str) -> str:
    return (
        ". /etc/profile; "
        f"export PATH=\"{HOME}/fdroidserver:$PATH\" PYTHONPATH=\"{HOME}/fdroidserver\"; "
        "export JAVA_HOME=$(java -XshowSettings:properties -version 2>&1 >/dev/null "
        "| sed -n 's/^ *java.home = //p'); "
        f"cd {HOME} && fdroid fetch_srclibs -v {app_ref} "
        f"&& python3 -c \"{HOST_CHECKOUT}\" {SOURCE_REPO} build/{APP_ID} {commit} "
        f"&& fdroid build --on-server --no-tarball -v {app_ref}"
    )


def docker_run_command(work: Path, fdroidserver: Path, name: str, app_ref: str,
                       commit: str) -> list[str]:
    mounts = []
    for directory in LAYOUT:
        target = f"{HOME}/.cache" if directory == "cache" else f"{HOME}/{directory}"
        mounts += ["-v", f"{work / directory}:{target}"]
    return [
        "docker", "run", "--rm", "--name", name, "-u", "vagrant", "-w", HOME,
        *mounts, "-v", f"{fdroidserver}:{HOME}/fdroidserver:ro",
        "--entrypoint", "/bin/bash", IMAGE, "-c", container_command(app_ref, commit),
    ]


def build(args: argparse.Namespace) -> int:
    commit = full_commit(args.commit)
    version_name, version_code = version_at(commit)
    work, out = Path(args.work).resolve(), Path(args.out).resolve()
    fdroidserver, fdroiddata = Path(args.fdroidserver).resolve(), Path(args.fdroiddata).resolve()
    if work.exists() and any(work.iterdir()):
        raise CanonicalBuildError(f"{work} is not empty; every canonical build starts clean")
    if out.exists() and any(out.iterdir()):
        raise CanonicalBuildError(f"{out} is not empty; outputs are never overwritten")
    if fdroidserver_commit(fdroidserver) != FDROIDSERVER_COMMIT:
        raise CanonicalBuildError(f"{fdroidserver} is not at the pinned {FDROIDSERVER_COMMIT}")
    for directory in LAYOUT:
        (work / directory).mkdir(parents=True, exist_ok=True)
    for lib in SRCLIBS:
        shutil.copy2(fdroiddata / "srclibs" / f"{lib}.yml", work / "srclibs" / f"{lib}.yml")
    (work / "metadata" / f"{APP_ID}.yml").write_text(
        render_metadata(commit, version_name, version_code, track_b=args.track_b), encoding="utf-8")

    app_ref = f"{APP_ID}:{version_code}"
    log = work / "logs" / "canonical-build.log"
    command = docker_run_command(
        work, fdroidserver, f"pocketclaw-canonical-{os.getpid()}", app_ref, commit)
    started = time.monotonic()
    with log.open("w", encoding="utf-8") as handle:
        result = subprocess.run(command, stdout=handle, stderr=subprocess.STDOUT)
    seconds = round(time.monotonic() - started)
    if result.returncode != 0:
        raise CanonicalBuildError(f"fdroid build failed after {seconds}s; see {log}")

    track_b_verified = None
    if args.track_b:
        # fdroidserver downloaded the published APK, compared it with this
        # rebuild (verify_apks) and checked the signer; it deletes the build
        # and fails if either check fails, so these lines are its verdict.
        text = log.read_text(encoding="utf-8", errors="replace")
        track_b_verified = (
            "compared built binary to supplied reference binary successfully" in text
            and "supplied reference binary has allowed signer" in text)
        if not track_b_verified:
            raise CanonicalBuildError(f"fdroid build did not verify the reference binary; see {log}")
    unsigned = work / "unsigned" / f"{APP_ID}_{version_code}.apk"
    if not unsigned.is_file():
        raise CanonicalBuildError(f"fdroid build reported success but {unsigned} is missing")
    out.mkdir(parents=True, exist_ok=True)
    apk = out / f"PocketClaw-v{version_name}+{version_code}-canonical-unsigned.apk"
    shutil.copy2(unsigned, apk)
    checkout = work / "build" / APP_ID
    private = out / "private"
    for source, name in ((checkout / "build/private-symbols", "private-symbols"),
                         (checkout / "build/app/outputs/mapping/release", "r8-mapping")):
        if not source.is_dir():
            raise CanonicalBuildError(f"expected private build support at {source}")
        shutil.copytree(source, private / name)
    for path in [private, *private.rglob("*")]:
        path.chmod(0o700 if path.is_dir() else 0o600)
    shutil.copy2(work / "metadata" / f"{APP_ID}.yml", out / f"{APP_ID}.yml")
    shutil.copy2(log, out / "canonical-build.log")
    record = {
        "commit": commit,
        "versionName": version_name,
        "versionCode": version_code,
        "image": IMAGE,
        "fdroidserverCommit": FDROIDSERVER_COMMIT,
        "fdroiddataCommit": fdroidserver_commit(fdroiddata),
        "checkout": f"{HOME}/build/{APP_ID}",
        "apk": apk.name,
        "apkBytes": apk.stat().st_size,
        "apkSha256": sha256_file(apk),
        "seconds": seconds,
        "trackBVerified": track_b_verified,
    }
    (out / "canonical-build.json").write_text(json.dumps(record, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(record, indent=2))
    return 0


VERIFY_SCRIPT = """
import sys, tempfile
from fdroidserver import common
common.config = common.read_config()
with tempfile.TemporaryDirectory() as tmp:
    result = common.verify_apks(sys.argv[1], sys.argv[2], tmp)
print('verify_apks: ' + ('MATCH' if result is None else 'MISMATCH\\n' + result))
sys.exit(0 if result is None else 1)
"""


def verify(args: argparse.Namespace) -> int:
    signed, unsigned = Path(args.signed).resolve(), Path(args.unsigned).resolve()
    fdroidserver = Path(args.fdroidserver).resolve()
    if fdroidserver_commit(fdroidserver) != FDROIDSERVER_COMMIT:
        raise CanonicalBuildError(f"{fdroidserver} is not at the pinned {FDROIDSERVER_COMMIT}")
    command = [
        "docker", "run", "--rm", "-u", "vagrant", "-w", "/tmp",
        "-v", f"{signed}:/tmp/in/signed.apk:ro", "-v", f"{unsigned}:/tmp/in/unsigned.apk:ro",
        "-v", f"{fdroidserver}:{HOME}/fdroidserver:ro", "--entrypoint", "/bin/bash", IMAGE, "-c",
        f". /etc/profile; export PYTHONPATH={HOME}/fdroidserver; "
        f"python3 -c \"$0\" /tmp/in/signed.apk /tmp/in/unsigned.apk", VERIFY_SCRIPT,
    ]
    return subprocess.run(command).returncode


def metadata(args: argparse.Namespace) -> int:
    commit = full_commit(args.commit)
    version_name, version_code = version_at(commit)
    text = render_metadata(commit, version_name, version_code, track_b=args.track_b)
    if args.out:
        Path(args.out).write_text(text, encoding="utf-8")
    else:
        sys.stdout.write(text)
    return 0


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    sub = parser.add_subparsers(dest="action", required=True)
    rendered = sub.add_parser("metadata", help="render fdroiddata metadata for a commit")
    rendered.add_argument("--commit", required=True)
    rendered.add_argument("--track-b", action="store_true",
                          help="add Binaries and AllowedAPKSigningKeys (upstream signature)")
    rendered.add_argument("--out")
    built = sub.add_parser("build", help="run the canonical unsigned build")
    built.add_argument("--commit", required=True)
    built.add_argument("--work", required=True, help="empty directory for the build layout")
    built.add_argument("--out", required=True, help="empty directory for the collected outputs")
    built.add_argument("--fdroidserver", required=True)
    built.add_argument("--fdroiddata", required=True)
    built.add_argument("--track-b", action="store_true",
                       help="add Binaries/AllowedAPKSigningKeys: fdroidserver verifies the published APK")
    checked = sub.add_parser("verify", help="fdroidserver verify_apks: signed against unsigned")
    checked.add_argument("--signed", required=True)
    checked.add_argument("--unsigned", required=True)
    checked.add_argument("--fdroidserver", required=True)
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)
    try:
        return {"metadata": metadata, "build": build, "verify": verify}[args.action](args)
    except CanonicalBuildError as error:
        print(f"canonical build FAILED: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
