# PocketClaw Session Handoff

## Secure GitHub auth — DNS fixed, awaiting physical UI acceptance, 2026-09-01

Branch `feature/secure-github-auth`. **Not merged.** Everything automated is
green and the DNS fix is verified on the device; the twelve-step UI flow is the
remaining gate.

The gh payload carries the resolver from `core/src/pkg/androiddns`, copied in by
`runtime/build-gh-android-arm64.sh` so there is one implementation, and the
runtime hands it `PICOCLAW_DNS_SERVER` on the **gh profile only** — gh is the
only bundled tool that resolves names in Go.

Proved over adb with one binary and one variable: without the variable,
`lookup api.github.com on [::1]:53: connection refused`; with it, HTTP 401 "Bad
credentials" from GitHub in 450 ms. DNS failure became an authentication answer.

`install_payload`'s alignment guard required exactly `0x4000` and rejected Go's
`0x10000`. It now requires a multiple of 16 KB, which is what Android needs;
Core, the launcher and the shipped gh have all been `0x10000` since Phase 1.

| Artifact | Value |
|---|---|
| gh payload | `3f56431f1fdd1497e9529f1c844090881abbfd5dd47a5e6a40abb7611bf8b9d8` |
| Catalog | `2.3.0` |
| APK | `build/app/outputs/flutter-apk/app-release.apk`, 64,282,270 bytes, versionCode 13 |
| APK SHA-256 | `4a7d6eb8eeef3873d1fca7168c631aeaf9f7698069bb2f0f647e1391747706f1` |

## GitHub auth blocked by Go DNS on Android — cause proven, 2026-09-01

Branch `feature/secure-github-auth`. **Not merged.** The credential storage,
injection and UI are done and green; gh cannot resolve DNS on Android, so
Connect cannot validate.

**Proven on device over adb, outside the app:** `gh api user` with `GH_DEBUG=1`
reports `dial tcp: lookup api.github.com on [::1]:53: connection refused`.
Android has no `/etc/resolv.conf`, so Go's resolver falls back to localhost.
`GODEBUG=netdns=2` shows `using the Go DNS resolver`; `netdns=cgo` cannot help
because the payload is `CGO_ENABLED=0`. The bundled curl gets HTTP 200 in the
same environment. Both CA stores are populated and the failure is unchanged with
`SSL_CERT_DIR` set either way, so it is not a certificate problem.

`pkg/androiddns` already solves exactly this for Core and the launcher via
`PICOCLAW_DNS_SERVER`. gh never got it, and that variable is not in the
runtime's inherited environment keys.

**The fix has two parts, neither applied yet:** give the gh payload the same
resolver shim at build time, and let `PICOCLAW_DNS_SERVER` reach managed tools.
The second changes what every managed tool sees, and the first repins a
checksum-pinned payload, so both were held pending a decision.

**This build** classifies failures instead of mislabelling them: auth,
connectivity, timeout, unavailable and other, from gh's stderr with `GH_DEBUG=1`,
with the candidate scrubbed and the detail sent to Debug Logs.

| Artifact | Value |
|---|---|
| APK | `build/app/outputs/flutter-apk/app-release.apk`, 64,282,982 bytes, versionCode 12 |
| APK SHA-256 | `09335f1a038b3470ba672babfeb257c871b4c66f522204e0a516cc94327ab96a` |

## Secure GitHub authentication — implemented, awaiting physical acceptance, 2026-08-31

Branch `feature/secure-github-auth`, from `develop` at `08c457e`.
**Not merged.** All automated gates are green; the nine physical tests are the
remaining gate.

### Design, after auditing what already existed

The injection half was already built: `applyGHProfile` sets `GH_TOKEN` and
`applyGitCredentials` sets an `http.https://github.com/.extraheader`
Authorization header through `GIT_CONFIG_KEY_0`/`VALUE_0`. Both are marked
secret, neither reaches argv, and no token-bearing URL is ever constructed. What
was missing was storage, a UI, and any way to configure the credential.
`Manager.Execute` is reused unchanged; no second execution path exists.

| Layer | Where it lives |
|---|---|
| At rest | `GitHubCredentialStore` — AES-256-GCM under a non-exportable Android Keystore key, ciphertext only, app-private |
| Host → Core | decrypted at Core launch, passed as `POCKETCLAW_GITHUB_TOKEN` |
| Core → tools | existing gh and git profiles, unchanged |
| Validation | `POST /api/pocketclaw/android/github/validate` on the loopback bridge, `gh api user` with a one-shot override |
| Status | `GET /api/pocketclaw/android/github/status`, the ambient credential |
| UI | `GitHubSettingsCard` — connect, test, disconnect; no reveal control |

Core's `credentials/github_token` plaintext fallback was removed. `gh auth login`
and `gh auth setup-git` are deliberately unused: both persist credentials outside
PocketClaw.

The manifest declared neither `allowBackup` nor `dataExtractionRules`, so
app-private files were backed up by default. Both are declared now and the
credential directory is excluded from cloud backup and device transfer.

### Applying a change

The credential is read at Core launch, so `ServiceManager.applyCredentialChange`
restarts Core through the same stop/start the config screen already uses. A
service that is mid-start is never interrupted: the change is queued and applied
from the existing status poll once it settles, and the card reports "saved, will
apply automatically" instead of claiming the credential is live.

### Reading failures

Destroying a credential is irreversible, so `GitHubCredentialStore.classify`
destroys only on positive evidence — a GCM tag that does not verify, ciphertext
that cannot be a valid block sequence, a malformed blob, or a permanently
invalidated key. A busy keystore or an unrecognised provider failure preserves
the ciphertext and reports the credential unavailable. The rule is a pure
function of the failure and is covered by JVM unit tests
(`./gradlew :app:testReleaseUnitTest`).

### Build

| Artifact | Value |
|---|---|
| Core `libpicoclaw.so` | `ae74a8584ea2010015591c3a65fe72cf02f2f1e9c89a6606d388906e3302eadd` |
| Core source fingerprint | `a61c0664f1932a577bac4498699be44ffca33105a0b757c2f2e1de7d0b6c1a7e` |
| APK | `build/app/outputs/flutter-apk/app-release.apk`, 64,278,210 bytes |
| APK SHA-256 | `eaddd9fc3ce1e9b0c02e4efb36e37fe44c3a406ab5e6c43be55d38c9a068f94f` |
| versionCode | 11 |

### Physical acceptance still to do

Connect; restart and stay connected; `gh api user`; `gh repo view` a private
repo; `git clone`, `fetch` and `pull` over HTTPS; inspect `.git/config`, gh
config, PocketClaw and Runtime logs and agent output for the raw token;
disconnect and confirm gh no longer authenticates; reconnect, install the APK
over itself without clearing data, and confirm the credential survives.

## Python Lite — Phase C PHYSICAL PASS and merged, 2026-08-31

Branch `feature/python-lite-agent-tool`, from `develop` at `de7ea53`.
**Physically validated inside the installed application, then merged to
`develop`.** Not released, `main` untouched, no tags moved.

### Root cause, confirmed on device

Android's CPython replaces `sys.stdout` and `sys.stderr` with `TextLogStream`,
which writes to the Android system log instead of file descriptors 1 and 2. The
runtime captures the descriptors, so managed runs wrote where the caller could
not read: `print()` succeeded and exited 0 with nothing captured, an uncaught
exception exited 1 with its traceback in logcat, and `os.write(1, ...)` worked
because it bypasses `sys.stdout`. CPython, the pipes, the capture layer and the
formatter were all correct.

### The fix

`pocketclaw_bootstrap`, a module inside the payload's appended standard library,
rebinds both streams to unbuffered UTF-8 wrappers over `os.dup(1)`/`os.dup(2)`,
then reads the program from stdin exactly as `python -` does and runs it as
`__main__`. The tool now invokes
`python <default_args> -m pocketclaw_bootstrap <caller args>`.

The source still travels on stdin only — not `-c`, not argv, not the
environment, not a log. Tracebacks compile under `<stdin>`, `sys.argv` is
`["-", ...]`, and the bootstrap's own frames are stripped so line numbers refer
to the submitted code. CPython is not patched.

Because the module ships inside the checksum-pinned payload, the payload hash was
repinned and the catalog moved to `2.2.0`. A new entry-point guard runs in the
Gradle release and in the Go tests, alongside the appended-stdlib, catalog and
source-freshness guards.

### Build

| Artifact | Value |
|---|---|
| Python payload | `dfa19e41ac57edfdaa7ba2d94de7d1c9fa8ce8e30db2c3385f1559c1d576848d`, 299 entries |
| Core `libpicoclaw.so` | `029f70307a9a84309f3d30ebca0cd2eab4fcd5ea9e18b90c49be477f3fcfd3a7` |
| Core source fingerprint | `535cbfd664415f66ea8bcc2dfcc2d1859e6905f118b4eeae44470f20cea17305` |
| Catalog | `2.2.0` |
| APK | `build/app/outputs/flutter-apk/app-release.apk`, 64,168,930 bytes |
| APK SHA-256 | `b92b958677794dbd7bca95d4ec040c6a41ed5707978e2f4bd5791aa787816884` |
| versionCode | 9 |

### Physical acceptance — PASS, 2026-08-31

Four tests, one `python` tool call each, no retries.

