# PocketClaw Decisions

## The WhatsApp Agent Channel returns under a new name, not the retired cards

- Date: 2026-09-02
- Decision: the console offers one experimental entry, `whatsapp_agent`, whose
  config block is still `whatsapp_native`. The retired `whatsapp` and
  `whatsapp_native` cards stay gone. The panel shows Pair and Disconnect and
  asks for no bridge URL, no session store path, and no `use_native` switch.
- Why: the earlier decision removed those cards because their first questions
  were a bridge URL and a session store path, which a phone user cannot answer.
  None of that changes now that the transport actually runs — the answers are
  simply not the user's to give: bridge mode is not offered, and Android owns
  the session path. Reusing the existing config key keeps an install that had
  one working, and keeps `pkg/migrate/sources/openclaw` importing as before.
- Consequence: `use_native` is written by the panel rather than shown, so the
  one control with exactly one valid value stops being a question. The entry
  sorts last in the sidebar, after every supported channel.

## Android owns the WhatsApp session path, and the config file cannot move it

- Date: 2026-09-02
- Decision: `POCKETCLAW_WHATSAPP_SESSION_DIR` names an app-private no-backup
  directory, and `pkg/whatsapp/session.Resolve` lets it beat a configured
  `session_store_path` outright rather than merging with it. The console never
  shows the field, and `/api/channels/whatsapp-agent/forget` erases only the
  host-named directory.
- Why: the whatsmeow store holds the linked device's Signal identity keys.
  Upstream defaults it to `<workspace>/whatsapp`, and on Android the workspace
  is `Download/pocketclaw` — public external storage, and the exact root the
  agent's file tools are restricted to under `RestrictToWorkspace`. The default
  therefore put the account keys inside the agent's own sandbox. A configured
  path is untrusted input for the same reason: the agent can write the config.
- Consequence: the keys are outside the workspace, outside cloud backup and
  device transfer (Android never includes `no_backup` in either), and outside
  the agent's reach — without touching the `credentials/` rules that already
  protect the GitHub token.

## The pairing QR travels as a file and is rendered server-side

- Date: 2026-09-02
- Decision: the gateway publishes the code into an app-private directory named
  by `POCKETCLAW_WHATSAPP_PAIR_DIR`; the console backend renders it as a PNG at
  `/api/channels/whatsapp-agent/qr.png`. The status JSON carries `has_qr` and
  never the payload. Nothing prints it.
- Why: upstream printed it to stdout with qrterminal, and PocketClaw captures
  Core's stdout into a persisted Logs screen — a pairing QR is credential
  material, and whoever scans it first links their device to the account. The
  gateway and the console are two processes, so the code has to cross a
  boundary; a loopback port would be reachable by every app on the device and
  would need a shared secret, which is the same reasoning that put the
  Core-to-host bridge in a directory. Rendering as an image means the payload
  never becomes a string in the browser, a response cache, or a devtools log.
- Consequence: the code is not purely in memory, which is unachievable across a
  process boundary without the socket that reasoning rejects. It is instead
  short-lived: written 0600, dropped on read under any non-pairing state, and
  removed on pair, on stop, and on Disconnect.

## Inbound WhatsApp denies an empty allow_from

- Date: 2026-09-02
- Decision: `WhatsAppNativeChannel.IsAllowedSender` returns false when
  `allow_from` is empty, rather than BaseChannel's allow-all. Allow-list
  entries and sender identities are both reduced to the bare subscriber number,
  and groups are dropped before their body is read.
- Why: allow-all is a reasonable default for a bot token nobody else holds. It
  is the wrong one for a transport linked to the user's personal account that
  feeds an agent holding shell and Python tools. Matching the number rather than
  the JID is what makes the closed default usable: the sender JID carries the
  sending device, which changes whenever the user relinks.
- Consequence: the normalization happens to the data in the constructor, not
  only in the override — `BaseChannel.HandleMessageWithContext` re-checks the
  allow-list from inside itself, where a method on the outer type cannot
  intervene.

## The build-path guard matches absolute paths only

- Date: 2026-09-02
- Decision: `core/build-android-arm64.sh` matches
  `(^|[^[:alnum:]._@+-])/(home|Users|root)/` rather than a bare
  `/home/|/Users/|/root/`.
- Why: the WhatsApp transport pulls in `go.mau.fi/libsignal`, whose correctly
  trimmed module path contains `keys/root/RootKey.go` — a cryptographic root
  key, not a developer's home directory. The bare pattern failed a build whose
  `-trimpath` was working perfectly.
- Consequence: every real leak still fails the build — `/home/<user>/...` at the
  start of a string, `file:///home/...`, `dir=/Users/...` — while a relative
  path segment that merely happens to be named `root` does not.


## A surface that shows its own apply state reports the outcome, not a toast

- Date: 2026-09-01
- Decision: `saveAndApplyGatewayConfig` takes an optional `onOutcome`. When a
  caller passes it, the helper delegates reporting and raises none of its own
  toasts. Callers that do not pass it behave exactly as before.
- Why: WhatsApp Self-Chat must never tell the user to restart anything by hand,
  and the shared deferred toast ends "Use Restart Gateway when you are ready".
  Suppressing that for one caller without giving it a way to say what happened
  would have left the card silent about a deferred apply.
- Consequence: the restart decision, the idle wait and the coalescing all stay
  in one place. Only the sentence shown to the user moves.


## The retired WhatsApp transports stay in the code and leave the console

- Date: 2026-09-01
- Decision: remove `whatsapp` and `whatsapp_native` from the channel catalog and
  the console's UI arms, and keep every line of their implementation.
- Why: they are not dead. `pkg/gateway` blank-imports both packages,
  `Manager.channelReadiness` splits them on `use_native`, `config` types their
  settings, `pkg/migrate/sources/openclaw` parses `channels.whatsapp.bridge_url`
  when importing an OpenClaw config, and six test files cover them.
- Consequence: an install that already configured a bridge keeps working, and
  nobody new is asked for a bridge URL. `/api/channels/whatsapp/config` now
  404s, which is what makes the entries unreachable rather than merely hidden.

## Self-Chat is a stored number, not a channel that runs

- Date: 2026-09-01
- Decision: `whatsapp_self_chat` lives in `channel_list` for the console's
  plumbing, registers no channel factory, and its readiness arm returns false
  however its config block is edited. The page hides the Enabled switch and the
  page-level Save, and Connect / Change / Disconnect persist directly.
- Why: it is a deep link the Android host opens, not a transport with a
  lifecycle. An Enabled switch would be a control with nothing behind it, and an
  enabled entry with no factory would log "Factory not registered" on every
  start.
- Consequence: "configured" means a stored number, everywhere — in Core, in the
  console, and in the sidebar's ordering.

## Core asks the Android host through a file, not a socket

- Date: 2026-09-01
- Decision: the `whatsapp_self_chat` tool writes `req-<id>.json` into an
  app-private directory named by `POCKETCLAW_ANDROID_HOST_OUTBOX` and waits for
  `res-<id>.json`; a `FileObserver` in the foreground service serves it.
- Why: Core runs as a child process and cannot start an Activity, so opening
  WhatsApp has to be a request. A loopback port would be reachable by every
  other app on the device and would need a shared secret to be safe; the
  directory is reachable only by this UID, so it needs neither.
- Consequence: the host answers every request it consumes, including ones it
  cannot serve, so the tool reports `not_installed` or `start_failed` rather
  than timing out. A request that is abandoned is deleted, because one left
  behind would open WhatsApp unprompted when the host next started.

## Regular WhatsApp wins, deterministically

- Date: 2026-09-01
- Decision: open `com.whatsapp` when it is installed, `com.whatsapp.w4b` when it
  is the only one, and report `not_installed` when neither is. No chooser.
- Why: the same code path serves a Settings button and an agent tool. A chooser
  raised by a tool call is a dialog nobody is looking at, and a tool that cannot
  say what happened is worse than one that picks.
- Consequence: both packages are declared in `<queries>`, because Android 11+
  hides everything else and the answer would otherwise always be "neither".

## The Chat attachment fix registers a file selector; it does not replace the pipeline

- Date: 2026-09-01
- Decision: register `AndroidWebViewController.setOnShowFileSelector` in
  `webview_android.dart` and change nothing about how a chosen image becomes a
  chat attachment.
- Why: the button was never broken in the page. It dispatches a click at a
  hidden `<input type="file">`, the WebView asks the host through
  `WebChromeClient.onShowFileChooser`, and with no handler registered the
  plugin returns `false` and Android shows nothing at all — silently. The
  data-URL pipeline behind it already worked.
- Consequence: the picker is Android's own (system photo picker on 33+,
  `ACTION_OPEN_DOCUMENT` below), so no storage permission is requested and
  `MANAGE_EXTERNAL_STORAGE` stays exactly as the log export left it.


## PocketClaw owns the GitHub credential, and the Keystore owns its key

- Date: 2026-08-31
- Decision: the token is encrypted with AES-256-GCM under a non-exportable
  Android Keystore key, stored as ciphertext in app-private storage, decrypted
  only by the Android service when it starts Core, and passed in as
  `POCKETCLAW_GITHUB_TOKEN`. The runtime's existing gh and git profiles inject it
  from there. Core's `credentials/github_token` file fallback was removed.
- Consequence: a credential Core can read off disk is a credential that survives
  on disk in usable form, which is exactly what the encrypted store exists to
  prevent. Reading it at launch means connect and disconnect take effect on the
  next start; that is a real cost, and it buys having no plaintext at rest.
- Consequence: there is no agent-facing way to read the token. The runtime tool
  takes no environment parameter, no tool returns a credential, and a test fails
  if one is ever registered whose name suggests it does. Python still runs as the
  application and is documented as not a sandbox, so this is a guarantee about
  PocketClaw's own surface, not about what a determined program can reach.

## gh is never asked to log in

- Date: 2026-08-31
- Decision: credential validation runs `gh api user` with a one-shot `GH_TOKEN`
  override through `Manager.Execute`. `gh auth login` and `gh auth setup-git` are
  not used, and git authenticates through a process-scoped
  `http.https://github.com/.extraheader` rather than a credential helper or a
  token-bearing URL.
- Consequence: gh writes no config of its own, `.git/config` carries nothing, and
  no cloned repository can leak the credential to whoever reads its remotes. The
  alternative was letting gh persist a credential in `~/.gitconfig` and in its
  own hosts file, which would move ownership of the secret outside PocketClaw for
  the sake of a shorter setup path.

## Android CPython logs its stdio, so PocketClaw ships its own entry point

- Date: 2026-08-31
- Decision: the Python tool runs `-m pocketclaw_bootstrap`, a module inside the
  payload's appended standard library, which rebinds `sys.stdout` and
  `sys.stderr` to `os.dup(1)`/`os.dup(2)` before executing the caller's program.
  CPython is not patched and Android's `TextLogStream` is left in place for
  anything else that embeds this interpreter.
- Consequence: on Android, CPython points `sys.stdout` and `sys.stderr` at the
  system log rather than at descriptors 1 and 2, so the managed runtime captured
  nothing. `print()` was a silent no-op that exited 0 and an uncaught exception
  exited 1 with no traceback — a failure that looked like a runtime defect and
  was not one. The descriptors are duplicated so interpreter shutdown closes the
  duplicate and never the pipe the runtime is still reading.
- Consequence: the module lives inside the payload rather than in app storage,
  so the checksum the registry verifies covers it. The alternative — writing it
  to the runtime metadata directory — would put the supervisor of model-authored
  code in a file that model-authored code can rewrite.
- Consequence: the caller's source did not move. It still travels on stdin,
  never in argv, the environment, a log or an event, and the bootstrap compiles
  it under `<stdin>` with `sys.argv[0]` set to `"-"`, which is what `python -`
  gives. Tests fail if the source reaches argv or if the argv contract changes.

## stdin carries the program and nothing else

- Date: 2026-08-31
- Decision: `pocketclaw_bootstrap` consumes standard input in full as the
  program. Phase C v1 adds no second stdin or data channel.
- Consequence: a program that needs structured input embeds it or reads a
  workspace file. A second channel would have to multiplex the one stream the
  interpreter reads, and the tool would gain an API before anything needed it.
  This is a limitation, recorded as one rather than worked around.

## The Core carries a fingerprint of the source it was built from

- Date: 2026-08-31
- Decision: `core/build-android-arm64.sh` stamps a content-addressed fingerprint
  of the Core's build inputs into `libpicoclaw.so` with `-X`, and the test gate
  recomputes it from `core/src` and fails if the staged binary does not carry it.
  `TestStagedCoreEmbedsTheCurrentCatalog` is kept alongside it.
- Consequence: the catalog guard answers "does this Core know the current
  catalog?", which is silent about a Go-only change: edit `python_tool.go`, leave
  `manifest.json` alone, and a Core built last week still embeds this week's
  catalog and every pinned checksum. That is exactly what Phase C did, and the
  only thing standing between it and a shipped stale Core was someone
  remembering to run the build script. The two guards answer different questions
  and both are permanent.
