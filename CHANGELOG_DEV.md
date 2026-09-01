# Development Changelog

## 2026-09-01 — WhatsApp Self-Chat, and the Chat attachment button that did nothing

Branch `feature/whatsapp-self-chat`. **Built and gated; physical acceptance
pending.** Not merged, `main` untouched, no tags moved.

### One WhatsApp entry, and one value

Channels showed "WhatsApp" and "WhatsApp Native". Both opened onto a bridge URL
and a session store path — configuration a phone user cannot supply for the
device the app is running on. Both are gone from the console. In their place is
WhatsApp Self-Chat, which stores the user's own number in canonical
international form and nothing else.

`Normalize` accepts `+20 101 234 5678`, `0020-101-234-5678` and
`201012345678`, because each names its country. It refuses `0101 234 5678`: the
leading trunk zero belongs to a national plan, and guessing the country from a
locale would quietly point the feature at a stranger's chat. The same rule is
implemented in Go for what Core reads off disk and in TypeScript for what the
console accepts, and both are tested against the same table.

The bridge and native transports were **not** deleted. `pkg/gateway` blank-imports
them, `Manager.channelReadiness` splits them on `use_native`, `config` types
them, `pkg/migrate/sources/openclaw` reads `channels.whatsapp.bridge_url`, and
six test files exercise them. Removing them would break installs that already
work. They are only no longer offered.

### Opened is not sent

The `whatsapp_self_chat` tool takes one argument, requires a configured number,
caps the body at 4096 characters, and returns `OPENED` with an explicit "it has
NOT been sent". Its description says the same thing, because the description is
all the model reads before deciding to call it. A test asserts the result never
claims delivery.

Neither the number nor the message body reaches the logs — only
`message_chars` and a status — and that is checked by capturing the log file
during a real call rather than by reading the call site.

### Core cannot start an Activity

So it asks. The tool writes `req-<id>.json` into an app-private directory named
by `POCKETCLAW_ANDROID_HOST_OUTBOX` and waits for `res-<id>.json`; a
`FileObserver` in the foreground service serves it and answers every request it
consumes, including the ones it cannot serve, so the tool reports a real reason
instead of a timeout. A request that is abandoned is deleted, because one left
behind would open WhatsApp out of the blue the next time the host started.

No socket, no token: the directory is reachable only by this UID.

### The attachment button

It was never wired to anything on the host side. The console's composer
dispatches a click at a hidden `<input type="file">`, and the Android WebView
asks the host through `WebChromeClient.onShowFileChooser`.
`webview_android.dart` built a plain `WebViewController` and never called
`AndroidWebViewController.setOnShowFileSelector`, so the plugin's
`onShowFileChooser` returned `false` and Android showed nothing — no error, no
picker, no log line. Registering the selector is the entire fix. The media
pipeline that turns the chosen file into a chat attachment was already correct
and is untouched.

The picker is the system photo picker on Android 13+ and `ACTION_OPEN_DOCUMENT`
below it. Neither needs a storage permission, so none was added.

`MainActivity.onResume` had a related defect: it jumped to the all-files-access
settings screen on every resume when `MANAGE_EXTERNAL_STORAGE` was missing,
despite its own comment saying it should prompt once. That includes the resume
returning from the picker, which would have looked like the app ejecting the
user mid-attachment. It now prompts once per launch.

### Also fixed

`useSidebarChannels` read `appConfig.channels` while `GET /api/config`
serialises that field under its `channel_list` JSON tag, so the enabled map was
always empty and the "configured channels first" ordering never took effect.


## 2026-09-01 — Android DNS for the bundled gh

Branch `feature/secure-github-auth`. **Physically validated and merged to
`develop`.** The proven cause from the diagnostic commit is fixed, verified on
the device over adb, and then through the full twelve-step UI flow with a real
token against a private repository.

### The fix

The gh payload now carries the same resolver Core and the launcher use.
`runtime/build-gh-android-arm64.sh` copies `core/src/pkg/androiddns/resolver.go`
into the gh tree and adds a three-line `init()` in `cmd/gh` that calls it. The
file is copied rather than reimplemented, so there is one resolver in the
repository; it imports nothing outside the standard library, and the build fails
if that stops being true or if the shim does not end up linked in.

`PICOCLAW_DNS_SERVER` is delivered on the **gh profile**, not through
`inheritedEnvKeys`. gh is the only bundled tool that resolves names in Go — curl,
git and its transport helper go through bionic and Android's own resolver — so
inheriting it everywhere would add reach without adding capability. It is not a
credential, but the narrow path costs nothing and stays honest about who needs
it. No agent-controlled environment field was added; the runtime tool still takes
no environment at all.

Off Android, or when the host supplies nothing, the shim installs no resolver and
gh behaves exactly as upstream does.

### Proved on the device, before any UI test

One binary, one variable, two outcomes:

```
patched gh, no PICOCLAW_DNS_SERVER
  dial tcp: lookup api.github.com on [::1]:53: connection refused

patched gh, PICOCLAW_DNS_SERVER set
  * Request took 450.852692ms
  {"message": "Bad credentials", "status": "401"}
```

A dummy credential now produces an HTTP 401 from GitHub instead of a DNS
failure. The request reaches GitHub; the credential is what it rejects.

### A guard that was testing the wrong property

`install_payload` required LOAD segments aligned to exactly `0x4000` and rejected
the rebuilt gh at `0x10000`. The requirement is 16 KB pages, and a 64 KB-aligned
segment satisfies them; the NDK links C payloads at `0x4000` while Go links arm64
at `0x10000`, which is why Core, the launcher and the shipped gh are all
`0x10000` and have run on device since Phase 1. The check now requires a multiple
of 16 KB rather than one exact value, which is the property Android cares about.

Rebuilding gh repinned its checksum and moved the catalog to `2.3.0`; Core was
rebuilt, and the catalog, source-freshness and payload guards all pass.

## 2026-09-01 — GitHub connectivity diagnostic, and the proven cause

Branch `feature/secure-github-auth`. **Not merged.** The credential path is
sound; gh cannot reach the network on Android, and this build says so honestly
instead of blaming the token.

### Root cause, proven on the device

Reproduced outside the app, over adb, on SM-A165F:

```
$ gh api user            # GH_DEBUG=1, dummy token, app-like environment
* Request to https://api.github.com/user
* dial tcp: lookup api.github.com on [::1]:53:
    read udp [::1]:36500->[::1]:53: read: connection refused
error connecting to api.github.com
```

Android has no `/etc/resolv.conf`. Go's resolver finds no nameservers and falls
back to `[::1]:53`, where nothing is listening, so the request never leaves the
device. `GODEBUG=netdns=2` confirms it: `using the Go DNS resolver`,
`hostLookupOrder(api.github.com) = files,dns`. `netdns=cgo` changes nothing —
the payload is built `CGO_ENABLED=0`, so there is no cgo resolver to select.

The bundled curl reaches `https://api.github.com/` with HTTP 200 in the same
environment, because it resolves through bionic and Android's netd.

This is not a CA problem: both `/system/etc/security/cacerts` (143) and
`/apex/com.android.conscrypt/cacerts` (145) are populated, and the failure is
identical with `SSL_CERT_DIR` set to either, or unset.

It is the same problem `pkg/androiddns` already solves for Core and the
launcher, which receive `PICOCLAW_DNS_SERVER` from the host and install a
resolver from it. gh is a separate Go binary that never got that treatment, and
`PICOCLAW_DNS_SERVER` is not among the runtime's inherited environment keys, so
it could not have used it anyway.

### What changed here

The message was wrong in a way that mattered. "GitHub rejected this token:
error connecting to api.github.com" tells the user to replace a credential that
was never checked. Failures are now classified from gh's stderr — with
`GH_DEBUG=1`, which adds the underlying transport error and prints no request
headers — into auth, connectivity, timeout, unavailable and other, each with its
own message and HTTP status, so the host can tell "your credential is wrong"
from "PocketClaw could not ask". The sanitized detail goes to the Debug Logs.

The candidate is scrubbed from every diagnostic before it can reach a reply or a
log, and no environment is dumped with it.

## 2026-08-31 — Secure GitHub authentication (Phase 1)

Branch `feature/secure-github-auth` from `develop` at `08c457e`. **Not merged**,
physical acceptance outstanding.

### What was already there

The audit found the injection half of this already built and correct.
`applyGHProfile` sets `GH_TOKEN` and `applyGitCredentials` sets an
`http.https://github.com/.extraheader` Authorization header through
`GIT_CONFIG_KEY_0`/`VALUE_0`, both marked secret, neither in argv, and no
credential-bearing URL anywhere. What was missing was everything below it: the
credential had no storage, no UI, and no way to be configured except an
environment variable.

`Manager.Execute` is reused unchanged. No second execution path was added.

### What changed

**Storage.** `GitHubCredentialStore` encrypts the token with AES-256-GCM under a
key generated inside the Android Keystore that cannot be exported from it.
`setRandomizedEncryptionRequired(true)` makes the provider draw the nonce from
the platform CSPRNG and refuse a caller-chosen one, so nonce reuse is impossible
rather than merely unlikely. Only ciphertext reaches storage, written whole so a
half-written blob cannot silently disconnect the user. Any decryption failure
deletes the material and reports "not connected": a credential that will not
authenticate is not one to carry forward.

**Core reads the credential from its environment and nowhere else.** The
previous `credentials/github_token` file fallback is gone. Nothing wrote it, and
a credential this process can read off disk is one that survives on disk, which
is what the encrypted store exists to prevent. The Android service decrypts at
Core launch and passes it in, so connecting or disconnecting takes effect on the
next start.

**Validation.** A candidate is checked before it is stored, by Core, over the
existing loopback Android bridge: `gh api user` through `Manager.Execute` with
the candidate as a one-shot `GH_TOKEN` override. gh is never asked to log in, so
it writes no config of its own. Core scrubs the candidate out of anything gh
said before replying, because a tool's error message may quote what it was
given. `gh auth setup-git` is deliberately not used: it makes gh a persistent
credential helper in `~/.gitconfig`, which is credential persistence outside
PocketClaw.

**UI.** A GitHub card in Settings showing connected state and account name, with
Connect, Test connection and Disconnect. There is no reveal control and no field
that could redisplay a stored value, because nothing above the Android service
has a copy to show.

**Backup.** The manifest declared neither `allowBackup` nor
`dataExtractionRules`, so app-private files were backed up by default. Both are
now declared and the credential directory is excluded from cloud backup and from
device transfer. The Keystore key does not travel, so a restored blob would fail
closed — excluding it keeps that from being the user's problem to discover.

### Pre-physical corrections

**Validate before save was already the flow, and is now structural.**
`GitHubCredentialStore.connect` requires a non-blank login, and the only source
of a login is a successful `gh api user`, so storing an unvalidated token is not
expressible. Tests assert that a rejected candidate stores nothing and triggers
no restart.

**Connect and disconnect apply themselves.** `ServiceManager.applyCredential
Change` reuses the stop/start the config screen already performs; it adds no
lifecycle machinery. A service that is mid-start is never interrupted: the
change is queued and applied from the existing status poll once the service
settles, and the card reports "saved, will apply automatically" rather than
claiming the credential is live. After the restart the old credential is gone
from Core, because the new process is built with a fresh environment.

**Decryption failures are classified rather than lumped together.** Destroying a
credential is irreversible, so it happens only on positive evidence: a GCM tag
that does not verify, ciphertext that cannot be a valid block sequence, a
malformed blob, or a key the platform reports permanently invalidated.
Everything else — a busy keystore, a provider that failed to load, an error this
code has not seen — preserves the ciphertext and reports the credential
unavailable, which the UI shows as a temporary state rather than a
disconnection. `GitHubCredentialStore.classify` is a pure function of the
failure and is covered by JVM unit tests, which is why the Android module now
has a `test` source set and a JUnit dependency.

### Known limitation

Test connection exercises the running Core. Immediately after connecting, the
restart it triggers must finish before the test reports authenticated; while the
service is still coming up the card says so rather than claiming success.

## 2026-08-31 — Python Lite Phase C: Android stdio root cause and the bootstrap

Branch `feature/python-lite-agent-tool`. **Physically validated and merged to
`develop`.** All four acceptance tests passed with one tool call each and no
retries: `stdout_bytes=18` for the printed line, `stderr_bytes=96` for a real
traceback ending `ValueError: TEST-ERROR`, `stdout_bytes=16` for `مرحبا 🐍`, and
`timed_out=true` at 2 s. The same calls previously reported zero bytes.

### Root cause

Android's CPython does not connect `sys.stdout` and `sys.stderr` to file
descriptors 1 and 2. It replaces both with `TextLogStream`, which writes to the
Android system log, because an app has no console. The managed runtime captures
the descriptors, so every managed run wrote its output somewhere the caller
could not read.

That explains each symptom exactly. `print()` succeeded and exited 0 with
nothing captured. An uncaught exception still exited 1 while its traceback went
to logcat. `os.write(1, ...)` worked, because it bypasses `sys.stdout`
altogether. Device evidence: `stdout = TextLogStream, fd=1, closed=false`, and
rebinding the streams by hand produced `stdout_bytes=33`, `stderr_bytes=97` and
the real `ValueError: TEST-ERROR`.

CPython, the runtime's pipes, the capture layer and the formatter were all
working. Nothing in PocketClaw was broken; the interpreter was pointed
elsewhere.

### The fix

`pocketclaw_bootstrap` is a small module shipped **inside** the payload's
appended standard library, so the checksum the registry verifies before
executing anything covers it. The Python tool now invokes
`python <catalog default_args> -m pocketclaw_bootstrap <caller args>`. The
bootstrap rebinds `sys.stdout` and `sys.stderr` to unbuffered UTF-8 streams over
`os.dup(1)` and `os.dup(2)` — duplicated so interpreter shutdown cannot close
the descriptors the runtime is reading — then reads the program from standard
input exactly as `python -` does and executes it.

The caller's source still travels on stdin and nowhere else. It is not moved to
`-c`, not put in argv, not written to the environment, and not logged. The
bootstrap compiles it under `<stdin>`, runs it as `__main__`, sets
`sys.argv` to `["-", ...caller args]`, and prints the caller's traceback with
its own frames removed, so line numbers still refer to the submitted code.

CPython is not patched. Upstream's Android logging behaviour is untouched, since
other embeddings may rely on it.

### Guards

The payload is checksum-pinned, so adding the module repinned it: catalog
`2.1.0` → `2.2.0`, python sha256 repinned, Core rebuilt. Three guards now cover
the payload — the appended-stdlib guard, a new entry-point guard in the Gradle
release and in the Go tests, and the existing catalog and source-freshness
guards.

The Python result also names the failure it cannot otherwise distinguish: a
non-zero exit with zero bytes on both streams is reported as an install fault
rather than a program failure, because that is the shape this bug had, and a
payload shipped without the entry point would reproduce it silently.