| Test | Device result | Verdict |
|---|---|---|
| `print("PYTHON-FINAL-PASS")` | `exit_code=0`, `stdout_bytes=18`, `PYTHON-FINAL-PASS` | **PASS** |
| `raise ValueError("TEST-ERROR")` | `exit_code=1`, `stderr_bytes=96`, real traceback ending `ValueError: TEST-ERROR` | **PASS** |
| `print("مرحبا 🐍")` | `exit_code=0`, `stdout_bytes=16`, `مرحبا 🐍` | **PASS** |
| infinite loop, `timeout_ms=2000` | `exit_code=-1`, `timed_out=true` | **PASS** |

The byte counters are the confirmation that this is the entry point working
rather than a coincidence: the same calls previously reported `stdout_bytes=0`
and `stderr_bytes=0` with the identical exit codes.


## Python Lite — Phase C physical FAIL, root cause not yet proven, 2026-08-31

Branch `feature/python-lite-agent-tool`, from `develop` at `de7ea53`.
**Not merged. The stdout/stderr failure is not fixed.**

### The physical result

| Test | Device result | Verdict |
|---|---|---|
| `print("PYTHON-FINAL-PASS")` | exit 0, stdout empty | **FAIL** |
| `raise ValueError("TEST-ERROR")` | exit 1, stderr empty | **FAIL** |
| infinite loop, `timeout_ms=2000` | exit -1, `timed_out=true`, ~2 s | PASS |
| `print("مرحبا 🐍")` | needed several calls; visible only after `os.write` | **FAIL** |

### What is proven

An empty `ExecResult.Stdout` has one possible cause: the capture layer received
nothing. A bounded buffer that saw bytes and kept none returns a truncation
marker rather than an empty string, so an empty stream can never mean "the
runtime dropped it". The `(empty)` text the device printed is therefore evidence
about the process, not about the formatter.

That rules out the formatter, the tool-result serialisation and the agent
message as the place the bytes disappear, and it rules out the hardening commit:
`git diff c4c0bf2..b42f813` touches the formatter, one log field and the new
fingerprint package — no capture, environment, catalog or argv code.

The whole production chain is now proven byte-exact on the host, from a real
child process's pipes through to `ContentForLLM()`.

### What is not proven

Where the bytes stop between CPython and the pipe. The evidence is consistent
with the interpreter's own `sys.stdout`/`sys.stderr` being disconnected — CPython
makes `print()` a silent no-op when `sys.stdout` is None, and an unhandled
exception still exits 1 when `sys.stderr` is None, which matches all four
observations including `os.write` working — but that is a hypothesis, not a
finding. It has not been reproduced: this machine has no device, no adb target
and no aarch64 emulation, and the host interpreter behaves correctly through the
same code.

### What the next physical run will settle

`ExecResult` now carries `StdoutBytes`/`StderrBytes`, measured at capture and
printed in the Python result:

- `stdout_bytes=18` beside `stdout: (empty)` → the loss is downstream of capture
- `stdout_bytes=0` → the interpreter wrote nothing the runtime could see

`stderr_bytes` also joins `bytes_out` on the INFO `runtime.exec.completed` event,
so a Debug Logs pull corroborates without a second run.

### Build

| Artifact | Value |
|---|---|
| Core `libpicoclaw.so` | `6210de2bd04980025aca045a6b3d8d6a0cbc1351aedae9fccdb8ec2264d33ff5` |
| Core source fingerprint | `25653d50d04fe4ae5bbb4cc7e62c9f357287933687fdf2178c5e4835756767d9` |
| APK | `build/app/outputs/flutter-apk/app-release.apk`, 64,164,250 bytes |
| APK SHA-256 | `c7844a51e9d5d3f0e59365b1fc587b578455cf1faad1d147a5b43bc0c98c690e` |
| versionCode | 8 |

Install over the existing app without clearing data, then run the four tests with
exactly one `python` call each and report the `stdout_bytes`/`stderr_bytes` line.

## Python Lite — Phase C hardened, awaiting the physical stderr recheck, 2026-08-31

Branch `feature/python-lite-agent-tool`, from `develop` at `de7ea53`.
**Not merged.** Core, Flutter and frontend gates are green and the ARM64 APK is
rebuilt; the physical stderr recheck is the only thing outstanding.

Phase C's own physical run passed on statistics, JSON, Unicode, an uncaught
exception and a 2 s timeout. A controlled single-call test then found the model
unable to report the traceback from `raise ValueError("TEST-ERROR")`. Two fixes
came out of it.

### The Python result names every field it has

`formatPythonResult` writes `exit_code`, `timed_out`, `cancelled`,
`stdout_truncated` and `stderr_truncated`, then both streams under their own
headings — an empty stream printed as `(empty)` rather than omitted. It used to
leave out a stream with no content, which made "the interpreter printed nothing"
and "the result dropped it" look the same to the model.

Nothing is reconstructed. The text is the runtime's own bounded, redacted
capture; a test fails if a traceback ever appears for a run that produced none.
The end-to-end tests run a real interpreter through `Manager.Execute` rather than
handing the formatter a hand-written `ExecResult`, because a hand-written one
cannot fail the way this failed. They cover the traceback reaching the agent, the
non-zero exit code, the bound on stderr, truncation being reported, and the
source staying out of the log.

Bounds, redaction and event privacy are unchanged, and the source still travels
on stdin only.

### Core staleness now covers Go-only changes

`pkg/coresource` hashes the Core's build inputs into one content-addressed
fingerprint: non-test Go source under `cmd/` and `pkg/`, the embedded catalog,
the embedded `workspace/`, `go.mod`, `go.sum`, and the Makefile that fixes the
build tags. `core/build-android-arm64.sh` computes it and stamps it into the
binary with `-X github.com/sipeed/picoclaw/pkg/coresource.Stamped=...`; the gate
recomputes it from the working tree and fails if the staged Core does not carry
it. The runtime's startup diagnostics log it as `core_source`, which is also what
keeps the linker from dropping the variable.

`TestStagedCoreEmbedsTheCurrentCatalog` stays. The two guards answer different
questions — does this Core know the current catalog, and was it built from the
current code — and Phase C is the case only the second one catches:
`python_tool.go` changed, `manifest.json` did not.

No mtime, timestamp, absolute path or build id takes part. The first draft hashed
every `*.json` under `pkg/`, which `pkg/cron`'s tests write into during a run, so
the gate failed against a Core that was current; embedded assets are named one by
one now, and a test reads the `//go:embed` directives out of the Core source so
the list cannot fall behind quietly.

The guard was proved in both directions in this session: it failed against the
Core staged before the rebuild — a Go-only change with the catalog untouched —
and passed against the rebuilt one.

### Build

| Artifact | Value |
|---|---|
| Core `libpicoclaw.so` | `c3af079fcb49403da3d8546f68d5e466b2bf83341fec5f1de56acf76b3d27390` |
| Launcher `libpicoclaw-web.so` | `c54098b5a5ce55c6e3c0251bd268e1a74516e69d19021441757898a53f3c8c8d` |
| Core source fingerprint | `9b9d44567c30380fa08e46ba39ef876e4c8f22e36f4b56fb1123f1d85615c8ae` |
| APK | `build/app/outputs/flutter-apk/app-release.apk`, 64,162,658 bytes |
| APK SHA-256 | `2e608a757580fe0503d7904cde2a341087884c457db640a31898072fcf48424a` |
| versionCode | 7 (bumped from 6 so the recheck cannot run against the old install) |

### Physical recheck still to do

1. `print("PYTHON-FINAL-PASS")`
2. `raise ValueError("TEST-ERROR")` — the Agent must report the real traceback
   ending in `ValueError: TEST-ERROR` and exit code 1, from one call
3. an infinite loop with `timeout_ms: 2000`
4. Unicode: `مرحبا 🐍`

## Python Lite — Phase C implementation record, 2026-08-31

Branch `feature/python-lite-agent-tool`, from `develop` at `de7ea53`.
**Not merged.** Automated gates are green; physical validation is outstanding.

The Agent now has a dedicated `python` tool:

```
python { "code": "...", "args": [...], "timeout_ms": ... }
```

It owns no execution machinery. `buildPythonRequest` produces an ordinary
`ExecRequest` and `Manager.Execute` does the rest, so resolution, checksum
verification, the `python` environment profile, catalog `default_args`, the
timeout ceiling, cancellation, process-group termination, output bounds, the
runtime event family and redaction all apply unchanged. The tool shares the
Managed Runtime's manager, so there is still one registry and one platform probe.

Source travels on **stdin** as `python -`, never in argv: argv is capped near
128 KB, is readable from `/proc/<pid>/cmdline`, and appears in argument
diagnostics, while stdin is accounted only as `bytes_in`. A regression test fails
if the implementation ever switches to `-c`.

Tracebacks name `<stdin>`, which is what `python -` reports. `<pocketclaw>` was
considered and rejected for v1: it would need a wrapper that reads stdin and
re-compiles the source, which is a cosmetic gain bought with an interpreter trick
around the exact path that carries user code.

v1 has no separate data channel. If a script needs structured input it can embed
it or read a workspace file; a second stdin field would compete with `code` for
the one channel the interpreter reads.

The tool description steers deliberately: jq for simple JSON, rg for search,
sqlite3 for a single query, curl for HTTP, and Python for arithmetic,
statistics, multi-step logic, custom parsing and work that would otherwise take
several runtime round-trips. It states plainly that Python is **not** a sandbox
and that the boundary is the application UID. Tests fail if that steering
disappears or if the description starts claiming containment.