- Consequence: the fingerprint takes only relative paths and file bytes — never
  an mtime, a build timestamp, an absolute path, a developer name or a build id.
  A value that moves on its own cannot say anything about staleness, and a guard
  that fires at random stops being read. The first draft hashed every `*.json`
  under `pkg/`, which `pkg/cron`'s own tests write into while the gate runs, and
  it failed against a Core that was current. Embedded assets are named
  individually now, and a test reads the `//go:embed` directives out of the Core
  source so the list cannot fall behind quietly.

## The Python result states every field, including the empty ones

- Date: 2026-08-31
- Decision: `formatPythonResult` always writes `exit_code`, `timed_out`,
  `cancelled`, `stdout_truncated` and `stderr_truncated`, then both streams under
  their own headings — an empty stream as `(empty)` rather than being left out.
- Consequence: a model that cannot tell "the interpreter printed nothing" from
  "the result dropped it" has to guess, and guessing about a traceback is the one
  thing this tool must never require. Phase C's physical run found exactly that:
  the execution was correct and the model still could not say why
  `raise ValueError("TEST-ERROR")` failed. Nothing in the report is synthesised —
  the text is the runtime's own bounded, redacted capture, and a test fails if a
  traceback appears for a run that produced none.

## Python source travels on stdin, and the traceback name is not worth a wrapper

- Date: 2026-08-31
- Decision: The Agent-facing `python` tool passes source through
  `ExecRequest.Stdin` as `python -`, never `-c`, and accepts `<stdin>` as the
  traceback filename for v1.
- Consequence: argv is capped near 128 KB, is readable from `/proc/<pid>/cmdline`
  and is recorded in argument diagnostics, so a program containing a secret would
  leak simply by being run. stdin is accounted only as `bytes_in`. A nicer
  `<pocketclaw>` traceback name is achievable, but only with a wrapper that reads
  stdin and recompiles the source — an interpreter trick wrapped around the exact
  path that carries user code, in exchange for cosmetics. A regression test fails
  if the implementation ever moves back to `-c`.

## The Python tool steers toward the cheaper tool, and says what it is not

- Date: 2026-08-31
- Decision: The tool description names jq, rg, sqlite3 and curl for the work each
  suits, names the cases where Python genuinely earns its cost, and states that
  Python is not a sandbox. Tests fail if either half disappears.
- Consequence: Python starts in about 90 ms against single-digit milliseconds for
  jq, so a model that reaches for it by default makes every simple task slower
  for no gain. And a model that believes Python is contained will write code on
  that assumption. Both failures are caused by the description, so both are
  guarded by tests on the description.

## A packaged payload is checked for reachability, not just presence

- Date: 2026-08-31
- Decision: The APK build guard verifies that the packaged Python payload still
  ends in a zip end-of-central-directory record, and the test gate verifies that
  the staged Core embeds the current catalog. Both guards are permanent.
- Consequence: Phase B produced the same failure twice in different clothes.
  Gradle's native-library strip removed the standard library appended to the
  Python payload, leaving a correctly-named, correctly-sized-looking interpreter
  that could not import anything. Separately, an APK shipped the new payload
  beside a Core built before the catalog knew the tool existed. In both cases the
  existing payload guard passed, the build was green, and the app ran — because
  the guard asked whether files were present, and the question that mattered was
  whether they were reachable.

## A catalog change is not shipped until Core is rebuilt

- Date: 2026-08-31
- Decision: The staged Core binary is checked against the current catalog in the
  normal test gate, and the check names the build script to run.
- Consequence: `manifest.json` is `//go:embed`-ed into `libpicoclaw.so`, and that
  binary is a committed artifact rather than something Gradle produces. Editing
  the catalog and rebuilding the APK produces an installable, runnable app that
  silently reports the previous catalog — the payload guard sees the new payload
  file and is satisfied, because presence is not the same as reachability. The
  first Phase B APK shipped exactly that. A green build and a correct APK payload
  list were both true while the feature was entirely absent on the device.

## Python ships as a Runtime tool, and it is not a sandbox

- Date: 2026-08-31
- Decision: CPython 3.14.7 is bundled as the Managed Runtime `python` tool and
  reached through the generic Runtime execution path. It is not, and will not be
  presented as, a sandbox.
- Consequence: Python is the first bundled tool that turns model-authored text
  into executed code. The security boundary is the Android application UID and
  nothing narrower. A Python script can read app-private config, write the
  workspace and spawn other processes, and `subprocess` bypasses the Runtime's
  registry, timeout policy, output bounds and event log entirely. Agent guidance
  to prefer the Runtime for external commands is advisory, not enforcement.
  The mitigations that are real are the ones that remove capability rather than
  ask for restraint: no pip, no ctypes, no `_socket`, no writable executable
  storage, and a catalog-fixed `-S` so a workspace `sitecustomize.py` cannot be
  imported automatically ahead of the caller's own code.

## Python's standard library rides inside the interpreter payload

- Date: 2026-08-31
- Decision: The stdlib is a `.pyc` zip appended to the interpreter ELF, and
  `PYTHONPATH` points at the payload itself. Extension modules are linked
  statically (`MODULE_BUILDTYPE=static`), so `lib-dynload` is empty.
- Consequence: One file is the whole runtime. Static linking removes the
  extension-module naming problem altogether — there is no
  `_ssl.cpython-314-aarch64-linux-android.so` to package under a `lib*.so` name,
  and nothing needs `dlopen` from a directory the platform would refuse. The
  `.pyc` form costs 943 KB more than source but starts about 4.5x faster,
  because `PYTHONDONTWRITEBYTECODE` means a source zip recompiles on every
  launch and nothing is ever cached. Stripping must happen before the zip is
  appended; `llvm-strip` rewrites the file and would discard it.

## Python builds every dependency from pinned source

- Date: 2026-08-31
- Decision: bzip2 1.0.8, XZ 5.4.7 and SQLite 3.50.4 are built from pinned source
  by `runtime/build-python-android-arm64.sh`. Upstream CPython's Android tooling
  is not used to obtain them.
- Consequence: Upstream `Android/android.py` downloads six prebuilt dependency
  tarballs over `curl -Lf` with **no checksum verification of any kind**. That is
  acceptable for a feasibility build and not for a shipped payload, so the whole
  dependency set is built here instead. OpenSSL and libffi are not built at all,
  because the reduced profile links neither.

## Android's shell is version-dependent, so Python's is too

- Date: 2026-08-31
- Decision: PocketClaw documents shell availability under Python as conditional
  on the Android version, and ships no shell.
- Consequence: The earlier blanket claim that Android has no `/bin/sh` is wrong
  for modern devices. Android 11 added a `/bin` -> `/system/bin` symlink, and on
  the tested Android 16 device `/bin/sh` is mksh, `subprocess(shell=True)`
  returns 0 and `os.system()` returns 0. `minSdkVersion` is 24, so on API 24-29
  those calls still fail. Neither behaviour may be assumed. The same wrong
  assumption underlies the recorded git `SHELL_PATH` limitation, which is logged
  as technical debt rather than changed here.

## Provider Resilience closed on physical evidence, not on a green build

- Date: 2026-08-30
- Decision: The milestone is closed as PHYSICAL PASS on the target ARM64 device:
  failover, the Fallback Models UI, active-turn restart safety and resume
  recovery were all exercised on hardware before merge.
- Consequence: The load-bearing observation is the restart sequence — a running
  request finished before the gateway restarted, on a real device. That is the
  behaviour every safety invariant in this milestone exists to produce, and it
  is the one thing a green test suite could not have established.

## The WebView is recovered on evidence, never on every resume

- Date: 2026-08-30
- Decision: On resume the Android WebView asks the page a question only a live
  page can answer — a readiness flag plus a non-empty `#root` — and reloads only
  when the answer is missing or wrong. A healthy page is left untouched, and
  recovery reloads the route the user was on rather than the console home page.
- Consequence: Android may kill a backgrounded WebView's renderer process to
  reclaim memory. The view keeps its layout and shows white, and
  `webview_flutter_android` 4.14.0 exposes no `onRenderProcessGone` callback, so
  Dart is never told — verified by grep, the API simply is not there. Probing is
  the only signal available.

  A blanket `onResume → reload()` was rejected: it would discard scroll position
  and page state on every single resume, add pointless work, and hide the defect
  rather than fix it. Recovery is capped at one attempt per page load, so a
  console that is genuinely broken stops being reloaded and leaves the Refresh
  control usable instead of flickering.

  Blankness is deliberately not detected by sampling pixels or background
  colour. The page reports its own health; a white area is a symptom, not a
  signal.

## The console has an error boundary so a crash is not silent

- Date: 2026-08-30
- Decision: The React tree is wrapped in `AppErrorBoundary`, which shows a
  reload affordance and clears the readiness flag.
- Consequence: Without one, an uncaught render error unmounts the whole tree and
  empties `#root`. That is visually identical to a killed renderer and equally
  unexplained, which is exactly what made the white-screen report ambiguous.
  Clearing the readiness flag also lets the host's resume probe recover a
  crashed console, so the two failure modes share one recovery path while
  remaining distinguishable in the logs.

## The fallback chain belongs to the default model, not to each model entry

- Date: 2026-08-30
- Decision: The Fallback Models UI lives on the Models page beside the default
  model, editing `Agents.Defaults.ModelFallbacks`. It is not in the per-model
  edit sheet.
- Consequence: `config.ModelConfig` also has a `Fallbacks` field, but that one
  serves multi-key expansion within a single provider and is generated, not
  user-edited. Putting the UI in the edit sheet would have targeted the wrong
  field and implied every model entry has its own chain. A fallback is stored as
  a **reference by model name**, so the referenced entry keeps its own provider,
  credentials, base URL and headers — the primary's API key is never copied to
  another provider's endpoint.

## Configuration saves restart the gateway automatically, when it is idle

- Date: 2026-08-30
- Decision: Saving a restart-requiring setting now applies it. The frontend
  helper `saveAndApplyGatewayConfig` persists the change, asks the backend
  whether a restart is required, and only then calls `POST
  /api/gateway/apply-config`, which waits for the gateway to become idle before
  restarting.
- Consequence: The restart decision is still the existing
  `gateway_restart_required` signature comparison — there is no second decision
  system to keep in agreement with the first, so cosmetic edits and
  hot-reloadable fields still restart nothing. Success means the gateway is
  running *and* its boot signature matches the saved config, not that the
  restart call returned 200. A failed restart never discards the save, because
  the config is written before any restart is attempted, and the manual Restart
  Gateway control remains as the recovery path.

## A restart for a settings change must never interrupt an answer

- Date: 2026-08-30
- Decision: Core's `/health` reports `active_requests` and `busy`, fed by the
  agent loop's existing in-flight counter. `apply-config` restarts the gateway
  only on two outcomes: the process is not running, or it reported itself idle.
  **Neither an unreadable busy signal nor an expired wait authorises a restart.**
- Consequence: This is why `apply-config` is a separate endpoint from
  `POST /api/gateway/restart`. The manual restart is immediate recovery the user
  asked for explicitly; a restart caused by saving settings must not cut off a
  Telegram reply mid-sentence or kill a `git push` half way through.

  The four outcomes are `not_running` and `idle` (restart proceeds),
  `busy_timeout` and `unverified` (it does not). The two-minute limit bounds how
  long PocketClaw *waits*, not how long the user's work is safe: reaching it
  leaves the configuration saved and unapplied. A running gateway that will not
  report its busyness is retried for five seconds and then also left alone —
  unknown is never read as idle, because a gateway too old to answer, or one
  whose health endpoint is briefly unreachable, may be part way through a turn.

  An earlier revision of this code did force a restart in both cases, on the
  reasoning that blocking on a signal that never arrives would make settings
  impossible to apply. That trade was wrong: the cost of a delayed setting is a
  banner, and the cost of a forced restart is destroyed work. The configuration
  is persisted before any restart is attempted, so a withheld restart loses
  nothing and the manual control still applies it.

  There is deliberately **no background retry**. A restart that fires at an
  arbitrary later moment is exactly the surprise the Auto-Start milestone was
  built to avoid; the restart-required indicator stays visible and the next save
  or the manual action applies the change.

## Provider retry is scoped to the request; the turn never rewinds

- Date: 2026-08-30
- Decision: Provider resilience was implemented as gap-closing on the existing
  `FallbackChain`, `CooldownTracker` and `ClassifyError`. No checkpoint
  subsystem, no tool fingerprinting, no per-tool side-effect classification and
  no runtime shell-command parser were built.
- Consequence: The agent loop already guaranteed the invariant those mechanisms
  would have defended. Tool results are committed to the turn and the session in
  `pipeline_execute.go` before control returns, and the loop in `turn_coord.go`
  never re-enters a completed tool execution — a retry re-sends the committed
  results rather than replaying the call. Adding machinery to protect a property
  the structure already provides would have added surface area and new ways to
  be wrong. It is protected instead by tests and one small guard.

## Tool replay identity is the provider's call id, and nothing else