## 2026-08-31 — Python Lite Phase C: physical stdout/stderr failure, boundary instrumented

Branch `feature/python-lite-agent-tool`. **Not merged. Not fixed.** The physical
recheck failed and the root cause is not yet proven; this commit makes the next
physical run prove it, and closes the test seam that let the failure through.

The device reported `print("PYTHON-FINAL-PASS")` as exit 0 with empty stdout, and
`raise ValueError("TEST-ERROR")` as exit 1 with empty stderr. The timeout still
behaved correctly.

### What the empty string proves

`ExecResult.Stdout == ""` has exactly one possible cause. A bounded buffer that
saw bytes and kept none returns a truncation marker, not an empty string, so an
empty stream can never mean "output arrived and was dropped in the runtime". It
means the capture layer received nothing. `TestEmptyCapturedOutputMeansNothing
WasWritten` pins that down.

That eliminates the formatter and the result serialisation as causes: the
`(empty)` text the device printed can only be reached from an empty capture. It
also eliminates the hardening commit, which changed no capture, environment or
catalog code at all — `git diff c4c0bf2..b42f813` touches only the formatter, one
log field and the new fingerprint package.

### The seam the tests missed

The Phase C end-to-end test stopped at `PythonTool.Execute` and read `ForLLM` off
the result. Production goes through `ToolRegistry.ExecuteWithContext`, which
normalises the result before the agent sees it. A gap between the tool and the
model was invisible from inside the tool.

There is now a production-path test: the test binary re-executes itself as the
interpreter, staged as a checksum-verified bundled payload with the catalog's
real `default_args`, so a real child writes to real pipes and the assertion is on
`ContentForLLM()` — the exact string the pipeline puts into the conversation.
stdout, stderr, the exit code, UTF-8 byte-for-byte, and the source staying out of
the ordinary tool log are all covered there.

### The instrument

`ExecResult` now carries `StdoutBytes` and `StderrBytes` — what the capture layer
actually received, before bounding. The Python tool prints them, and the runtime
logs `stderr_bytes` next to the existing `bytes_out` at INFO rather than only at
DEBUG. One physical call now says which side of the pipe lost the bytes:
`stdout_bytes=18` beside `stdout: (empty)` would mean the loss is downstream of
the capture; `stdout_bytes=0` means the interpreter wrote nothing the runtime
could see.

## 2026-08-31 — Python Lite Phase C: raw stderr and a Core source fingerprint

Branch `feature/python-lite-agent-tool`. Automated gates green; **not merged**,
the physical stderr recheck is outstanding.

Phase C passed physically on its core behaviour — statistics, JSON, Unicode, an
uncaught exception and a 2 s timeout — but a controlled single-call test found
the model unable to report the traceback from `raise ValueError("TEST-ERROR")`.
Two things came out of chasing that.

**The Python result now names every field it has.** `formatPythonResult` writes
`exit_code`, `timed_out`, `cancelled`, `stdout_truncated` and
`stderr_truncated`, then both streams under their own headings — including an
empty one, printed as `(empty)`. It previously omitted a stream that had no
content, which left "the interpreter printed nothing" and "the result dropped
it" looking identical to the model. Nothing is reconstructed: the text is the
runtime's own bounded, redacted capture, and a test fails if a traceback ever
appears for a run that produced none.

The end-to-end proof runs a real interpreter through `Manager.Execute` rather
than handing the formatter a hand-written `ExecResult` — a hand-written one
cannot fail the way this failed. The tests cover the traceback reaching the
agent, the non-zero exit code, the bound on stderr, truncation being reported,
and the source staying out of the log.

**Core staleness now covers Go-only changes.** `pkg/coresource` hashes the Core's
build inputs — non-test Go source under `cmd/` and `pkg/`, the embedded catalog,
the embedded `workspace/`, `go.mod`, `go.sum` and the Makefile — into one
content-addressed fingerprint. `core/build-android-arm64.sh` computes it and
stamps it into the binary with `-X`; the test gate recomputes it from the working
tree and fails if the staged Core does not carry it.

`TestStagedCoreEmbedsTheCurrentCatalog` stays: the two guards answer different
questions. Does this Core know the current catalog, and was this Core built from
the current code? Phase C itself is the case only the second one catches —
`python_tool.go` changed, `manifest.json` did not, and the catalog guard had
nothing to notice.

The fingerprint is content-addressed on purpose. No mtime, no build timestamp,
no absolute path, no build id: a value that changes on its own cannot say
anything about staleness, and a guard that cries wolf stops being read. The first
draft hashed every `*.json` under `pkg/`, which `pkg/cron`'s tests write into
during a run, so the gate failed against a Core that was in fact current.
Embedded assets are named one by one now, and a test reads the `//go:embed`
directives out of the Core source to fail if the list falls behind.

## 2026-08-31 — Python Lite Phase C: the Agent-facing Python tool

Branch `feature/python-lite-agent-tool`. Automated gates green; **not merged**,
physical validation outstanding.

The model gets a `python` tool taking `code`, optional `args` and optional
`timeout_ms`. It adds no execution machinery: the tool builds an ordinary
`ExecRequest` and `Manager.Execute` provides resolution, checksum verification,
the environment profile, catalog `default_args`, the timeout ceiling,
cancellation, process-group termination, output bounds, events and redaction. It
shares the Managed Runtime's manager rather than building a second one.

Source travels on stdin as `python -`, never in argv, and a regression test fails
if that ever changes to `-c`. argv is capped near 128 KB, readable from
`/proc/<pid>/cmdline` and recorded in argument diagnostics; stdin is accounted
only as `bytes_in`.

Tracebacks keep `<stdin>`. `<pocketclaw>` would need a wrapper that reads stdin
and recompiles the source — an interpreter trick around the exact path carrying
user code, bought for cosmetics. Not worth it in v1.

The description steers the model rather than inviting it to reach for Python by
default: jq for simple JSON, rg for search, sqlite3 for a single query, curl for
HTTP; Python for arithmetic, statistics, multi-step logic, custom parsing and
work that would otherwise cost several round-trips. It states that Python is not
a sandbox and that the boundary is the application UID. Tests fail if the
steering is removed or the wording starts claiming containment it does not have.

One shared improvement came out of this: truncated output was recorded in the
event log but never shown to the model, so a bounded result looked complete.
`formatExecResult` now says when stdout or stderr was cut, which the runtime tool
benefits from too.

`python` is enabled by default and switchable independently of `runtime`,
because it runs arbitrary code as the application.

## 2026-08-31 — Python Lite Phase B: Runtime integration (PHYSICAL PASS)

Branch `feature/python-lite-runtime`, commits `bfe47e6`, `c376837`, `c60b15f`,
`bfc2074`. **Physically validated inside the installed application** on
SM-A165F / Android 16 / API 36, then merged to `develop`. Not released, `main`
untouched, no tags moved. Phase A was a physical PASS in its own right.

Python is a bundled Managed Runtime tool reached through the generic execution
path. **There is no Agent-facing Python tool** — that is Phase C.

| Check | Result |
|---|---|
| In-app catalog 2.1.0, 56 tools, 53 available on device | **PASS** |
| `python` present and resolvable in the installed app | **PASS** |
| `runtime {tool: python, args: ["--version"]}` → `Python 3.14.7` | **PASS** |
| `PYTHON-PASS`, `SUM=5`, `مرحبا 🐍`, `SQLITE-PASS` through the Runtime | **PASS** |
| No shell required for the acceptance test | **PASS** |
| CPython 3.14.7, NDK 28.2.13676358, API 24, arm64-v8a | **PASS** |
| bzip2 1.0.8 and XZ 5.4.7 built from pinned source | **PASS** |
| SQLite 3.50.4 from PocketClaw's pinned amalgamation | **PASS** |
| Dependency provenance — no third-party prebuilt binaries | **PASS** |
| Build-path leakage | **NONE** |
| Upgrade install with no Clear Data, user data preserved | **PASS** |

The payload is one self-contained PIE ELF: CPython with every extension module
linked in statically and the standard library appended as a `.pyc` zip.
`lib-dynload` is empty. Dependencies are Android platform libraries only.

### Two packaging defects found and permanently guarded

**Gradle stripped the appended standard library.** The first APK packaged the
payload at 9,041,016 bytes against 11,509,517 on disk — the native-library strip
had removed the appended zip exactly, and the interpreter would have shipped
unable to import anything. `keepDebugSymbols` already existed for this reason
and Python was missing from it. The build guard now also verifies the packaged
payload still ends in a zip end-of-central-directory record, because a presence
check cannot see a payload that was mangled rather than dropped.

**A stale Core shipped beside the new payload.** The catalog is `//go:embed`-ed
into `libpicoclaw.so`, which is a committed artifact rather than something
Gradle produces. `manifest.json` was updated to 2.1.0 with Python and Core was
never rebuilt, so an APK shipped the new payload beside a Core that reported
catalog 2.0.1 and could not see it. The installed app was correct to report no
Python. Nothing persisted wrongly: the live Core matched the packaged one byte
for byte. `TestStagedCoreEmbedsTheCurrentCatalog` now fails in the normal gate
when the staged Core predates the catalog, checking the catalog version and
every bundled payload's name and pinned checksum, and naming the script to run.

Both guards are permanent. The shared lesson is the same one twice: the payload
guard checks that files are present, and presence is not reachability.

### Security posture, unchanged and stated plainly

Python is **not** sandboxed and is not described as such. The boundary is the
Android application UID. `subprocess` bypasses the Runtime's registry, timeout
policy, output bounds and event log, and guidance to prefer the Runtime is
advisory, not enforcement. No pip, no ctypes, no direct sockets, no writable
executable storage, and no shell is shipped. Shell availability is
version-dependent: Android 11+ provides `/bin/sh`, API 24-29 does not.

## 2026-08-30 — Provider Resilience & Automatic Failover (PHYSICAL PASS)

Branch `feature/provider-resilience-failover`, commits `68443c1`, `7b67493`,
`ebf49b4`, `812a003`, `3446b0f`. **Physically validated on the target ARM64
device** and merged to `develop`. Not released, `main` untouched, no tags moved.

| Check | Result |
|---|---|
| Provider automatic failover | **PASS** |
| Fallback Models UI, ordered selection | **PASS** |
| Failing primary answered by its configured fallback | **PASS** |
| Single user-visible final answer | **PASS** |
| Automatic gateway restart after model and fallback changes | **PASS** |
| Active-turn safety | **PASS** |
| White-screen resume recovery, no loop | **PASS** |

The restart sequence observed on device: active request running → configuration
saved → "Restarting Gateway" → the request finished → gateway restarted →
configuration active. **The active request was not interrupted.**

### The shape of the work

The existing `FallbackChain`, `CooldownTracker` and `ClassifyError` were reused
rather than rewritten, and three subsystems the brief allowed for were
deliberately not built: no checkpoint machinery, no semantic tool fingerprinting,
no side-effect classification. The agent loop already guaranteed that a provider
retry does not rewind completed tool execution, so that invariant is held by
tests plus one exact-`toolCallID` result-reuse guard rather than by new
mechanism that could itself be wrong.

Shipped alongside: single-candidate cooldown, `Retry-After` in both legal forms,
hard quota separated from transient throttling, 502/503/504 classified by what
each actually indicates, a conservative capability gate, streaming failover only
before first visible output, steering and cancellation preserved across retries,
a `provider.*` event family with redaction inside the emitter, the Fallback
Models UI, and automatic safe gateway config apply.

### The invariants that survived review

A busy gateway is **never** force restarted when the two-minute wait expires, and
an unverified busy state is **never** read as idle. Both leave the configuration
saved and unapplied; the manual Restart Gateway control remains the way to apply
it. An earlier revision forced the restart in both cases and was corrected in
`ebf49b4`.

Resume diagnostics name what was observed — `probe_failed`,
`page_unresponsive` — not a renderer death this layer cannot observe, since
`webview_flutter_android` 4.14.0 exposes no `onRenderProcessGone`. A test fails
if the old `renderer_gone` label returns.

### Counts

Tools **18**. Skills **7/7** existing workspace, **6/6** fresh install.

## 2026-08-30 — Resume white-screen recovery (physical PENDING)

Branch `feature/provider-resilience-failover`, on top of `ebf49b4`. Not merged.

Physical validation reported an intermittent blank content area after returning
from the background: Flutter chrome and navigation intact, only the embedded
console white, and the existing Refresh button fixing it immediately.

### What the code actually showed

Three findings, each verified by reading the source rather than inferred from
the symptom:

- `webview_android.dart` had **no lifecycle observer at all**. Nothing ran on
  resume, so nothing could notice or recover a lost page.
- `webview_flutter_android` 4.14.0 exposes **no `onRenderProcessGone`
  callback** — grep finds no such API anywhere in the plugin. If Android kills a
  backgrounded renderer, Dart is never told, and the WebView keeps its layout
  while rendering nothing. Both the widget and its container paint white, which
  is exactly the reported appearance.
- The console had **no React error boundary**, so an uncaught render error would
  unmount the tree and empty `#root` — visually identical to a killed renderer.

That leaves two failure modes that look the same from outside. Rather than guess
between them, the recovery covers both and the new logging tells them apart.

### Recovery, gated on evidence

On resume the host asks the page a question only a live page can answer: a
readiness flag set after the first paint, plus a non-empty `#root`. A healthy
page is left completely alone. Only a missing or wrong answer triggers a reload,
and the reload targets the route the user was on, not the console home page.

A blanket `onResume → reload()` was rejected: it would discard scroll position
and page state on every resume, add pointless work, and hide the defect. Recovery
is capped at one attempt per page load, so a genuinely broken console stops being
reloaded and the Refresh control stays useful.

Blankness is deliberately not detected by sampling pixels or background colour.
The page reports its own health; white is a symptom, not a signal.

### Also

An `AppErrorBoundary` now wraps the console, showing a reload affordance instead
of an empty page and clearing the readiness flag so the host probe can recover a
crashed tree too.

Resource errors during a gateway configuration restart are logged but never
trigger recovery on their own — the backend being briefly unavailable is
expected, and only a resume that finds a dead page acts.

### Unchanged

The physically validated deferred-restart behaviour is untouched: an active turn
still completes before a configuration restart runs.

### Not verified

Physical validation is PENDING. The new `[webview]` lifecycle logs record what
was observed rather than a cause: `probe_failed` means the page could not be
asked at all, `page_unresponsive` means it answered but is not rendering. A
killed renderer is the likeliest reading of the first, but this layer cannot
prove it, so the logs do not claim it.

## 2026-08-30 — Fallback models UI and automatic gateway restart (physical PENDING)

Branch `feature/provider-resilience-failover`, on top of `68443c1`. Not merged.

Physical validation of Provider Resilience found that the failover chain had no
UI: `Agents.Defaults.ModelFallbacks` was configurable only by editing JSON, so
the feature could not be exercised from the product at all.

### Fallback Models