`python` is enabled by default and switchable independently of `runtime`,
because it runs arbitrary code as the application.

## Python Lite — Phase B PHYSICAL PASS and merged, 2026-08-31

Branch `feature/python-lite-runtime`, commits `bfe47e6`, `c376837`, `c60b15f`,
`bfc2074`. **Physically validated inside the installed application** on
SM-A165F / Android 16 / API 36, then merged to `develop`. Not released, `main`
untouched, no tags moved. Phase A was a physical PASS in its own right.

Observed in the real app: catalog **2.1.0**, **56** tools, 53 available, `python`
present. `runtime {tool: python, args: ["--version"]}` returned
**Python 3.14.7**, and a script through the Managed Runtime produced
`PYTHON-PASS`, `SUM=5`, `مرحبا 🐍`, `SQLITE-PASS`. No shell was required.

| | |
|---|---|
| CPython | 3.14.7, NDK 28.2.13676358, API 24, arm64-v8a |
| Payload | `libpocketclaw-python.so`, 11,509,517 bytes, `a302c990…ad1b` |
| Core | `95a9b33b…`, embeds catalog 2.1.0 |
| Provenance | bzip2 1.0.8, XZ 5.4.7, SQLite 3.50.4 — all built from pinned source |

Static extension modules, `lib-dynload` empty, standard library appended to the
ELF as a `.pyc` zip. No pip, no ctypes, no direct Python sockets, no writable
executable storage.

**Python is not a sandbox.** The boundary is the Android app UID; `subprocess`
remains a Runtime-observability bypass and that guidance is advisory, not
enforcement. Shell availability is version-dependent: Android 11+ ships
`/bin/sh`, API 24-29 does not.

Two packaging defects were found and are permanently guarded: Gradle stripping
the appended stdlib (`keepDebugSymbols` plus an EOCD check in the build guard),
and a stale Core shipping beside a new payload
(`TestStagedCoreEmbedsTheCurrentCatalog`). **Core must be rebuilt whenever the
embedded Runtime catalog changes.**

Phase C — the Agent-facing Python tool — is open on
`feature/python-lite-agent-tool` and not started.

## Python Lite — Phase A COMPLETE (build + measurement), 2026-08-31

Branch `feature/python-lite-runtime`. Architecture review approved for Phase A
only. **Phase A is host-side build and measurement. Nothing is integrated:** no
catalog entry, no tool count change, no `python_tool.go`, no production APK
payload, no merge.

CPython **3.14.7** cross-built for `aarch64-linux-android` API 24 on PocketClaw's
own NDK **28.2.13676358**, using upstream `Android/android.py` with a single
patched line (the NDK version). Extension modules linked statically, so the
interpreter is one self-contained PIE ELF and `lib-dynload` is empty.

Measured, not estimated:

| | bytes |
|---|---|
| Interpreter, stripped, LTO | 9,123,056 |
| Payload (interpreter + `.pyc` stdlib) | 11,591,387 |
| APK increase, measured against the shipped APK | +5,814,942 |
| Projected APK | 64,147,589 |

Both Phase A gates pass: APK increase 5.55 MiB (limit 7.5 MB), installed
11.06 MiB (limit 13.0 MB).

The stdlib is appended to the ELF as a zip and imported by `zipimport` with
`PYTHONHOME`/`PYTHONPATH` set; verified functionally on the host. `.pyc` is
kept over `.py` despite costing 943,381 bytes because it starts ~4.5x faster.

`hashlib`, `hmac` and `secrets` work with no OpenSSL. `socket`, `ssl`, `ctypes`,
`multiprocessing`, `email` and `http` are absent by construction.

**PHYSICAL VALIDATION PASSED (2026-08-31)** on Samsung SM-A165F, Android 16,
API 36, arm64-v8a: 52 checks passed, 0 failed. Standalone CPython runs from
`nativeLibraryDir` under the app uid, the appended-zip stdlib imports on
hardware, `sqlite3` 3.50.4 works with FTS5 and JSON1, `hashlib` works with no
OpenSSL, Arabic and emoji round-trip, a runaway loop dies in 25 ms with no
orphan. Startup: bare 90 ms median, typical imports 121 ms median. RSS 11.2 MB
bare, 20.8 MB for a 20k-object JSON workload.

Correction carried out of the run: **Android 11+ does have `/bin/sh`** (a
symlink `/bin` -> `/system/bin`, mksh), so `subprocess(shell=True)` and
`os.system()` work on API 30+ and the architecture review was wrong to call them
unusable. PocketClaw's minSdk is 24, so shell availability is conditional on the
device. This sharpens the existing "subprocess is a bypass, guidance is
advisory" conclusion rather than changing it.

Second gap: upstream's Android tooling downloads prebuilt dependency binaries
with no checksum verification. The Phase A build script pins them by SHA-256,
but bzip2 and xz remain third-party binaries. Phase B must build them from
pinned source, as SQLite already is.

Full record: `runtime/PYTHON_LITE_PHASE_A.md`.

## Next milestone — Python Lite Runtime

Branch `feature/python-lite-runtime`, from `develop` at `b46921e`. Nothing
implemented; the branch exists so the work starts from the merged Provider
Resilience baseline.

**The next session produces an architecture review and nothing else.** Do not
compile, download or bundle Python until that review is approved.

### What the review must not assume it may use

No Linux distribution, PRoot, apt, compiler toolchain, GCC/Clang, make, Node,
npm, arbitrary executable downloads, pip by default, native wheel compilation or
shell environment emulation. v1 is an interpreter plus a selected standard
library under PocketClaw-controlled execution, with no unrestricted package
ecosystem.

### The execution model is already settled

Managed Runtime proved it physically: executables ship in the APK and run from
`nativeLibraryDir`, and writable executable storage is **not** used. Do not
propose writing a native Python binary into `filesDir` and exec'ing it — Android
refuses that, and the whole Runtime design exists because of it. Writable Python
data may live app-private; stdlib resources may ship as non-executable assets.

One inherited limitation worth carrying into the review: PocketClaw's git ships
without a usable `/bin/sh`, because Android has none. Anything in Python that
assumes a shell — `os.system`, `subprocess` with `shell=True`, some `tempfile`
and `webbrowser` paths — needs the same honesty applied to it.

### Honesty requirement

The review must state the real security boundary. Python cannot be perfectly
sandboxed on top of this architecture, and claiming otherwise would be worse
than shipping nothing: it would let the Agent treat Python as safe when it is
an escape hatch around Runtime security. Say what actually holds.

### Size gate

APK is ~55.6 MB today. Exact projections required before inclusion; 100+ MB
needs explicit approval.

## Shipped — Provider Resilience & Automatic Failover (2026-08-30, PHYSICAL PASS)

Branch `feature/provider-resilience-failover`, merged to `develop`. Commits
`68443c1`, `7b67493`, `ebf49b4`, `812a003`, `3446b0f`. Not released, `main`
untouched, no tags moved.

Physically validated on the target ARM64 device: automatic failover, the
Fallback Models UI with ordered selection, a failing primary answered by its
configured fallback with one user-visible answer, automatic gateway restart after
both model and fallback changes, active-turn safety, and white-screen resume
recovery with no loop.

### What a next session must not undo

- **No checkpoint subsystem, no tool fingerprinting, no side-effect
  classification.** The agent loop already guarantees a provider retry does not
  rewind completed tool execution. It is protected by tests plus one exact
  `toolCallID` reuse guard. Do not match on tool name or arguments: asking for
  the same command twice in a turn is legitimate.
- **Two restart invariants are absolute.** A busy gateway is never force
  restarted when the two-minute wait expires, and an unverified busy state is
  never read as idle. Both leave the config saved and unapplied. An earlier
  revision did force both; it was wrong.
- **Unknown is never idle, and never a claimed cause.** The resume diagnostics
  log `probe_failed` and `page_unresponsive`, not `renderer_gone` —
  `webview_flutter_android` 4.14.0 has no `onRenderProcessGone`, so renderer
  death is not observable here. A test fails if the old label returns.
- **Healthy resumes are never reloaded.** The probe exists so page state and
  scroll survive; a blanket reload would hide the defect rather than fix it.
- **Streaming failover stops at first visible output**, or the answer duplicates
  on screen.
- **Fallbacks are references by model name**, so each keeps its own provider and
  credentials. Never copy the primary's key into a fallback.

### Counts

Tools **18**. Skills **7/7** on an existing workspace, **6/6** fresh. The GitHub
Skill stays removed and `picoclaw-agent` stays unseeded; the count is not a
target.

### Next

Python Lite, on `feature/python-lite-runtime` from the new `develop` HEAD.
Architecture review only — nothing to be compiled or bundled until that review
is approved.

## Shipped — Lean Runtime Pack v2 (2026-08-30, PHYSICAL PASS)

Branch `feature/lean-runtime-pack-v2`, fix commit `7ebd254`, from `develop` at
`fa27ad2`. **Physical validation PASSED** on a real ARM64 device on 2026-08-30,
then merged to `develop`. Not released, `main` untouched. Read `RUNTIME.md`
first.

Physical: Git HTTPS PASS, `git clone` of a public GitHub repository PASS,
**Git helper symlink execution on Android PASS**, git 2.51.0, gh 2.82.1, curl
HTTPS PASS, ripgrep PASS, sqlite3 PASS.

