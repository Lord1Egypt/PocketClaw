# Development Changelog

## 2026-09-08 — Two things were in the same directory for no reason

Release Hardening A2. The workspace, the gateway credential and the diagnostic
log all lived under `Download/pocketclaw` because they all started there, not
because they belong together. One of those three is a product feature; the other
two were a credential and a prompt archive sitting where any app with storage
access could read them.

The credential is the sharper problem. It authenticates `POST /reload` and
detailed `/health`, and it is well made — CSPRNG, rotated every gateway start,
never logged, never in a URL, compared constant-time. The defect was purely
placement: it was a field inside `.picoclaw.pid`, and that file is on shared
external storage where the 0600 it is written with is synthesised by the
filesystem rather than enforced. Android does not isolate loopback sockets
between apps either, so reading that file was one step from using it.

Splitting it was mostly a question of who owns the secret. Core keeps generating
it, which is what preserves per-start rotation and leaves desktop and server
installs untouched; it writes to `PICOCLAW_GATEWAY_TOKEN_FILE` when a host names
one, and the record is then serialised from a copy with the token cleared. The
`omitempty` on that field is the entire mechanism, which is worth saying out
loud because deleting one struct tag would silently undo the milestone. The
Android host names a path under `noBackupFilesDir` — the same boundary the
realtime credential already used — and `HealthChecker` reads a bare token from
it rather than parsing JSON, so there is no adjacent field to pick up by
accident.

The logs turned out easier than expected, and the reason is worth recording:
nothing reads the file. The in-app Logs screen reads a 200-line in-memory buffer
fed from the child process's stdout, and no Dart or Kotlin code opens
`gateway.log` at all. So moving it was a one-line resolver plus an environment
variable, with no UI consequence. What it did need was rotation, because there
was none — pure append, which is how a real install ended up with an 18 MB file
still holding lines an August build had written, including full LLM requests and
system-prompt previews from before the logger was cleaned up.

That history is why the cleanup exists, and why it is as narrow as it is: three
exact filenames, no pattern matching, no recursion, and the directory removed
only if those were all it contained. Everything else under that path is the
user's, and the rule from the lobster investigation still stands — application
output can be deleted, user content cannot.

Redaction got three rules rather than one. The temptation is a single "redact
anything long and random" pattern, which would eat session keys, source
fingerprints, model names and file paths; there is now a test asserting exactly
those survive, alongside one asserting a dozen fabricated credential shapes do
not.

The realtime channel's `==` became `subtle.ConstantTimeCompare`, matching what
the health server has always done, and query-string authentication is now
refused outright when the credential came from the host. The Dashboard toggle is
hidden too, but the runtime check is the real fix: hiding a control leaves a
config file able to re-enable it.

Everything here is keyed on a compatibility name the coming migration will
rename. That is written down in three places, because a rename on one side only
puts the credential and the logs back where they were without breaking anything
anybody would notice.

## 2026-09-08 — vc56 proved the version came from the file we said it did

Release Hardening A1 merged to `develop` with `--no-ff`, physically accepted as
vc56 on SM-A165F. The interesting thing about the validation build is what it
did *not* pass on the command line.

vc56 was built with `-Ptarget-platform=android-arm64 -PallowDebugSigning=true`
and nothing else. No `-PversionCode`. No version in `local.properties` — those
two lines were deleted during A1. So the `versionCode=56` in the packaged
manifest could only have come from `pubspec.yaml`, and that is the entire proof
that the tracked source is now authoritative. A build that produced 1, or 55,
would have failed the milestone on its own terms.

The upgrade also had a gate worth keeping. Before installing, the signing
certificate of the built APK was compared against the certificate of the
`base.apk` pulled off the device — both `15cf75f9…`, so `install -r` was a real
in-place update and `firstInstallTime`, `dataDir`, uid and application data all
survived. That check is not ceremony: the moment a production key exists, it
will fail, and it should, because a different signer cannot update an
installation in place. Better to learn that from a gate than from a device.

On permissions the device reported 11 where the APK declares 13, which looks
like a discrepancy and is not. `WRITE_EXTERNAL_STORAGE` caps at API 28 and
`READ_EXTERNAL_STORAGE` at 32, and the validation device is API 36, so Android
drops both from the requested set. vc55 had the same two-entry gap. What changed
is the six that went away — the telephony permission, the advertising ID, the
two AdServices entries, the Play install-referrer binding and the OEM push
permission — with nothing added in their place.

The accepted baseline advanced from 55 to 56 in the closeout commit, which is
the only place it is allowed to move. Bump `pubspec.yaml` to build a candidate;
bump `android/release-baseline.properties` when that candidate survives a
device. Keeping those two events apart is what stops the floor from being a
number that agrees with whatever was built last.

## 2026-09-08 — Four defaults that produced the wrong artifact without failing

`feature/release-hardening-a1`, the first Production Release Hardening
milestone. Nothing in it is a feature; all four items are the same shape — a
default that quietly did the wrong thing and never said so.

The signing one is the worst. The release `signingConfig` selected the
production keystore if `storeFile?.exists()`, and otherwise fell through to
`signingConfigs.getByName("debug")`. With no `KEYSTORE_*` in the environment
that branch always won, so every artifact this project has produced — vc55
included, the one physically accepted two days ago — carries a local development
signing identity rather than a release one. The build printed nothing. It now resolves to a real signer, or to debug **only** under
`-PallowDebugSigning=true`, or to `null` with `validateReleaseSigning` failing
before anything compiles. The production key is deliberately not created yet:
the test device runs a debug-signed install and changing signers forces an
uninstall, so that is its own scheduled step rather than a side effect of this
one.

The version had the same shape. `pubspec.yaml` said `0.2.0+13` while the device
ran 55, because the Flutter Gradle plugin reads `flutter.versionCode` from the
gitignored `local.properties` and **defaults it to 1** when absent. So the
tracked file was wrong, the authoritative value was untracked, and a clean
checkout would have built versionCode 1 — an artifact Android refuses to install
over 55, for reasons that would have taken an hour to work out on a device.
Gradle now reads `pubspec.yaml` directly and rejects a version in
`local.properties` rather than obeying it.

