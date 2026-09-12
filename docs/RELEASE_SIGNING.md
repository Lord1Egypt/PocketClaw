# Release signing

Operational guide for creating, storing and using PocketClaw's production
signing key. Read it before the key ceremony, not during.

> **Status: H2 closed on 2026-09-11.** The developer production key exists, the
> owner confirmed a separate backup, its public certificate is enrolled, and a
> private production-signed validation APK passed artifact verification. It was
> not installed, published or accepted as a release.

---

## 1. What a signing key actually decides

Android identifies an app by its package name **and** its signing certificate
together. The certificate is not branding — it is the thing that decides whether
a new APK is allowed to replace an installed one.

Two consequences follow, and both are permanent:

- **An app signed with a different key cannot update an existing install.**
  Android refuses. The user must uninstall first, losing all app data.
- **A lost key cannot be recreated.** There is no reissue, no authority to
  appeal to. Without Play App Signing (§8), losing the key ends your ability to
  ship updates to everyone who already installed the app.

This is why the ceremony is worth doing carefully once.

## 2. Secret versus public

| Item | Secret? | Where it lives |
| --- | --- | --- |
| Keystore file (`.jks` / `.p12`) | **Secret** | Outside this repository. Backed up. |
| Keystore password | **Secret** | Password manager or CI secret store. |
| Key password | **Secret** | Password manager or CI secret store. |
| Key alias | Not secret, but not useful alone | Environment / CI variable. |
| Certificate SHA-256 fingerprint | **Public** | `android/release-signing-cert.sha256`, committed. |

The fingerprint is the public half. Google Play displays it, and anyone can read
it out of any signed APK. Committing it is deliberate: it is what lets the
release gate prove an artifact was signed by *this* key rather than by some
other key that also verifies.

**Never commit the keystore or either password.** Not in `local.properties`, not
in `gradle.properties`, not in a `.env`, not in a comment. A key committed once
is in the history forever, and the only remedy is rotation — which breaks the
update path for every existing install.

## 3. Where the keystore goes

Outside the repository. The build enforces this: `KEYSTORE_PATH` pointing
anywhere inside the working tree fails the build with an explanation, because
`.gitignore` stops an accidental `git add` and does nothing about `git add -f`
or a future pattern change.

A reasonable location:

```
~/.android-keystores/pocketclaw-release.jks      # chmod 600
```

Back it up somewhere that survives losing this machine — an encrypted archive in
offline storage, or a password manager that accepts file attachments. Store the
two passwords separately from the file, so one compromised location is not
enough.

## 4. Proposed key parameters

Verified against the JDK 17 `keytool` and Build Tools `apksigner 0.9` in this
project's toolchain.

| Parameter | Proposed | Why |
| --- | --- | --- |
| Algorithm | `RSA` | Universally accepted by Play and every Android version this app supports. EC is valid but has more edge cases across older verifiers. |
| Key size | `4096` | 2048 is still acceptable; 4096 costs nothing at signing time and removes the question. |
| Signature algorithm | `SHA256withRSA` | Never `SHA1` — deprecated and rejected by modern tooling. |
| Validity | `10950` days (~30 years) | Must outlive the app. Play requires validity through at least 2033; a certificate that expires strands updates. |
| Keystore type | `PKCS12` | The modern default. `JKS` is the legacy proprietary format. |
| Alias | `pocketclaw-release` | Descriptive and stable. It is recorded in the keystore forever. |

**Certificate subject (`-dname`) is deliberately not decided here.** It is
baked in permanently and is a choice about identity — a personal name, a project
name or an organisation — that belongs to the owner, at ceremony time.

## 5. The ceremony (H2 — performed 2026-09-10)

The owner ran [`tool/create_release_keystore.sh`](../tool/create_release_keystore.sh)
and created the permanent PocketClaw developer app-signing key. The helper is
interactive, refuses a destination inside the repository, and never takes a
password as a command-line argument.

The keystore and both passwords are **owner-held and outside this repository**.
Nobody else has them, which is the point, and it is also why the production
signing path cannot be exercised by anyone but the owner. This repository has no
record of where the keystore lives and must not acquire one.

The *public* certificate SHA-256 is enrolled as the single bare line in
`android/release-signing-cert.sha256`:

    176dca6b198b9552fb4d9ad3ca18da8d6f23c0a3f5ed4bd6b75a0700f9f0efcf