Skills: **7/7** on an existing upgraded workspace, **6/6** on a fresh install.
Both are correct — seeding only writes and never deletes, so an existing device
keeps the GitHub Skill it already had, while a fresh workspace gets the six
seeded skills (`picoclaw-agent` is deliberately unseeded).

git 2.51.0, gh 2.82.1, curl 8.11.1, ripgrep 14.1.1 and sqlite3 3.50.4 now ship
alongside jq. APK 34.7 MB -> 58.3 MB.

### Fixed after the first physical run

`git clone` failed with `unable to find remote helper for 'https'`. That message
names the protocol and sends you to TLS; the actual cause was that git spawns
`git remote-https` and resolves the literal name `git` through PATH, and the
helper directory held only the two remote helpers. git now declares itself as
one of its own helpers. If a tool re-invokes itself by name, it needs an entry
for itself — that is the general lesson.

### The platform question that is now answered

`git clone` over HTTPS depends on **executing through a symlink** in app-private
storage that points at a packaged payload. Android cannot package a file named
`git-remote-https`, so the runtime builds a symlink directory and points
`GIT_EXEC_PATH` at it.

**The device confirmed this works.** A symlink is not an executable: the kernel
resolves it and runs the read-only packaged file, so the API 29+ restriction on
executing writable app storage does not apply. This is the mechanism that makes
multi-executable tools possible on Android at all, and it is now evidence rather
than reasoning.

Do not "simplify" it by copying a helper into app storage. That is the one thing
Android definitely refuses, and the reason this design exists.

### What a next session must not undo

- **No credential ever goes in argv.** git gets its token through
  `GIT_CONFIG_COUNT`/`GIT_CONFIG_KEY_n`/`GIT_CONFIG_VALUE_n`, gh through
  `GH_TOKEN`. Do not "simplify" this to `https://TOKEN@github.com/...`: that
  leaks the credential into the command line, into git's on-disk remote config,
  and into any error quoting the URL.
- **Helper payloads are verified like main payloads.** A tool whose helper fails
  its checksum resolves unavailable on purpose. git with an unverified transport
  helper looks installed and then fails at the first `https://` URL.
- **gh is not a precedent.** Anything else above roughly 10 MB installed needs
  its own justification. yq was rejected at 11.25 MB with jq already present.
- **The build-path privacy check is deliberately narrow.** It tests for this
  build's own home directory plus boundary-anchored developer roots. A bare
  `/root/` search false-positives on Go's trimmed module paths such as
  `pkg/root/trusted_root.go`. Do not widen it back.
- **curl is real curl.** Do not replace it with a lookalike, and do not name any
  in-house HTTP tool `curl`.

### Where things are

- `runtime/android-build-env.sh` — shared cross-build setup and the
  strip/verify/install step every payload goes through.
- `runtime/build-{curl,git,ripgrep,sqlite3,gh}-android-arm64.sh` — one per
  payload. Run curl before git; git links the libcurl it leaves behind.
- `runtime/patches/git-android-pthread-cancel.h` — the bionic shim, and the only
  PocketClaw modification to git's source. It is also what satisfies the GPL
  source-offer obligation alongside the pinned tarball.
- `core/src/pkg/pcruntime/{helpers,environment}.go` — the symlink farm and the
  per-tool environment profiles.

### Known limitation

git ships with its default compiled-in `SHELL_PATH` of `/bin/sh`, which Android
lacks. **Shell-dependent git features — hooks above all, and git's `ENOEXEC`
fallback — are not guaranteed on Android.** Clone, fetch and push do not need a
shell. Overriding `SHELL_PATH` breaks git's own cross-build because its Makefile
uses the same variable for its build recipes; see `DECISIONS.md`.

### Next

Provider Resilience and Automatic Failover, on its own branch from the new
`develop` HEAD. SSH is worth reconsidering now that HTTPS git is proven, and the
deferred release-engineering items in `TASKS.md` still stand, including the
high-priority `extractBinaryFromApk()` dead path.

## Shipped — Managed Runtime Foundation (2026-08-30, PHYSICAL PASS)

Branch `feature/managed-runtime-foundation`, commit `ee236da`, based on
`v0.2.0-rc2` / `404ef44`. **Physical validation PASSED** on a real ARM64 device
on 2026-08-30, then merged to `develop`. Not released, `main` untouched. Read
`RUNTIME.md` first; it is the architecture of record.

Physical results: runtime tool registered; jq 1.7.1 executed and processed JSON;
`sha256sum`, `grep`, `sed`, `tar`, `uname`, `df`, `ping` executed; stderr
captured; non-zero exit preserved; timeout terminated a long-running command;
lifecycle logs appeared; no secret leakage; the runtime was driven through
Telegram; Auto-Start and the Gateway PID fix held.

**43 of 44** catalog tools were available. `traceroute` was correctly reported
unavailable — the resolver measuring the device instead of trusting the catalog,
which is exactly what it is for.

**The writable-app-data probe returned INCONCLUSIVE on that device**, and that is
fine. Nothing depends on it: the bundled jq payload executed from
`nativeLibraryDir` on the same run. Do not read an inconclusive probe as licence
to try writable execution — the two supported routes are unchanged.

### Counts, and one that will look wrong

- Tools: **18**
- Skills: **7/7**

**Skills 7/7 is correct and is not a regression.** The user deliberately removed
the incomplete GitHub Skill. Do not restore it, do not write a migration for it,
and do not treat 8/8 as the target.

Note for whoever picks this up: `core/src/workspace/skills/github/` is still in
the repository, so a *freshly seeded* workspace would receive it again. The
device count of 7 and the repo's seeded set are therefore consistent today only
because the device workspace already exists. If 7/7 is meant to hold for new
installs too, the skill needs removing from the seeded tree or adding to
`unseededTemplates` in `cmd/picoclaw/internal/onboard/helpers.go`. That was not
done here because it was not asked for.

### What a next session must not undo

- **There is no provisioning subsystem, and that is deliberate.** PocketClaw
  targets SDK 36; an app targeting API 29+ cannot `execve()` a file in its own
  writable data directory, and `File.setExecutable(true)` does not change it
  because the restriction is enforced on the app's SELinux domain rather than by
  the file mode. Do not add a download-verify-activate pipeline for executables:
  it could not run, and its absence is what makes "no arbitrary binary
  installation" structural instead of a policy. Any future provisioning
  abstraction is limited to non-executable assets.
- **Bundled payloads must be named `lib*.so`** and must appear in both
  `requiredArm64NativeLibraries` and `keepDebugSymbols` in
  `android/app/build.gradle.kts`. The package manager only unpacks
  `lib/<abi>/*.so` into `nativeLibraryDir`; any other name ships a payload that
  can never run. Without `keepDebugSymbols`, Gradle strips the executable during
  packaging, changing its bytes and breaking the catalog's pinned SHA-256, which
  reaches the device as a `checksum_mismatch` that looks like a corrupt install.
- **System tools carry no hash, on purpose.** The OS owns `/system/bin` and
  replaces it on every update, so `security_class: system` reports
  `platform_owned` and the catalog format refuses to record a checksum. Do not
  "improve" this by pinning one.
- **Managed execution never uses `sh -c`.** It is deliberately narrower than the
  existing `exec` tool, which does. Do not relax one to match the other.
- **Flag-directed redaction is scoped per tool.** Applying curl's flag list
  globally would blank the filename in `sort -u notes.txt` and the pattern in
  `grep -E '<expr>' file`. Do not make it global.

### Where things are

- `core/src/pkg/pcruntime` — manifest and catalog, resolver, execution API,
  lifecycle events, redaction, inventory, execution probe, storage policy.
- `core/src/pkg/pcruntime/manifest.json` — the catalog. Adding a bundled tool
  means updating its SHA-256 here; `TestBundledPayloadsMatchTheirPinnedChecksums`
  fails while the catalog and the packaged payload disagree.
- `core/src/pkg/tools/runtime_tool.go` — the `runtime` agent tool, wired in
  `pkg/agent/instance.go` and gated by `cfg.Tools.Runtime`.
- `runtime/build-jq-android-arm64.sh` — the jq payload build.
- `PicoClawService.buildEnvironment()` exports `POCKETCLAW_RUNTIME_LIB_DIR` and
  `POCKETCLAW_RUNTIME_DIR`.

### Known gap for physical validation

`core/src/workspace/AGENT.md` is seeded only when the file is **missing**
(`copyMissingEmbeddedToTarget`). A device with an existing workspace keeps its old
`AGENT.md` and will not see the new runtime guidance. The same applies to the
GitHub Skill. This is why the essential rules are also in the `runtime` tool's
own description, which is always in the prompt when the tool is registered — but
a tester checking the workspace file should use a fresh workspace.

### Next

Lean Runtime Pack v2 on `feature/lean-runtime-pack-v2`: Git and GitHub CLI
first, then an HTTP/TLS capability, ripgrep, sqlite3 and yq. See `TASKS.md` for
the deferred release-engineering items, including the high-priority
`extractBinaryFromApk()` dead path.

## Shipped — v0.2.0-rc2, Auto-Start and Gateway PID ownership (2026-08-30)

`v0.2.0-rc2` is frozen, merged to `develop`, tagged, and published as a GitHub
pre-release. Both commits PASSED physical validation on a real ARM64 device on
2026-08-30, with reference APK SHA-256
`182b85183156428aa93baf3113492484258a3a1eace95f7bccd9ed82035177a3`.