The backup finding is the one with real data at stake. Core writes
`.security.yml` next to `config.json` under `files/picoclaw/`, and Android
onboarding answers "n" to credential encryption, so that file holds provider API
keys and channel bot tokens in plaintext. It is 0600 and app-private, which is
the right answer to "can another app read it" and no answer at all to "does it
leave the device". `allowBackup` is true and the rules excluded exactly one
directory — `credentials/` — so every provider key went to Google's backup
transport and to device-to-device transfer verbatim. One line in each of two
files fixes it. The interesting part is what keeps it fixed: the exclusion is
matched by path name, and the coming namespace migration renames exactly that
name, so the test asserts the rule files and
`PicoClawService.buildEnvironment` against the same literal.

The analytics item is where the audit turned out to be wrong, which is the
useful part. The hypothesis was that the unconditional Umeng dependency was
dragging in `AD_ID`, the AdServices pair and the Play install-referrer service.
Merging the release manifest with and without the dependency settled it: Umeng
contributes **one** entry, `freemme.permission.msa`, and `READ_PHONE_STATE` came
from our own manifest, not from the SDK. The four advertising permissions are
Firebase Analytics's, via `play-services-measurement`, and they stayed. Firebase
is unconfigured by default too, so it is the same problem — but its plugins
register from `pubspec.yaml` and unwinding them touches Dart and the
device-feedback surface, so it is recorded as its own decision instead of being
removed on a guess. Proving attribution took two Gradle manifest merges and
changed the answer.

Umeng itself is `compileOnly` rather than deleted. `AnalyticsReporter` compiles
against it unchanged and is already guarded at runtime, so the SDK stays off the
runtime classpath — and therefore out of the merged manifest — while an
analytics build asks for it by name and gets the real dependency.

One user-visible line came out of all this and went into What's New: the app no
longer asks for the Phone permission. Everything else here is invisible by
design, and release notes are not a changelog.

Review sent three things back. The build's own explanation of debug signing was
wrong — it said the debug key's private half ships with every SDK install, which
is not how Android debug signing works. Debug material is local development
material that varies between environments; the consequence worth stating is that
an artifact signed with a different key cannot update an existing installation in
place, and that is what the message says now.

The version floor was pinned at 55 in the build file, which would have kept
accepting 56 long after 120 shipped — a floor that never moves stops being one.
It now reads `lastAcceptedVersionCode` from a tracked
`android/release-baseline.properties`, advanced by hand in the commit that
records a physical acceptance. Setting it to 120 and watching both versionCode
55 and an override of 56 be rejected is the whole proof.

And the Firebase question got the trace it deserved rather than a guess.
`firebase_analytics` and `firebase_core` are used by exactly one file, behind
the device-feedback Settings toggle, and Firebase initializes only when four
dart-defines are set — all empty by default, with no `google-services.json`
anywhere. So it is inert in the default build but is a real feature, not dead
code, and deleting the plugin would have removed a feature to shorten a
permission list. The feature logs one custom event and needs no advertising ID,
so the four advertising permissions came out through `tools:node="remove"` —
Google's documented opt-out — with the components untouched. The default merged
manifest is 13 permissions now, down from 19 at vc55, and every one that is left
is doing a job.

The last item was a failure mode nobody had hit. Could a runtime provider
selection reach an SDK the APK did not package? It could not: the Kotlin guard
checked the same BuildConfig string that decided the Gradle dependency, and its
condition was strictly stronger. But that is a coincidence between two
independently editable conditions, not a contract, so the packaging decision now
sets `BuildConfig.PICOCLAW_UMENG_PACKAGED` and the guard reads it, with a
`LinkageError` catch behind that. An optional capability that was not built is
disabled, and says so.

## 2026-09-08 — The prompt was clean and the reply still signed itself

`feature/final-user-facing-polish` merged to `develop` with `--no-ff`,
physically accepted as vc55. Two changes: the About dialog moved onto the
Aperture tokens, and the shared kernel identity stopped carrying a mascot emoji.

The About work was the smaller half and the more ordinary. The dialog had been
redesigned around the edges but still had a fixed 148px label column inside it,
which a longer translation overflows and a narrow phone cannot afford. Stacking
each label over its value removes the width there was to get wrong; the values
keep their left-to-right monospace so a build stamp cannot reverse in Arabic.
Rendering both labels before the versions arrive stopped the dialog resizing
mid-open, which is the kind of thing nobody reports and everybody notices.

The identity change is the one worth writing down. `kernel.identity` is the
single prompt part every agent on every channel receives on every turn, so a
decorative character in it is not decoration — it is an instruction to imitate,
and the model was mirroring it at the end of replies. Deleting it from the
header is a one-character fix and the correct one; a formatter that strips the
character from generated text would also strip it from text a user asked for.

Then vc55 shipped, the Core was verified on the device to carry the new header,
and the first reply still signed off with the mascot.

The instinct at that point is to distrust the fix. The useful move was to
distrust the *scope* instead: the identity is one of about nine things that
reach the model, and only one of them is compiled in. Tracing all of them found
the shipped binary clean, every repository default clean, every persisted
persona file on the device clean, the skill catalog clean — it emits only names
and descriptions — and no session file carrying a system message at all. The
failing turn was a two-message session created after the install, so there was
no earlier reply to copy either.

What was left was the user's own long-term memory file, loaded verbatim into the
prompt on every turn, with exactly one mascot emoji in a Markdown heading that
happens to sit directly above the passage about who the assistant is. The reply
had also quoted two project names that appear only in that file, which is how we
knew it was in the prompt. Removing that single character — in the user's own
runtime file, not in the repository — was enough; the next neutral message came
back without a sign-off, and with a different emoji it chose for itself, which is
exactly the behaviour we wanted.

The general lesson is that "the source is fixed" and "the running system is
fixed" are different claims, and persisted user state is where they come apart.
The specific rule is narrower: no migration rewrites a user's memory, and no
filter edits a model's output.

That trace also caught something unrelated and real. Seeded workspace templates
are copied only when absent, so an install created before a default improved
keeps the old text indefinitely — the validated device is still running an
`AGENT.md` that predates the Managed Runtime section, meaning that agent has
never been told the Managed Runtime exists. Fixing it properly means versioning
the templates and merging rather than overwriting, so it is deferred rather than
patched in a closeout.

## 2026-09-07 — The launcher icon had to stop being its own drawing