A **Fallback Models** section on the Models page, beside the default model, with
add, remove, reorder and save. Candidates are picked from configured model
entries rather than typed by hand, and each entry shows its provider and model
identifier.

Placement is deliberate. `config.ModelConfig` also has a `Fallbacks` field, but
that one serves multi-key expansion within a single provider and is generated
rather than user-edited; putting the UI in the per-model edit sheet would have
targeted the wrong field and implied every entry has its own chain.

A fallback is stored as a **reference by model name**, so the referenced entry is
used with its own provider, credentials, base URL and headers. The primary's API
key is never copied to another provider's endpoint.

`POST /api/models/fallbacks` rejects unknown entries, duplicates, virtual models,
non-chat models, and a primary listed as its own fallback — the last because it
would make the chain retry the candidate that just failed. Zero fallbacks stays
valid, and a config written before this feature keeps working untouched.

### Automatic gateway restart

Saving a restart-requiring setting now applies it instead of leaving "Gateway
restart required" as the resting state. `saveAndApplyGatewayConfig` persists the
change, asks the backend whether a restart is needed, and calls the new
`POST /api/gateway/apply-config`.

The restart decision is unchanged: it is still the existing
`gateway_restart_required` signature comparison, so there is no second decision
system to keep in agreement, and cosmetic or hot-reloadable edits still restart
nothing. Success means the gateway is running *and* its boot signature matches
the saved config — not that the restart call returned 200.

**A settings change must never interrupt an answer.** Core's `/health` reports
`active_requests` and `busy`, fed by the agent loop's existing in-flight counter,
and `apply-config` restarts only when the gateway is not running or reports
itself idle. That is why it is a separate endpoint from the manual restart: the
manual action is immediate recovery the user asked for, while this one must not
cut off a Telegram reply mid-sentence or kill a `git push` half way through.

**Neither an unreadable busy signal nor an expired wait is permission to
restart.** The idle check yields one of four outcomes — `not_running` and `idle`
allow a restart, `busy_timeout` and `unverified` do not. The two-minute limit
bounds how long PocketClaw waits, not how long the user's work is safe: reaching
it leaves the configuration saved and unapplied, reported to the UI as
`saved_not_applied` with HTTP 202 and a warning rather than an error. A running
gateway that will not report its busyness is retried for five seconds and then
also left alone, because a gateway too old to answer — or one whose health
endpoint is briefly unreachable — may be mid-turn.

An earlier revision of this code forced the restart in both cases. That trade was
wrong: a delayed setting costs a banner, a forced restart costs destroyed work.
There is deliberately no background retry either; the restart-required indicator
stays visible and the next save or the manual action applies the change.
Concurrent saves coalesce into one restart at both the frontend and the launcher.

A failed restart never discards the save — the config is written first — and the
manual Restart Gateway control remains as the recovery path.

### Not verified

Physical validation is PENDING and is not claimed.

## 2026-08-30 — Provider Resilience & Automatic Failover (physical validation PENDING)

Branch `feature/provider-resilience-failover`, from `develop` at `0a0b3fa`. Not
merged, not released, `main` untouched.

Gap-closing on the existing `FallbackChain`, `CooldownTracker` and
`ClassifyError` rather than a new subsystem. No checkpoint machinery, no tool
fingerprinting, no side-effect classification, no shell-command parser: the
agent loop already guarantees that a provider retry does not rewind a completed
tool execution, and that property is now protected by tests plus one small
guard instead of by new mechanism.

### Error classification

The 5xx family is split by what each status actually indicates — 502 network,
503/521/522/523/529 overloaded, 500/504/524 timeout — instead of collapsing into
timeout, because a log that reports "timeout" for a gateway that could not reach
upstream sends diagnosis the wrong way.

A new `hard_quota` class separates an exhausted plan or credit balance from
transient throttling. A 429 alone cannot tell them apart, so the response body
refines it. The patterns are deliberately narrow in both directions: reading
ordinary throttling as hard quota would put a healthy provider into a long
cooldown, and the reverse produces a retry storm against a provider that has
already said no.

`AllowsSameCandidateRetry` is new and distinct from `IsRetriable`. Moving to
another candidate and retrying the same one fail for different reasons: a bad
key, an exhausted quota and an unpaid bill will all still be bad a second later.

### Retry, Retry-After and cooldown

`Retry-After` is parsed at the point the HTTP response is still available, in
both legal forms, refusing stale dates and malformed values so a bad header can
never become an unbounded wait. With a fallback candidate available the chain
does not wait it out — it puts the candidate into cooldown and moves on, which
is the latency this milestone exists to remove. Without one, the wait is honoured
up to a 30-second ceiling and stays cancellable.

Single-candidate failures now feed the same cooldown state. Previously a sole
configured provider failed with no memory at all, so every turn re-ran the same
doomed request. Same-candidate retries drop to one when a fallback exists, so
the worst case stops multiplying retries by candidates by iterations.

### Safety

`completedToolResults` records a finished tool result under the provider's own
`tool_call_id` and reuses it if that exact id reappears in the turn. Matching on
tool name or arguments is deliberately excluded — asking for the same command
twice in one turn is legitimate — and a call without an id is not recorded,
because inventing identity from arguments is the fingerprinting this design
rejects.

The capability gate skips a fallback candidate only when it is explicitly unable
to serve a tool-calling turn. Unknown is not a refusal: PocketClaw routes to
providers whose model lists it does not enumerate, and treating unrecognised as
unusable would disable failover exactly where it is needed.

Cross-provider context-overflow fallback is **deferred**: no reliable per-model
context capacity metadata exists, and failing over on a guess would overflow
again after paying the latency. Compact-and-retry on the current candidate is
unchanged.

### Empty responses

The placeholder no longer claims a provider error or a token limit. It fires for
ordinary contentless turns — a tool-only turn, a graceful interrupt, an
iteration limit — and the old wording was read as evidence of an outage when no
provider had failed. `responseIsUserVisiblyEmpty` is for logging and the
placeholder only, never for failover, and a response carrying tool calls is not
empty.

### Observability

A `provider.*` event family with emitter-level redaction, so a new call site
cannot forget it. Logs distinguish the configured model name, the provider, the
model identifier actually sent upstream, and the protocol — these diverge under
routing aliases, and blurring them makes failover diagnosis guesswork. An
unrecognised model reports its protocol as "unknown" rather than as the one that
will be attempted.

### Preserved unchanged

Streaming already gated failover on visible output and still does: a failure
before publication may recover, one after it must not, or the answer would be
duplicated on screen. Cancellation still cuts backoff short, and steering still
survives a provider retry — both verified rather than modified.

### Not verified

Physical validation is PENDING and is not claimed.

## 2026-08-30 — Lean Runtime Pack v2 (PHYSICAL PASS)

Branch `feature/lean-runtime-pack-v2`, fix commit `7ebd254`. **Physically
validated on a real ARM64 device on 2026-08-30** and merged to `develop`.

| Check | Result |
|---|---|
| Git HTTPS | **PASS** |
| `git clone` of a public GitHub repository | **PASS** |
| **Git helper symlink execution on Android** | **PASS** |
| `git --version` | 2.51.0 |
| `gh` | 2.82.1 |
| curl HTTPS | PASS |
| ripgrep | PASS |
| sqlite3 | PASS |

**The symlink question is settled.** Android permits executing through a symlink
in app-private storage that points at a packaged payload in `nativeLibraryDir`.
The kernel resolves the link and runs the read-only packaged file, so the API 29+
restriction on executing writable storage does not apply. That is what makes
multi-executable tools possible on Android, and it is now evidence rather than
reasoning. It still does not license copying an executable into app storage.

Skills read **7/7** on an existing upgraded workspace and **6/6** on a fresh
install. Both are correct: seeding only writes and never deletes, so a device
that already had the GitHub Skill keeps it, while a fresh workspace receives the
six seeded skills (`picoclaw-agent` is deliberately unseeded).

**Known limitation.** git ships with its default compiled-in `SHELL_PATH` of
`/bin/sh`, which Android lacks, so **git features depending on a shell — hooks in
particular, and the `ENOEXEC` fallback — are not guaranteed on Android.** Clone,
fetch and push do not need a shell.

## 2026-08-30 — git HTTPS remote-helper fix

Branch `feature/lean-runtime-pack-v2`, on top of `b13f297`.

The v2 physical run passed everything except `git clone` over HTTPS, which
failed with `unable to find remote helper for 'https'`.

### Root cause, proven rather than guessed

It was not TLS and not symlink execution. curl HTTPS passed on the same device,
and the failing code path never reached an exec.

git does not exec its transport helper directly. `get_helper()` builds
`remote-https …` with `git_cmd = 1`; `prepare_git_cmd()` prepends the literal
string `git`; `prepare_cmd()` then resolves **that** name through
`locate_in_PATH`, with `GIT_EXEC_PATH` already prepended to `PATH` by
`setup_path()`. PocketClaw's helper directory held `git-remote-http` and
`git-remote-https` but not `git`, and Android's `PATH` is
`/system/bin:/system/xbin`. The lookup returned `ENOENT`, and `get_helper()`
converts exactly that errno into the observed message — which names the protocol
and points at TLS, nowhere near the actual fault.

Reproduced natively before changing anything: with a `PATH` containing no `git`
and a `GIT_EXEC_PATH` holding both remote helpers, a host git produced the
identical error; adding a `git` symlink to the same directory made the clone
succeed with all three entries as symlinks.

### Fix

One catalog change: git declares `git` among its own helpers. No architecture
change, no git patch, no writable executable copy. The git payloads are
byte-identical to `b13f297`.

`TestGitDeclaresItselfAsAHelperSoItsOwnLookupSucceeds` was verified to fail with
the entry removed and pass with it present, so it genuinely guards the bug.

### Also in this commit

- `runtime.helpers.prepared` and `runtime.helpers.failed` record the exec path
  built and the logical names in it, so a helper failure can be told from a
  lookup failure without guesswork.
- The incomplete GitHub Skill was removed from the seeded workspace, and the
  onboarding tests that referenced it now use a skill that still exists.
- `SHELL_PATH` is knowingly left at git's default `/bin/sh`, which Android does
  not have. Overriding it breaks git's cross-build because its Makefile uses the
  same variable for its own recipes; nothing on the clone path needs a shell.

## 2026-08-30 — Lean Runtime Pack v2 (physical validation PENDING)

Branch `feature/lean-runtime-pack-v2`, from `develop` at `fa27ad2`. Not merged,
not released, `main` untouched.

Six bundled tools now ship, and the runtime learned two things it needed in
order to carry them: helper payloads and per-tool environment profiles.

### What ships

| Tool | Version | Installed | License |
|---|---|---:|---|
| git | 2.51.0 | 3.23 MB | GPL-2.0-only |
| git-remote-http | (same build) | 2.98 MB | GPL-2.0-only |
| gh | 2.82.1 | 55.9 MB | MIT |
| curl | 8.11.1 | 1.30 MB | curl licence + Apache-2.0 (mbedTLS) |
| ripgrep | 14.1.1 | 4.27 MB | MIT / Unlicense |
| sqlite3 | 3.50.4 | 1.23 MB | public domain |

`zip`, `unzip`, `diff`, `patch`, `file` and `tree` were added as `system`
entries costing nothing: shipping a catalog entry *is* the probe, and the
physical run will report which the platform provides.

### Helper payloads

git looks its transport helper up as `git-remote-https` inside `GIT_EXEC_PATH`,
and Android's package manager only unpacks `lib/<abi>/*.so`, so no packaged file
can carry that name. A catalog entry now declares helpers by logical name; the
runtime verifies each payload's checksum and ABI alongside the main one and, at
execution time, builds a directory of **symlinks** to them.

Nothing is written into app storage and executed — the kernel resolves the link
and runs the read-only packaged file — so the delivery model is unchanged. This
is the one platform assumption v2 rests on, so the probe now also reports
`symlink_exec` on every device instead of it being trusted.

A tool whose helper fails verification resolves as *unavailable*. git with an
unverified transport helper would otherwise look installed and fail at its first
`https://` URL, which is a much harder failure to read.

### Environment profiles and credentials

Tools may declare an `environment_profile` the runtime prepares and the caller
cannot. `git` gets its helper path, prompts disabled so a missing credential
fails fast instead of hanging until the timeout, a private HOME inside runtime
storage, and the platform CA path — probed across both the Android 14 Conscrypt
APEX location and the older one. `gh` gets the helper directory on PATH so it
finds the runtime's verified git, with prompts, pager and update checks off.

A GitHub token is injected through git's environment-based config and through
`GH_TOKEN`, never through argv. `https://TOKEN@github.com/...` is forbidden: it
leaks the credential into the command line, into git's on-disk remote config,
and into any error quoting the URL. Both values are redacted before any writer.

A `transfer` timeout profile (30 minutes) was added, because a clone or a release
upload is bounded by a network peer and must not share a utility command's budget.

### Notable

- curl is real curl. mbedTLS instead of OpenSSL is why the whole TLS stack costs
  about 1.3 MB, and the same libcurl is linked into `git-remote-http`, so curl is
  nearly free once git is present. This reverses the Foundation milestone's
  deferral, on measurement rather than opinion.
- gh's 56 MB is a sanctioned exception, not a precedent. yq was measured at
  11.25 MB and left out on the size policy, since jq already covers JSON.
- Three bionic portability issues had to be solved for git, none of them papered
  over: no pthread cancellation (a documented no-op shim, which is correct where
  nothing can be cancelled), no `sync_file_range`, and no separate `libpthread`.
  `arc4random` was selected as the CSPRNG because bionic provides it natively.
- The build-path privacy check was tightened. The old pattern matched any
  `/root/` substring, so it false-positived on Go's own trimmed module paths like
  `pkg/root/trusted_root.go`. It now tests for this build's actual home directory
  plus boundary-anchored developer roots. It also caught Rust baking cargo
  registry paths into ripgrep's panic strings, now fixed with a neutral
  `CARGO_HOME` and `--remap-path-prefix`.

### Not verified

Physical validation is PENDING and is not claimed. In particular the
`symlink_exec` assumption behind `git clone` has not been observed on hardware.

## 2026-08-30 — Managed Runtime Foundation (PHYSICAL PASS)

Branch `feature/managed-runtime-foundation`, commit `ee236da`, based on
`v0.2.0-rc2` / `404ef44`. **Physically validated on a real ARM64 device on
2026-08-30** and merged to `develop`. Not released, `main` untouched.

Physical results: the runtime tool registered; jq 1.7.1 executed and processed
JSON; `sha256sum`, `grep`, `sed`, `tar`, `uname`, `df` and `ping` executed;
stderr was captured; a non-zero exit code was preserved; a timeout terminated a
harmless long-running command; runtime lifecycle events appeared in the logs; no
secret leakage was observed; and the runtime was driven end to end through
Telegram. Service and Gateway Auto-Start still worked and the Gateway PID
ownership false positive did not return.