- Date: 2026-08-30
- Decision: `completedToolResults` records a finished tool result under its
  `protocoltypes.ToolCall.ID` and reuses it if that exact id is ever presented
  again in the same turn. Matching on tool name, arguments, or any hash of them
  is deliberately excluded, and a call with no id is not recorded at all.
- Consequence: Asking for the same command twice in one turn is legitimate —
  re-reading a file after editing it, retrying a fetch the model believes went
  stale — and collapsing those into one execution would silently change what the
  agent did. A missing id means no identity: inventing one from the arguments
  would reintroduce exactly the fingerprint matching this rejects. The guard is
  defensive; if it ever fires, something tried to replay a turn, which is why it
  logs at warning severity.

## The 5xx family is classified by what each status means

- Date: 2026-08-30
- Decision: 502 classifies as network, 503/521/522/523/529 as overloaded, and
  500/504/524 as timeout, replacing a single collapse into timeout. 429 is
  refined against the response body into transient rate limiting or hard quota.
- Consequence: A log that says "timeout" for a gateway that could not reach
  upstream is actively misleading during diagnosis, and the classes now carry
  different cooldown treatment. The hard-quota patterns are deliberately narrow:
  reading ordinary throttling as an exhausted quota would put a healthy provider
  into a long cooldown, and the reverse produces a retry storm against a
  provider that has already said no. A 429 alone cannot distinguish them, so
  only the body does.

## Capability gating returns unknown, and unknown is not a refusal

- Date: 2026-08-30
- Decision: `SupportsToolCalls` returns supported, unsupported or unknown, and
  the fallback chain skips a candidate only on an explicit unsupported. The only
  seeded negatives are model families that are not chat models at all —
  embedding, TTS, transcription, moderation, rerank.
- Consequence: PocketClaw routes to providers whose model lists it does not
  enumerate. Treating unrecognised as unusable would disable failover for
  exactly the configurations that need it most. No tool-support metadata was
  invented for chat models, because it varies by provider, version and endpoint,
  and a wrong guess either skips a working candidate or lets a doomed one
  through while claiming it was checked.

## Cross-provider context-overflow fallback is DEFERRED

- Date: 2026-08-30
- Decision: Context overflow keeps its existing compact-and-retry path on the
  current candidate. It is not made retriable into a different candidate.
- Consequence: Choosing a fallback for context overflow requires knowing the
  alternate model's context capacity, and no reliable per-model capacity
  metadata exists — `ContextWindow` is an agent default, not a model property.
  Guessing would mean failing over to a model that overflows too, having spent
  the latency. Recorded as deferred rather than implemented on invented data.

## An empty completion is not an outage

- Date: 2026-08-30
- Decision: `responseIsUserVisiblyEmpty` is used for logging and the end-of-turn
  placeholder only. It never triggers a retry or a failover. A response carrying
  tool calls is not empty. The placeholder text no longer asserts a provider
  error or a token limit.
- Consequence: The old placeholder — "This may indicate a provider error or
  token limit" — fires for entirely ordinary contentless turns and was read as
  evidence of an outage when no provider had failed. Genuinely
  provider-attributable emptiness (zero-byte body, truncated framing,
  unparseable JSON) arrives as an error from the provider adapter and is
  classified there, which is the only place it can be told apart from a model
  choosing to say nothing.

## Symlink execution into nativeLibraryDir works on Android, proven on hardware

- Date: 2026-08-30
- Decision: The helper model stands. Android permits executing through a symlink
  in app-private storage whose target is a packaged payload in
  `nativeLibraryDir`, confirmed end to end by a real `git clone` over HTTPS on a
  physical ARM64 device.
- Consequence: This was the one open platform question behind the whole helper
  design, and it is now evidence rather than reasoning. The kernel resolves the
  link and executes the read-only packaged file, so the API 29+ restriction on
  executing writable app storage does not apply — a symlink is not an
  executable. This is what makes multi-executable tools such as git possible on
  Android at all. It does **not** license copying an executable into app storage,
  which remains refused by the platform; the probe keeps reporting
  `symlink_exec` so a device that behaves differently says so in its own logs.

## Shell-dependent git features are unsupported on Android

- Date: 2026-08-30
- Decision: PocketClaw ships git with its default compiled-in `SHELL_PATH` of
  `/bin/sh`, a path Android does not have, and documents the consequence rather
  than working around it.
- Consequence: **Git hooks and git's `ENOEXEC` fallback are not guaranteed on
  Android.** Clone, fetch, push, log, diff and status are unaffected, because the
  transport helpers are ELF executables and no shell is on that path. Overriding
  `SHELL_PATH` is not available: git's Makefile uses the same variable as the
  shell that runs its own build recipes and code generators, so setting it breaks
  the cross-build — verified twice, including with make's `SHELL` overridden
  separately. Revisit only if a real use for hooks appears, and expect it to
  require patching git's build rather than passing a flag.

## A tool that re-invokes itself by name must be its own helper

- Date: 2026-08-30
- Decision: git's catalog entry declares `git` among its own helpers, alongside
  `git-remote-http` and `git-remote-https`.
- Consequence: git does not exec its transport helper directly. It spawns
  `git remote-https`, and `prepare_cmd()` resolves the literal name `git`
  through `PATH`, which `setup_path()` builds by prepending `GIT_EXEC_PATH`.
  Without a `git` entry that lookup returns `ENOENT`, and `get_helper()` turns
  that specific errno into `unable to find remote helper for 'https'` — a message
  that names the protocol and sends the reader to TLS, which is not where the
  fault is. This cost a physical release cycle to find. The general rule is that
  a tool which re-invokes itself by name needs an entry for itself, and
  `TestGitDeclaresItselfAsAHelperSoItsOwnLookupSucceeds` enforces it.

## git ships with the wrong compiled-in SHELL_PATH, knowingly

- Date: 2026-08-30
- Decision: git is built with its default `SHELL_PATH` of `/bin/sh`, which does
  not exist on Android, rather than `/system/bin/sh`.
- Consequence: git's Makefile uses `SHELL_PATH` both as the compiled-in constant
  *and* as the shell that runs its own build recipes and code generators, so
  overriding it breaks the cross-build — verified twice, once with `SHELL`
  overridden as well. Nothing on the clone path needs a shell: the transport
  helpers are ELF executables. This is recorded as a known limitation affecting
  hooks and git's `ENOEXEC` fallback, not worked around, and should be revisited
  only if a real use appears.

## Helper executables are presented by symlink, never by copy

- Date: 2026-08-30
- Decision: A tool that invokes auxiliary executables by a logical name the APK
  cannot use as a filename — git looking up `git-remote-https` in
  `GIT_EXEC_PATH` — gets a runtime-built directory of symlinks pointing at its
  packaged payloads. Helper payloads are hashed and ABI-checked exactly like the
  main one, and a tool whose helper fails verification resolves as unavailable.
- Consequence: Nothing is ever written into app storage and executed; the kernel
  resolves the link and runs the read-only packaged file, so the delivery model
  is unchanged. This is the single platform assumption Pack v2 rests on, so
  `ProbeExecution` measures it as `symlink_exec` on every device rather than
  trusting it. If a device refuses it, `git clone` over HTTPS cannot work there
  and the runtime will say so instead of failing obscurely. Logical names are
  validated as filenames and never paths, so a catalog entry cannot place a link
  outside the helper directory.

## Credentials reach git and gh through the environment, never argv

- Date: 2026-08-30
- Decision: A GitHub token is read from `POCKETCLAW_GITHUB_TOKEN` or from
  app-private runtime storage and injected via git's
  `GIT_CONFIG_COUNT`/`GIT_CONFIG_KEY_n`/`GIT_CONFIG_VALUE_n` mechanism, and via
  `GH_TOKEN` for gh. Both are marked secret and redacted before any log writer
  sees them.
- Consequence: `https://TOKEN@github.com/...` is forbidden and must not be
  reintroduced as a convenience. It leaks the credential three ways at once —
  into the command line where any process listing can read it, into git's remote
  config on disk, and into every error message that quotes the URL. The
  environment-config form leaks none of them. The agent is told not to supply
  credentials itself, because the runtime already does.

## curl is real curl, and mbedTLS is why it is affordable

- Date: 2026-08-30
- Decision: curl 8.11.1 is bundled, built against mbedTLS rather than OpenSSL.
  The finished binary is about 1.3 MB because zlib comes from the Android system
  image and mbedTLS is a fraction of OpenSSL's size.
- Consequence: This reverses the Foundation milestone's deferral of curl, on
  measurement rather than opinion — the estimate there was several megabytes and
  a heavy dependency chain. The same libcurl is statically linked into
  `git-remote-http`, so curl the binary is nearly free once git is present. A
  lightweight in-house HTTP tool was rejected: it would have had to be called
  something other than `curl`, and would have cost more to maintain than 1.3 MB.

## gh's 56 MB is a sanctioned exception, not a precedent

- Date: 2026-08-30
- Decision: gh 2.82.1 ships at roughly 56 MB installed and 16.5 MB compressed,
  taking the APK from about 35 MB to about 57 MB. The user accepted this
  explicitly because Git and GitHub capability is core to the agent.
- Consequence: The size policy still stands for everything else — any other
  single tool above roughly 10 MB installed needs its own justification before
  inclusion. yq was measured at 11.25 MB and left out on exactly that basis,
  since jq already covers JSON. Do not cite gh as a reason to bundle the next
  large tool.

## Tier 1 growth is probed, not bundled

- Date: 2026-08-30
- Decision: `zip`, `unzip`, `diff`, `patch`, `file` and `tree` were added as
  `system` catalog entries rather than bundled payloads.
- Consequence: They cost zero bytes and the resolver reports per device what the
  platform actually provides, so the physical run tells us which are worth
  bundling instead of us guessing. Shipping a catalog entry *is* the probe. Only
  bundle one of these if a device is shown not to provide it and the capability
  is genuinely needed.

## An inconclusive writable-exec probe is information, not a blocker

- Date: 2026-08-30
- Decision: On the device that physically validated the Managed Runtime, the
  writable-app-data execution probe returned `inconclusive` — it could neither
  run its staged copy nor observe a clean permission refusal. The delivery model
  is unchanged: Android system executables and APK payloads unpacked into
  `nativeLibraryDir`, both read-only to the app.
- Consequence: The probe exists to report the platform's behaviour, not to gate
  anything, which is why an inconclusive answer costs nothing — the bundled jq
  payload executed from `nativeLibraryDir` on the same run, and that is the path
  the runtime actually uses. Do not read `inconclusive` as permission to attempt
  writable execution, and do not "fix" the probe by making it assume a verdict;
  reporting what was actually observed is the whole point of it.

## Skills 7/7 is the expected value; the GitHub Skill is not coming back

- Date: 2026-08-30
- Decision: The incomplete GitHub Skill was deliberately removed by the user. The
  expected Skills count is 7/7 and the expected agent tool count is 18.
- Consequence: 7/7 must not be reported as a regression, no migration should
  restore the Skill, and 8/8 must not be used as a target in any acceptance list.
  One loose end is recorded rather than silently resolved: the Skill still exists
  in the seeded workspace tree, so a fresh install would receive it again. If
  7/7 is meant to hold for new installs, remove it from `core/src/workspace/skills`
  or add it to `unseededTemplates` — a deliberate change, not a side effect of
  another milestone.

## Android forbids executing writable app storage, so the runtime has no installer