`feature/final-launcher-icon` merged to `develop` with `--no-ff` at head
`7df0bf7`, physically accepted as vc54. The Android launcher was the glossy 3D
mark, and the visible problem was that it does not survive being 48px: gradients,
bevels and a specular orb mush at the size mdpi actually draws. The flat APERTURE
mark was refined specifically to hold together small, and it is the identity on
every other surface.

Two defects surfaced underneath the artwork question. The pre-26 icons had been
generated from a transparent source, so they carried no background tile — and
`minSdk` is 24, so on API 24/25 the launcher drew a floating mark on the
wallpaper. The adaptive path masked that from API 26 up, which is why nobody had
seen it. And `flutter_launcher_icons` still pointed its Android source at the 3D
mark, so any future run would have silently reverted the launcher no matter what
was committed.

The interesting decision was where the geometry lives. The mark already existed
in exactly three files, held together by a Core test, precisely because it
drifted the last time it was refined. Writing the paths again to emit Android
PNGs would have made a fourth copy of a thing with a known drift history. Core
was read-only in this milestone, so the export could not be added there either.
`tool/generate_android_launcher_icons.py` imports the canonical
`generate-brand-assets.py` by path and calls its `draw_mark`, which adds an
output without adding a source; the result is byte-reproducible.

`flutter_launcher_icons` was disabled for Android rather than repointed, because
an opaque legacy tile and a transparent adaptive foreground are different
pictures and it only takes one.

## 2026-09-07 — A status page is only worth having if you can believe it

`feature/status-dashboard-v1` merged to `develop` with `--no-ff` at head
`85eb93a`, physically accepted as vc53. The feature is small — one screen of
numbers — and almost all of the work went into making sure each number means
what its label says.

The investigation found the metrics were mostly already in memory. Active turns
and subagents are `activeTurnStates` ranged by depth. Queued work is the length
of the session mailboxes. Channel state is three maps the manager already keeps.
What had to be added was six atomics and a place to put them, and the places
chose themselves: `runTurn`'s terminal defer already owns a turn's final
classification, and `recordToolExecution` already owns a tool's.

Deriving them from the runtime event bus looked cleaner and is wrong. Its
subscriptions have a sixteen-event buffer and a drop policy, so a bus-derived
counter undercounts under exactly the load that makes someone open a status
page.

Three numbers shipped in the first draft claiming more than the code knows, and
an audit caught each one.

**"Failed to start" was a guess.** A channel with no worker might have failed,
or might simply not have been started yet — `StartAll` keeps its failures in a
local variable for a log line and throws them away. Both states produce the same
snapshot, so any label naming one a failure mislabels the other. The label is
gone; there is a test asserting the two snapshots are byte-identical, which is
where to start if a retained failure state is ever added.

**"Fallbacks" was off by one for every agent.** `resolveModelCandidates` puts
the primary first and `ExecuteCandidate` walks the whole list, so index 0 is the
model in use. An agent configured with no fallbacks was being told it had one.

**`active_requests` is not turns.** The health server's own comment says agent
turns; the increments are around `provider.Chat`, including background
summarization. It is the right signal for "is a restart safe" and the wrong one
for anything a person reads. Status counts turns separately and leaves that
field to the launcher.

Then the device found two more. Uptime rendered as `27.707765309s` because Go's
`Duration.String()` was piped through four boundaries as opaque text and handed
to a `Text` widget — nothing on that path had ever parsed it, so no locale could
render it properly. The fix is a number with a unit,
`detail.system.uptime_seconds`, leaving the legacy `/health` string untouched
for the launcher. And a 100px spacer that had reserved room for a navigation bar
that never overlapped the content sat invisible for as long as the page ended
with the access hint; moving that hint into the QR card left the reservation
behind as an empty band.

The security shape is worth recording. The detailed payload is richer process
information than `/health` has ever exposed anonymously, so it is authenticated
with the gateway bearer token that already guards `/reload` — no second
credential — and it fails closed rather than downgrading to a basic response.
The token stays in Kotlin. Anonymous `/health` is byte-identical to what it
always returned, which is what made the change safe to ship at all.

One thing that did not change: the numbers are in memory and reset with the
gateway, and the screen says "Since Gateway start". A restart showing
`Completed: 0` is correct, and adding a database to avoid that would have been
the largest thing in the milestone by far.

## 2026-09-07 — The command still exists; it just points somewhere else

`feature/telegram-interactive-menus` merged to `develop` with `--no-ff` at head
`39450df`, physically accepted as vc50. The milestone that shipped is not the
one the branch was opened for, which is the whole story: the interactive `/model`
picker was built, verified on the device as vc47 and vc48, removed on product
grounds, and then `/model` came back as four dozen bytes of constant.

    🤖 Model selection is managed from PocketClaw Settings.

Deleting `/model` outright was the obvious move after the picker was rejected,
and it was the wrong one. `/model` is the command people type when they want to
change the model; leaving it unregistered sent that question to the LLM, which
would answer it with a guess. A decision that model selection is Dashboard-owned
is only useful if it is discoverable from the place the question gets asked.

What makes the replacement safe is that it is informational by construction
rather than by care. The handler signature is `func(_ context.Context, req
Request, _ *Runtime) error` — it discards the `Runtime`, so `SwitchModel` and
`GetModelInfo` are unreachable from it, not merely unused. The reply is a
constant, so it cannot leak a model, provider or endpoint no matter what the
config holds. And a handled command returns from `handleCommand` before
`runAgentLoop`, so there is no LLM call and no history entry. Arguments are
ignored: `/model gpt-4o` gets the same sentence, because a command that reads its
arguments is one refactor away from acting on them.

The tests are written the same way. Rather than asserting on the reply — which a
command that switched the model and then printed this sentence would also pass —
the runtime handed in fails the test if `SwitchModel` or `GetModelInfo` is ever
called, and a structural test fails if `Request` grows a menu, button or callback
field. At the agent level the no-call and no-history assertions are followed by
an ordinary message as a control that must do both, so they cannot pass by the
harness recording nothing. Each guard was checked by removing what it protects.

One process note worth keeping. After the picker removal I reported that no Core
rebuild was needed, on the strength of a freshness-guard run that had happened
*before* the `/switch` wording change and was never re-run. The guard was right
and the report was wrong: `cmd_switch.go` is fingerprint input 246 of 536, the
staged Core still carried the old "Switch model or channel" string, and
substituting only that one file's blob hash reproduces the old fingerprint
exactly. A green result is evidence about the tree it ran against, and nothing
else.