**43 of 44** catalog tools were available on the tested device. `traceroute` was
correctly reported unavailable rather than assumed present — the resolver
measuring the device instead of trusting the catalog, which is what it is for.

The writable-app-data execution probe returned **inconclusive** on that device:
it could neither run its staged copy nor observe a clean permission refusal, and
it said so rather than guessing. That costs nothing, because the architecture
never used writable executable storage — the bundled jq payload executed from
`nativeLibraryDir` on the same run, which is the path the runtime actually uses.

Counts are Tools **18** and Skills **7/7**. Skills 7/7 is not a regression: the
incomplete GitHub Skill was removed deliberately and is not being restored.

The PocketClaw Agent now has a controlled, observable, verified local tool
environment. It names a tool; the runtime resolves that name through a versioned
catalog, verifies it, and runs it under bounded execution with a full structured
lifecycle. No physical binary path is ever handed to the model.

### The finding that shaped the design

PocketClaw targets Android SDK 36. Since API 29 an app may not `execve()` a file
in its own writable data directory, and `File.setExecutable(true)` does not
change that — the restriction is enforced on the app's SELinux domain rather than
by the file mode. The previously planned app-private `runtime/bin` cannot work.

Executables reach the device by two routes instead, both read-only to the app:
the platform's `/system/bin`, and APK payloads the package manager unpacks into
`nativeLibraryDir` — the same mechanism the Core payload has always used. There
is consequently no download-and-execute code path in the runtime at all, which
makes "no arbitrary binary installation" structural rather than a policy someone
has to keep enforcing.

The assumption is not merely asserted: an execution probe copies a harmless
system binary into writable storage, tries to run it, records the verdict in the
Debug Logs, and deletes the copy. The runtime never depends on the answer.

### New

- `core/src/pkg/pcruntime`: versioned manifest and catalog, tool resolver,
  bounded execution API, structured lifecycle events, redaction, read-only
  inventory, execution probe, storage layout policy.
- `runtime` agent tool with `list`, `info` and `run`. Agent tool count moves
  17 -> 18; all 17 existing tools are unchanged.
- Runtime Pack v1. Tier 1 is catalogued as system-provided and probed per device,
  because Android already ships toybox and bundling BusyBox would duplicate the
  platform at the cost of tens of megabytes and a GPLv2 source-offer obligation.
  jq 1.7.1 is cross-built from the pinned official release tarball by
  `runtime/build-jq-android-arm64.sh` and bundled as `libpocketclaw-jq.so`,
  proving the packaging contract end to end.
- `RUNTIME.md`, `runtime/README.md`, runtime guidance in
  `core/src/workspace/AGENT.md`, the `runtime` tool description, and the GitHub
  Skill.

### Behaviour worth knowing

- Managed tools run by direct `argv`, never `sh -c`, so shell metacharacters in
  an argument are inert. This is deliberately narrower than the existing `exec`
  tool and does not relax to match it.
- The child environment is constructed from an allowlist rather than inherited,
  so provider keys held by the Core process cannot reach a child by accident.
  `PATH`, `LD_PRELOAD`, `LD_LIBRARY_PATH`, `LD_AUDIT` and the `DYLD_*`
  equivalents are refused to callers.
- Timeout profiles are ceilings a caller may lower and never raise. Cancellation
  terminates the child's whole process group and always reaps it.
- Runtime logs record that a tool ran, never what it printed: only `stdout` and
  `stderr` byte counts are persisted. The caller still receives the real output.

### Verified in this session

- `go test ./pkg/pcruntime/ ./pkg/tools/` green, including argv preservation,
  timeout, cancellation, process-group termination, output truncation,
  working-directory policy, checksum and ABI rejection, concurrent-resolution
  deduplication, redaction, and complete lifecycles on success, failure, timeout
  and cancellation.
- The jq payload survives Gradle packaging byte-identical: the entry extracted
  from the release APK hashes to
  `3c1f61c100d7b8f3a68355f9cd697952bae27579cba516a0a3e43ac54926c997`, matching
  the catalog pin.

### RC2 preserved

No change to Service or Gateway Auto-Start, manual Stop authority,
`START_NOT_STICKY`, the Gateway PID ownership fix, Core loopback binding,
Dashboard auth, Public Mode, Telegram, credential redaction, or the internal
PocketClaw chat.

## 2026-08-30 — v0.2.0-rc2 frozen (PHYSICAL PASS)

Release candidate 2, published as a GitHub pre-release. Not a production
release and not published to Google Play. `main` untouched; `v0.2.0-rc1` and
`phase2-milestone-d` not moved.

Physical validation PASSED on a real ARM64 Android device on 2026-08-30 for
both commits in this candidate. Physical reference APK SHA-256:
`182b85183156428aa93baf3113492484258a3a1eace95f7bccd9ed82035177a3`
(`com.lord1egypt.pocketclaw` 0.2.0, versionCode 4). It covered fresh install,
Service Auto-Start, Gateway Auto-Start, manual Service stop and start, Gateway
starting automatically after the Service, no immediate Service resurrection,
internal PocketClaw chat, Core startup, Skills 8/8, Tools 17, Core bound only
to `127.0.0.1:18790` / `[::1]:18790`, a working Dashboard, and no recurrence of
the Gateway PID ownership false positive.

What RC2 adds over RC1: simplified Service and Gateway Auto-Start, both
preferences persisted in the Android canonical SharedPreferences store,
evaluation only at a legitimate app launch with no resume-triggered
orchestration, authoritative manual Stop, `START_NOT_STICKY`, removal of the
experimental FAILED/stale lifecycle architecture, and the Android Gateway PID
ownership false-positive fix that stops a valid running Gateway's pid file from
being deleted while keeping foreign, dead, and stale PID protection.

`feature/autostart-foundation` was NOT merged and remains reference only. The
shipped implementation was rebuilt from `v0.2.0-rc1`.

Release identity: versionName stays `0.2.0`, versionCode moves `4` → `5`. The
candidate identity lives in the tag, matching how `v0.2.0-rc1` shipped. The
versionCode bump means the released APK is not byte-identical to the physically
validated one; the native payload is, see below.

Native payload provenance: `libpicoclaw.so` and `libpicoclaw-web.so` are
committed build inputs under `android/app/src/main/jniLibs/arm64-v8a/`, not
build outputs. Core was deliberately not rebuilt for this release, so the
released APK carries the physically validated Core binaries byte for byte:
`0bf50e618a5f3cb92205365e9eb46df6d72e2c72d7d900ed76bb725563ce09da` and
`d38f200df217b3b31e79d5bc0dfbe0a8393fc845341f37a3abb1273718b5e758`.

Final verification: `flutter analyze` clean and 134 Flutter tests pass;
`go test -tags goolm,stdjson ./...` exit 0 with no failing package and
`go vet -tags goolm,stdjson ./...` exit 0; focused Gateway/PID, Auto-Start
gate, Telegram, auth, Public Mode, and binding suites pass; the RC1→RC2 diff
contains no credential literals, no wildcard authorization, and no new LAN
exposure; the shipped Core binaries carry zero developer paths.

## 2026-08-30 — Gateway PID ownership false positive on Android

The Auto-Start safe rebuild PASSED physical validation. One defect remained in
its logs: while a launcher-spawned Gateway was serving traffic on PID 5312,
status polling logged

    ignore pid file for PID 5312: pid belongs to another process; ignoring
    stale pid file
    removed stale pid file for PID 5312

and deleted a valid pid file.

Root cause. `validateGatewayPidData` proves ownership by running
`ps -o command= -p <pid>` and looking for a bare `gateway` token, which is the
subcommand in the launch line `<binary> gateway -E --no-color`. That message is
reachable only when `ps` succeeded and returned non-empty output that contained
no such token, so Android's `ps` answered with something that is not the argv
the matcher expects. On Android the Gateway is executed as
`.../lib/arm64/libpicoclaw.so`, and the `gateway` subcommand is absent from what
`ps` reports for it.

Why startup and status disagreed: the readiness goroutine in
`startGatewayLocked` accepts the pid file on `pd.PID == pid` alone and never
calls `sanitizeGatewayPidData`, so it logged "Gateway pidFile detected". Only
the later status, realtime-proxy, and start paths ran the command-line
heuristic, and only those rejected the same live process.

Two small changes, no framework:

- `launcherOwnsGatewayPID` short-circuits validation when this launcher spawned
  the exact pid via `exec.Cmd` and that process is still alive. First-hand
  `exec.Cmd` ownership outranks a platform-dependent `ps` description. It is
  safe against pid reuse because the child remains a zombie holding its pid
  until `cmd.Wait()` returns, and the monitor goroutine clears `gateway.cmd` at
  that moment. Attached (not spawned) processes are excluded and still take the
  ordinary path.
- `classifyGatewayCommandLine` makes a negative command-line match decisive
  only when `ps` actually returned a command line. A bare executable name is now
  reported as un-inspected, which falls through to the existing health probe
  and its pid-identity check rather than deleting the pid file.

Dead-process cleanup is unaffected: it lives in `ppid.ReadPidFileWithCheck`,
runs before validation, and never consulted the command line.

PID validation was not disabled, the pid file is still removed for a dead or
decisively foreign process, and the discarded experimental
`gatewayRuntimeReady` framework was not restored.

Observability: a `gateway.pid.validation` DEBUG event records
`ownership_signal`, `validation_result`, `stage`, and `reason`. Repeats of the
same verdict for the same pid are suppressed, so status polling cannot spam it.
No credentials, paths, or command lines are logged.

- Diff: 3 Core source files, Core-only. No Flutter, Kotlin, Auto-Start,
  preference, auth, port, version, or packaging change.
- Validation: 12 new focused Go tests; full Go sweep passes with
  `-tags goolm,stdjson` (exit 0); `flutter analyze` clean and 134 Flutter tests
  pass; Core rebuilt with zero developer paths; provenance patch regenerated
  (141 files, no new files).
- Physical verification: PENDING. No merge, no tag, no release.

## 2026-08-30 — Auto-Start safe rebuild from the RC1 baseline

`feature/autostart-foundation` passed automated tests but failed physical
acceptance, most seriously by turning a manual Start Service press into a
FAILED state that RC1 cannot reach. That branch is retained for reference only.
It was neither merged nor cherry-picked. This work restarts from
`develop @ e470bb6` (`v0.2.0-rc1`) on `fix/autostart-safe-rebuild`.

Why the experimental branch failed, and what was therefore discarded:

- It changed native `PicoClawService.start()` to return a `Boolean` and taught
  Flutter to map `false` onto `ServiceStatus.failed`. But `false` also meant
  "skipped", returned whenever the process-lifetime `isStarting`, `isStopping`,
  or `manualStopActive` companion flags were set. Those flags outlive the
  Service object and have paths that never clear them, so one stale flag turned
  every later manual Start into FAILED.
- It set `hasFailed = true` after *every* child-process exit, including a
  requested stop, from a thread running outside the lock that had just cleared
  it. A clean manual Stop therefore left the service marked failed.
- `inspectServiceState()` latched: its fallback returned FAILED whenever
  `_status` was already FAILED, even when the native side reported a clean stop.
- Several later commits existed only to mask races the earlier ones introduced
  (transition generations, the owned-PID fast path, the `hasFailed` mask). Both
  halves of each pair were discarded rather than carried forward.

The rebuild is RC1 plus a preference and one call:

- `LaunchAutoStartPreferences` (Android) is the single canonical store, in the
  existing `picoclaw_prefs` file under new keys, written with a synchronous
  `commit()` and acknowledged only from a post-commit readback. RC1's separate
  `auto_start` boot-receiver key is untouched. Both new preferences default ON.
- `ServiceManager.evaluateLaunchAutoStart()` runs once, from `main()`, after a
  true app-process launch. There is no resume hook, watchdog, boot receiver
  change, crash-restart policy, or resurrection path, so a manual Stop stays
  stopped until the user starts it again or relaunches the app.
- Gateway auto-start is a preference only. The Service exports
  `POCKETCLAW_GATEWAY_AUTOSTART` into the Core child environment and Core's
  existing `TryAutoStartGateway()` is gated on it. No Flutter gateway lifecycle
  code, no Android gateway bridge, and no second definition of gateway
  readiness. An absent or unparsable value keeps RC1's always-on behaviour.
- `START_STICKY` became `START_NOT_STICKY`, and a null restart intent now stops
  the Service instead of re-entering the start branch. This addresses the
  observed unexpected Android Service restarts. RC1's `runWebService` crash
  restart is deliberately unchanged; it is guarded by `stopped` and never fires
  on a manual stop.
- `ServiceStatus` stays `{ stopped, running, starting }`. No `failed` state, so
  no stale failure can be displayed or latched. Settings shows the persisted
  preference and RC1's live 3-second runtime poll as two separate lines.

Also carried across, independent of Auto-Start: the log sanitizer now redacts
JSON credential fields, `key=value` credential assignments, `Bearer` tokens,
and `sk-` API keys, which RC1 did not cover.

- Diff versus RC1: 11 files, +458/-9, plus three new files. The experimental
  branch was 32 files and +5048/-246.
- Validation: `flutter analyze` clean, 134 Flutter tests pass, the full Go
  sweep passes with `-tags goolm,stdjson`, Core rebuilt with zero developer
  paths, and the Core provenance patch was regenerated (141 files).
- Physical verification: PENDING. No merge, no tag, no release.

## 2026-08-26 — DEBUG log cleanup micro-pass

- Physical DEBUG export from `f663d25a...71c621d` isolated two small remaining
  defects without invalidating its physically passed Telegram surface, caller,
  branding, terminal cleanup, or exactly-once queue behavior.
- Android Export Logs used `Uint8List.fromList(content.codeUnits)`. That
  truncated UTF-16 code units into bytes, so valid Go `53.616µs` reached the
  exported file as invalid byte `B5` and decoded as `53.616�s`. Export now uses
  UTF-8. Strict MethodChannel transport tests preserve Arabic, emoji, `µ`,
  punctuation, and ANSI-wrapped multibyte text without introducing U+FFFD.
- DEBUG HTTP middleware logged the successful `/api/gateway/logs` and
  `/api/gateway/status` requests used by the UI to monitor itself. Only exact
  expected GET+2xx polls are now omitted. Errors, unexpected methods, redirects,
  unknown routes, config/models, and all other requests remain visible.
- Validation: Flutter analyze clean and 97 tests; frontend 36 tests, `tsc -b`,
  lint; tagged Go logger/gateway/API/middleware suites pass. Core patch
  regenerated (95 files), canonical arm64 Core build has zero developer paths,
  and the APK guard passed.
- Candidate: `eacbbc86b99429f114aba9ba1dca57224122fa176f6b4d99edf350454423f9a8`,
  34,239,073 bytes, `com.lord1egypt.pocketclaw` 0.1.3 (3). Core hashes:
  `5c09eb72...4d763bc` / `cb6b10cc...4b03b52`. Secret/path scan clean apart
  from the already-deferred generated Dart source URI. Physical verification
  remains pending; no merge or release.

