# PocketClaw Managed Runtime

The Managed Runtime gives the PocketClaw Agent a controlled, observable, verified
local tool environment on Android ARM64.

The agent names a tool. The runtime resolves that name to a concrete executable
through a versioned catalog, verifies it, and runs it under bounded execution
with a full structured lifecycle. The agent never sees a physical binary path,
so a prompt cannot redirect execution at a file of its choosing.

```
Agent
  ↓  runtime tool
Managed Runtime API      pkg/pcruntime.Manager
  ↓
Runtime Registry         pkg/pcruntime.Registry   (versioned catalog)
  ↓
Tool Resolver            measures the device, does not trust the catalog
  ↓
Verified Tool            SHA-256 + ABI for bundled payloads
  ↓
Bounded Process          argv, no shell; timeout, cancellation, output bounds
  ↓
stdout / stderr / exit status / diagnostics
```

## The constraint everything else follows from

PocketClaw targets Android SDK 36. Since API 29 an app may not `execve()` a file
in its own writable data directory, and `File.setExecutable(true)` does not
change that — the restriction is enforced on the app's SELinux domain, not by the
file mode. Loading executable code from a writable file is restricted the same
way.

An executable can therefore reach the device by exactly two routes, both
read-only to the app:

| Delivery  | Where it lives                          | Integrity                     |
|-----------|-----------------------------------------|-------------------------------|
| `system`  | `/system/bin`, shipped in the OS image  | the platform's, not ours      |
| `bundled` | the APK, unpacked into `nativeLibraryDir` | SHA-256 pinned at build time |

**There is no third route, and so there is no provisioning subsystem.** The
runtime contains no code that downloads a binary, and none that makes a file
executable. "No arbitrary binary installation" is not a policy this codebase
enforces; it is a thing the platform cannot do. `EnsureTool` reports; it never
acquires.

Any future provisioning abstraction is limited to **non-executable assets**
unless the platform model changes.

## Availability is measured, never asserted

The catalog says what PocketClaw supports. The device says what is present. A
`system` tool is resolved by probing `/system/bin`, `/system/xbin` and
`/vendor/bin`, because the toybox command set varies by Android version and by
vendor. Nothing in this runtime claims Linux compatibility that Android does not
actually provide.

Resolution results are cached per process and deduplicated by a per-tool lock, so
concurrent callers share one verification pass rather than each hashing the same
payload.

## Integrity, honestly labelled

A **bundled** payload is hashed at build time and checked against the on-device
file before its path is ever exposed. It is also checked as an ELF of the
declared ABI, so a payload built for the wrong machine is diagnosed rather than
failing later with a bare `ENOEXEC`. A payload that fails either check is never
activated and never yields an executable path.

A **system** binary carries `security_class: system` and
`verification_result: platform_owned`. PocketClaw does not pin a hash for it and
must not imply that it verified one: the OS owns the file and replaces it on
every system update. The catalog format refuses to record a checksum for a
system tool, so this cannot be overstated by accident.

## Storage

| Area                | Location                                  | Contents               |
|---------------------|-------------------------------------------|------------------------|
| Executables         | `nativeLibraryDir` and `/system/bin`      | read-only to the app   |
| Runtime metadata    | `filesDir/picoclaw/runtime`               | inventory, probe records |
| User workspace      | `/storage/emulated/0/Download/pocketclaw` | the user's own files   |

The layout refuses to start if runtime metadata would land inside the user
workspace: user Skills write freely there, and runtime state a Skill can rewrite
is worse than no runtime state at all. No executable is ever placed in writable
storage — on Android one could not be run from there anyway.

The Android Service exports `POCKETCLAW_RUNTIME_LIB_DIR` and
`POCKETCLAW_RUNTIME_DIR`. When the first is absent the payload directory is
derived from `PICOCLAW_BINARY`, which already points into `nativeLibraryDir`.

## Execution