## 2026-09-06 — The picker worked; the product it implied did not

`feature/telegram-interactive-menus` is abandoned by product decision and
converged back to `develop`. Not merged, not reverted-after-merge — nothing of
it ever entered `develop`, so the branch was simply brought back to `develop`
plus one test fix.

It is worth being precise about what failed, because it was not the code. The
`/model` picker shipped as vc47 and vc48 and was used on the device. Buttons
listed only models PocketClaw could actually switch to, a tap revalidated before
switching, callback payloads were opaque handles with a TTL that carried no
model name or credential, taps were bound to the chat and sender that opened the
picker, the answer edited the picker in place instead of stacking cards in the
conversation, cancel closed it, and opening a second picker retired the first.
Every one of those had a test that was shown to fail when its protection was
removed, and every one of them behaved on the phone.

What physical use exposed is that "the model" meant two different things. The
Dashboard configures the default that lives in `config.json`. The picker moved
the running `AgentInstance` and nothing else. So the Dashboard could show a model
the picker never offered, and a model picked in Telegram never became the
default — two places to set one thing, disagreeing, with no indication to the
user which one had won. The only repairs are synchronizing runtime state back
into configuration or making the Dashboard follow the runtime, and neither is
worth building for a convenience this size. Model management stays
Dashboard-owned; Telegram stays a conversation and control surface.

Removal was total on purpose. `bus.InteractiveMenu` and the `OutboundMessage.Menu`
field, the menu-action delegate, the Telegram callback registry with its TTL
handles and picker retirement, `commands.Menu` and `ReplyMenu`,
`Definition.Instant`, `Runtime.GetModelPicker`, `BaseChannel.Bus()` — all gone.
None of it had a user outside the picker, and generic interaction plumbing left
in place "for later" is just code nobody is testing against a real requirement.

`pkg/modelaccess` went too, which deserves an explanation since it was the one
piece that looked reusable. It was a verbatim lift of `hasModelConfiguration`
and `requiresRuntimeProbe` out of `web/backend/api` so `pkg/commands` could
import them. With the picker gone its only consumer is the package it came from,
and the lift had left `hasLocalAPIBase` implemented in both places — the exact
drift the shared package was justified as preventing. A shared package with one
consumer that also duplicates a rule is not an improvement, so the Dashboard's
eligibility logic went back where it is owned.

Two things were kept. `harnessWorkspace` replaces `t.TempDir()` in the Telegram
cancellation harness: a turn released during teardown writes its session while
`t.TempDir()` is removing the directory, so a passing suite went red on a
filesystem detail. That race is latent on `develop` and the fix is test-only.
And `/switch` is now described as "Advanced runtime controls" rather than
"Switch model or channel", with its no-argument guidance pointing at the
Dashboard for choosing a default instead of promising a picker that is not
coming. The advanced form still works and still moves the running agent; it just
no longer advertises chat as the place to choose a model.

## 2026-09-06 — Prose instead of grammar, and a struct that was never meant to be read

Branch `feature/telegram-command-ux` closed at `896021f` and merged to `develop`
with `--no-ff`. Physically accepted as vc46.

Phase A of two, and the split is the point. This one was text and privacy; the
interaction model is Phase B and was deliberately left alone.

The defect worth writing down is `/subagents`. It printed the agent's active-turn
struct with `%+v`, and that struct carries `UserMessage` — the user's own prompt
— along with their session key and their chat id. So a command that looks like a
debugging convenience was echoing private routing state back into Telegram, with
a year 1 timestamp attached for any turn that had not started. Fixing the format
string would have left the next person one `%+v` away from the same thing, so
the fix moved to the type boundary instead: the runtime hands `pkg/commands` a
`SubagentInfo` carrying a status, a duration and a nesting depth, and the agent
maps the fields across one at a time. The sensitive fields are not unprinted;
they are unreachable. Turns are numbered rather than named for the same reason —
the only human label a turn carries is the prompt.

The `Usage:` responses had a single source: the executor's empty-sub-command
branch. That made the fix small and the right shape — a `Definition` field the
executor consumes, so a new command writes a sentence rather than another
special case. Wrong arguments still get the exact usage string, because there
the grammar genuinely is the answer.

`/help` was every command rendered as its usage expression, which is where all
the angle brackets and pipes live. It is an overview now. Command descriptions
had been written for a developer reading source; they are also what Telegram's
native "/" menu shows, so rewording them fixed both surfaces through the
mechanism that already existed.

Two existing tests asserted the old behaviour — that `/help` contains
`/show [model|channel|agents|mcp <server>]`, and that a bare `/btw` answers
`Usage: /btw <question>`. They had pinned the thing being removed. A third
correction came from a test I wrote: the first `/switch` guidance suggested
`/list models` for browsing, and `/list models` enumerates nothing — it reports
the current model and says to edit config.json. Sending someone there would have
been a dead end dressed as help.

What Phase A did not do is the part to be clear about. A bare `/switch` answers
with prose now instead of grammar, but the user still reads, then types, then
has to know a model name. That is a different problem and it needs an
interaction subsystem PocketClaw does not have — no channel here handles any
interactive component, and `bus.OutboundMessage` carries only text. Phase B
starts there, with `/model` as the only proving ground, against an explicit
criterion: change model without typing anything and without seeing an internal
identifier.

## 2026-09-06 — Three switches that could never work, and a skill that said otherwise

Branch `feature/android-hardware-tool-cleanup` closed at `14a88ba` and merged to
`develop` with `--no-ff`. Physically accepted as vc45: the user opened the
Android Tool Library and the hardware tools and their category were gone.

The Tool Library offered `i2c`, `spi` and `serial` on a phone. The investigation
that preceded the fix is the part worth keeping, because it changed what the fix
should be. All three are compiled into the Android Core — `GOOS=android`
satisfies the `linux` build constraint, so `i2c_linux.go` and friends come along
— but none can work: `adb shell`, which is more privileged than the app UID,
finds no `/dev/i2c-*`, no `/dev/spidev*` and no `/dev/tty*`, and the app declares
no USB host support, so serial over USB-OTG would need an Android-native
implementation rather than a termios wrapper. They also cost nothing worth
reclaiming: one already-vendored dependency, no permission, no startup or memory.

