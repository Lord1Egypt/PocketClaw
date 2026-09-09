#!/usr/bin/env bash
#
# Creates PocketClaw's production signing keystore. Interactive, by design.
#
# This script is NOT run by any build, gate or CI job. It is run once, by the
# owner, at the key ceremony — and what it produces cannot be recreated if it is
# lost. Read docs/RELEASE_SIGNING.md first.
#
# Passwords are never accepted as arguments: an argument is visible in `ps`, in
# shell history and in any process listing a shared machine can see. keytool
# prompts for them instead, and nothing here echoes them.
#
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"

# Proposed parameters. See docs/RELEASE_SIGNING.md §4 for why each one.
KEY_ALG="RSA"
KEY_SIZE="4096"
SIG_ALG="SHA256withRSA"
VALIDITY_DAYS="10950"     # ~30 years; must outlive the app
STORE_TYPE="PKCS12"       # modern default; JKS is the legacy proprietary format

die() { printf '\nerror: %s\n' "$1" >&2; exit 1; }

command -v keytool >/dev/null 2>&1 || die \
"keytool is not on PATH. It ships with the JDK; set JAVA_HOME/bin on PATH.
  This project's toolchain JDK: \$REPO_ROOT/../PocketCLaw/.tooling/jdk-17"

cat <<'INTRO'
PocketClaw production signing key
=================================

This creates the key that decides, permanently, whether a future PocketClaw
build is allowed to update an existing installation.

  - It cannot be recreated. A lost key or a lost password ends your ability to
    ship updates to anyone who already installed the app.
  - The keystore and both passwords are secret and must never be committed.
  - Only the certificate fingerprint is public, and it is enrolled separately
    and deliberately, not by this script.

INTRO

read -r -p "Keystore destination (absolute path, outside this repository): " DEST
[ -n "$DEST" ] || die "no destination given."

case "$DEST" in
    /*) ;;
    *) die "give an absolute path. A relative path is resolved against wherever
  you happened to be standing, which is not a property you want a signing key
  to depend on." ;;
esac

[ -e "$DEST" ] && die "$DEST already exists.
  Refusing to touch it. Overwriting a keystore destroys the key inside it, and
  that is not recoverable. Move the existing file aside deliberately if you are
  certain it is not in use."

# Canonical containment, so a symlink or a ../ path cannot walk back into the
# tree. The parent is resolved because the file itself does not exist yet.
DEST_DIR="$(cd "$(dirname "$DEST")" 2>/dev/null && pwd -P)" \
    || die "the directory $(dirname "$DEST") does not exist. Create it first."
DEST_RESOLVED="$DEST_DIR/$(basename "$DEST")"

case "$DEST_RESOLVED/" in
    "$REPO_ROOT"/*) die "that path is inside the PocketClaw repository.

  A signing key must live outside the working tree. Ignoring it is not
  protection: 'git add -f', a changed ignore pattern, or a copy left behind
  after a build would all commit it — and a key in git history cannot be
  un-published, only rotated, which invalidates the update path for every
  existing install.

  Somewhere like ~/.android-keystores/ is fine." ;;
esac

read -r -p "Key alias [pocketclaw-release]: " ALIAS
ALIAS="${ALIAS:-pocketclaw-release}"

cat <<'SUBJECT'

Certificate subject
-------------------
This is baked into the certificate permanently and is visible to anyone who
inspects a signed APK. It is an identity choice — your name, a project name, an
organisation — and this script deliberately does not choose it for you.

keytool will now ask for it field by field. Leaving a field blank is allowed;
"Unknown" is what it records.

SUBJECT

read -r -p "Create the keystore now? [y/N]: " CONFIRM
case "$CONFIRM" in [yY]|[yY][eE][sS]) ;; *) echo "Aborted. Nothing was created."; exit 0 ;; esac

# Restrictive permissions from the moment of creation, rather than after: a
# world-readable window, however brief, is a window.
OLD_UMASK="$(umask)"
umask 077

echo
echo "keytool will prompt for the keystore and key passwords. They are not"
echo "echoed, and this script never sees or stores them."
echo

keytool -genkeypair \
    -keystore "$DEST_RESOLVED" \
    -storetype "$STORE_TYPE" \
    -alias "$ALIAS" \
    -keyalg "$KEY_ALG" \
    -keysize "$KEY_SIZE" \
    -sigalg "$SIG_ALG" \
    -validity "$VALIDITY_DAYS"

umask "$OLD_UMASK"
chmod 600 "$DEST_RESOLVED" 2>/dev/null || true

cat <<EOF

Created: $DEST_RESOLVED
  algorithm $KEY_ALG $KEY_SIZE, $SIG_ALG, valid $VALIDITY_DAYS days, $STORE_TYPE
  alias     $ALIAS
  mode      $(stat -c '%a' "$DEST_RESOLVED" 2>/dev/null || echo 'check manually')

Certificate SHA-256 fingerprint
-------------------------------
EOF

# The public half. Printing it is safe and is the point of the last step.
keytool -list -v -keystore "$DEST_RESOLVED" -alias "$ALIAS" 2>/dev/null \
    | awk '/SHA256:/ { print $2; exit }' \
    | tr -d ':' | tr 'A-F' 'a-f' \
    || echo "(re-run: keytool -list -v -keystore '$DEST_RESOLVED' -alias '$ALIAS')"

cat <<EOF

Next, deliberately and by hand:

  1. Back the keystore up somewhere that survives losing this machine, and store
     the two passwords separately from it.
  2. Enroll the fingerprint above as a single bare line in:
       android/release-signing-cert.sha256
     That file is public metadata. Commit only that.
  3. Never commit the keystore or either password.

This script does not enroll the fingerprint for you. Writing a signing identity
into the repository should be an act someone chose, not a side effect.
EOF