## 2026-08-26 — user-facing log privacy and duplication fix (device-found)

Found on a physical device after Milestone D closed. The GitHub milestone
release is on hold until this is re-tested. Application version unchanged at
0.1.3 (3).

- **Caller leaked the upstream module path.** `-trimpath` removed
  `/home/lordegypt/...` exactly as intended, but what replaces an absolute path
  under `-trimpath` is the Go module path, so the Logs screen showed
  `github.com/sipeed/picoclaw/web/backend/api/gateway.go:298`. Every existing
  assertion searched for `/home/`, so the substitution went unnoticed. Fixed by
  setting `zerolog.CallerMarshalFunc` in `logger.init()` — the earliest layer
  that sees caller metadata, so every writer and every exported log inherits
  `gateway.go:298`. No message text and no in-message path is rewritten.
- Eight user-visible message strings that named the project were individually
  reworded, including four provider errors telling users to run a CLI command
  that does not exist on Android.
- **One event was rendering as hundreds.** Not repeated emission and not a
  lifecycle fault: every duplicate carried the identical timestamp and PID, and
  the counter sat at the full 500. Native `lastLog` is a sticky snapshot that
  never clears, and the Flutter three-second status poll appended it on every
  tick, so a single warning refilled the buffer indefinitely and evicted all
  real history. Fixed by making the producer match the consumer: `publishLog`
  queues each line and `takeNewLogs` drains it, so every line is delivered
  exactly once. This also recovers lines emitted between polls, which the
  snapshot silently dropped.
- Both regression tests were confirmed to fail against the old code — 25 polls
  produced 25 copies before the fix, one after — and the Dart test drives the
  real polling path rather than a helper.
- Core rebuilt, `-trimpath` clean, upstream patch regenerated (67 files).
  Candidate APK `543c759b04b0e4c77dd7831435753aceac0b1e16a7a45fdec3fb37ed2e45479a`,
  guard PASS, secret scan clean.
- **Telegram connection state was out of sync between the two surfaces.** The
  console showed Connected while native Settings hardcoded "Connect PocketClaw
  to Telegram" and opened a new pairing. Both now derive from the persisted
  `channel_list.telegram` entry through `TelegramConnectionReader`; no stored
  boolean exists to drift. An already-connected user gets a connected page with
  Open Chat, an explicit confirmed Reconnect, and Advanced / Manual. Replacement
  was verified safe: the config writer runs only after a new token arrives, so a
  cancelled or expired pairing leaves the working bot intact.
- Recorded but not fixed: `libapp.so` carries the Flutter plugin registrant's
  source URI, a developer path present in the device-verified APK and every
  earlier one. It reaches users only in a Dart stack trace, Dart has no
  `-trimpath` equivalent, and it needs its own decision.

## 2026-08-26 — Phase 2 Milestone D COMPLETE (physical E2E PASS)

- **Milestone D, Telegram Managed-Bot Onboarding, is closed.** The full flow
  passed a physical end-to-end test on a real Android device against the
  production service, including a real Telegram → PocketClaw Core → AI provider
  → Telegram message round trip with conversation context persisting across
  consecutive messages.
- Automatic managed-bot onboarding is now the **default** Telegram setup path.
  Manual Bot Token entry is the **advanced fallback**, retained in full.
- Verified reference artifact, superseding the Milestone C APK:
  `b6fea5d8ec5c3c66ba8a1320b0a217afcca322e75b5b26cc4082bbbb08a57f94`,
  34,220,929 bytes, `com.lord1egypt.pocketclaw` 0.1.3 (3).
- The Core pair rebuilt for the UI integration fix is now **device-verified**
  with that APK and supersedes the Milestone C pair: `libpicoclaw.so`
  37,224,801 `33f8b4efbc88333747c5df30b3ddc6864c924b35dba99e9c8c3b91df3470e98a`;
  `libpicoclaw-web.so` 24,641,889
  `5400cb02322ece5c7035356595355bd3c116adfc6a5bb6b78f3e2bd22dbcb3bd`.
- Live architecture: PocketClaw Android → PocketClaw Telegram Setup on Vercel →
  `@PocketClawSetupBot` → Telegram Managed Bots → Upstash Redis pairing state
  (REST, 600 s TTL) → automatic PocketClaw Telegram configuration. The service
  repository `Lord1Egypt/PocketClaw-Telegram-Setup` stays independent and
  public; the APK builds from none of it and carries only its public base URL.
- Device-observed Core regression: startup and Flutter first frame, Gateway and
  Core lifecycle, Telegram channel lifecycle, branding, no black screen, no
  abnormal slowdown, log path privacy intact. Android DNS/model discovery,
  Provider Catalog, Gemini, OpenCode Zen, OpenCode Go, Skill Hub, Workspace and
  MQTT keep their existing automated coverage and were not re-tested physically
  in this round.
- Security confirmed: no manager token, webhook secret, pairing secret, Redis
  credential or child bot token in the APK or in Git; no child token in the QR;
  no raw token shown in normal UI or required from the user. No secret value is
  recorded in this repository.
- Merged into `develop` with `--no-ff` and tagged `phase2-milestone-d`. `main`
  remains deliberately untouched at `100a51d`.
- Still open: rotating the manager bot token, which was pasted into a chat log
  after deployment. The service is unaffected and keeps working; this is
  hygiene, and it is the one outstanding security item.

## 2026-08-26 — Milestone D UI integration fix (device-found)

- **Root cause of the device failure**: PocketClaw renders Telegram on two
  surfaces. Milestone D added managed-bot onboarding to the native Settings tab
  (`lib/src/ui/config_page.dart`) while Channels → Telegram — the path a user
  actually takes — is the Core web console's
  `channel-config-page.tsx` → `telegram-form.tsx`, which knew nothing about it.
  The flow was complete, tested, and unreachable.
- Connected the console's Telegram route to the existing Dart flow rather than
  rewriting pairing in TypeScript. A new `PocketClawHost` WebView bridge lets
  the console render the entry point and ask the Flutter host to run the flow;
  `TelegramOnboardingLauncher` is now the single way in, called by both
  surfaces.
- Channels → Telegram now shows managed onboarding first when there is no token
  and a host that can pair, a connected summary when a token is set, and the
  full manual form when there is no host or no compiled-in endpoint. The legacy
  Bot Token / API Base URL / proxy / allow_from / typing / streaming /
  placeholder form is unchanged and still reachable in every state.
- Bridge security: no secret in the injected payload, the bot handle is
  JSON-encoded so it cannot break out of its string, `openExternal` takes only
  absolute http(s) URLs, and both injection and message handling are scoped to
  the console's own origin so a followed outbound link cannot drive the app.
- **The tests are the actual deliverable here.** The new web tests render
  `ChannelConfigPage channelName="telegram"` — the same component the APK
  renders — and were verified to fail against the old wiring: reverting the one
  line back to `TelegramForm` failed all 8, restoring it passed all 8. Adding
  `jsdom` and `@testing-library/react` was unavoidable, since the entire failure
  was that nothing rendered the real route.
- Verification: 36/36 frontend, 88/88 Flutter, `flutter analyze` clean,
  `tsc -b` clean, `pnpm lint` clean.
- **Core was rebuilt** because `core/src/web/frontend` changed. New binaries
  `libpicoclaw.so` `33f8b4ef...3470e98a` and `libpicoclaw-web.so`
  `5400cb02...2dbcb3bd` replace the device-verified pair, `-trimpath` verified
  clean, and `core/pocketclaw-core-v0.3.1.patch` was regenerated (58 files).
  The regression sweep on the device is therefore mandatory, not optional.
- Replacement APK
  `b6fea5d8ec5c3c66ba8a1320b0a217afcca322e75b5b26cc4082bbbb08a57f94`,
  34,220,929 bytes, guard PASS, secret scan clean over 425,247 strings.
- Not merged; `main` untouched.

## 2026-08-26 — Milestone D live-endpoint APK built

- Built the first PocketClaw APK that carries a real onboarding endpoint:
  SHA-256 `9a0f74070f0129b2180b4b3237fbfacaf001ee6c8808a26e128d7ae06bb1be7f`,
  34,211,833 bytes, `com.lord1egypt.pocketclaw` 0.1.3 (3). No Milestone D
  feature behavior changed; the only difference from the previous candidate is
  that `POCKETCLAW_ONBOARDING_BASE_URL` is now set.
- The endpoint travels through the canonical Gradle release command, not around
  it. `-Pdart-defines=<base64 of KEY=VALUE, comma-separated>` is the Gradle-path
  equivalent of `--dart-define`: `FlutterPlugin.kt` reads the `dart-defines`
  property and `BaseFlutterTaskHelper.kt` forwards it to `flutter assemble` as
  `--DartDefines`. Recorded because the obvious move — switching to
  `flutter build apk` to get `--dart-define` — would have left the canonical
  release path for no reason.
- The arm64 native-payload guard passed and printed all three libraries, and
  the 34.2 MB size confirms `-Ptarget-platform=android-arm64` was honoured
  rather than silently producing a ~50 MB universal APK.
- Secret scan over the printable strings of every file in the APK: zero
  Telegram bot tokens of any shape, zero occurrences of
  `TELEGRAM_MANAGER_BOT_TOKEN`, `TELEGRAM_WEBHOOK_SECRET`, `PAIRING_SECRET`, or
  any `KV_REST_API_*`/`UPSTASH_*`/`REDIS_URL` name, and zero `upstash` or
  `redis://` strings. The 1,432 64-hex hits — the shape of the webhook and
  pairing secrets — are fully attributed: 23 are `google_fonts` font-asset
  checksums in `libapp.so`, and the rest live in Core binaries that are
  byte-identical to the pair compiled and device-verified on 2026-08-25, before
  the service existed. A secret that did not exist at compile time cannot be
  inside them.
- Regression run before the build: `flutter analyze` clean and 77/77 Flutter
  tests. Core was deliberately not rebuilt — no Core source changed and the
  committed binaries hash-match the device-verified pair.
- Not merged. `feature/telegram-managed-onboarding` stays unmerged and `main`
  untouched until the Android → Telegram → managed bot → PocketClaw end-to-end
  test passes on a physical device.

## 2026-08-26 — Onboarding service live; can_manage_bots verified

- The service is deployed at
  `https://pocketclaw-telegram-setup-bot-83ai.vercel.app` and every server-side
  check passes: manager authentication, webhook registration, Upstash Redis
  storage, and a live test pairing that was created and read back.
- **`can_manage_bots = true`, verified live.** This was the last link in the
  chain that nothing else could establish — the BotFather UI showing management
  mode enabled and a manually-opened deep link were both strong evidence, but
  only the running service makes that API assertion.
- Fixed three deployment defects found by actually deploying, none of which any
  amount of local testing would have surfaced:
  - The build failed because the repository satisfied both Vercel Go build
    modes at once. Committed to the framework preset, removed the `api/`
    function, and added `internal/deployconfig` plus CI so a repeat fails on
    push rather than in a deploy log.
  - The storage health check only sent `PING`, which a read-only credential
    answers happily. Since a Vercel Redis store injects both
    `KV_REST_API_TOKEN` and `KV_REST_API_READ_ONLY_TOKEN`, the wrong paste
    would have shown green and failed every pairing. It now does a real write
    round-trip.
  - A working deployment still displayed "Connect a Redis database" under a
    green storage row, because the help block was revealed at first paint and
    never hidden again.
- Corrected the storage variable documentation. A real Vercel Redis store
  injects the `KV_REST_API_*` names, not the `UPSTASH_REDIS_REST_*` ones the
  README had led with. Both are current; which appears depends on how the
  database was attached.
- Android is untouched. The remaining work is a rebuild against the deployment
  and the device test.

## 2026-08-26 — Telegram onboarding extracted, reworked for Vercel, published

- Operator state corrected: `@PocketClawSetupBot` **exists**, Bot Management
  Mode is **enabled**, and the managed-bot deep link has been opened
  successfully against it. The remaining unverified link is the live
  `getMe` → `can_manage_bots` assertion, which needs the deployed service.
- The onboarding service moved out of this repository to
  **`Lord1Egypt/PocketClaw-Telegram-Setup`** — public, MIT, zero external Go
  dependencies. `services/README.md` is the pointer and explains why it is not
  vendored: it is infrastructure, the APK does not build from it, and the app
  holds only a public base URL.
- **Audited the storage before deploying, and it would not have survived.**
  Pairing state was a process-local Go map. On Vercel the create request, the
  Telegram webhook, and the token collection can each land in a different
  function instance, so a Go map works in development and fails intermittently
  in production. State now lives in a Redis-compatible store over its REST API.
- Two operations are atomic server-side rather than in application code:
  `SET username:… NX` claims a suggested bot username, and `GETDEL token:…`
  delivers the child token exactly once. Both `Store` implementations run
  against one conformance suite, including a test that twelve racing callers
  produce exactly one winner, and a test asserting the Redis path really issues
  `GETDEL` and `SET … NX`.
- **Long-polling became a webhook**, because serverless has no long-lived
  process. The endpoint is gated by the secret Telegram echoes in
  `X-Telegram-Bot-Api-Secret-Token`, compared in constant time before parsing.
  An undecodable body still answers 200 so Telegram does not retry forever.
- Added an operator status page and `/privacy`. The page performs no privileged
  action itself: each button asks the server, which reads credentials from its
  own environment and answers with a boolean and a non-secret message.
  Deliberately unlike the earlier DukeBot pattern, no Telegram token ever
  reaches browser JavaScript.
- Poll tokens are now stored as HMAC-SHA256 keyed with `PAIRING_SECRET`, so a
  storage dump is inert without the server's key, and rotating the secret
  invalidates every live pairing at once.
- Deploy-to-Vercel button, `.env.example`, `README.md`, `PRIVACY.md`,
  `SECURITY.md`, and `LICENSE` written for a standalone public project.
- Tests in the new repository: 7 packages, all passing, including a fake
  Upstash REST server so the Redis command construction is exercised for real.
  Nothing requires a live bot or a production credential.
- The Android side is unchanged. The API contract did not move during the
  extraction — same three endpoints, same fields, same 404-for-anything-gone
  behaviour the Flutter client already expects.
- No credentials are recorded anywhere in either repository. The previously
  issued manager token is treated as exposed and must be revoked before
  deployment.

## 2026-08-26 — Milestone D: Telegram managed-bot onboarding

- Closed Milestone C first: merged `feature/provider-catalog` into `develop`
  (`36bc88d`), tagged `phase2-milestone-c`, and branched
  `feature/telegram-managed-onboarding` from the verified `develop`.