So the answer was not deletion. It was a product boundary, in three places.

The Tool Library drops the hardware category on Android. The managed Gateway is
launched with the three tools forced off in its environment, which is what stops
a config imported from a Linux machine or hand-edited from re-enabling them —
registration is already gated on `IsToolEnabled`, so a config that resolves as
disabled is what keeps them out of `ToolRegistry` and out of the definitions the
model is sent.

The third place mattered most and was nearly missed. `workspace/skills/hardware/
SKILL.md` is embedded in the Core binary and reached the system prompt on every
turn, telling the model it could drive I2C and SPI peripherals on Sipeed boards.
Hiding three Tool Library rows while continuing to tell the model it had the
capability would have fixed the smaller half. It is filtered at discovery rather
than at onboarding, because an upgraded installation already has the file on disk
and declining to copy it for new users would have done nothing for existing ones.

The skill declares `requires: {tools: ["i2c","spi"]}`, which looks like the
principled hook — and is not. Nothing in the tree parses that metadata; the
loader reads only `name` and `description`. Teaching it to read and enforce
`requires.tools` would have changed what every skill means, to solve one product
question. The bundled skill is identified instead by the two fields already
parsed, with a guard that loads the real shipped file so a reword upstream fails
loudly rather than silently ending the filter.

One process note. The first draft of the skill tests passed with the filter
unwired — they called the policy function directly and never proved discovery
used it. Deleting the hook and re-running is what caught that. The platform is a
parameter everywhere now, so non-Android behaviour is asserted for linux, darwin,
windows and freebsd from a single host rather than for whichever one the suite
happens to run on.

## 2026-09-06 — What physical testing found after the code was "done"

Branch `feature/chat-lifecycle-durability` closed at `8024094` and merged to
`develop` with `--no-ff`. Physically accepted on SM-A165F / Android 16 across
vc42, vc43 and vc44: `/stop`, FIFO, multi-image fallback, safe error text, and
live channel reconciliation with the typing indicator toggled both ways.

Two defects survived a green suite and were found only on the device, which is
the part worth remembering.

The first was error formatting. The exhausted-chain path had been made concise;
the single-model path had not, and that is the path an ordinary default-model
setup takes. A 401 from one configured model put the provider's whole response
into Telegram under an "Original error:" heading — raw JSON, the account
message, a billing link. Three existing tests asserted that heading was present:
they encoded the leak as the contract, which is why the suite was green. Both
paths go through one rule now, and a 401 whose body is about money no longer
tells the user their API key is invalid — that would send them to replace a key
that works. A bare 500 also stopped being described as a timeout; the classifier
folds it into the timeout bucket because that is the safest transient read for a
retry decision, which is right for failover and wrong for a person.

The second was the reconcile itself, and it was a regression from this very
milestone. Disabling Telegram's Typing Indicator did nothing until the Gateway
was restarted by hand — and restarting would not have helped either. Three
independent faults, each sufficient alone. `f55668a` had rebuilt the reconcile
hash from the channel's raw settings to make it independent of decode state, and
in narrowing it dropped every field `config.Channel` keeps *beside* `Settings`:
`typing`, `placeholder`, `allow_from`, `reasoning_channel_id`, `group_trigger`.
All five were visible before that commit and none after. The milestone's own
reconcile tests never caught it because they only ever varied settings JSON and
credentials — never a common field. Underneath that, `Channel.Typing.Enabled`
had exactly one reader in the entire tree: IRC. Telegram defers its activity
signals to the agent and so never runs the `BaseChannel` code that would have
been the natural place to check, and `Manager.StartTyping` started the indicator
unconditionally. And underneath *that*, `gateway.hot_reload` defaults to off, so
the config watcher was never armed and `Manager.Reload` never ran at all.

The hot-reload default is a Core decision that should stay as it is — a server
deployment should not reload itself. PocketClaw Android wants the opposite, so
the managed Gateway is handed `PICOCLAW_GATEWAY_HOT_RELOAD=true` in the
environment the launcher already builds for it. Guards pin both halves: the
Android env entry, and the Core default staying `false`.

One thing is deliberately still broken. The Dashboard shows "Gateway restart
required" after a successful hot reload, because `bootConfigSignature` is set
only when the launcher starts or attaches to a gateway and nothing refreshes it
when the gateway reloads itself in-process. It is cosmetic, it was reported
rather than fixed, and fixing it needs a reload-completion signal between two
processes that do not have one.

## 2026-09-05 — Cancellation that waited its turn, and a fallback that answered for the primary

Branch `feature/chat-lifecycle-durability`, from `develop` at `2312f35`. Three
commits at the time of writing; see the 2026-09-06 entry for what physical
testing then found and how the milestone closed.

`/stop` did nothing while a turn was running, which is the only time anyone
sends it. Telegram is the one channel using the independent response lifecycle,
so a message arriving while the session is busy is retained in the session
mailbox and the dispatcher moves on without reading it. The stop handler is
reached only on the steering path, which Telegram never takes. The command was
therefore dequeued after the turn it was meant to cancel had finished on its
own — indistinguishable, from the chat, from nothing happening at all. Control
traffic now refuses to queue. The cleanup that follows had a second bug behind
the first: the typing indicator and the "Thinking…" placeholder are recorded per
inbound lifecycle, so neither was reachable through the /stop message's own
context. The acknowledgement is published on the cancelled turn's lifecycle
instead, which stops its indicator and turns its stale placeholder into the
reply.

Channel reconcile had the shape of a working feature and two defects underneath.
The reconcile hash was not a pure function of the configuration: `config.Channel`
serializes from its raw settings until something decodes it and from the decoded
struct afterwards, the startup hash is taken after `initChannels` has decoded
every channel, and every reload hash is taken on a config freshly loaded from
disk. The two shapes never matched, so the first save after startup stopped and
restarted every enabled channel — a Telegram edit dropping the live Discord
connection is the opposite of the isolation this is supposed to provide. And
because a decoded channel's credentials marshal to a placeholder, they were
re-introduced by a hand-maintained switch over channel names; weixin, vk and
pico_client were never in it, so their token changes were invisible and the
channel was never restarted. The hash is now built from the raw settings plus
the enabled flag and type, with credentials contributed as SHA-256 digests
recovered by walking the decoded struct — no per-channel list to fall out of
date, and no plaintext credential in the reconcile state.