- Date: 2026-08-30
- Decision: The Managed Runtime delivers executables by exactly two routes, both
  read-only to the app: `system` (the platform's own `/system/bin`) and
  `bundled` (APK payloads the package manager unpacks into `nativeLibraryDir`).
  There is no download, verify-and-activate provisioning pipeline for
  executables, because PocketClaw targets SDK 36 and an app targeting API 29+
  may not `execve()` a file in its own writable data directory. `setExecutable`
  does not change that: the restriction is enforced on the app's SELinux domain,
  not by the file mode.
- Consequence: "No arbitrary binary installation" stops being a policy someone
  must keep enforcing and becomes a structural property — there is no
  download-and-execute code path in this repository to audit or to regress.
  `EnsureTool` reports availability; it never acquires. This supersedes the
  earlier `TASKS.md` plan for an app-private `runtime/bin`, which cannot work.
  Any future provisioning abstraction is limited to non-executable assets unless
  the platform model changes.

## Bundled runtime payloads must be named `lib*.so`

- Date: 2026-08-30
- Decision: Every bundled Managed Runtime executable ships as
  `lib<name>.so` under `android/app/src/main/jniLibs/<abi>/`, is listed in
  `requiredArm64NativeLibraries` and in `keepDebugSymbols`, and is exposed to the
  agent under its logical name. jq 1.7.1 ships as `libpocketclaw-jq.so` and
  resolves as `jq`. The catalog format rejects a bundled entry whose
  `library_name` is not `lib*.so`.
- Consequence: Android's package manager only unpacks `lib/<abi>/*.so` into
  `nativeLibraryDir`, the one directory the app may execute from, so any other
  name ships a payload that can never run. `keepDebugSymbols` matters just as
  much: without it Gradle strips the executable during packaging, changing its
  bytes and breaking the SHA-256 the catalog pins, which reaches the device as a
  `checksum_mismatch` that looks like a corrupt install.

## Tier 1 runtime tools are system-provided, not bundled

- Date: 2026-08-30
- Decision: Android already ships toybox in `/system/bin`, covering nearly the
  whole Tier 1 command list. Those tools are catalogued with
  `delivery_type: system` and probed on the device rather than bundled.
  BusyBox is not shipped.
- Consequence: The APK does not carry tens of megabytes duplicating the platform,
  and PocketClaw takes on no GPLv2 source-offer obligation. The cost is that the
  command set varies by Android version and vendor, which is why availability is
  measured per device and never read from the catalog. A `system` tool carries
  `security_class: system` and `verification_result: platform_owned`; the catalog
  format refuses to record a SHA-256 for one, because the OS replaces those files
  on every system update and a pinned hash would be a guarantee PocketClaw
  cannot keep.

## Managed tools run by argv, with a constructed environment

- Date: 2026-08-30
- Decision: The runtime executes `argv` directly and never invokes `sh -c`. The
  child environment is built from an allowlist rather than inherited, and
  `PATH`, `LD_PRELOAD`, `LD_LIBRARY_PATH`, `LD_AUDIT` and the `DYLD_*`
  equivalents may not be set by a caller. Working directories are confined to the
  user workspace, and the tool's timeout profile is a ceiling a caller may lower
  but never raise.
- Consequence: Shell metacharacters in an argument are inert, provider keys held
  by the Core process cannot reach a child by accident, and a caller cannot make
  the loader substitute code for the binary the registry just verified. This is
  deliberately narrower than the existing `exec` tool, which does use `sh -c`
  behind deny patterns; the two are separate surfaces and the runtime does not
  relax to match it.

## Runtime logs record that a tool ran, never what it printed

- Date: 2026-08-30
- Decision: Redaction happens inside the runtime's event emitter, before any
  writer sees a value. Argument vectors, environment maps and free text are
  redacted; `stdout` and `stderr` are logged only as byte counts. The caller
  still receives the real output in memory.
- Consequence: A tool's stdout is exactly where a fetched credential appears —
  `gh auth token` prints one — so bounding what may be persisted is more reliable
  than pattern-matching what was printed. Flag-directed redaction is scoped per
  tool rather than applied globally, because blanking the element after `-u` or
  `-E` everywhere would erase the filename in `sort -u notes.txt` and the pattern
  in `grep -E '<expr>' file`, destroying the diagnostics the logs exist for.

## Auto-Start is an app-launch decision, not a supervision framework

- Date: 2026-08-30
- Decision: Auto-Start evaluates exactly once per app process, from `main()`,
  after a true launch. There is no resume hook, watchdog, boot-receiver change,
  crash-restart policy, or resurrection path, and `START_NOT_STICKY` plus a
  null-intent guard stop Android from re-creating the Service on its own.
- Consequence: A manual Stop stays stopped structurally rather than by a flag,
  and "start automatically" can only ever mean "on the next legitimate app
  launch". Do not add resume evaluation, a watchdog, or a background daemon
  policy under the Auto-Start banner; battery/background work is a separate,
  still-deferred milestone.

## `feature/autostart-foundation` is reference only and must never be merged

- Date: 2026-08-30
- Decision: The experimental Auto-Start branch (`c854d7a`) is abandoned. The
  shipped implementation, `fix/autostart-safe-rebuild`, was rebuilt from
  `v0.2.0-rc1` and reimplements only individually reviewed pieces.
- Consequence: Its automated tests were green while a manual Start Service
  press could land in a FAILED state RC1 cannot represent. Never merge or
  cherry-pick it. In particular do not reintroduce a `Boolean` return from
  native `PicoClawService.start()` mapped onto a failure state, the
  process-lifetime `isStarting`/`isStopping`/`manualStopActive`/`hasFailed`
  companion flags, or `gatewayRuntimeReady()`.

## Native process ownership outranks command-line inspection

- Date: 2026-08-30
- Decision: If the launcher spawned a pid itself via `exec.Cmd` and that
  process is alive, it is the managed Gateway — decided before `ps` is
  consulted. A negative command-line match is decisive only when `ps` actually
  returned a command line; a bare executable name is un-inspected, not foreign.
- Consequence: Android's `ps` does not report the `gateway` subcommand for a
  process executed as `libpicoclaw.so`, which was deleting a live Gateway's pid
  file. Dead-process cleanup is unaffected because it lives in
  `ppid.ReadPidFileWithCheck` and runs before validation. Do not restore
  bare-name rejection, and do not make string matching on a command line the
  sole ownership proof.

## Independent repository, not a permanent FUI fork

- Date: 2026-08-24
- Decision: PocketClaw keeps an independent Git history and uses PicoClaw FUI
  as a manually reviewed reference.
- Consequence: No `git merge` maintenance path from FUI; upstream changes must
  be classified as security, bug fix, Android runtime, Core compatibility,
  Telegram, provider, MCP, performance, useful feature, UI-only, or not
  relevant before adoption.

## Preserve the physical-device-proven runtime behavior

- Date: 2026-08-24
- Decision: Keep the Android active-network DNS bridge, pinned Core `v0.3.1`,
  and optional feedback defaults in the independent foundation.
- Consequence: Never revert to `[::1]:53`, hardcode public DNS, include
  Firebase credentials, or make analytics mandatory.

## Delay package identity and product branding

- Date: 2026-08-24
- Decision: Keep the inherited technical package identity until the independent
  foundation builds and is smoke-tested.
- Consequence: Product branding and package-ID changes are a later isolated
  milestone.

## No final license for PocketClaw-authored code yet

- Date: 2026-08-24
- Decision: Preserve upstream MIT notices and defer choosing a license for new
  PocketClaw code until product-owner approval.

## Bound the local Gradle JVM for reproducible foundation builds

- Date: 2026-08-24
- Decision: Set the inherited Gradle JVM maximum heap to 4 GiB and metaspace to
  2 GiB.
- Reason: The inherited 8 GiB/4 GiB settings exceed the available headroom in
  the supported local build environment and can terminate the daemon before
  APK assembly.
- Consequence: This changes only build-process memory limits; Android runtime
  behavior and the pinned Core binaries are unaffected.

## Product identity is PocketClaw; PicoClaw remains the credited engine

- Date: 2026-08-24
- Decision: Use PocketClaw for all product-visible identity and
  `com.lord1egypt.pocketclaw` for the independent Android/Dart integration
  identity. Retain PicoClaw Core names, binaries, environment variables,
  protocol identifiers, and attribution where they are integration contracts.
- Consequence: The PocketClaw and reference APKs can coexist; Android private
  app data is intentionally not migrated across package identities. The
  compatible `Downloads/picoclaw` external workspace remains until a separately
  designed migration exists.

## Skill Hub regression is DNS-resolved, not a separate product defect

- Date: 2026-08-24
- Decision: Keep the existing Skill Hub/ClawHub implementation unchanged.
- Evidence: On a physical device after the active-network DNS fix, Skill Hub
  opened and `Crypto` search returned 20 results with metadata, URLs, and
  visible install actions.
- Consequence: Treat Skill Hub as regression-sensitive; do not install skills
  automatically and do not change its registry configuration without new
  source/runtime evidence.

## Use the selected original PocketClaw mark consistently

- Date: 2026-08-24
- Decision: The second generated PocketClaw mark is the primary visual
  direction. Launcher, adaptive, splash, and monochrome notification variants
  must simplify that identity rather than introduce a cartoon lobster/mascot.
- Consequence: Preserve a restrained professional Android/developer-tool
  appearance and keep future visual work non-derivative of PicoClaw artwork.

## Android splash layers must contain drawables

- Date: 2026-08-24
- Decision: Keep the branded PocketClaw launch mark, but express the splash
  background as an explicit Android color drawable reference rather than
  `android:color` on a `layer-list` item.
- Evidence: `LayerDrawableItem` accepts `android:drawable`; the Milestone B
  resource supplied neither a drawable attribute nor a child drawable, blocking
  launch-theme inflation before Flutter's first frame on the physical device.
- Consequence: `launch_background.xml` and its v21 variant reference
  `@color/pocketclaw_splash_background`; a Flutter regression test guards both
  files. This is not a package-ID migration or Core integration change.

## The `jni` CMake configure cache lives outside the project and can poison builds

- Date: 2026-08-25
- Decision: Treat `~/.pub-cache/hosted/pub.dev/jni-<version>/android/.cxx/` as a
  build input that must be purged when an Android native library goes missing
  from a release APK. Verify `lib/arm64-v8a/libdartjni.so` is packaged before
  releasing any PocketClaw APK.
- Evidence: On 2026-08-24 02:52 the CMake configure of the `jni` package for
  `arm64-v8a` failed inside this workspace with a transient filesystem error
  (`unable to open output file '...CMakeCCompilerABI.c.o': No such file or
  directory`). CMake concluded the C compiler was unusable for that ABI and
  cached a configure result with an empty target list
  (`"libraries": {}`, `"cFileExtensions": []`). Every later build reported the
  JSON "up-to-date", built no arm64 target, and shipped an APK with no
  `lib/arm64-v8a/libdartjni.so`.
- Consequence: `com.github.dart_lang.jni.JniPlugin` calls
  `System.loadLibrary("dartjni")` from a **static initializer**, and
  `GeneratedPluginRegistrant.registerWith` only catches `Exception`. The
  resulting `ExceptionInInitializerError`/`UnsatisfiedLinkError` is an `Error`,
  so it escapes the registrant during `FlutterActivity.onCreate` and the
  Flutter view never attaches — a permanent black screen. `path_provider_android`
  pulls in `jni`/`jni_flutter`, so this affects every build of this app.
- Why it survived cleanups: the poisoned cache is in `~/.pub-cache`, not in the
  project `build/` tree, so deleting build intermediates never cleared it.

## Fresh installs use `Download/pocketclaw`; old data is left alone

- Date: 2026-08-25
- Decision: The Android workspace directory for fresh PocketClaw installs is
  `Download/pocketclaw`, including the no-permission fallback and the
  log-export directory. This supersedes the earlier decision to keep the
  compatible `Downloads/picoclaw` path.
- Consequence: Any existing `Download/picoclaw` tree is left untouched and no
  migration runs at startup. A migration, if it is ever wanted, is a separate
  designed feature and must not block the first frame.

## Debrand the embedded web runtime at source, never by binary patching

- Date: 2026-08-25
- Decision: PocketClaw wording in the embedded web console comes from edits to
  the Core web frontend/backend source, rebuilt through the Core Makefile
  targets. The compiled `.so` files are never patched.
- Consequence: `core/pocketclaw-core-v0.3.1.patch` and `core/README.md` carry
  the exact source diff and build commands. Build only via
  `make build-launcher-android-arm64`; calling `make -C web build-android-arm64`
  directly drops the root `LDFLAGS` and yields an unstripped binary, which is
  what produced the earlier oversized `libpicoclaw-web.so`.
- Consequence: use `pnpm lint` on the frontend, not `pnpm check` — the latter
  runs `prettier --write` across the whole tree and rewrites unrelated files.

## The release build fails closed on an incomplete arm64 payload

- Date: 2026-08-25
- Decision: `packageRelease` verifies that the APK contains
  `lib/arm64-v8a/libdartjni.so`, `libpicoclaw.so`, and `libpicoclaw-web.so`,
  and fails the build otherwise.
- Reason: the black-screen regression shipped for days because a missing
  `libdartjni.so` is silent at build time and fatal at launch.
- Consequence: `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64`
  is the canonical PocketClaw release path. Do not use a universal
  `flutter build apk --release` for releases.

## The black-screen incident is closed on physical evidence

- Date: 2026-08-25
- Decision: Treat the black-screen incident as RESOLVED and treat APK
  `2717f32e9580cd5b5ea5da70b2cb9fcf13f6f14451423addcb5686e0278a1de4` as the
  reference physically verified PocketClaw artifact.
- Evidence: a physical Android device passed install, app launch, Flutter first
  frame, Gateway/Core startup, navigation, PocketClaw branding, PocketClaw
  workspace path, and the QR/access page, with no abnormal device slowdown.
  This confirms the missing `lib/arm64-v8a/libdartjni.so` diagnosis and the
  build-pipeline fix.
- Consequence: the release guard and the canonical arm64 Gradle command are
  permanent parts of the release process, not temporary debugging aids. Removing
  either reopens the failure mode that caused this incident. Any future
  regression should be compared against this artifact before new hypotheses are
  formed.

## MQTT defaults to `/pocketclaw`, but never rewrites a configured prefix

- Date: 2026-08-25
- Decision: `mqtt.DefaultTopicPrefix` is `/pocketclaw`. `topicPrefix()`
  substitutes it only when the configured value is empty, so any explicitly
  configured prefix — including the legacy `/picoclaw` — is returned unchanged.
- Reason: the topic prefix is a broker-side contract shared with every other
  subscriber. A fresh PocketClaw install should not advertise the upstream name,
  and an existing deployment must not have its topics moved underneath it.
- Consequence: the Go default and the frontend preview/placeholder/hint must be
  changed together or the preview will misreport the topic an unconfigured
  channel publishes on. Because `topic_prefix` is `omitempty`, a config that
  relied on the *implicit* old default is indistinguishable from a fresh one and
  will move to `/pocketclaw`; pinning it would need a config migration, which is
  deliberately out of scope. Set `topic_prefix` explicitly to keep the old topic.

## The upstream agent skill is not seeded, and not renamed

- Date: 2026-08-25
- Decision: `skills/picoclaw-agent` is excluded from a freshly seeded workspace
  through the `unseededTemplates` list in
  `cmd/picoclaw/internal/onboard/helpers.go`.
- Reason: the skill documents the real upstream `picoclaw` CLI and repository
  internals. Rebranding its text would make its instructions technically wrong,
  so the choice was to seed it or not, and PocketClaw does not.
- Consequence: seeding only ever writes files, so a user who already has that
  skill keeps it, and it can still be installed later from a registry or by
  hand. The general skill loading/install mechanism is untouched.

## Factual third-party hardware references are kept accurate

- Date: 2026-08-25
- Decision: Sipeed, LicheeRV Nano, MaixCAM, and NanoKVM stay in the `hardware`
  skill, as do the real `picoclaw` binary name and `~/.picoclaw/workspace` path
  in shell examples.
- Reason: these are factual references to third-party hardware and to the real
  executable, not PocketClaw product branding.
- Consequence: do not rewrite documentation to reach a superficial zero string
  count. The branding audit classifies occurrences rather than merely counting
  them.

## Milestone B closes on physical evidence and merges non-destructively

- Date: 2026-08-25
- Decision: Treat APK
  `ba4f067df9811bd0e4af713343bdba632abbf96a41e3a5b47cf154740f70a4b8` as the
  verified reference artifact and Phase 2 Milestone B as COMPLETE.
- Evidence: a physical Android device passed install, launch/first frame,
  Gateway/Core lifecycle, navigation, branding, workspace path, QR/access page,
  Skill Hub, and the provider/model flow, with no black screen and no abnormal
  slowdown.
- Decision: merge `recovery/pocketclaw-clean-debrand` into `develop` with
  `--no-ff`, keeping the recovery branch and its history intact. `main` is not
  updated yet, and no branch is deleted or rewritten.
- Consequence: `phase2-milestone-b` tags the closure point, so any later
  regression can be bisected against a known-good, physically verified state.
  Milestone C stays unstarted until the user authorizes it.

## The provider catalog stays backend-owned; PocketClaw extends it

- Date: 2026-08-25
- Decision: Milestone C extends `pkg/providers.modelProviderOptionsByName` and
  the `provider_options` payload rather than creating a PocketClaw-side catalog
  in the UI or in Flutter.
- Evidence: the Core already exposes a 42-entry catalog through
  `ModelProviderOptions()`, and `web/frontend/.../provider-registry.ts` is a
  pure projection of that payload with no provider URLs of its own.
- Consequence: there is exactly one place to add or correct a provider. A
  second catalog in the UI would drift from the one the Core actually dispatches
  on. The full audit is in `docs/PROVIDER_ARCHITECTURE.md`.

## A provider preset is only real when the protocol switch knows it

- Date: 2026-08-25
- Decision: every preset added to the catalog is added in the same change to
  the OpenAI-compatible arm of `CreateProviderFromConfig`, and a Go test asserts
  that each PocketClaw preset actually constructs a provider.
- Reason: `CreateProviderFromConfig` ends in `default: unknown protocol %q`. A
  catalog-only addition looks correct in the UI, saves successfully, and then
  fails on the first message — the worst possible failure shape for a preset
  whose entire purpose is to remove guesswork.
- Consequence: `TestPocketClawPresetsResolveToAProvider` fails the build rather
  than shipping a preset that cannot run.

## Only presets that can be specified accurately are shipped

- Date: 2026-08-25
- Status: SUPERSEDED for OpenCode on 2026-08-25 — the user supplied the
  verified official endpoints, so both now ship. See "OpenCode Zen and Go are
  mixed-protocol gateways" below. The principle itself stands.
- Decision: xAI, Together AI, and Fireworks AI ship as presets. OpenCode Zen and
  OpenCode GO do not.
- Reason: the first three are Bearer-auth OpenAI-shaped APIs whose base URLs are
  well established. For the two OpenCode entries, the base URL, authentication
  header, and model-listing endpoint could not be established from the pinned
  Core or anything else in this workspace.
- Consequence: a guessed base URL would produce a preset that fails with a
  confusing error, which is worse for the user than not offering it. Both remain
  configurable today through Custom OpenAI-Compatible, and can be promoted to
  presets once their configuration is verified.

## Custom OpenAI-Compatible has no default base URL

- Date: 2026-08-25
- Decision: the `custom-openai` catalog entry deliberately ships with an empty
  `DefaultAPIBase`, and its base URL is a required field in the normal flow.
- Reason: falling back to a default would silently point a user's "custom"
  endpoint at OpenAI and produce an authentication error that describes the
  wrong service.
- Consequence: an empty base surfaces as an explicit configuration error. This
  is the supported path for self-hosted, VPS, local gateway, and unlisted
  OpenAI-compatible providers, and it is why the previous workaround — provider
  `openai` plus a custom `api_base` — is no longer the only option. Existing
  entries configured that older way keep working untouched.

## Gemini discovery needs its own fetch branch, not a flag

- Date: 2026-08-25
- Decision: `gemini` gains `SupportsFetch`, together with a dedicated branch in
  `fetchUpstreamModels`.
- Evidence: the shared fetch path sends `Authorization: Bearer`, but the Gemini
  provider authenticates with `X-Goog-Api-Key`
  (`pkg/providers/httpapi/gemini_provider.go`), and Google's native listing
  returns `{"models":[{"name":"models/<id>"}]}` rather than the OpenAI shape.
- Consequence: the branch keys off the base URL. A base ending in `/openai` is
  treated as Google's OpenAI-compatibility surface and uses Bearer with the
  standard response shape; anything else uses `X-Goog-Api-Key` and strips the
  `models/` prefix. URL construction stays base-relative, so a custom Gemini
  proxy path keeps working. Merely flipping the flag would have produced 401s.

## The alias is derived, not demanded

- Date: 2026-08-25
- Decision: `model_name` is no longer a required field in the normal Add
  Provider flow. When left blank it is derived from the model ID, with a numeric
  suffix on collision, and it stays editable under Advanced.
- Reason: the alias is a local label with no protocol meaning, but it was the
  first required field in the old form — asking the user to invent a name before
  they had even chosen a model.
- Consequence: the Core contract is unchanged; `model_name` is still required
  and unique at the API layer. Only who supplies it has changed.

## Provider marks are rendered locally, never fetched

- Date: 2026-08-25
- Decision: `provider-icon.tsx` renders a local text mark. The runtime requests
  to `cdn.simpleicons.org` and `https://www.google.com/s2/favicons` are removed.
- Reason: on a mobile device those requests disclose to two third parties which
  AI providers a user has configured, and they leave a broken mark whenever the
  device is offline or behind a restrictive network.
- Consequence: no provider logo assets are bundled and no licensing question is
  raised. Verified absent from the built binary: 0 occurrences of either host.

## Android API key storage is a separate milestone

- Date: 2026-08-25
- Decision: Milestone C does not change how API keys are stored.
- Evidence: `SaveConfig` encrypts keys to `enc://` only when
  `PICOCLAW_KEY_PASSPHRASE` and an SSH key are present, neither of which exists
  on an Android device, so keys are written as plaintext in the workspace
  config. Transport and UI exposure are already sound: `GET /api/models` returns
  `maskAPIKey(...)`, and `PUT` preserves the stored key when `api_key` is empty.
- Consequence: moving to Android Keystore touches config loading, the secret
  resolver, and migration of existing config files. Folding that into a UX
  milestone would put the verified Core config path at risk. It is recorded in
  `docs/PROVIDER_ARCHITECTURE.md` §6 and deferred to its own controlled
  milestone. Milestone C must not make the posture worse, and does not.

## OpenCode Zen and Go are mixed-protocol gateways, routed per model

- Date: 2026-08-25
- Decision: `opencode_zen` and `opencode_go` ship as presets, and the request
  protocol is resolved per model rather than fixed per provider. Every routing
  decision lives in `pkg/providers/opencode_routing.go`.
- Evidence: both gateways expose OpenAI Responses (`/responses`),
  OpenAI-compatible chat (`/chat/completions`), and Anthropic Messages
  (`/messages`) behind one base URL and one API key. Modelling either as a
  plain OpenAI-compatible provider would produce a preset that saves cleanly
  and then fails on the first inference — precisely the failure the catalog
  work exists to prevent.
- Consequence: the catalog entry carries only the base URL and key policy. The
  factory arm calls `ClassifyOpenCodeModel`, which matches the longest model-ID
  family prefix and returns both a protocol and whether the match was known.
  Model-name conditionals appear in that one file and nowhere else.
- Consequence: routing is family-based (`gpt-`/`codex` to Responses, `claude`
  to Messages, `kimi`/`deepseek`/`glm`/etc. to chat completions) rather than an
  enumerated model list. OpenCode adds and retires models frequently, and Fetch
  Models already returns the authoritative live list, so a pinned enumeration
  would be wrong within weeks.

## An unknown OpenCode model falls back, but never silently

- Date: 2026-08-25
- Decision: a model that matches no known family is still configurable and
  still attempted, using chat completions as the fallback, and the attempt logs
  a warning naming the model, the chosen protocol, and what a 404 would imply.
- Reason: erroring out would strand every future OpenCode model until PocketClaw
  ships an update, which is worse than a documented best guess. Routing it
  quietly would violate the rule against sending a knowingly-unverified request
  with no trace.
- Consequence: `ClassifyOpenCodeModel` returns `known bool` alongside the
  protocol, so no caller can mistake a fallback for a confirmed route. The
  warning never contains the API key.

## The Core needed a generic Responses provider, and now has one

- Date: 2026-08-25
- Decision: added `pkg/providers/openai_responses`, a Responses-over-HTTP
  provider driven by a caller-supplied base URL and bearer key.
- Evidence: the Core already spoke Responses twice, but neither was reusable —
  `azure` hardcodes Azure's deployment path, and `oauth/codex_provider`
  hardcodes `https://chatgpt.com/backend-api/codex` plus Codex-specific
  headers and instructions.
- Consequence: the new package reuses the shared
  `openai_responses_common` translation and adds only transport, so request and
  response handling stay in one place. It is the first Responses path in
  PocketClaw usable against a third-party gateway.

## The OpenCode Messages route sends both authentication headers

- Date: 2026-08-25
- Decision: for OpenCode's Anthropic Messages surface, the provider sends both
  `X-API-Key` and `Authorization: Bearer` with the same OpenCode key, via a new
  opt-in `anthropicmessages.WithBearerAuth()`.
- Reason: OpenCode issues one account key for every surface, and its OpenAI
  surfaces use the bearer form, but which form the Messages surface expects
  could not be established from anything available in this workspace. Sending
  both satisfies either convention. The option is off by default, so
  `api.anthropic.com` and the existing `anthropic-messages` and
  `alibaba-coding-anthropic` presets are byte-for-byte unchanged.
- Consequence: this is the one part of the OpenCode work that a green test
  suite cannot settle. If a `claude-*` model returns 401 on a device while
  `gpt-*` and `kimi-*` succeed, the answer is this header pair, not the routing.
- Consequence: `common.NormalizeBaseURL` strips and re-appends `/v1`, which is
  a no-op for both OpenCode bases. That round trip is asserted directly, since
  a base of `.../zen/go/v1` silently becoming `.../zen/v1` would be a
  hard-to-spot production failure.

## A catalog entry must never outlive its ability to run

- Date: 2026-08-25
- Decision: `TestEveryHTTPChatProviderInCatalogIsConstructible` asserts that
  every HTTP-API provider in the catalog that can drive a chat model
  constructs from a plain key-plus-base configuration, and fails if fewer than
  30 providers were exercised so the assertion cannot quietly become vacuous.
- Reason: the per-preset test only covers presets someone remembered to list.
  This one covers the catalog itself, which is the actual contract: anything
  offered in the picker has to work at inference time.

## The upstream adoption timeline is recorded, not remembered

- Date: 2026-08-25
- Decision: PocketClaw permanently records which upstream release and commit it
  started from, when it adopted them, and what upstream state was known at each
  review. `UPSTREAM_BASELINE.md` holds the immutable adoption record
  (adopted 2026-08-24; Core `v0.3.1` at
  `2cf030d2fd3b871d7ec17e3be34c24688aac76da`; historical FUI reference
  `d689c94c1b67f625f70ec4111a9aa3f01be9cbb3`, tag `v0.1.4`).
  `UPSTREAM_TRACKING.md` holds a moving Checkpoint plus an append-only Upstream
  Review History. Existing review entries are never edited or deleted.
- Reason: without a durable checkpoint, every future "what is new in PicoClaw?"
  restarts from v0.3.1 and re-reviews ranges that were already classified. The
  checkpoint makes each review begin exactly where the last one ended.
- Consequence: the review workflow is fixed — read the reviewed-through commit,
  fetch upstream, compare only what came after it, classify, selectively adapt,
  then update the checkpoint and append a new review entry.

## Upstream version claims are verified against upstream, or marked NOT VERIFIED

- Date: 2026-08-25
- Decision: the "latest upstream release known at review time" field is only
  ever filled from real upstream metadata. If it cannot be checked during a
  review, it is recorded as `NOT VERIFIED` rather than guessed.
- Evidence: for Review 1 it was verified live on 2026-08-25 against the GitHub
  API. Core `releases/latest` and the full tag list both name `v0.3.1`
  (published 2026-07-03) as the newest version tag, and its commit is the
  PocketClaw baseline commit; the rolling `nightly` release tag points at that
  same commit. FUI `main` HEAD equals the baseline commit and equals tag
  `v0.1.4`. Core `main` has moved to
  `bbf6893ca7afad27f1d00a0f5a45982a549c6ed6` (2026-08-19).
- Consequence: PocketClaw is level with the newest upstream release and 19
  first-parent commits behind Core `main`. That range is bounded and recorded
  as unreviewed, so the gap is a known quantity rather than an open question.
  The raw `2cf030d2..main` count of 2583 is not that gap — Core `main` has
  merged an unrelated root history — so first-parent count is the figure
  review planning uses.

## The PocketClaw repository is the canonical source-of-truth

- Date: 2026-08-25
- Decision: the Core source is vendored into this repository at `core/src/` and
  is the only source the build reads. The external checkout at
  `/home/lordegypt/PocketCLaw/.upstream/picoclaw-core-v0.3.1` survives as a
  historical reference for upstream review and is no longer a build dependency.
- Reason: a build that reaches outside the repository is not reproducible by
  anyone but the machine that has that directory. A `git clone` of PocketClaw
  now contains every line of application and runtime source needed to build the
  APK. External toolchains — Go, Flutter, JDK, Android SDK/NDK, Node/pnpm — stay
  external, because they are tools, not source.
- Deliberately not a submodule: a submodule would reintroduce "clone that other
  repository first" as a precondition, which is the exact failure being fixed.
- Evidence: `core/verify-no-external-source.sh` renames the external checkout
  away, runs the full Core build, and restores it. It passed on 2026-08-25.
- Consequence: adapting an upstream change now means editing `core/src/` and
  regenerating the provenance patch. Upstream is an update source that is
  fetched for review only.

## Vendoring moves the source, not the authorship

- Date: 2026-08-25
- Decision: `core/src/` keeps upstream copyright headers untouched, keeps
  `LICENSE` in place, and `core/pocketclaw-core-v0.3.1.patch` is retained and
  regenerated as the record of exactly what PocketClaw changed.
- Reason: physically storing MIT-licensed upstream source in our repository
  changes where it lives and nothing else. The patch is what keeps the two
  readable apart — 52 changed files, of which 13 are PocketClaw-authored.
- Evidence: upstream `v0.3.1` plus the regenerated patch reproduces `core/src/`
  byte-for-byte, verified on 2026-08-25.
- Consequence: `core/regen-upstream-patch.sh` must be run after any change under
  `core/src/`, or the divergence record goes stale. Running it and getting an
  empty `git diff` is the verification.

## Two upstream paths are deliberately not vendored

- Date: 2026-08-25
- Decision: `assets/` and `pkg/seahorse/.omc/` are excluded from `core/src/`,
  and excluded from both sides of the provenance patch so they never appear as
  PocketClaw deletions.
- Reason: `assets/` is 13 MB of upstream README screenshots and marketing GIFs
  with no build role, and carrying PicoClaw marketing imagery inside PocketClaw
  works against the branding position. `pkg/seahorse/.omc/state/` is an upstream
  developer's tool-state file committed by accident; it contains no code and
  leaks an upstream contributor's home-directory path. Vendoring developer
  machine state was explicitly out of scope.
- Consequence: neither is referenced by any build target, and `core/README.md`
  records the omissions so a future reviewer does not read them as loss.

## Release binaries are built with `-trimpath`

- Date: 2026-08-25
- Decision: `-trimpath` is added to the four Android arm64 `go build` lines in
  `core/src/Makefile` and `core/src/web/Makefile`, and
  `core/build-android-arm64.sh` fails the build if the shipped binaries contain
  any `/home/`, `/Users/`, or `/root/` string.
- Evidence: this was not theoretical. Before the change the shipped
  `libpicoclaw.so` contained 2,501 absolute `/home/lordegypt/...` paths and
  `libpicoclaw-web.so` contained 1,346 — every compiled file's full path on the
  build machine, reachable from the user-facing Logs screen. Both are now zero.
- Reason: `-s -w` strips symbols but does not remove the file paths Go records
  for tracebacks. Moving the source into the repository would have replaced one
  set of developer paths with another, so the fix had to be the compiler flag,
  not the directory layout.
- Consequence: the binaries shrank by roughly 197 KB and 131 KB, and their
  hashes will not match any previously recorded pair. Build provenance now
  lives in `core/README.md` and `UPSTREAM_BASELINE.md`, which is where it
  belongs.

## Upstream's `onboard` ignore rule is anchored in the vendored tree

- Date: 2026-08-25
- Decision: `core/src/.gitignore` changes the bare `onboard` rule to `/onboard`.
- Reason: the bare rule sits in upstream's "Secrets & Config" block and was
  plainly meant for a generated root-level `onboard` artifact, but an unanchored
  pattern matches at every depth. It swallowed
  `cmd/picoclaw/internal/onboard/`, which carries PocketClaw changes to
  `helpers.go` and `helpers_test.go`. Upstream never noticed because those files
  were already tracked there; a fresh vendored copy would silently have dropped
  four source files.
- Consequence: this also had to be handled in `core/regen-upstream-patch.sh`,
  which force-adds both sides. Without that, the upstream baseline commit
  excluded the same files and the patch misreported unmodified upstream source
  as PocketClaw-authored additions.

## Go caches are toolchain, and live outside the repository

- Date: 2026-08-25
- Decision: the Go build and module caches moved from
  `.upstream/picoclaw-core-v0.3.1/.cache/` to
  `/home/lordegypt/PocketCLaw/.tooling/go/`, alongside the JDK, Flutter, pnpm,
  and Android SDK. `core/build-android-arm64.sh` points `GOCACHE`/`GOMODCACHE`
  there and accepts a `GO_CACHE_ROOT` override.
- Reason: 3.3 GB of cache had been living inside the external Core checkout,
  which made that directory a build prerequisite for a second, non-obvious
  reason. It also holds the Go 1.25.11 toolchain that `core/src/go.mod`
  requires and the system Go 1.22.2 cannot satisfy. Caches are regenerable
  build state, so they belong with the tools, not in the repository and not in
  a source checkout.
- Consequence: the caches were moved, not deleted, and a fresh machine can
  instead let Go use its defaults and download from `core/src/go.sum`.

## The source migration closes on physical evidence, and `-trimpath` is now proven end to end

- Date: 2026-08-25
- Decision: the self-contained source migration and the OpenCode provider
  completion are marked physically verified on APK
  `588bbec144fe0c84b8429f4f053a73b44b9b3e8d9f24e31dab04b2165ff3a90b`, which
  becomes the reference artifact, superseding Milestone C's
  `b3dd892b...bce569b`.
- Evidence: 18 device checks, all PASS, on a real Android device. The three
  that carry the argument: the user-facing Logs screen showed no developer
  absolute paths, which is the first on-device confirmation that `-trimpath`
  actually removed what the build-time assertion said it removed; Skill Hub
  search and Fetch Models both worked, which is end-to-end proof that the
  Android active-network DNS integration survived being rebuilt from a
  relocated source tree, since both fail closed without working DNS; and both
  OpenCode providers returned real responses, exercising per-model protocol
  routing live rather than by assumption.
- Reason this needed a device at all: the migration changed no feature. It
  moved where source is read from and added a compiler flag. A green build and
  a green test suite cannot distinguish "the runtime still works" from "the
  runtime still links", and the previous black-screen incident is the standing
  proof of that gap.
- Consequence: the binaries `cb9b2cde...fb895818` and `b6b356f7...656db9ba5`
  are now the device-proven pair. `785ccd94...56058c6` is retired without ever
  being tested; its functionality is contained in the verified artifact.

## The OpenCode Messages header pair is still unconfirmed

- Date: 2026-08-25
- Decision: despite the OpenCode PASS, the Anthropic Messages route is recorded
  as unconfirmed, not settled.
- Reason: the route sends both `X-API-Key` and `Authorization: Bearer` because
  the correct form could not be established offline, and only a `claude-*`
  model returning a real response settles it. The device report records
  "OpenCode Zen real request/response: PASS" without naming the model families
  exercised, so it does not distinguish a `claude-*` success from a `gpt-*` one.
- Consequence: a later 401 on a `claude-*` OpenCode model, while `gpt-*` and
  `kimi-*` succeed, points at the header pair rather than the routing table.
  Recorded as an open item in `TASKS.md` rather than closed by association.

## PocketClaw owns its Telegram onboarding service, end to end

- Date: 2026-08-26
- Decision: the managed-bot onboarding service is PocketClaw-authored and lives
  in this repository at `services/telegram-onboarding/`. It is a separate Go
  module with zero external dependencies, and it depends on no third-party
  onboarding provider at runtime — only on Telegram itself.
- Reason: the self-contained source-of-truth rule from the previous milestone
  applies to every part of the product, not just Core. An onboarding flow that
  ran through someone else's setup service would put a third party between a
  user and their own bot token.
- Deliberately a separate module rather than code inside `core/src/`: anything
  added under `core/src/` shows up in `core/pocketclaw-core-v0.3.1.patch` as
  divergence from upstream PicoClaw, which this is not. Keeping it outside
  leaves the upstream provenance record honest.
- Consequence: Hermes was read as a behavioural reference for the shape of the
  flow — a pairing session, a deep link, polling, a token handed back — and
  nothing else. No Hermes or Nous service, endpoint, or source is involved at
  build time or at runtime.

## The managed-bot flow is built on verified Telegram API, not on inference

- Date: 2026-08-26
- Decision: every Telegram capability this milestone relies on was checked
  against `core.telegram.org` before implementation, not inferred from another
  project's code.
- Evidence: managed bots are Bot API 9.6, published 2026-04-03. The changelog
  states support for
  `https://t.me/newbot/{manager_bot_username}/{suggested_bot_username}[?name={suggested_bot_name}]`
  — note that `?name=` is optional, which the square brackets make explicit.
  The reference documents `User.can_manage_bots` (Boolean),
  `Update.managed_bot` carrying `ManagedBotUpdated{user User, bot User}`,
  `Message.managed_bot_created` carrying `ManagedBotCreated{bot User}`,
  `getManagedBotToken(user_id) -> String`, and
  `replaceManagedBotToken(user_id) -> String`.
- One correction this produced: the field is `can_manage_bots`, not
  `bot_can_manage_bots`. The latter appears in discussion of the feature but is
  not what the API returns, and coding against it would have made the manager
  verification silently always fail.
- `allowed_updates` defaults to everything except `chat_member`,
  `message_reaction`, and `message_reaction_count`, so `managed_bot` arrives
  without opting in. The client names it explicitly anyway, so a future default
  change cannot quietly stop onboarding from working.
- Consequence: Telegram still shows its own confirmation screen and the user
  still presses Create. PocketClaw pre-fills the name and username; it does not
  and cannot create a bot without that confirmation, and nothing in the design
  pretends otherwise.

## Polling and token collection are separate endpoints

- Date: 2026-08-26
- Decision: `GET /telegram/pairings/{id}` never returns a bot token. A separate
  `POST /telegram/pairings/{id}/token` delivers it exactly once and destroys
  the session.
- Reason: polling is idempotent and happens every couple of seconds; delivery
  is a one-shot transition. Folding them together forces a choice between
  repeating the token on every poll and making polling destructive, and both
  are worse than one extra round trip.
- Consequence: "the token is single-use" is a property of the API shape rather
  than of careful client behaviour, and it is directly testable. A failed
  collection attempt with the wrong token does not burn the delivery.

## A pairing is matched by its suggested username, and fails closed otherwise

- Date: 2026-08-26
- Decision: an incoming `managed_bot` update is bound to a pairing by the child
  bot's username, which is why each pairing suggests a unique random one. If
  nothing matches, the update is ignored and the pairing expires.
- Reason: Telegram reports the created bot and its creator, but nothing that
  ties either back to the app instance that issued the link. The suggested
  username is the only carrier of that link, and it is one PocketClaw controls.
- Consequence: if the user edits the suggested username on Telegram's
  confirmation screen, the pairing will not complete. That is deliberate. The
  alternative — matching loosely, for instance to the only pending pairing —
  would let one user's bot be delivered into another user's app. The UI
  surfaces the expiry with a retry and the manual fallback.

## The pairing tells us who the owner is, so the bot is locked to them

- Date: 2026-08-26
- Decision: `ManagedBotUpdated.user` is written into the child bot's
  `allow_from` list in Core's configuration.
- Reason: Telegram tells us exactly which user created the bot. A freshly
  paired bot therefore answers only its owner rather than anyone who discovers
  its username. Manual setup cannot do this, because nothing in a pasted token
  identifies the owner.
- Consequence: this is a security improvement the automatic path gets for free,
  and one more reason it is the default rather than a convenience wrapper.

## Automatic onboarding reuses the existing Telegram channel, and never a second one

- Date: 2026-08-26
- Decision: onboarding writes the same `channel_list.telegram` entry that
  manual setup writes, merging into it rather than replacing it, and then
  restarts Core.
- Reason: the Telegram channel, its streaming behaviour, MarkdownV2 handling,
  and the PocketClaw agent identity behind `/start` are all already correct and
  device-verified. Onboarding is a configuration path, not a runtime.
- Consequence: existing settings — proxy, base URL, MarkdownV2, streaming,
  reasoning channel — survive pairing, and every other channel is untouched.
  Both paths converge on one implementation, so the branded `/start` reply
  needed no change.

## The app ships with no onboarding endpoint until an operator supplies one

- Date: 2026-08-26
- Decision: `POCKETCLAW_ONBOARDING_BASE_URL` is a build-time `--dart-define`
  that defaults to empty. When it is unset, the app reports that automatic
  setup is unavailable and offers manual token entry.
- Reason: no PocketClaw manager bot exists yet, so there is no endpoint to
  point at. Inventing one, or falling back to somebody else's, was not an
  option. A build with no endpoint has to say so.
- The URL must also be HTTPS. Plain HTTP would put the poll token, and once the
  bot token, on the wire in the clear; the app refuses rather than downgrades.
- Consequence: the whole flow is implemented and tested, and end-to-end
  verification is blocked on operator setup rather than on code. See
  `SESSION_HANDOFF.md` for exactly what the operator must create.

## Onboarding strings are not localized yet, and are not machine-translated

- Date: 2026-08-26
- Decision: the Telegram onboarding screen's text lives in
  `TelegramOnboardingStrings` rather than in `lib/l10n/*.arb`.
- Reason: this repository keeps all twelve locales at exact parity — 84 of 84
  keys in every file. Adding roughly thirty new keys means either breaking that
  invariant or shipping twelve languages of unverified translation into a
  product's locale files. Guessed translations look authoritative and are worse
  than visibly-English text.
- Consequence: the screen is English in every locale for now, one class holds
  every string so the move to `.arb` is mechanical, and the work is recorded as
  an open item in `TASKS.md` rather than left implicit.

## The onboarding service is a separate public repository

- Date: 2026-08-26
- Decision: the Telegram onboarding service moved from
  `services/telegram-onboarding/` in this repository to its own public MIT
  repository, `Lord1Egypt/PocketClaw-Telegram-Setup`. This repository keeps a
  pointer at `services/README.md`.
- Reason: it is infrastructure, not application source. The APK does not build
  from it and does not contain it — the app holds only a public HTTPS base URL
  supplied at build time. The self-contained source-of-truth rule covers what
  builds the APK, which is `core/src/` and `lib/`. A deployable service also
  needs its own deploy button, its own issue tracker, and its own README, none
  of which work from a subdirectory of an Android app.
- Consequence: the API contract is now a boundary between two repositories and
  is documented on both sides. It did not change during the extraction: the
  same three endpoints, the same fields, and the same 404-for-everything-gone
  behaviour the Flutter client already expects.

## In-memory pairing state could not survive contact with Vercel

- Date: 2026-08-26
- Decision: pairing state moved from a process-local Go map to a
  Redis-compatible key/value store addressed over its HTTP REST API.
- Reason: on Vercel the request that creates a pairing, the Telegram webhook
  that completes it, and the request that collects the token can each execute
  in a different function instance. A Go map works perfectly in development and
  fails in production intermittently, which is the worst possible failure mode.
  This was caught by auditing the storage before deploying rather than after.
- Two operations must be atomic, and both are done by the store rather than in
  application code: `SET username:… NX` claims a suggested bot username so two
  instances cannot hand out the same one, and `GETDEL token:…` delivers the
  child token so exactly one caller can ever receive it.
- Evidence: both `Store` implementations run against the same conformance
  suite, including a concurrency test asserting that exactly one of twelve
  racing callers receives the token. A separate test asserts the Redis path
  actually issues `GETDEL` and `SET … NX`, so a refactor to `GET`+`DEL` breaks
  a test rather than exactly-once delivery in production.
- Consequence: the in-memory store still exists for local development and
  tests, but the service refuses to treat it as production — `config.Load`
  reports it as a problem the setup page renders in full.

## Long-polling became a webhook, because serverless has no long-lived process

- Date: 2026-08-26
- Decision: `getUpdates` long-polling was replaced by a Telegram webhook at
  `POST /telegram/webhook`.
- Reason: long-polling needs a process that stays alive, which a serverless
  function is not. It also permits only one consumer per bot token, which
  conflicts with more than one instance.
- The endpoint is authenticated by the secret Telegram echoes in
  `X-Telegram-Bot-Api-Secret-Token`, compared in constant time before the body
  is parsed. An undecodable body still answers 200, because a non-200 makes
  Telegram retry an update that can never be parsed.
- Consequence: registration is a server-side action. The setup page has a
  button that asks the server to call `setWebhook`; the browser never receives
  the token. This is deliberately unlike the earlier DukeBot pattern, where a
  Telegram token reached browser JavaScript.

## The operator page asks the server to act, and holds nothing

- Date: 2026-08-26
- Decision: the status page performs no privileged operation itself. Every
  button posts to an endpoint that reads credentials from the server
  environment and reports back a boolean plus a non-secret message.
- Reason: the page is public on a public deployment. Anything it holds is
  disclosed. Making it a remote control rather than a client keeps the manager
  token in exactly one place.
- Evidence: a test renders the page and asserts that none of the manager token,
  webhook secret, pairing secret, or storage token appears in it, and the same
  assertion covers `/api/status` and the register-webhook response.
- Consequence: the page is deliberately unauthenticated. Its actions are
  idempotent and expose nothing, and `SECURITY.md` records that as a considered
  choice with the option to put the deployment behind platform access control.

## PAIRING_SECRET keys the stored poll-token verifier

- Date: 2026-08-26
- Decision: poll tokens are stored as HMAC-SHA256 keyed with `PAIRING_SECRET`,
  not as a plain digest, and compared in constant time.
- Reason: storage is now a third-party service. A plain SHA-256 of 32 random
  bytes is already infeasible to reverse, so this is not about brute force —
  it is that a keyed verifier is inert to anyone holding a storage dump but not
  the server's key. Rotating the secret also invalidates every live pairing in
  one move, which is a useful thing to have if exposure is suspected.
- Consequence: the service refuses a secret under 16 characters rather than
  silently falling back to an unkeyed digest, which would drop the property
  without any visible symptom.

## Managed-bot onboarding is the default Telegram path; manual is the fallback

- Date: 2026-08-26
- Decision: Channels → Telegram opens managed-bot onboarding when Telegram is
  unconfigured and a host that can pair is present. The Bot Token / API Base
  URL / proxy / allow_from / typing / streaming / placeholder form is retained
  in full, behind "Advanced / Manual setup" when unconfigured and "Advanced
  Settings" once connected.
- Reason: asking a new user for a BotFather token was the single largest
  barrier to a working PocketClaw. The managed flow removes the copy/paste
  entirely and additionally scopes `allow_from` to the creating Telegram user,
  which manual setup cannot know.
- The manual path is not legacy debt and is not scheduled for removal. It is
  the only path for an existing bot, a self-managed bot, a custom Telegram Bot
  API endpoint, debugging, and any user who does not want managed onboarding.
  When no host or no compiled-in endpoint is present it is not merely available
  but primary, shown in full with nothing to expand.
- Evidence: physically verified end to end on an Android device on 2026-08-26,
  including a real Telegram → Core → AI provider → Telegram round trip.
- Consequence: the surface decision is a pure function,
  `resolveTelegramSurface`, and every one of its three outcomes is asserted
  through the real route component rather than in isolation.

## The web console asks the Flutter host to pair; it does not pair

- Date: 2026-08-26
- Decision: Channels → Telegram renders the onboarding entry point in React and
  hands off to the native Dart flow over a `PocketClawHost` WebView bridge. The
  pairing protocol has exactly one implementation, in Dart.
- Reason: Milestone D shipped a complete, fully tested onboarding flow that no
  user could reach, because it was wired to the native Settings tab while the
  Channels list users actually navigate is rendered by the Core web console.
  The obvious repair — reimplementing pairing in TypeScript so the console
  could run it — would have recreated the same split it was fixing, and would
  have needed CORS on the onboarding service plus a second copy of the polling,
  expiry, and config-merge logic to keep in step.
- The bridge is deliberately narrow: it publishes whether an endpoint was
  compiled in and the bot's public handle, and accepts exactly two requests.
  It carries no token. The handle is JSON-encoded so it cannot break out of its
  string literal, `openExternal` accepts only absolute `http`/`https` URLs so
  the bridge cannot become an arbitrary-launch primitive, and both injection
  and message handling are scoped to the console's own origin because the
  WebView will follow an outbound link if a user taps one. Unparseable input
  fails closed.
- Consequence: `TelegramOnboardingLauncher.open` is the single entry to
  onboarding and both surfaces call it. Adding a third surface means calling it
  too, never writing a second flow.

## A test that does not enter through the app's own route proves nothing about reachability

- Date: 2026-08-26
- Decision: any claim that a PocketClaw UI feature is reachable must be backed
  by a test that renders the same component the app renders — for Telegram,
  `ChannelConfigPage channelName="telegram"` — not the feature widget alone.
- Reason: Milestone D passed 77 Flutter tests, `flutter analyze`, the release
  guard, and a full secret scan, and still failed on the device. Every test
  constructed the onboarding widget directly, so all of them were true and none
  of them was evidence that a user would ever meet it. A green build was
  actively misleading here.
- Evidence: the replacement web tests were confirmed to fail against the
  pre-fix wiring — reverting the one line to `TelegramForm` failed all eight —
  before being accepted. A regression test that has never been observed failing
  is an assumption, not a test.
- Consequence: `jsdom` and `@testing-library/react` were added to the Core
  frontend's dev dependencies. That widens the upstream divergence, and it was
  accepted deliberately: the project had no DOM renderer, which is precisely
  why nothing could test the real route.

## Native Telegram Settings is a neutral shortcut; Core owns management state

- Date: 2026-08-26
- Decision: native Settings always shows `Telegram / Manage Telegram
  connection` and navigates to Core console `/channels/telegram`. It never
  displays connected/disconnected state and never starts pairing on card tap.
- Reason: physical evidence proved that duplicating Core's masked, split secure
  configuration in Flutter creates contradictory state. The Core page already
  has the correct connected/unconfigured, Open Chat, reconnect, and advanced
  behavior.
- Consequence: the native status reader/page was removed. Managed pairing is
  only an explicit host action requested by the canonical Core page.

## Core is the sole Telegram credential persistence authority

- Date: 2026-08-26
- Decision: Android sends a successful managed/manual credential to a
  loopback-only, per-process-authenticated, write-only Core endpoint. Core uses
  `SaveConfig`; the bridge exposes no GET and returns no token.
- Reason: masking and `.security.yml` precedence invalidate raw native
  `config.json` parsing/writing. Keeping the old bot untouched until a
  replacement succeeds also makes reconnect failure safe.
- Consequence: the random bridge credential is never persisted/logged and the
  Telegram token is never returned to Flutter.

## Telegram requests have bounded, independently completing lifecycles

- Date: 2026-08-26
- Decision: Telegram HTTP uses a 45-second deadline; every inbound gets a safe
  process-local correlation ID; same-session Telegram arrivals are FIFO full
  requests rather than steering; final delivery is synchronous; edit errors
  fall back to send and correlated placeholder cleanup.
- Reason: an unbounded outbound call matched the physical stall, while
  swallowed edit errors and chat-only placeholder keys could independently
  leave `Thinking` permanent or attach completion to the wrong request.
- Consequence: empty provider output, tool errors/timeouts, normal output, edit
  failure, and close arrivals release independently. Safe trace events locate
  future physical stalls without logging content or identity.

## Existing workspaces are repaired non-destructively to seven baseline skills

- Date: 2026-08-26
- Decision: Core console startup runs `onboard ensure-workspace`, which copies
  only missing embedded files. Fresh PocketClaw seeds seven bundled skills;
  `picoclaw-agent` remains deliberately unseeded but an existing user copy is
  never deleted.
- Reason: existing config skipped onboarding, so a workspace missing seeds
  could remain with only a later imported `gh`. Import itself is additive.
- Consequence: no hardcoded remembered count of eight; user/global skills can
  increase discovery, and invalid/missing SKILL metadata can correctly exclude
  a directory.

## User logs are plain Unicode text at one storage boundary

- Date: 2026-08-26
- Decision: sanitize Core output before it enters `ServiceManager.logs`; Logs
  and Export consume the same representation. Core also receives `NO_COLOR=1`
  and `TERM=dumb` and prints a plain Android banner.
- Reason: Android Logs is not a terminal. Display-only cleaning leaves exports
  dirty; stripping non-ASCII corrupts Arabic and emoji.
- Consequence: terminal protocols/control bytes are removed, printable Unicode
  is preserved, and the exactly-once queue remains the transport model.

## Background/battery and runtime statistics remain future work

- Date: 2026-08-26
- Decision: add neither battery-management hacks nor telemetry UI to this
  pre-release fix. Record a future user-guided Background & Battery page and a
  local-by-default Runtime / Statistics tab.
- Reason: neither expands the evidence-backed current fix, and PocketClaw
  cannot silently grant itself unrestricted battery operation.

## Export Logs encodes the sanitized representation as UTF-8

- Date: 2026-08-26
- Decision: Android Export Logs converts the already-sanitized Dart string with
  `utf8.encode` before sending bytes through the MediaStore MethodChannel.
- Reason: Dart strings expose UTF-16 code units. Truncating those units into an
  8-bit list turned valid `µ`, Arabic, and emoji into malformed bytes even
  though Go output, native string transport, sanitization, and the Logs UI were
  Unicode-safe.
- Consequence: export tests must exercise the actual platform-channel byte
  boundary and strict UTF-8 decoding, not only the standalone sanitizer. No
  global replacement of U+FFFD, `µ`, or non-ASCII text is permitted.

## Routine successful log/status self-polls are not diagnostics

- Date: 2026-08-26
- Decision: HTTP middleware omits only exact 2xx `GET` requests to
  `/api/gateway/logs` and `/api/gateway/status` from DEBUG output.
- Reason: those UI monitoring requests generated the history they were reading
  and displaced useful diagnostics without representing failures.
- Consequence: poll failures, redirects, unexpected methods, unknown routes,
  and every other API request remain logged. This filter is not deduplication
  and must never replace the exactly-once `publishLog`/`takeNewLogs` drain.

## User-visible log surfaces share one plain-text contract

- Date: 2026-08-27
- Decision: normalize captured gateway output before `LogBuffer` storage, and
  enforce the same canonical fixture contract at the existing native boundary
  and as an idempotent browser guard. Web Logs render plain text rather than
  terminal styling.
- Reason: native/export and Web Console followed separate paths. The Web ring
  stored raw output while its SGR-only renderer left non-SGR terminal protocols
  and startup artifacts visible.
- Consequence: ANSI/OSC/cursor/erase/CR/backspace/control data cannot reach a
  normal rendered surface, while Arabic, emoji, `µs`, and legitimate printable
  Unicode remain intact. This is not an ASCII conversion or brand-wide string
  replacement.

## Compatibility identifiers stay internal, with source-level neutralization

- Date: 2026-08-27
- Decision: keep `/pico/ws`, `libpicoclaw.so`, `libpicoclaw-web.so`, module
  paths, and environment names unchanged. Omit only successful routine
  WebSocket events, reword failures to `/internal realtime connection`, and
  omit the executable path from the startup message. Captured no-color startup
  uses a single PocketClaw text banner rather than block art.
- Reason: those identifiers are runtime compatibility contracts, not product
  copy. Renaming them would add release risk; exposing them adds no normal user
  diagnostic value.
- Consequence: real failures remain observable without leaking implementation
  naming, and interactive/internal compatibility behavior is preserved.

## The internal Pico transport has a PocketClaw-only summary label

- Date: 2026-08-27
- Decision: when formatting only the gateway startup/reload enabled-channel
  summary, display exact `config.ChannelPico` as `pocketclaw`.
- Reason: `pico` is the internal Web Console WebSocket/media protocol and config
  identity, but the raw ID is not meaningful product copy.
- Consequence: `/pico/ws`, `/pico/media`, channel/config IDs, factories, tokens,
  session semantics, and other compatibility identifiers remain unchanged. The
  formatter copies its input and performs no substring/global replacement.

## Internal WebSocket logger identities have display-only names

- Date: 2026-08-27
- Decision: at the shared user-visible normalization boundary only, map exact
  structured logger component `pico` to `realtime` and exact caller basename
  `pico.go` to `realtime.go`, preserving the caller line number.
- Reason: logger component and source basename are implementation details on
  normal product diagnostic surfaces, while the underlying compatibility
  identities cannot safely be renamed.
- Consequence: Native Logs, Export Logs, and Web Console Logs agree. Actual Go
  package/file, `ChannelPico`, routes, config, and protocol stay unchanged;
  substring matches are explicitly rejected by regression tests.

## Web Logs own an explicit pre-paint scroll policy

- Date: 2026-08-27
- Decision: identify frontend log events by Core run ID plus absolute append
  offset; render memoized plain-text rows with browser-native wrapping; track
  whether the viewport is within 24 px of bottom from actual scroll events; and
  apply bottom following in `useLayoutEffect` only for followers.
- Reason: passive post-paint correction exposed a transient old scroll offset,
  while measuring the whole changing content box to hard-wrap text could
  rewrite long rows on append. Array-index keys did not express event identity.
- Consequence: new logs remain live and bottom followers remain pinned without
  an animated bounce. Scrolled-up users receive no forced movement. An
  unchanged long entry keeps the same DOM node and text across polling updates;
  native/export transport and shared sanitization remain untouched.

## Third-party Telegram logs retain no credential fragments

- Date: 2026-08-27
- Decision: replace the compatible logger's partial Telegram token mask with
  complete pre-writer redaction. At Web `LogBuffer` normalization, replace Bot
  API URLs with `Telegram API call: <operation>` and redact Authorization
  credentials; keep a browser-side idempotent guard for legacy/raw lines.
- Reason: the full token was already masked before stdout/storage, but keeping
  the bot ID and secret prefix/suffix exposed credential fragments in normal
  Web Console diagnostics. Those fragments add no troubleshooting value.
- Consequence: Web history retains HTTP method, Bot API operation, status,
  timeout/error, and latency without any credential fragment. This finding is
  not evidence of full-token persistence, and requires no automatic credential
  rotation. Native/Export implementation and Telegram lifecycle remain
  unchanged.

## Classified compatibility fields receive semantic display normalization

- Date: 2026-08-27
- Decision: at the shared Web pre-storage log boundary, map only exact
  structured `channel=pico` / `type=pico`, hide exact `path=/pico/` only on a
  line carrying that internal channel identity, replace only classified Pico
  protocol/reasoning messages, and semanticize the exact compatibility PID
  path. Keep the browser guard idempotent.
- Reason: these are runtime implementation details visible in normal product
  diagnostics, but their underlying identifiers are compatibility contracts.
- Consequence: ChannelPico, serialized config, Go packages/files, routes,
  `.picoclaw.pid`, libraries, environment variables, and provenance remain
  untouched. Security and failure diagnostics remain visible. Exact-token and
  exact-message matching explicitly leaves unrelated `pico` substrings alone.

## Orphaned CSI recovery is numeric-only

- Date: 2026-08-27
- Decision: retain standards-compliant ESC/C1 CSI removal, but recognize a CSI
  suffix whose introducer was lost only when its parameters begin with a digit
  or `?` plus numeric parameters. Preserve all other printable angle-bracket
  text. Normalize the exact successful Telego nil field to `Err: none`.
- Reason: Telego correctly emitted `[<nil>]`; the old permissive suffix pattern
  treated `<` as a parameter and `n` as a final byte, deleting `[<n` before Web
  storage. An introducer-less control cannot safely use the full ambiguous CSI
  grammar on ordinary text.
- Consequence: numeric orphaned RGB/SGR/cursor controls remain removable,
  `<nil>`, comparisons, tags, Arabic, emoji, and punctuation remain valid text,
  and React continues to escape rather than interpret them as HTML.

## Routine empty Telegram long polls are pre-writer noise

- Date: 2026-08-27
- Decision: for exact DEBUG component `telego`, omit Bot API `getUpdates`
  request lines and exact successful responses with nil error plus `Result: []`
  before any writer. Keep idempotent storage/render guards for raw/legacy input.
- Reason: Telego logs the request before it knows the result, then emits a
  distinct execution-error or API-response line. Omitting the request ensures
  an empty success is silent while a later failure/non-empty response still
  carries the operation and useful details.
- Consequence: routine 30-second polling cannot consume history. Failures,
  non-empty updates, non-success responses, other Bot API operations, token
  redaction, and Telegram runtime behavior are unchanged.

## Normal logs are metadata-only at the pre-writer boundary

- Date: 2026-08-28
- Decision: normal PocketClaw logs omit prompt/message/response/reasoning/tool-
  argument bodies, redact exact session and internal identity fields on a copy,
  and retain lifecycle counts, lengths, model, tool names, status, duration and
  safe failures. Telego payloads normalize before stdout to operation/status and
  non-empty update count/type metadata.
- Reason: Web DEBUG previously stored and rendered full Agent payloads and
  Telego result JSON containing conversation content and personal identifiers.
  Frontend-only masking would leave those values in backend history.
- Consequence: runtime routing/session/Telegram values are not modified. Fresh
  default assistant identity is PocketClaw, while user-authored prompt overlays,
  internal compatibility identifiers, legal attribution and provenance remain
  untouched. Backend/React/Dart guards are defense-in-depth for historical/raw
  input, not the primary privacy boundary.

## The managed Core gateway is pinned to loopback, unconditionally

- Date: 2026-08-29
- Decision: `gatewayHostOverride()` always returns `localhost`. The launcher
  exports it to the gateway child through `PICOCLAW_GATEWAY_HOST`, and the
  environment value wins over the config file, so Core on 18790 binds to
  loopback no matter what `gateway.host` or `-host`/`-public` say.
- Reason: Core speaks to the agent and providers with an installation-random
  bearer credential and no password/session layer. The authenticated Dashboard
  on 18800 is the only ingress that should ever face a LAN.
- Consequence: a Core reachable from the LAN is now a structural impossibility
  rather than a configuration discipline. `gateway.host` no longer influences
  the managed gateway's bind; a future deployment that genuinely needs a
  non-loopback Core must reintroduce an explicit, separately reviewed path.

## Public Mode rebinds the Dashboard listener, never the Core process

- Date: 2026-08-29
- Decision: an in-process listener supervisor owns port 18800 and can replace
  only its listeners. Applying Public Mode closes the old bind (including
  hijacked WebSocket connections that `net/http` no longer owns after upgrade),
  opens the new one, rolls back to the previous bind on failure, and persists
  the setting only after the new bind succeeds.
- Reason: the listener used to be created once at service startup, so changing
  Public Mode in the UI updated persistence without moving the socket. The
  obvious alternative — restarting the service — would drop the agent runtime,
  session store, and Telegram polling for a network-scope change.
- Consequence: OFF→ON and ON→OFF apply live and the LAN URL and QR refresh on
  their own. A browser connected over the old bind loses its socket but keeps
  its session cookie and can reconnect through the new listener. Android learns
  the real outcome over the authenticated loopback bridge and only then writes
  the preference, so a failed bind cannot leave the UI claiming a mode the
  socket is not in.

## Owner identity is server-derived; clients never assert it

- Date: 2026-08-29
- Decision: the internal realtime channel binds every inbound message to its
  authenticated connection and to a Core-owned owner principal, and it is
  constructed with an owner-only allowlist regardless of what is on disk.
  Dashboard sessions live in a server-side store that can be revoked. Telegram
  refuses to start without exactly one paired numeric owner.
- Reason: a client-supplied session or sender field, or a stale permissive
  `allow_from`, must never become the effective authorization identity.
- Consequence: forged sender/user/session/owner/chat fields cannot bypass
  authorization, and a Telegram username can never stand in for the numeric ID.
  Onboarding fails closed for empty, wildcard, and username-only owners rather
  than degrading to "allow anyone".

## Ship the RC as 0.2.0, not the inherited 0.1.3 baseline

- Date: 2026-08-29
- Decision: the app version moves to `0.2.0+4` for `v0.2.0-rc1`.
- Reason: the version had carried the inherited FUI `0.1.3+3` since bootstrap.
  A release tagged `v0.2.0-rc1` whose APK reports `0.1.3` is an artifact
  identity that cannot be explained to a tester.
- Consequence: the version code advances 3 → 4 so the candidate installs over
  the previous build. The physically validated APK reported `0.1.3 (3)`; the
  released candidate differs from it in version metadata only, and carries a
  byte-identical Core native payload.

## Build the Core with the `goolm` tag rather than accepting a libolm gap

- Date: 2026-08-29
- Decision: Go verification runs as `-tags goolm,stdjson`, which selects the
  pure-Go Olm implementation.
- Reason: earlier sessions recorded "the full Go sweep fails only because
  optional Matrix support needs `olm/olm.h`" as an accepted host-dependency
  exception. It was avoidable: with `goolm`, `go build ./...`, `go vet ./...`,
  and the complete test suite are green with no system libolm present.
- Consequence: there is no longer any accepted failing region in the Go sweep,
  so a genuine future failure cannot hide behind a Matrix/libolm explanation.