- Verified Telegram's managed-bot API against `core.telegram.org` before
  writing any code. It is Bot API 9.6, 2026-04-03: `User.can_manage_bots`,
  `Update.managed_bot` carrying `ManagedBotUpdated{user, bot}`,
  `Message.managed_bot_created`, `getManagedBotToken(user_id)`,
  `replaceManagedBotToken(user_id)`, and
  `t.me/newbot/{manager}/{suggested}[?name=]`. One correction fell out of this:
  the field is `can_manage_bots`, not `bot_can_manage_bots`; coding against the
  latter would have made manager verification always fail.
- Added `services/telegram-onboarding/`, PocketClaw's own onboarding service. A
  separate Go module with **zero external dependencies**, deliberately outside
  `core/src/` so it never appears in the upstream provenance patch. No Hermes
  or Nous service is involved at build time or runtime; Hermes was read for the
  shape of the flow and nothing else.
- Pairing API is three endpoints. Polling never returns a token; a separate
  single-use collection endpoint delivers it once and destroys the session.
  That separation is what makes single-use a property of the API shape rather
  than of careful client behaviour.
- Pairing security: 16-byte pairing IDs and 32-byte poll tokens from
  `crypto/rand`, independent of each other; poll tokens stored only as SHA-256
  and compared in constant time; a wrong token and an unknown pairing both
  answer 404 so live pairings cannot be enumerated; per-client rate limiting;
  a 10-minute TTL after which the session and any token material are swept.
- Child bots are named `PocketClaw Agent` / `pocketclaw_<random>_bot` with an
  8-character random segment. Telegram's username rules are enforced, and
  `hermes`, `picoclaw`, and `sipeed` are rejected outright in both names and
  usernames.
- The manager bot token never leaves the server, and is redacted from every
  error path — including transport errors, which quote the request URL and
  therefore the token.
- The service verifies its manager bot at startup and refuses to run without
  `can_manage_bots`, rather than issuing links that could never resolve.
- Flutter: a new Telegram screen with Connect, Open Telegram, a QR carrying only
  the public creation link, live progress, an expiry countdown, Connected with
  Open Chat, and retry. Reached from Settings.
- Android lifecycle is handled properly: polling stops on background and
  resumes with an immediate check, the pairing survives Telegram taking focus,
  and it survives the app being killed via app-private storage of the pairing
  identifiers — never the bot token.
- Auto-configuration reuses the existing `channel_list.telegram` entry rather
  than adding a second Telegram runtime. It merges, so proxy, base URL,
  MarkdownV2, streaming, and the reasoning channel survive pairing, and it sets
  `allow_from` to the Telegram user who created the bot — something manual
  setup cannot do, since a pasted token identifies nobody.
- Manual token entry stays available behind "Set up manually" and writes the
  same configuration. The default path never shows a user a token.
- No secret ships in the APK. The endpoint is a build-time
  `--dart-define=POCKETCLAW_ONBOARDING_BASE_URL` that defaults to empty and
  must be HTTPS; an unconfigured build says automatic setup is unavailable
  instead of guessing an endpoint or reaching for someone else's.
- Tests: 62 service tests across six Go packages, all against a fake Telegram,
  none needing a real bot or a production secret; 50 new Flutter tests, 77
  total. Core regression 92 packages ok, `pnpm lint` clean, `flutter analyze`
  clean.
- End-to-end physical verification is **blocked on operator setup**: the real
  PocketClaw manager bot does not exist yet. No credentials were invented.

## 2026-08-25 — Source migration and OpenCode completion PASS on device

- Physical Android device test of
  `588bbec144fe0c84b8429f4f053a73b44b9b3e8d9f24e31dab04b2165ff3a90b` returned
  PASS across all 18 checks. That APK is now the verified reference artifact,
  superseding Milestone C's `b3dd892b...bce569b`. The never-tested OpenCode
  completion APK `785ccd94...56058c6` is retired; its functionality ships in
  the verified artifact.
- The logs check passed on device: no developer absolute paths on the
  user-facing Logs screen. That is the on-device half of the `-trimpath` fix —
  the build-time assertion proved the strings were gone from the binaries, and
  this proves nothing surfaces them to a user.
- Skill Hub search and Fetch Models both passed, which is the end-to-end proof
  that the Android active-network DNS integration survived being rebuilt from
  a relocated source tree. Both fail closed without working DNS, so this is a
  behavioral result, not a string check.
- Both OpenCode providers passed Fetch Models and a real request/response, so
  per-model protocol routing is exercised live for the first time.
- Still open, and deliberately not closed by association: the OpenCode
  Anthropic Messages route sends both `X-API-Key` and a bearer header on an
  unverified assumption, and the device report does not name which model
  families were exercised. A `claude-*` inference is what settles it.
- Device-proven Core binaries: `libpicoclaw.so` 37,224,801
  `cb9b2cde...fb895818`; `libpicoclaw-web.so` 24,641,889 `b6b356f7...656db9ba5`.
- `feature/provider-catalog` is verified and not merged. Merging into `develop`
  and tagging the closure point awaits explicit instruction, and `main` needs
  its own. No new feature was started.

## 2026-08-25 — Self-contained source migration

- PocketClaw now builds entirely from its own repository. The Core source is
  vendored at `core/src/` (1,372 files, 16 MB) and is the canonical build
  source. No build script, Makefile target, or Gradle task reads
  `/home/lordegypt/PocketCLaw/.upstream/picoclaw-core-v0.3.1` any more. That
  checkout survives as a historical upstream-review reference.
- The vendored tree was not reconstructed by hand. It was copied from the
  reviewed working tree and then proved equal to upstream `v0.3.1` plus
  `core/pocketclaw-core-v0.3.1.patch` plus `pkg/androiddns/`, byte-for-byte.
  All previously required modifications were verified present: the Android
  active-network DNS integration and `PICOCLAW_DNS_SERVER`, the provider
  catalog extensions, the Gemini and OpenCode discovery branches, OpenCode
  Zen/Go with per-model protocol routing, the generic Responses provider, the
  opt-in Anthropic Messages bearer header, the MQTT `/pocketclaw` default, the
  seeded workspace, and the user-facing wording.
- Added `core/build-android-arm64.sh` as the canonical Core build. It builds
  both binaries from `core/src` through the root Makefile targets, installs
  them into `jniLibs`, and prints sizes and hashes.
- Added `-trimpath` to the four Android arm64 `go build` lines. This fixed a
  real leak, not a hypothetical one: the previously shipped `libpicoclaw.so`
  carried 2,501 absolute `/home/lordegypt/...` paths and `libpicoclaw-web.so`
  carried 1,346, all reachable from the user-facing Logs screen. Both now carry
  zero, and the build script fails if that regresses.
- Added `core/verify-no-external-source.sh`, which renames the external
  checkout out of the way, runs the full Core build, and restores it. It
  passed: the Core builds with that directory unavailable.
- Rewrote `core/regen-upstream-patch.sh` and regenerated the provenance patch.
  Two defects were found and fixed while doing so. The patch was claiming
  PocketClaw had authored two unmodified upstream files, because upstream's
  unanchored `onboard` ignore rule kept them out of the baseline commit; and
  the file list now comes from git, so build outputs under `core/src` cannot
  leak into the patch. Upstream `v0.3.1` plus the regenerated patch now
  reproduces `core/src/` exactly — 52 changed files, 13 of them new and
  PocketClaw-authored.