The seven-image report was not a provider problem. Seven images succeeded
against the primary model alone and were rejected with HTTP 400 as soon as
fallbacks were configured, which is only possible if the request reaching the
first candidate changed. Per-candidate providers were registered under the
runtime `provider/model` key and only for the fallbacks. Two model_list entries
can name the same protocol and model id — a second API key for one model is the
ordinary reason — so a fallback sharing the primary's pair took ownership of it,
and the primary's request went to the fallback's endpoint with the fallback's
credentials and extra body. The same collision made two fallbacks for one model
collapse onto a single provider, which is why the two DeepSeek entries in the
report failed identically. Registration is keyed by the entry's stable identity
now, the primary is registered alongside the fallbacks, and the runtime pair
survives as a secondary key so a bare `provider/model` reference still resolves.

A regression test states the invariant directly: the same seven-image turn, run
once with no fallbacks and once with two, must produce a byte-identical
normalized request for the first candidate. It fails on the old keying.

Exhausted-chain errors stopped being a wall of JSON. The user gets one line per
candidate naming the model and the classified reason with its status code; the
raw provider bodies stay in the developer log under the existing redaction
rules.

## 2026-09-05 — Thirty providers nobody configured, and the filter that was missing

Branch `feature/configured-model-discovery`, from `develop` at `2b37af2`, merged
with `--no-ff`. Four commits, physically accepted as vc41.

The Fallback Models picker offered Azure, Cerebras, Anthropic, Groq, Ollama and
Volcengine to a user who had configured OpenCode and Gemini. The obvious reading
was that the picker was showing the built-in provider catalog. It was not.
`config.DefaultConfig()` seeds `model_list` with thirty keyless provider
templates — they ship in every fresh config — and the backend already labelled
every one of them `status: "unconfigured"`. The chat Default selector read that
flag and hid them. The Fallback picker filtered on virtual, duplicate and
self-reference, and never on whether the provider existed. One list, two
filters, and only one of them was right.

Adding a third filter would have set the same trap again, so the rule moved into
`lib/configured-model-source.ts` and every routing selector was pointed at it. A
structural test now fails the suite if a selector re-implements the rule, imports
the provider preset registry, or reads `common_models` — the guard is the part
that makes this stay fixed.

Live discovery then had somewhere safe to live. `POST /api/models/fetch` already
existed and already took a `model_index` instead of a key, resolving the stored
credential on the server after checking the provider and base match; the work was
to call it per configured provider and merge the results. Under
`Promise.allSettled`, not `Promise.all` — one provider timing out shows a retry
on that group and leaves the others' results standing, which is the difference
between a degraded picker and an empty one. Nothing falls back to the global
catalog, on failure or otherwise.

Selecting a discovered model needed one more thing. Default and fallback
references are validated against `model_list` by name, and that invariant was
not worth relaxing for a picker, so `POST /api/models/materialize` finds or
creates the entry *and* applies the role against a single in-memory config with
one `SaveConfig`. A rejected role writes nothing at all. Identity is the provider
instance plus the id — normalized provider, normalized API base, model id —
because two configured providers can both expose `deepseek-chat` and collapsing
them would route one through the other's credential.

A dedicated Vision / Image Model surface was built on this branch and then
rejected. It turned out Core had carried `agents.defaults.image_model` and
`routeMediaTurn` all along and the only gap was the API never reporting the
field, but the product answer was that two model roles is enough: a user who
needs images picks a multimodal default. The surface was reverted in `66d80ee`,
the whole tree returned byte-identical to the discovery commit, and
`libpicoclaw-web.so` came back to its exact pre-Vision byte count — the embedded
frontend confirming the removal rather than the source tree claiming it. Core's
image fields and routing were left untouched and remain config-file reachable.

One thing is deliberately not atomic and is documented as such: a discovered
*fallback* is materialized first and the ordered chain is saved by the existing
Save flow. Appending server-side would have silently clobbered unsaved
reordering. If the save fails, the model is a valid configured entry outside the
chain — a recoverable state, not a broken one.

Frontend `tsc`, `eslint` and 391 tests green. `web/backend/...`, `pkg/agent` and
`pkg/config` Go suites pass. Final artifact `0.2.0` versionCode 41, 64,211,166
bytes, SHA-256
`4fd3c201bb9a07017a954ade90f5eb471b97d281e057a58fad4b7e18cf76e7ec`.

`TestNoUserFacingWhatsAppSurface` still fails identically to the develop
baseline. Unrelated, and not fixed here.

## 2026-09-05 — APERTURE, and four defects that were never a matter of taste

Branch `feature/visual-redesign-aperture`, from `develop` at `9877365`, merged
with `--no-ff`. Two commits, physically accepted as vc38 and vc39 on
SM-A165F / Android 16.

The console did not merely resemble every other shadcn admin panel; it was one.
`index.css` defined every neutral at chroma 0 and, in dark mode, a near-white
primary, so there was no brand hue anywhere in the token set. That is the
finding the whole redesign turns on, and the fix is one line of arithmetic
repeated across a ramp: give every grey a small chroma at the brand hue, rising
as the surface darkens, the way physical dark materials behave. Nothing else in
this milestone bought as much for as little.

Three more defects sat beside it. `main.dart` drew light mode from
`PocketClawDesign` and dark mode from `AppTheme` — a Material 3 seed and a
FlexColorScheme theme, two unrelated systems — so toggling the theme did not
adjust PocketClaw, it swapped products. The web still shipped the PicoClaw
lobster and a PICOCLAW wordmark while Android shipped something else entirely.
And neither mark had a flat 16px form, so neither could actually be a favicon.

APERTURE answers all four with tokens and assets. Three hues on one spine:
Graphite 245 for neutrals, Claw 208 for interaction, Signal 195 for live
machine state and nothing else. The existing shadcn variables are remapped onto
the new layer rather than deleted, so every component inherited the identity
without being edited — which is what made a change this broad tractable at all.
One Flutter theme now emits both brightnesses, and the six user-selectable
modes survive intact: same enum, same order, same persisted index, re-based
from six products into six accents over one structure.