Managed tools are executed by direct `argv`. The runtime never builds a command
line and never invokes `sh -c`, so pipes, redirection, globs and `$(...)` are not
interpreted — an argument containing shell metacharacters reaches the tool as
that literal string.

- **Environment** is *constructed* from an allowlist, not inherited, so provider
  keys and channel tokens held by the Core process cannot reach a child by
  accident. Callers may add variables; `PATH`, `LD_PRELOAD`, `LD_LIBRARY_PATH`,
  `LD_AUDIT` and the macOS `DYLD_*` equivalents are refused, because each of them
  changes which code the loader runs and would defeat the verification above.
- **Working directory** must be inside the user workspace, and defaults to its
  root. Runtime metadata storage is deliberately unreachable, so a managed tool
  cannot rewrite the runtime's own records.
- **Timeouts** come from the tool's declared profile — `quick` 15s,
  `standard` 2m, `extended` 10m. A caller may lower the budget and can never
  raise it, so no managed process can outlive the profile the catalog declares.
- **Cancellation** terminates the child's whole process group with `SIGTERM`,
  then `SIGKILL` after a two-second grace, and always waits for the child so it
  is reaped rather than left as a zombie. The group matters: killing only the
  direct child would leave a grandchild — a pipeline started by `tar` or `xargs`
  — running with nothing left to reap it.
- **Output** is bounded at the point of capture, per the tool's
  `max_output_bytes`, and truncation is marked inside the returned value so a
  truncated result cannot be mistaken for a complete one. `stderr` is captured
  and reported even on success.
- A **non-zero exit code** is the tool's answer, not a runtime failure. The
  operation status stays `completed` and the caller reads `ExitCode`.

## Observability

Every operation carries an `operation_id` and emits a started event and exactly
one terminal event, so an operation can never be left with no record of how it
ended. Events are logged under component `runtime`:

```
runtime.resolve.started    .completed  .failed
runtime.verify.started     .completed  .failed
runtime.provision.started  .unsupported
runtime.exec.queued  .started  .stdout  .stderr
                     .completed  .failed  .timeout  .cancelled
runtime.cleanup.started    .completed  .failed
runtime.probe.started      .completed
runtime.inventory.completed
```

Fields include `operation_id`, `tool`, `tool_version`, `runtime_version`,
`source`, `stage`, `duration_ms`, `exit_code`, `bytes_in`, `bytes_out`,
`timeout_ms`, `status`, `expected_sha256`, `observed_sha256` and
`verification_result`.

There is deliberately no `runtime.download.*` or `runtime.install.*` family. Those
stages do not exist — see the constraint above — and emitting events for a
pipeline that cannot run would misrepresent what the runtime does.
`runtime.provision.unsupported` is what a request for an unavailable tool
produces, carrying the reason and diagnostics.

### Redaction

Redaction runs inside the event emitter, before any writer sees a value, so a new
call site cannot forget to apply it. Argument vectors, environment maps and free
text are all covered: `Authorization` and `Cookie` headers, bearer credentials,
Telegram bot tokens, GitHub tokens, provider API keys, URL userinfo, and
assignments to secret-named variables.

Flag-directed redaction — blanking the element *after* `-u` or `--password` — is
scoped per tool on purpose. A password is often an ordinary-looking word that no
pattern can recognise, but the same short flags mean something harmless
elsewhere: applying curl's flag list globally would blank the filename in
`sort -u notes.txt` and the pattern in `grep -E '<expr>' file`, destroying the
diagnostics these logs exist to provide.

**Tool output is never persisted.** Only the byte counts of `stdout` and `stderr`
reach the log. The caller receives the real output in memory; the log does not,
because a tool's stdout is exactly where a fetched credential appears —
`gh auth token` prints one.

## The execution probe

`ProbeExecution` copies a harmless system binary into app-private writable
storage, tries to run it, records the verdict, and deletes the copy. The runtime
never relies on the answer — bundled payloads live in `nativeLibraryDir`
precisely so that they do not — but the assumption the whole delivery model rests
on should be visible in the Debug Logs of every real device instead of being
taken on trust. The verdict is `supported`, `blocked`, or `inconclusive`, always
with an explanation.