That is the only thing from the ceremony that is committed, ever.

### The defect the real ceremony found

The first run created the key correctly and then printed its "Certificate
SHA-256 fingerprint" heading with **nothing underneath**, leaving the owner
holding a key they could not enroll.

`keytool` sends its password prompt and its warnings to stderr and only the
listing to stdout. The helper discarded stderr and piped stdout into `awk`, so
the prompt never reached the terminal. keytool does not treat a missing password
as an error: it prints an integrity warning nobody could see, lists the entry
with `Certificate chain length: 0` — no certificate, therefore no `SHA256:` line
— and exits 0. `awk` matched nothing, the pipeline ended in `tr` and so still
succeeded, which meant the `|| ...` fallback could not fire either.

Fixed by leaving stderr alone and checking the result's shape instead of
trusting an exit status. There is now one implementation, reachable
non-interactively so the value can be re-read without another ceremony:

```bash
tool/create_release_keystore.sh --print-fingerprint <keystore> <alias>
```

It prints 64 lowercase hex characters, or fails and says why. A blank success is
no longer reachable, and `tool/test_create_release_keystore.py` builds a
disposable keystore and proves both halves.

### Private signing validation (H2 — performed 2026-09-11)

The owner confirmed that a separate backup of the production keystore exists,
then entered both passwords through hidden local terminal input. A temporary
helper outside the repository exported them only inside its own process and
unset all four signing variables on exit. No secret value entered a command
line, repository file, Gradle property, log or report.

Gradle's `validateReleaseSigning` task selected the production keystore from the
environment. `:app:assembleRelease -Ptarget-platform=android-arm64` then built
one private validation APK:

    path       build/app/outputs/apk/release/app-release.apk
    size       64359287 bytes
    sha256     f0d83298c2ce061c01a9fc931ad29676e4d4b646bb5b204a9bf0002b11a7f46f
    package    com.lord1egypt.pocketclaw
    version    0.2.0 (62)
    product ABI arm64-v8a; plugin stubs also present for armeabi-v7a and x86_64
    signer     176dca6b198b9552fb4d9ad3ca18da8d6f23c0a3f5ed4bd6b75a0700f9f0efcf

Independent `apksigner` inspection reported one signer, matching the enrolled
production certificate and differing from the development certificate. The
production artifact gate passed with 20 PASS, 0 FAIL and one SKIP:
`artifact.dart_snapshot_paths`, still pending final binary hardening.

This artifact is private validation evidence. It was not installed or
published, is not an accepted release, and does not advance the physical
baseline. `vc62` / `lastAcceptedVersionCode=62` remains accepted. H2 is
complete. H3A later established the Dart-hardening build path with a separate
development-signed artifact; H3B later validated that hardened path with the
same enrolled production identity.

### Production-signed Dart-hardening validation (H3B — performed 2026-09-11)

The owner entered both passwords only through hidden local input in a temporary
helper outside the repository. The corrected helper completed Java 17 and all
other non-secret prerequisites first, validated `:app:validateReleaseSigning`,
and invoked the canonical command:

```text
python3 tool/build_hardened_android.py --signing production --clean
```

The resulting private APK is 63,787,907 bytes with SHA-256
`ceef6640d8abd9d084c3ff37d8e903aaf3c82b287de65ec15a37d91124bdebe6`.
Independent inspection found one v2 signer matching
`176dca6b198b9552fb4d9ad3ca18da8d6f23c0a3f5ed4bd6b75a0700f9f0efcf`.
Dart AOT SHA-256 is
`c7b2a885ff843a20c57097a0d16ba07c728bd64cf17455a1ce61e6f463a5ae77`
and private split-DWARF SHA-256 is
`0f52873bc712fe0c17d636f5bdb7d08ee80cdacfe633cf6a786dd2c6d93b8acc`,
both byte-identical to H3A. The production artifact gate passed 25 / 25.

The APK and private symbols were not committed, installed, or published. The
accepted physical baseline remains vc62 / 62. H3B does not authorize later
hardening or release work.

### Production-signed native/ELF validation (H5C — performed 2026-09-12)