The mobile menu turned out to be a component bug rather than a design gap. The
header already passed `<IconMenu2 />` to `SidebarTrigger`; the trigger
hard-coded `<IconLayoutSidebar />` between its own tags, and JSX children
written between the tags win over children arriving through a spread. The icon
had been supplied correctly and silently discarded for as long as the component
existed. Rendering `children ?? default` fixes every call site at once.

The vc38 review then found the desktop brand appearing twice, and that was
structural too: the header spanned the whole viewport while the sidebar was
pushed 3.5rem down by an `!important` override to clear it, manufacturing a
brand slot directly above the sidebar's own. Moving the header into the content
column removed the duplication and the override together.

The mark's small-size problem was the one place measurement beat opinion. Users
described it as reading like a crown; rasterising at 16x and reading scanlines
back showed why — two pocket walls and two claw arms presenting as four
verticals of equal weight over walls that ended in bare stroke caps. Four evenly
spaced prongs is a fork. An inward lip at the mouth fixes it at every tested
size; deepening the pocket, the obvious first idea, makes it worse. Both the
evidence and the rejected variants are kept as an artifact rather than
described.

Chat keeps its asymmetry deliberately: a contained bubble for the user, an
editorial block on a logical-start rail for the assistant. Two bubbles facing
each other is a messaging app. Everything Aperture repeats — the active-nav
bracket, the assistant rail, the card accent, the composer's attachment edge —
is expressed with logical properties, which is why Arabic mirrors the entire
system with no locale branch anywhere.

112 raw Tailwind colour utilities across nineteen files were replaced with the
tokens they were approximating, and their `dark:` duplicates deleted — a token
already resolves per theme, so a second declaration for dark mode was a second
source of truth. A guard now fails the suite on any of that coming back.

Light mode was never physically reviewed, so it is held by arithmetic instead:
surfaces step monotonically, every text role clears the darkest surface by 0.3
lightness, the accent is genuinely darkened rather than reused from dark mode,
and the status colours stay inside a readable band on white.

Frontend `tsc`, `eslint` and 349 tests green. `flutter analyze` clean, 223
Flutter tests green. Core freshness, fingerprint, staged-core, developer-path,
runtime payload and Python payload guards pass; the Gradle arm64 release guard
verified all eleven native payloads. Final artifact `0.2.0` versionCode 39,
64,200,254 bytes, SHA-256
`5e576b541ddb8158ac4e94bdf0aeb3f770403f20721f34fc4f47337bac0ac636`.

`TestNoUserFacingWhatsAppSurface` still fails over
`test/widgets/whats_new_page_test.dart`. It was reproduced in a clean worktree
of `develop` at `9877365` before this branch existed, and that file is
byte-unchanged here. Recorded, not fixed — folding it into a visual merge would
have hidden it.

## 2026-09-05 — Settings gets What's New, and a Row that starved its own title

Branch `feature/whats-new`, from `develop` at `09790ac7`, merged with `--no-ff`.
Two commits, physically accepted as vc37 on SM-A165F / Android 16.

Settings now has a What's New entry immediately before About, opening a full
page of 0.2.0 release notes in all twelve app locales. A localized NEW badge
sits on the entry until the notes are read.

The identity question was the one worth getting right. The badge is keyed on
the versionName and nothing else — `buildNumber` is never read. The Android
versionCode is bumped for every internal physical candidate, and this milestone
alone produced two of them for the same release; keying on it would have
re-announced 0.2.0 to a user who had already read it. The mark is written once
the route is on screen, so a user who merely passes through Settings keeps their
badge and a user who reads the page and kills the app from it does not.

Release structure is typed Dart and every user-facing word resolves through the
ARB bundles, so a release note cannot ship untranslated English. Sixteen keys
across twelve locales, no placeholder drift, and a test that joins all rendered
copy and fails on `WhatsApp`, `BlueStacks`, `Auto-Start` or `versionCode` —
0.2.0 must advertise only what shipped and was accepted. The page is laid out
with `EdgeInsetsDirectional` and no Arabic branch; the RTL proof is geometric,
asserting the bullet marker sits left of its text in English and right of it in
Arabic.

Then vc36 passed the feature and exposed a layout defect worth recording. The
header was a `Row` with the title in an `Expanded`. A `Row` gives its
inflexible children their natural width first and the `Expanded` whatever is
left, so the title was last in line for space every time. With the unseen NEW
badge widening the What's New button, what was left on a 360px-wide phone was
**nothing**: the title got 0px against the 176px it needed and rendered
`Setti...`. Opening What's New retired the badge, returned the width and hid
the defect — which is exactly why it read as a badge problem. It was not an
English problem either; Arabic needed 198px and truncated even in the seen
state.

The header is a `Wrap` now. Title and actions keep their natural widths, sit at
opposite edges while they share a line, and drop to a second line when they
cannot; the actions are a nested `Wrap` so a long locale breaks between the two
buttons instead of overflowing. Nothing is measured against a language, so RTL
still mirrors on its own. The guard measures the title's rendered box against
its intrinsic width at 360x800 in English and Arabic, badge visible and badge
seen, and all four cases fail against the old `Row`.

`flutter analyze` clean, `flutter test` 203 passed. vc36 verified the feature,
vc37 the fix, both installed in place with app data preserved. Final artifact
`0.2.0` (versionCode 37), 64,543,774 bytes, SHA-256
`7e617eb7e4e6a0738bf9cc7ce3da56204953911b387fee8781aa59ad363424e1`, all eleven
arm64 payload guards passing. vc37 is also the first physically accepted build
carrying the pt-BR/zh Dashboard locale cleanup at `65dfc02`.

## 2026-09-04 — Telegram Context Memory becomes a setting, and the console speaks twelve languages

Branch `feature/telegram-context-settings`, merged to `develop` with `--no-ff`.
Fifteen commits, physically accepted as vc35 on SM-A165F / Android 16.

The bounded-context algorithm shipped earlier with its window hard-coded at 15.
That number is now a setting — 10, 15 (Recommended), 20, 25, or a Custom value
between 5 and 50 — behind an explicit Save that stays disabled until the value
is both valid and changed. It writes
`agents.defaults.telegram_recent_context_messages`, and 15 remains the default
because that is the value the algorithm was validated at.

