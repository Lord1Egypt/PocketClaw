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
| User workspace      | `Android/data/<package>/files/pocketclaw` (app-specific external storage; `filesDir/pocketclaw` without one) | the user's own files   |

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
diagnostics Android actually provides: `ping ip netstat`.

## Helper payloads

Some tools invoke auxiliary executables by a logical name the APK cannot use as
a filename. git looks its transport helper up as `git-remote-https` inside
`GIT_EXEC_PATH`, and Android's package manager only unpacks `lib/<abi>/*.so`, so
no packaged file can carry that name.

A catalog entry therefore declares its helpers:

```json
"helpers": [
  {"logical_name": "git-remote-http",  "library_name": "libpocketclaw-git-remote-http.so", "sha256": "..."},
  {"logical_name": "git-remote-https", "library_name": "libpocketclaw-git-remote-http.so", "sha256": "..."}
]
```

Several logical names may share one payload — `git-remote-http` and
`git-remote-https` are the same binary upstream — and the catalog is rejected at
build time if it pins two different checksums for the same payload. Every helper
is hashed and ABI-checked alongside the main payload, and a tool whose helper
fails verification resolves as *unavailable*. That matters: git with an
unverified transport helper would look installed and then fail at its first
`https://` URL, which is a far harder failure to read than "unavailable,
checksum mismatch".

### A tool must be able to find itself

git does **not** exec its transport helper directly. `get_helper()` builds the
argument vector `remote-https …` with `git_cmd = 1`, `prepare_git_cmd()` prepends
the literal string `git`, and `prepare_cmd()` then resolves that name — not the
helper's — through `locate_in_PATH`. `setup_path()` has meanwhile prepended
`GIT_EXEC_PATH` to `PATH`.

So a helper directory holding only `git-remote-http` and `git-remote-https` is
not enough: the lookup for `git` itself returns `ENOENT`, and `get_helper()`
converts exactly that errno into

```
fatal: unable to find remote helper for 'https'
```

which names the wrong thing entirely and sends you looking at TLS. The git
catalog entry therefore declares **`git` as one of its own helpers**, and
`TestGitDeclaresItselfAsAHelperSoItsOwnLookupSucceeds` fails if that is ever
removed. `gh` declares the same three entries, because it shells out to git.

The general rule: if a tool re-invokes itself by name, it needs an entry for
itself.

At execution time the runtime builds a directory of **symlinks** to the packaged
payloads and points `GIT_EXEC_PATH` at it. The distinction is the whole design:
a symlink is not an executable, so nothing is ever written into app storage and
run — the kernel resolves the link and executes the read-only packaged file.
Logical names are validated as filenames, never paths, so a catalog entry cannot
place a link outside that directory. Stale links are repaired rather than reused,
because an app update moves `nativeLibraryDir`.

**This works on Android, proven on hardware.** A physical ARM64 device cloned a
public GitHub repository over HTTPS through exactly this path on 2026-08-30. The
kernel resolves the link and executes the read-only packaged file, so the API 29+
restriction on executing writable app storage does not apply — a symlink is not
an executable. This is the mechanism that makes multi-executable tools such as
git possible on Android at all.

It does **not** license copying an executable into app storage, which the
platform still refuses. `ProbeExecution` keeps reporting `symlink_exec` so a
device that behaves differently says so in its own Debug Logs, and
`runtime.helpers.prepared` records the directory built and the logical names in
it, so a helper failure can be told apart from a lookup failure without
guesswork.

## Environment profiles

Tools that need more than the base allowlist declare an `environment_profile`.
The runtime prepares it; the caller cannot. Profile values are applied over the
inherited allowlist and before caller additions, but the denied list still wins,
so a caller cannot reach `PATH` or `LD_PRELOAD` by way of a profile.

- **`git`** sets `GIT_EXEC_PATH` to the helper directory, `GIT_TERMINAL_PROMPT=0`
  and an empty `GIT_ASKPASS` so a missing credential fails fast instead of
  hanging until the timeout, `GIT_CONFIG_NOSYSTEM=1`, a private `HOME` inside
  runtime metadata storage, and `GIT_SSL_CAPATH` from the platform store.
- **`gh`** puts the helper directory on `PATH` so gh finds the runtime's verified
  git rather than whatever the platform exposes, and disables prompts, the pager
  and update checks.

### Credentials