The same hidden-input model, from a temporary helper outside the repository
that completed every non-secret prerequisite before its first prompt and
unset all four variables on exit. The keystore and alias were additionally
confirmed to open by piping the password to `keytool` on stdin — never as an
argument — before committing to a long build.

    path       build/app/outputs/apk/release/app-release.apk
    size       63564563 bytes
    sha256     3774202ef9832c70ffa376e663db1da69e17ae9318df4cc8cb31156fc0c7eae7
    package    com.lord1egypt.pocketclaw
    version    0.2.0 (62)
    signers    exactly one, v2 scheme only
    signer     176dca6b198b9552fb4d9ad3ca18da8d6f23c0a3f5ed4bd6b75a0700f9f0efcf
    subject    CN=PocketClaw, OU=PocketClaw Release, O=PocketClaw
    key        RSA 4096

The development certificate `15cf75f9945d5354e75707e0326b7cffc60ac51a68df38156db318ef4578a27c`
is absent. The artifact is exactly 4,096 bytes larger than the H5B
development-signed candidate, which is the signing block: a 4096-bit production
key against the debug key's 2048. That difference is the *only* one — all 18
packaged ELF entries, the Dart AOT, the private split DWARF and the private R8
mapping are byte-identical between the two builds.

The production release gate passed **55 PASS / 0 FAIL / 0 SKIPPED** with
`releasable: true` — the first PocketClaw artifact to reach
`PASS — production release candidate` with no skips.

**This artifact must not be installed over the current device state.** The
Samsung carries the development-signed H5B build; the production certificate is
a different identity, so a cross-signer `adb install -r` is forbidden and would
fail. The transition needs its own migration, clean-install and data-safeguard
milestone. See [`RELEASE_PROCESS.md`](RELEASE_PROCESS.md).

The APK and private symbols were not committed, installed, or published. The
accepted physical baseline remains vc62 / 62.

### Owner-local prerequisite ordering

Any helper that collects owner signing passwords must finish all non-secret
prerequisites before its first hidden prompt. At minimum it verifies the
required JDK version and `JAVA_HOME`, Python, the Gradle wrapper, repository and
canonical build-helper paths, and external keystore presence. A prerequisite
failure must exit without requesting either password. Only then may the helper
use hidden terminal input and export credentials inside the trapped process.

## 6. Building with the production key

All four come from the environment. Nothing is read from a tracked file:

```bash
export KEYSTORE_PATH="$HOME/.android-keystores/pocketclaw-release.jks"
export KEY_ALIAS='pocketclaw-release'
# The two passwords: read them from your password manager rather than typing
# them, so they stay out of shell history. Placeholders shown, never values.
read -rs KEYSTORE_PASSWORD   && export KEYSTORE_PASSWORD
read -rs KEY_PASSWORD        && export KEY_PASSWORD

python3 tool/build_hardened_android.py --signing production --clean
```

Prefer a method that keeps the values out of shell history — a password
manager's CLI, or reading them into the environment from a file outside the
repository.

If any of the four is missing the build stops and names **which** are missing,
never their values. If only some are set, the build stops even with
`-PallowDebugSigning=true`: partial configuration means production signing was
intended, and answering that with a debug-signed artifact is exactly the silent
downgrade this design exists to prevent.

### CI

Store the same four as encrypted CI secrets, materialise the keystore into a
path outside the checkout for the duration of the job, and delete it afterwards.
Never echo them; never write them into a workspace file that a later step could
archive.

## 7. Local test builds

Development and device validation still need a release-shaped APK:

```bash
cd android && ./gradlew :app:assembleRelease \
  -Ptarget-platform=android-arm64 -PallowDebugSigning=true
```

This is **LOCAL TEST / NON-RELEASABLE**. The release gate labels it as such and
refuses it under `--release-class production`, keyed on the certificate digest
rather than on the certificate's subject line — a subject is attacker-chosen
text, a digest is not.

## 8. Three distribution channels, three distinct keys

PocketClaw ships through **direct APK**, **Google Play** and **official
F-Droid**. That makes key identity a design decision rather than a formality,
because these three names are routinely confused and conflating them is the
usual source of an unrecoverable mistake:

| Identity | What it is | Who holds it | Used for |
| --- | --- | --- | --- |
| **PocketClaw developer app-signing key** | The long-lived identity Android checks when installing an update. | You. | Direct APK **and** developer-signed F-Droid builds. |
| **Google Play upload key** | Proves to Google that an upload came from you. Nothing else. | You. | Authenticating uploads only. |
| **Google Play app-signing key** | What Play re-signs distributed artifacts with. | Google. | Play installs only. |

### A — the PocketClaw developer app-signing key