Saving applies to the next turn. No app restart, no Gateway restart. The bug
that had to be found first is worth restating: `SaveConfig` writes through
`WriteFileAtomic`, so a save renames a temporary file over the target rather
than editing it. Save 17, then save 10, and the two payloads are the same length
and can land inside one coarse timestamp tick — a cache keyed on size and mtime
answers 17 for the rest of the process. The key is `os.SameFile` plus size and
mtime, and reverting it makes `TestSameSizeRapidReplacementIsNotMissed` fail
with exactly that symptom. On the device, Custom = 17 held at `limit=17` across
`history_total` 26, 28, 30, 32 and 34.

The embedded console was the other half. With the app in Arabic, Manage Models
opened an English left-to-right dashboard, because nine of the twelve app
locales had no resource to resolve to. All twelve resolve now — `pt` onto the
existing `pt-BR` bundle rather than a duplicate — and every one of the thirteen
non-English bundles is complete at 907 keys with no placeholder drift. Arabic is
genuinely right-to-left, from `i18n.dir()` rather than a test for Arabic. The
pre-existing `bn-IN` and `cs` resources are untouched and still offered.

Two things the structural gate could not see, both caught late:

`pt-BR` and `zh` are *app* locales, and both were 42 keys short — the entire
Telegram managed-onboarding surface, and the language menu's own labels. The
nine newer bundles being complete said nothing about them. Parity is now
measured against the set of locales the app can be set to, which is the only
set that matters to a phone.

Those same two bundles had never been held to the English-copy rule and were
shipping real prose byte-identical to English: all of Fallback Models, the
gateway-restart lifecycle, most of the provider picker. Translated now, in each
bundle's own conventions rather than a dictionary's — Brazilian rather than
European phrasing, and the gateway named 服务 in zh to match the 「重启服务」
button those sentences point at. Because Latin script cannot distinguish a
loanword from an oversight by equality alone, both locales get an exact-key
allowance list, and a second test fails any allowance whose key has since been
translated so a stale exemption cannot sit there covering a regression.

Three hand-written language dropdowns became one shared selector listing every
app locale by endonym. Endonyms are deliberately not resource keys: a
"translated" endonym would be wrong.

The last defect came from the device, not the gate. vc34 rendered Arabic
correctly and put the sidebar trigger at the top right — and then slid the
drawer in from the left, away from the finger that opened it. `ui/sidebar.tsx`
hard-coded `side = "left"`, and that one value fed both the mobile Sheet and the
desktop container, so no page was at fault and no page needed patching. The
default comes from `i18n.dir()` now, an explicit prop still wins, and a live
language switch moves the anchor with the drawer open. The sidebar's inner
border was a physical `border-r`, which lands on the outer screen edge once the
sidebar moves right; it is the logical `border-e`.

Tests: 237 frontend, up from 112 at the start of the milestone. The new ones
render real components through real i18next in five locales and assert the
English wording is *absent*, rather than checking that a key exists. Two static
guards keep literal accessibility names out of the source and every `t()`
default backed by a real English key. Each new guard was confirmed to fail
against the code it replaced. Flutter: 181 tests, `flutter analyze` clean.

vc35 — `0.2.0`, versionCode 35, built at `1505e33`, SHA-256
`fcec23a5430969fef892bb83ccc784bd35a1a33729fdd926de2869c355282077` — installed
over the existing install with data preserved, and accepted physically. The
final commit `65dfc02` is locale JSON and tests only; nothing runtime changed
after acceptance, so it was deliberately not rebuilt. That does mean the pt-BR
and zh cleanup is on `develop` but not yet on the device.

## 2026-09-01 — WhatsApp Self-Chat applies itself

Branch `feature/whatsapp-self-chat`. The device found the one gap the gate
could not: Connect, Change and Disconnect saved the number and then asked for a
manual Core restart. Everything else in the milestone passed physically.

The save now goes through `saveAndApplyGatewayConfig` — the same helper the
models pages use, which calls `POST /api/gateway/apply-config`, which waits for
the gateway to be idle before restarting it. No new lifecycle code: the busy
wait, the "unknown is not idle" rule, the two-minute ceiling that ends the wait
without ever interrupting a turn, and the coalescing of concurrent saves are all
the machinery that was already there.

What is new is that the card reports the outcome instead of a toast. The shared
helper takes an optional `onOutcome`; when a caller passes it, the helper skips
its own toasts — including the one that ends "Use Restart Gateway when you are
ready", which is exactly what this surface must never say. Callers that do not
pass it are untouched.

Four states, four truthful answers. A gateway that is stopped, still starting,
or already running the saved configuration needs no restart and is not touched.
A busy one is left alone: "Saved and in use. PocketClaw is waiting for the
gateway to finish what it is doing." A failed apply says PocketClaw could not
confirm the gateway picked it up. None of them claims the change is live when it
is not, and none of them asks the user to restart anything.

The number itself was never the thing needing a restart: `ConfiguredNumber`
reads `config.json` on every tool call, so Connect, Change and Disconnect reach
the agent tool immediately. `TestWhatsAppSelfChatFollowsTheConfigFileWithout
ARestart` drives the real provider against a file changing underneath it,
through Connect, Change and Disconnect, and asserts the tool reports "not
configured" after the last one. What the restart is for is the rest of the
gateway, and the console's restart-required indicator.

Tests: seven scenarios through the real apply machinery at the page level —
Connect, Change and Disconnect while running; a stopped Core; a Core still
starting; a busy Core deferring; and a failed apply — plus the `onOutcome`
delegation arm in `restart-required.test.ts` and a table for
`gatewayRestartRequiredBySignature`, which had none.


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

### Two bugs the gate could not have caught

`FileObserver(File, int)` is API 29. PocketClaw ships minSdk 24, so on anything
below Android 10 the watcher would have thrown `NoSuchMethodError` in
`PicoClawService.onCreate` and taken the foreground service — and with it
Core, Auto-Start and the gateway — down before it started. The path
constructor is deprecated but universal, and is what ships.

A background activity start does not throw on Android 10+; `startActivity`
returns normally and nothing appears. The launcher would therefore have
reported OPENED for a window nobody saw. It now asks
`ActivityManager.getMyMemoryState` first: a foreground service reports
importance 125, a visible activity 100, and only the latter earns the start.
An agent call arriving while PocketClaw is backgrounded is answered with that
reason instead.

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