- Excluded two upstream paths from the vendored tree: `assets/` (13 MB of
  README screenshots and marketing GIFs, no build role) and
  `pkg/seahorse/.omc/` (an upstream developer's tool-state file, committed by
  accident, which leaks an upstream contributor's home directory path).
- Anchored upstream's bare `onboard` ignore rule to `/onboard` in
  `core/src/.gitignore`. Unanchored, it matched at every depth and would have
  silently dropped the four files under `cmd/picoclaw/internal/onboard/`, two
  of which carry PocketClaw changes.
- Moved the 3.3 GB Go build and module caches out of the external checkout to
  `/home/lordegypt/PocketCLaw/.tooling/go/`, next to the JDK, Flutter, pnpm,
  and Android SDK. They are toolchain, not source, and they were the second
  hidden reason that directory was a build prerequisite. Nothing was deleted.
- `core/README.md` is rewritten as the authoritative Core guide: source
  location, upstream origin, the modification list, build procedure and exact
  release flags, jniLibs packaging, hash verification, the release guard and
  its `.cxx` recovery procedure, patch regeneration, and the upstream review
  model.
- Validation: Go suites 92 packages ok / 0 failed; frontend `vitest` 28 passed;
  `pnpm lint` clean; `flutter analyze` no issues; `flutter test` 27 passed. The
  arm64 release APK built through the canonical Gradle path and the release
  guard verified `libdartjni.so`, `libpicoclaw.so`, and `libpicoclaw-web.so`.
- No feature work was started. This is a source-of-truth and reproducibility
  migration, and the resulting APK needs a physical regression test.

## 2026-08-25 — Milestone C device PASS, plus the OpenCode completion

- Physical Android device test of
  `b3dd892bdea86e8dfe7d1c2eb87e89f4e2832b1d1dbe39fc1decf20dabce569b` returned
  PASS. That APK is now the verified reference artifact, superseding
  Milestone B's `ba4f067d...70f70a4b8`. Still not merged: the OpenCode
  completion below lands on the same branch and needs its own device test.
- Added the OpenCode Zen (`https://opencode.ai/zen/v1`) and OpenCode Go
  (`https://opencode.ai/zen/go/v1`) presets, using the official endpoints the
  user verified. Both require an API key, both support Fetch Models against
  `{base}/models`, and both keep the base URL hidden in the normal flow.
- These are mixed-protocol gateways, so they are not modelled as plain
  OpenAI-compatible providers. One base URL and one key front OpenAI Responses,
  OpenAI-compatible chat completions, and Anthropic Messages, and the protocol
  is a property of the selected model. All routing lives in
  `pkg/providers/opencode_routing.go`: `ClassifyOpenCodeModel` matches the
  longest model-ID family prefix and returns both a protocol and whether the
  match was known. No model-name conditional was added anywhere else.
- Routing is family-based rather than an enumerated model list, because
  OpenCode changes its lineup frequently and Fetch Models already returns the
  authoritative live list. `gpt-*`/`o*`/`codex*` route to Responses, `claude-*`
  to Messages, and `kimi-*`/`deepseek-*`/`glm-*`/`qwen-*`/`grok-*` and friends
  to chat completions.
- Built the generic Responses provider the Core was missing
  (`pkg/providers/openai_responses`). The two existing Responses paths were not
  reusable: Azure hardcodes its deployment path, and the Codex provider
  hardcodes the ChatGPT backend plus Codex headers and instructions. The new
  package reuses the shared `openai_responses_common` translation and adds only
  transport.
- Extended `anthropic_messages` with an opt-in `WithBearerAuth()` so the
  OpenCode Messages route sends both `X-API-Key` and `Authorization: Bearer`
  with the one OpenCode account key. It is off by default, so Anthropic's own
  endpoint and the existing `anthropic-messages` and `alibaba-coding-anthropic`
  presets are unchanged. Which form OpenCode's Messages surface actually wants
  is the single assumption only a device can settle.
- Model IDs go out bare. The `opencode-go/` style CLI namespace prefix is
  stripped before the request is built and never persisted into the wire model.
- An unrecognized model stays configurable and is still attempted, falling back
  to chat completions, with a warning naming the model, the fallback protocol,
  and what a 404 would imply. The warning never contains the API key.
- Added a catalog-wide invariant test: every HTTP-API provider that can drive a
  chat model must construct from a plain key-plus-base configuration, with a
  floor of 30 providers exercised so the assertion cannot become vacuous. This
  is the general form of the rule the earlier per-preset test only spot-checked.
- Validation: `flutter analyze` clean; 27/27 Flutter tests; 28 frontend tests;
  new Go tests covering OpenCode routing and catalog metadata, the Responses
  transport, the Messages bearer/base-path behavior, and discovery for both
  endpoints; full Go suites for providers, config, web/backend/api, androiddns,
  mqtt, onboard, and commands all pass; frontend `tsc -b` and `pnpm lint` clean.
- Every new test that can touch a credential asserts the key never appears in
  an error, and the routing warning was inspected in real log output.
- Rebuilt both Core binaries through the documented Makefile targets. Stripped,
  0 debug sections, `PICOCLAW_DNS_SERVER` present, launcher still 0 `PicoClaw` /
  0 `Sipeed` / 31 `PocketClaw`:
  `libpicoclaw.so` 37,421,409 `e48e8af0...12e78938`;
  `libpicoclaw-web.so` 24,772,961 `5faaf82c...c383fe2abf`.
- One build note worth keeping: running `:app:packageRelease` without
  `-Ptarget-platform=android-arm64` produced a 50 MB universal APK. Caught on
  the size check and rebuilt on the canonical path. The flag is not optional,
  and APK size is the cheapest signal that it was dropped.
- OpenCode completion APK: 34,129,765 bytes, SHA-256
  `785ccd94cfa351ee2996ac340f9a55e828a0c8f736bec67a3edac906a56058c6`,
  `com.lord1egypt.pocketclaw` 0.1.3 (code 3), label PocketClaw, release guard
  passed. NOT merged and NOT physically verified.
- No Telegram work, no UI changes beyond the two presets flowing through the
  existing picker, no release-pipeline changes, no dependency upgrades, and the
  `libdartjni` guard is untouched.

## 2026-08-25 — Phase 2 Milestone C: Provider Catalog + Easy API-Key Setup

Implementation complete on `feature/provider-catalog`, branched from the
verified `develop` @ `14e6991`. Not merged; physical-device testing is the gate.

- Audited the provider architecture before changing anything and recorded it in
  `docs/PROVIDER_ARCHITECTURE.md`. Two findings shaped the work: all AI provider
  configuration lives in the Core web console, not in Flutter, and the provider
  catalog is already backend-owned by `pkg/providers`, so Milestone C extends it
  rather than building a competing catalog.
- Catalog: added `category` and `documentation_url` to `ModelProviderOption` and
  classified all 42 entries as cloud, local, managed, custom, or speech. Added
  the xAI (`https://api.x.ai/v1`), Together AI (`https://api.together.xyz/v1`),
  Fireworks AI (`https://api.fireworks.ai/inference/v1`), and Custom
  OpenAI-Compatible presets, each also registered in the OpenAI-compatible arm of
  `CreateProviderFromConfig`. A catalog-only addition would have saved cleanly
  and then failed at request time with `unknown protocol`.
- Deferred OpenCode Zen and OpenCode GO: their base URL, auth header, and model
  listing endpoint could not be established accurately, and a guessed preset is
  worse than none. Both work today through Custom OpenAI-Compatible.
- Gemini model discovery now works. It needed a dedicated fetch branch, not just
  the `supports_fetch` flag: the shared path sends `Authorization: Bearer` while
  Gemini authenticates with `X-Goog-Api-Key` and returns
  `{"models":[{"name":"models/<id>"}]}`. The branch selects by base URL, so a
  `/openai` compatibility base keeps using Bearer and the standard shape.
  URL construction stays base-relative; a custom Gemini proxy path is unaffected.
- Broadened model-fetch error classification to distinguish invalid key /
  unauthorized, rate limited, DNS and network failure, provider unavailable, a
  missing listing endpoint, and a malformed response. A test asserts no error
  path echoes the API key.
- UX: the Add Model form became a two-step Add Provider flow — choose a provider
  from a searchable, category-grouped card list, paste an API key, fetch or type
  a model, save. The alias is derived from the model ID instead of being the
  first required field. Base URL, alias, and optional keys moved into Advanced;
  local and custom providers keep a visible base URL with an Android-specific
  hint that `localhost` means the phone. Keyless providers are not asked for a
  key. Saving never requires a successful fetch.
- Backward compatibility: stored provider, model ID, and custom base URL are
  untouched. A base that differs from the preset is now labeled as an override
  in the edit sheet rather than silently presented as the default.
- Removed runtime provider-logo fetching from `cdn.simpleicons.org` and Google's
  favicon service. Those requests disclosed which providers a user had
  configured and broke when offline. Provider marks render locally; verified 0
  occurrences of either host in the built binary.
- Localization: new keys added across all five web-console locales, with English
  and Chinese translated and the rest carrying English fallbacks. URLs and model
  IDs are pinned LTR and the picker uses logical properties.
- API key storage is deliberately unchanged. Transport and UI exposure are
  already sound (masked on GET, preserved on PUT when omitted), but at rest on
  Android keys are plaintext because Core encryption needs
  `PICOCLAW_KEY_PASSPHRASE` and an SSH key that no device has. Android Keystore
  is recorded as its own controlled milestone.
- Validation: `flutter analyze` clean; 27/27 Flutter tests; new Go tests (8
  catalog, 5 model discovery) plus full suites for providers, config,
  web/backend/api, androiddns, mqtt, onboard, commands, and agent all pass;
  22 new frontend tests on a newly added vitest runner; frontend `tsc -b` and
  `pnpm lint` clean.
- Rebuilt both Core binaries through the documented Makefile targets. Stripped,
  0 debug sections, `PICOCLAW_DNS_SERVER` present, and the launcher still has
  0 `PicoClaw` / 0 `Sipeed` / 31 `PocketClaw` strings:
  `libpicoclaw.so` 37,421,409 `cbe568af...5556468a`;
  `libpicoclaw-web.so` 24,772,961 `86e53457...0cb0a4cd`.
- Built through the canonical arm64 Gradle path; the release guard printed its
  verification line for `libdartjni.so`, `libpicoclaw.so`, and
  `libpicoclaw-web.so`. `libdartjni.so` is byte-identical to Milestone B.
- Milestone C APK: 34,123,401 bytes, SHA-256
  `b3dd892bdea86e8dfe7d1c2eb87e89f4e2832b1d1dbe39fc1decf20dabce569b`,
  `com.lord1egypt.pocketclaw` 0.1.3 (code 3), label PocketClaw. NOT merged and
  NOT physically verified. The Milestone B APK `ba4f067d...70f70a4b8` remains
  the verified reference artifact.
- Telegram QR/deep-link onboarding was deliberately not started; it is the next
  milestone. No release hardening or obfuscation was enabled.

## 2026-08-25 — Phase 2 Milestone B COMPLETE and physically verified

- Physical Android device test of `ba4f067df9811bd0e4af713343bdba632abbf96a41e3a5b47cf154740f70a4b8`
  returned PASS across every check: install, app launch / first frame, no black
  screen, Gateway/Core lifecycle, navigation, PocketClaw branding, workspace
  path, QR/access page, Skill Hub, provider/model flow, and no abnormal
  slowdown.
- This APK is now the verified reference artifact, superseding
  `2717f32e...0278a1de4`.
- Phase 2 Milestone B is CLOSED: independent product identity, the black-screen
  build-pipeline fix and its permanent release guard, full user-facing
  debranding, and the two branding edge cases.
- Merged `recovery/pocketclaw-clean-debrand` into `develop` with a
  non-fast-forward merge so the recovery history stays intact and auditable.
  `main` is intentionally untouched.
- Tagged the closure point as `phase2-milestone-b`.
- Milestone C is NOT started and requires explicit authorization.

## 2026-08-25 — Phase 2 Milestone B final cleanup

- Closed both remaining branding edge cases. No new features; Milestone C not
  started; no dependency, Flutter, Gradle, AGP, or Kotlin changes.
- MQTT: the Core default topic prefix is now `/pocketclaw`, exposed as
  `mqtt.DefaultTopicPrefix`. `topicPrefix()` substitutes the default only for an
  empty value, so an explicitly configured prefix — including the legacy
  `/picoclaw` — is preserved and broker-side topics are never rewritten. The
  frontend topic preview, placeholder, and the localized hint in all five
  locales were updated together. New tests in
  `pkg/channels/mqtt/topic_prefix_test.go` cover fresh default, explicit legacy
  prefix, custom prefix, and normalization.
- Seeded workspace: `skills/picoclaw-agent` is no longer written into a fresh
  workspace. It joins the existing `AGENTS.md` / `IDENTITY.md` exclusions via a
  named `unseededTemplates` list. It was not renamed, because it documents the
  real upstream CLI. Seeding only writes files, so existing user copies survive;
  three new tests in `cmd/picoclaw/internal/onboard/helpers_test.go` cover the
  exclusion, the surviving user copy, and the path matcher.
- Retained factual third-party hardware references (Sipeed, LicheeRV Nano,
  MaixCAM, NanoKVM) in the `hardware` skill rather than falsifying documentation
  for a zero string count. Full classification in `docs/BRANDING_AUDIT.md`.
- Rebuilt both Core binaries through the documented Makefile targets. Stripped,
  0 debug sections, `PICOCLAW_DNS_SERVER` verified present:
  `libpicoclaw.so` 37,421,409 `eb895f08...40bd9c88`;
  `libpicoclaw-web.so` 24,772,961 `6d282df0...1195a5a3`.
  The built frontend contains 0 `PicoClaw` and 0 `/picoclaw` strings.
- Validation: `flutter analyze` clean; 27/27 Flutter tests; Go tests pass for
  `pkg/channels/mqtt`, `cmd/picoclaw/internal/onboard`, `pkg/androiddns`,
  `web/backend/api`, `pkg/commands`; frontend `pnpm lint` clean.
- Built through the canonical arm64 Gradle path; the release guard passed for
  `libdartjni.so`, `libpicoclaw.so`, and `libpicoclaw-web.so`.
- Cleanup APK: 34,119,477 bytes, SHA-256 `ba4f067df9811bd0e4af713343bdba632abbf96a41e3a5b47cf154740f70a4b8`.
  Not merged; physical-device approval is the merge gate.

## 2026-08-25 — Black-screen incident RESOLVED; Stage B physically verified

- Physical Android device test of `2717f32e9580cd5b5ea5da70b2cb9fcf13f6f14451423addcb5686e0278a1de4`
  returned PASS across the board: install, app launch, Flutter first frame,
  Gateway/Core startup, navigation, PocketClaw branding, PocketClaw workspace
  path, and the QR/access page, with no abnormal device slowdown.
- The black-screen incident is CLOSED. The missing `lib/arm64-v8a/libdartjni.so`
  diagnosis and the build-pipeline fix are physically confirmed.
- This APK is now the reference physically verified PocketClaw artifact.
- Retained deliberately and not to be removed: the `packageRelease` guard over
  `libdartjni.so`, `libpicoclaw.so`, and `libpicoclaw-web.so`, and the canonical
  arm64 release command
  `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64`.
- Still open, awaiting a product decision: the MQTT `/picoclaw` topic prefix and
  the bundled `picoclaw-agent`/`hardware` seeded skills — see
  `docs/BRANDING_AUDIT.md`.
- Nothing merged to `develop` or `main`.

## 2026-08-25 — Release build guard and user-facing debranding

- Physical device confirmed the black-screen fix: the Stage A replacement APK
  `d8742534...d7e405` launches and reaches a usable Flutter first frame.
- Added a `packageRelease` guard that fails the build when the APK is missing
  `lib/arm64-v8a/libdartjni.so`, `libpicoclaw.so`, or `libpicoclaw-web.so`.
  It printed the verification line on this build.
- Debranded every normal user-facing surface. The full classification, with the
  retained legal and internal-compatibility occurrences and the two open product
  decisions, is in `docs/BRANDING_AUDIT.md`.
- Rebuilt the embedded web runtime from source, not by patching the binary.
  `libpicoclaw-web.so` contains 0 `PicoClaw`/`Sipeed` strings and 31
  `PocketClaw` strings.
- Rebuilt the gateway too, because the assistant identity (`/start` reply and
  the seeded `AGENT.md`/`SOUL.md`) lives in `libpicoclaw.so`. Verified the
  Android DNS integration survived: `PICOCLAW_DNS_SERVER` is still present.
- Recorded the Core source diff and the exact rebuild commands in `core/`.
- Android workspace default for fresh installs is now `Download/pocketclaw`;
  existing `Download/picoclaw` data is untouched and nothing migrates at startup.
- Removed three stale generated `lib/l10n/app_localizations*.dart` copies that
  still carried the `PicoClaw UI` title. `l10n.yaml` generates into
  `lib/src/generated/l10n`, so they were dead files.
- Validation: `flutter analyze` clean; 27/27 Flutter tests; Go tests pass for
  `pkg/androiddns`, `web/backend/api`, and `pkg/commands`; frontend `pnpm lint`
  clean.
- Candidate APK: `build/app/outputs/apk/release/app-release.apk`,
  34,119,837 bytes, SHA-256
  `2717f32e9580cd5b5ea5da70b2cb9fcf13f6f14451423addcb5686e0278a1de4`.
  Not merged to `develop`; awaiting physical test.

## 2026-08-25 — Black-screen root cause: missing arm64 `libdartjni.so`

- Diagnosed the persistent PocketClaw black screen by differential forensics on
  build artifacts and the AGP/CMake configure caches. It is a build-environment
  defect, not a Stage A source defect.
- Root cause: the `jni` package's CMake configure for `arm64-v8a` failed inside
  this workspace on 2026-08-24 02:52 with a transient filesystem error and was
  cached as a valid-but-empty configure in
  `~/.pub-cache/hosted/pub.dev/jni-1.0.3/android/.cxx/RelWithDebInfo/6n1p6673/`.
  Every subsequent build from `/home/lordegypt/PocketClaw-App` therefore
  packaged **no** `lib/arm64-v8a/libdartjni.so`.
- `JniPlugin`'s static initializer calls `System.loadLibrary("dartjni")`, and
  `GeneratedPluginRegistrant` catches only `Exception`; the resulting `Error`
  escapes during `FlutterActivity.onCreate`, so Flutter never renders a frame.
- The physically working APK `45be7269...ebe9e` was built from a different
  directory (`/tmp/pocketclaw-runtime-fix`), which produced a separate cache
  (`.cxx/RelWithDebInfo/4l131246`) whose arm64-v8a configure succeeded. That is
  the only material difference between the working and failing artifacts.
- Fix: purged the poisoned `.cxx` cache and rebuilt with the documented
  `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64` command and
  the pinned Flutter 3.47.1 / Dart 3.13.1 / Java 17 / AGP 8.11.1 toolchain.
  No source change was required.
- Replacement APK: `build/app/outputs/apk/release/app-release.apk` (also copied
  to `build/app/outputs/flutter-apk/app-release.apk`), 34,119,437 bytes,
  SHA-256 `d87425344cca526afd8ef3eca41ef3648791593e69016a463aeee86693d7e405`.
  Verified: `lib/arm64-v8a/libdartjni.so` present (131,248 bytes, AArch64),
  package `com.lord1egypt.pocketclaw`, label PocketClaw, launchable
  `com.lord1egypt.pocketclaw.MainActivity`, pinned Core hashes unchanged,
  drawable-backed splash layer intact.
- `flutter analyze` clean; all 27 Flutter tests pass.
- Closed hypotheses: stale `libpicoclaw-web.so`, workspace migration, splash
  resources, and Stage A wording changes are all excluded by this evidence.

## 2026-08-24 — Milestone B runtime-regression fix (physical retest pending)

- Recorded Milestone B physical validation as BLOCKED after the branded APK
  installed and launched but remained on a black screen with no usable Flutter
  UI.
- Root-caused the regression to invalid Android splash `layer-list` syntax:
  both launch backgrounds used `android:color` on an item instead of supplying
  a drawable. Replaced the invalid item with the named
  `@color/pocketclaw_splash_background` drawable reference.
- Added `test/unit/android_launch_theme_resource_test.dart` to reject the
  invalid color-only layer in both resource variants.
- Passed `flutter analyze`, all 29 Flutter tests, focused Core
  `pkg/androiddns`/`web/backend/api` tests, debug APK build, and release APK
  build. Inspected the compiled release resource and verified package, label,
  MainActivity, and unchanged pinned Core hashes.
- Replacement APK: `build/app/outputs/flutter-apk/app-release.apk`,
  33,308,653 bytes, SHA-256
  `45be7269af920df4a36eb4eb37171770bbcfa242ed7c071da28874d9c27ebe9e`.
- No merge to `develop`; physical-device retest is required before Milestone B
  can be approved.

## 2026-08-24 — Phase 2 Milestone B identity foundation

- Started `feature/pocketclaw-identity` from the validated `develop` state.
- Implemented the PocketClaw product-visible naming and planned independent
  Android package identity `com.lord1egypt.pocketclaw`, while preserving
  PicoClaw Core integration identifiers and compatible external workspace path.
- Added branding, package-migration, asset, image-generation, and Skill Hub
  regression documentation; established centralized Material 3 design tokens.
- Selected the second original PocketClaw mark as the primary visual direction
  and created matching Android launcher/adaptive, splash, and monochrome
  notification treatments. No cartoon lobster/mascot was adopted.
- Recorded physical Skill Hub/ClawHub success after the DNS fix: the prior
  registry-unavailable symptom is resolved by Android DNS, not a separate hub
  defect.
- Passed `flutter analyze`, all 28 Flutter tests, and focused pinned-Core
  Android DNS/model API tests; built the new PocketClaw APK.
- Inspected the APK: package `com.lord1egypt.pocketclaw`, label PocketClaw,
  branding resources, and pinned Core hashes are correct. The release build
  had no Firebase app ID/API key/project ID and cleaned generated resources.
- Committed as `34b0f6b` and pushed `feature/pocketclaw-identity` to private
  `origin`; `develop` and `main` remain unchanged pending physical approval.

## 2026-08-24 — Phase 2 Milestone A bootstrap

- Created the independent PocketClaw Android/Flutter foundation directory.
- Selectively adapted the reviewed FUI Android, Flutter, test, asset, and tool
  foundation without copying upstream Git history, build outputs, or caches.
- Preserved the pinned Core `v0.3.1` replacement binaries, Android
  active-network DNS integration, and optional feedback behavior.
- Added provenance, upstream-tracking, decision, handoff, task, and
  third-party-license records.
- Passed `flutter analyze`, all 28 Flutter tests, and focused Core Android
  DNS/model API tests.
- Built and inspected the independent arm64 foundation APK; it retains the
  pinned Core hashes and has no Firebase app ID, API key, or project ID.
- Created private GitHub repository `Lord1Egypt/PocketClaw` and pushed initial
  commit `950d4a3` to both `main` and `develop`.

## 2026-08-26 — CODEX SOL HANDOFF — PRE-RELEASE FIX

- Preserved and pushed the exact `e5b88ff` rollback checkpoint as branch
  `checkpoint/pre-codex-sol-prerelease-fix` and annotated tag
  `pre-codex-sol-prerelease-fix-20260826`.
- Replaced duplicated native Telegram status with a neutral shortcut to Core's
  authoritative `/channels/telegram` page; card tap cannot start pairing.
- Added the authenticated loopback Android→Core credential-write boundary so
  managed/manual setup updates Core's split secure config and failed reconnect
  leaves the old bot intact.
- Fixed Telegram request completion: bounded HTTP deadline, safe correlation,
  independent same-session FIFO requests, synchronous final delivery, error
  propagation, edit→send fallback, and terminal placeholder cleanup. Added
  deterministic empty-provider/idle/sequential/close/failure coverage.
- Audited tool execution, Android service ownership, and the recent log queue.
  No `gh`-specific stall or log-queue causal link was found; physical
  background/locked validation remains pending.
- Established the intended seven-skill fresh baseline, added non-destructive
  startup repair, preserved existing `picoclaw-agent`, and proved `gh` import
  adds without replacing.
- Preserved basename caller and exactly-once logs; neutralized the PID warning;
  added a plain Android banner and one Unicode-preserving terminal sanitizer
  shared by Logs and Export.
- Regenerated the 93-file Core provenance patch and updated intentional
  divergence tracking.
- Passed `flutter analyze`, 94 Flutter tests, 36 frontend tests, `tsc`, lint,
  and required Go suites. Canonical Core build passed `-trimpath` with zero
  developer paths.
- Built one successful candidate after a compile-only Kotlin getter clash was
  caught and fixed. APK: `build/app/outputs/apk/release/app-release.apk`,
  34,239,649 bytes,
  `f663d25a2fffb0ce969ad4a9ce3405c1e563b6263c7af37e90768eef471c621d`.
  Guard, package/version/SDK, endpoint, native hashes, and secret scans pass.
- No merge, release, tag movement, or `main` change. Physical testing is
  pending and release remains blocked.

## 2026-08-27 — Web Console log parity and identifier visibility

- Traced Core Web Console startup boxes to the captured Core CLI's Unicode
  block-art no-color banner and a React Logs renderer that interpreted only SGR
  while the backend ring stored raw child output.
- Added a fixture-backed user-visible plain-text contract at the Web log ring,
  the existing native/export sanitizer, and an idempotent real-page browser
  guard. Removed the terminal-style renderer; preserved Arabic, emoji,
  punctuation, and `53.616µs`.
- Captured gateway launches now force `--no-color`; no-color startup is one
  `PocketClaw` line. The startup event no longer prints an executable or
  `libpicoclaw.so` path.
- Successful exact `GET /pico/ws` 101/2xx events no longer enter normal DEBUG
  history. Failures/unexpected methods remain visible as `/internal realtime
  connection`; the endpoint and library identifiers were not renamed.
- Fixed the provenance generator to preserve tracked deletions when building
  its rsync file list, then regenerated the 108-file Core patch.
- Passed Flutter analyze and 99 tests; frontend 37 tests, TypeScript, and lint;
  tagged Go logger/gateway/API/middleware/CLI suites; canonical Core build with
  zero developer paths; and the permanent APK payload guard.
- Built candidate `3e138b4a53a0389b76dbef045649d906fe2db785cd2af606826f7a0f87170adc`
  (34,241,381 bytes). Core hashes are `c9c348e9...68d236e` and
  `c891ca03...d34840a`. Physical validation is pending; no merge/release/main
  change was made.

## 2026-08-27 — Final enabled-channel log brand micro-fix

- Recorded physical PASS for the Web terminal cleanup, PocketClaw banner,
  UTF-8/`µs`, hidden internal library path, and restored 8/8 skills / 17 tools
  on commit `3611ca1`.
- Traced the remaining `[telegram pico]` summary entry to Core's internal
  singleton Web Console WebSocket/media channel ID.
- Preserved the internal `pico` config/factory/channel/routes/protocol and
  mapped only its copied startup/reload display name to `pocketclaw`.
- Added a focused regression proving the exact output, unchanged internal
  input, and no substring/global replacement. Relevant tagged Go suites pass.
- Regenerated the 110-file Core patch, rebuilt both zero-path Core libraries,
  and built guarded ARM64 APK `aab3c565...25b3582` (34,241,857 bytes).
  Physical confirmation of the final label is pending; no merge/release/main
  change was made.

## 2026-08-27 — Final user-visible caller brand fix

- Mapped only structured user-visible logger component `pico` to `realtime`
  and caller basename `pico.go` to `realtime.go`, preserving exact line numbers.
- Applied the canonical contract at Web `LogBuffer`, native/export sanitizer,
  and React legacy/raw guard. Internal packages, filenames, channel/config IDs,
  routes, and protocol were not renamed.
- Added exact and substring-negative fixtures plus direct Go, stored/export,
  and real DOM assertions. Flutter analyze/99 tests, frontend 37/tsc/lint, and
  relevant tagged Go suites pass.
- Regenerated the 110-file Core patch, rebuilt zero-path Core libraries, and
  built guarded APK `1eeca7c9...ad089f7` (34,242,865 bytes). Physical device is
  the final gate; no merge/release/main change was made.

## 2026-08-27 — Web Console Logs viewport stability micro-pass

- Traced physical Web-only jitter to a passive post-paint bottom snap and a
  content-resize-driven JavaScript hard-wrap loop that could rewrite long rows.
- Moved conditional bottom following to `useLayoutEffect`; scrolled-up users
  receive no scroll writes and repeated empty polls do not change `scrollTop`.
- Added stable `run_id:absolute_offset` keys and memoized rows. Removed the
  whole-content `ResizeObserver`, manual `wrap-ansi` hard wrapping, and its
  now-unused direct dependency; CSS wraps the unchanged sanitized string.
- Added real Logs page DOM cases for bottom following, scrolled-up preservation,
  long Telegram-style row node/text stability, and no-new-log rerenders.
- Passed frontend 41/41, TypeScript, lint, relevant tagged Go API/middleware,
  and Native/Export log regressions. Regenerated 113-file Core provenance,
  rebuilt zero-path libraries, and passed the permanent APK payload guard.
- Built APK `be5d7cbb...5070fc96` (34,239,873 bytes), with Core hashes
  `715cd790...6143cf1` and `e3930ae2...f5f5da14`. Physical validation is
  pending; no merge/release/main change was made.

## 2026-08-27 — Telegram Web-log credential redaction

- Confirmed Telego's full Bot API URL was partially masked before stdout and
  Web backend storage; no complete token was persisted through this path, but
  the retained bot ID and secret prefix/suffix were user-visible in Web Logs.
- Replaced partial masking at the same pre-stdout third-party logger boundary
  with full credential and Authorization redaction. No credentials were read,
  rotated, or modified.
- Added Web pre-storage normalization to render Bot API URLs as
  `Telegram API call: <operation>` while preserving methods, failures, status,
  timeout, and latency. Added an idempotent React legacy/raw guard.
- Added synthetic regressions for standard/arbitrary calls, success/failure,
  timeout, encoded/bare/Authorization forms, Web ring storage, public metadata,
  and the real Logs DOM. Native/Export source stayed unchanged.
- Passed relevant tagged Go suites, frontend 42/42/tsc/lint, and unchanged
  Native/Export 7/7 regression. Regenerated 115-file provenance, rebuilt both
  zero-path Core libraries, and passed the permanent APK guard.
- Built APK `8257e9f0...7c7050fe` (34,240,641 bytes), with Core hashes
  `0e914550...8b555e9` and `7d7b254b...d9898c7`. Physical validation is
  pending; no merge/release/main change was made.

## 2026-08-27 — Final legacy brand visibility sweep

- Traced physical Web log leaks to exact structured ChannelPico fields, the
  internal `/pico/` webhook field, Pico protocol lifecycle wording, and the
  `.picoclaw.pid` compatibility path.
- Added display-only normalization before Web `LogBuffer` storage plus the
  idempotent React guard. Exact fields now display `pocketclaw`; protocol and
  reasoning messages use realtime wording; the PID success line is semantic.
- Preserved the complete security warning and genuine failures. Did not rename
  any channel/config ID, source package/file, route, PID file, library, env var,
  migration, or provenance identifier; no global/substring replacement exists.
- Added backend storage, representative startup, real Logs DOM, and negative
  substring regressions. Representative normal output has zero unintended
  legacy brand occurrences.
- Passed frontend 44/44, TypeScript, lint; relevant tagged Go logger/gateway/
  channels/Pico/Telegram/Skills/API/middleware/CLI suites; Flutter analyze and
  99 tests. Regenerated 115-file provenance and rebuilt zero-path Core.
- Built guarded APK `309f6d7a...a5f3030`, with Core hashes
  `49f89ae2...be656f` and `98f3fa9d...08bae`. Physical validation is pending;
  no merge/release/main change was made.

## 2026-08-27 — Telegram DEBUG final cleanup

- Proved Telego emitted valid `Err: [<nil>]` and PocketClaw's pre-stdout secret
  redactor preserved it; the Web pre-storage orphaned-CSI regex removed `[<n`
  and persisted malformed `il>]`.
- Narrowed orphaned CSI recovery to numeric/private-numeric suffixes across Go,
  React, and Dart. Exact successful Telego nil fields now display `Err: none`;
  ordinary angle brackets and multilingual Unicode remain unchanged and render
  only as safe text nodes.
- Suppressed exact DEBUG `getUpdates` request lines and successful empty
  responses before stdout, with idempotent storage/render guards. Failures,
  non-empty results, API errors, send/edit operations, and lifecycle events stay
  visible and credential-free.
- Added repeated-poll history, failure/non-empty/operation, token-negative,
  cross-surface angle text, ANSI, Unicode, and DOM-injection regressions.
- Passed frontend 45/45/tsc/lint, relevant tagged Go suites, Flutter analyze and
  101 tests. Regenerated 115-file provenance and rebuilt zero-path Core.
- Built guarded APK `2c00720a...2da27c` (34,243,585 bytes), with Core hashes
  `7c1d3918...38ef1f` and `1611b6e1...d09256`. Physical validation is pending;
  no merge/release/main change was made.

## 2026-08-28 — Agent DEBUG and Telegram payload privacy

- Confirmed the legacy name was in the actual freshly generated system prompt;
  PocketClaw defaults now identify as PocketClaw without rewriting custom
  prompts or internal/upstream compatibility names.
- Removed normal prompt previews, full LLM message/tool dumps, raw reasoning,
  and tool-argument previews. Added exact-field pre-writer redaction on a copy,
  preserving runtime session/routing values and useful lifecycle metadata.
- Traced raw Telegram PII/content to Telego `Response.String()` before stdout
  and Web storage. Telego now emits concise operation/status/count/type metadata
  before writers; backend, React and Dart guards cover historical/raw input.
- Added regressions for runtime-value immutability, fresh prompt identity,
  synthetic session/internal values, raw Agent payloads, Telegram IDs/profile/
  messages, arbitrary Bot API operations, failures, real Web DOM and export.
- Passed tagged relevant Go suites, frontend 46/46/tsc/lint, Flutter
  analyze/101. Regenerated 124-file provenance, rebuilt zero-path Core, verified
  the live endpoint and permanent payload guard.
- Built APK `46ca983a...2b908e` (34,251,141 bytes), with Core hashes
  `8ed15601...9be24a` and `21001004...5fc1d7`. Physical validation is pending;
  no merge/release/main change was made.

## 2026-08-29 — v0.2.0-rc1: owner authorization and live LAN Dashboard mode

- Owner authorization became server-derived on every surface. The internal
  realtime channel binds each inbound message to its authenticated connection
  and to a Core-owned owner principal, so payload fields and a stale or
  permissive on-disk allowlist can no longer choose the effective sender,
  session, or routing identity.
- Dashboard sessions moved from one process-wide cookie to a server-side store
  with issue/validate/revoke and a 24-hour lifetime (was 31 days). Logout now
  revokes server-side rather than only clearing the browser cookie, and the
  realtime WebSocket upgrade requires a same-origin request.
- Telegram fails closed unless exactly one paired numeric owner is configured.
  The Android bridge rejects a pairing without a numeric owner instead of
  writing an empty allowlist, and manual onboarding requires the numeric ID.
  Usernames are never a security identity.
- Credential generation fails closed when the platform CSPRNG is unavailable;
  the previous timestamp fallback for the realtime token is gone. The Android
  host's realtime credential is now a per-installation CSPRNG value kept in
  no-backup storage, replacing a constant compiled into the app.
- The managed Core gateway is pinned to loopback unconditionally. It no longer
  inherits the launcher's bind host, so exposing the Dashboard cannot expose
  Core on 18790 by any configuration or environment path.
- Public Mode applies live. A Dashboard listener supervisor rebinds only port
  18800 between loopback and wildcard while the Core process, session store,
  Telegram polling, and agent runtime keep running. It closes hijacked
  WebSocket connections belonging to the old bind, rolls back to the previous
  listener when the new bind fails, and persists the setting only after the
  bind succeeds. Android drives it over the authenticated loopback bridge, so
  OFF→ON and ON→OFF no longer need a manual service restart.
- The advertised LAN address now comes from an active Wi-Fi or Ethernet link.
  Cellular-only, link-local, loopback, and wildcard addresses are never offered
  as connect targets, and the QR falls back to an explicit "no LAN address"
  state instead of encoding an unreachable URL.
- Logging keeps the internal realtime route out of user-facing output by
  capturing the channel/path relationship before display names are normalized.
- Validation: `flutter analyze` clean, 114/114 Flutter tests; frontend 46/46,
  `tsc -b`, lint; the complete Go suite, `go build ./...`, and `go vet ./...`
  green under `-tags goolm,stdjson`. Core provenance regenerated to 141 files
  and reproduced byte for byte; zero developer paths; permanent APK guard pass.
- Both Core binaries were reproduced byte for byte from this source with their
  build timestamps pinned, proving the released native payload is the payload
  physically validated on device.
- Known pre-existing: `-race` on `web/backend/api` fails in
  `TestStartGatewayLocked_UsesReloadedConfigForBootSignature`, where the test's
  cleanup and the production monitor goroutine both call `cmd.Wait()`. It
  reproduces identically on `develop` at `8f861bc`. Not a production race.
