# RECONSTRUCTED OPERATING RECORD — H5C production-signed native/ELF validation

This is an evidence-based closeout record, not a claim to reproduce the owner
prompt verbatim.

## Status and boundary

- **Status:** CLOSED. The H5B native/ELF contract is validated under the
  enrolled production signer.
- **Working branch:** `feature/final-release-hardening`.
- **Starting commit:** `33f0db672eab86985986a76598b66f968e2f44a8` (H5B closeout).
- **Ending commit:** the H5C closeout commit containing this record.
- **Version/baseline:** `0.2.0+62`; the accepted physical baseline remains
  vc62 / 62 and is untouched.

H5C changed no native source, no build recipe and no export map. It built one
private artifact, verified it, and wrote documentation. It did not access the
device, use ADB, install anything, publish, merge, tag, or advance the baseline.

## H5B physical validation (recorded here because it preceded H5C)

The exact H5B LOCAL TEST / NON-RELEASABLE APK, SHA-256
`d4fe2c4a035051e3b6500d2a2fe9bdad639c97323c355b26a3f8ae6f215b9dd8`, was
installed in place on the Samsung SM-A165F, serial `RK8Y6016N5V`, with
`adb install -r` through the Windows platform-tools binary. The upgrade was
same-identity: the already-installed vc62 was verified as carrying the same
development certificate
`15cf75f9945d5354e75707e0326b7cffc60ac51a68df38156db318ef4578a27c` before the
install, so no signature-mismatch data wipe was possible.

Install returned `Success`. The base APK pulled back off the device hashed
`d4fe2c4a…`, byte-identical to the candidate. Package, versionCode 62,
versionName 0.2.0, `appId=10666`, `dataDir=/data/user/0/com.lord1egypt.pocketclaw`
and `firstInstallTime=2026-08-26 05:21:06` were all unchanged; the external data
tree and the ten granted permissions compared identical before and after. All
fourteen native libraries unpacked into `nativeLibraryDir` at their H5B sizes.

The owner then performed the manual native/runtime smoke and reported **PASS**
with no failing item. That is the evidence that the H5B native changes — Core
stripped by `llvm-strip` rather than Go's `-s -w`, CPython's normalized `VPATH`,
jq's replaced build configuration — are correct on real hardware.

## Production build

The owner ran a temporary hidden-input helper outside the repository. It
completed every non-secret prerequisite — branch, HEAD, clean worktree, Java 17,
Python, Flutter, the Gradle project root, the canonical build helper, the
enrolled digest, and that no signing variable was already set — before
requesting either password, confirmed the keystore opens and the alias resolves
by piping the password to `keytool` on stdin, ran
`:app:validateReleaseSigning`, and then invoked the canonical command:

```text
python3 tool/build_hardened_android.py --signing production --clean
```

No password entered a command line, Gradle property, repository file, log or
report. All four signing variables were exported only inside the helper process
and unset by its exit trap.

## The production artifact

| Field | Value |
| --- | --- |
| Path | `build/app/outputs/apk/release/app-release.apk` |
| Size | 63,564,563 bytes |
| SHA-256 | `3774202ef9832c70ffa376e663db1da69e17ae9318df4cc8cb31156fc0c7eae7` |
| Package | `com.lord1egypt.pocketclaw` |
| Version | `0.2.0` (62) |
| Signers | exactly one, v2 scheme only |
| Signer SHA-256 | `176dca6b198b9552fb4d9ad3ca18da8d6f23c0a3f5ed4bd6b75a0700f9f0efcf` |
| Signer subject | `CN=PocketClaw, OU=PocketClaw Release, O=PocketClaw` |
| Key | RSA 4096 |

The development certificate `15cf75f9…27c` is absent. The artifact is 4,096
bytes larger than the H5B candidate, which is the signing block: a 4096-bit
production key against the debug key's 2048.

This is private validation evidence. It was not installed, published, accepted,
or committed, and it does not advance vc62.

## Native byte comparison — the central result

Production signing must not change a shipped native payload, and it did not.
All **18** packaged ELF entries are byte-identical between the physically
validated H5B APK and the production-signed H5C APK, compared entry by entry on
SHA-256 and size:

| Payload | SHA-256 (H5B = H5C) |
| --- | --- |
| `libpocketclaw.so` | `62f741be6f71f7518ba0df8f4457e0f7b666dfe512cde88e79e213ca0907e901` |
| `libpocketclaw-web.so` | `42d418bd3e2d1863d2dda4d46357e541a222b831cfffb17e5f36e442e455e9f5` |
| `libpocketclaw-curl.so` | `9dd4b75a8cf4ad0e3e078e77884336ae6e84110f3cd48111ababe664bad15a96` |
| `libpocketclaw-gh.so` | `804ed93f89b0cdfc20ae282466a89be5e2b23bfe9848eb78b7e7c5ca159a44ac` |
| `libpocketclaw-git.so` | `a8a342ac2cff1c39bb9d8d3165921a4b6843b18908e8714b9be051bad68e7681` |
| `libpocketclaw-git-remote-http.so` | `baea79ac073e5b3ceff5b14ada00a56931462652604b1c38e0db8b19ec30c260` |
| `libpocketclaw-jq.so` | `843b324de37727e59f9ec88b40c4f347832b91f4bd52719eebb5a7baeb4f097d` |
| `libpocketclaw-python.so` | `8b52e36dd4975763af46c9c26ab07b40e25d407e2df4415166e9d3d7d26a5b44` |
| `libpocketclaw-rg.so` | `0a5fc43f8df2fca3dbe70365e0496670df2e904239d9a4b5358fee40c4c70e4e` |
| `libpocketclaw-sqlite3.so` | `b940785d77bb7c82d43843a721f9135c4b59e641f911ea4406eae67cc201cf43` |

The four dependency-owned payloads are unchanged as well, at their H5A values:
`libapp.so` `c7b2a885…`, `libdartjni.so` `47dae44d…`, `libflutter.so`
`d12fd96d…`, `libdatastore_shared_counter.so` `d3e48717…`, plus the three
armeabi-v7a/x86_64 plugin ABI stubs.

Because the bytes are identical, the owner's Samsung smoke result transfers to
this artifact: the production APK contains exactly the native payloads that were
physically exercised.

The committed catalog `core/src/pkg/pcruntime/manifest.json` was independently
cross-checked against the staged payloads before the build: all eight bundled
entries, including `libpocketclaw-git-remote-http.so` as a pinned helper of both
`git` and `gh`, agree.

## Private native support archive

The archive survived the clean production build untouched. Its ten artifact
entries are identical to H5B; only the APK binding moved, which is expected
because the whole APK signature changed.

- ten expected entries, none missing, none unexpected
- every support hash, size and relative path valid and inside the private root
- every entry bound to its shipped ELF hash, size and build ID
- debug data plus a resolved representative function present for each
- manifest rebound to `3774202ef9832c70ffa376e663db1da69e17ae9318df4cc8cb31156fc0c7eae7`
- nothing tracked by Git, nothing packaged in the APK, manifest mode `0600`

## Verification totals

| Check | Result |
| --- | --- |
| Enforced native audit (`--enforce-target --native-support-manifest`) | **194 PASS / 0 FAIL / 0 SKIP** |
| Native audit, check-by-check against H5B | **no status differences** |
| Production release gate (`--full … --release-class production`) | **55 PASS / 0 FAIL / 0 SKIPPED**, `releasable: true` |
| Production source gate | 26 PASS / 0 FAIL / 0 SKIPPED |
| Python tool suites (7 files) | 95 tests, all pass |
| Flutter Android/signing/placement contracts | 76 tests, all pass |
| Go `pkg/coresource` + `pkg/pcruntime` | pass |

The gate reports `PASS — production release candidate`, the first time a
PocketClaw artifact has reached that state with zero skips.

## Dart and R8 regression

All three are byte-identical to H3A, H3B, H4A, H4B and H5B:

- packaged Dart AOT `c7b2a885ff843a20c57097a0d16ba07c728bd64cf17455a1ce61e6f463a5ae77`
- private split DWARF `0f52873bc712fe0c17d636f5bdb7d08ee80cdacfe633cf6a786dd2c6d93b8acc`
- private R8 mapping `14d49fad46e773e32da69b7b2336b7a968808cd1130f0319f7806ca4d09c1beb`

## Defect disposition

- `PC-DEF-009` RESOLVED, now confirmed under the production signer.
- `PC-DEF-010` RESOLVED, now confirmed under the production signer.
- `PC-DEF-011` RESOLVED, now confirmed under the production signer.
- `PC-DEF-012` OPEN by evidence-based decision. H5C narrowed no export and
  produced no new reachability evidence; the packaged export counts are
  unchanged.

One environment-only anomaly was seen and explained rather than filed:
`tool/test_create_release_keystore.py` reports 1 failure and 4 skips in a shell
without `keytool` on `PATH`, because the script's JDK precondition fires before
the usage check under test. With the toolchain JDK on `PATH` — the environment
the gate itself uses — all six pass.

## Physical-device boundary

The production APK was **not** installed. The installed vc62/H5B path carries
the development certificate and the production certificate is a different
identity, so a cross-signer `adb install -r` is forbidden and would fail. The
production-signer physical transition remains a separately authorized
migration / clean-install / data-safeguard milestone.

## Secret handling

No password value entered chat, a command line, the repository, a log, a
document or an artifact. After signing, all four signing variables are unset,
the temporary helper is deleted, and the temporary comparison files are removed.
The repository contains only the enrolled public certificate digest; no keystore
or key material exists anywhere in the working tree.

## Next milestone

Final release exposure audit: secrets/configuration and APK + AAB inspection.
It must be separately authorized and must not be started from this closeout.