A GitHub token is read from `POCKETCLAW_GITHUB_TOKEN`, or from
`credentials/github_token` under runtime metadata storage — app-private and
outside the user workspace. The environment is checked first so the Android
Service can pass a credential it holds without it ever touching disk.

It is injected through **git's environment-based config**, never argv:

```
GIT_CONFIG_COUNT=1
GIT_CONFIG_KEY_0=http.https://github.com/.extraheader
GIT_CONFIG_VALUE_0=Authorization: Basic <base64>
```

This is deliberately not `https://TOKEN@github.com/...`, which leaks the token
into the command line where any process listing can read it, into git's own
remote config on disk, and into any error message that quotes the URL. gh
receives `GH_TOKEN` the same way. Both are redacted before any log writer sees
them.

### Certificate store

Android 14 moved the system certificates into the Conscrypt APEX, so the runtime
probes `/apex/com.android.conscrypt/cacerts` and then
`/system/etc/security/cacerts` and uses whichever exists. A single hard-coded
path would silently break TLS on either older or newer devices.

### Physically observed on hardware, 2026-08-30

Runtime Pack v2: Git HTTPS **PASS**, `git clone` of a public GitHub repository
**PASS**, Git helper symlink execution **PASS**, git 2.51.0, gh 2.82.1, curl
HTTPS **PASS**, ripgrep **PASS**, sqlite3 **PASS**.

From the Foundation run, **43 of 44** catalog tools resolved as available on the
tested ARM64 device.
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

## Runtime Pack v2

| Tool | Version | Installed | Why |
|---|---|---:|---|
| git | 2.51.0 | 3.23 MB | repository work over HTTPS |
| git-remote-http | (same build) | 2.98 MB | git's transport helper |
| gh | 2.82.1 | 55.9 MB | GitHub issues, PRs, releases, API |
| curl | 8.11.1 | 1.30 MB | HTTP and HTTPS requests |
| ripgrep | 14.1.1 | 4.27 MB | recursive source search |
| sqlite3 | 3.50.4 | 1.23 MB | local databases |

### Known limitation: no shell

git is built with its default compiled-in `SHELL_PATH` of `/bin/sh`, which
Android does not have. **Git features that depend on a shell — hooks above all,
and git's `ENOEXEC` fallback — are not guaranteed on Android.** Clone, fetch,
push, log, diff and status are unaffected: the transport helpers are ELF
executables and no shell is on that path.

This is not an oversight to fix with a flag. git's Makefile uses `SHELL_PATH`
both as the compiled-in constant and as the shell that runs its own build
recipes and code generators, so overriding it breaks the cross-build.

`gh` is a deliberate strategic exception to the size policy, accepted because
GitHub capability is core to the agent. It is **not** a precedent: any other
single tool above roughly 10 MB installed needs its own justification.

curl is real curl, not a lookalike. It links mbedTLS rather than OpenSSL, which
is why the entire TLS stack costs about 1.3 MB, and the same libcurl is what
`git-remote-http` uses — so curl the binary is nearly free once git is present.

`unzip`, `diff`, `patch` and `file` are catalogued as `system` entries costing
zero bytes. The resolver measures each device and reports what the platform
actually provides rather than assuming it.

`traceroute`, `zip` and `tree` were catalogued the same way until catalog
`2.4.0` and are now removed. The Android 16 image ships none of them, so they
were a permanently unmet promise rather than a probe, and each capability is
already covered: `ping` and `ip` for the diagnostics an ordinary app UID can
actually perform, `tar` with `gzip` and Python's `zipfile` for archives, and
`find` for directory listing. None is bundled: a payload for a duplicate
capability is not worth the APK weight, and `traceroute` needs raw sockets an
app UID does not get.

Still not shipped, with reasons:

- **yq** — 11.25 MB installed for YAML alone, above the size policy, and jq
  already covers JSON. Revisit if YAML handling proves to matter.
- **wget** — curl covers the same ground.
- **OpenSSH, rsync** — deferred. SSH is worth reconsidering once HTTPS git is
  proven on hardware.
- **Node, npm, compilers, ffmpeg, ImageMagick** — out of scope. PocketClaw is
  not becoming a Linux distribution. Python is the exception and now ships: a
  bundled Python 3.14 runtime with standard-library support and a dedicated
  Agent execution tool, running offline on-device, with no package installation
  and no direct network access.
- **dig, host, nslookup, ss, traceroute, zip, tree** — Android does not ship
  them, and listing catalog entries that can only ever probe unavailable would
  be noise, not information.

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