- `75ac0d9` — Auto-Start safe rebuild: Service and Gateway auto-start, manual
  stop and start, no resurrection, internal chat, Telegram, Skills 8/8, Tools
  17, Core bound only to `127.0.0.1:18790` / `[::1]:18790`.
- `90194c8` — Gateway PID ownership fix: the false positive did not recur.

`feature/autostart-foundation` was NOT merged and must not be. The shipped work
was rebuilt from `v0.2.0-rc1`; see `DECISIONS.md`.

The next milestone is Managed Runtime Foundation, per `TASKS.md`.

### Gateway PID ownership on Android

`validateGatewayPidData` proved ownership by running `ps -o command= -p <pid>`
and looking for the bare `gateway` subcommand from the launch line
`<binary> gateway -E --no-color`. On Android the Gateway is executed as
`.../lib/arm64/libpicoclaw.so` and `ps` does not report that argv, so a live,
launcher-spawned Gateway was classified foreign and its pid file deleted.

The log message "pid belongs to another process" is reachable only when `ps`
succeeded with non-empty output lacking the token — that is how the cause was
established rather than guessed.

Startup and status disagreed because the readiness goroutine in
`startGatewayLocked` accepts the pid file on `pd.PID == pid` alone and never
calls `sanitizeGatewayPidData`. Only status, the realtime proxy, and manual
start ran the heuristic.

Two rules now hold, and should not be undone:

- `exec.Cmd` ownership outranks `ps`. If this launcher spawned the exact pid and
  the process is alive, it is owned, full stop. Pid reuse cannot defeat this:
  the child stays a zombie holding its pid until `cmd.Wait()` returns, and the
  monitor goroutine clears `gateway.cmd` at that moment.
- A negative command-line match is decisive only when `ps` actually returned a
  command line. A bare executable name is un-inspected, not foreign, and falls
  through to the health probe. Do not "restore" bare-name rejection.

Dead-process cleanup is independent of all of this — it lives in
`ppid.ReadPidFileWithCheck` and runs first — so relaxing the heuristic cannot
leak stale pid files.

`feature/autostart-foundation` is abandoned. Do not base work on it, merge it,
or cherry-pick from it. It is kept intact for reference only. Its automated
tests were green, but on a device a manual Start Service press could land in a
FAILED state that RC1 cannot even represent — RC1's `ServiceStatus` is
`{ stopped, running, starting }`, with no `failed` member.

The three mechanisms behind that regression, so it is not reintroduced:

1. It changed native `PicoClawService.start()` to return a `Boolean` and mapped
   `false` to `ServiceStatus.failed` in Flutter. `false` also meant "skipped",
   returned whenever the process-lifetime `isStarting` / `isStopping` /
   `manualStopActive` companion flags were set. Those flags outlive the Service
   object and had paths that never cleared them, so one stale flag made every
   later manual Start fail.
2. It set `hasFailed = true` after every child-process exit, including a
   requested stop, from a thread outside the lock that had just cleared it. A
   clean manual Stop therefore left the service marked failed.
3. `inspectServiceState()` latched on FAILED and reported it even when the
   native side said the service was cleanly stopped.

Design rules this rebuild holds to, and the reasons:

- Auto-start evaluates exactly once, from `main()`, at a true app-process
  launch. There is no resume hook. A manual Stop stays stopped until the user
  starts it again or relaunches, and that is guaranteed structurally rather
  than by a flag.
- `LaunchAutoStartPreferences` (Android) is the only store: the existing
  `picoclaw_prefs` file under new keys, synchronous `commit()`, acknowledged
  only from a post-commit readback. RC1's `auto_start` boot-receiver key is a
  different preference and is untouched.
- Gateway auto-start is a preference, not a lifecycle. The Service exports
  `POCKETCLAW_GATEWAY_AUTOSTART` and Core's existing `TryAutoStartGateway()`
  is gated on it. Do not add a Flutter gateway lifecycle, an Android gateway
  bridge, or a second definition of gateway readiness — the experimental
  branch's `gatewayRuntimeReady()` also introduced a new HTTP 400
  `precondition_failed` failure mode on the RC1 web console's own Gateway
  button.
- `START_NOT_STICKY`, and a null restart intent stops the Service. RC1's
  `runWebService` crash restart is intentionally unchanged; it is guarded by
  `stopped` and never fires on a manual stop.

## Previous objective

**v0.2.0-rc1 is frozen, merged to `develop`, tagged, and published as a GitHub
pre-release. No further RC work is pending.** The next objective is the
deferred roadmap in `TASKS.md`, not another release pass.

The DEBUG log-cleanup physical gate that previously blocked release is CLEARED.
The user physically validated
`5760247a17ccff68d188180875f1812c150dd7ae6de45e3f109d7ba07c2186b9`
on a real ARM64 Android device on 2026-08-29, covering startup, Skills 8/8,
Tools 17, Telegram owner-only authorization and final delivery, Web/realtime
owner authorization, dashboard password/session auth, Core staying loopback-only
on 18790, Public Mode OFF/ON with live OFF→ON→OFF→ON rebinding and no manual
service restart, a real `192.168.x.x:18800` LAN URL, authenticated LAN Dashboard
access from a computer, QR/connect refresh, Telegram continuity across the mode
change, Web Logs stability, UTF-8/Arabic/emoji/µs rendering, and every earlier
privacy expectation.

What a later session most needs to know:

- The Go sweep has no accepted failing region any more. Build and test with
  `-tags goolm,stdjson` from `core/src`; the pure-Go Olm implementation removes
  the `olm/olm.h` host dependency entirely. Treat any Go failure as real.
- `go test -race ./web/backend/api` fails in
  `TestStartGatewayLocked_UsesReloadedConfigForBootSignature`. The test's
  cleanup and the production monitor goroutine both call `cmd.Wait()` on the
  same `exec.Cmd`. It is a test-harness defect, it reproduces identically on
  `develop` at `8f861bc`, and the same Kill+Wait pattern appears in several
  other tests in that file. Fixing it is deferred and tracked in `TASKS.md`.
- Core is not byte-reproducible by default because `BuildTime` is stamped via
  `-ldflags`; two builds of identical source differ in exactly 64 bytes at the
  same length. To compare a binary against a reference, re-run
  `make build-android-arm64 BUILD_TIME_RAW='<the reference stamp>'` and the
  hashes match exactly. That is how this RC's native payload was proven to be
  the physically validated payload.
- Core on 18790 is loopback-only structurally, not by configuration.
  `gatewayHostOverride()` returns `localhost` unconditionally and is exported to
  the gateway child as `PICOCLAW_GATEWAY_HOST`, which outranks the config file.
  Do not "restore" config-driven gateway host binding without a security review.
- Dashboard sessions are in-memory, so restarting Core signs everyone out. That
  is intended, not a bug.

Install:

    build/app/outputs/flutter-apk/app-release.apk
    com.lord1egypt.pocketclaw 0.2.0 (4) · arm64-v8a

All of the following were confirmed on the device in this pass and need no
re-verification unless a concrete regression appears:

1. A DEBUG log export containing HTTP durations carries valid `53.616µs`, with
   no U+FFFD introduced by PocketClaw.
2. Arabic, emoji, Unicode punctuation, and ANSI/control removal are correct in
   both Logs and Export.
3. Sustained successful `GET /api/gateway/logs` and `GET /api/gateway/status`
   polling does not consume user-visible history, while failed polls and
   ordinary requests such as config/models stay visible.
4. Basename callers, user-facing debranding, and exactly-once queue/drain
   behavior are correct.
5. Telegram, skills, background/locked operation, providers, Skill Hub,
   startup, and performance all pass.

The Core binaries in this build are the exact ones validated in that run;
rebuilding this source with their timestamps pinned reproduces them byte for
byte.

### What the DEBUG-export bugs were

The sanitizer was already Unicode/rune-safe. Corruption happened later in
Android Export Logs: Dart `content.codeUnits` (UTF-16 code units) was truncated
to bytes, making `µ` a lone invalid `B5`. The MediaStore bridge now receives
real UTF-8 bytes from `utf8.encode`; the actual MethodChannel export boundary
has a strict-decode regression test.

The polling entries were genuine DEBUG middleware events, not the old sticky
snapshot bug. Middleware now omits only exact successful 2xx `GET` requests to
`/api/gateway/logs` and `/api/gateway/status`. Errors, unexpected methods,
redirects, unknown routes, and other API requests remain logged. Do not alter
`publishLog`/`takeNewLogs` or reintroduce `status['lastLog']`.

### What the two bugs actually were

`-trimpath` removed `/home/lordegypt/...` from the caller and left the Go
**module path** in its place; every existing check looked only for `/home/`, so
nothing caught it. Fixed with `zerolog.CallerMarshalFunc` in
`pkg/logger/logger.go` — the earliest structured layer, so all writers and
exports inherit it.

The repeated warning was **not** the backend emitting repeatedly. Native
`lastLog` is a sticky snapshot, and the Flutter three-second poll appended it
every tick, filling the 500-entry buffer from one event and evicting real
history. Fixed by draining: `publishLog`/`takeNewLogs` hand each line out once.

Do not reintroduce `status['lastLog']` as a log source — that is the bug.

The third defect: native Settings hardcoded "Connect PocketClaw to Telegram"
and never read any state, so it disagreed with the console. Both surfaces now
derive from `channel_list.telegram` via `TelegramConnectionReader`. Do not add
a stored "connected" boolean — that is what one source of truth prevents.