On the device that physically validated this runtime the verdict was
**`inconclusive`**: the probe could neither run its staged copy nor observe a
clean permission refusal. It reported that rather than guessing, which is the
behaviour to preserve. An inconclusive probe is not a licence to attempt writable
execution — on the same run the bundled jq payload executed from
`nativeLibraryDir`, which is the route the runtime actually uses.

## Runtime Pack v1

**Tier 1 is system-provided.** Android already ships toybox in `/system/bin`,
covering nearly the whole Tier 1 list. Bundling BusyBox would add a GPLv2
source-offer obligation and tens of megabytes to duplicate what the platform
already provides, so the catalog declares these as `system` and probes them.

Catalogued as `system`: `cat cp mv rm mkdir rmdir ls pwd touch chmod stat find
grep sed awk cut sort uniq head tail wc xargs tee which env printenv date sleep
timeout ps uname df du base64 sha256sum md5sum tar gzip gunzip`, plus the network
diagnostics Android plausibly provides: `ping traceroute ip netstat`.

### Physically observed on hardware, 2026-08-30

**43 of 44** catalog tools resolved as available on the tested ARM64 device.
`traceroute` was the one reported unavailable, correctly — availability is
measured per device, so a platform that does not ship a command says so instead
of failing later at exec. jq resolved, verified against its pinned checksum, and
executed from `nativeLibraryDir`.

**`jq` is the first bundled payload**, and exists to prove the packaging contract
end to end. jq 1.7.1 is cross-built from the official release tarball with the
Android NDK, packaged as `libpocketclaw-jq.so`, and exposed to the agent under
the logical name `jq`.

The `lib*.so` filename is not cosmetic: Android's package manager only unpacks
`lib/<abi>/*.so` entries into `nativeLibraryDir`, and `nativeLibraryDir` is the
only directory the app may execute from. A payload under any other name would
ship inside the APK and never be runnable. The catalog format enforces the naming
rule at build time rather than letting such a tool ship and fail on every device.

Not in this milestone, with reasons:

- **curl, wget, openssl** — no system binary exists and each needs an NDK
  cross-build with a TLS stack. Deferred as the highest-effort items.
- **git** — C, and it `exec`s helpers (`git-remote-https`, `git-http-fetch`) from
  a `libexec` layout that `nativeLibraryDir`'s flat `lib*.so` namespace cannot
  represent.
- **gh** — pure Go and easy to cross-compile, but roughly 40 MB and of little use
  without `git`.
- **dig, host, nslookup, ss** — Android does not ship them, and listing catalog
  entries that can only ever probe unavailable would be noise, not information.

## Skills are not evidence that a tool exists

A PocketClaw Skill is instructions and knowledge. It is not proof that a native
executable is present. The GitHub Skill asks the runtime for `gh`; the runtime
resolves it or reports accurately that it is unavailable. A Skill must never
assume a command exists because it appears in the Skill's own text.

## Adding a tool

1. For a bundled tool, add a build script under `runtime/` that pins the upstream
   source by SHA-256, builds in a fixed neutral directory so no developer path is
   baked into the binary, and installs the payload as `lib<name>.so` into
   `android/app/src/main/jniLibs/arm64-v8a/`.
2. Add the payload to `requiredArm64NativeLibraries` and to `keepDebugSymbols` in
   `android/app/build.gradle.kts`. Without the second, Gradle's strip rewrites the
   file and breaks the pinned checksum, and the tool resolves as a corrupt
   install on every device.
3. Add the catalog entry to `core/src/pkg/pcruntime/manifest.json` with the
   payload's checksum. `TestBundledPayloadsMatchTheirPinnedChecksums` fails until
   the catalog and the packaged payload agree.
4. Record the upstream licence in `THIRD_PARTY_NOTICES.md`.
