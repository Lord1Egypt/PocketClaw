# Release signing

Operational guide for creating, storing and using PocketClaw's production
signing key. Read it before the key ceremony, not during.

> **Status: no production key exists yet.** Nothing here has been executed.
> `android/release-signing-cert.sha256` carries no fingerprint, and production
> artifact verification fails by design until it does.

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

## 5. The ceremony (H2 — not yet performed)

A helper exists: [`tool/create_release_keystore.sh`](../tool/create_release_keystore.sh).
It is interactive, refuses a destination inside the repository, never takes a
password as a command-line argument, and prints the certificate fingerprint at
the end. **H1 does not run it.**

Afterwards, enroll the *public* fingerprint by adding one bare 64-character
lowercase hex line to `android/release-signing-cert.sha256`, and commit that.
Nothing else from the ceremony gets committed, ever.

## 6. Building with the production key

All four come from the environment. Nothing is read from a tracked file:

```bash
export KEYSTORE_PATH="$HOME/.android-keystores/pocketclaw-release.jks"
export KEY_ALIAS='pocketclaw-release'
# The two passwords: read them from your password manager rather than typing
# them, so they stay out of shell history. Placeholders shown, never values.
read -rs KEYSTORE_PASSWORD   && export KEYSTORE_PASSWORD
read -rs KEY_PASSWORD        && export KEY_PASSWORD

cd android && ./gradlew :app:assembleRelease -Ptarget-platform=android-arm64
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

This is the key created at the H2 ceremony. It is the app identity for every
channel PocketClaw controls directly.

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

# From the keystore (prompts for the password; never pass it as an argument)
keytool -list -v -keystore <keystore> -alias <alias>
```

`keytool` prints the digest uppercase and colon-separated; the enrollment file
wants it lowercase with colons removed.

The gate does this for you:

```bash
python3 tool/release_gate.py --verify-artifact <apk> --release-class production
```

### Signature schemes

Observed on the current toolchain (AGP 8.11.1, apksigner 0.9): the local test
artifact verifies under **v2 only**. v1 is not expected — `minSdk` is 24, and
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
certificate**. Once a production key exists, a production-signed build **cannot**
`adb install -r` over it — Android requires signing continuity, and this is the
protection working, not a bug.

Do not weaken production signing to preserve that install, and do not reuse the
debug key as the production key. Production physical validation needs a clean
install, after deliberately preserving anything on the device worth keeping.