## Milestone D — COMPLETE

Phase 2 Milestone D, Telegram Managed-Bot Onboarding, passed its physical
end-to-end test on a real Android device on 2026-08-26 and is closed. Merged
into `develop` with a non-fast-forward merge and tagged `phase2-milestone-d`.
`main` remains deliberately untouched at `100a51d`.

Automatic managed-bot onboarding is now the **default** Telegram setup path.
Manual Bot Token entry is the **advanced fallback**, still complete and
reachable.

### The verified reference artifact

    APK              b6fea5d8ec5c3c66ba8a1320b0a217afcca322e75b5b26cc4082bbbb08a57f94
                     34,220,929 bytes · com.lord1egypt.pocketclaw 0.1.3 (3)
    libpicoclaw.so       37,224,801 · 33f8b4efbc88333747c5df30b3ddc6864c924b35dba99e9c8c3b91df3470e98a
    libpicoclaw-web.so   24,641,889 · 5400cb02322ece5c7035356595355bd3c116adfc6a5bb6b78f3e2bd22dbcb3bd

This supersedes the Milestone C APK `588bbec1...5ff3a90b` and the Milestone C
Core pair. Diff any future regression against it first.

### Live architecture

    PocketClaw Android
      → PocketClaw Telegram Setup  (Vercel)
        → @PocketClawSetupBot      (manager, Bot Management Mode)
          → Telegram Managed Bots
            → Upstash Redis        (pairing state, REST, 600 s TTL)
              → automatic PocketClaw Telegram configuration

Official manager `@PocketClawSetupBot`. Service
<https://pocketclaw-telegram-setup-bot-83ai.vercel.app>. Public service
repository `Lord1Egypt/PocketClaw-Telegram-Setup` — independent, public, MIT,
and not a build input to the APK.

What was verified physically: the full Channels → Telegram onboarding UI, bot
creation through Telegram, automatic pairing detection and token delivery,
automatic owner and channel configuration, the connected state, Open Chat, and
a real Telegram → Core → AI provider → Telegram message round trip with context
persisting across consecutive messages. The full record is in
`PROJECT_STATE.md`.

### Still open, and deliberately so

- **Rotate the manager bot token.** It was pasted into a chat log after
  deployment. The service keeps working; rotate it in BotFather and update the
  Vercel environment variable. This is the one outstanding security item and it
  is not blocked by anything.
- Localize the Telegram onboarding strings — English in all twelve locales.
- Confirm a `claude-*` model on OpenCode, carried over from Milestone C.

### Architecture worth knowing before touching Telegram again

Telegram has two entry points and they must stay converged on one flow:

- Native Settings → Telegram (`lib/src/ui/config_page.dart`)
- Channels → Telegram in the embedded console
  (`core/src/web/frontend/.../channel-config-page.tsx` → `telegram-panel.tsx`)

Both call `TelegramOnboardingLauncher.open`. The console never runs pairing
itself; it asks the Flutter host over the `PocketClawHost` bridge
(`lib/src/ui/webview/pocketclaw_host_bridge.dart`). Do not reimplement pairing
in TypeScript — that split is what made Milestone D ship unreachable the first
time. The bridge is scoped to the console's origin and `openExternal` accepts
only absolute http(s) URLs; both are tested, do not loosen either.

The lesson worth keeping: a Flutter widget test proves the code runs, never
that a user meets it. Any UI claim here needs a test entering through the same
route the app does.

## Milestone C state

Merged into `develop` as `36bc88d` and tagged `phase2-milestone-c` on
2026-08-26, after passing on a device. The Core source lives in this repository
at `core/src/`, and `core/pocketclaw-core-v0.3.1.patch` (52 files) is its
divergence from upstream `v0.3.1`; the arm64 binaries are committed under
`android/app/src/main/jniLibs/arm64-v8a/`.

What it delivers: a provider-first Add Provider flow (choose provider → API key
→ fetch or type a model → save) with the alias derived automatically; the
backend catalog extended with categories, documentation links, and the xAI,
Together AI, Fireworks AI, and Custom OpenAI-Compatible presets, all registered
in the protocol switch; working Gemini model discovery; broader fetch error
classification; and no runtime logo fetching from third-party hosts.

Read `docs/PROVIDER_ARCHITECTURE.md` before touching provider code. Two facts in
it will save a wrong turn: AI provider configuration lives in the Core web
console and not in Flutter, and a provider added to the catalog without a
matching arm in `CreateProviderFromConfig` saves cleanly and then fails at
request time with `unknown protocol`.

Deliberately not done: Telegram QR/deep-link onboarding (next milestone), any
release hardening or obfuscation, and any change to API key storage. The last
is not an oversight — see the deferral decision in `DECISIONS.md`.

## OpenCode completion state

OpenCode Zen (`https://opencode.ai/zen/v1`) and OpenCode Go
(`https://opencode.ai/zen/go/v1`) are mixed-protocol gateways: one base URL and
one API key front OpenAI Responses, OpenAI-compatible chat completions, and
Anthropic Messages, and the protocol is a property of the model. All routing is
in `pkg/providers/opencode_routing.go` — if you need to change how an OpenCode
model is dispatched, that is the only file to touch.

Supporting changes: `pkg/providers/openai_responses` is a new generic
Responses-over-HTTP provider (the Azure and Codex ones are hardcoded to their
own endpoints and were not reusable), and `anthropic_messages` gained an opt-in
`WithBearerAuth()` used only by the OpenCode Messages route.

The one thing tests cannot settle: whether OpenCode's Messages surface wants
`X-API-Key` or `Authorization: Bearer`. Both are sent. If a `claude-*` model
401s on the device while `gpt-*` and `kimi-*` succeed, that is the header
question, not the routing.

Build gotcha found the hard way: `:app:packageRelease` without
`-Ptarget-platform=android-arm64` silently produces a ~50 MB universal APK
instead of ~34 MB. Check the size before trusting any release build.

## Milestone B closure

APK `ba4f067df9811bd0e4af713343bdba632abbf96a41e3a5b47cf154740f70a4b8` passed a
full physical Android device test on 2026-08-25: install, app launch / first
frame, no black screen, Gateway/Core lifecycle, navigation, PocketClaw branding,
workspace path, QR/access page, Skill Hub, provider/model flow, and no abnormal
slowdown. It is the verified reference artifact — compare any future regression
against it before forming new hypotheses.

`recovery/pocketclaw-clean-debrand` (@ `f25d38e`) was merged into `develop` with
a non-fast-forward merge, `225be3c`, and the closure point is tagged
`phase2-milestone-b`. The merged `develop` tree is byte-identical to the tested
recovery tip. The recovery branch, `feature/pocketclaw-identity` (`354fc38`
WIP), and `main` are all intact; nothing was deleted or rewritten.

What Milestone B delivered: independent PocketClaw product and package identity;
the black-screen root cause and its permanent fail-closed release guard; full
user-facing debranding including the embedded web runtime rebuilt from source;
`Download/pocketclaw` for fresh installs; the MQTT `/pocketclaw` default with
backward compatibility; and the upstream agent skill dropped from fresh
workspace seeding.

## Permanent release requirements — do not remove

- `packageRelease` fails the build unless the APK contains
  `lib/arm64-v8a/libdartjni.so`, `libpicoclaw.so`, and `libpicoclaw-web.so`.
  It prints "Verified arm64-v8a native payload" on success.
- Canonical release command:
  `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64` with
  `JAVA_HOME=/home/lordegypt/PocketCLaw/.tooling/jdk-17` and
  `GRADLE_USER_HOME=/home/lordegypt/PocketClaw-App/.tooling/gradle-stage-a-clean`.
  A universal `flutter build apk --release` is not a PocketClaw release path.
- If the guard reports `libdartjni.so` missing, do not change application code:
  purge `~/.pub-cache/hosted/pub.dev/jni-*/android/.cxx/` and rebuild.
- Rebuild Core with `core/build-android-arm64.sh`, which builds from
  `core/src/` through the root Makefile targets. Calling
  `make -C web build-android-arm64` directly drops the root `LDFLAGS` and
  produces an unstripped binary. Use `pnpm lint`, never `pnpm check`.
- Never point a build at a Core checkout outside this repository. `core/src/`
  is the source-of-truth; `core/verify-no-external-source.sh` proves the build
  needs nothing else. Upstream is fetched for review only.
- `-trimpath` on the Android arm64 build lines is a release requirement, not a
  nicety. Without it the shipped binaries carry thousands of build-machine
  paths that reach the user-facing Logs screen.
  `core/build-android-arm64.sh` fails the build if the count is not zero.
- Regenerate `core/pocketclaw-core-v0.3.1.patch` with
  `core/regen-upstream-patch.sh` after any change under `core/src/`, or the
  divergence record goes stale.
- The onboarding service lives in its own public repository,
  `Lord1Egypt/PocketClaw-Telegram-Setup`. It is infrastructure, not part of the
  APK build; `services/README.md` explains why it is not vendored here.
- Never commit the manager bot token, and never add it to a `--dart-define`.
  The app receives only a public HTTPS base URL; the child bot token reaches it
  once, over TLS, and goes straight into Core's config.
- Telegram onboarding must keep writing the existing `channel_list.telegram`
  entry. Do not add a second Telegram runtime; both the automatic and the
  manual path converge on the same channel.