This is the key created at the H2 ceremony on 2026-09-10, certificate
`176dca6b198b9552fb4d9ad3ca18da8d6f23c0a3f5ed4bd6b75a0700f9f0efcf`. It exists,
it is enrolled, and it is the app identity for every channel PocketClaw controls
directly. The Play upload key (B below) does **not** exist and is not part of
this milestone.

**Signing continuity is the goal:**

```
   Direct APK  ←──── same developer certificate ────→  F-Droid APK
                                                       (reproducible,
                                                        developer-signed)
```

A user who installed the direct APK can update from F-Droid, and the reverse,
because both carry the same certificate. This is worth designing for now: it
cannot be retrofitted after the first public release without forcing an
uninstall.

F-Droid supports this. It does **not** have to generate the application
identity: where a build reproduces bit-for-bit from source, F-Droid can publish
the developer-signed APK and pin the developer certificate via
`AllowedAPKSigningKeys`. That path depends on reproducibility, which is why
[`FDROID_RELEASE.md`](FDROID_RELEASE.md) treats it as a build requirement rather
than a nice-to-have.

If reproducibility cannot be achieved, the fallback is an F-Droid-signed build —
a **different** certificate, and therefore a separate installation that cannot
update a directly-installed PocketClaw. That is a real cost, not a detail.

### B — the Google Play upload key

Separate, deliberately. Do not reuse the developer app-signing key as the upload
key merely because both exist: an upload key is replaceable, and keeping it
distinct means a compromised CI credential does not put the identity that
governs direct and F-Droid updates at risk.

### C — Google Play App Signing

Under Play App Signing, Google holds the key that signs what Play distributes.
The certificate a Play-installed user verifies is Google's, not yours.

**The consequence for a three-channel product:** a Play install and a
direct/F-Droid install will not share a signing identity, so they are not
mutually updatable. This is inherent to Play App Signing and is accepted, not
worked around. The direct and F-Droid channels remain continuous with each
other; Play is its own lineage.

There is one alternative — uploading your own app-signing key to Play so all
three match. It requires handing Google the key that governs every channel, and
it is not recommended here. **Never weaken key security to force cross-store
signer equality.**

Nothing has been enrolled with Google Play, and no PEPK material has been
generated.

## 9. Verifying a signer

```bash
# From a built artifact
apksigner verify --print-certs --verbose <apk>

# From the keystore, already in the form the enrollment file wants
# (prompts for the password; never pass it as an argument)
tool/create_release_keystore.sh --print-fingerprint <keystore> <alias>

# Or raw, if you want to see the whole certificate
keytool -list -v -keystore <keystore> -alias <alias>
```

`keytool` prints the digest uppercase and colon-separated; the enrollment file
wants it lowercase with colons removed, which is what `--print-fingerprint`
emits. If it prints nothing it now fails and says so — see §5.

The gate does this for you:

```bash
python3 tool/release_gate.py --verify-artifact <apk> --release-class production
```

### Signature schemes

Observed on the current toolchain (AGP 8.11.1, apksigner 0.9): both the local
test artifact and the H2 private production validation artifact verify under
**v2 only**. v1 is not expected — `minSdk` is 24, and
JAR signing is only needed below API 24. v3 enables key *rotation* and v4
supports incremental install; both are worth revisiting when the production key
exists, since the choice interacts with the key itself. The gate records which
schemes verified as a reported fact rather than enforcing a list, so it cannot
invent a requirement nobody agreed to.

## 10. Rotation and loss

- **Key compromised:** without Play App Signing, rotation via v3
  `SigningCertificateLineage` lets *new* installs accept the new key, but
  existing installs still require the old one. Plan for a compatibility window,
  and treat compromise as a serious incident.
- **Key lost, no backup:** you cannot update any existing installation. The
  practical path is a new package name, which is a new app to every user.
- **Password lost, keystore intact:** the same as losing the key. There is no
  recovery mechanism.

Back up the keystore. Store the passwords separately. Test the restore path once
while it still does not matter.

## 11. Note on the accepted vc62 install

The accepted `0.2.0+62` build on the test device is signed with the **local test
certificate**. The production key now exists, so a production-signed build
**cannot** `adb install -r` over it — Android requires signing continuity, and this is the
protection working, not a bug.

Do not weaken production signing to preserve that install, and do not reuse the
debug key as the production key. Production physical validation needs a clean
install, after deliberately preserving anything on the device worth keeping.