- Core binaries currently committed and DEVICE-VERIFIED (2026-08-25):
  `libpicoclaw.so` 37,224,801 `cb9b2cde...fb895818`;
  `libpicoclaw-web.so` 24,641,889 `b6b356f7...656db9ba5`. Both stripped, PIE
  `ARM aarch64`, `-trimpath`, with `PICOCLAW_DNS_SERVER` present and zero
  developer-machine paths. They supersede Milestone C's `cbe568af...5556468a`
  and `86e53457...0cb0a4cd`.
  Core hashes are not reproducible across rebuilds — `BuildTime` is stamped in
  via `-ldflags` — so compare sizes and the zero-path count, not hashes.

## Exact State

This is a standalone Git repository at `/home/lordegypt/PocketClaw-App`.
It has no inherited PicoClaw FUI history. The initial source foundation is a
selective adaptation from the FUI baseline documented in `UPSTREAM_BASELINE.md`.
It is private at `https://github.com/Lord1Egypt/PocketClaw`, with `main` and
`develop` tracking `origin`. The initial commit is `950d4a3`.

The Android service and bundled Core retain the physically verified DNS bridge:
Android active-network DNS servers are passed as `PICOCLAW_DNS_SERVER`; public
resolvers are not hardcoded. Optional Firebase/Umeng feedback defaults to
disabled and requires explicit complete configuration.

Validation completed: `flutter analyze` is clean; `flutter test` passed all
28 tests; focused Core `pkg/androiddns` and `web/backend/api` tests passed.
The Milestone A independent arm64 APK was
`build/app/outputs/apk/release/app-release.apk`, 32,619,416 bytes, SHA-256
`207a5e4623b6c6ae295a29a874c7a7d6ac9a17552e2ddb2511daa093b085fa4d`.
Its package remains `com.sipeed.picoclaw` version `0.1.3` (code `3`) by
design. Embedded Core hashes match the
pinned records. The release build reported no Firebase app ID, API key, or
project ID and removed generated Firebase resources after packaging.

The foundation physical regression was subsequently confirmed PASS, including
AI request/response, Telegram send/receive, Core restart/configuration
persistence, and no feedback log spam. Skill Hub/ClawHub was also verified:
searching `Crypto` returned 20 results with metadata, URLs, and visible install
actions. The earlier unavailable-registry symptom is resolved by the Android
DNS fix; do not rewrite Skill Hub without new source/runtime evidence.

Milestone B changes product-visible naming to PocketClaw and migrates only the
Android/Dart integration identity to `com.lord1egypt.pocketclaw`. Core names,
binary names, `PICOCLAW_DNS_SERVER`, protocol identifiers, and the compatible
external `Downloads/picoclaw` workspace remain deliberately intact. The second
original PocketClaw mark is the selected primary visual direction; launcher,
adaptive, and monochrome variants derive from it. Do not replace it with a
cartoon lobster/mascot.

The Milestone B physical test then found a black startup screen and noticeable
device slowdown. The root cause is the branded Android launch drawable: both
`launch_background.xml` variants used `<item android:color=...>` in a
`layer-list`; `LayerDrawableItem` requires `android:drawable` (or a child
drawable), so Android cannot inflate the launch window before Flutter's first
frame. The replacement changes only those two items to reference
`@color/pocketclaw_splash_background` and adds a regression test. The 1254 px
RGBA splash mark is 654,845 bytes on disk and about 6.0 MiB decoded, which is
within a reasonable startup budget and was not the root cause.

The fixed validation results are: `flutter analyze` clean; all 29 Flutter
tests pass; focused Core `pkg/androiddns` and `web/backend/api` tests pass.
Debug and release APKs built successfully. The replacement release is
`build/app/outputs/flutter-apk/app-release.apk`, 33,308,653 bytes, SHA-256
`45be7269af920df4a36eb4eb37171770bbcfa242ed7c071da28874d9c27ebe9e`.
`aapt2` confirms `com.lord1egypt.pocketclaw`, label PocketClaw, launchable
`com.lord1egypt.pocketclaw.MainActivity`, and a compiled splash layer with a
drawable-backed first item. Embedded Core hashes remain
`3b849072a7c2858b0d2c0db5cbcfa42b542353e834f4c473399eda571ab16f3d` and
`252b38c64cbc4dc52277c206ca1b069cc7c3bb97b8a9c276e23f8edc3aaf95e3`.
The service/Core code was unchanged: a fresh Flutter launch does not issue a
Core start request, and the native service has its bounded three-restart guard.
No connected ADB device was available to measure the reported slowdown or
capture final device logs, so that symptom must be confirmed during retest.

## Next Exact Steps

1. Revoke the exposed manager token in BotFather.
2. Deploy `Lord1Egypt/PocketClaw-Telegram-Setup` to Vercel with the variables
   above, including a Redis store.
3. Register the webhook and verify the manager from the setup page. The
   `can_manage_bots` assertion happens here.
4. Rebuild the app with `POCKETCLAW_ONBOARDING_BASE_URL` and run the device
   checklist at the end of `PROJECT_STATE.md`.
5. On PASS: merge `feature/telegram-managed-onboarding` into `develop` with
   `--no-ff`, tag the closure point, and record the result. `main` needs
   separate instruction.
6. On FAIL, the likely causes in order: the app was built without
   `POCKETCLAW_ONBOARDING_BASE_URL`; the webhook was never registered, or was
   registered against a preview deployment URL rather than the production one
   (set `PUBLIC_BASE_URL` if so); or the suggested username was edited on
   Telegram's confirmation screen, which by design fails closed and expires the
   pairing.
7. Two open items unrelated to the blocker: localizing the onboarding strings,
   and confirming a `claude-*` model on OpenCode. Both are in `TASKS.md`.
8. Do not start another milestone.

## CODEX SOL HANDOFF — PRE-RELEASE FIX

Continue on `fix/user-facing-log-privacy`. The exact pre-Codex state is
recoverable from pushed branch `checkpoint/pre-codex-sol-prerelease-fix` or
annotated tag `pre-codex-sol-prerelease-fix-20260826`, both at
`e5b88ff1a4c8f76e321c07af97eff5ca23d59d78`. Historical Milestone D remains
`phase2-milestone-d` / `8f861bca1c82b43b306e95b14e277269260bbab0`.

Automated work is complete and one candidate exists:

- APK: `build/app/outputs/apk/release/app-release.apk`
- Size: 34,239,649 bytes
- SHA-256: `f663d25a2fffb0ce969ad4a9ce3405c1e563b6263c7af37e90768eef471c621d`
- Core: `87653601023974156be1b1a255387ec3c93a0c154254a7b8d9e1012ac2e926e5`
  and `a8328f1d66932ba8da544060905965278898744c04c2ece5a993717e361400c5`.
- Guard, metadata, endpoint, `-trimpath`, and secret scans: PASS.
- Flutter analyze/test (94), frontend Vitest (36)/tsc/lint, and required Go
  suites: PASS.

Native Telegram is now only a shortcut to Core's `/channels/telegram` page;
Core owns credential persistence through a narrow authenticated loopback write;
Telegram outbound HTTP is deadline-bounded; request/placeholder state is
correlated and same-session Telegram requests are independent FIFO lifecycles;
edit/send failures are no longer swallowed; seven fresh skills are repaired
non-destructively; Android logs are sanitized once for display/export and use a
plain banner/neutral PID warning. Full root causes, file groups, tests, and exact
physical symptoms are in the matching section of `PROJECT_STATE.md`.

Physical testing is still **PENDING**, so release is **BLOCKED**. Do not merge,
touch `main`, move `phase2-milestone-d`, or create a GitHub release. Test a lone
`تسلم` with no wake-up message, empty-response recovery, sequential/close
messages, foreground/background/locked screen, placeholder/typing/streaming
variants, neutral native Telegram navigation, reconnect preservation, skills
repair/import, Unicode logs/export, provider, Skill Hub, startup, and speed.

Deferred only: Background & Battery UX; Runtime/Statistics tab; controlled Dart
source-URI hardening. Manager-token rotation remains pending unless explicitly
confirmed externally; never retrieve or print the token.

## WEB CONSOLE LOG PARITY CANDIDATE — 2026-08-27

Continue on `fix/user-facing-log-privacy`. The previous UTF-8 export,
successful log/status poll suppression, exactly-once drain, basename callers,
and neutral Telegram card all remain passing and must not be reopened.

The current automated candidate is
`3e138b4a53a0389b76dbef045649d906fe2db785cd2af606826f7a0f87170adc`
(34,241,381 bytes). Core hashes are
`c9c348e9c637a7552e810396bfba5460ad4a3f65ab506d933ac5d05ea68d236e`
and `c891ca033ededb6b8941d997fbc8e0a8d69eb18f44a6ed2176f878f65d34840a`.
The three-library ARM64 guard passed, Core paths are clean, the onboarding
endpoint is present once, and package/version remain 0.1.3 (3).

Root causes and fixes: the gateway's no-color banner was still Unicode block
art; captured launches now force `--no-color` and render one `PocketClaw` line.
The Web log ring stored raw output and React only interpreted SGR; the ring now
stores the shared fixture-backed plain-text contract and the real Logs page has
an idempotent parity guard instead of an ANSI renderer. Startup no longer logs
the executable/library path. Routine successful `/pico/ws` events are omitted;
failures remain under `/internal realtime connection`. No compatibility
identifier was renamed.

Automated validation is complete: Flutter analyze plus 99 tests, frontend 37
tests/tsc/lint, and tagged Go logger/gateway/API/middleware/Core CLI suites all
pass. Physical validation remains mandatory and release remains blocked. Next:
install only this APK, open Core Web Console Logs for several minutes, export a
DEBUG log, and compare all three surfaces for intact `53.616µs`, Arabic, emoji,
zero controls/boxes/ANSI, zero routine `/pico/ws`, zero internal library path or
unintended upstream branding, and a visible neutral genuine-error line. Do not
merge, release, touch `main`, move `phase2-milestone-d`, or start a milestone.

## FINAL LOG BRAND MICRO-FIX CANDIDATE — 2026-08-27

Physical validation of `3611ca1` passed the Web terminal cleanup, banner,
UTF-8/`µs`, library-path visibility, and skills checks (8/8, 17 tools). The sole
remaining line was `Channels enabled: [telegram pico]`.

Internal meaning: `pico` is Core's singleton authenticated Web Console
WebSocket/media transport and compatibility/config ID. It remains unchanged
everywhere internal. Only the startup/reload summary copies the enabled names
and maps exact `config.ChannelPico` to display label `pocketclaw`.

New APK: `aab3c565bd6bec2eb714443756e16be5d2ebe8d4496d94e64a6b8ecac25b3582`
(34,241,857 bytes). Core hashes: `10446d81...71b45f1` and
`683463df...e5f5ce4`. Tagged relevant Go suites, Core provenance/build,
zero-path checks, endpoint check, packaged hashes, and permanent guard pass.
Physical action: restart Core and confirm `[telegram pocketclaw]` while all
previously passed log and 8/8 skills behavior remains intact. No merge/release.

## FINAL USER-VISIBLE CALLER BRAND CANDIDATE — 2026-08-27

The last physical leak was structured logger header `INF pico
pico.go:<line>`. Internal package/file/channel/routes/config remain unchanged.
At the shared Go/Dart/React normalization contract, only exact logger component
`pico` displays as `realtime` and exact caller basename `pico.go` as
`realtime.go`, preserving the numeric line. Substrings and `pico_client` are
unchanged.

New APK: `1eeca7c993d657f0e6e763d94691a054e584592d0989445454144ac15ad089f7`
(34,242,865 bytes). Core hashes: `a379ae45...07eafb9` and
`584dd9ae...1fa4a01`. Flutter analyze/99 tests, frontend 37/tsc/lint, relevant
tagged Go suites, provenance, zero paths, endpoint, packaged hashes, and guard
pass. Physical device remains the final gate; no merge or release.

## WEB CONSOLE LOG VIEWPORT STABILITY CANDIDATE — 2026-08-27

Continue on `fix/user-facing-log-privacy`. The previous candidate's complete
log sanitization/branding behavior passed physically. This micro-pass changes
only the Core React Logs viewport.

Root cause: bottom correction ran after paint in `useEffect`, and long lines
were hard-wrapped from a `ResizeObserver` measurement of content whose height
changes on every append. That could expose an old-offset frame and then rewrite
all long-row layout. Index keys also lacked the API's real event identity.

Fix: conditional pre-paint `useLayoutEffect` bottom following, no writes while
scrolled up, stable `run_id:absolute_offset` keys, memoized rows, and native CSS
wrapping of the unchanged plain-text entry. No native queue, logger sanitizer,
poll filter, Telegram, Skills, Provider, or internal route logic changed.

New APK: `be5d7cbb18c0378dad0a3d53d2d3a4e71001411121fa056f7a6606645070fc96`
(34,239,873 bytes). Core hashes: `715cd790...6143cf1` and
`e3930ae2...f5f5da14`. Frontend 41/41, tsc/lint, relevant Go, Native/Export
regression, provenance, zero paths, packaged hashes, and guard pass. Physical
device is the final gate; no merge/release/main change.

## TELEGRAM WEB LOG REDACTION CANDIDATE — 2026-08-27

Physical clarification: token fragments appeared only in Core Web Console
Logs; Native Android Logs remained clean. This is case A. Telego's full Bot API
URL was partially masked in PocketClaw's logger before stdout and before Web
`LogBuffer` storage. The full token was not persisted through this path, but
the old bot-ID/first-four/last-four fragments were stored and rendered.

The pre-stdout logger now retains no credential fragment. Web pre-storage
normalization turns Bot API URLs into `Telegram API call: <operation>` and
keeps method/status/error/timeout/latency; React has an idempotent legacy guard.
Native/Export sources and all Telegram credentials/lifecycle code are
unchanged. No rotation occurred.

New APK: `8257e9f091039f2c29332b8f14e2d397d36bbb735593596a4caa0e567c7050fe`
(34,240,641 bytes). Core hashes: `0e914550...8b555e9` and
`7d7b254b...d9898c7`. Relevant Go suites, frontend 42/42/tsc/lint, unchanged
Native log regression, 115-file provenance, zero paths, packaged hashes, and
guard pass. Physical Web Console validation remains mandatory; no merge,
release, main change, or credential mutation.

## FINAL LEGACY BRAND VISIBILITY SWEEP CANDIDATE — 2026-08-27

The physically observed `.picoclaw.pid`, structured `channel=pico` /
`type=pico`, `/pico/` registration path, and Pico Protocol lifecycle wording
were traced to gateway compatibility details entering the Web ring. The shared
pre-storage normalizer now gives only these exact classified identities semantic
PocketClaw/realtime/internal/gateway display wording; React keeps the same
idempotent guard for raw or historical lines. The security warning remains
fully visible and only its exact channel field changes.

All runtime compatibility identities and files remain unchanged. Substring
negative cases remain byte-for-byte unchanged. A representative full startup
stored in `LogBuffer` and the real Logs DOM have zero unintended legacy brand
occurrences. New APK: `309f6d7a...a5f3030`. Core hashes:
`49f89ae2...be656f` and `98f3fa9d...08bae`. Frontend 44/44/tsc/lint,
relevant tagged Go including Skills/Pico/Telegram, Flutter analyze/99 tests,
115-file provenance, zero paths, and guard pass. Physical device remains the
final gate; no merge/release/main change.

## TELEGRAM DEBUG FINAL CLEANUP CANDIDATE — 2026-08-27

Telego emits correct `Err: [<nil>]`; pre-stdout redaction preserved it. The Web
backend's orphaned-CSI fallback then misclassified `[<n` as a control suffix,
so `LogBuffer` stored `il>]`. The fallback is now restricted to numeric CSI
remnants. Known successful Telego nil fields display as `Err: none`, while
ordinary angle-bracket text is preserved and React continues safe text-node
rendering.

Exact DEBUG `getUpdates` request lines and successful empty responses are now
discarded before stdout, with idempotent shared-boundary guards. Failures,
non-empty updates, API errors, send/edit operations, lifecycle diagnostics, and
credential redaction remain. Four repeated empty polls produce zero stored
events. No Telegram lifecycle/onboarding/delivery, Skills, Provider, Pico,
viewport, or native queue logic changed.

New APK: `2c00720a...2da27c` (34,243,585 bytes). Core hashes:
`7c1d3918...38ef1f` and `1611b6e1...d09256`. Frontend 45/45/tsc/lint,
relevant tagged Go including Telegram/Skills, Flutter analyze/101 tests,
115-file provenance, zero paths, and build guard pass. Physical device remains
the final gate; no merge/release/main change.

## AGENT DEBUG PRIVACY + TELEGRAM PAYLOAD CANDIDATE — 2026-08-28

Continue on `fix/user-facing-log-privacy`. Physical evidence proved the actual
fresh default model prompt was legacy-branded and that normal Web DEBUG exposed
Agent prompt/message/tool/reasoning/session data plus complete non-empty Telego
result payloads. The default prompt now identifies as PocketClaw; custom prompt
overlays and all internal compatibility values remain unchanged.

Explicit prompt/full-request/raw-reasoning and tool-argument log paths are
removed. A pre-writer exact-field copy redacts internal/session identities and
raw-content fields without mutating runtime maps. Telego request/response data
is normalized before stdout/backend storage to operation/status/count/type
metadata; raw Telegram IDs, profiles and message bodies no longer enter normal
history. Backend, React and Dart guards cover legacy/raw input.

New APK: `46ca983a...2b908e` (34,251,141 bytes). Core hashes:
`8ed15601...9be24a` and `21001004...5fc1d7`. Tagged relevant Go suites,
frontend 46/46/tsc/lint, Flutter analyze/101, 124-file provenance, zero paths,
live endpoint, packaged hashes, and permanent guard pass. Physical device is
the final gate; no merge/release/main change.

## v0.2.0-rc1 freeze record — 2026-08-29

The candidate is merged to `develop`, tagged `v0.2.0-rc1`, and published as a
GitHub **pre-release**. `main` was not touched and `phase2-milestone-d` was not
moved. Nothing was published to Google Play.

Explicitly deferred and NOT in this candidate: Flutter/Dart obfuscation,
R8/ProGuard, symbol stripping, anti-reverse-engineering protection, and
generated Dart source URI cleanup. Those belong to the final production release
and are tracked in `TASKS.md` alongside the Statistics/Runtime page, background
and battery settings, service/gateway auto-start, the managed tool runtime, and
Android Keystore migration for provider secrets.

The candidate is validated on one ARM64 Android device. Do not describe it as
universally compatible.
