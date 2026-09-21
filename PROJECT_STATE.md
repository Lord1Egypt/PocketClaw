# PocketClaw Project State

## Authoritative current snapshot — H5C

This section is the current project-state authority. It is verified against Git,
tracked release inputs, the production signer enrollment, and current GitHub
release metadata. Detailed milestone narratives below are immutable historical
evidence and describe the state at the date of each entry.

| Field | Current fact |
| --- | --- |
| Working branch | `feature/final-release-hardening` |
| PC-1 state-basis HEAD | `0be6afbd92209953d918d5c0516662bc54f0d081` (PC-1's verified starting commit; the documentation-only closeout commit necessarily follows it) |
| H3A state-basis HEAD | `9a5a5dd9fd4a0697451d27948efe2c5be6e5c028` (verified H3A starting commit; the H3A closeout commit follows it) |
| H3B state-basis HEAD | `6491ccc6f611c7513506d622dbb1ad4a75c93a43` (verified H3B starting commit; the H3B defect fix and closeout commit follow it) |
| H4A state-basis HEAD | `30ec1951cb91df2d3ab80ce09d3e1176611242f7` (verified H4A starting commit; the H4A closeout commit follows it) |
| H4B state-basis HEAD | `a6034c065becccc0a01ed7e734dad6b2558a0ef1` (verified H4B starting commit; the H4B closeout commit follows it) |
| H5A state-basis HEAD | `9aa7066cb30ca00800291b980de34e0ebccdce89` (verified H5A starting commit; the audit-only closeout commit follows it) |
| H5B state-basis HEAD | `6c24f9ac67989a8bb2e08344ef9dcd113cdf7f18` (verified H5B starting commit; the source commit and the Core-staging closeout commit follow it) |
| H5B canonical Core build-input commit | `aa24d9e906a28f71eb8231c8bf0f236cb1f96410` |
| H5C state-basis HEAD | `33f0db672eab86985986a76598b66f968e2f44a8` (verified H5C starting commit; the closeout commit follows it) |
| PC-DEF-019 state-basis HEAD | `ea43369289c8b6c618faa08f7b91355882fc050c` (verified UI-1 closeout; the Core staging/closeout commit follows it) |
| PC-DEF-019 canonical Core build-input commit | `ea43369289c8b6c618faa08f7b91355882fc050c` |
| Staged Core pair | `libpocketclaw.so` `0a28bd5e…` 37,658,976 bytes; `libpocketclaw-web.so` `9ae1d2d9…` 25,319,424 bytes; BuildTime `2026-09-13T00:24:56+0000` |
| PC-DEF-023 canonical Core build-input commit | `54ff2525fa555744d017aae56c9a26e2049812e1` |
| PC-DEF-020 state-basis HEAD | `76064c91033860653de1e11a08e29d7245061ec1` (verified exposure-audit closeout; source and Core-staging commits follow it) |
| PC-DEF-020 canonical Core build-input commit | `f8bc52a0757f7b0a9f6c0704d2a3586db929e33f` |
| Version | `0.2.0+62` |
| Accepted physical baseline | vc62 / `lastAcceptedVersionCode=62` |
| Current phase | Final Production Release Hardening; H5C production-signed native/ELF validation closed |
| Developer production signer | `176dca6b198b9552fb4d9ad3ca18da8d6f23c0a3f5ed4bd6b75a0700f9f0efcf` |
| Core fingerprint | `9c22cb22f792e568c3a55c0f25c618c9ecc7a1d7ce0576182d070b3e5c1899ab`; the staged Core pair carries it — last moved by the Credentials withdrawal, a Dashboard change and therefore a Core build input |
| Distribution targets | Direct APK, Google Play, Official F-Droid |
| Exposure-audit state-basis HEAD | `25753cef5fa4d956e11d37b5a6176cdef977f015` (verified PC-DEF-019 closeout; the audit closeout commit follows it) |
| Final release exposure audit | **CLOSED / PASS** on the re-run at `6031898`. No release blocker remains for the GitHub / direct APK release. `PC-DEF-022`, `PC-DEF-023` and `PC-DEF-024` have all since been RESOLVED; only `PC-DEF-006` (F-Droid path) and `PC-DEF-012` remain open |
| Production candidate | **BUILT AND GATED.** `4d4bc33a…`, 63,472,307 bytes, one v2 signer `176dca6b…`; production artifact gate 57 PASS / 0 FAIL / 0 SKIPPED. Private validation evidence — not installed, published or accepted |
| Next authorized milestone | **Samsung physical pass over the remaining source-verified items.** Source-verified and pending device confirmation: PC-DEF-050 (rotate a key and watch the next request), PC-DEF-052 (no hosting URL is ever visible in Telegram onboarding), PC-DEF-055 (Set Default A→B survives restart), PC-DEF-062 (desktop Disconnect/Replace, after Telegram's creation cooldown), PC-DEF-063 (About shows `PocketClaw 0.2.0` / a resolved Core version), PC-DEF-058 (notification dialog, fresh install), PC-DEF-066 (`/help` has no lobster or Pico identity), PC-DEF-070 (the notification never says Running over a stopped runtime — scenarios A, D, E, F), PC-DEF-071 (Channels → Telegram with the gateway stopped, and with a token saved but the channel disabled, must not show a spinning "Starting Telegram…"). PC-DEF-072 is source-fixed and its clean-lifecycle half is physically confirmed; its ACTION_RESTART join-timeout matrix is covered by CoreRuntimeOwnershipTest because the intent cannot be raised from adb. PC-DEF-073 has no on-demand physical reproduction — it needs the pairing service to re-issue a live bot. Newly owed: the Credentials withdrawal (no sidebar item on device and in a desktop browser, `/credentials` lands on Models, and Models still saves/rotates a key and sets a default), PC-DEF-070's reopened direction (fresh install, grant the prompt, notification appears while the runtime is up), and PC-DEF-074 (pair a fresh disposable bot, press Start once, expect exactly one `Hello! I am PocketClaw.`). Physically verified and not to be reopened without contradictory evidence: fresh-install first password (PC-DEF-065), Public Mode reconciliation (PC-DEF-040), Telegram Managed Connect (PC-DEF-060), the first `/start` answered on the first send (PC-DEF-061), `PC-E-AI-004` (PC-DEF-053) |
| Release-evidence reconciliation | [`docs/RELEASE_EVIDENCE_RECONCILIATION.md`](docs/RELEASE_EVIDENCE_RECONCILIATION.md) — the Zero-Pico allowlist `19 → 22` growth (scope widened by `d4e7107`, three migration/legal exemptions added) and the native ELF `194 → 188` (the 194 runs passed `--native-support-manifest`, adding exactly six checks) |
| **Consolidated verification APK (Credentials withdrawal + PC-DEF-050/052/055/061/062/063/066/067/068/069/070/071/072/073/074)** | `8660e72feebab36a7867c2708c02557501ddd20a67422c5576250aee3d816395`, 63,632,711 bytes, `0.2.0+62`, package `com.lord1egypt.pocketclaw`, development signer `15cf75f9…`, Dart AOT `a135bfe6…`, Core fingerprint `9c22cb22…`, Core BuildTime `2026-09-21T02:12:52+0000`. Source gate 29 rows all PASS, artifact gate 33/33, native ELF 188 PASS / 0 FAIL, Zero-Pico 22 entries all in use, Go vet/test exit 0, frontend 566 passed, Flutter 579 passed, Android unit tests 50 passed. Verified inside the packaged Dashboard: `navigation.credentials` 0 occurrences, `/credentials` present only as the `beforeLoad` redirect to `/models`. Archived read-only at `build/forensic/apk-8660e72f…/`. **This is the single build for the remaining physical pass.** LOCAL TEST / NON-RELEASABLE |
| Consolidated verification APK (PC-DEF-050/052/055/061/062/063/066/067/068/069/070/071/072/073, superseded) | `884ef88294c8832d2326873286423c6351415865570b1b032796ea4ca92ac5e5`, 63,634,691 bytes, `0.2.0+62`, package `com.lord1egypt.pocketclaw`, development signer `15cf75f9…`, Dart AOT `a135bfe6…`, Core fingerprint `762d1502…`, Core BuildTime `2026-09-20T21:00:32+0000`. Source gate 29 rows all PASS, artifact gate 33/33, native ELF 188 PASS / 0 FAIL, Zero-Pico 22 entries all in use, Go vet/test exit 0, frontend 556 passed, Flutter 579 passed, Android unit tests 46 passed. **Installed and physically exercised on the Samsung device:** one Core web process, one gateway, one listener on 18800, clean teardown, and three rapid kill/relaunch cycles with no accumulation. Archived read-only at `build/forensic/apk-884ef882…/`. **This is the single build for the remaining physical pass.** LOCAL TEST / NON-RELEASABLE |
| Consolidated verification APK (PC-DEF-050/052/055/061/062/063/066/067/068/069/070/071, superseded) | `ec0a9fadb3bc3439762a191592e4e41185d4c22498c4420c5b8224bb52bf901d`, 63,632,103 bytes, `0.2.0+62`, package `com.lord1egypt.pocketclaw`, development signer `15cf75f9…`, Dart AOT `a135bfe6…`, Core fingerprint `d6a606ad…`, Core BuildTime `2026-09-20T20:20:12+0000`. Source gate 29 rows all PASS, artifact gate 33/33, native ELF 188 PASS / 0 FAIL, Zero-Pico 22 entries all in use, Go build/vet/test all exit 0, frontend 556 passed, Flutter 579 passed, Android unit tests 36 passed. Archived read-only at `build/forensic/apk-ec0a9fad…/`. **This is the single build for the remaining physical pass.** LOCAL TEST / NON-RELEASABLE |
| Consolidated verification APK (corrected) (PC-DEF-050/052/055/061/062/063/066/067/068/069, superseded) | `de325ce64d1270c58c6bca14999b6b93d96db03538b4115668654d14a945c45e`, 63,631,635 bytes, `0.2.0+62`, package `com.lord1egypt.pocketclaw`, development signer `15cf75f9…`, Dart AOT `a135bfe6…`, Core fingerprint `f2fd1ee7…`, Core BuildTime `2026-09-17T21:02:29+0000`. Source gate all rows PASS, artifact gate 33/33, native ELF 188 PASS / 0 FAIL, Zero-Pico 22 entries all in use, frontend 540 passed, Flutter 579 passed. Archived read-only at `build/forensic/apk-de325ce6…/`. Carries the **no-offset** ownership probe (a negative offset would forget pending updates). **This is the single build for the remaining physical pass.** LOCAL TEST / NON-RELEASABLE |
| Consolidated verification APK (PC-DEF-050/052/055/061/062/063/066/067/068/069, superseded) | `250c1c47b3cea18ca54df26e1c355f65f9011619cac3da8dcd243fe76068d437`, 63,633,491 bytes, `0.2.0+62`, Core fingerprint `aaaccb58…`, Core BuildTime `2026-09-17T20:36:35+0000`. **Superseded before physical testing: its ownership probe used `offset=-1`.** Archived read-only at `build/forensic/apk-250c1c47…/`. LOCAL TEST / NON-RELEASABLE |
| Verification APK (PC-DEF-067, 068, superseded) | `0821349ca4f9d0907680c04bd61108d19d62600f0d490b097cf881abf6105905`, 63,624,503 bytes, `0.2.0+62`, development signer `15cf75f9…`, Dart AOT `a135bfe6…`, Core fingerprint `ef212d21…`, Core BuildTime `2026-09-17T19:41:57+0000`. Artifact gate 33/33, native ELF 188 PASS / 0 FAIL, Zero-Pico PASS. Archived read-only at `build/forensic/apk-0821349c…/`. LOCAL TEST / NON-RELEASABLE |
| Verification APK (PC-DEF-058 third attempt, PC-DEF-066) | `8980c921e37bdf956b874ec7d247a9c6095edcab9dace4bd0b08c985efc03cca`, 63,596,703 bytes, `0.2.0+62`, development signer `15cf75f9…`, Dart AOT `08b44517ff47c5679b8ebead16e11c14471053b51ef12b333e1e931a211fba98`. Source gate **29/29** including `journey.fresh_install`, `flutter.suite 567 passed`, artifact gate 25/25, native ELF 188 PASS / 0 FAIL, Zero-Pico PASS. `POST_NOTIFICATIONS` confirmed declared and `targetSdkVersion` confirmed 36, both read from this APK. Archived read-only at `build/forensic/apk-8980c921…/`. **PC-DEF-058 needs a genuine uninstall.** LOCAL TEST / NON-RELEASABLE; the two open items are not physically verified |
| Verification APK (PC-DEF-065 first password, PC-DEF-058 second attempt, superseded) | `d653d6c4ff64b3bc1509058e14caa54077cf635f52a9c6ff8f2541a85acbc8d0`, 63,595,479 bytes, `0.2.0+62`, development signer `15cf75f9…`, Dart AOT `08b44517ff47c5679b8ebead16e11c14471053b51ef12b333e1e931a211fba98`. Source gate **29/29** including the new `journey.fresh_install` row, `flutter.suite 567 passed`, artifact gate 25/25, native ELF 188 PASS / 0 FAIL, Zero-Pico PASS. Fixes the two fresh-install blockers: the first Dashboard password (the claim reconciliation was closing the listener carrying its own response) and the notification prompt (asked only from a page a fresh install never opens). Archived read-only at `build/forensic/apk-d653d6c4…/`, whose `FORENSIC.md` leads with the ordered fresh-install journey. **Requires a genuine uninstall first.** LOCAL TEST / NON-RELEASABLE; nothing in it is physically verified |
| Verification APK (PC-DEF-061 readiness, 062, 063, 064, superseded) | `008d9c4df2436f29e63981a7b45cb98d7c17031b86c6b6576bfd78aa75f3aae8`, 63,591,331 bytes, `0.2.0+62`, development signer `15cf75f9…`, Dart AOT `08b44517ff47c5679b8ebead16e11c14471053b51ef12b333e1e931a211fba98`. Source gate 28/28 with `flutter.suite 567 passed`, artifact gate 25/25, native ELF 188 PASS / 0 FAIL, Zero-Pico PASS. Carries authoritative Telegram readiness, the desktop disconnect and replace lifecycle, the Core-version loading fix and the audited What's New. Archived read-only at `build/forensic/apk-008d9c4d…/`, whose `FORENSIC.md` carries the first-message acceptance test. **PC-DEF-040 and PC-DEF-058 both need a FRESH INSTALL.** LOCAL TEST / NON-RELEASABLE; nothing in it is physically verified |
| Verification APK (PC-DEF-061 first-message intake, superseded) | `38da4acb1d30a9e46c8226a1f1e66067b6150e59c5bd8c1959cd2e5d4ac5e585`, 63,572,435 bytes, `0.2.0+62`, development signer `15cf75f9…`, Dart AOT `af14f0ec2b1604181e4d2780248e62dcc216778205732232f4eeb0a6b7b86189`. Source gate 27/27 with `flutter.suite 563 passed`, artifact gate 25/25, native ELF 188 PASS / 0 FAIL, Zero-Pico PASS. Carries the PC-DEF-061 intake ordering and the new polling instrumentation. Archived read-only at `build/forensic/apk-38da4acb…/`, whose `FORENSIC.md` lists the exact per-defect device checks. **PC-DEF-040 and PC-DEF-058 both need a FRESH INSTALL.** LOCAL TEST / NON-RELEASABLE; nothing in it is physically verified |
| Verification APK (PC-DEF-060 desktop onboarding, superseded) | `249b765df37985067479455fa734cc0b0d96ef1247b1d1c4230306774f851847`, 63,573,451 bytes, `0.2.0+62`, development signer `15cf75f9…`, Dart AOT `af14f0ec2b1604181e4d2780248e62dcc216778205732232f4eeb0a6b7b86189`. Source gate 27/27 with `flutter.suite 563 passed`, artifact gate 25/25, native ELF 188 PASS / 0 FAIL, Zero-Pico PASS. Confirmed inside the packaged Core: the PC-DEF-060 endpoints and desktop UI, the PC-DEF-040 claim reconciler, the PC-DEF-059 redirect, and the compiled-in onboarding URL. Archived read-only at `build/forensic/apk-249b765d…/`, whose `FORENSIC.md` lists the exact per-defect device checks. **PC-DEF-040 and PC-DEF-058 both need a FRESH INSTALL.** LOCAL TEST / NON-RELEASABLE; nothing in it is physically verified |
| Verification APK (PC-DEF-040 reopened, superseded) | `ce151adbde6dc40e368908e213cd0178d04236ed7ab227be7771f051bb03ce80`, 63,554,007 bytes, development signer `15cf75f9…`, Dart AOT `af14f0ec2b1604181e4d2780248e62dcc216778205732232f4eeb0a6b7b86189`. Source gate 27/27 with `flutter.suite 563 passed`, artifact gate 25/25, native ELF 188 PASS / 0 FAIL. PC-DEF-040's reconciler is confirmed inside the packaged Core. Archived read-only at `build/forensic/apk-ce151adb…/`. **PC-DEF-040 and PC-DEF-058 both need a FRESH INSTALL.** LOCAL TEST / NON-RELEASABLE; nothing in it is physically verified |
| Verification APK (PC-DEF-059 second attempt, PC-DEF-060, superseded) | `41af6c4f96e8a1e64c62be075e0441ba072cd4b218b910c3c8e79daae749dbb1`, 63,554,799 bytes, development signer `15cf75f9…`, Dart AOT `af14f0ec2b1604181e4d2780248e62dcc216778205732232f4eeb0a6b7b86189`. Source gate 27/27 with `flutter.suite 563 passed`, artifact gate 25/25, native ELF 188 PASS / 0 FAIL. The PC-DEF-059 fix is confirmed inside the packaged Core, not just the source tree. Archived read-only at `build/forensic/apk-41af6c4f…/`, whose `FORENSIC.md` lists the exact per-defect device checks. **PC-DEF-058 still needs a FRESH INSTALL.** LOCAL TEST / NON-RELEASABLE; nothing in it is physically verified |
| Verification APK (PC-DEF-057/058/059, superseded) | `6bb32b387cc82c436247fde50c04216ae2a434a95015ad0f3a147b6d5935ba18`, 63,552,763 bytes, development signer `15cf75f9…`, Dart AOT `af14f0ec2b1604181e4d2780248e62dcc216778205732232f4eeb0a6b7b86189`. Source gate 26/26 with `flutter.suite 563 passed`, artifact gate 25/25, native ELF 188 PASS / 0 FAIL. Archived read-only at `build/forensic/apk-6bb32b38…/`, whose `FORENSIC.md` lists the exact per-defect device checks. **PC-DEF-058 needs a FRESH INSTALL.** LOCAL TEST / NON-RELEASABLE; nothing in it is physically verified |
| Verification APK (PC-DEF-056/057, superseded) | `7c8eefd399318b6187b0fb87c8bd687d7d08c3537959e31f38767a642908e3cb`, 63,539,059 bytes, development signer `15cf75f9…`, Dart AOT `007c23b6be22a9a44f429a53f8fb9eaf930180e67bb20f111cadceedb68e22f4`. Source gate 26/26 with `flutter.suite 561 passed`, artifact gate 24/24, native ELF 188 PASS / 0 FAIL. Archived read-only at `build/forensic/apk-7c8eefd3…/`. LOCAL TEST / NON-RELEASABLE. **Nothing in it is physically verified** |
| Verification APK (PC-DEF-049..055, superseded) | `6df7abaa6bec5a5124d21d30b582fc37d2837be036fa75e93eb3d30d5634894c`, 63,528,459 bytes, development signer `15cf75f9945d5354e75707e0326b7cffc60ac51a68df38156db318ef4578a27c`, Dart AOT `d5d52742ab6cc5672e7c3910dc20c50e5c4da430c80d65c49501d12ea17a7968`. Source gate 26/26, artifact gate 24/24, native ELF audit 188 PASS / 0 FAIL. Archived read-only at `build/forensic/apk-6df7abaa…/` with its private R8 material separated. LOCAL TEST / NON-RELEASABLE |
| Staged Core freshness | **CURRENT.** Rebuilt from the source commit `3ba9ffa` and staged in `8e59a21`, which touches no build input. Fingerprint `9c22cb22f792e568c3a55c0f25c618c9ecc7a1d7ce0576182d070b3e5c1899ab` (was `762d1502…`), BuildTime `2026-09-21T02:12:52+0000`. `core.staged_freshness` and `native.elf_audit_contract` both pass |
| Flutter suite | Green — 579 passed, 0 failed — and the **complete** suite is now a release gate (`flutter.suite`) |
| Public release asset policy | APK only. An AAB is a Play-upload artifact and is never a public release asset — `PC-DEF-021` |

## 2026-09-21 — Credentials withdrawn from the v0.2.0 surface

**A scoped release decision, not a defect, so no defect ID is allocated.**

Account-login credential management is not finished for v0.2.0. OpenAI browser
OAuth reaches a real authentication screen and returns `unknown_error`, and the
Claude, ChatGPT/Codex and Google account-login flows are not implemented. A
stable release must not show a user a door that does not open, so the way in is
withdrawn for this release — not decorated with a disabled control or a "coming
soon" label, both of which are still a door.

**What was withdrawn.** The sidebar's Credentials entry, the `/credentials`
post-auth `?next=` destination, and the ability to reach the page by normal
navigation. `/credentials` is redirected to `/models` in `beforeLoad`, so a
bookmark or a stale embedded bundle cannot mount unfinished auth controls on the
way past and lands where provider access is actually configured.

**What was deliberately kept.** The page, its hook, the OAuth API, the provider
credential models and all `credentials.*` i18n strings across 14 locales. This
is a product-surface change, not a deletion: the later feature phase — Google
account login, Claude subscription login, ChatGPT/Codex subscription login and
the finished credential-management UI — starts from this code.

**Unaffected, and asserted.** API-key provider configuration keeps its own
finished path through Models: add and manage a provider, key rotation, Set
Default, and the runtime apply. None of it routes through the withdrawn page,
and `manage-provider-sheet`, `delete-provider-dialog`, `fallback-models-section`
and the models routing tests all still pass.

**Navigation is data now.** `components/app-navigation.ts` builds the sidebar's
groups as a pure function, so the *absence* of Credentials is assertable — a
snapshot of a rendered sidebar records whatever is there, which is exactly the
wrong instrument for proving something is gone. Mobile and desktop render the
one builder, so parity is structural rather than two lists agreeing.

## 2026-09-20 — One Core owner, and never probing our own Telegram generation

**PC-DEF-072 — a slow service thread was orphaned and could double-start Core.**
Two authorities were wrong in the same direction. `stopService()` joined the
worker for five seconds and then cleared `serviceThread` whether or not it had
exited, so a bounded wait that expired was recorded as proof the thread was gone;
`startService()`'s duplicate guard reads that field and started a second worker
over a live one. It also cleared `stopped`, a single flag shared by every worker
that had ever run, which un-stopped the abandoned one — so it carried on into
`runWebService()` and spawned a second Core reaching for port 18800. The join
expired in the first place because stop could only reach the web process: a
worker inside `ensureOnboarded` or the crash backoff had no reachable child and
no interrupt. `CoreRuntimeOwnership` now holds one invariant — at most one worker
may hold the current epoch, and only its holder may start or register a Core. The
stopper never releases ownership; the worker releases from its own `finally`; a
start arriving in between is queued and runs on release, so a restart is neither
silently ignored nor forced over a live runtime.

**PC-DEF-073 — the ownership probe could collide with our own poller.** Candidate
validation ends in a real `getUpdates`, and Telegram allows one long-polling
consumer per bot, so a candidate that is a bot this install already polls makes
PocketClaw compete with itself; the runtime treats the resulting 409 as terminal
with no retry and retires a healthy generation. Reachability is not decidable
here — the token comes from an external pairing service — and an unprovable input
is not an unreachable state, so it is guarded rather than assumed away.
Validation is skipped only when the candidate is byte-identical to the committed
token, the gateway reports Running with a non-zero generation and no runtime
failure, and no apply is pending. In that state the live poller has already
proved everything the validation would ask, including the absence of a webhook.

**Telegram "configured" authority** is written up as a design recommendation in
`docs/RELEASE_EVIDENCE_RECONCILIATION.md` §4 and deliberately not changed: it is
a payload contract two surfaces and the Flutter client read, neither fix above
depends on it, and it belongs in its own pass.

## 2026-09-20 — Runtime-state truthfulness: the notification, and the Telegram card

**PC-DEF-070 — "PocketClaw Running" over a stopped runtime.** The persistent
notification was a log of the last thing the service thread did, not a rendering
of what was true. `"Running (PID: n)"` was written once, when the Core process
was spawned, and nothing ever revised it against that process still being alive;
only `ACTION_STOP` removed the notification, so every other teardown relied on
the platform cancelling it, and on the owner's device it did not. The
authoritative subject is now stated: the foreground service and the Core runtime
process it owns — the same subject as the notification's Stop action, and
deliberately not the Gateway, whose auto-start the owner may legitimately turn
off. `RuntimeNotificationPolicy` derives RUNNING from a live process read under
`serviceLock` and cannot reach it from intent; `onDestroy` removes the
notification in every path; and `PocketClawApp.onCreate` — which Android runs
before any component of a new process — cancels a notification that survived the
previous one rather than rewriting it into a new claim.

**PC-DEF-071 — terminal Telegram states shown as "Starting Telegram…".** The
connected card's heading was an else-chain ending in `startingTitle`, so every
readiness state without a branch of its own was dressed as a stage of starting.
`not_configured`, `gateway_stopped` and `authentication_failed` are not stages of
anything, and each already had an accurate body sentence, so the card rendered
two contradictory answers at once. `authentication_failed` is terminal and stops
the poll, so its spinner never stopped. The starting stages are now named
positively and anything else renders its own sentence, once, under a warning
icon. No new user-facing string, so no locale drifted.

**PC-DEF-072 opened, not fixed.** `stopService()` clears `serviceThread` whether
or not the join succeeded, so `ACTION_RESTART` can start a second service thread
over a live one and spawn a second Core. Not observed physically; the one-line
change makes the restart silently do nothing instead, so it needs a design and
its own device test rather than a patch in a fix-only pass.

## 2026-09-17 — Telegram conflict handling, and the consolidated verification APK

**PC-DEF-069 — a bot already owned by another service.** `getMe` proves a token
is valid, not that PocketClaw can own the update stream. Two conflict classes are
now terminal and non-destructive. An active webhook is found with
`getWebhookInfo` and refused — PocketClaw never calls `deleteWebhook` or
`setWebhook`. A `getUpdates` 409 (webhook or another poller) is classified by the
status code, the exact generation is revoked and retired, and Telego's retry loop
is stopped via a cancellation-shaped error, so PocketClaw never fights the other
service. Replacement is now a pre-commit transaction: `getMe` → `getWebhookInfo`
→ a non-consuming `getUpdates` probe with **no offset** (a negative offset was
tried first and corrected before any physical test: Telegram forgets all earlier
updates on a negative-offset call, which would discard a pending first `/start`),
and a candidate owned elsewhere is rejected
before any mutation, so the previously working bot stays authoritative and
recoverable. Readiness reports `telegram_conflict` / `webhook_active` or
`bot_in_use`, never ready and never generation-authorized. 401 (`invalid_credentials`),
409 (`telegram_conflict`) and missing owner (`setup_required`) stay distinct. The
conflict logs carry the generation, subtype and transition only — never the token,
webhook URL, owner id or chat id.

**One consolidated DEV APK** now carries all current Telegram fixes for the
remaining physical pass: `de325ce64d1270c58c6bca14999b6b93d96db03538b4115668654d14a945c45e`,
63,631,635 bytes, `0.2.0+62`, development signer `15cf75f9…`, Core fingerprint
`f2fd1ee7…`, Core BuildTime `2026-09-17T21:02:29+0000`, Dart AOT `a135bfe6…`.
Archived read-only at `build/forensic/apk-de325ce6…/`. Source gate all rows PASS,
artifact **33/33**, native ELF **188 PASS / 0 FAIL**, Zero-Pico PASS, frontend
**540 passed**, Flutter **579 passed**. No device was attached, so nothing in it
is physically verified.

**Correction before physical testing — the ownership probe was destructive.**
The first version of this build (`250c1c47…`) probed another poller with
`getUpdates offset=-1`. Telegram forgets all earlier updates on a negative-offset
call, so that probe could have discarded a pending first `/start` — the exact
update PC-DEF-061 preserves. It was corrected before any physical test: the probe
now carries **no offset** (`limit 1`, `timeout 0`), and an update is confirmed
only when a later `getUpdates` carries a higher offset, so the probe returns
pending updates without confirming or dropping them. The 409 subtype is now
decided by a fresh non-destructive `getWebhookInfo` re-check, not by English
description matching. The superseded `250c1c47…` APK must not be used for the
physical pass.

## 2026-09-17 — Telegram UX: Copy link, and the owner-missing state

Two Telegram follow-ups, both fixed in source; neither is physically verified.

**PC-DEF-067 — desktop Copy link failed on a plain-HTTP origin.** The component
was the only copy surface in the Dashboard calling `navigator.clipboard.writeText`
directly, and the launcher serves plain HTTP: a LAN browser is not a secure
context, the API is undefined, and the bare `catch` turned the `TypeError` into a
failure toast. It now uses the shared `copyText` helper (Clipboard API, then
`execCommand`); when both fail the canonical Telegram link is shown in a
selectable read-only field with guidance, never a dead-end. `copyText` no longer
throws when `execCommand` is missing, and the copy path finally has tests.

**PC-DEF-068 — a valid token with no owner was a silent dead bot.** The desktop
manual save (`PATCH /api/config`) never ran the owner contract, `NewTelegramChannel`
then refused the channel, the Manager skipped it, and the bot polled nothing while
the Dashboard still said configured. Zero owners is now an explicit owner-missing
state: the channel starts and, **before the allowlist**, a private sender gets
deterministic setup guidance once, including their own numeric id; a group is left
unanswered. Nothing else runs — no agent, provider, tool, command, session or
config mutation, no auto-claim. The base allowlist carries a non-matching sentinel
so it can never read as open access. Readiness reports `setup_required` /
`owner_missing`, derived from persisted config so every writer converges on the
same contract, and the Dashboard shows "Telegram setup incomplete" with the Allowed
From field opened instead of claiming Connected. The one-owner contract, silent
non-owner rejection, Disconnect/Replace, generation ownership and 401 fail-fast
are unchanged.

Gates: Go `pkg/channels/...`, `pkg/status`, `web/backend/api` green; `gofmt` and
`go vet` clean; frontend **536 passed** with `tsc -b` and ESLint clean. APK
`0821349ca4f9d0907680c04bd61108d19d62600f0d490b097cf881abf6105905`, 63,624,503
bytes, artifact gate **33/33**, native ELF **188 PASS / 0 FAIL**, Zero-Pico PASS.
No device or browser was attached, so **nothing in this APK is physically
verified**.

## 2026-09-17 — Release-hardening sweep; Telegram creation cooldown respected

An external Telegram cooldown blocked creating another bot, so this round closed
the non-Telegram release blockers that share config/runtime/UI surfaces. No
Telegram bot was created and no Telegram API was hammered.

**PC-DEF-052 had one real hole left.** The Android native launch already resolved
the setup link in the background and refused anything that was not Telegram, but
three other paths still accepted the service's value verbatim: the Dashboard
anchor and `copyLink`, the Android QR, and the Go handler that forwarded
`deep_link`/`qr_payload` — despite comments claiming Core validated them. Core now
drops any non-Telegram destination (`telegramDestinationOrEmpty`, mirroring
`telegram_link.dart`); the desktop component refuses to render one anyway; and the
QR is omitted when its payload is not a Telegram destination. Confirmed in the
packaged `libpocketclaw-web.so` (the warning string and the host list are present).

**PC-DEF-055 had a data-loss bug beside the UX one.** The add/edit request shapes
carry no `enabled` field, so both handlers wrote Go's zero value back: a model
added through the API started disabled and editing one disabled it. An omitted
field now means enabled on add and preserves the stored value on edit. The Set
Default handler-level contract is now tested (persistence, A→B switch, enabled
field, key rotation preserving the selection). **Recorded limitation:** the
product exposes no enable/disable control, so `validateDefaultModelSelection` was
not widened to require `Enabled`; the default-selectability gate remains
`available` + `default_model_allowed` + non-virtual, and delete clears the default
to empty.

**PC-DEF-062/063/066 — tests, not code.** A direct pair-over-pair Replace test
asserts exactly one owner and the new token. The About widget test now pins the
real failed-probe representation (`coreVersion: null`) to Unavailable. The `/help`
branding test now drives the runtime-supplied `ListDefinitions()` path and asserts
the exact header. PC-DEF-066 already resolved the lobster; nothing was re-touched.

**Reconciled, not "fixed":** the Zero-Pico allowlist `19 → 22` and the native ELF
`194 → 188` are both scope changes, fully accounted for in
[`docs/RELEASE_EVIDENCE_RECONCILIATION.md`](docs/RELEASE_EVIDENCE_RECONCILIATION.md).

Gates: source **all rows PASS** (`flutter.suite 579 passed`, `native.elf_audit_contract`,
`core.staged_freshness`), artifact **33 PASS / 0 FAIL / 0 SKIPPED**, native ELF
**188 PASS / 0 FAIL / 0 SKIP**, Zero-Pico **22 entries, all in use**, frontend
**527 passed** with `tsc -b` and ESLint clean, `flutter analyze` clean. Verification
APK `adea8f173adc90268ff7119a60f00bd07e8a22dbbaa64704a9d089ac7fdbcf6f`,
63,620,051 bytes, `0.2.0+62`, development signer, Core fingerprint `5c1aa8b4…`,
Core BuildTime `2026-09-17T18:44:31+0000`, Dart AOT `a135bfe6…`, archived read-only
at `build/forensic/apk-adea8f17…/`. **LOCAL TEST / NON-RELEASABLE.** No device was
attached, so nothing in this APK is physically verified.

## 2026-09-15 — The journey passes; two items left

**Physically verified on the Samsung this round**, and none of it is to be reopened
without contradictory evidence: the fresh-install **first Dashboard password** succeeded
on the first attempt, **Public Mode reconciled** to LAN without a toggle, Telegram
**Managed Connect** worked from the desktop, **`PC-E-AI-004`** behaved, and the
**first `/start` was answered on the first send** — `polling.started` and
`polling.ready` at 06:18:13, the delivered update at 06:18:33 with `first_update=true`
and `message_chars=6`. That closes PC-DEF-061 on the device: both the intake ordering
and the readiness gate that made "Connected" mean receiving.

**PC-DEF-058 — third attempt, and this time the audit found a wrong input rather than a
wrong placement.** Two candidates were ruled out *from the built APK* rather than from
source: `targetSdkVersion` is **36**, so `POST_NOTIFICATIONS` is a runtime permission and
requestable, and `aapt2 dump permissions` confirms it is declared. What was provably
wrong: the "have we asked" record lived in `shared_prefs/pocketclaw_prefs.xml`, and this
app ships `allowBackup="true"` with only three *file*-domain paths excluded. The
preference store is not one of them, **so a reinstall can restore
`notification_permission_asked = true` from a previous install** — the state machine then
resolves `DENIED`, whose action is "offer Settings, do not ask", and a genuinely fresh
install never sees the dialog. "Have we asked *this* install" is per-install state, so it
is now a marker under `noBackupFilesDir`; the legacy key is deliberately not migrated,
since reading it would carry the restored value straight back.

And because the source has now looked correct twice, the state machine is
**instrumented** with exactly the safe fields the owner specified — api level, declared,
granted, `should_show_rationale`, the marker, the resolved state, lifecycle state,
whether this resume returned from the all-files screen, attempted, and result. No chat,
account or credential value. If the dialog still does not appear, that line names the
wrong input and there is nothing left to guess.

**PC-DEF-066 — `/help` opened with a lobster.** Pre-Aperture branding, and the first
thing a new user's first command showed them. Removed rather than substituted: the
product's mark is not an emoji. `pkg/env.go`'s `Logo` constant is upstream CLI branding
and is **not referenced from `pkg/` or `web/` at all**, so it reaches no PocketClaw
surface and was left alone rather than forking the upstream baseline. Verified in the
packaged binary: the `/help` header string with the mascot is gone, and the only
remaining occurrences are that unreachable constant.

Nothing caught this — Zero-Pico is lexical, the i18n parity suites cover the Dashboard
bundles and the Flutter ARB files rather than Core's Go strings, and the menu tests assert
names and counts. `branding_test.go` now checks the assembled `/help` plus every command
and subcommand string, and asserts `/help` still names the product so it cannot pass by
deleting the header. Restoring the lobster fails it.

Gates: source **29 PASS / 0 FAIL / 0 SKIPPED**, artifact **25 PASS / 0 FAIL / 0
SKIPPED**, native ELF **188 PASS / 0 FAIL / 0 SKIP**, Zero-Pico PASS, Android unit tests
green. Verification APK `8980c921e37bdf956b874ec7d247a9c6095edcab9dace4bd0b08c985efc03cca`, 63,596,703 bytes, archived read-only at
`build/forensic/apk-8980c921e37bdf956b874ec7d247a9c6095edcab9dace4bd0b08c985efc03cca.../`. No device was attached to this session, so the two open
items are **not** physically verified.

## 2026-09-15 — Two fresh-install blockers, and the journey gate that will catch the next one

A real fresh install on the Samsung could not create its first Dashboard password, and
the notification dialog never appeared. Both are release blockers and both are fixed;
more importantly, the ordered path is now a gate row, because **every isolated test
passed while first setup was impossible.**

**PC-DEF-065 — the first Dashboard password.** The page answered *"must be authenticated
to change password"*, which is impossible for a first setup. Reproduced in a test before
anything was changed, and the path is exact:

1. `POST /api/auth/setup` takes the first-claim branch; loopback satisfies PC-DEF-039;
   **the password is written.**
2. `handleSetup` then calls PC-DEF-040's reconciliation **on the request's own
   goroutine**, after `w.Write` but before the handler returns — so the response is still
   in `net/http`'s buffer. Written is not flushed.
3. `ApplyPublicMode` closed the old listener group with `server.Close()` plus an explicit
   close of every tracked connection — **including the one carrying that request.**
4. The browser saw an aborted request; the password was already set; the retry found
   `initialized == true` and was refused by the change-password rule.

So the two flows were never conflated — the initial-claim and authenticated-change
branches are correctly distinct, and the report was the *second* attempt hitting the
second branch after the first had silently succeeded. The fix has two halves, either
alone insufficient: **a swap that only widens access now drains** (`Shutdown`, bounded)
because it revokes nothing, while narrowing still closes hard since there a remote client
is being revoked; and **the reconciliation is asynchronous**, which it must be, because
draining inline would deadlock against `Shutdown` waiting for the caller's own handler.
Security is untouched and asserted: loopback-only first claim, spoofed headers refused,
no unauthenticated password change, `/launcher-setup` not a reset path.

**PC-DEF-058 reopened — the cause was placement, not policy.** The manifest entry, the
SDK gate, the policy rules and the platform call were all correct and all **unreached**:
the only trigger was `ConfigPage.initState`, and a fresh install never opens Settings —
the shell starts on the Dashboard. The ask now happens on `MainActivity.onResume`, which
every launch takes, and deliberately waits for the return when that same resume sent the
user to the all-files-access screen.

**The process change.** `journey.fresh_install` drives the ordered path against the real
HTTP server, the real listener swap, the real middleware and the real bcrypt store —
first claim, login, anonymous change refused, remote claim refused from three addresses
with spoofed headers, ownership surviving a restart. **Proven to catch the regression:**
with the old inline close restored it fails with `the response never arrived: EOF`, which
is the user-visible failure reproduced in a test. Any change to auth, launcher setup,
first claim, Public Mode, Android permissions, the Service lifecycle or Dashboard
middleware runs it before an APK is called green.

Gates: source **29 PASS / 0 FAIL / 0 SKIPPED**, artifact **25 PASS / 0 FAIL / 0
SKIPPED**, native ELF **188 PASS / 0 FAIL / 0 SKIP**, Zero-Pico PASS, Android unit tests
green (11 in the notification policy alone). Verification APK `d653d6c4ff64b3bc1509058e14caa54077cf635f52a9c6ff8f2541a85acbc8d0`, 63,595,479 bytes, archived
read-only at `build/forensic/apk-d653d6c4ff64b3bc1509058e14caa54077cf635f52a9c6ff8f2541a85acbc8d0.../`. No device was attached to this session, so
**nothing in this APK is physically verified** — and it must not be called release-ready
until a real fresh install passes the whole journey.

## 2026-09-15 — Connected means receiving; the desktop Telegram lifecycle closes

**The PC-DEF-061 ordering fix is confirmed on the device.** The new instrumentation
shows the intended order: `polling.started` and `polling.ready` at 02:58:42, with
`Telegram bot identity resolved` only at 02:58:48 — so the getMe call is provably off
the intake path and the four-second window is gone. Telegram's command menu is also
**physically verified**: `defined=14 sent=14`, and typing `/` shows it.

A first `/start` was still unanswered. Telegram's own timestamps are minute-resolution,
so the evidence cannot say whether that send fell just before or just after
`polling.ready` — and **no claim is made about a Telegram-side drop.** What the
boundary does settle is that the product contract was unmet either way.

**PC-DEF-061, second part: readiness is authoritative now.** The Dashboard said
Connected when the gateway process had been restarted, which says nothing about whether
Telegram is receiving — the channel is built and starts polling asynchronously inside
the gateway, and the command menu is published asynchronously after that.

- `status.Channel` carries a **three-valued** `commands_registered`. Absent means the
  channel publishes no menu, which is not the same as a menu that has not landed; a gate
  conflating them would wait forever on a channel that was never going to report.
- `GET /api/telegram/readiness` maps the gateway's own snapshot to the stages the UI
  renders. The credential for the authenticated detail probe is the gateway's **own**
  bearer token — the private token file on Android, the pid record on desktop — so
  nothing new is minted.
- Completion announces nothing. The managed flow waits, names the stage it is waiting
  on, and says Connected only on `ready`; the wait is bounded at 90s and offers Check
  again rather than claiming anything. An unreadable status reads as **Telegram status
  unavailable**, which is neither connected nor starting.

**A second real bug surfaced while testing the restart case**, and it was found by a
test rather than guessed: `Stop` returned while Telego still held its long-polling lock,
so `Start` on a stopped channel failed with "long polling already running" and left
Telegram down. `Stop` now waits for the poller to unwind, bounded, and says so if it
does not.

**PC-DEF-062 — the desktop lifecycle closes.** Pairing worked from a browser but nothing
could undo it: the connected card's only actions were host calls, so the owner had to
pick the phone up. `clearTelegramCredentials` is the deliberate mirror of
`writeTelegramCredentials` and clears the token, the owner allowlist and the enabled flag
**together** — a disabled channel still holding a token and an owner reads as connected
to every surface that asks, and a retained owner would silently authorise the next bot
paired there. Going through the runtime apply is what stops the old bot polling. Replace
bot reveals the managed flow that is already verified rather than reimplementing pairing.

**PC-DEF-063 — Unknown was not a loading state.** Reading the Core version runs the Core
binary, and every layer answered the literal string `unknown` for a transient failure:
the Kotlin probe, the method channel, both Dart adapters. That string passed the cache's
non-empty test and **became** the displayed version until something re-probed. A failure
is an absence now, is never cached, and the About dialog decides loading from the future
rather than from the value — so Loading, a version, and a genuinely failed probe are
three distinct states.

**PC-DEF-064 — What's New had moved on.** It still said Telegram was set up from
Settings. Every bullet was audited against the source (the bundled-tool and Python 3.14.7
claims hold), one was reworded and seven added for physically verified capabilities. The
notification permission is deliberately **not** claimed, since PC-DEF-058 is unverified.
And the drift itself is now a gate: `tool/release_notes.py` renders
`docs/RELEASE_NOTES.md` from the app's own release structure and English strings, and
`release.notes_match_whats_new` fails the source gate on a stale file — proven to fire.

Gates: source **28 PASS / 0 FAIL / 0 SKIPPED** with `flutter.suite 567 passed`, artifact
**25 PASS / 0 FAIL / 0 SKIPPED**, native ELF **188 PASS / 0 FAIL / 0 SKIP**, Zero-Pico
PASS; frontend 523 passed across 39 files with `tsc -b` and ESLint clean; Android unit
tests green. Verification APK `008d9c4df2436f29e63981a7b45cb98d7c17031b86c6b6576bfd78aa75f3aae8`, 63,591,331 bytes, archived read-only at
`build/forensic/apk-008d9c4df2436f29e63981a7b45cb98d7c17031b86c6b6576bfd78aa75f3aae8.../`. No device was attached to this session, so **nothing in
this APK is physically verified**.

## 2026-09-15 — PC-DEF-060 verified; PC-DEF-061: the first owner message

**PC-DEF-060 is PHYSICALLY VERIFIED PASS** on the desktop Dashboard and the Samsung.
Managed pairing is offered from a browser, Telegram opens directly, no hosting origin is
ever shown, the bot is created and configured, the channel starts, the bot answers, and
command registration reached Telegram — `defined=14 sent=14`. The command menu was a
false alarm: it was there, reached by typing `/`.

**PC-DEF-061, the one real finding from that run.** The owner's *first* `/start` went
unanswered; the second was answered. The instruction was to prove where the update went,
not to add a delay, so every hop was read rather than guessed:

- **PocketClaw persists no Telegram update offset.** None exists anywhere in
  `pkg/channels/telegram`; the offset lives only inside Telego's copy of the params, and
  every `Start` passes it unset — which asks Telegram for everything it still holds. A
  replaced or reconnected bot therefore *cannot* inherit an offset and skip its own first
  updates. That is the owner's bot-identity question answered by construction, now pinned
  by a test.
- **PocketClaw makes no webhook call at all**, and never passes `drop_pending_updates`.
- **The onboarding service cannot consume the child bot's updates**: its Telegram client is
  built once, with the *manager* token. The `drop_pending_updates=true` in that repository
  is on the manager bot's own webhook. The child token is retrieved, stored and delivered,
  and no client is ever built with it.
- **The 45s HTTP timeout does not race the 30s long poll**, and Telego's poller blocks on a
  100-deep buffer rather than dropping.

**The defect this found is real, narrow, and proven by test.** Long polling is at-least-once
only while the client behaves: `getUpdates` hands over a batch and the *next* call, carrying
the advanced offset, is what makes Telegram delete it permanently — and Telego issues that
next call immediately. So the gap between the poller starting and the handler consuming is
the one place an update that already arrived can still be lost. `Start` put a **blocking
`getMe` inside that gap**: `bot.Username()` resolves lazily, and the device measured
**four seconds** (21:51:42 → 21:51:46). `SetRunning(true)` fired inside the same window, so
the channel reported **Running while nothing could receive**.

The consumer now goes live first, `Running` waits on the handler's own state rather than a
delay, and the identity call moved into its own goroutine — the bot is still named, since a
replaced managed bot has to be tellable from the one before it. `observeUpdates`, unbuffered
so it adds no second place an update can sit, logs `polling.update_delivered` with the
`update_id` and the offset it confirms, and a **WARN** `polling.update_dropped` when one is
lost to shutdown. The silent version of that is what made this undiagnosable.

**Proven to catch the regression:** with the old ordering restored,
`TestFirstPollUpdateIsDeliveredWhileGetMeIsStillBlocked` does not merely fail — it hangs
until the test timeout, because `Start` never returns while `getMe` is blocked. 11 cases in
`polling_test.go` drive the real `Start`; the new log fields are separately proven to
survive redaction, without which the instrumentation would prove nothing.

**What this does not claim.** It does not explain the observed loss on its own: the device
log shows Telegram returning nothing before 21:51:59, and the first poll asked with an unset
offset, so by elimination the update was already gone from Telegram's queue. The one hop
neither repository can audit is Telegram's own queueing across managed-bot token issuance.
The next physical run settles it — `polling.started` followed by `polling.update_delivered
… first_update=true` for the *second* message and none for the first proves polling was live
and consuming while Telegram returned nothing.

**Also found, reported rather than built:** desktop pairing reports `applied` when the
gateway process has restarted, not when Telegram is receiving. It did not cause this failure
— the owner's first `/start` preceded even the completion response, since the bot's chat
exists in Telegram before PocketClaw has the token — and gating it needs the authenticated
health-detail token plumbed into the backend. Left for an owner decision.

Gates: source **27 PASS / 0 FAIL / 0 SKIPPED** with `flutter.suite 563 passed`, artifact
**25 PASS / 0 FAIL / 0 SKIPPED**, native ELF **188 PASS / 0 FAIL / 0 SKIP**, Zero-Pico PASS.
Verification APK `38da4acb1d30a9e46c8226a1f1e66067b6150e59c5bd8c1959cd2e5d4ac5e585`, 63,572,435 bytes, archived read-only at `build/forensic/apk-38da4acb1d30a9e46c8226a1f1e66067b6150e59c5bd8c1959cd2e5d4ac5e585.../`.
No device was attached to this session, so **nothing in this APK is physically verified**.

## 2026-09-14 — PC-DEF-060: managed Telegram onboarding from a desktop browser

Built to the owner's architecture. Core performs the pairing and the browser only ever
calls same origin, so it never needs cross-origin access to the hosted service and never
handles a credential.

- `pkg/telegramonboarding` — a Go client matching the Dart client's wire contract field
  for field, because both speak to the same deployment.
- Five same-origin endpoints under `/api/telegram/onboarding`, all requiring a Dashboard
  session because none is in the launcher auth allowlist.
- Completion goes through **`writeTelegramCredentials`**, extracted from
  `handleAndroidTelegramConfigure` so the Android bridge and the desktop flow share one
  writer. The owner contract — `AllowFrom` = exactly one positive numeric owner, plus
  PC-DEF-030's apply — lives there and is not reimplemented.

**What the browser never receives, and it is asserted rather than assumed:** the poll
token that authorises token collection (Core holds it against the pairing id; the fake
service 404s without it, so a passing status poll proves Core supplied it), the
onboarding service's URL (PC-DEF-052's rule kept for this client), and the bot token at
any point.

**The blocker is resolved with one source of truth.** Core did not know where the service
lives. Gradle already decodes the dart-defines, so
`android/official-onboarding.properties` → dart-define → `BuildConfig` → Core's
environment as `POCKETCLAW_ONBOARDING_BASE_URL`. No second place to set it. Only `https`
is accepted, since plain HTTP would put the poll token and once the bot token in the
clear. A deployment without the variable reports managed onboarding unavailable and keeps
the manual form.

**Deliberately not built: the QR code.** It was one option among several the owner
listed; rendering one needs a new frontend dependency and `pnpm` is not on PATH here, so
the other-device case is served by an openable *and copyable* Telegram link. Named rather
than silently skipped.

Verification: 10 backend cases and 13 UI cases; frontend **515 passed** across 38 files
with `tsc -b` and ESLint clean; 19 i18n keys in all 14 locales with parity green; Go suite
green under `-tags goolm` apart from the staged-Core freshness guard; `flutter analyze`
clean and Flutter **563 passed**; Android unit tests green; Zero-Pico PASS.

**Verification APK built, gated and archived.**
`249b765df37985067479455fa734cc0b0d96ef1247b1d1c4230306774f851847`, 63,573,451 bytes,
`0.2.0+62`, development signer `15cf75f9…`, Core fingerprint `d927abb9…` on both packaged
binaries, BuildTime `2026-09-14T01:42:43+0000`, Dart AOT `af14f0ec…` (5,768,072 bytes).

Gates: source **27 PASS / 0 FAIL / 0 SKIPPED**, artifact **25 PASS / 0 FAIL / 0
SKIPPED**, native ELF **188 PASS / 0 FAIL / 0 SKIP**, Zero-Pico PASS. Four fixes were
confirmed *inside the packaged Core* rather than in the source tree — the PC-DEF-060
endpoints and desktop UI, the PC-DEF-040 claim reconciler and the PC-DEF-059 redirect —
along with the onboarding URL compiled into `classes.dex`, which is what lets Core run
the pairing at all.

`FORENSIC.md` spells out that **PC-DEF-040 and PC-DEF-058 both require a fresh install**,
and that PC-DEF-060 is checked from a desktop browser with Telegram still unconfigured,
since a configured channel never offers Connect.

No device was attached to this session, so **nothing in this APK is physically
verified**. Archived read-only at
`build/forensic/apk-249b765df37985067479455fa734cc0b0d96ef1247b1d1c4230306774f851847/`.

## 2026-09-14 — PC-DEF-040 reopened and refixed at the claim; PC-DEF-059 verified

**PC-DEF-059 is PHYSICALLY VERIFIED PASS.** Native Settings → Manage Telegram /
Manage Models → authentication → the requested destination. The lesson is recorded
because it generalises: the first attempt passed every test it had and failed on the
device, because all of that coverage was route-level while the redirect that discarded
the destination happened before the routes existed.

**PC-DEF-040 reopened — physically reproduced, and it is not a regression I
introduced.** Audited at the owner's request: the last commits to
`launcher-setup.tsx`, `webview_android.dart`, `service_manager.dart` and
`public_mode_reconciliation.dart` all predate this session, and the `?next=` redirect
does not bypass the setup path (`Uri.path` ignores the query, so the hook's URL match
is unaffected). The original fix was structurally incomplete.

Root cause: `PC-DEF-039` narrows an unclaimed dashboard to loopback whatever the user
asked for, so desired and effective necessarily diverge until something re-applies the
preference — and the only thing that ever did was **the Android app noticing its
embedded WebView navigate away from `/launcher-setup`**. That is an inference from one
client's UI navigation rather than the event itself, so any other route to a first
claim left the listener on loopback with a manual toggle as the only recovery. The
bridge and the live rebind were never at fault, which is precisely why OFF→ON worked.

Now Core reacts to the claim itself. The runtime remembers `desiredPublic` beside the
effective value and exposes `ReconcileAfterDashboardClaimed()`, which
`POST /api/auth/setup` calls after a **first** claim — after the response, since
applying it replaces the listener carrying that request. PC-DEF-039 is intact and that
mattered more than the fix: the hook fires only on a first claim, that path already
refuses any non-loopback request, a remote claim is rejected before reaching it, a
failed claim never does, and a password change on an owned dashboard does not fire it.
An explicit `ApplyPublicMode` now also updates `desiredPublic`, so a retracted desire
is not resurrected by a later claim. The Android hook is kept as a second detector.

**PC-DEF-060 is partially addressed and honestly incomplete.** The wording fix shipped
and the owner confirmed the desktop page has BotFather, token, API base and proxy — and
confirmed the convenient "Connect to Telegram" flow is still absent. The owner's
architecture is right and is scoped in the defect log: a Go onboarding client,
same-origin proxy endpoints, completion through the existing authoritative writer
(`handleAndroidTelegramConfigure`, factored out rather than reimplemented), and the
Dashboard UI.

It is **not built**, and the reason is a blocker worth a decision rather than a guess:
**Core does not know the onboarding service URL.** It is a build-time dart-define in
the APK and has no Go equivalent. Either the Android host pushes it to Core at startup
(one source of truth, but no managed onboarding for a desktop-only deployment) or Core
gains a config field (covers desktop-only, but a second place to be wrong). That is
the first step of the milestone.

Verification: Go suite green under `-tags goolm` apart from the staged-Core freshness
guard (correct for the source commit); `web/backend` and `web/backend/api` green with
11 new cases; `flutter analyze` clean and Flutter **563 passed**; frontend **502
passed**.

Incidentally cleared: the long-standing `gofmt` struct-alignment drift in
`web/backend/unclaimed_dashboard_exposure_test.go`, which had been reported as
pre-existing for several milestones. `gofmt -l pkg/ web/` is now empty, so it is usable
as a clean signal again.

**Verification APK built, gated and archived.**
`ce151adbde6dc40e368908e213cd0178d04236ed7ab227be7771f051bb03ce80`, 63,554,007 bytes,
`0.2.0+62`, development signer `15cf75f9…`, Core fingerprint `886ce837…` on both
packaged binaries, BuildTime `2026-09-14T00:21:08+0000`, Dart AOT `af14f0ec…`
(5,768,072 bytes).

Gates: source **27 PASS / 0 FAIL / 0 SKIPPED**, artifact **25 PASS / 0 FAIL / 0
SKIPPED**, native ELF **188 PASS / 0 FAIL / 0 SKIP**, Zero-Pico PASS. PC-DEF-040's fix
lives entirely inside the Core binaries, so it was verified there rather than in the
source tree.

`FORENSIC.md` spells out that **PC-DEF-040 and PC-DEF-058 both require a fresh
install** — PC-DEF-040 because the fix reacts to a *first* claim, so the check is
meaningless unless the dashboard starts unclaimed, and clearing ownership means clearing
app data.

No device was attached to this session, so **nothing in this APK is physically
verified**.

## 2026-09-14 — PC-DEF-059 refixed server-side, PC-DEF-060, PC-DEF-057 part-verified

**PC-DEF-059 physically FAILED its first attempt.** The fix was frontend-only and
did nothing on the device. Traced through the real path this time: `_webUrl` loads
the WebView at `http://127.0.0.1:18800/models?lng=en`, and
`rejectLauncherDashboardAuth` answered `http.Redirect(w, r, "/launcher-login", 302)`
— **server-side, before one line of JavaScript loaded.** So `next` was absent
because the server never put one there, the login page correctly fell back to `/`,
and the router guard that built the `?next=` URL never ran because it lives on the
`/models` page, which was never served.

The frontend half was not wrong, it was unreachable — and every test from that round
was route-level, which cannot see a redirect that happens before the routes exist.
That is the lesson worth keeping: a fix for a navigation defect has to be tested
against the URL the native app actually launches.

Now the server carries it: `web/backend/middleware/post_auth_destination.go` builds
`/launcher-login?next=<path>` for a rejected page request whose path is a Dashboard
route, bare otherwise, with API and websocket rejections keeping their 401 shapes.
The value goes into a `Location` a browser follows and the path comes from the
request, so it is **allowlisted, never sanitised** — and a request-supplied `next`
is never reflected. The backend list is duplicated from the frontend's because Go
cannot import TypeScript, and a test **parses the TS file** to fail on drift; that
guard was confirmed to fail on a deliberately removed route.

**PC-DEF-060 — desktop Telegram, audited before changing anything.** The cause is
the host-bridge check (`window.__pocketclawHost`, injected only by the Android
WebView) — not responsive CSS, not a user-agent test, not a different route. Managed
pairing cannot run in a browser by mechanism: it needs the native flow to launch
Telegram and write the token.

The owner's report overstates slightly and it is worth recording accurately: desktop
does **not** silently omit management — the manual token form is the whole page and a
connected channel still shows its summary, because `configured` outranks the host
check. The real defect is narrower: the explanation said *"not available in this
build"*, which on a browser is untrue and sends the user looking for a different
build. One sentence was serving two different causes. Now `no-host` and
`host-without-endpoint` are separated, and the desktop copy names the actual route
(open PocketClaw on the phone, or create a bot with BotFather) with a BotFather link.
A browser-side managed flow was deliberately **not** built: it needs either
cross-origin calls the service does not permit or a second pairing state machine in
Go, which is its own milestone.

**PC-DEF-057 part-verified.** The device log confirms safe token metrics and provider
DEBUG fidelity: `max_tokens=32768`, `provider.request` with
`authorization_present=true`, `custom_header_count=4`, the OpenCode Go endpoint,
`session_header_present=true`, `tools=19`, and `provider.response` with `status=200`
and `duration_ms`; `prompt_tokens`/`completion_tokens`/`total_tokens` visible and no
sensitive value present. It stays OPEN overall until the rest of the acceptance
criteria are checked.

Verification: Go suite green under `-tags goolm` apart from the staged-Core freshness
guard (correct for the source commit); middleware package **40 tests**;
`flutter analyze` clean and Flutter **563 passed**; frontend **502 passed** with
`tsc -b` and ESLint clean; 2 i18n keys added in all 14 locales, parity green.

**A gate failure I caused, and closed.** The first Core built for this round failed
the native ELF audit — 187 PASS / 1 FAIL,
`libpocketclaw-web.so.build_path_privacy: prohibited path strings=1`. The string was
my own German translation: `PROHIBITED_PATH_MARKERS` in `tool/native_elf_audit.py`
includes the literal `PocketClaw-App` because it is this repository's directory name,
and I had written "in der PocketClaw-App unter Android", the idiomatic hyphenated
compound. Locale strings are compiled into `libpocketclaw-web.so`, so the audit
scanned it and cannot tell prose from a leaked path — it is right to be blunt.

Reworded to "in der App PocketClaw"; all 14 locales were scanned and no other
collided. A guard was added at the i18n layer so this is caught before a Core rebuild
and an APK build rather than after both, and it was confirmed to fail on the exact
string that broke the gate. Cost: one extra Core and APK cycle.

**Verification APK built, gated and archived.**
`41af6c4f96e8a1e64c62be075e0441ba072cd4b218b910c3c8e79daae749dbb1`, 63,554,799 bytes,
`0.2.0+62`, development signer `15cf75f9…`, Core fingerprint `5d19bf4a…` on both
packaged binaries, BuildTime `2026-09-13T23:42:30+0000`, Dart AOT `af14f0ec…`
(5,768,072 bytes). ABI arm64-v8a with plugin stubs and `lib/arm64-v8a/libdartjni.so`
present.

Gates: source **27 PASS / 0 FAIL / 0 SKIPPED** (`flutter.suite 563 passed`,
`core.staged_freshness PASS`, `native.elf_audit_contract PASS`), artifact **25 PASS /
0 FAIL / 0 SKIPPED**, native ELF **188 PASS / 0 FAIL / 0 SKIP**, Zero-Pico PASS.

Because PC-DEF-059's fix is in the Go middleware and the dashboard bundle, both of
which ship inside the Core binaries, the packaged Core was checked directly: it
contains the `launcher-login?` redirect target and the new desktop Telegram copy, and
zero prohibited markers. A stale pair would have tested the old behaviour, which is
how the previous attempt could have looked fixed and not been.

Archived read-only at
`build/forensic/apk-41af6c4f96e8a1e64c62be075e0441ba072cd4b218b910c3c8e79daae749dbb1/`.
No device was attached to this session, so **nothing in this APK is physically
verified**.

## 2026-09-14 — Notification permission, post-auth destination, logging fidelity

**PC-DEF-056 is PHYSICALLY VERIFIED PASS** (the owner calls it 054; see the
identifier note). Managed onboarding completes with no second "Open Chat in
Telegram". The first reply can take about **15-25 s** during initial runtime
activation — recorded as a **performance observation, not a defect**, and
deliberately not optimised. The stages are instrumented instead:
`runtimeReadyLatency` / `onboardingLatency` plus a
`pocketclaw.onboarding stage=telegram_running runtime_wait_ms=… onboarding_total_ms=…`
mark, so a later session can attribute the wait rather than guess. Both reset per
flow.

**PC-DEF-057 stays OPEN / PARTIAL** at the owner's instruction, with two
follow-ups fixed:

1. **Redaction was too aggressive** — the device log showed
   `max_tokens=<redacted>`. `token` is a substring of every credential worth
   hiding *and* every usage metric worth keeping, so explicit safe metadata is now
   evaluated **before** the broad match: an enumerated set of token measurements,
   the generic `_tokens` / `_token_count` / `_token_percent` shapes, and **the
   value's type deciding where the name cannot** — `tokens` is a count as a log
   field and a *map of credentials* as a struct field, so numeric is a metric and
   string stays secret. A hole was found and closed at the same time: the
   "facts about a credential" suffixes had included `_hash`/`_digest`, which made
   `dashboard_password_hash` read as safe. A hash of a secret is offline-crackable;
   both suffixes are gone.
2. **A configuration block was logged as a runtime failure** — `PC-E-AI-004`
   arrived as `ERR agent > LLM call failed`. `ErrorPayload` gained a
   `Classification` and `configuration_blocked` now yields **warning** severity,
   logged as `WARN agent > Turn blocked by configuration` with `reason` and `code`,
   and no failover-exhausted event since nothing was attempted. The **event kind is
   unchanged** so kind-routing consumers keep working, the zero value means
   ordinary failure, and the **user-facing reply is byte-identical** — asserted.

**PC-DEF-058 — first run never requested notification permission.**
`POST_NOTIFICATIONS` was *declared* and never requested, so the "PocketClaw
Running" notification never appeared on a fresh install. The rules are in a
testable `NotificationPermissionPolicy`: API 33 is the boundary, PocketClaw keeps
**its own record of having asked** because Android's
`shouldShowRequestPermissionRationale` cannot tell "never asked" from "refused for
good", the dialog is shown exactly once, a refusal offers Settings instead, and
"granted" is kept separate from "will appear" because below API 33 the Settings
switch is the only control. The ask is recorded *before* the dialog, since the
callback does not fire if the activity is recreated mid-dialog.

**PC-DEF-059 — authentication discarded the requested destination.** The native
cards open `/models` and `/channels/telegram`; the login page called
`location.assign("/")` on success. The destination now travels as `?next=`,
captured once on mount so a wrong-then-right password still lands correctly.
`next` is untrusted input ending in a navigation, so it is **matched against the
route set**, not sanitised — schemes, `//host`, backslash smuggling, control
characters, traversal, unknown paths and the auth pages are all rejected to a home
fallback. Routes were read from the generated route tree; none was invented.

Verification: Go suite green under `-tags goolm` apart from the staged-Core
freshness guard (correct for the source commit); `pkg/logger` **177 assertions**;
Android unit suite **26 tests, 0 failures** including 7 new policy cases;
`flutter analyze` clean and Flutter **563 passed**; frontend **493 passed** with
`tsc -b` and ESLint clean; 4 l10n keys added in all 12 locales.

**Verification APK built, gated and archived.**
`6bb32b387cc82c436247fde50c04216ae2a434a95015ad0f3a147b6d5935ba18`, 63,552,763
bytes, `0.2.0+62`, development signer `15cf75f9…`, Core fingerprint `66c247c3…` on
both packaged binaries, BuildTime `2026-09-13T22:48:21+0000`, Dart AOT
`af14f0ec…` (5,768,072 bytes). ABI arm64-v8a with plugin stubs and
`lib/arm64-v8a/libdartjni.so` present.

Gates: source **26 PASS / 0 FAIL / 0 SKIPPED** (`flutter.suite 563 passed`,
`core.staged_freshness PASS`), artifact **25 PASS / 0 FAIL / 0 SKIPPED** as
`non-publish-audit`, native ELF **188 PASS / 0 FAIL / 0 SKIP**, Zero-Pico PASS.

Archived read-only at
`build/forensic/apk-6bb32b387cc82c436247fde50c04216ae2a434a95015ad0f3a147b6d5935ba18/`.
Its `FORENSIC.md` carries the exact per-defect device checks rather than only the
identity, and flags that **PC-DEF-058 requires a fresh install** — which means
uninstalling first, so anything needed from the workspace must be saved beforehand.

No device was attached to this session, so **nothing in this APK is physically
verified**.

## 2026-09-14 — Telegram readiness race and logging hardening

Samsung results moved the Telegram diagnosis decisively, and two defects follow.
A third identifier collision: the owner allocated **PC-DEF-054** to the readiness
race, but 054 was already the signature-plaintext defect, so the race is
**PC-DEF-056** and the logging work **PC-DEF-057**.

**Physically verified PASS, now RESOLVED in the log:** PC-DEF-032 (real OpenCode
Go inference), PC-DEF-049 (Manage Provider present, including provider deletion
and credential management), PC-DEF-053 (Telegram replied
`PC-E-AI-004` with the actionable disabled-model text, classified and delivered
per the runtime log), and **PC-DEF-051**, which was never a
missing-registration defect: the device shows `getMyCommands → ok=true`,
`registered=14`, and the menu visibly exposes `/start` and the rest.

**PC-DEF-056 — the final bot chat opened before the runtime was ready. Root cause
proven in source, two independent faults.**

1. `telegram_onboarding_page.dart` called `pausePolling()` on
   `paused`/`hidden`/`detached` — exactly the window the user spends in Telegram.
   So the pairing result was never consumed while it became ready: no token, no
   config, no Telegram channel. The bot chat Telegram navigated to belonged to a
   bot PocketClaw had not finished creating. On return, `resumePolling()` did all
   of it at once, which is why the second Open Chat worked and why no manual
   restart was ever involved.
2. `connected` was declared as soon as `_reloadCore()` returned, and that returns
   when the restart has been *requested* — `restartCore` hands Android one intent
   and comes back. Configuration applied is not runtime running.

Fixed by keeping polling alive across the handoff (still bounded by the pairing's
`expiresAt`, with `restore()` covering a killed process), and by gating
`connected` on Core reporting the channel running through PC-DEF-027's
`resolveTelegramRuntimeState` — bounded at 45 s, polled rather than slept, with a
failed status read counted as silence rather than failure. On reaching connected
the flow opens the bot chat itself, exactly once, so no second action is needed. A
runtime that never starts yields `runtimeNotReady`: the bot is saved, the start is
outstanding, and the user is never sent into a dead chat.

The tests found a bug the device could not have shown cheaply: `reset()` during
the readiness wait did not abort it, so a cancelled pairing whose runtime came up
later declared itself connected and opened a chat the user had walked away from.

**PC-DEF-057 — structured log fields were not recursively redacted.** Two shapes
survived: a field *named* for a credential whose value no pattern recognises
(`token: "hunter2"` is not `sk-…`, has no vendor prefix, is not `KEY=value`), and
anything nested in a map, slice or struct, which the `default:` branch handed
straight to the encoder. The sensitive-name list held three entries and none of
them was `api_key`, `authorization`, `bot_token` or `password`.

Now a central layer classifies field names — normalised, substring-matched, so
`bot_token`, `proxy_password`, `crypto_passphrase` and `x-opencode-session` are
all caught — and walks non-primitives by JSON shape, so what is checked is exactly
what the encoder would have written. DEBUG stays worth reading: a bool is never
redacted, `_present`/`_set`/`_changed`/`_count`/`_digest` suffixes survive, and
`auth_method`, `changed_fields` and `token_type` are metadata. Added the owner's
exemplar line, `provider.request`/`provider.response`/`provider.transport_failed`,
with endpoints reduced to scheme+host+path and header facts as booleans.

Verification: 46 redaction cases driven by a canary chosen so only the name rule
can catch it, including an end-to-end pass through the real emit path at every
level reading the writer's bytes. `pkg/logger` 50 tests. Full Go suite green under
`-tags goolm` apart from the staged-Core freshness guard, which correctly reports
the pair as stale for the source commit. `flutter analyze` clean, Flutter **561
passed**, frontend **475 passed**, `tsc -b` and ESLint clean.

**Verification APK built, gated and archived.**
`7c8eefd399318b6187b0fb87c8bd687d7d08c3537959e31f38767a642908e3cb`, 63,539,059
bytes, `0.2.0+62`, development signer `15cf75f9…`, Core fingerprint
`d1d3b98f…` on both packaged binaries, BuildTime `2026-09-13T21:44:42+0000`, Dart
AOT `007c23b6…` (5,702,536 bytes). ABI arm64-v8a with plugin stubs and
`lib/arm64-v8a/libdartjni.so` present.

Gates: source **26 PASS / 0 FAIL / 0 SKIPPED** (`flutter.suite 561 passed`,
`core.staged_freshness PASS`), artifact **24 PASS / 0 FAIL / 0 SKIPPED** as
`non-publish-audit`, native ELF **188 PASS / 0 FAIL / 0 SKIP**, Zero-Pico PASS.

Archived read-only at
`build/forensic/apk-7c8eefd399318b6187b0fb87c8bd687d7d08c3537959e31f38767a642908e3cb/`
with `FORENSIC.md` naming what to check on the device, and the private R8 material
in `private-do-not-distribute/`.

No device was attached to this session, so **nothing in this APK is physically
verified**. Outstanding on the Samsung: PC-DEF-056, PC-DEF-057, and the still
unverified PC-DEF-050, PC-DEF-052 and PC-DEF-055.

## 2026-09-13 — Onboarding privacy, actionable errors, signature secrets

Four items: two owner requirements added after the previous milestone, and two
this session had disclosed and was told to act on rather than defer. Recorded as
**PC-DEF-052..055**; the owner named 052 and 053, and 054/055 took the next free
identifiers because each carries its own evidence and resolution.

**PC-DEF-054 — signature material carried credentials in plaintext. RESOLVED,
not deferred.** Investigated first, because it had a stop-and-ask condition on it.
Wider than originally disclosed: the mechanism is `canonicalizeSignatureValue`
resolving `SecureString`/`SecureStrings` to plaintext, and it fed channel settings
as well as the `webcfg:` component. Proven by test that a Brave web-search key, a
proxy URL password and a Telegram bot token were each present verbatim.

The owner's five exposure questions, answered: **not** logged, **not** persisted,
**not** returned to any client (only the derived `gateway_restart_required`
boolean is), **not** in any error or panic text, and **no crash reporter exists at
all** — Firebase/Crashlytics went under PC-DEF-R005. So it was never a disclosure;
it was unnecessary plaintext retained in long-lived package state. Fixed by
digesting each payload, which is why no approval to defer was needed. Nine
sub-cases prove the signature still moves when each secret changes.

**PC-DEF-052 — onboarding exposed the hosting origin. FIXED IN SOURCE.** Every
launch site was read: the screen opens exactly two URLs and `qr_payload` is
rendered rather than navigated to, so `pairing.deepLink` is the only candidate,
and the tracked endpoint in `android/official-onboarding.properties` is a
`*.vercel.app` deployment whose service returns its own redirect there. That
service lives in a separate repository, so the fix could not be "change the
service". The app now resolves the setup link to a Telegram destination **in the
background** and opens only `t.me` / `telegram.me` / `telegram.dog` / `tg://`; a
link that does not resolve is **refused, not opened**. It is also the security
boundary that was missing — a URL from a network response was going straight to
the OS — now bounded to 5 hops, https-only, and sending no credential to an
unvetted host.

**PC-DEF-053 — first-run Telegram got an internal error. FIXED IN SOURCE.** Root
cause proven by reading the path: `startupBlockedProvider.Chat` returned
`fmt.Errorf("no default model configured; gateway started in limited mode")`,
which no classifier recognises, so it reached the chat window as "Error processing
message: ..." — jargon, no instruction, and nothing separating "Telegram works"
from "no AI configured". Replaced by `agent.UserFacingError` with stable codes
`PC-E-AI-001..004`, chosen by `AIConfigurationProblem`, checked first by
`formatProcessingError`.

A wrong first attempt is worth recording: the check was written as a precondition
in `processMessage` and broke two existing tests, which were right —
`NewAgentLoop` takes an **injected** provider, so an empty `model_list` does not
mean there is nothing to send a request to. It belongs where the gateway already
decides it cannot build one. The check also deliberately does **not** judge
credential usability; that needs the OAuth store and probe
`hasModelConfiguration` owns, and a second copy is the drift that had
`pkg/modelaccess` reverted.

The other error categories the owner listed were already covered by
`error_format.go` and `provider_detail.go`. The web UI's equivalent for the
configuration category is already localised in all 14 bundles
(`chat-empty-state.tsx`, verified key by key). **Core has no locale field and no
i18n layer**, so Telegram replies are English as every other Core reply is; the
codes exist so a later layer can localise without touching Core. Open item, not
guessed at.

**PC-DEF-055 — Set Default audit. FIXED IN SOURCE.** Label and default-state
visibility pass (badge, accent edge, filled star). Touch discoverability failed
twice — a target smaller than the 40px Edit and Delete use on the same card, and a
disabled reason living only in a Radix tooltip that never opens on touch. Fixed to
exactly those two points, no redesign.

Verification: source release gate **26 PASS / 0 FAIL / 0 SKIPPED** in test class,
including `flutter.suite 549 passed` and `core.staged_freshness PASS`; full Go
suite green under `-tags goolm`; `go vet` clean; frontend **475/475** across 35
files with `tsc -b` and ESLint clean; `flutter analyze` clean; `no_active_pico`
passes with 19 allowlist entries all in use.

Pre-existing and untouched: `gofmt` reports struct-field alignment drift in
`core/src/web/backend/unclaimed_dashboard_exposure_test.go`, which this work does
not touch and no gate checks.

**Verification APK built, gated and archived.**
`6df7abaa6bec5a5124d21d30b582fc37d2837be036fa75e93eb3d30d5634894c`, 63,528,459
bytes, `0.2.0+62`, development signer `15cf75f9…`, Core fingerprint
`212131a8…` on both packaged binaries, BuildTime `2026-09-13T19:56:34+0000`,
Dart AOT `d5d52742…` (5,702,536 bytes). ABI arm64-v8a with plugin stubs, and
`lib/arm64-v8a/libdartjni.so` present — the black-screen trap from 2026-08-25 is
not reintroduced.

Gates: source **26 PASS / 0 FAIL / 0 SKIPPED** (including `flutter.suite 549
passed` and `core.staged_freshness PASS`), artifact **24 PASS / 0 FAIL / 0
SKIPPED** as `non-publish-audit`, native ELF audit **188 PASS / 0 FAIL / 0 SKIP**,
Zero-Pico PASS. `artifact.dart_snapshot_paths` passes here, where the H2
production artifact had it as its one known skip.

Archived read-only at
`build/forensic/apk-6df7abaa6bec5a5124d21d30b582fc37d2837be036fa75e93eb3d30d5634894c/`
before any later build can overwrite the Gradle output, with `FORENSIC.md`
recording the identity and the private R8 material moved into
`private-do-not-distribute/` — `PC-DEF-021`'s rule applied to the archive itself.

**Build gotcha worth keeping:** the hardened build needs `JAVA_HOME`. The first
attempt failed at `:app:clean` with "JAVA_HOME is not set", which a backgrounded
`nohup` reported as exit code 0. The toolchain JDK is
`/home/lordegypt/PocketCLaw/.tooling/jdk-17`.

Not merged, not tagged, nothing published.

## 2026-09-13 — Provider CRUD, credential apply, and Telegram command menu

Samsung physical results corrected the ledger and opened three defects. The
owner's numbers 047/048/049 collided with entries already in use, so they are
recorded as **PC-DEF-049, PC-DEF-050, PC-DEF-051**; `docs/DEFECT_LOG.md` carries
an identifier note and each entry names the owner's number.

Confirmed physically PASS and now RESOLVED in the log: **PC-DEF-030** (Telegram
automatic runtime apply), **PC-DEF-032** (OpenCode Go — `deepseek-v4.1-flash`
produced a real Chat response), **PC-DEF-033** (the amber warning block). Model
Delete exists and PC-DEF-047's fix is confirmed; the management gap was at the
provider level, not the model level.

**The data-model audit the owner asked for is `docs/PROVIDER_ARCHITECTURE.md`
section 13.** Its finding: credentials are **model-scoped** and there is no
provider record anywhere in the configuration schema. `config.Config` holds only
`model_list`, and each entry carries its own `provider` label, `api_base`,
`api_keys`, `proxy` and `custom_headers`. Two models of one provider that share a
key hold two copies of the same secret. A "provider" is a derived grouping over
that label, so provider management is implemented as a view over `model_list` and
introduces no provider object — a derived view cannot disagree with the models it
is derived from, a stored one can. A stored provider record with per-model
overrides remains open as a schema change; it needs a config version, a migration
and a rule for which of several disagreeing keys wins.

**PC-DEF-050's root cause is proven, not inferred.** `computeConfigSignature`
decides `gateway_restart_required`, and it covered no credential, endpoint or
header of any `model_list` entry. A rotated key was persisted correctly, the
console was told no restart was required, it reported success, and the running
gateway kept the old credential. Computing the signature either side of a
rotation before the fix produced a byte-identical string — likewise for
`api_base` and `custom_headers`. Fixed by digesting that material into the
signature; secrets are SHA-256 digests, never plaintext.

**PC-DEF-051 is OPEN with cause UNKNOWN.** The command registry is intact — 14
definitions, all publishable, `/start` first — and `Start` still passes the whole
set to registration with retry. What was provably wrong is that the success log
printed the number of definitions *received* rather than what Telegram accepted,
so the historical `count=14` never proved the menu was populated. That
observability is fixed; the cause must come from a device log line, not a guess.

Verification at this point: full Go backend suite green
(`web/backend/...`, 26 s), `pkg/commands` and `pkg/channels/...` green under
`-tags goolm`, `go vet` clean; frontend 469/469 Vitest across 35 files, `tsc -b`
clean, ESLint clean on every changed file; i18n parity green with 26 new keys
added in all 14 locales; `tool/no_active_pico.py` passes with 19 allowlist
entries all in use; source release gate 15 PASS / 0 FAIL in test class. No Dart
file changed, so the Flutter suite is untouched by this work.

Not merged, not tagged, nothing published — per the owner's instruction.

The H2 production validation APK has SHA-256
`f0d83298c2ce061c01a9fc931ad29676e4d4b646bb5b204a9bf0002b11a7f46f`
and signer SHA-256
`176dca6b198b9552fb4d9ad3ca18da8d6f23c0a3f5ed4bd6b75a0700f9f0efcf`.
Its production artifact gate result is 20 PASS / 0 FAIL / 1 SKIPPED, with
`artifact.dart_snapshot_paths` the one known skip. It is private validation
evidence only: not published, not installed, not accepted, and not a final
release.

H3A established the canonical hardened Android build command in
`tool/build_hardened_android.py`. Its second clean LOCAL TEST / NON-RELEASABLE
APK is `build/app/outputs/apk/release/app-release.apk`, 63,783,811 bytes, SHA-256
`23dbaa24f375057faf30b469b3a8cafb1a1c235c9afacea15f944df41ad63894`, signed
by the development certificate
`15cf75f9945d5354e75707e0326b7cffc60ac51a68df38156db318ef4578a27c`.
The test artifact gate is 25 PASS / 0 FAIL / 0 SKIPPED. In that artifact,
`artifact.dart_snapshot_paths` passes, the stable generated URI is
`package:pocketclaw_generated/dart_plugin_registrant.dart`, and Dart AOT SHA-256
is `c7b2a885ff843a20c57097a0d16ba07c728bd64cf17455a1ce61e6f463a5ae77`.
The external private DWARF SHA-256 is
`0f52873bc712fe0c17d636f5bdb7d08ee80cdacfe633cf6a786dd2c6d93b8acc`.
Neither this APK nor its symbols are committed, installed, published, accepted,
or production-signed. It remains the H3A local-test evidence that H3B later
validated under the enrolled production signer.

H3B validated the same Dart-hardening contract with the enrolled production
signer. The private validation APK is
`build/app/outputs/apk/release/app-release.apk`, 63,787,907 bytes, SHA-256
`ceef6640d8abd9d084c3ff37d8e903aaf3c82b287de65ec15a37d91124bdebe6`,
package `com.lord1egypt.pocketclaw`, version `0.2.0` (62), with an arm64 product
payload and the already accepted plugin ABI stubs. Independent `apksigner`
inspection found exactly one v2 signer with certificate SHA-256
`176dca6b198b9552fb4d9ad3ca18da8d6f23c0a3f5ed4bd6b75a0700f9f0efcf`.
The production artifact gate passed 25 checks with 0 failures and 0 skips.

Its Dart AOT SHA-256 is
`c7b2a885ff843a20c57097a0d16ba07c728bd64cf17455a1ce61e6f463a5ae77`,
byte-identical to H3A. The private split DWARF is 3,513,592 bytes with SHA-256
`0f52873bc712fe0c17d636f5bdb7d08ee80cdacfe633cf6a786dd2c6d93b8acc`,
also byte-identical to H3A. Every ZIP entry payload matches H3A; the whole APK
hash differs because the signing block differs. Private symbols remain ignored,
external to the APK, untracked, and unpublished. The APK was not installed,
published, accepted, or committed, and it does not advance vc62.

Private keystores, keys, passwords, tokens, and recovery secrets are external to
Git. The repository carries only the public certificate digest and public build
evidence. The owner confirmed a separate backup of the developer production
keystore exists.

H4A removed PocketClaw's blanket application, Flutter, plugin, Firebase, Umeng,
Tika, JSON-constructor, enum, and resource keep rules. The project application
rules file is intentionally empty of active rules: aapt keeps manifest-created
components, Flutter 3.47.1 supplies its embedding rule, and dependencies supply
their consumer rules. The inherited file-picker/Tika, background-service, and
Dart-JNI broad consumer rules remain dependency-owned runtime contracts and
were not overridden without evidence.

The clean H4A LOCAL TEST / NON-RELEASABLE APK is
`build/app/outputs/apk/release/app-release.apk`, 63,556,335 bytes, SHA-256
`db7fa8cb190fcebc160b2c718d9c120de296efb378d8a1196a5ff722ba3e1f78`,
signed by the development certificate
`15cf75f9945d5354e75707e0326b7cffc60ac51a68df38156db318ef4578a27c`.
Its two DEX entries total 2,283,480 bytes, 614,820 bytes below the H3B DEX
baseline. Three sampled internal PocketClaw classes are renamed and three are
removed or folded; all six clear descriptors are absent from DEX, while the
four manifest components remain preserved. The private R8 mapping is
`build/app/outputs/mapping/release/mapping.txt`, 13,630,085 bytes, SHA-256
`14d49fad46e773e32da69b7b2336b7a968808cd1130f0319f7806ca4d09c1beb`.
It is ignored, untracked, external to the APK, and unpublished. The local-test
artifact gate passed 30 checks with 0 failures and 0 skips.

H3 Dart hardening is unchanged: H4A AOT SHA-256 remains
`c7b2a885ff843a20c57097a0d16ba07c728bd64cf17455a1ce61e6f463a5ae77`,
the controlled generated URI and `artifact.dart_snapshot_paths` pass, and the
external Dart symbol SHA-256 remains
`0f52873bc712fe0c17d636f5bdb7d08ee80cdacfe633cf6a786dd2c6d93b8acc`.
The APK and both private support artifacts were not installed, committed,
published, or accepted. The production signer was not accessed.

H4B validated the exact H4A R8/ProGuard contract under the enrolled production
signer. The private production-validation APK is
`build/app/outputs/apk/release/app-release.apk`, 63,560,431 bytes, SHA-256
`14ba7d138a4092aefe264c7e2af6240c97fc1b782ded69918cbf545351eb5eb2`,
package `com.lord1egypt.pocketclaw`, version `0.2.0` (62), with an arm64 product
payload and accepted plugin ABI stubs. Independent `apksigner` inspection found
exactly one v2 signer with certificate SHA-256
`176dca6b198b9552fb4d9ad3ca18da8d6f23c0a3f5ed4bd6b75a0700f9f0efcf`;
the development signer was not used. The production artifact gate passed 30
checks with 0 failures and 0 skips.

Both DEX entries are byte-identical to H4A: `classes.dex` is 2,069,524 bytes,
SHA-256 `6de32319f187e9dd26d8313a4a44cb8f94f2f08c2e0773bd9d5f0e4e067a386b`,
and `classes2.dex` is 213,956 bytes, SHA-256
`e16d9e1acfb5180aab18d10942562486710c53636daec37606f4f0d3a18e6e5f`.
The 13,630,085-byte private mapping remains byte-identical to H4A at SHA-256
`14d49fad46e773e32da69b7b2336b7a968808cd1130f0319f7806ca4d09c1beb`.
The Dart AOT and private-symbol hashes also remain the H3/H4 values
`c7b2a885ff843a20c57097a0d16ba07c728bd64cf17455a1ce61e6f463a5ae77`
and `0f52873bc712fe0c17d636f5bdb7d08ee80cdacfe633cf6a786dd2c6d93b8acc`.
Core and all eight Managed Runtime payloads are byte-identical to H4A. Mapping
and Dart symbols remain ignored, untracked, external to the APK, and private.
The APK was not installed, published, accepted, or committed and does not
advance vc62.

H5A audited all 18 ELF entries in that exact H4B APK without rebuilding or
modifying any binary. All have correct ABI/type/entrypoint roles, at least
16 KiB-compatible PT_LOAD alignment, non-executable stacks, no writable plus
executable load segment, no TEXTREL, and no packaged source DWARF, `.symtab`, or
`.strtab`. Imported dynamic ELFs have GNU RELRO plus BIND_NOW; static PIE
payloads have no lazy-binding surface. `PC-DEF-008` is resolved as verified
non-blocking because packaged `libapp.so` remains stripped and its private Dart
DWARF remains external.

The audit records four final-policy findings for H5B: one build-only RUNPATH in
`libpocketclaw-git-remote-http.so`, and neutral
`/tmp/pocketclaw-runtime-build` strings in curl, Git HTTP, and Python. No ELF
contains `/home/lordegypt` or the PocketClaw checkout path. Current Core/runtime
recipes also lack a private symbol-capable companion, and broad dependency
exports require reachability evidence before any safe narrowing. These are
tracked as `PC-DEF-009` through `PC-DEF-012`; none blocks the completed
audit-only H5A, but `PC-DEF-009` through `011` must be addressed before final
native-policy acceptance.

The durable native policy is category-specific. Private support artifacts will
live under ignored `build/private-symbols/native/android-arm64/`, with hashes,
build IDs, source/toolchain inputs, and the exact shipped APK association in a
private manifest. No native support artifact was created, packaged, published,
or committed in H5A. The full inventory, per-entry hashes/build IDs, security
matrix, export analysis, and ordered H5B plan are in
[`docs/prompts/history/H5A_NATIVE_ELF_AUDIT.md`](docs/prompts/history/H5A_NATIVE_ELF_AUDIT.md).

H5B rebuilt the ten project-owned native executables and left the four
dependency-owned ELFs untouched. `PC-DEF-009` and `PC-DEF-010` are resolved at
the source/link step: the Git HTTP helper's `DT_RUNPATH` is gone because the
recipe passes explicit `CURL_CFLAGS`/`CURL_LDFLAGS` instead of `CURLDIR`, and no
packaged ELF contains any build root, `/home/lordegypt` or the checkout path.
jq and CPython needed their generated source inputs normalized, because a
prefix map cannot rewrite a string the build compiled in as a C literal. No
finished binary was patched.

`PC-DEF-011` is implemented. Every owned recipe now builds with debug
information, strips the shipped payload and derives a `.debug` companion from
the same link through `tool/native_support.py`, which proves the companion
symbolizes a representative function and records the pair in an ignored private
manifest under `build/private-symbols/native/android-arm64/`. That manifest has
ten entries, binds each companion to its shipped hash, size and build ID, and is
bound to the exact candidate APK. `PC-DEF-012` remains open by decision: H5B
produced no reachability evidence, so no export map was added.

The staged Core pair is `libpocketclaw.so`, 37,724,640 bytes, SHA-256
`62f741be6f71f7518ba0df8f4457e0f7b666dfe512cde88e79e213ca0907e901`, and
`libpocketclaw-web.so`, 25,517,952 bytes, SHA-256
`42d418bd3e2d1863d2dda4d46357e541a222b831cfffb17e5f36e442e455e9f5`. Both carry
source fingerprint
`86369a32a9873715672f7867b31dcd72a7d19088c49cdb1df2b584c548ba4c73` and
`BuildTime` `2026-09-12T05:01:21+0000`.

All eight Managed Runtime payloads and all ten companions reproduced
byte-identically across three independent build roots; the Core pair reproduced
across three output roots. `runtime/android-build-env.sh` pins
`RUNTIME_EPOCH=1789157892` as a build input because `libpocketclaw-python.so`
embeds its date, so the catalog stays reproducible from the tree that records
it.

The H5B candidate is a LOCAL TEST / NON-RELEASABLE APK,
`build/app/outputs/apk/release/app-release.apk`, 63,560,467 bytes, SHA-256
`d4fe2c4a035051e3b6500d2a2fe9bdad639c97323c355b26a3f8ae6f215b9dd8`, with exactly
one v2 signer, the development certificate
`15cf75f9945d5354e75707e0326b7cffc60ac51a68df38156db318ef4578a27c`. The enrolled
production signer was not used. The enforced native audit is 194 PASS / 0 FAIL /
0 SKIP and the full test-class release gate passes with `releasable: false`.
Packaged Dart AOT, the private Dart DWARF and the private R8 mapping are
byte-identical to H3A/H3B/H4A/H4B. The APK was not installed, published,
accepted, or committed, and it does not advance vc62. Full evidence is in
[`docs/prompts/history/H5B_NATIVE_HARDENING.md`](docs/prompts/history/H5B_NATIVE_HARDENING.md).

H5B was then validated physically and under production signing.

The exact H5B LOCAL TEST APK `d4fe2c4a…` was installed in place on the Samsung
SM-A165F (`RK8Y6016N5V`) as a same-identity `adb install -r` upgrade — the
installed vc62 was first verified as carrying the same development certificate,
so no signature-mismatch wipe was possible. Install returned Success; package,
version, `appId`, `dataDir`, `firstInstallTime`, the external data tree and all
ten permission grants were unchanged, and the base APK pulled back off the
device was byte-identical to the candidate. The owner's manual native/runtime
smoke returned **PASS** with no failing item.

H5C rebuilt the same tree under the enrolled production signer and changed no
native source, recipe or export map. The private production artifact is
`build/app/outputs/apk/release/app-release.apk`, 63,564,563 bytes, SHA-256
`3774202ef9832c70ffa376e663db1da69e17ae9318df4cc8cb31156fc0c7eae7`, with exactly
one v2 signer, RSA 4096, certificate
`176dca6b198b9552fb4d9ad3ca18da8d6f23c0a3f5ed4bd6b75a0700f9f0efcf`. The
development certificate is absent. It is 4,096 bytes larger than the H5B
candidate purely because of the signing block.

All **18** packaged ELF entries are byte-identical between the physically
validated H5B APK and the production-signed H5C APK, so the owner's device smoke
transfers to this artifact: it carries exactly the native payloads that were
exercised on hardware. The enforced native audit is 194 PASS / 0 FAIL / 0 SKIP
with no status difference from H5B, and the ten-entry private support manifest
is rebound to the production APK with every shipped/support hash, size and build
ID still matching. Packaged Dart AOT, the private Dart DWARF and the private R8
mapping remain byte-identical to H3A through H5B.

The production release gate passes **55 PASS / 0 FAIL / 0 SKIPPED** with
`releasable: true` — `PASS — production release candidate`. The artifact was not
installed, published, accepted, or committed. The production APK must not be
installed over the current device state: the installed path is
development-signed and a cross-signer `adb install -r` is forbidden. Full
evidence is in
[`docs/prompts/history/H5C_PRODUCTION_NATIVE_VALIDATION.md`](docs/prompts/history/H5C_PRODUCTION_NATIVE_VALIDATION.md).

UI-1, an isolated guided-tour repair authorized between H5C and the exposure
audit, then fixed six confirmed console defects — `PC-DEF-013` through
`PC-DEF-018`, all resolved. The tour now places its card logically and clamps it
inside the viewport, blocks click-through to the control it spotlights, restores
focus on every termination path, resolves targets under a bounded frame budget,
keeps geometry synchronized while a step is live, always offers Escape and
click-outside, and versions its persisted state. A step that pointed at a
documentation button the console does not have was removed. The full frontend
suite is 418 tests in 25 files, all passing; Chrome and Brave confirm every step
on-screen and reachable in both Arabic and English.

UI-1 changed no native binary, signing input, Core payload or Managed Runtime
payload, and did not touch the release roadmap. It did move one thing:
`libpocketclaw-web.so` embeds the compiled dashboard, so the Core source
fingerprint is now
`bd4a8629a2682e2f05aa3859a400be8a77fb4954ad14994e5703ccbe365d05ec` and the
staged Core pair predates it. UI-1 had no authority to rebuild Core, so this is
tracked as `PC-DEF-019`: **Core must be rebuilt and re-staged before the next
artifact build.** The H5C production artifact and its evidence remain valid for
the dashboard they were built from. Evidence is in
[`docs/prompts/history/UI-1_GUIDED_TOUR_HARDENING.md`](docs/prompts/history/UI-1_GUIDED_TOUR_HARDENING.md).

`PC-DEF-019` is now **RESOLVED**. Core was rebuilt and re-staged from canonical
build-input commit `ea43369289c8b6c618faa08f7b91355882fc050c` — the UI-1 source
commit itself — under the two-commit rule, so the staged pair landed in a
following commit that changes no build input and the source fingerprint stayed
`bd4a8629a2682e2f05aa3859a400be8a77fb4954ad14994e5703ccbe365d05ec`.

    libpocketclaw.so       37,724,640 bytes
                           f273b9ced85f4d00cb542df9c2f4c691b4151526cb0ac9c2c7612a1432d7230f
                           build ID 25e206ab402f8cd44a766bc03935468beebd8633
    libpocketclaw-web.so   25,517,952 bytes
                           900c43fcaad2094017c6959eed623d1e2499cfd560f01f2cff36dd34202b86b9
                           build ID 84afbe2439b779722b22c2b4c6aa1300cd3ef199
    BuildTime              2026-09-12T07:27:12+0000

Both binaries were reproduced byte-identically in three independent output
roots, one with a cold Go build cache, and so were both private-support
companions. `core.staged_freshness` and `build.reproducibility_tests` are PASS.
The embedded dashboard is identified by content rather than by timestamp: UI-1's
`__pocketclaw_tour_probe__` is present and the deleted docs-step copy is absent,
both reversed in the binary staged at `ea43369`. The pair holds the H5B/H5C
native hardening contract at 22 PASS / 0 FAIL under the repository's own ELF
audit logic. Only the Core pair changed — no Managed Runtime payload, export map
or `PC-DEF-012` disposition — and no APK or AAB was built, so the private
support manifest stays bound to the H5C candidate until the next artifact build
rebinds it.

This was a prerequisite repair, not a release milestone. The H5C production APK
`3774202ef9832c70ffa376e663db1da69e17ae9318df4cc8cb31156fc0c7eae7` remains
historically valid for the dashboard it was built from and is **not** relabelled
as containing UI-1; this Core pair belongs to the post-UI-1 source state and
reaches the next artifact build. Evidence is in
[`docs/prompts/history/PC-DEF-019_CORE_REBUILD_RESTAGE.md`](docs/prompts/history/PC-DEF-019_CORE_REBUILD_RESTAGE.md).

The **final release exposure audit** then ran at `25753ce` and is **BLOCKED**.
It completed every part that does not depend on the production signing identity
and opened six defects, fixing none: each carries a narrow fix plan and waits
for its own authorization.

Two are release blockers. `PC-DEF-020`: the dashboard's public/loopback decision
has two persisted authorities and the Android Public Mode OFF path never writes
`launcher-config.json`, so a `public: true` left in that file rebinds the
console to all interfaces on the next service start while the native toggle
still reports OFF. Enforcement is not the problem — the password wall holds in
every state and the Core gateway on 18790 is loopback-only in both — durability
is. `PC-DEF-021`: `:app:bundleRelease` embeds ~39.5 MB of release-support
material in `BUNDLE-METADATA/`, including `proguard.map` byte-identical to the
private R8 mapping and native debug symbols for the obfuscated Dart AOT
library. That is correct for a Play upload and wrong for a public release asset
— and `PocketClaw-v0.2.0-rc1.aab` and `-rc2.aab` are published GitHub assets
today, so the practice that would leak it is already established. The APK is
clean; this is bundle-only.

The other four: `PC-DEF-022`, an `/api/update` route that fetches and extracts
an arbitrary caller-supplied URL with no provenance check, authenticated and
unused by any PocketClaw UI; `PC-DEF-023`, a base64-wrapped third-party Google
OAuth client secret embedded in both Core binaries, found by entropy review
after the pattern scan missed it; `PC-DEF-024`, a dead `um.placeholder://`
BROWSABLE deep link exported on `MainActivity` for an SDK that is not packaged;
and `PC-DEF-025`, `flutter test` red since `aa24d9e` on a stale assertion, which
went unnoticed because the release gate runs three Flutter test files rather
than the suite.

`PC-DEF-002` is **resolved as explained**: `0.0.0.0:18800` is Public Mode ON and
nothing else, proved with reproducible bind evidence, and its one actionable
residue is now `PC-DEF-020`. `PC-DEF-006` stays open — one artifact was built,
not two compared. `PC-DEF-012` is unchanged by decision.

The audit deliberately did **not** request the owner signing ceremony: a
production candidate built before these blockers are fixed would have to be
rebuilt, and its hash, native-support binding and gate evidence would all be
superseded. It audited a fresh LOCAL TEST / NON-RELEASABLE APK from this exact
tree instead — 63,560,203 bytes, SHA-256
`5460d86a80d74219a39554e4da2ceb819c708c32050789396c95e394c533346a` — plus a
debug-signed NON-PUBLISH structural AAB. Both carry the post-UI-1 Core pair
`f273b9ce…` / `900c43fc…` and the H5B Managed Runtime byte-for-byte. Native
audit 194 PASS / 0 FAIL / 0 SKIP with the support manifest rebound to the fresh
APK; source gate 25 PASS / 0 FAIL / 0 SKIPPED; artifact gate 54 PASS / 0 FAIL /
0 SKIPPED in test class; Dart AOT, private DWARF and private R8 mapping all
byte-identical to H3A onward. Full evidence is in
[`docs/prompts/history/EXPOSURE_AUDIT.md`](docs/prompts/history/EXPOSURE_AUDIT.md).

`PC-DEF-020` is **RESOLVED**. The dashboard's public/loopback decision had two
persisted authorities and no rule for which won: the Android host passed
`-public` only when Public Mode was on, so "off" arrived as silence and fell
through to `launcher-config.json`'s `public` field, which the console's own
Config page can set to true. The host now always states the decision —
`-public=true` or `-public=false` — and Go's `flag.Visit` reports an explicit
false as supplied, so the persisted field is never consulted on Android. The
resolution rule itself is unchanged; what changed is that the fallback branch is
now unreachable there rather than merely discouraged.

The Config page was the second half, and is fixed by what the API reports rather
than by a UI change: where the host owns the decision the page returns the
effective mode and a save persists that instead of the submitted value, which
also repairs a file that had already drifted. `effectiveLauncherPublic` already
encoded the precedence and had no product caller; it was wired up rather than
duplicated, and extended to prefer a runtime rebind over the startup flag. The
frontend is untouched.

Enforcement was never the failure — the password wall held in every state and
the Core gateway on 18790 was loopback-only throughout — so nothing in the
authentication or transport boundary was changed. The required state matrix is
covered by three new test files, including the end-to-end case that a stale
stored `true` with an explicit off opens loopback sockets and nothing else, and
the gateway staying loopback for every gateway host value.

The fix touched `core/src`, so the source fingerprint moved from `bd4a8629…` to
`2692de41b2fe2487475911b62cec519193d581b25cf6d0ebe935fc63973229df` and the pair
was rebuilt and re-staged under the two-commit rule from build-input commit
`f8bc52a`:

    libpocketclaw.so       37,724,640  602ce034…  build ID ed130bed…
    libpocketclaw-web.so   25,517,952  b5cce071…  build ID f61a369f…
    BuildTime              2026-09-12T18:54:26+0000

Byte-identical in three independent output roots, one with a cold Go cache;
both private companions likewise. Native contract 22 PASS / 0 FAIL for the pair;
no Managed Runtime payload rebuilt. Evidence is in
[`docs/prompts/history/PC-DEF-020_PUBLIC_MODE_AUTHORITY.md`](docs/prompts/history/PC-DEF-020_PUBLIC_MODE_AUTHORITY.md).

`PC-DEF-021` is **RESOLVED**, and the framing changed in the process. The defect
was never that AGP writes the R8 mapping and native debug symbols into a
bundle's `BUNDLE-METADATA/` — Google Play consumes those to symbolicate crashes
and never delivers them to an installed client, so deleting them would remove
Play's ability to read a stack trace and fix nothing. The defect was that
PocketClaw had no way to say what an artifact was **for**, and so published AABs
while treating that same material as private, wrote a policy requiring it to
stay "outside APK/AAB files" that AGP cannot satisfy for a bundle, and shipped an
`artifact.r8_mapping_private` check that was a hardcoded `True`.

Purpose is now declared. `tool/artifact_policy.py` defines `public-release`,
`play-upload` and `non-publish-audit`; `--artifact-class` is required for every
artifact phase and has no default, so an unclassified artifact fails closed
rather than being assumed publishable. **An AAB may never be `public-release`**,
and the refusal does not depend on contents — a bundle with no metadata at all
is still forbidden, because what makes the format unpublishable is what it is
for. Detection reads the archive rather than the extension, so renaming a bundle
to `.apk` does not launder it. For `play-upload` the metadata is expected and
the gate names it by entry, size and category instead of passing over it in
silence. Unrelated private material — keystores, `.env`, the private support
tree, `.debug`/`.dwarf`, `.symbols`, signing helpers — still fails in every
class, and the metadata exemption covers exactly two known AGP entry shapes
after this milestone's own tests proved a directory-wide exemption would have
let a keystore through.

`artifact.r8_mapping_private` now reads the artifact and cites what it scanned,
and it runs standalone before the R8 contract — which raises early, so the named
check had been unreachable in the one case it existed for. Demonstrated on two
real APKs: the fresh one reports `496 archive entries scanned, deobfuscation
entries = 0`, and the same code fails the same APK once a mapping is injected. A
public release asset allowlist, checkable via `--release-assets`, forbids
`*.aab`, mapping and usage reports, native companions, `.symbols`, symbol and
private-support archives, keystores and `.env`, while leaving the APK,
checksums, notices, licences and source archives permitted.

`tool/build_hardened_android.py --package bundle` is now the repository-owned
hardened bundle path, running `:app:bundleRelease` through the same hardening
contract as the APK and sharing the Dart verification helpers rather than
duplicating them. Both artifacts built LOCAL TEST from this tree and carry the
PC-DEF-020 Core pair: APK 63,560,039 bytes `7155de0a…`, audit AAB 74,028,994
bytes `00bde2c9…`. Against that real bundle, `public-release` fails (exit 1),
`play-upload` passes and reports 1 mapping plus 5 debug-symbol entries totalling
39,584,662 bytes as allowed, `non-publish-audit` passes with the NOT
PLAY-READY / NOT PUBLIC-RELEASE-SAFE / NOT A GITHUB RELEASE ASSET notice, and an
unclassified run is refused.

**`PocketClaw-v0.2.0-rc1.aab` and `-rc2.aab` were deliberately left in place.**
This milestone had no authority to mutate published releases, so nothing was
deleted and no history was rewritten. They predate Dart obfuscation and R8, so
what they disclose is not the current mapping, but they are the practice this
policy retires. The exact owner action for removal, and what it does and does
not achieve, is recorded in `docs/RELEASE_PROCESS.md`. Full evidence is in
[`docs/prompts/history/PC-DEF-021_AAB_RELEASE_POLICY.md`](docs/prompts/history/PC-DEF-021_AAB_RELEASE_POLICY.md).

`PC-DEF-025` is **RESOLVED**, and it had two halves. The stale assertion in
`namespace_n3_native_identity_test.dart` pinned a *path* —
`build/picoclaw-android-arm64` — when the contract is a *rename boundary*:
upstream emits two artifacts under its own names and PocketClaw's identity is
applied at the install step. H5B made the output root overridable for
reproducibility runs, the literal stopped existing, and the test went red while
the guarantee was intact. It now asserts the install pairing
(`picoclaw-android-arm64` → `libpocketclaw.so`, `picoclaw-launcher-android-arm64`
→ `libpocketclaw-web.so`) through `$CORE_OUTPUT_ROOT`, asserts the
private-support step consumes the same two names, and asserts the *absence* of a
hard-coded `build/` root — strictly stronger than what it replaced, and
mutation-tested both ways against the real script.

The second half is why it mattered. The release gate ran three named Flutter
files, so twenty-five others were outside it entirely and the suite stayed red
through five milestones with the gate reporting green. `flutter.suite` now runs
the **complete** suite and is the acceptance criterion, resolved through a
deterministic `find_flutter()` that prefers the repository toolchain and puts
`PATH` last. The suite runs once through the JSON reporter and the three named
contract items are derived from that run rather than being the whole of it. A
non-zero exit can never be reported as PASS, and exit 0 with no parsed results
fails — a suite that did not run must not look like one that passed.

Proven for real: a deliberately failing test in a file none of the named
contracts covers made the gate exit 1 and name it, while those three stayed
green. `flutter analyze` is clean and `flutter test` is **490 passed, 0 failed**.
`tool/test_release_gate.py` grew from 24 to 35 tests.

The `PC-DEF-021` 55-vs-56 reporting difference was reconciled rather than
propagated: re-running the gate at `3e3941f` gives **56 PASS / 0 FAIL / 0
SKIPPED** on a clean tree and 55 PASS / 1 SKIPPED on a dirty one — the same 56
items, with `repo.clean_worktree` flipping status. No gate-execution
inconsistency and no defect. This milestone adds one item: source 25 → 26,
full APK 56 → 57. Evidence is in
[`docs/prompts/history/PC-DEF-025_FLUTTER_SUITE_GATE.md`](docs/prompts/history/PC-DEF-025_FLUTTER_SUITE_GATE.md).

**The Final Release Exposure Audit is CLOSED / PASS.** The re-run at `6031898`
found **no remaining release blocker** for the intended GitHub / direct APK
stable release, and opened no new defect.

All three defects the first run produced were re-validated from the current tree
rather than taken on trust. `PC-DEF-020`: the Android host still states the
Public Mode decision unconditionally, and a live bind probe re-confirmed
loopback-only when off, wildcard when on, host override winning, and the Core
gateway on 18790 loopback-only in **both** states; 94 matching tests and all
seven network/auth packages green, with the unauthenticated surface unchanged.
`PC-DEF-021`: the bundle classification matrix behaves exactly as specified —
`public-release` exit 1, `play-upload` exit 0 with its metadata inventoried,
`non-publish-audit` exit 0 with the notice, unclassified exit 2 — and the asset
allowlist rejects `*.aab`. `PC-DEF-025`: `flutter analyze` clean and
`flutter test` 490 passed / 0 failed, with the gate reporting the full suite.

Fresh artifacts, both development-signed: APK 63,560,039 bytes `113a8382…`
(LOCAL TEST / NON-RELEASABLE) and audit AAB 74,028,994 bytes `d45efcbf…`
(NON-PUBLISH). Both carry the current Core pair `602ce034…` / `b5cce071…` and
the H5B Managed Runtime byte for byte. Native audit **194 PASS / 0 FAIL / 0
SKIP** with the private support manifest rebound to the fresh APK; source gate
**26 PASS / 0 FAIL / 0 SKIPPED** in both classes; full artifact gate **57 PASS /
0 FAIL / 0 SKIPPED**; frontend 418 tests; Core 98 packages; 139 release-tool
tests. Secrets and entropy scans reproduced the first run's results with no new
candidates, and no packaged entry carries a developer path.

`PC-DEF-022`, `PC-DEF-023` and `PC-DEF-024` keep their classifications, each
re-confirmed against current source and the fresh manifest, and none blocks the
GitHub APK path. `PC-DEF-006` stays open for the F-Droid reproducibility path —
not down-ranked, simply a different distribution path — and `PC-DEF-012` stays
open with no new evidence.

One thing the audit deliberately does not claim: the candidate inspected is
development-signed. H5C proved production and development builds of one tree
differ only by the signing block, so the findings transfer, but the
production-signed candidate has not been built or gated under
`--release-class production`. That is the next milestone. Evidence is in
[`docs/prompts/history/EXPOSURE_AUDIT_RERUN.md`](docs/prompts/history/EXPOSURE_AUDIT_RERUN.md).

`PC-DEF-022` is **RESOLVED by removal**. `POST /api/update` took a
caller-supplied URL, downloaded it, extracted the archive and handed the result
to `selfupdate.Apply`. It required a dashboard session, no PocketClaw UI ever
called it, and on Android the apply step could not succeed against the read-only
install directory — but it remained an authenticated arbitrary-URL fetch and
archive-extraction surface on a route the product does not use. Securing an
unused self-update subsystem would have been the wrong repair.

`api/update.go` is deleted and `router.go` no longer registers it. No special
response was invented: `embed.go` already answers an unknown `/api/` path with
`http.NotFound`, so the route is now an ordinary 404. **`pkg/updater` stays** —
`cmd/picoclaw` registers its CLI update command, a legitimate non-HTTP consumer,
and the library's archive-traversal guards and tests are untouched.

The binaries confirm it independently: `/api/update` occurs zero times in either
Core binary, and `libpocketclaw-web.so` shrank by 132,864 bytes as the linker
dropped the unreachable paths. Tests pin the removal across five HTTP methods,
an authenticated request past the auth wall, six plausible renames, the
unauthenticated allowlist and the handler file's absence — and are
mutation-tested by restoring the route.

The fix touched `core/src`, so the fingerprint moved from `2692de41…` to
`6f00359dc9e8bf7ee24f9d170754b2792a41fb880d9da4f34a8600dd8f99df00` and the pair
was rebuilt and re-staged under the two-commit rule from build-input commit
`a0be2a7`:

    libpocketclaw.so       37,724,640  7ebeebd1…  build ID 512ed36a…
    libpocketclaw-web.so   25,385,088  b682b76d…  build ID 4eeb1385…
    BuildTime              2026-09-12T23:25:34+0000

Byte-identical in three independent output roots, one with a cold Go cache;
both companions likewise. Native contract 22 PASS / 0 FAIL; no Managed Runtime
payload rebuilt. Public Mode, the session wall, the WebSocket boundary and the
gateway's loopback pin are unchanged. Evidence is in
[`docs/prompts/history/PC-DEF-022_UPDATE_SURFACE_REMOVAL.md`](docs/prompts/history/PC-DEF-022_UPDATE_SURFACE_REMOVAL.md).

`PC-DEF-024` is **RESOLVED by removal**. `MainActivity` carried a second
`VIEW` + `DEFAULT` + `BROWSABLE` intent filter whose scheme came from a manifest
placeholder; H1.5 removed the analytics SDK from the shipping build, so the
placeholder resolved to the literal `um.placeholder` and the release manifest
advertised a web-reachable entry point into an exported activity for an SDK that
is not in the APK. `MainActivity` then logged the incoming URI from it.

Removal rather than a conditional, because the repository had already decided
that shape for the same integration: the manifest's advertising-permission
comment records that an analytics build gets what the SDK's own AAR manifest
declares, and that an app-level declaration it needs belongs to that build's
manifest. The filter, the Gradle link-scheme plumbing and the
`logIncomingIntent` branch are gone, along with the `TAG`, `Log` and `Bundle`
symbols that existed only for them. `setIntent(intent)` stays — FlutterActivity
and plugins read `getIntent()`, and removing analytics logging must not remove
real intent handling. `POCKETCLAW_UMENG_APP_KEY`, `_CHANNEL` and `_PACKAGED` are
kept: `AnalyticsReporter` and two `meta-data` entries consume them, and they are
not an exported surface.

A fresh LOCAL TEST APK (`f580cadc…`, 63,500,459 bytes) confirms it in the
packaged merged manifest: zero `um.placeholder`, zero `BROWSABLE`, zero
`android:scheme`, zero `action.VIEW`, with `category.LAUNCHER` and
`.MainActivity` still present and `debuggable`/`testOnly` still absent. Seven
new tests guard it and are mutation-tested; they strip XML comments and assert
declarations, which is what caught that Gradle carries comments through its
merge while `aapt2` strips them.

`flutter test` is **497 passed, 0 failed** (up from 490 by the seven new tests),
`flutter analyze` clean. No Core build input was touched: the fingerprint stays
`6f00359dc9e8bf7ee24f9d170754b2792a41fb880d9da4f34a8600dd8f99df00` and Core was
not rebuilt. Evidence is in
[`docs/prompts/history/PC-DEF-024_DEAD_DEEP_LINK_REMOVAL.md`](docs/prompts/history/PC-DEF-024_DEAD_DEEP_LINK_REMOVAL.md).

`PC-DEF-023` is **RESOLVED: Google Antigravity is not shipped in v0.2.0.** This
was an owner product decision, recorded as `PC-D014`.

The classification from the exposure audit stands — it is **not** a secret
disclosure. An installed-app OAuth client cannot keep a secret, so nothing that
was ever protected was published and no PocketClaw or user credential was
involved. What blocks shipping is **ownership**: the client ID and secret
belonged to another project, and a third party can revoke them at any time,
breaking the provider for every user for a reason PocketClaw could neither
predict nor fix.

The provider is removed from every surface a user or caller can reach — the
OAuth API, the provider catalogue, the factory, the default `model_list`, the
legacy-import mapping, the CLI (including the `auth models` subcommand that
existed only to list its models), the dashboard credential card, all fourteen
locale bundles, and the embedded agent guidance. `antigravity` and
`google-antigravity` now return the ordinary unsupported-provider error, and the
provider is absent from the catalogue rather than hidden behind a removed UI.

Shared OAuth infrastructure is kept: `ClientSecret` and the confidential-client
token exchange, and `canonicalProvider`'s trim/case normalisation. **Gemini is
untouched** and is a different provider entirely — its own catalogue entry,
API-key auth and `generativelanguage.googleapis.com` base — with a test pinning
it and its `google` alias. OpenAI OAuth, the Anthropic token flow, PKCE, state
and session handling are unchanged.

Both staged binaries carry zero occurrences of every Antigravity and credential
marker, while Gemini's endpoint and display name remain. The first rebuild was
not clean: three strings survived in the embedded agent skill document, which
still advertised the provider. Checking the binary rather than trusting the
source diff is what caught it; it was corrected and the pair rebuilt.

Fingerprint moved from `6f00359d…` to
`bc35a598d3a836e0a0c95afc73314fe49a38877b985b5b0f15bab11460184fa9`, rebuilt and
re-staged under the two-commit rule from build-input commit `54ff252`:

    libpocketclaw.so       37,658,976  0a28bd5e…  build ID c657e80d…
    libpocketclaw-web.so   25,319,424  9ae1d2d9…  build ID 45355d87…
    BuildTime              2026-09-13T00:24:56+0000

Both shrank by exactly 65,664 bytes as the provider left the binaries.
Byte-identical in three independent roots, one with a cold Go cache; native
contract 22 PASS / 0 FAIL; no Managed Runtime payload rebuilt. Evidence is in
[`docs/prompts/history/PC-DEF-023_ANTIGRAVITY_REMOVAL.md`](docs/prompts/history/PC-DEF-023_ANTIGRAVITY_REMOVAL.md).

**The final v0.2.0 production-signed candidate exists and passes every gate.**
Built at `1c477e60` by the canonical hardened path through an owner-run
hidden-input signing helper outside the repository; no password reached a
command line, a repository file, a Gradle property or a log.

    path    build/app/outputs/apk/release/app-release.apk
    bytes   63,472,307
    sha256  4d4bc33a63059450383c4eedb34e2902486fbbc8c0d85b91413ab9654b4f3dac
    signer  one v2 signer, 176dca6b198b9552fb4d9ad3ca18da8d6f23c0a3f5ed4bd6b75a0700f9f0efcf
            (the enrolled production certificate; the development signer is absent)

It carries Core generation `bc35a598…` — `libpocketclaw.so` `0a28bd5e…` and
`libpocketclaw-web.so` `9ae1d2d9…`, BuildTime `2026-09-13T00:24:56+0000` — with
all eight Managed Runtime payloads at their pinned checksums, and Dart AOT
`c7b2a885…` plus private DWARF `0f52873b…` byte-identical to H3A onward.

**Full production artifact gate: 57 PASS / 0 FAIL / 0 SKIPPED**, printing
`PASS — production release candidate`. Native audit 194 PASS / 0 FAIL / 0 SKIP
over 18 ELF entries, with the private support manifest rebound to this exact
APK. All thirteen private/residue categories clean, and `um.placeholder`,
`/api/update`, `google-antigravity` and every third-party credential marker
occur zero times. One `antigravity` string remains and is not a finding: it is
CPython's own stdlib module name inside the byte-unchanged `libpocketclaw-python.so`.

**Provenance warning for future readers.** Two superseded literals survive in
commit messages — fingerprint `f9a2d2a8…` in `54ff252` and BuildTime
`00:23:11` in `1c477e6` — both predating an amend and the rebuild that followed.
The authoritative values are `bc35a598…` and `2026-09-13T00:24:56+0000`, which
are what the binaries carry; neither superseded literal appears in any binary or
tracked document. The commits were deliberately not rewritten.

This is **private validation evidence**. It was not installed, published,
accepted or released, and the baseline is unchanged. **Do not install it over
the current device state** — the Samsung is development-signed and a
cross-signer install is forbidden. Evidence is in
[`docs/prompts/history/V0_2_0_PRODUCTION_CANDIDATE.md`](docs/prompts/history/V0_2_0_PRODUCTION_CANDIDATE.md).

### Completed major milestones

- vc62 Zero-Pico namespace closeout: physically accepted and merged.
- Repository polish: closed and merged to `develop`.
- H1 production-signing architecture: closed.
- H1.5 canonical/F-Droid cleanup: closed.
- H1.5D source-built runtime adoption: closed.
- Safety checkpoints: created and unchanged.
- H2 production signer enrollment and private validation: closed.
- PC-1 continuity protocol: represented by this documentation closeout.
- H3A Dart binary hardening architecture and non-releasable validation: closed.
- H3B production-signed Dart-hardening validation: closed.
- H4A R8/ProGuard hardening and non-releasable validation: closed.
- H4B production-signed R8/ProGuard validation: closed.
- H5A native/ELF audit and private-symbol policy: closed.
- H5B targeted native hardening and private native-symbol archive: closed,
  with owner Samsung physical native smoke PASS.
- H5C production-signed native/ELF validation: closed.
- UI-1 guided tour hardening: closed (console defect repair, not a release milestone).
- PC-DEF-019 Core rebuild and re-stage: resolved (artifact prerequisite, not a release milestone).
- Final release exposure audit: run and BLOCKED on `PC-DEF-020` and `PC-DEF-021`; not closed.
- PC-DEF-020 Public Mode authority fix: resolved (release blocker cleared; not a release milestone).
- PC-DEF-021 AAB privacy and release-artifact policy: resolved (second release blocker cleared).
- PC-DEF-025 Flutter suite and release-gate integrity: resolved (the full suite is now a gate).
- Final release exposure audit: **CLOSED / PASS** on re-run; no release blocker remains.
- PC-DEF-022 update-surface removal: resolved (the unused self-update route is gone).
- PC-DEF-024 dead analytics deep-link removal: resolved (the dead BROWSABLE surface is gone).
- PC-DEF-023 third-party OAuth dependency removal: resolved (Google Antigravity is not shipped in v0.2.0).
- Final v0.2.0 production-signed candidate: built, inspected and gated 57/0/0; not installed or released.

See [`docs/ROADMAP.md`](docs/ROADMAP.md) for the phase sequence and
[`docs/AI_HANDOFF.md`](docs/AI_HANDOFF.md) for the mandatory read order.

### Pending hardening and deferred work

- Production-signer physical transition: a separately authorized migration,
  clean-install and data-safeguard milestone. The installed device path is
  development-signed, so a cross-signer in-place upgrade is forbidden.
- Secrets/configuration plus APK/AAB exposure audit.
- Full APK/AAB production inspection and production-class gate.
- APK-level reproducibility for the target F-Droid path.
- External-view reverse-engineering audit and final Samsung physical smoke.
- Deferred non-release work is tracked in
  [`docs/DEFECT_LOG.md`](docs/DEFECT_LOG.md).

### Rollback and checkpoint refs

| Ref | Commit/object | State |
| --- | --- | --- |
| `develop` | `47cde00cecb44cb672fa1ae5d8633a3283a1e830` | Unchanged by H1/H2/PC-1 |
| `main` | `100a51de6a88a485ee109406ca96172285de4c4e` | Unchanged |
| `checkpoint/vc62-accepted` | `47cde00cecb44cb672fa1ae5d8633a3283a1e830` | Accepted-source rollback branch |
| `checkpoint/pre-h2` | `fb38c7d3c3fd31420c45a1977e1c1425ede3a378` | Pre-enrollment hardening checkpoint |
| annotated tag `checkpoint-vc62-accepted` | object `fafb19a88511f8dcfa96a628a89d87a12324537f`; target `47cde00cecb44cb672fa1ae5d8633a3283a1e830` | Unchanged, not a release tag |

### Tags and GitHub prereleases

Current annotated tags and peeled targets:

| Tag | Tag object | Target commit | Meaning |
| --- | --- | --- | --- |
| `phase2-milestone-b` | `d9125659b0e889fe77dc6f79d3b30cb115e50f18` | `225be3c31a025131d5c130cd337b47dc786f530e` | Historical milestone |
| `phase2-milestone-c` | `e3f8cf751f21df5efd13f384e90d4b2246b0e925` | `36bc88de299d61064f5119c65aa208b6ba60609d` | Historical milestone |
| `phase2-milestone-d` | `f6731dafe9da780d843328f282a3d99afbc06b17` | `8f861bca1c82b43b306e95b14e277269260bbab0` | Historical milestone |
| `pre-codex-sol-prerelease-fix-20260826` | `6024dadea5d86d003885b564136022848558a0d3` | `e5b88ff1a4c8f76e321c07af97eff5ca23d59d78` | Historical checkpoint |
| `v0.2.0-rc1` | `8959e579deb9a3191a0d6ed307163041f60caf45` | `e470bb62cce924f17c9f6d46705d513c2765c2aa` | Published prerelease tag |
| `v0.2.0-rc2` | `a523562c2488bda5fd47d15757b01799069570fc` | `404ef44af5a58fe306690828fac617c07e89ef85` | Published prerelease tag |
| `v0.2.0-rc3` | `eb3a8381c4a682265b52423954df5963227ad76c` | `2312f355a33ea3f9d975fa6ad59c6c25a55064a1` | Published prerelease tag |
| `checkpoint-vc62-accepted` | `fafb19a88511f8dcfa96a628a89d87a12324537f` | `47cde00cecb44cb672fa1ae5d8633a3283a1e830` | Source checkpoint, not a release |

GitHub currently has exactly three releases, all prereleases:

| GitHub name | Tag | Published (UTC) | Current metadata update (UTC) |
| --- | --- | --- | --- |
| PocketClaw v0.2.0-rc1 | `v0.2.0-rc1` | 2026-08-28 23:29:50 | 2026-08-28 23:29:50 |
| PocketClaw v0.2.0-rc2 | `v0.2.0-rc2` | 2026-08-30 01:25:43 | 2026-08-30 01:25:43 |
| PocketClaw v0.2.0-rc3 | `v0.2.0-rc3` | 2026-09-05 17:42:14 | 2026-09-05 17:42:14 |

The final-hardening branch begins from `47cde00` and its first H1 commit is dated
2026-09-10. All three prereleases and their annotated tags therefore predate
H1/H2. GitHub reports no metadata update after each original publication, and
the H1/H2 commit range contains no tag or release operation. The supported
conclusion is that H1/H2 did not modify them.

Older entries saying “no release” describe what a particular milestone did, or
the repository state before RC publication. They are historical facts, not the
current release inventory.

### Exact next action

The exposure audit is closed and no release blocker remains. The next milestone
is the one that needs the owner:

1. **Production-signed release candidate.** The owner signing ceremony, then
   `tool/build_hardened_android.py --signing production` and
   `tool/release_gate.py --full <apk> --release-class production
   --artifact-class public-release`, then the final Samsung physical smoke under
   its own device authorization. Note the installed device path is
   development-signed, so the production transition needs its migration /
   clean-install / data-safeguard plan.

The production candidate is built and gated, so the remaining work is physical
and then editorial:

1. **Samsung physical acceptance**, under its own prompt. The device carries a
   development-signed lineage and the candidate carries the production
   certificate, so a cross-signer `adb install -r` is forbidden and would fail.
   That milestone needs a migration / clean-install / data-safeguard plan before
   any device action.
2. Only after physical acceptance: advance the baseline in the commit that
   records it, then tag and publish under explicit owner authorization, with the
   APK-only asset set the allowlist already validates.

`PC-DEF-006` gates the F-Droid reproducibility path only; `PC-DEF-012` awaits
reachability evidence. Neither blocks the GitHub / direct APK release.

Prepare and review an explicit final release exposure audit prompt —
secrets/configuration plus full APK and AAB inspection — from
[`docs/prompts/PROMPT_TEMPLATE.md`](docs/prompts/PROMPT_TEMPLATE.md) against
[`docs/prompts/REVIEW_PROTOCOL.md`](docs/prompts/REVIEW_PROTOCOL.md), and wait
for owner authorization. Do not begin it from this closeout.

Two items remain open independently of that audit: APK-level reproducibility for
the F-Droid path, and the versioned non-destructive bootstrap update strategy.
`PC-DEF-012` also stays open, by decision, until dependency-specific
reachability evidence exists.

# Historical milestone archive

## Consolidated physical-defect hardening PD2 — SOURCE COMPLETE, PHYSICALLY UNVERIFIED

- **Date:** 2026-09-13, continuing PD1 on new owner evidence.
- **Branch:** `feature/final-release-hardening`. Not merged, not pushed, not
  tagged, not published.

### What changed the picture

PD1 recorded PC-DEF-032 as a model configured against the wrong OpenCode
endpoint. The owner then proved, directly against the live service, that the
cause is different and that the service is healthy:

| `POST https://opencode.ai/zen/go/v1/chat/completions`, `deepseek-v4.1-flash` | Result |
| --- | --- |
| no `x-opencode-session` | HTTP 400 `MissingSessionID` |
| `x-opencode-session: <uuid>` + `User-Agent: PocketClaw/0.2.0` | HTTP 200, `OK` |

Service, key, model and endpoint are PROVEN WORKING. PocketClaw sent no session
header at all. The endpoint-inventory finding from PD1 remains true and remains
guarded, but it was not this failure. The correction is recorded in the defect
log rather than overwritten.

### Defects

| ID | State |
| --- | --- |
| PC-DEF-032 | **OPEN.** Fixed in source; not closed until the Samsung produces a real successful OpenCode Go answer. |
| PC-DEF-045 | Save/Update unreachable in the real flow — fixed in source, physical confirmation required. |
| PC-DEF-046 | **UNKNOWN — INVESTIGATION REQUIRED.** Android-client-specific or device-specific. Audited, not patched. |
| PC-DEF-047 | No obvious Delete/Remove action — fixed in source, physical confirmation required. |
| PC-DEF-048 | Hardened build skipped Flutter AOT — fixed in source. |

PC-DEF-046 is deliberately not attributed. The Core log does not explain it and
the console is fast in a desktop browser, which narrows it to the Android client
or the device without deciding between them. The defect log records the
discriminator that does decide it — the same phone's browser against the same
phone's app — and separates the audit's proven code properties from their
unproven causal role.

### Artifact identity

| Field | Value |
| --- | --- |
| Verification APK | `234d1d69ea63c505ad47fba931ab12b86c66abf301293d2cc664cb854f5440f1` |
| Size | 63,501,635 bytes |
| Version / package | `0.2.0+62` · `com.lord1egypt.pocketclaw` |
| Signer | development — LOCAL TEST / NON-RELEASABLE |
| Core source fingerprint | `5933e74f9e94e8efa631e24369b767547e3b1c589402b7c312e9b2e8edc56313` (both binaries) |
| Core BuildTime | `2026-09-13T15:53:52+0000` (build-input commit `7b6c43c`) |
| `libpocketclaw.so` | `5111ab8fa3add72011786aec131cbe1d8f5d7c365b2a24965508e26a5dea2c3b` |
| `libpocketclaw-web.so` | `f7f015b250c0a4e8142b957c97309930504799d45d35dbe9a18238e9f2f48de1` |
| Dart AOT | `f5d84c9477617c6f4253819e6fcc5bc8c0f42a5c7977c13bb62ea59c5067e5a5` |
| Dart symbols | `e04449135e470d4d8a279354f696fa7022cbb716d5495daaea56e25fd77fbee2` |

Present in the packaged artifact, verified by direct inspection rather than
inferred from the build succeeding: `x-opencode-session`, `PocketClaw/0.2.0`,
the pending-apply supervisor, the provider error detail, the model provenance
labels, `com.lord1egypt.pocketclaw.action.RESTART` in the Dex, and the official
onboarding endpoint.

### Gate counts

| Gate | Result |
| --- | --- |
| Artifact gate | 33 PASS / 0 FAIL / 0 SKIPPED |
| Native ELF audit (`--enforce-target`) | 188 PASS / 0 FAIL / 0 SKIP |
| Source gate | PASS |
| Zero-Pico | PASS, no new occurrences |
| Core staleness (`pkg/coresource`) | PASS |
| Core Go suite (`-tags goolm ./...`) | PASS |
| Frontend | 32 files / 452 tests PASS; `tsc` and `eslint` clean |
| Flutter | 530 tests PASS; `flutter analyze` clean |

PC-DEF-048's fix was exercised rather than assumed: this artifact was built
**without** `--clean`, under the no-Dart-change condition that reproduced the
defect, and the private symbol file was produced.

The Core was rebuilt once more after the staged pair was already correct.
Untracking a state file the test suite had written changed a path under
`core/src`, and `BUILD_INPUTS` covers `core/src` whole — deliberate
over-inclusion, per `resolve-build-time.sh`: "an extra commit moves the
timestamp slightly more often than strictly necessary, whereas a missing path
means a real build-input change that does not move it at all." The fingerprint
was unchanged at `5933e74f…` throughout, because it excludes tests and stray
JSON by name; only the timestamp moved, and rebuilding was the right answer
rather than narrowing a provenance rule.

### Not physically verified

No Android device reached this session: `adb devices` was empty throughout,
`/dev/bus/usb` does not exist under this WSL2 kernel, and Windows interop is
unavailable in this shell, so neither usbipd attach nor a host-side adb could be
driven. Every user-visible status in this milestone reads FIXED IN SOURCE.

## Consolidated physical-defect hardening PD1 — SOURCE COMPLETE, PHYSICALLY UNVERIFIED

- **Date:** 2026-09-13.
- **Branch:** `feature/final-release-hardening`. Not merged, not pushed, not
  tagged, not published.
- **Basis:** owner Samsung physical testing of verification APK
  `b6d6e6f9bd6316e24308a63265dcb1e4c15ef7bd1b7e04ba921a5d6b0a1e63b7`.
- **Commits:** `361c89b` (source) and `56ca091` (staged Core pair), plus this
  documentation commit.

### Artifact identity

| Field | Value |
| --- | --- |
| Verification APK | `dd792aabc23482b49312dcdb2393213e0934d60a465b4311e5fac5b512e41cea` |
| Size | 63,495,923 bytes |
| Path | `build/app/outputs/apk/release/app-release.apk` (ignored; not committed) |
| Version | `0.2.0+62`, package `com.lord1egypt.pocketclaw` |
| Signer | development — LOCAL TEST / NON-RELEASABLE |
| Core source fingerprint | `1a40356e985d333be014f82988376931dfb176cf50a76e5abbb510cd65e268fc` (both binaries) |
| Core BuildTime | `2026-09-13T09:14:52+0000` (from build-input commit `361c89b`) |
| `libpocketclaw.so` | 37,659,104 bytes, `6593229b3707377f2c0aab5bcb4a31ac26b41875ed265e034f55539800bded02` |
| `libpocketclaw-web.so` | 25,385,088 bytes, `7122d7f0b94f72d96694c4a769c32361921869d4a5b414924f54e9c31c23156e` |
| Dart AOT | `f5d84c9477617c6f4253819e6fcc5bc8c0f42a5c7977c13bb62ea59c5067e5a5` |
| Onboarding endpoint | `https://pocketclaw-telegram-setup-bot-83ai.vercel.app` (PC-DEF-026 contract intact) |

### Gate counts

| Gate | Result |
| --- | --- |
| Source gate (`--verify-source --release-class test`) | PASS; `repo.clean_worktree` SKIPPED while the tree carried the milestone changes |
| Artifact gate (`--verify-artifact ... --artifact-class non-publish-audit`) | 32 PASS / 0 FAIL / 0 SKIPPED |
| Native ELF audit (`--enforce-target`) | 188 PASS / 0 FAIL / 0 SKIP |
| Zero-Pico | PASS — 19 allowlist entries, all in use, no new occurrences |
| Core staleness (`pkg/coresource`) | PASS after the rebuild |
| Core Go suite (`-tags goolm ./...`) | PASS |
| Frontend (`vitest`) | 31 files / 448 tests PASS |
| Frontend (`tsc -b`, `eslint .`) | clean |
| Flutter (`flutter test`, `flutter analyze`) | 530 tests PASS; no analyzer issues |
| Python tool suites | PASS (`test_create_release_keystore.py` needs the toolchain JDK on PATH) |

### Defects closed in source

PC-DEF-030, PC-DEF-032, PC-DEF-033, PC-DEF-041, PC-DEF-042, and the new
PC-DEF-043 (model removal left references and failed silently) and PC-DEF-044
(a replaced Telegram token skipped the owner contract). Root causes, evidence
and verification are in `docs/DEFECT_LOG.md`. The audited provider matrix is
`docs/PROVIDER_COMPATIBILITY.md`.

PC-DEF-031 (Telegram remove/disconnect/reconnect/replace) was audited, not
rewritten: the existing implementation stands, and the one repair it needed was
PC-DEF-044.

### What is NOT verified

No Android device was attached to this session (`adb devices` empty), so **no
user-visible defect in this milestone is physically verified**. Every status in
the defect log reads FIXED IN SOURCE. The artifact is ARTIFACT VERIFIED by the
gates above and by direct string inspection of the packaged Core, Dart and Dex.
Phase D remains entirely outstanding.

### Forensic artifact note

The build wrote to the repository's standard `build/app/outputs` path and
therefore replaced the `b6d6e6f9` verification APK in place, which is how every
previous milestone's artifact was also handled. Its two Core binaries were
extracted before the build and preserved outside the repository at
`/home/lordegypt/pocketclaw-forensics/b6d6e6f9/`
(`libpocketclaw.so` `ee7c3d75a9376cb16a25c8414b67489051fa96baa350b443f1f9006c4a25f36e`,
`libpocketclaw-web.so` `c1c2f5bb39d80989d623556f9cb76b6feb6617d25879babbfe80d247a7e0bad5`),
together with a copy of the new APK. The full `b6d6e6f9` APK itself is gone and
would have to be rebuilt from `e65ecc2`, which APK-level reproducibility does
not yet guarantee byte-for-byte.

## Final Production Release Hardening H4B — CLOSED / PRIVATE VALIDATION

H4B started from `a6034c065becccc0a01ed7e734dad6b2558a0ef1` and validated
the exact H4A R8/ProGuard contract under the enrolled production signer. The
APK, signer, DEX, mapping, Dart, Core, runtime, test, and gate evidence is
recorded in the current snapshot and the H4B reconstructed operating record.

The first owner run exposed a temporary-helper routing defect: its signing
validation invoked the Android wrapper while leaving Gradle's project root at
the repository root. The corrected external helper uses `-p android` for every
Gradle invocation and exercises that exact validation route before prompting
for secrets. The corrected owner run completed successfully. The helper trap
cleared all four signing variables, and no secret was printed, stored, or
committed.

The first passing artifact manifest still labeled H4B as pending. That stale
status was removed as `PC-DEF-R013` with a regression test while the remaining
open F-Droid reproducibility and bootstrap-strategy items stayed intact. The
metadata-only correction did not require an APK rebuild.

This APK is private validation evidence. It was not installed, published,
accepted, or committed. The accepted physical baseline remains vc62 / 62.
`PC-DEF-006` remains open, `PC-DEF-008` remains deferred, and native hardening
has not started.

## Final Production Release Hardening H4A — CLOSED / NON-RELEASABLE

H4A started from `30ec1951cb91df2d3ab80ce09d3e1176611242f7` and removed
the project-owned blanket R8 keeps after the merged configuration proved the
framework, manifest, and dependency rules independently retain their actual
entry points. The canonical helper now invalidates stale R8 reports and requires
fresh private mapping/shrinking evidence plus artifact-level proof of real
renaming or removal. Exact evidence is recorded in the current snapshot and
the H4A reconstructed operating record.

This artifact uses the explicit local development signer and is LOCAL TEST /
NON-RELEASABLE. It was not installed, published, accepted, or committed. The
production key was not accessed. `PC-DEF-008` remains deferred to native/symbol
hardening, which H4A did not start. H4B production validation has not started.

## Final Production Release Hardening H3B — CLOSED / PRIVATE VALIDATION

H3B started from `6491ccc6f611c7513506d622dbb1ad4a75c93a43` and validated
the H3A Dart-hardening contract under the enrolled production signer. The exact
APK, AOT, private-DWARF, signer, and gate evidence is recorded in the current
snapshot and the H3B reconstructed operating record.

The first owner run exposed a directly related cache defect: Flutter 3.47.1
could reuse cached AOT after the helper deleted external split debug info,
because that external file is not a tracked cache output. The canonical helper
now clears only `.dart_tool/flutter_build` before assembly so AOT and DWARF are
regenerated together. The corrected owner helper also completes Java and other
non-secret prerequisite checks before requesting hidden signing input. Both
defects were re-tested and resolved during H3B.

This APK is private validation evidence. It was not installed, published,
accepted, or committed. The accepted physical baseline remains vc62 / 62.
R8/ProGuard and native hardening have not started.

Everything below this heading is phase-scoped evidence preserved from earlier
closeouts. When an older statement conflicts with the authoritative snapshot,
read it as “true at that milestone,” not as current state. Do not rewrite an
accepted historical record to make it sound current.

## Final Production Release Hardening H3A — CLOSED / NON-RELEASABLE

H3A started from `9a5a5dd9fd4a0697451d27948efe2c5be6e5c028` and established a
fail-closed Gradle/Flutter 3.47.1 Dart-hardening contract: obfuscation, external
split debug info, arm64 product target, and a stable package URI for Flutter's
generated Dart plugin registrant. Two clean equivalent builds using different
symbol-output roots produced byte-identical `libapp.so` and byte-identical
split DWARF. This is scoped Dart evidence, not full-APK reproducibility.

The exact validation APK and hashes are recorded in the current snapshot. It
uses the explicit development signer and is LOCAL TEST / NON-RELEASABLE. The
production signer and owner signing secrets were not accessed. No artifact was
installed, published, accepted, or committed. `artifact.dart_snapshot_paths`
is resolved on real H3A artifact evidence; H3B production validation remains
required. R8/ProGuard and native hardening have not started.

## Final Production Release Hardening H2 — CLOSED

The developer production signing path was privately validated on 2026-09-11 on
`feature/final-release-hardening`. The owner confirmed that a separate backup of
the production keystore exists and supplied both passwords only through hidden
local terminal input. Gradle selected production signing and built
`build/app/outputs/apk/release/app-release.apk` from
`:app:assembleRelease -Ptarget-platform=android-arm64`.

    size       64359287 bytes
    sha256     f0d83298c2ce061c01a9fc931ad29676e4d4b646bb5b204a9bf0002b11a7f46f
    package    com.lord1egypt.pocketclaw
    version    0.2.0 (62)
    product ABI arm64-v8a; plugin stubs armeabi-v7a and x86_64
    signer     176dca6b198b9552fb4d9ad3ca18da8d6f23c0a3f5ed4bd6b75a0700f9f0efcf

Independent `apksigner` inspection found exactly one signer and matched it to
the enrolled production certificate; the development signer was not used. The
production artifact gate passed with **20 PASS, 0 FAIL, 1 SKIP**. The skip is
`artifact.dart_snapshot_paths`, retained for H3 binary hardening rather than
waived or claimed complete. No new defect was found.

This is a private validation artifact. It was not installed, published or
accepted, and no device or emulator was touched. The accepted physical baseline
remains vc62 / `lastAcceptedVersionCode=62`; version remains `0.2.0+62`; Core
fingerprint remains
`876b87f5950452ba903301b6cce4cc96502ab4ff25d90d13e31b9da537e24b44`.
Core and Managed Runtime payloads were unchanged. H2 is complete. H3 has not
started.

## Zero-Pico namespace migration — CLOSED / ACCEPTED

- **Accepted build: PocketClaw `0.2.0+62`.** Accepted baseline **62**, previously
  59. APK `1470e02d43039e78c6507a1fb12e1c9368033903a8b99c2e2f56014eb0c002e2`.
- Merged to `develop` with a true `--no-ff` merge. `main`, tags and releases
  untouched; there is still no `v0.2.0` stable tag, and there should not be one
  until production hardening lands.
- Feature branches `feature/zero-pico-runtime` and
  `feature/namespace-n3-native-binaries` are retained for provenance.

### Candidate history

    vc60   SUPERSEDED during N3 physical validation — a broad native-process
           ownership classification regression
    vc61   N3 native identity physically PASS, but not accepted as baseline:
           the Zero-Pico migration was still in progress
    vc62   final Zero-Pico candidate. Machine migration validation PASS,
           manual user acceptance PASS. ACCEPTED.

### The current namespace contract

    native binaries      libpocketclaw.so, libpocketclaw-web.so
    Core private state   filesDir/pocketclaw-core/
    workspace            the external canonical workspace, with a
                         filesDir/pocketclaw/ fallback
    PID record           .pocketclaw.pid
    environment emitted  POCKETCLAW_*
    managed channel      pocketclaw
    client channel       pocketclaw_client
    owner principal      pocketclaw-user
    realtime routes      the pocketclaw route namespace
    dashboard cookie     pocketclaw_launcher_auth
    notification channel pocketclaw_service
    SharedPreferences    pocketclaw_prefs
    IRC default nick     pocketclaw

### Legacy Pico policy

**ZERO ACTIVE PICO: COMPLETE.** No PocketClaw-owned production surface writes,
issues, defaults to, emits, registers or advertises a Pico-family identity.
What remains is permitted only as upstream identity, legal attribution,
historical evidence, legacy migration (read, parse, normalize, migrate, delete,
expire, redact, or protect from backup), or explicit external compatibility.
`tool/no_active_pico.py` enforces this on every gate run.

`wecomQRSourceID` stays as it is, deliberately: it is sent to
`work.weixin.qq.com` as `source`/`sourceID`, so it is what a third party was
registered to recognise rather than this project's name for itself. The
immutable historical digest
`47b63011a55eaa659470f2ab09d05532e9942800848020a1cfe443e6f21aca76` is unchanged.

### Physical acceptance evidence (vc62, SM-A165F / Android 16)

    package        com.lord1egypt.pocketclaw, versionCode 62
    processes      gypt.pocketclaw · libpocketclaw.s · libpocketclaw-w
    Core config    …/files/pocketclaw-core/config.json, physically in use
    PID record     .pocketclaw.pid — host, pid, port, version only; token-free
                   .picoclaw.pid ABSENT
    Gateway        127.0.0.1:18790 and [::1]:18790, loopback only
    native legacy  libpicoclaw* ABSENT
    Managed Runtime 8 payloads
    notifications  pocketclaw_service ACTIVE; picoclaw_service and
                   picoclaw_foreground both deleted tombstones

User confirmed manually: Running PASS, Dashboard PASS (with the expected
one-time re-login), Web Chat PASS, notification behaviour PASS.

No private config or token contents were read at any point.

### Still open, and deliberately not touched here

Namespace acceptance is not production hardening. The web console binding
`0.0.0.0:18800` remains tracked separately; the production signing key does not
exist yet; the Dart snapshot-path item, R8/ProGuard narrowing, obfuscation and
split debug info are all still outstanding.

### Next milestone

**Final production release hardening** — real production signing, Dart
obfuscation and split debug info, a symbols archive, R8/ProGuard review, a final
secrets and config audit, full APK/AAB inspection, a production-class release
gate, a final stable physical smoke, and only then a tag and release.

## vc62 — Zero-Pico candidate, ACCEPTED

- Status: **accepted on the device, 2026-09-09.** `0.2.0+62`,
  `lastAcceptedVersionCode` advanced **59 → 62** in the same commit that records
  the acceptance, which is what keeps that floor meaningful.
- Installed in place on `RK8Y6016N5V` (Samsung SM-A165F, Android 16, arm64-v8a)
  with `adb install -r` over Windows adb 37.0.1. No uninstall, no clear data, no
  permission reset. The device was absent on the first attempt and attached
  partway through; usbipd was never used.

### Candidate artifact

    path      build/app/outputs/flutter-apk/app-release.apk
    size      63493154 bytes
    sha256    1470e02d43039e78c6507a1fb12e1c9368033903a8b99c2e2f56014eb0c002e2
    package   com.lord1egypt.pocketclaw
    version   0.2.0 (62)
    signer    CN=Android Debug, SHA-256
              15cf75f9945d5354e75707e0326b7cffc60ac51a68df38156db318ef4578a27c
              local-test only; no production signing material exists yet

Built with the repository toolchain (JDK 17, Flutter 3.47.1) via
`:app:assembleRelease -Ptarget-platform=android-arm64 -PallowDebugSigning=true`.
Gradle exit 0 and BUILD SUCCESSFUL were checked independently of each other, and
the APK was confirmed to carry versionCode 62 by inspection rather than by
assuming the bump reached it.

### The Core pair was not rebuilt

Packaged against staged, byte for byte:

    libpocketclaw.so       37749089  9e85e471…  identical to staged
    libpocketclaw-web.so   25559393  479003e0…  identical to staged

Both carry fingerprint `6e2382ae…` and BuildTime `2026-09-09T18:30:34+0000`. No
`libpicoclaw*.so` is packaged and no alias was created. Managed Runtime is
exactly 8 payloads (curl, gh, git, git-remote-http, jq, python, rg, sqlite3);
`libpocketclaw-web.so` is Core, not a runtime tool, and is not counted as one.

### Gates

Production source gate: **exit 0, 17/17 PASS**. Artifact gate (`--release-class
test`): **exit 0**, every check PASS with one SKIP —
`artifact.dart_snapshot_paths`, the established PENDING_FINAL_HARDENING item,
left at its documented status rather than weakened for this candidate.

### A gate defect this phase found

`artifact.core_provenance_pair` failed on the first artifact run, reporting both
binaries as "present (not matched to source)". The binaries were fine; the check
was not. It read `gate.facts["coreSourceFingerprint"]`, which only
`source_gates` ever set, so under `--verify-artifact` the fact was absent and
the comparison had nothing to compare against — the check could not pass in the
one mode where it matters most, inspecting an artifact you did not just build.
Introduced in N4K-A and missed there because that phase only ran
`--verify-source`. Fixed in `84080a5`: one helper resolves the fingerprint on
demand and caches it, so both paths read the same value from the same code.
`tool/` is not packaged and is not a build input, so no rebuild followed.

### What's New

Four bullets in all twelve locales, on the existing mechanism: steadier
reliability, and the three one-time effects of upgrading — one more Dashboard
sign-in, a fresh Web chat conversation, notification preferences worth one
check. No old product name, paths, cookies, routes, environment variables or
security internals; a release note is not a changelog for its authors.

### Install and preservation

    Success (streamed install)
    versionCode      61 -> 62,  versionName 0.2.0 unchanged
    appId            10666                         preserved
    dataDir          /data/user/0/com.lord1egypt.pocketclaw   preserved
    firstInstallTime 2026-08-26 05:21:06           preserved
    lastUpdateTime   04:04:31 -> 22:10:03          changed, as expected
    permissions      11 requested, identical set
    installed base.apk sha256 == the candidate's, 1470e02d…

Installed `nativeLibraryDir` carries both Core binaries and exactly the 8
Managed Runtime payloads. No `libpicoclaw*.so`.

### The migration ran, and it was observed rather than provoked

Package replacement started the application on its own — the app process and
both Core binaries (`libpocketclaw.so`, `libpocketclaw-web.so`) were running
afterwards. Nothing was launched over adb, no UI was touched, no service was
started by hand.

Notification channels, read straight from `dumpsys`:

    picoclaw_service       mDeleted=true    retired
    picoclaw_foreground    mDeleted=true    retired, dead channel, no twin
    pocketclaw_service     mDeleted=false   live, mImportance=2

That is the `KEEP_CANONICAL_DELETE_LEGACY` path in
`PocketClawNotificationChannels`, and the importance carried across from the
legacy channel rather than resetting to a default. Deleted channels stay in the
dump because Android keeps the record — an app must not be able to resurrect a
channel to wipe a user's settings — so their continued presence is the expected
shape of a completed migration, not an incomplete one.

SharedPreferences migration (`picoclaw_prefs` → `pocketclaw_prefs`) could not be
observed: `run-as` refuses a non-debuggable release build, which is correct, and
root was not used to work around it.

Pre-install state, for comparison: `picoclaw_service` and `picoclaw_foreground`
both live, `pocketclaw_service` absent. The shared workspace directory was
already canonically named and held neither pid record.

### Next

The user opens PocketClaw, starts the Service/Gateway, waits for Running. Then
physical Zero-Pico migration validation, and only then does the baseline move to
62.

## Final Zero-Pico Core rebuild — staged, NOT a candidate yet

- Status: **built and staged on `feature/zero-pico-runtime`, 2026-09-09. NOT
  merged, NOT a candidate.** `0.2.0+61`, baseline 59, no vc62, no What's New.
- **The staged Core is no longer intentionally stale.** This is the first
  canonical rebuild since the Zero-Pico Core-source migration began, and the
  production source gate is green end to end for the first time since N4E.

### Canonical build evidence

    build-input commit   5558220   (not the staging commit; see invariance)
    epoch                1788978634   (SOURCE_DATE_EPOCH unset; resolver-derived)
    BuildTime            2026-09-09T18:30:34+0000
    source fingerprint   6e2382aee9d3fa4beef32ed34678ee08eed0a9db43288c8b879db3c5276d6b1f

    libpocketclaw.so       37749089 bytes
      9e85e471164b53bfee2888f219c5329275a854f3ee7ca6ffdc872d3da3e98d89
    libpocketclaw-web.so   25559393 bytes
      479003e0ed315431153764e7b0b8eb58e857b9cff0fec162aad490a3f342d918

Both binaries carry that one fingerprint and that one BuildTime, verified by
reading the staged bytes rather than trusting the build log. That is what N4K-A
was for: before it the launcher carried no stamp at all, because nothing in
web/backend read the `-X` target and the linker dropped it, value and all.

Built by `./core/build-android-arm64.sh`, exit 0, with the repository toolchain.
No ad-hoc `go build`.

### The embedded bundle was rebuilt, not reused

`web/backend/dist` was emptied to its tracked `.gitkeep` before the build, and
the canonical path regenerated it through `pnpm build:backend`. The bundle that
had been sitting there hashed `039c2d35…`; what the build produced hashes
`d77fe654…`, so the old one really was stale and really would have shipped.

### Reproducibility, both binaries

A second build from the same source with `SOURCE_DATE_EPOCH` pinned to the
resolved epoch — and `dist` emptied again first, so the frontend went through
the full contract on that run too — produced:

    libpocketclaw.so       byte-identical
    libpocketclaw-web.so   byte-identical
    web/backend/dist       identical digest (d77fe654…)

### Freshness now discriminates between the two binaries

The N4J failure shape was: web/backend changes, the staged launcher goes stale,
the gateway is current, and every check stays green. Proven impossible now — with
the gateway untouched and only the staged launcher de-stamped,
`TestStagedCoreWasBuiltFromTheCurrentSource` fails naming `libpocketclaw-web.so`.
The canonical pair was restored and re-verified against its recorded hashes
afterwards.

### Native hardening, both binaries

ARM aarch64, PIE, stripped, `BuildID` present; `GNU_STACK` is RW, never RWX;
maximum `LOAD` `p_align` is `0x10000` (64 KiB), which satisfies Android's 16 KiB
page requirement; zero developer absolute paths; zero Go VCS stamps
(`-buildvcs=false` held); no `.symtab` or `.debug_*` sections. No
`libpicoclaw*.so` is staged, and no compatibility alias was created.

### Staging invariance

The staging commit moved HEAD to `13a2bc3`, and the build-input commit stayed
`5558220`, the BuildTime stayed `2026-09-09T18:30:34+0000`, and the fingerprint
stayed `6e2382ae…`. Committing binaries does not re-date the build, and no
rebuild is implied by HEAD moving. `version.txt` and the guard allowlist changed
alongside; neither is a fingerprint or BuildTime input.

### Gate

`tool/release_gate.py --verify-source --release-class production` exits 0 with
all 17 checks PASS and nothing skipped, including `a1.contracts` and
`a2.placement_guards`, which needed the repository toolchain Flutter on PATH —
they had been silently skipping. `namespace.no_active_pico` PASS: 0 unclassified
occurrences, 19 allowlist entries all in use (upstream 9, legacy_migration 9,
external_compatibility 1).

The gate's `PENDING_FINAL_HARDENING` list no longer claims the namespace
migration is outstanding; it has a dedicated check now, and the standing note
contradicted it.

### Toolchain

Flutter lives at `/home/lordegypt/PocketCLaw/.tooling/flutter/bin`, not on the
default PATH. Flutter 3.47.1 / Dart 3.13.1. `flutter analyze lib/ test/` clean
and all 433 Dart tests pass, including the Zero-Pico phase guards. Nothing in
the source needed correcting for that — N4K-B's Dart edit was already right; only
the invocation had been missing.

### Next

vc62: bump, canonical APK, artifact gate, then device acceptance. The Core is
evidence-complete; nothing here is physically verified yet.

## Zero-Pico N4K-B — final active sweep and enforcement guard, NOT closed

- Status: **implemented on `feature/zero-pico-runtime`, 2026-09-09. NOT merged.**
  `0.2.0+61`, baseline 59, no candidate, no What's New entry.
- Core source fingerprint moved `b9742fe0…` → `6e2382ae…`.
  **Staged Core remains EXPECTED STALE — FINAL ZERO-PICO CORE REBUILD PENDING.**

### The policy

**Zero Active Pico.** No PocketClaw-owned production surface may write, issue,
default to, emit, register or advertise a Pico-family identity. A remaining
occurrence is legitimate only as one of: upstream identity, legal attribution,
historical evidence, legacy migration (read/parse/normalize/migrate/delete/
expire/redact), or external compatibility with named evidence.

This is not a repository-wide rename, and a repository-wide grep would be wrong:
the Go module is still `github.com/sipeed/picoclaw`, the copyright is still
PicoClaw contributors, and several on-disk names must stay readable so an
upgraded install keeps working.

### What moved

    POCKETCLAW_DISTRIBUTION_CHANNEL   was PICOCLAW_; no build supplied the old
                                      name, so it moved without an alias
    irc nick default "pocketclaw"     was "picoclaw"; IRC is compiled into the
                                      shipped Core, so an enabled channel with
                                      no chosen nick sent the old identity on
                                      the wire. Only the default moved — a nick
                                      already in a user's config is their data
    channels.name.pocketclaw          the i18n key was still `pico` while the
                                      channel became `pocketclaw` in N4H, so the
                                      dashboard had silently lost the "Web"
                                      label and fell back to a title-cased key
    <home>/pocketclaw/runtime         the Managed Runtime metadata fallback
                                      created a directory named picoclaw. The
                                      Android host always sets
                                      POCKETCLAW_RUNTIME_DIR so the shipped
                                      product never reached it, but the fallback
                                      recreated the very directory the migration
                                      retires
    pocketclaw-oauth-result           postMessage type, moved on both sides at
                                      once (Go emitter and dashboard listener)
    data-pocketclaw-code-block        DOM attributes the dashboard writes, with
    data-pocketclaw-highlight-theme   their CSS selectors
    pocketclaw:* browser keys         last-session-id, code-block-wrap, tour
                                      state, assistant-detail-visibility
    pocketclaw-web                    the frontend package id (private, unlisted
                                      in the lockfile)

Go and TypeScript symbols followed: `pocketClawToken`, `pocketClawCfg`,
`createPocketClawHTTPProxy`, `refreshPocketClawTokensLocked`,
`ensurePocketClawTokenCachedLocked`, `pocketClawGatewayProtocol`,
`findJSONLSession(s)`, `jsonlSessionRef`, `PocketClawMessage`,
`handlePocketClawMessage`. User-visible log and HTTP error strings that said
"Pico" now say PocketClaw. The misleading `picoSessionPrefix` alias — a
test-only name for a legacy constant that read as current — was deleted rather
than renamed.

Per-browser dashboard preferences reset once on upgrade. That is the whole cost:
the tour reappears, code-block wrap returns to its default, and the chat opens a
new session. Nothing server-side is touched.

### What stays, and why

    upstream            github.com/sipeed/picoclaw, cmd/picoclaw, BINARY_NAME,
                        picoclaw/picoclaw.exe process lookups, the macOS and IRC
                        upstream scripts, desktop launch-at-login identifiers
                        (unreachable: runtime.GOOS is "android" on the product)
    legal               upstream file headers and copyright
    historical          the vc60/vc61 Managed Runtime digest
                        47b63011a55eaa659470f2ab09d05532e9942800848020a1cfe443e6f21aca76,
                        unchanged, and the migration record in this file
    legacy_migration    .picoclaw.pid, filesDir/picoclaw/, picoclaw_prefs, the
                        legacy notification channel ids, picoclaw_launcher_auth,
                        PICOCLAW_* env inputs, pico/pico_client/pico-user, the
                        /pico route redaction arms, the legacy session prefix
    external_compat     wecomQRSourceID = "picoclaw"

### WeCom: preserved, on evidence

`wecomQRSourceID` is sent to `https://work.weixin.qq.com/ai/qc/generate` as the
`source` and `sourceID` query parameters. It is not this project's name for
itself — it is what a third-party service was registered to recognise, so
changing it is a claim to someone else's system rather than a rename, and an
unregistered value would break WeCom QR login rather than fail loudly.

There are **two** independent copies of the constant. The one in
`cmd/picoclaw/internal/auth/wecom.go` is upstream CLI only, but the one in
`web/backend/api/wecom.go` is the dashboard, which ships — so this value really
does reach Tencent from the product. That was found by the guard's own test, not
assumed. Both are pinned, and held identical, by `tool/test_no_active_pico.py`.

### The guard

`tool/no_active_pico.py` scans tracked PocketClaw-owned **production** source
and requires every Pico-family match to be claimed by an allowlist entry
carrying a category and a reason. Unclaimed matches fail; so does an entry that
matches nothing, because a stale exemption silently re-permits whatever moves
back under it. 18 entries, all in use.

Out of scope, deliberately and stated in the tool: the vendored upstream tree
(renaming it forks the baseline), documentation (it has to name the old
identities to describe them), tests (a test proving Pico is gone must write the
word), staged binaries (build output), and the guard's own two files — a rule
listing what it forbids cannot be scanned by itself without either flagging its
allowlist or claiming its own patterns and looking clean by construction.

The gate runs it as `namespace.no_active_pico`, placed **above** the
`--no-tests` early return: it is a source scan, not a test suite, and skipping
tests must not skip it. Proven end to end — injecting
`picoclaw_test_output` into a production file turns the gate red, and removing
it turns it green.

The lexical guard complements the semantic ones from N4B–N4J; it does not
replace them. Those still prove direction — that `.picoclaw.pid` is never
written, that the legacy cookie is never issued, that Android emits only
`POCKETCLAW_*`.

### Generated web bundle freshness

`libpocketclaw-web.so` embeds `web/backend/dist`, which the fingerprint
deliberately excludes and covers through `web/frontend/` instead. That trade is
only sound if the canonical build always regenerates the bundle from those
sources — otherwise a source edit would move the fingerprint, the build would
succeed, and the embedded UI would still be the previous one.

The existing Makefile already guarantees it: `build-android-arm64` depends on
`build-frontend`, whose `pnpm build:backend` runs unconditionally (only the
dependency install is guarded), and `vite build --outDir ../backend/dist
--emptyOutDir` wipes the directory first so a removed file cannot survive.
Nothing needed fixing; `web_bundle_freshness_test.go` pins the chain. Both
mutations were proven: dropping the `build-frontend` dependency, and moving the
bundler inside the conditional, each turn it red.

### Next

The final deterministic rebuild of both binaries, then vc62.

## Zero-Pico N4K-A — dual-binary Core provenance, NOT closed

- Status: **implemented on `feature/zero-pico-runtime`, 2026-09-09. NOT merged.**
  `0.2.0+61`, baseline 59, no candidate, no What's New entry.
- Core source fingerprint moved `db8deae0…` → `b9742fe0…`, because the input set
  itself changed rather than the source. It is not a built identity yet.
  **Staged Core remains EXPECTED STALE — FINAL ZERO-PICO CORE REBUILD PENDING.**

### The gap

The canonical build stages **two** native binaries and ships both:

    libpocketclaw.so       Core gateway,  built from ./cmd/picoclaw
    libpocketclaw-web.so   dashboard,     built from ./web/backend

`pkg/coresource` fingerprinted only `cmd/`, `pkg/`, `workspace/` and three root
files. `web/` was excluded on the true but insufficient ground that the Core
imports none of it — it does not have to, being separately compiled and
separately shipped. N4J changed dashboard auth middleware, moved no fingerprint,
and left a stale dashboard binary that no guard would report. `core/staged_*`
went green on the strength of a binary that was not the one at issue.

### One provenance unit

`includedRoots` now names `web` alongside `cmd`, `pkg` and `workspace`, and one
fingerprint speaks for both binaries. Under `web/`, everything counts except:

    web/backend/dist/**              the generated bundle
    web/frontend/node_modules/**     installed, not tracked
    *_test.go, *.test.ts, *.test.tsx tests reach neither binary

`dist/` is the compiled frontend, written by `pnpm build:backend` and untracked
apart from a `.gitkeep`. Hashing it would fold a build output into the
fingerprint of its own inputs and make the value depend on whether the builder
had run pnpm, so it is covered through `web/frontend/` instead — a stronger
relation, not a weaker one: editing a component moves the fingerprint at once,
without anyone rebuilding the bundle first. `TestEveryEmbeddedAssetIsAFingerprintInput`
knows about that indirection through `generatedEmbedSources` and asserts the
generator is covered, so the exemption cannot become a hole.

The frontend rule is deliberately coarser than the Core's. Deciding exactly
which of Vite's inputs can alter the emitted bundle means re-deriving Vite's
behaviour by hand and being wrong quietly; an over-broad rule costs an
occasional unnecessary rebuild, and an under-broad one is what N4J walked into.

### Both binaries are now verifiable

`-X coresource.Stamped` already reached the launcher build through the LDFLAGS
the root Makefile passes down, but nothing in `web/backend` read it, and the
linker drops an `-X` target with no live reader — value and all. So the flag
succeeded and the binary carried nothing. `web/backend/main.go` now logs
`coresource.Describe()` at startup, which is what keeps it. Verified directly: a
host build carries the fingerprint, and the same build with the reader removed
does not.

`core/build-android-arm64.sh` now checks the stamp in **both** staged binaries,
and `web/Makefile` carries its own `SOURCE_FINGERPRINT` plumbing so a direct
`make -C web build-android-arm64` cannot emit an unstamped dashboard.

### Freshness and the gate

`TestStagedCoreWasBuiltFromTheCurrentSource` iterates `StagedCoreBinaries` and
fails per binary, so a stale dashboard can no longer pass on the Core's
freshness. The release gate's artifact check gained
`artifact.core_provenance_pair`: both packaged binaries must carry the *same*
fingerprint, and it must be the one the current source produces.

### BuildTime was already right

`core/resolve-build-time.sh` has covered all of `core/src` — web included —
since it was written, and its comment documented the asymmetry as deliberate.
The asymmetry is what allowed the gap, so the comment is corrected rather than
the scope. The only behavioural change is excluding `*.test.ts` / `*.test.tsx`
under `web/frontend`, which brings its test-exclusion into line with the
fingerprint's; `TestBuildTimeCoversEverythingTheFingerprintDoes` holds the two
together from now on.

### Cookie classification (N4J correction)

`picoclaw_launcher_auth` is **LEGACY COOKIE CLEANUP ONLY**, not "read-only
session handoff". Nothing reads its value: it is never issued, never validated
and never consulted for authentication, and the only production use is writing
an expiry. The runtime behaviour was already correct; only the wording
overstated it. A test now fails on the word "handoff".

### Not in this phase

`PICOCLAW_DISTRIBUTION_CHANNEL`, the IRC default nick, the WeCom source id, the
remaining source-only Pico identifiers and the final active-Pico guard are
N4K-B. The deterministic byte-level rebuild of both binaries is the final phase.

## Zero-Pico N4J — dashboard session cookie, NOT closed

- Status: **implemented on `feature/zero-pico-runtime`, 2026-09-09. NOT merged.**
  `0.2.0+61`, baseline 59, no candidate, no What's New entry.
- Core source fingerprint **did not move**: `db8deae0…` before and after. That
  was not a sign the edit failed to land — it was a real gap in the staleness
  guard, described under "The fingerprint does not cover this change" below and
  closed by N4K-A.
  **Staged Core remains EXPECTED STALE — FINAL ZERO-PICO CORE REBUILD PENDING.**

### Canonical and legacy

    pocketclaw_launcher_auth   canonical; the only name this build ever issues
    picoclaw_launcher_auth     LEGACY COOKIE CLEANUP ONLY

Both are declared once, in
`core/src/web/backend/middleware/launcher_dashboard_auth.go`. The legacy name is
a single unexported constant, and the only thing production does with it is
write an expiry.

### There is no session handoff, on purpose

The brief allowed reissuing a verified legacy session under the canonical name.
That was rejected after reading the session store: `LauncherDashboardSessions`
is an in-memory `map[string]time.Time` built fresh at process start, so a cookie
issued by an older build names a session in a process that is gone. No legacy
value can ever be validated. Honouring one would mean trusting a bearer token
with no server-side record — precisely the bypass the store exists to prevent.

So the upgrade costs **one dashboard login**, which is what it already cost: the
session store has never survived a restart under either name. This is the
brief's "prefer forcing ONE login rather than weakening the session model" arm.

### Precedence

`validLauncherDashboardAuth` reads the canonical cookie and nothing else. The
legacy name is never a fallback — not when the canonical cookie is absent
(pointless), and above all not when it is present but invalid, which would let
an old cookie rescue a rejected session. The server-side store stays the sole
authority.

    valid canonical                    → authenticated
    valid canonical + any legacy       → authenticated, legacy expired
    legacy only, any value             → unauthenticated
    invalid canonical + valid legacy   → unauthenticated

### Unchanged

Cookie attributes are the pre-N4J contract exactly: `HttpOnly`, `SameSite=Lax`,
`Path=/`, host-only, `MaxAge` 24h, `Secure` from the same detector. Migration
extends no lifetime and creates no second session. Logout still revokes the
server-side session first and now expires both names; the legacy expiry uses
`Path=/` and the original attributes, because a deletion whose Path does not
match silently leaves the cookie in place. Auth scope and routing are untouched:
`/pocketclaw/*` and `/api/pocketclaw/*` are exactly as N4I left them.

### The fingerprint does not cover this change

`pkg/coresource` fingerprints `cmd/`, `pkg/`, `workspace/` and three root files.
It deliberately excludes `web/`, on the documented ground that the Core imports
none of it — and `go list -deps ./cmd/picoclaw` confirms no `picoclaw/web`
package is reachable, so for `libpocketclaw.so` that is correct.

But `core/build-android-arm64.sh` stages **two** binaries, and the second,
`libpocketclaw-web.so`, is built from `./web/backend` by
`build-launcher-android-arm64`. That is the dashboard, and it ships on the
device. A change to dashboard auth middleware therefore moves no fingerprint and
raises no staleness signal, even though the staged artifact is now behind the
tree. N4J is such a change.

Nothing was altered in N4J itself. **N4K-A closed this**, by widening the one
canonical fingerprint to cover both binaries rather than giving the launcher a
second fingerprint universe. See the N4K-A section at the top of this file.

### Deferred

`PICOCLAW_DISTRIBUTION_CHANNEL`, the IRC default nick and the WeCom source id.

## Zero-Pico N4I — realtime routes and log sanitization, NOT closed

- Status: **implemented on `feature/zero-pico-runtime`, 2026-09-09. NOT merged.**
  `0.2.0+61`, baseline 59, no candidate.
- Core source fingerprint moved `eb3c8b4d…` → `db8deae0…`.
  **Staged Core remains EXPECTED STALE — FINAL ZERO-PICO CORE REBUILD PENDING.**
- **FLAG FOR FINAL RELEASE NOTES** (not written yet): PocketClaw's internal
  realtime namespace changed, and an existing Web-channel conversation starts a
  new PocketClaw session after the upgrade. No implementation or security detail
  in the eventual user-facing wording.

### Canonical routes

    /pocketclaw/ws            realtime socket (gateway, and the console proxy)
    /pocketclaw/media/{id}    attachment download
    /api/pocketclaw/info      console management API
    /api/pocketclaw/token
    /api/pocketclaw/setup

`/pico/events` and `/pico/send` were **not** carried across: nothing called their
URL builders and no handler ever served them. They are removed rather than
renamed into a namespace they never reached.

`pkg/config/realtime_routes.go` derives all of them from `ChannelPocketClaw`, and
the channel, the console proxy and the middleware read them from there. Before
this, the same strings were written out in three packages.

### No alias, on purpose

There is no `/pico/*` compatibility route. The gateway, the console frontend and
the app ship in one artifact, so the only client that can still ask for the old
path is a browser tab left open across the upgrade, which reloads. A legacy
request now 404s, and a test asserts that rather than leaving it to inspection.

### Redaction moved in the same change

The route is what redaction keys on, so it moved together with:

    pkg/logger/logger.go                     internal-route field redaction
    web/backend/api/user_visible_log.go      plain-text normalizer
    web/frontend/src/lib/plain-text-log.ts   console log view
    lib/src/core/plain_text_log_sanitizer.dart   app Logs screen

Every pattern accepts both spellings. The canonical arm is what this build
emits; the legacy arm is what a log file written before the upgrade contains,
and dropping it would make old logs *less* redacted than they were.

`pkg/logger` cannot import `pkg/config` — config imports logger — so it keeps a
pinned copy of the prefix, marked as such, with a test holding the two together.

The sanitizer tests are behavioural: real lines through the real sanitizer, on
all four surfaces. They also assert the opposite property — that ordinary lines,
`/pocketclawish/ws` and a `picometer` component come through untouched — because
a sanitizer broad enough to hide the route by hiding everything would pass the
positive tests and be worthless.

### Components and messages

The realtime log components are `pocketclaw` and `pocketclaw_client`. Seven
agent log messages that still said "pico" now emit the wording the sanitizers
were already rewriting them to, so the user-visible text is unchanged and Core
stops emitting the word at all. The compatibility maps keep those entries for
older log files.

The client channel's conversation id is `pocketclaw_client:` and its remote
sender is `pocketclaw-remote`, both with legacy-tolerant parsing that mints
nothing.

### Auth and media unchanged

The dashboard-auth WebSocket origin check and the unauthorized-response shape
now key on the canonical path via the shared constant. Nothing was broadened:
the legacy path is no longer special-cased at all, which the middleware tests
prove by no longer reaching the WebSocket branch. Media path extraction,
traversal validation, authorization and content-type behaviour are untouched;
only the prefix constant changed, and it is the same constant the URL builder
uses.

### Deferred

`PICOCLAW_DISTRIBUTION_CHANNEL`, the IRC default nick and the WeCom source id.
The `picoclaw_launcher_auth` cookie name was deferred here and taken by N4J.

## Zero-Pico N4H — channel, client and owner canonical, NOT closed

- Status: **implemented on `feature/zero-pico-runtime`, 2026-09-09. NOT merged.**
  `0.2.0+61`, baseline 59, no candidate, no What's New entry.
- Core source fingerprint moved `2b06b4a8…` → `eb3c8b4d…`.
  **Staged Core remains EXPECTED STALE — FINAL ZERO-PICO CORE REBUILD PENDING.**

### Canonical identities

    pocketclaw          managed realtime channel type and config key
    pocketclaw_client   its client half
    pocketclaw-user     owner principal, the only value owner-only accepts

`pico`, `pico_client` and `pico-user` are migration input only, declared once in
`pkg/config/channel_legacy.go`. No writer emits them; a test walks every
production Go file to prove it, exempting only the deferred log-component call
shape.

### Migration, and why it edits the document

`migrateChannelIdentities` runs at the top of `LoadConfig`, before channel
construction, owner authorization or the token lookup. It is a **targeted edit of
the serialized config**, not a load-and-save: `SaveConfig` marshals the typed
`Config` and is lossy for anything the struct does not model, and a namespace
migration is the wrong moment to discover that. Unrelated entries are carried as
raw bytes and keys keep their order, so the user's file changes on exactly the
lines the identity does.

It covers both `channel_list` (current) and `channels` (pre-v3), and it covers
**`.security.yml` as well** — the channel token is filed there under the channel
name and merged back by name at load, so renaming only `config.json` would leave
the credential under a name nothing looks for. The credential file is written
first: a crash between the two writes leaves a legacy config with a canonical
security file, which migrates again on the next start; the other order would
silently lose the token.

    legacy only        rename key, type and owner principal
    canonical only     untouched, byte for byte
    both, identical    canonical wins, legacy key dropped
    both, differing    FAIL CLOSED — nothing changed, load refused

Two differing definitions can carry two different tokens, and nothing on disk
says which the user meant. `ErrChannelMigrationConflict` is fatal to the load
rather than resolved by guessing.

### Owner principal

Rewritten only inside a channel's `allow_from`, never as a global string
replacement. Both spellings in one list collapse to one owner — a rename, not a
widening, and the list never grows. Owner-only enforcement compares against
`PocketClawOwnerPrincipal` alone, and the legacy label is now explicitly in the
rejection set alongside `PICO-USER`, `pocketclaw_user` and the rest.

### Sessions re-key. This is unavoidable and intentional.

`CanonicalScopeSignature` includes `channel=`, so renaming the channel changes
the session key for the Web channel. **Existing Web-channel conversation history
is not carried across.** No amount of chat-id compatibility avoids it — the
channel name alone re-keys the hash — so the conversation-id prefix moved too,
with legacy-tolerant parsing so a message already in flight still routes. Other
channels are unaffected.

### Deferred to the route/sanitizer phase — explicitly

    /pico/, /pico/ws, /pico/events, /pico/send, /pico/media/*
    /api/pico/info | token | setup
    logger component "pico" / "pico_client"
    web/backend/api/pico.go, frontend api/pico.ts, use-pico-chat

These travel through one redaction path. The Go sanitizer keys `path=/pico/`
against the channel field, and the frontend matches the component token and the
caller filename. Splitting them leaves a build whose internal route stops being
redacted, so they move as one piece. Both sanitizers were extended here to
accept the canonical channel beside the legacy route, and the frontend's caller
pattern now matches `pocketclaw.go` — the file moved with its package.

### Env token

Unchanged by decision: `POCKETCLAW_CHANNELS_POCKETCLAW_TOKEN` still resolves
through the canonical-env adapter onto the upstream tag
`PICOCLAW_CHANNELS_PICO_TOKEN`. **Zero struct tags renamed.** Retagging that one
field would put a single canonical name among ~175 legacy ones and drop legacy
env support for it, for no behavioural gain.

## Zero-Pico N4G — `.pocketclaw.pid` canonical, NOT closed

- Status: **implemented on `feature/zero-pico-runtime`, 2026-09-09. NOT merged.**
  `0.2.0+61`, baseline 59, no candidate, no What's New entry.
- Core source fingerprint moved `369d0892…` → `2b06b4a8…`.
  **Staged Core remains EXPECTED STALE — FINAL ZERO-PICO CORE REBUILD PENDING.**

### What changed, and what did not

`.pocketclaw.pid` is the only record this build writes. `.picoclaw.pid` is
discovery input and cleanup target only: read to find a Gateway an older build
started, removed once that Gateway is gone, never written, never recreated,
never symlinked, never copied back to.

The record's contents and security scope are untouched — `host`, `pid`, `port`,
`version`, and no credential — and it stays in `POCKETCLAW_HOME` beside the
workspace. This is a rename, not a relocation: it did not move to
`pocketclaw-core/`, `credentials/`, `logs/` or `auth/`.

### One resolver

`pkg/pid/pidfile_migration.go` holds both names and the whole policy. Startup,
the status API, shutdown and the console's cleanup all reach the records through
`resolvePidRecords` + `activeRecord`, so migration, precedence and stale rules
cannot drift apart per caller.

There are deliberately **two liveness predicates over one policy**. Startup asks
"is this PID a live Core executable", because wrongly honouring a recycled PID
wedges the Gateway behind a record for a process that is not it. The status API
asks only "is this PID alive", which is exactly what it asked before N4G;
tightening it there would change what the console reports, and this phase is a
filename migration. The difference is the predicate only.

### State machine

    neither                       write canonical
    canonical only                unchanged behaviour
    legacy only, live             honour it, block a duplicate start,
                                  leave the file, fabricate no canonical record
    legacy only, stale/malformed  clean it, write canonical
    both, same live PID           canonical wins, legacy removed as redundant
    canonical live, legacy stale  canonical used, stale legacy removed
    canonical stale, legacy live  live legacy honoured, stale canonical removed,
                                  legacy NOT converted behind the running process
    both live, different PIDs     FAIL CLOSED
    both stale                    clean both, write canonical

A live legacy record is never removed, renamed or rewritten. The process holding
it knows itself by no other name, and moving the file out from under it would
leave its shutdown removing a name that no longer exists while its record
lingered forever.

Two live Gateways is a split-brain discovery state. Nothing on disk can say which
is meant, so `WritePidFile` returns a conflict naming both PIDs and paths, and
`ReadPidFileWithCheck` returns nothing rather than picking one. Neither process
is killed, neither record is overwritten, and no third Gateway is started.

### Ownership is unchanged

The N3 exact-executable rule stands: `libpocketclaw.so` / `libpocketclaw-web.so`
and their 15-byte truncated comm forms, nothing else. The Android app process
`gypt.pocketclaw`, all eight Managed Runtime payloads, unrelated processes, and
the pre-N3 `libpicoclaw.so` / `libpicoclaw-web.so` are all foreign — now proven
through the legacy record path as well as the canonical one.

### History is kept

`.picoclaw.pid` stays in the tests, in the physical vc61 evidence and in the
Kotlin comments that explain why the gateway credential left it. That the file
once existed and once named the live Gateway is a fact about this project, and
the guard is written against filename literals in production Go rather than
against the word, so the reasoning that records it is not itself a violation.

## Zero-Pico N4F — Core private state at `pocketclaw-core`, NOT closed

- Status: **implemented on `feature/zero-pico-runtime`, 2026-09-09. NOT merged.**
  `0.2.0+61`, baseline 59, no candidate, no What's New entry.
- **`core/src` untouched.** Fingerprint stays `369d0892…`.
  **Staged Core remains EXPECTED STALE — FINAL ZERO-PICO CORE REBUILD PENDING.**

### The path contract, now in force

    Download/pocketclaw/workspace   USER WORKSPACE — POCKETCLAW_HOME. Unchanged.
    files/pocketclaw/               USER WORKSPACE FALLBACK when external storage
                                    is unavailable. The user's documents. Untouched.
    files/pocketclaw-core/          CANONICAL Core private state. config.json,
                                    .security.yml, runtime/ metadata.
    files/picoclaw/                 LEGACY. Migration input only.
    noBackupFilesDir/               Gateway token, logs/, auth/. Unchanged.

Three storage boundaries, still three. Nothing moved into `pocketclaw-core/` for
namespace symmetry: the workspace is the user's, the credentials and logs and
Dashboard verifier are A2's, and only Core's config directory moved.

### One owner

`PocketClawCoreState` holds both names. Every consumer — `buildEnvironment` and
so every spawn path, onboarding, the web service, and `getConfig` / `saveConfig`
/ `getConfigPath` on the method channel — asks it for the directory rather than
spelling a path, so no caller can reach a directory whose migration has not run.
A guard fails if any other Kotlin file builds a path from either name.

### The state machine

    legacy only        rename to pocketclaw-core, verify both sides
    canonical only     use it
    neither            create pocketclaw-core; never create picoclaw
    both               FAIL CLOSED — no merge, no overwrite, no delete

Both-present is an interrupted migration or a downgrade, and the two directories
can hold different provider keys and channel tokens. Nothing on disk says which
the user meant, so it is reported rather than guessed at.

### Rename, and deliberately no copy fallback

Both directories are direct children of `filesDir`, so they are always on one
filesystem and the move is a single `rename(2)`: the tree arrives whole or not at
all, including hidden files, unknown files and nested directories. Nothing
enumerates the contents, so nothing can migrate a subset — the JVM test seeds
`.hidden-state`, `unknown-future-file.dat` and `runtime/nested/deeper/leaf.txt`
and reads all three back on the other side.

There is no copy fallback by choice. A recursive copy would have to reproduce
the permissions on `.security.yml`, which holds provider API keys and channel bot
tokens in plaintext because Android onboarding declines credential encryption,
and Java's file APIs cannot express those modes or fsync a directory. The honest
outcomes were a second plaintext copy of the user's secrets in a
partially-written tree, or a delete of the original after an unverifiable copy.
Since the two paths cannot be on different filesystems, a failed rename means a
real filesystem or permission fault — so it fails closed, having changed nothing.

### Fail closed, everywhere

`directory()` throws rather than returning a usable-looking path. A caller handed
a fresh empty directory after a failed migration would let Core onboard into it,
and the user would see an install with no providers, no channels and no memory:
a factory reset presented as a successful start, with the real state still on
disk and nothing pointing at it. The method channel surfaces
`CORE_STATE_UNAVAILABLE` instead of an empty config for the same reason.

### One-way, on purpose

After migration, an older APK that only knows `files/picoclaw/` will not see the
migrated state. **No second copy is kept to support downgrade.** That would be an
active write to the legacy path and would create two directories that disagree —
which is precisely the both-present state this phase refuses to resolve
automatically. This is an intentional one-way storage migration, and no
seamless-downgrade claim is made.

`picoclaw/` stays in both backup rule files as a LEGACY SECURITY EXCLUSION:
an interrupted migration or a downgrade can leave secrets there, and changing
where PocketClaw writes must not make what is already there backup-eligible.

### `POCKETCLAW_CONFIG`

Now `files/pocketclaw-core/config.json`. `.security.yml` follows it without being
named anywhere: Core derives that path from the config file's own directory
(`securityPath(configPath)`), which is why the directory rather than the file is
the unit that moves. `POCKETCLAW_RUNTIME_DIR` likewise moves to
`pocketclaw-core/runtime`, carried by the same rename.

### Deferred

`.picoclaw.pid`, the serialized "pico" channel and `picoTokenForHost` — path
analysis puts the token on the channel, not on this directory —
`pkg/channels/pico`, and `PICOCLAW_DISTRIBUTION_CHANNEL`.

## Zero-Pico N4E — canonical `POCKETCLAW_*` environment, NOT closed

- Status: **implemented on `feature/zero-pico-runtime`, 2026-09-09. NOT merged.**
  `0.2.0+61`, baseline 59, no candidate, no What's New entry.
- **Key names changed. No value moved.** Core state is still written to
  `filesDir/picoclaw/`, the config is still `filesDir/picoclaw/config.json`, and
  the workspace, gateway token, logs and Dashboard verifier are all in exactly
  the places A2 put them.
- Core source fingerprint moved `f015c644…` → `369d0892…`.
  **Staged Core is EXPECTED STALE — FINAL ZERO-PICO CORE REBUILD PENDING.**

### The split

The Android host emits only `POCKETCLAW_*`. The vendored Core still reads
`PICOCLAW_*` — 175 struct tags and a dozen direct lookups — and a canonical
adapter translates between them. Renaming upstream's tags would be a permanent
divergence for no behavioural gain, so the translation is one table instead:

    core/src/pkg/canonicalenv/canonicalenv.go

That file is the entire compatibility surface, which is what lets the final
Zero-Pico guard allowlist it precisely rather than chase scattered literals.
Legacy names there are input this build still accepts, never output it produces.

### The twelve

    POCKETCLAW_HOME                        PICOCLAW_HOME
    POCKETCLAW_CONFIG                      PICOCLAW_CONFIG
    POCKETCLAW_BINARY                      PICOCLAW_BINARY
    POCKETCLAW_GATEWAY_TOKEN_FILE          PICOCLAW_GATEWAY_TOKEN_FILE
    POCKETCLAW_LOG_DIR                     PICOCLAW_LOG_DIR
    POCKETCLAW_DASHBOARD_AUTH_DIR          PICOCLAW_DASHBOARD_AUTH_DIR
    POCKETCLAW_DNS_SERVER                  PICOCLAW_DNS_SERVER
    POCKETCLAW_GATEWAY_HOT_RELOAD          PICOCLAW_GATEWAY_HOT_RELOAD
    POCKETCLAW_TOOLS_I2C_ENABLED           PICOCLAW_TOOLS_I2C_ENABLED
    POCKETCLAW_TOOLS_SPI_ENABLED           PICOCLAW_TOOLS_SPI_ENABLED
    POCKETCLAW_TOOLS_SERIAL_ENABLED        PICOCLAW_TOOLS_SERIAL_ENABLED
    POCKETCLAW_CHANNELS_POCKETCLAW_TOKEN   PICOCLAW_CHANNELS_PICO_TOKEN

The token is the one pair whose suffix also changes. PocketClaw cannot emit a
canonical name containing PICO, and the serialized channel is still called
"pico" until the channel phase, so the halves differ on purpose.

### Two reader classes, not one

N4A's map found two `caarlos0/env` decode points. It did not find the second
class: seven of the twelve are read by direct `os.Getenv`, not by a struct tag —
`HOME`, `CONFIG`, `BINARY`, `GATEWAY_TOKEN_FILE`, `LOG_DIR`,
`DASHBOARD_AUTH_DIR`, `DNS_SERVER`. A parser-only adapter would have left those
seven unread the moment the host went canonical-only, silently returning three
A2 security boundaries — the gateway token file, the private log directory and
the Dashboard verifier directory — to their shared-storage defaults. Both
classes go through the same table.

### Precedence

Canonical wins when set. Legacy still works when canonical is absent, so
existing upstream installs are unaffected. **Presence decides, not emptiness**:
`POCKETCLAW_LOG_DIR=""` is set and beats a non-empty `PICOCLAW_LOG_DIR`, because
a host that blanks a variable is saying something and falling through would
restore exactly what it was turning off.

### No global mutation

The adapter never calls `os.Setenv`. The parser gets a constructed map through
`env.ParseWithOptions`; direct readers ask the resolver. `os.Getenv` of a legacy
name returns what it always did, including nothing when it was never set. Had
the shim written legacy names into the live environment they would have been
inherited by every child the Core spawns — reintroducing the namespace being
removed, one process deeper.

### Guards that changed owner

`namespace_n1_boundary_test.dart`, `namespace_n3_native_identity_test.dart`,
`zero_pico_n4b_test.dart` and `zero_pico_n4c_test.dart` each pinned legacy env
names as proof their own phase had not widened. N4E migrated those names, so
each guard failed, and each was amended with a note naming the new owner rather
than quietly deleted. `zero_pico_n4e_test.dart` owns the emitted set now; the
private directory stays pinned where it was, because it is still deferred.

### Deferred, deliberately

`filesDir/picoclaw/`, `pocketclaw-core/`, `.picoclaw.pid`, the serialized "pico"
channel, `picoTokenForHost` and the other Android-local `pico` identifiers, and
`PICOCLAW_DISTRIBUTION_CHANNEL` — a compile-time dart-define no build supplies,
so not an emission and not in this phase.

## Zero-Pico N4D + N4D-R — private-path protection map, NOT closed

- Status: **implemented on `feature/zero-pico-runtime`, 2026-09-09. NOT merged.**
  `0.2.0+61`, baseline 59, no candidate, no What's New entry. `core/src`
  untouched; fingerprint `f015c644…`, staged Core FRESH.
- **Protection only. No data was migrated.** Core state is still written to
  `filesDir/picoclaw/`, `PICOCLAW_CONFIG` is unchanged, nothing was moved,
  created or deleted, and the runtime behaves exactly as before.

### The path contract, corrected

    files/pocketclaw/        USER WORKSPACE FALLBACK — the user's AGENT.md,
                             SOUL.md, USER.md and memory/ when external storage
                             access is unavailable. Not Core state. Not migrated,
                             not deleted, not backup-excluded.

    files/picoclaw/          LEGACY Core private state — config.json,
                             .security.yml. Read / migrate / protect only.

    files/pocketclaw-core/   CANONICAL Core private state. Protected now,
                             created later.

### Protected in every section

    backup_rules.xml            full-backup-content   credentials/ pocketclaw-core/ picoclaw/
    data_extraction_rules.xml   cloud-backup          credentials/ pocketclaw-core/ picoclaw/
    data_extraction_rules.xml   device-transfer       credentials/ pocketclaw-core/ picoclaw/

`pocketclaw-core/` has exactly the coverage `picoclaw/` had — three sites, same
`domain="file"`, same schema per file. Nothing was weakened, and the two files
keep their different structures because Android's formats differ.

The ordering is the point. The exclusion for the canonical path exists *before*
any code creates that directory. A migration that ran first would leave secrets
in an unprotected path for as long as it took the rules to catch up, and an
interrupted one would leave them there indefinitely.

### `picoclaw/` is a LEGACY SECURITY EXCLUSION, kept deliberately

It is no longer an active product path, and it is **not** a Zero-Pico failure.
An interrupted migration or a downgrade to an older build can leave sensitive
files in the historical directory, and changing where PocketClaw writes must not
make what is already there backup-eligible. Both rule files now say so in those
words, and a test asserts the classification is present — so a later Zero-Pico
sweep cannot delete the line as leftover namespace. It is retired only when the
migration compatibility window is intentionally closed in a later release.

The standing policy this phase establishes: **PocketClaw never creates active
Pico state, but it may still protect or read legacy Pico state during
migration.**

### The collision, and the correction (N4D-R)

Writing N4D's guard failed, because `filesDir/pocketclaw/` **already existed**:
the workspace fallback at `PocketClawService.kt:222`, taken when
`MANAGE_EXTERNAL_STORAGE` is denied. It holds the user's `AGENT.md`, `SOUL.md`,
`USER.md` and `memory/`, not Core's secrets.

So the originally planned target was wrong twice over. Moving Core state there
would have mixed `config.json` and `.security.yml` into the user's documents on
permission-denied installs; and N4D's exclusion of `pocketclaw/` silently
changed that fallback's backup semantics as a side effect of a namespace
migration. Whether a private workspace fallback should be backed up is a privacy
decision on its own terms — it contains `MEMORY.md`, which the Bootstrap
contract treats as the user's private data — and it is not this migration's to
make.

**Cancelled:** `files/picoclaw/` → `files/pocketclaw/`.
**Adopted:** `files/picoclaw/` → `files/pocketclaw-core/`.

The `pocketclaw/` exclusion was reverted **before any data migration**, so the
fallback's pre-N4D backup behaviour is restored exactly. No user file was moved,
read or modified; no secret or runtime state was moved; `PICOCLAW_CONFIG` and
every environment variable are unchanged. A guard now asserts the fallback is
absent from the Core-private exclusion contract, so it cannot be swept back in.

On the validated SM-A165F the permission is granted, the workspace lives at
`/sdcard/Download/pocketclaw`, and the fallback is not in use there.

## Zero-Pico N4C — notification channels, NOT closed

- Status: **implemented on `feature/zero-pico-runtime`, 2026-09-09. NOT merged.**
  `0.2.0+61`, baseline 59, no candidate, no What's New entry. `core/src`
  untouched; fingerprint stays `f015c644…`, staged Core FRESH.

### The audit came back with one live channel and one dead one

`picoclaw_service` is **active**: `PocketClawService.kt:993` builds every
foreground-service notification against it. Migrated to `pocketclaw_service`.

`picoclaw_foreground` is **dead**. `initializeBackgroundService()` created it and
handed it to `flutter_background_service`, but that service is configured
`autoStart: false` and `startService()` is called nowhere in `lib/` — nothing has
ever posted a notification on it. It is deleted on upgrade and **no
`pocketclaw_foreground` twin is created**: a replacement would add a second,
permanently empty entry to the user's notification settings purely for
namespace symmetry. The Flutter configuration now points at the one real
channel, so it stays valid if that service is ever actually started.

### Migration

`PocketClawNotificationChannels` owns both the canonical id and the only
permitted mentions of the legacy ones, and runs from `PocketClawApp.onCreate()`
before anything can post. The decision is a pure function over
(legacy exists, canonical exists):

| legacy | canonical | action |
|---|---|---|
| yes | no | copy settings → create → **verify** → delete legacy |
| yes | yes | canonical wins untouched, delete legacy |
| no | no | create canonical |
| no | yes | nothing |

The legacy channel is the only record of the user's settings until the
replacement exists, so it is deleted last and only after a read-back confirms
the new channel is there.

**Transferred:** importance, description, group, sound URI and audio
attributes, vibration enable and pattern, lights enable and colour, show-badge,
lockscreen visibility. `bypassDnd` is copied best-effort and is silently ignored
by Android unless the app holds DND policy access, which PocketClaw does not
request and will not start requesting for this.

**Cannot be transferred, and is not pretended otherwise:** a channel the user
blocked or muted through system UI, any Do Not Disturb override we lack policy
access to set, and the deletion history Android keeps against the retired id.
Changing a channel id resets some channel-specific settings; that is the cost of
an id Android will not rename, and the product requirement takes precedence.

### Also fixed here

`PocketClawApp` constructed a `NotificationChannel` with **no API-26 guard**
despite `minSdk 24`, so application start would have thrown on API 24–25. The
migration owner is guarded and the construction now lives behind it.

### Legacy read-only

`picoclaw_service` and `picoclaw_foreground` survive only inside
`PocketClawNotificationChannels`, both marked `LEGACY READ-ONLY MIGRATION`, used
only to read settings and to delete. No current code creates either.

Channel groups: **none exist**, so nothing to migrate.

Two earlier boundary guards pinned `picoclaw_foreground` as proof that N1 and N3
had not touched persisted state. N4C legitimately retires it, and those guards
failing is how the scope change was declared rather than absorbed silently; the
assertions moved to `zero_pico_n4c_test.dart`.

## Zero-Pico N4B — Android-local identities, NOT closed

- Status: **implemented on `feature/zero-pico-runtime`, 2026-09-09. NOT merged.**
  Child branch of `feature/namespace-n3-native-binaries` at `904388f2`, which is
  itself unmerged: N3 native identity is physically proven on vc61 but
  deliberately not accepted while Zero-Pico continues. `0.2.0+61`, baseline 59,
  no candidate, no What's New entry.
- **`core/src` untouched.** Fingerprint stays
  `f015c6445d6e19deb9a47c9a41249001fcf5c51265dfae55cd8767a68e560f03`, staged
  Core FRESH, no rebuild, no restage, no APK, no ADB.

### What N4B changed

| Surface | From | To |
|---|---|---|
| SharedPreferences store | `picoclaw_prefs` | `pocketclaw_prefs` |
| Wake-lock tag | `PicoClaw::ServiceWakeLock` | `PocketClaw::ServiceWakeLock` |
| Log-reader thread | `picoclaw-web-log-reader` | `pocketclaw-web-log-reader` |
| logcat tag | `PicoClawChannel` | `PocketClawChannel` |
| Export filename default | `picoclaw_logs.txt` | `pocketclaw_logs.txt` |
| MethodChannel method | `getPicoToken` | `getPocketClawToken` |
| Orphan cleanup | `killPicoClawOrphanProcesses` | `killPocketClawOrphanProcesses` |

**`picoclaw_prefs` is LEGACY READ-ONLY MIGRATION from now on.** It is read once
to carry an existing install's values across and is never created or written
again. All current reads and writes use `pocketclaw_prefs`.

`PocketClawPreferences` is the single place that names the store — three files
declared it independently before, which is how a rename lands in two of them.
Every call site goes through `open()`, so the migration cannot be bypassed by a
caller reaching for `getSharedPreferences` directly. That matters most for
`BootReceiver`: on the first boot after upgrade it would otherwise read an empty
canonical store and silently revert the user's auto-start choice.

Migration order is the design. Copy → synchronous `commit()` → read back and
verify every value → only then `deleteSharedPreferences` on the legacy store.
The legacy file is the only copy of those settings until the new one is proven
durable, so it is deleted last. If both stores exist, canonical wins; a legacy
store that agrees entirely is redundant and removed, and one that disagrees is
**preserved with a logged conflict** rather than merged or discarded.

### The N3 regression this phase fixes

`killPicoClawOrphanProcesses` matched `cmdline.contains("picoclaw")`. After N3
the command line is `libpocketclaw.so`, which does not contain that substring,
so orphan cleanup had silently been matching nothing since the native rename.

It now compares the **basename of argv[0]** against `GATEWAY_BINARY_NAME` and
`WEB_BINARY_NAME` — the same constants used to spawn Core, so cleanup and the
names it cleans up cannot drift apart. Exact comparison, deliberately:
`contains("pocketclaw")` or a `libpocketclaw` prefix would sweep in all eight
Managed Runtime payloads, and killing `gh` or `python` mid-operation would
surface as a random tool failure.

### Deferred, and asserted as deferred

Notification channel IDs `picoclaw_service` and `picoclaw_foreground`
(N4C — Android channel IDs are persisted system objects whose user settings
cannot be transferred perfectly), `filesDir/picoclaw/`, the backup exclusion
rules, `.picoclaw.pid`, the `PICOCLAW_*` environment, `"pico"`/`pico_client`,
`pico-user`, `/pico/*`, `picoclaw_launcher_auth`, the IRC default, the WeCom
source id, and upstream module and build identity. A test asserts five of these
are still Pico, so N4B cannot have quietly widened.

No repository-wide Zero-Pico guard yet — that belongs to the final phase, when
the remaining surfaces have actually moved.

## Namespace Migration N3 — vc60 SUPERSEDED, corrections applied, NOT closed

- Status: **corrections on `feature/namespace-n3-native-binaries`, 2026-09-09.
  NOT merged, NOT accepted.** Baseline stays **59**; `pubspec.yaml` stays
  `0.2.0+60` — vc61 is created only after this corrected source and Core pass
  review.

### vc60 — SUPERSEDED DURING PHYSICAL VALIDATION

    APK  83ba7e2c47df2afc501a2b6f1889dd7c9ce96606388cdcd932bbe0d2ed8f75df

Not accepted. It proved the native rename works physically and, in doing so,
exposed a runtime ownership regression. Its hash stays here as the historical
evidence that produced that finding.

**What vc60 proved.** Installed over vc59 with identity preserved. The device
reports exactly the comm values TASK_COMM_LEN predicts:

    Core gateway  pid 1261  comm = libpocketclaw.s
    launcher/web  pid 1188  comm = libpocketclaw-w
    Android app   pid 23674 comm = gypt.pocketclaw

Gateway loopback-only on `127.0.0.1:18790` and `[::1]:18790`; `.picoclaw.pid`
carrying exactly `host`, `pid`, `port`, `version` with `pid` matching the live
gateway; no `libpicoclaw` process anywhere; installed `nativeLibraryDir` holding
only the new names.

**What it exposed.** Android truncates an *application* process name from the
**left**, so `com.lord1egypt.pocketclaw` becomes `gypt.pocketclaw` — which
contains `pocketclaw`. The substring ownership rule therefore classified
PocketClaw's own UI process as a live Core runtime. A stale `.picoclaw.pid`
whose PID the kernel recycled onto the app would have been honoured and the
gateway would have refused to start, losing the self-healing that existed before
N3 (`gypt.pocketclaw` does not contain `picoclaw`).

### Correction 1 — ownership compares whole names, not fragments

The authorized fix was a `libpocketclaw` prefix. Inspection against the real
payload showed that is **not** the narrowest rule: every Managed Runtime binary
is `libpocketclaw-*`, so `gh`, `git`, `python`, `curl`, `rg`, `jq`, `sqlite3`
and `git-remote-http` would still have counted as the runtime a pid file refers
to — the same bug one size smaller. Ownership now compares the whole comm
against the two canonical Core executables, in truncated and full form, with the
truncation derived in one place.

Twenty cases pin it, every comm value read from the device or produced by the
kernel's own truncation, and the stale-PID recoveries run through
`WritePidFile` rather than the classifier alone. Removing the fix fails exactly
the six that should fail. `libpicoclaw.so` and the upstream desktop name are
**not** aliases.

### Correction 2 — the Managed Runtime count stopped counting

`libpocketclaw-web.so` matches the Managed Runtime prefix, so vc60's gate
reported 9 payloads where 8 exist. It passed, which was the problem: the check
is a floor, so seven real tools plus the launcher would also have reached 8 and
the guard had quietly lost the ability to notice a dropped payload. `CORE_LIBS`
is the exclusion authority now. Four cases pin it. Tooling only — no Core
provenance effect, which the build input confirms by pointing at the pid fix
rather than the gate commit.

### The corrected Core

    build-input commit  d01f47ade57f34c69a478440c1f015bbd494db0f
    epoch               1788914581
    BuildTime           2026-09-09T00:43:01+0000
    fingerprint         f015c6445d6e19deb9a47c9a41249001fcf5c51265dfae55cd8767a68e560f03
    libpocketclaw.so     fecde504c094a15c5396728add388fea3f88a626d833e10797777b4842b9a583  37,683,553
    libpocketclaw-web.so 2b433058551f64a07ff7979641fc3261af37028756c1da5d3f476b85fb4df517  25,493,857

Both stripped, NX stack, 64 KiB aligned, zero developer paths, zero Go VCS
stamps, both carrying the same embedded BuildTime, fingerprint stamped in
`libpocketclaw.so`. No `libpicoclaw*.so` staged. Fingerprint moved from
`259e3422…` because the ownership fix is shipping production Go source.

## Namespace Migration N3A — implemented, NOT merged, NOT physically accepted

- Status: **implemented on `feature/namespace-n3-native-binaries`, 2026-09-09.
  NOT merged, NOT physically accepted.** Branch cut from `develop` at
  `b6a098f`. N3B owns the physical candidate. `0.2.0+59`, baseline 59, no
  candidate, no What's New entry.

### The canonical Android native identity

    libpicoclaw.so       →  libpocketclaw.so
    libpicoclaw-web.so   →  libpocketclaw-web.so

That name is not cosmetic: it is what reaches `nativeLibraryDir`, the APK
payload and `/proc/<pid>/comm`, so the build script, the launcher, Gradle
packaging, the release gate and Core's process-ownership check all moved
together. The Managed Runtime payloads were already `libpocketclaw-*.so`, so
Core now matches the convention its own siblings already used.

**Upstream build identity is deliberately unchanged.** `core/src/Makefile` still
emits `picoclaw-android-arm64` and `picoclaw-launcher-android-arm64`,
`BINARY_NAME=picoclaw`, `cmd/picoclaw` and the Go module are untouched, and the
Makefile's own `build-android-bundle` staging target still writes the old names
because it is upstream's universal-zip path, not PocketClaw's shipping path. The
rename happens at the install destination in `core/build-android-arm64.sh` —
the boundary between upstream's artifact and PocketClaw's package.

### Process ownership — the part that had to change

`pkg/pid/classifyProcComm` matched the substring `"picoclaw"`. `libpocketclaw.so`
does not contain it, so without this change the launcher would have read its own
live gateway as a foreign process and deleted a valid pid file. It now matches
`ownedProcessName = "pocketclaw"`.

**The old name is deliberately not an accepted alias.** A stale `.picoclaw.pid`
can name a PID the kernel has since handed to something else, and every extra
accepted name is another way for that process to be honoured as ours and wedge
startup behind it. N0's conclusion that no fallback loader is needed was
re-verified against the implementation: Android replaces `nativeLibraryDir`
wholesale on package update, so nothing can still be running under the old name.
Tests pin the live match, the reused-PID rejection, the stale pre-N3 name as
foreign, and that the shared PID record still carries no credential.

`looksLikeGatewayCommandLine` needed no change — it matches the `gateway`
subcommand token, not the executable filename, and was already name-agnostic.

### Core source fingerprint moved, and the cause was measured

    e7acbff7…  develop
    4a88a400…  with only pkg/pid/pidfile_unix.go changed
    259e3422…  current, adding a one-line comment in pkg/coresource/fingerprint.go

Both are fingerprinted source. The ownership change had to happen; the comment
did not have to, but the rebuild was already required so its marginal cost was
zero, and the fingerprint is content-addressed over source bytes and
deliberately does not try to tell comments from code.

### The build

    build-input commit  d520e1e8188c38fea9617611efe4d178f3125fe2
    epoch               1788905005
    BuildTime           2026-09-08T22:03:25+0000
    fingerprint         259e342283c78537fdeb5d3392ee64dec45ec63abaddca0afa0090df7263ef4f
    libpocketclaw.so     5c4d8e6c0546705932fcb1f9b11c6cc09839415167202204a59cf2379f9406a2  37,683,553
    libpocketclaw-web.so d6307859e174770e97772effd9c0c89e954452ca086b4b51b79719701362b17a  25,493,857

Both stripped, NX stack, 64 KiB aligned, zero developer paths, zero Go VCS
stamps, and both carry the same embedded BuildTime. `core/build-android-arm64.sh`
is a canonical BuildTime input, so N3 moving the timestamp is expected and not
a reuse of the vc59 value. Staged at `ef40299`; resolving before and after that
commit gave the identical value. A rebuild at the same explicit epoch reproduced
both binaries byte for byte.

Only the PocketClaw-named binaries are staged — the pre-N3 files were removed
from the tree rather than left beside them, so no APK can carry both.

### Old-name references that remain, and why

Upstream Makefile bundle target; the comment in `pidfile_unix.go` explaining why
the old name is rejected; the negative ownership tests; the historical incident
comment in `gateway_test.go`; and two log-redaction fixtures
(`logs-page.test.tsx`, `user_visible_log_contract.json`) that exercise hiding
*upstream* identity from user-visible output. The sanitizer strips the whole
parenthesised path regardless of filename, so redaction covers the new name too.

`core/src/pkg/pcruntime/manifest.go` is unformatted on `develop` already and was
left alone rather than swept into this diff.

## Namespace Migration N2 — CLOSED

- Status: **closed on `feature/namespace-n2-brand-assets`, 2026-09-09, merged to
  `develop` with `--no-ff`.** Branch cut from `develop` at `43500bf`, retained.
  `main` untouched, no tags moved, no release.
- **`core/src` untouched.** Core fingerprint remains
  `e7acbff7000bb58ac8074bfaf290528326df115bf1fae4bb6242defbf78d9a23`, staged
  Core FRESH, no rebuild, no restage, no APK, no ADB. `0.2.0+59`, baseline 59.
- **No What's New entry**, and the inspection says why: the two assets are
  declared Flutter assets, so they were bundled into every APK, but nothing on
  Android reads them — `main.dart` gates the tray behind
  `!Platform.isAndroid && !Platform.isIOS`. No Android user has ever seen them.

### What changed

`assets/app_icon.png` and `assets/icon.ico` were the last PicoClaw lobster
artwork in the tree, and they were shipping: 149,251 bytes of another product's
crustacean inside every APK, unread. They are now derived from the same
canonical APERTURE geometry as the launcher and Core's own favicon.

    before  app_icon.png  512×512 RGBA   104,580 bytes  PicoClaw FUI lobster
            icon.ico      1 frame, 256    44,671 bytes  PicoClaw FUI lobster
    after   app_icon.png  512×512 RGBA    20,797 bytes  APERTURE rounded tile
            icon.ico      7 frames 16–256 23,078 bytes  APERTURE rounded tile

105,376 bytes smaller in every APK, as a side effect rather than the goal.

The ICO gained the frames Windows actually asks for. A single 256px frame — what
the lobster file had — leaves a 16px tray request to be downscaled at draw time
by whoever is asking; rendering the frames from the geometry is the only way
they are anti-aliased rather than resampled from a bitmap.

### Source of truth

`tool/generate_android_launcher_icons.py` was extended rather than joined by a
second script, and it still imports the geometry from
`core/src/web/frontend/scripts/generate-brand-assets.py` read-only. The
composition is the canonical tab-icon idiom — `icon(size, radius_fraction=0.22,
scale=0.66)` — taken from the brand module's own use for `favicon.ico`, not
invented here. There is still exactly one place that decides what the mark looks
like and exactly one that decides how it is framed. The full chain is documented
in `docs/ASSET_INVENTORY.md`.

The generator gained a `--check` mode: it regenerates every artifact in memory
and compares bytes, writing nothing. That is what the guard runs, so proving the
assets are derived needs no perceptual image comparison — either the bytes match
or the tree is stale. Verified in both directions: `--check` passes on the
committed tree and fails on a deliberately perturbed asset.

### The re-audit says N0 was right

Every tracked raster and vector outside `core/src` — 20 files — is now either
APERTURE-derived or PocketClaw-original. Those two were the last stale ones.
`licenses/sipeed-picoclaw-fui-MIT.txt` and the `THIRD_PARTY_NOTICES.md` entry
**stay**: the FUI licence covers the origin of the Flutter application itself,
not merely those two images.

### The standing rule

`assets/app_icon.png` and `assets/icon.ico` are **generated artifacts**. Do not
hand-maintain them, do not hand-edit them, and do not define APERTURE strokes or
geometry anywhere outside
`core/src/web/frontend/scripts/generate-brand-assets.py`. The repo-level
generator stays deterministic and offline-capable.
`test/unit/brand_asset_contract_test.dart` enforces all of that, including that
the generator never grows its own `STROKES` table.

Final accepted artifacts:

    assets/app_icon.png  512×512 RGBA, transparent  20,797 bytes
                         354e1749b138bb7f199b86e3cbadc16dd51ec83dd75963557bc32e39aa34467d
    assets/icon.ico      7 frames, 16–256, RGBA     23,078 bytes
                         4d48c41354557804b7696dd818ddb8eec25fe353a6ef2856ef10b1df65c36b8f

The historical lobster digests stay pinned in the guard as values these files
must never carry again, so restoring either from git history fails a test.

## Namespace Migration N1 — CLOSED

- Status: **closed on `feature/namespace-n1-source-identities`, 2026-09-09,
  merged to `develop` with `--no-ff`.** Branch cut from `develop` at `75377f8`,
  retained. `main` untouched, no tags moved, no release. No version bump and no
  physical candidate: `0.2.0+59`, baseline 59, no What's New entry.
- **One test-only `core/src` change**, authorized as a narrow amendment:
  `android_hot_reload_test.go` follows the renamed Android service filename.
  Plus the A3 build-input alignment below. The Core **source fingerprint is
  unchanged** at `e7acbff7000bb58ac8074bfaf290528326df115bf1fae4bb6242defbf78d9a23`
  — shipped program logic is identical to the accepted vc59 Core — but the Core
  was rebuilt and restaged once because the *build contract* changed. No APK, no
  ADB, no physical candidate.

### Renamed — PocketClaw-owned source identity with no persistence

| From | To |
|---|---|
| `lib/src/core/picoclaw_channel.dart` | `pocketclaw_channel.dart` |
| `PicoClawChannel` | `PocketClawChannel` |
| `com.lord1egypt.pocketclaw/picoclaw` | `com.lord1egypt.pocketclaw/pocketclaw` |
| `PicoClawApp.kt` / `PicoClawApp` | `PocketClawApp.kt` / `PocketClawApp` |
| `PicoClawService.kt` / `PicoClawService` | `PocketClawService.kt` / `PocketClawService` |
| `PicoClawMethodChannel.kt` / class | `PocketClawMethodChannel.kt` / class |
| `PICOCLAW_ANALYTICS_PROVIDER` | `POCKETCLAW_ANALYTICS_PROVIDER` |
| `PICOCLAW_UMENG_*` (5 defines) | `POCKETCLAW_UMENG_*` |
| `PICOCLAW_FIREBASE_*` (5 defines) | `POCKETCLAW_FIREBASE_*` |

The MethodChannel needed no alias: both ends ship in one APK and upgrade
together. The dart-defines needed no legacy fallback either — no tracked
workflow supplies them, and `POCKETCLAW_ONBOARDING_BASE_URL` had already set the
naming precedent.

### RESOLVED — the A3 build-input rule now matches the fingerprint rule

Both blockers are closed. `5c81160` updated the two path constants in
`android_hot_reload_test.go`; `c4fbe02` then aligned the A3 rule that the first
fix exposed.

**The rule, stated once:** Core `*_test.go` files are excluded from **both** the
Core source fingerprint and the default BuildTime history query, because they
cannot change the shipped binary. Everything that does take part in producing it
stays provenance-bearing — production Go source, config, embedded assets,
`core/build-android-arm64.sh`, and `core/resolve-build-time.sh` itself. Editing
any of those still moves the timestamp and still requires a rebuild.

The resolver's self-provenance is the load-bearing half. Exempting it from its
own query would have made the alignment commit free, which is precisely why it
was not done: a dating rule that does not date itself is how provenance quietly
stops meaning anything. `TestBuildTimeInputRules` pins all six cases
independently, and removing the exclusion fails exactly the two that should fail.

The rebuild that followed is the honest cost of that choice:

    build-input commit e7acbff7-source, resolver commit c4fbe02
    epoch              1788901472
    BuildTime          2026-09-08T21:04:32+0000
    fingerprint        e7acbff7000bb58ac8074bfaf290528326df115bf1fae4bb6242defbf78d9a23  (unchanged)
    libpicoclaw.so     af5ebc9244c716e139906a9eeddb9c24340cf1ac5ebac3d229e76106deeabbf2  37,683,553
    libpicoclaw-web.so c829cc90c8d3ddb1938935cb06350fe11ea7a25c9e1eed9f9d4255fdd015676e  25,493,857

Both stripped, NX stack, 64 KiB aligned, zero developer paths, zero Go VCS
stamps. Staged at `5c0a52a`; resolving before and after that commit gave the
identical value, so staging still does not redate the build.

The upstream divergence patch was deliberately **not** regenerated: it was last
written 2026-08-30 and already predates A2, A3 and Bootstrap, so regenerating it
here would sweep unrelated history into N1. That remains a separate
upstream-review concern.

### Superseded — the original blocker

`5c81160` updated the two path constants, authorized as a narrow N1 amendment.
Both guards pass, the Core fingerprint is unchanged at `e7acbff7…` and staged
freshness still passes — the `_test.go` fingerprint exclusion held exactly as
predicted.

**But the production source gate is now red on `core.staged_build_time`.**
`core/resolve-build-time.sh` scopes `BUILD_INPUTS` to `core/src` as a whole,
while `coresource/fingerprint.go:159-161` excludes `_test.go`. A four-line test
edit therefore moves the build-input epoch from `08781fe`/`2026-09-08T18:36:44`
to `5c81160`/`2026-09-08T20:52:05`, and the gate compares that against staged
binaries that provably cannot differ. Two A3 guards now disagree about the same
tree.

Excluding `':!core/src/**/*_test.go'` from the `BUILD_INPUTS` query restores the
epoch and turns the gate green — verified by hand, not applied, because
`resolve-build-time.sh` is itself a canonical build input and an A3 contract.
**N1 is therefore NOT closed and NOT merged**: the closeout was conditional on a
green production source gate. See `TASKS.md`.

### Original blocker — two guards inside `core/src` named the old Kotlin file

`core/src/pkg/coresource/android_hot_reload_test.go:38,104` hard-code
`.../service/PicoClawService.kt` and `t.Fatalf` when it cannot be read. After
the rename both `TestAndroidManagedGatewayEnablesHotReload` and
`TestAndroidManagedGatewayDisablesHostBusTools` fail with "no such file or
directory".

The file is **PocketClaw-authored**, not upstream — it appears nowhere in
`core/pocketclaw-core-v0.3.1.patch`. The fix is two path constants. It is also
**fingerprint-neutral**: `coresource/fingerprint.go:159-161` excludes `_test.go`
deliberately, so the edit needs no Core rebuild and no restage.

It was not made, because N1's scope says `core/src` must not be touched and
that widening the phase silently is not acceptable. **This needs an explicit
decision before N1 can be merged**, since the branch currently leaves those two
guards red.

### Deliberately preserved

Upstream identity (`github.com/sipeed/picoclaw`, `cmd/picoclaw`,
`BINARY_NAME=picoclaw`, `picoclaw-launcher`); Core runtime env
(`PICOCLAW_HOME`, `_CONFIG`, `_GATEWAY_TOKEN_FILE`, `_LOG_DIR`,
`_DASHBOARD_AUTH_DIR`, `_CHANNELS_PICO_TOKEN`, `_DNS_SERVER` and the ~200
upstream tags); native binaries `libpicoclaw.so` / `libpicoclaw-web.so`;
persisted and wire values `.picoclaw.pid`, `filesDir/picoclaw/`,
`picoclaw_foreground`, `picoclaw_launcher_auth`, `"pico"`, `"pico_client"`,
`/pico/*`, `"pico-user"`; user state; the historical digest `47b63011…`; all
legal and provenance strings; and `wecomQRSourceID` / IRC `nick`.

The desktop adapter keeps `picoclaw-launcher` and `picoclaw`: those are the
filenames `core/src/Makefile` actually produces, so they are artifact lookups
rather than our identity, and a comment now says so in place.

`test/unit/namespace_n1_boundary_test.dart` asserts both halves — what N1
renamed and what it must not have touched. A blanket "no picoclaw anywhere"
guard would be wrong by architecture and is deliberately absent.

## Bootstrap Architecture — PHYSICALLY ACCEPTED and CLOSED on vc59

- Status: **closed on `feature/bootstrap-architecture`, 2026-09-08. Merged to
  `develop` with `--no-ff`.** `main` untouched, no tags moved, no release.
  Branch cut from `develop` at `01495dc`, retained.
- **Physically accepted as vc59** (`0.2.0`, versionCode 59) on SM-A165F /
  Android 16, installed with `adb install -r`. No uninstall, no clear-data;
  UID, dataDir and firstInstallTime preserved. `lastAcceptedVersionCode`
  advanced 58 → 59 in the acceptance commit. `pubspec.yaml` stays `0.2.0+59`.
  No What's New entry: this fixes prompt and bootstrap ownership.

### The accepted artifact

    APK            44101679d94a97ad12b96dd756afb5dd412fe53fa636aee0dbf780cbbae1ffa3
                   63,565,050 bytes, com.lord1egypt.pocketclaw 0.2.0 (59)
    local signer   15cf75f9945d5354e75707e0326b7cffc60ac51a68df38156db318ef4578a27c
    fingerprint    e7acbff7000bb58ac8074bfaf290528326df115bf1fae4bb6242defbf78d9a23
    BuildTime      2026-09-08T18:36:44+0000  (build-input commit 08781fe)
    libpicoclaw.so 0a4d9c856d0d4a260349cdee8b3ac0c0be8fff2f2ee9fb23952f82c8c62cdef1  37,683,553
    libpicoclaw-web.so 7f693fd0de6f5e6bb32df986b804b5adfbb012961c74596ee1e005dbdb47a8a0  25,493,857

The signer is the local-test development key, by explicit
`-PallowDebugSigning=true`. vc59 is a physically accepted **local test**
artifact, not a production release artifact; no production signing material was
created.

### Physical acceptance evidence

Machine checks, all pass: app, launcher backend and Core gateway all started;
the gateway is loopback-only on `127.0.0.1:18790` and `[::1]:18790`; the shared
PID record carries exactly `host`, `pid`, `port`, `version` and no credential;
application identity and the 10-permission set were preserved across the update.

The decisive evidence is what did **not** happen. The workspace at
`/sdcard/Download/pocketclaw/workspace` already held `AGENT.md` (2026-09-03),
`SOUL.md` and `USER.md` (2026-08-26) and `memory/MEMORY.md` (2026-09-08 00:38).
Every one of those timestamps predates the vc59 run, which wrote its bootstrap
record at 22:06:18. The app started, recorded what it had done, and modified no
user file. No file content was read during validation.

The record it wrote has `bootstrapVersion` 1, `managedGuidanceVersion` 1, and an
**empty** `templates` map, because this run seeded nothing. An empty record on a
populated workspace is the correct answer: PocketClaw must not claim provenance
for files it did not write.

Chat acceptance confirmed the managed guidance actually governs behaviour: the
agent treats `action=list` as the authority on what exists, derives tool
availability from the runtime inventory rather than from a Skill naming a tool,
refuses to download or install a missing tool, and reports unavailability
plainly. No-shell semantics were confirmed too — `|`, `>`, `*` and `$(...)` are
not interpreted, arguments pass verbatim, and compound work is split across
calls.

A2 invariants re-checked and intact: no `launcher-auth.db` (or `-wal`, `-shm`,
`-journal`) on shared storage at either the home or workspace root, no shared
`logs/` or `gateway.log`, private runtime state still starting correctly.

### Non-blocking security-review backlog

The launcher/web console was observed listening on `0.0.0.0:18800`, where the
Core gateway is correctly loopback-only. **This predates this branch and was not
introduced by vc59** — the diff against `develop` touches nothing under
`core/src/web`, `core/src/pkg/config` or `android/app/src`. Recorded for a later
security review; deliberately not fixed in this closeout.

### What moved, and why it had to

Seeding writes a bundled template only when the file is absent, so `AGENT.md`
is written once and never refreshed. That rule is right — the file is the
user's — and its consequence is that an install seeded before a default improved
keeps the old text forever. The device validated on 2026-09-08 still carried an
`AGENT.md` predating the Managed Runtime, so that agent had never been told the
Managed Runtime exists.

Guidance now splits by **owner** rather than by topic:

| Kind | Where it lives | Lifecycle |
|---|---|---|
| `AGENT.md`, `SOUL.md`, `USER.md` | workspace | seeded once, user-owned, never rewritten |
| `memory/MEMORY.md` | workspace | seeded once, never read or migrated by bootstrap |
| skills and their assets | workspace | product content, replaceable, not tracked |
| capability guidance | the binary | upgrades with the app, nothing to migrate |

The Managed Runtime section is the first to move. It is `capability.managed_runtime`,
contributed by the new `runtime.managed_guidance` prompt source at
capability/tooling, and it has been removed from the seeded template so a fresh
workspace does not receive a second copy that would then age on its own.

Being a prompt part rather than a file also made it conditional, which it never
was before: a sub-turn restricted to a tool set without `runtime` no longer
receives instructions to ask the runtime first. An unrestricted caller always
does.

### Existing installs keep their copy on disk and lose it from the prompt

An install seeded before this change still has PocketClaw's own Managed Runtime
text inside its `AGENT.md`. The assembler drops that section from the prompt —
never from the file — and only when it hashes to exactly what PocketClaw seeded
(`47b63011…`). One edited character and the user's version is kept, with the
managed part alongside it, because at that point it is their instruction. A
same-named section the user wrote themselves is never touched, which is why the
match is by digest and not by heading.

### Measured prompt impact

Estimated with the repository's own heuristic (2.5 characters per token).

    managed guidance part                              1563 chars   624 tokens
    install that already carried PocketClaw's copy     1263 -> 1058  -205 tokens
    fresh install, old template vs new                 1711 -> 1506  -205 tokens
    install that never had the guidance                 430 -> 1058  +628 tokens

The first two are the same install shape seen twice: the old inline copy is
dropped and the shorter managed part replaces it, so those installs get smaller.
The last is the whole point — those installs were missing the guidance
altogether.

The guidance went through one editorial pass: 2081 chars to 1563 (-25%), 831
estimated tokens to 624. The prose that went was framing and repetition — "Four
things to hold on to", a seven-row markdown table restating what `action=list`
returns, and a second sentence saying again that a tool absent from PATH may
still exist. Every operational rule survived, and a test pins each one by
substring so a future pass cannot quietly drop one. What deliberately stayed is
the short reason that installing is impossible: without it a model treats
"unavailable" as an obstacle to work around and burns a turn trying.

### Prompt order, and who wins a disagreement

The assembled system prompt, asserted in `TestSystemPromptPartOrder` rather than
merely described, because the order is part of the mechanism:

    1  kernel       identity       runtime.kernel             kernel.identity
    2  instruction  workspace      workspace.definition       instruction.workspace
    3  capability   tooling        runtime.managed_guidance   capability.managed_runtime
    4  capability   skill_catalog  skill:index                capability.skill_catalog
    5  context      memory         memory:workspace           context.memory
    6  context      output         runtime.output             context.output_policy.split_on_marker

The managed guidance sits at 3: after the workspace text it must outrank, and
before the skill catalog, whose skills may name tools that do not exist on this
device.

Ownership is split, and the split is narrow:

| Owned by the workspace | Owned by the managed guidance |
|---|---|
| persona and tone | what the runtime currently provides |
| the user's preferences | which bundled and system tools exist |
| the user's own operating instructions | what this build can and cannot do |

When a user's `AGENT.md` carries a stale capability claim — "PocketClaw has no
`jq`", "install what you need with apt" — nothing of theirs is edited or
removed. The current facts follow their text and say so explicitly: *these facts
describe the build you are running and are authoritative for what this device
can do … Persona, tone and the user's preferences remain the workspace's.* Both
halves are tested, including one that fails if the guidance ever acquires
broad-override language like "ignore the workspace".

### The bootstrap record

`.pocketclaw/bootstrap.json` stores, for each tracked document, the digest of
**what PocketClaw wrote** rather than of the file as it now stands. That makes
three states distinguishable without diffing prose or retaining every historical
template: recorded and matching (ours, untouched), recorded and differing (the
user edited it), unrecorded (provenance unknown — assume the user's). It records
only files a run actually wrote, so a workspace that already had `AGENT.md` gets
no entry rather than a false claim, and a rerun that writes nothing leaves the
file byte-identical.

**The record is advisory. It is not an authorization boundary.** It lives in the
workspace, which the user can edit and which on Android may sit on shared
storage, so it is untrusted input. It may inform a non-destructive migration, an
offer to upgrade, a guess that a default is untouched, or a diagnostic. It may
never, by itself, authorize overwriting or deleting a user-owned file — anyone
who can edit the record can make any document look pristine by recording the
digest of its current contents. That is not an escalation, since they could edit
the document directly, but it must not become a way to make PocketClaw destroy
the document for them. No function in the package returns "you may overwrite
this", and none should be added.

`bootstrap.Provenance` reports one of three states and `UserOwns` is its safe
reading. Every ambiguity collapses to hands-off: no record, an unreadable or
malformed one, an empty file, a JSON array where an object belongs, a schema
version this build does not know, a missing or empty entry, a digest that does
not match, an unreadable document, an untracked path. Nine of those are pinned
by name in `TestAmbiguousProvenanceAlwaysMeansHandsOff`; a corrupt record is
also never silently rewritten, because a record we cannot read is exactly when
we know least.

Neither function has a production caller yet, deliberately: the upgrade
experience is still deferred in `TASKS.md`, and what is settled here is the
record it will consult and the rule it must obey. `MEMORY.md` reports
`ProvenanceUnknown` even when its recorded digest matches the file on disk, so
it cannot become an upgrade candidate even if a later change forgets that it
must not.

### RECHECK AFTER FULL NAMESPACE MIGRATION

- `.pocketclaw/` and `bootstrap.json` — new state, already PocketClaw-named on
  purpose, so this is the one piece of on-disk state the migration does not have
  to rename. Verify nothing later re-derives it from the package id.
- Prompt source id `runtime.managed_guidance`, part id `capability.managed_runtime`.
- Tracked template names `AGENT.md`, `SOUL.md`, `USER.md`, `memory/MEMORY.md`.
- The superseded digest `47b63011a55eaa659470f2ab09d05532e9942800848020a1cfe443e6f21aca76`
  pins text containing the word PocketClaw. It describes bytes already on users'
  devices and must **not** be regenerated to match a renamed string. The text it
  digests is now pinned as the literal `legacyManagedRuntimeSectionV1` in
  `core/src/pkg/agent/managed_guidance_legacy.go`, extracted from git history
  rather than derived from the live guidance — deriving it was a latent bug that
  the first editorial pass would have triggered, silently redefining "the bytes
  on a user's device" to mean the current wording.
- The managed guidance text names the `runtime` tool and `action=list`, plus
  `git`, `gh`, `rg`, `jq`, `sqlite3`, `curl` — user-visible product surface.
- Go import path `github.com/sipeed/picoclaw/pkg/bootstrap`, which carries the
  upstream module name like every other package.
- `$PICOCLAW_HOME` and its `workspace/` subdirectory, the shared PID record
  `.picoclaw.pid`, and the staged library names `libpicoclaw.so` /
  `libpicoclaw-web.so` — all compatibility PicoClaw identifiers that remain.

`.pocketclaw/bootstrap.json` is already PocketClaw-named and must not be
gratuitously renamed; it is the one piece of new on-disk state the migration
does not have to touch.

## Production Release Hardening A3 — CLOSED

- Status: **closed on `feature/release-hardening-a3`, 2026-09-08. Merged to
  `develop` with `--no-ff`.** `main` untouched, no tags moved, no release.
- **No physical candidate and no version bump are associated with A3.** It
  changes build and release engineering, not runtime behaviour, so there is
  nothing a device could validate. `pubspec.yaml` stays `0.2.0+58` and
  `lastAcceptedVersionCode` stays `58`; vc58 remains the accepted **A2**
  artifact and is not an A3 artifact.
- Branch from `develop` at `b68f86d`. Retained, not deleted.
- Two areas: deterministic Core builds, and a release gate.

### The deterministically-built Core is staged

The first Core built under the new contract, with `SOURCE_DATE_EPOCH`
deliberately **unset** so it exercises the default path rather than a special
case:

    build-input commit e9e68d5979a3737ddfaba75fe24082d02cf4c7a6
    epoch              1788884697
    BuildTime          2026-09-08T16:24:57+0000
    source fingerprint 5d6f00cd381d1c792a956642dd4746056cdd86f12942685b0f512bde55aacbb9
    libpicoclaw.so     714d7a126309697c881d7e6564e3cf97ac878b567965c1d789c7137a6b1eca0c  37683553 bytes
    libpicoclaw-web.so c6376b6dee76116eebbf037fd0af3b0f8b6e2b4e0d8117fa976d15624bd331b4  25493857 bytes

Both binaries carry that same BuildTime — the property the resolve-once design
exists to guarantee. Staged freshness passes, both are stripped,
non-executable-stack and 64 KiB aligned with zero developer-machine paths, and
the fingerprint is stamped in `libpicoclaw.so`. The launcher does not carry that
stamp by design: the fingerprint describes the Core gateway's source, and the
build script asserts it only where it belongs.

Both binaries also carry **no toolchain VCS stamp**, which is the second half of
the guarantee and the part that was missing until the final verification pass —
see below.

**Staging proved the point it was meant to, and then proved a stronger one.**
Resolving the BuildTime before and after the commit that staged those binaries
gave the identical value, in the real repository: build output is not a build
input, so staging cannot redate the build that produced it. Then the rebuild
from that moved HEAD — two commits past the build-input commit — reproduced both
binaries **byte for byte**. That is the comparison that matters, because it is
the one a person reproducing a release actually performs, and it is the one that
failed before `-buildvcs=false`.

### The Core build is reproducible

`core/src/Makefile` derived `BUILD_TIME_RAW` from `date`, so identical source
produced different binaries purely because the clock had moved. The source
fingerprint was unaffected — it is content-addressed and excludes anything
time-varying — but nobody could reproduce a released artifact byte for byte.

`core/resolve-build-time.sh` is now the only thing that decides that value. It
takes `SOURCE_DATE_EPOCH` when given and validates it, and otherwise uses **the
most recent commit that touched a canonical Core build input**. Not HEAD: HEAD
moves for documentation, staged binaries and unrelated application changes, so
dating from it would give identical Core source a different timestamp on the
next unrelated commit and quietly undo the guarantee. With neither available it
**fails** rather than falling back to the wall clock, which is the failure mode
that would be hardest to notice. Output is `%FT%T%z` fixed to UTC, so the
historical format is preserved and the builder's timezone cannot change it.

The build-input path set is explicit and documented in the script:

    core/src
    core/build-android-arm64.sh
    core/resolve-build-time.sh

Deliberately broader than the Core *source fingerprint*, which names only
`cmd/`, `pkg/`, `workspace/`, `go.mod`, `go.sum` and the `Makefile` because
those are what reach the Core gateway compiler. The canonical build also
produces `libpicoclaw-web.so` from `core/src/web` and stamps both binaries with
one timestamp, so `web/` materially affects the bytes being dated. The two sets
answer different questions and are allowed to differ. Deliberately excluded: the
staged JNI binaries (build output — folding them in would make every staging
commit redate the build that produced them), documentation, the acceptance
baseline, and the Flutter application. Over-inclusion is safe here and
under-inclusion is not.

`core/build-android-arm64.sh` resolves once and passes `BUILD_TIME=` explicitly
into both make invocations, so the gateway and the launcher cannot carry
different timestamps. That detail matters: `BUILD_TIME_RAW` was a `:=`
assignment, which Make does **not** let an exported environment variable
override — only a command-line assignment works. The Makefile's own default now
calls the same resolver through a recursive `=`, so a bare `make` is
deterministic too and the subprocess is skipped entirely when the value is
passed in.

**Proven, not asserted — and the first two proofs were not enough.** Two
consecutive builds at a fixed `SOURCE_DATE_EPOCH` matched, and two builds on the
default path 65 seconds apart matched. Both pairs ran at the same HEAD, so both
were blind to the toolchain's VCS stamp. The proof the contract now rests on is
the one that crosses a commit: build, commit the binaries, rebuild from the
moved HEAD, and compare. That produced
`714d7a126309697c881d7e6564e3cf97ac878b567965c1d789c7137a6b1eca0c` and
`c6376b6dee76116eebbf037fd0af3b0f8b6e2b4e0d8117fa976d15624bd331b4` on both
sides of the staging commit, byte for byte.

Commit scoping is proven against real git history rather than by reading the
script: a temporary repository commits a build input, then documentation, an
acceptance baseline, a staged binary and an application file, and asserts the
resolved epoch does not move — then commits a build input again and asserts it
does. The staging-commit case has its own test, because that is the shape that
actually occurs.

### Two ways the guarantee was still escaping

Both were found by verification rather than by review, and both were silent.

**The Go toolchain was stamping HEAD in behind the resolver.** `go build` writes
`build.vcs.revision`, `build.vcs.time` and `build.vcs.modified` into every binary
automatically, reading the enclosing repository's HEAD. Identical build inputs
therefore produced different bytes after any unrelated commit — including the
staging commit itself, so a staged Core could never be reproduced from the
commit containing it. The earlier byte-identity proofs had passed because both
builds in each pair ran at the same HEAD, which held the stamp constant.

The canonical Android recipes build with `-buildvcs=false`, through a named
`REPRODUCIBLE_BUILD_FLAGS` so a later edit cannot drop it by writing `-trimpath`
back in. Provenance is unaffected: the build pins `-X config.GitCommit` and
stamps the source fingerprint, both verifiable, so the toolchain's copy was
redundant before it was harmful. Dropping the flag is invisible — the binary
still builds, runs, and carries the right fingerprint and BuildTime — so it is
enforced twice, by a test on the recipe and by `core.no_vcs_stamp` on the
artifact.

**A shallow clone silently restored the unscoped-HEAD behaviour.** Git treats a
shallow graft boundary as a root commit, so every path looks introduced by the
tip and the path-scoped query returns the tip's timestamp. The release-gate
workflow checked out at `fetch-depth: 1`, so CI would have dated every build by
whatever documentation or merge commit it was running on. Worse than the non-git
case, which fails loudly: this succeeded with a plausible wrong answer. The
resolver refuses a shallow clone now — an explicit `SOURCE_DATE_EPOCH` still
works there — and the workflow fetches full history.

### The release gate

`tool/release_gate.py` is one command that decides whether an artifact is
releasable, so the answer does not depend on who is asking. It delegates to the
authoritative guards — the staged-Core freshness test, the A1 signing and
version contracts, the A2 private-storage guards, the Gradle payload verifiers —
rather than restating their logic, because a second implementation of a rule is
a second thing to get wrong.

    tool/release_gate.py --verify-source
    tool/release_gate.py --verify-artifact <apk>
    tool/release_gate.py --full <apk>
    --release-class test|production   --manifest <path>   --no-tests

Two signing classes, chosen by the caller and never guessed: `test` permits the
known development signer and can never report a production release; `production`
treats that signer as an unconditional failure. Verified both ways on the
accepted vc58 artifact — PASS as `test`, FAIL as `production`.

Two further gates close the gaps that recording inputs alone would leave.

**Clean worktree.** Verifying an artifact built from uncommitted edits proves
nothing about anything anyone else can obtain, and the build timestamp is
derived from committed history — so a dirty tree can produce bytes whose inputs
no longer exist. Git's own porcelain status decides what is dirty, so ignored
caches stay ignored without a second rule. Production fails; a test build is
classified NON-RELEASABLE.

**Embedded BuildTime.** The gate reads the timestamp actually stamped in the
packaged Core and compares it against what a canonical build of this tree would
produce. Recording only the *input* would have missed the entire class of
failure this milestone is about — a binary built before the contract, or by a
`make` that fell back to `dev`, looks fine from outside. A mismatch fails
production and is reported as LEGACY/NON-RELEASABLE for a test artifact;
`BuildTime=dev`, an unreadable stamp, or an expected value that could not be
computed all fail production. `--full <apk> --release-class production` can
never skip it.

Running it against vc58 demonstrated the point: that artifact's embedded stamp
is `2026-09-08T06:11:35+0300` — in local time, which is exactly the timezone
dependence the UTC fix removes — against an expected `2026-09-08T04:02:42+0000`.

Every check reports PASS, FAIL or SKIPPED, a failure names expected and
observed, and the exit code is non-zero when any selected check fails. The JSON
release manifest records package id, version, APK hash, Core fingerprints and
hashes, `coreBuildInputCommit`, `buildTimeExpected`/`buildTimeObserved`/
`buildTimeDerivation`, ABI, signer fingerprint, permission contract, result and
classification — and no secrets or machine paths.

### What the gate found

Writing it surfaced two things worth recording rather than hiding. The packaged
`libapp.so` embeds one generated-source URI, which is exactly the tracked
"controlled Dart generated-source URI strategy" item; the gate reports it as
`PENDING_FINAL_HARDENING` rather than failing every build on it or passing
silently. And `libpocketclaw-gh.so` carries `/home/runner/work/` paths from
upstream's own CI — not this machine's, and not ours to fix — so the strict
zero-developer-paths rule is scoped to Core, where the build script already
enforces it and where it holds.

### The source gate passes, with nothing unexplained

From the clean committed branch, `python3 tool/release_gate.py --verify-source`
exits **0** with 15 checks PASS and no SKIPPED: clean worktree, tracked version,
accepted baseline, no `local.properties` version identity, three lockfiles,
deterministic build-time contract, staged-Core embedded BuildTime, developer-path
guard, staged-Core freshness, the reproducibility tests, and the delegated A1
and A2 guards. `--release-class production` also passes in source mode.

Two gate semantics were tightened to get there. **Production requires Git
provenance:** a non-git checkout used to be SKIPPED in every mode, which is fine
for inspecting an artifact locally and wrong for a release — outside a worktree
there is no revision, no cleanliness and no build-input commit, so there is no
way to say what a canonical build would have produced. And the gate now verifies
the **staged** Core's embedded BuildTime, not only a packaged one: the
fingerprint answers "is this the right content" but not "was it built from the
inputs currently committed".

### The WhatsApp guard self-conflict is resolved

`TestNoUserFacingWhatsAppSurface` had been red since vc46. The cause was not a
product regression: it scans `test/` for the word outside a comment, and
`whats_new_page_test.dart` declares a `forbiddenSubstrings` list naming WhatsApp
precisely in order to forbid it in release notes. One guard was reading another
guard's prohibition as a violation of that same prohibition.

Test-only fix: a file may declare itself enforcement data with an explicit
`WHATSAPP-GUARD-ENFORCEMENT-DATA` marker, and the surface scan skips only files
carrying it. Opt-in and greppable rather than a blanket `test/` exemption, so
marking a real product file would be a visible act a reviewer would question.
No production code changed and the prohibition is not weakened — regression
tests hold that a genuine surface is still detected, an *unmarked* file naming
WhatsApp is still a violation, a comment recording the removal is still not a
surface, and **exactly one** file in the tree may claim the exemption.

## Production Release Hardening A2 — PHYSICALLY ACCEPTED AND CLOSED

- Status: **PASS on a physical Android device (SM-A165F / Android 16), 2026-09-08
  as vc58. Merged to `develop` with `--no-ff`.** `main` untouched, no tags moved,
  no release created.
- Branch `feature/release-hardening-a2`, from `develop` at `e62f083`. Retained,
  not deleted.
- Five areas: the gateway credential, log placement and rotation, secret
  redaction, realtime authentication, and the Dashboard credential verifier.

### Physical acceptance evidence

    versionName 0.2.0, versionCode 58, arm64
    APK  a039dde854c1199f54f118a5e2f40827eecb7fc2b4a066f6d2d9db6448250e95
    source fingerprint 3a9ae19c12041ff104f1344081dc3e645553791e2e84d6e603ee503ec035f06d
    libpicoclaw.so     e42677a25caf2498c74dcd3cbfac4d0b4b700177706977b4b9042d23d8771164
    libpicoclaw-web.so b500427ec6cf6927671ab87fa83e8fea75e83bdc6247568670ee17f3ccc51779

Installed with `adb install -r` — no uninstall, no clear-data — preserving
install time, dataDir, uid and application data.

| Observed | Result |
| --- | --- |
| Shared `.picoclaw.pid` carries only pid, version, port, host | PASS |
| No credential key or credential-shaped value in the shared record | PASS |
| Gateway credential reached the private no-backup contract | PASS |
| No production Gateway log under `Download/pocketclaw/logs` | PASS |
| The three legacy shared logs removed, directory removed | PASS |
| No new shared `gateway.log` recreated while running | PASS |
| Shared `launcher-auth.db` and all exact sidecars retired | PASS |
| The existing Dashboard password still authenticates | PASS |
| Detailed Status metrics populate | PASS |
| Runtime logs still visible in the app after the file moved | PASS |
| Workspace present and user-accessible, contents untouched | PASS |

The Dashboard login is the load-bearing one: it is the end-to-end proof that the
verifier survived the move from shared to private storage. A migration that had
silently reset or lost it would have failed exactly there.

**vc57 was superseded, not accepted.** It proved the gateway credential, log
placement and legacy log cleanup, but the Dashboard verifier move landed after
it, so the baseline advances directly 56 → 58 and no acceptance record exists
for vc57.

### The final invariant

On Android:

| Shared, user-accessible | App-private, no-backup |
| --- | --- |
| `Download/pocketclaw/workspace` | Gateway bearer credential |
| `.picoclaw.pid` — safe discovery metadata only | Gateway persistent diagnostic logs |
| | Dashboard credential verifier database |

### The invariant this milestone establishes

**User data stays where the user can reach it. Runtime control state does not.**
`Download/pocketclaw/workspace` is deliberately user-visible and is unchanged.
Three things moved out of that directory, none of them user content: the gateway
bearer credential, the diagnostic log, and the Dashboard credential database.

On Android, all security-sensitive runtime and auth state is now app-private and
no-backup:

| State | Location |
| --- | --- |
| Gateway bearer credential | `noBackupFilesDir/gateway_auth` |
| Gateway diagnostic logs | `noBackupFilesDir/logs/` |
| Dashboard credential verifier | `noBackupFilesDir/auth/launcher-auth.db` |
| User workspace | `Download/pocketclaw/workspace` — unchanged, user-accessible |

### Gateway credential

Core still generates it per gateway start, so rotation is unchanged. When
`PICOCLAW_GATEWAY_TOKEN_FILE` is set the token is written to that path alone and
the shared `.picoclaw.pid` record is written **without** a token field; the
record keeps its discovery fields. Unset — desktop and server, where
PICOCLAW_HOME is already private — behaviour is exactly as before.

The Android host names that path under `noBackupFilesDir`, the same boundary the
realtime credential already used, and `HealthChecker` reads the credential from
there instead of parsing the pid record. The file holds the bare token and
nothing else, is 0600, and is removed when the gateway shuts down. A new start
never adopts a token left behind in an old shared record.

### Logs

`PICOCLAW_LOG_DIR` overrides the historical `PICOCLAW_HOME/logs` for both the
gateway and the launcher backend; the Android host points it at private
no-backup storage. Rotation was added where there was none. The active file is a
counting writer: it tracks bytes as it writes, seeded from the file's existing
size, and rotates when it crosses 2 MiB — within one process lifetime, not only
when the file is opened. A gateway is long-lived, and rotating only at startup
would have let it append past the threshold for as long as it ran, which is how
the 18 MiB file observed on a real install came about. Two retained generations,
oldest deleted, bounded at roughly 6 MiB; the record that crosses the threshold
completes in the old file so no line is ever split. The writer is mutex-guarded
for concurrent callers, and every rotation failure path is silent and non-fatal
because this code runs underneath the logger and cannot report a problem by
logging one. Files are created 0600 in a 0700 directory.

The in-app Logs screen is unaffected: it reads an in-memory 200-line buffer fed
from the child process's stdout, never the file. Nothing in Dart or Kotlin ever
read `gateway.log`.

Legacy shared logs are deleted once, after the runtime is up and writing to the
private directory. Only three exact filenames are matched —
`gateway.log`, `gateway_panic.log`, `launcher_panic.log` — nothing by pattern,
nothing recursive, and the containing directory is removed only if those were
all it held. Best effort, idempotent, and never fatal to service start. This is
application output, not user content: older builds wrote full LLM requests and
system-prompt previews into it.

### Redaction and realtime authentication

Central redaction gained three narrow rules — api-key headers, credential query
parameters, and unambiguous vendor prefixes — with a guard test proving ordinary
diagnostic text, session keys and fingerprints survive untouched. The realtime
channel now compares credentials with `subtle.ConstantTimeCompare`, matching the
health server, and refuses query-string authentication outright whenever the
credential was supplied by the host. The Dashboard toggle is hidden **only in
that case** — the backend omits `allow_token_query` from the realtime channel's
config response when `PICOCLAW_CHANNELS_PICO_TOKEN` is set, and the form already
renders the control only for a field the response carries. A self-managed
deployment, where a browser client genuinely cannot set a header, keeps both the
capability and the control.

### Dashboard credential database

Found by a source-only audit after vc57 passed, and fixed before A2 closes
because it is the same boundary and a stronger vector than the one already
fixed. `launcher-auth.db` holds a single bcrypt verifier (cost 12) — no
plaintext, no session token, and sessions are in-memory only, so **reading** it
grants nothing directly. **Writing** it is the problem: on shared storage an app
with storage write access can replace the verifier with one for a password it
chose, then authenticate normally over loopback, which Android does not isolate
between apps. That is an authentication bypass that never has to break bcrypt.

`PICOCLAW_DASHBOARD_AUTH_DIR` moves it; unset, the store stays under
PICOCLAW_HOME exactly as before, so desktop and server installs are unchanged.
Authentication semantics are untouched: same bcrypt cost, same verification,
same rate limiting, same 24-hour in-memory sessions, same login UX, no schema
change and no new crypto.

An existing password survives via a one-time migration that runs before the
store is opened. It refuses to overwrite an existing private database — private
state wins, so a rollback to attacker-controlled shared state is not one file
copy away — opens the legacy database through the real store first so SQLite
settles any journal a crashed writer left, copies to a temp file inside the
destination and fsyncs it, renames within that one filesystem (`os.Rename`
across `/sdcard` and app-private storage would be a cross-device error), then
validates the destination through the same store contract, and only then deletes
the legacy file. Every failure path leaves the legacy database intact for
recovery, and never resets the user's password.

**When the override is set, failure is fatal rather than a fallback.**
`PICOCLAW_DASHBOARD_AUTH_DIR` is a security boundary, not a preference: reopening
the shared store after a failed migration would re-arm exactly the
attacker-writable state the override exists to escape. So the launcher refuses
to start, naming the directory and the reason, with the legacy database left
untouched. Without the override — desktop and server — the historical
`picoHome` behaviour is unchanged.

**A validated private database also retires the shared one.** An existing
private database is validated through the store contract rather than trusted for
existing; once it opens, the superseded shared copy and its exact sidecars are
deleted best-effort, so no rollback artifact is left lying around. A cleanup
failure cannot move authority back, because the private store is already
authoritative. A private database that does *not* validate promotes nothing: the
legacy file is kept for recovery, the corrupt file is left as evidence, and
startup fails closed.

The plain file copy is safe because the store uses SQLite's default rollback
journal, not WAL — measured from the database header rather than assumed, and
asserted by `TestStoreUsesRollbackJournalAndLeavesNoSidecars`, which fails if
the mode ever changes and makes a main-file copy lossy.

### RECHECK AFTER FULL NAMESPACE MIGRATION

Every one of these is keyed on a compatibility name that migration will change:
`.picoclaw.pid`, `PICOCLAW_GATEWAY_TOKEN_FILE`, `PICOCLAW_LOG_DIR`,
`PICOCLAW_DASHBOARD_AUTH_DIR`, the legacy `launcher-auth.db` filename the
migration matches by name, `PICOCLAW_CHANNELS_PICO_TOKEN`, the `picoclaw`
private directory name already guarded for backup, and the `pico` channel name. A rename on one side only would
silently undo the separation without failing anything else.

### Core was rebuilt for vc58

Rebuilt through `./core/build-android-arm64.sh` and staged in its own commit.
The freshness guard passes, the packaged binaries are byte-identical to the
staged ones, both carry zero developer-machine paths, and both are stripped,
non-executable-stack and 64 KiB aligned.

### The accepted baseline advanced

`android/release-baseline.properties` moves `lastAcceptedVersionCode=56` to
`58` in this closeout — the commit that records the acceptance, which is the
only place it may move. `pubspec.yaml` stays at `0.2.0+58`: the candidate became
the accepted build, so the two now agree.

## Production Release Hardening A1 — PHYSICALLY ACCEPTED AND CLOSED

- Status: **PASS on a physical Android device (SM-A165F / Android 16), 2026-09-08
  as vc56. Merged to `develop` with `--no-ff`.** `main` untouched, no tags moved,
  no release created. Core source and the staged vc55 binaries are byte-for-byte
  unchanged and were not rebuilt for this milestone.
- Branch `feature/release-hardening-a1`, from `develop` at `941f45a`. Retained,
  not deleted.
- Four areas, nothing else: release signing, backup exclusion, version source of
  truth, and the unused analytics surface.

### Physical acceptance evidence

    versionName 0.2.0, versionCode 56, arm64
    APK  eea28fbe13c25e05f687d78e1a43c5ace40faea9ced63942479358ed4cef7b25

Built with `-Ptarget-platform=android-arm64 -PallowDebugSigning=true` and no
`-PversionCode` override; `android/local.properties` carries no version, so the
manifest's `versionCode=56` came from `pubspec.yaml` alone. That is the proof
the tracked source is authoritative. Installed with `adb install -r`, no
uninstall and no clear-data.

| Observed | Result |
| --- | --- |
| Release build fails closed without the signing opt-in | PASS |
| The opt-in announces debug signing in the build output | PASS |
| versionCode 56 / versionName 0.2.0 from the tracked source | PASS |
| Signing certificate identical before and after the upgrade | PASS |
| firstInstallTime, dataDir, uid and application data preserved | PASS |
| Installed `base.apk` hash matches the staged artifact exactly | PASS |
| Six permissions removed, none added | PASS |
| Packaged backup rules exclude `credentials/` and `picoclaw/` | PASS |
| Packaged Core byte-identical to the staged vc55 Core | PASS |

The upgrade kept its signing identity: the certificate was
`15cf75f9…` before and after, which is what made `install -r` a real in-place
update rather than a reinstall.

### Permissions: 13 declared, 11 requested on this device

The APK declares 13 `uses-permission` entries; the device reports 11. The two
missing are `WRITE_EXTERNAL_STORAGE` (`maxSdkVersion=28`) and
`READ_EXTERNAL_STORAGE` (`maxSdkVersion=32`), which Android drops on an API 36
device. vc55 showed the same two-entry gap (19 declared, 17 requested), so the
behaviour is unchanged — 11 is the intended set, not a shortfall.

On-device diff, vc55 to vc56 — six removed, nothing added:

    android.permission.READ_PHONE_STATE
    com.google.android.gms.permission.AD_ID
    android.permission.ACCESS_ADSERVICES_AD_ID
    android.permission.ACCESS_ADSERVICES_ATTRIBUTION
    com.google.android.finsky.permission.BIND_GET_INSTALL_REFERRER_SERVICE
    freemme.permission.msa

### The accepted baseline advanced

`android/release-baseline.properties` moves to `lastAcceptedVersionCode=56` in
this closeout, which is the commit that records the acceptance. `pubspec.yaml`
stays at `0.2.0+56`: the candidate became the accepted build, so the two now
agree, and the next build is the one that bumps pubspec again.

### Signing now fails closed

The release `signingConfig` used to fall through to the debug key whenever the
`KEYSTORE_*` environment was incomplete, silently. Every artifact to date,
vc55 included, is therefore debug-signed. It now resolves to the production
signer, or to the debug key **only** under `-PallowDebugSigning=true`, or to
`null` — and `validateReleaseSigning`, wired to `preReleaseBuild`,
`packageRelease` and `bundleRelease`, fails the build before anything compiles.
No production key was created: the device under test still runs a debug-signed
install, and switching signers would force an uninstall and data reset.

### Core secrets are out of Android backup

`files/picoclaw/` — `config.json` and the plaintext `.security.yml` holding
provider API keys and channel bot tokens — is now excluded from full backup,
cloud backup and device transfer, alongside the existing `credentials/`.
`allowBackup` stays `true` by decision; non-secret state remains restorable.

### The version is reproducible from git

`pubspec.yaml` moves from `0.2.0+13` to `0.2.0+55`, matching the physically
accepted build, and Gradle reads it directly. The Flutter Gradle plugin defaults
`flutter.versionCode` to 1 when the gitignored `local.properties` omits it, so a
clean checkout would have built versionCode 1; a version declared there is now
rejected with an error rather than silently obeyed. `-PversionCode` /
`-PversionName` are the explicit override and are validated against a floor of
55.

### Analytics attribution, measured rather than assumed

The Umeng SDK is `compileOnly` unless `POCKETCLAW_ANALYTICS_PROVIDER=umeng`, and
`READ_PHONE_STATE` is no longer declared. Merging the release manifest with and
without the dependency showed the SDK contributes **exactly one** entry,
`freemme.permission.msa`. `READ_PHONE_STATE` came only from our own manifest.

Correcting the audit: `AD_ID`, `ACCESS_ADSERVICES_AD_ID`,
`ACCESS_ADSERVICES_ATTRIBUTION` and `BIND_GET_INSTALL_REFERRER_SERVICE` are
**not** Umeng's. They come from Firebase Analytics via
`play-services-measurement`, they are unchanged by this milestone, and they were
left alone rather than removed on a guess. Firebase is also unconfigured by
default, so the same question applies to it — but its plugins are registered
from `pubspec.yaml` and removing them touches Dart, so it is its own decision
and is recorded in `TASKS.md`.

### Review follow-up, 2026-09-08

Three corrections after the implementation was accepted in principle.

- **Signing wording.** The build claimed the debug key's "private half ships
  with every Android SDK install". That is wrong and is gone. Debug signing
  material is local development material that differs between environments, and
  the practical consequence — an artifact signed with a different key is not an
  in-place update of an existing installation — is what the message now says.
  The fail-closed behaviour is unchanged.
- **The version floor advances.** `acceptedVersionCodeFloor` was a constant 55
  in the build file, which would still have accepted 56 after 120 shipped. It
  now reads `lastAcceptedVersionCode` from
  `android/release-baseline.properties`, advanced by hand in the commit that
  records a physical acceptance. Verified by setting the baseline to 120 and
  watching versionCode 55 and an override of 56 both be rejected.
- **Firebase kept, its advertising surface removed.** Traced the dependency
  rather than guessing: `firebase_analytics` and `firebase_core` are in
  `pubspec.yaml`, used only by `lib/src/core/firebase_device_reporter.dart`,
  behind the device-feedback Settings toggle. Firebase initializes only when all
  four `POCKETCLAW_FIREBASE_*` dart-defines are set; they are empty by default and
  there is no `google-services.json`, so it is inert in the default build but is
  a real feature, not dead code. It logs one custom event and needs no
  advertising ID, so `AD_ID`, both `ACCESS_ADSERVICES_*` and the Play
  install-referrer permission are removed with `tools:node="remove"`. Firebase's
  components are untouched and the feature still works when configured.

The default merged manifest is now **13 permissions**, down from 19 at vc55:
`READ_PHONE_STATE`, `freemme.permission.msa` and those four advertising entries
are gone, and every one that remains is product-required.

An analytics capability that was not packaged can no longer be selected at
runtime. `BuildConfig.POCKETCLAW_UMENG_PACKAGED` comes from the same value that
decides the dependency, so the guard cannot drift from what was built. No crash
path existed beforehand — the guard already implied the packaging condition —
but it did so by coincidence rather than by contract.

### Still open

`main` still has no authentic signing key. Every artifact so far, vc56 included,
carries a local development signing identity and is not releasable. Producing a
production key — and the uninstall it forces on the test device, since a
different signer cannot update an installation in place — is a deliberate later
step, tracked in `TASKS.md`.

## Final User-Facing Polish — PHYSICALLY ACCEPTED AND CLOSED

- Status: **PASS on a physical Android device (SM-A165F / Android 16), 2026-09-08
  as vc55. Merged to `develop` with `--no-ff`.** `main` untouched, no tags moved,
  no release created.
- Branch `feature/final-user-facing-polish`, from `develop` at `1db809e`, the
  Final Launcher Icon merge. Retained, not deleted.
- Two focused changes, nothing else.

### Physical acceptance evidence

    versionName 0.2.0, versionCode 55, arm64
    APK  cf7ae7858ae69162f2e421bc293e4438a2b2df5cd859908130066b65ff3e32d8

Core was rebuilt for vc55 because this milestone changed Go source; the staged
binaries and the ones the APK packages are byte-identical, and the installed
`base.apk` hashes to the built artifact. Installed as an upgrade, preserving
application data.

| Observed | Result |
| --- | --- |
| About renders in the Aperture visual language | PASS |
| About renders correctly in Arabic, right-to-left | PASS |
| The app version renders correctly | PASS |
| The runtime version renders correctly | PASS |
| No overflow, no visible loading or layout defect | PASS |
| Close dismisses the dialog | PASS |
| A neutral message produces a reply with no fixed sign-off | PASS |

The assistant used a different, contextually chosen emoji in that reply, which
is the intended behaviour: there is no output filter and no emoji ban.

### About dialog

Still a `showDialog` + `AlertDialog`; no Settings page was created and no new
artwork was added. What changed is the pre-redesign detail underneath it.

- Raw `SizedBox(height: 12 / 16)` spacing became `ApertureTheme.spaceXs` and
  `spaceMd`, and the version block is an `ApertureBracket` on `surface1` with
  the standard border and radius — the same card device the What's New sections
  and the Public Mode card use.
- The hardcoded 148px label column is gone. A version row now stacks its label
  over its value, which has no width to get wrong: a long translation cannot
  overflow it, a narrow phone cannot squeeze it, and Arabic mirrors it with no
  second layout. The values themselves stay `Directionality.ltr` monospace, as
  before — a reversed digest is a bug, not localization.
- A 40dp `accentSoft` icon tile with `Icons.camera` in the accent leads the
  PocketClaw title, matching the icon-tile treatment already used on the
  Public Mode card.
- The dialog no longer resizes when the versions arrive. Both labels render
  immediately and each value slot holds a one-line-high 14dp indicator until it
  is replaced, so loading and loaded are the same height.
- `Close` is a themed `FilledButton` rather than a bare `TextButton`.
- Unchanged on purpose: the app version still comes from
  `ServiceManager.getAppVersion()` and the runtime version from
  `getCoreVersion()`, `_normalizeAboutVersion` still handles empty/unknown, and
  focus still returns to the About button on dismissal. No release number is
  written into the dialog, and a test asserts none appears.

### The shared kernel identity no longer carries the lobster emoji

`getIdentity()` in `core/src/pkg/agent/context.go` built the header
`# PocketClaw <lobster> (%s)`. That is the `kernel.identity` prompt part, which
every agent on every channel receives on every turn, so a decorative character
in it reads as a signature to imitate — and the model mirrored it at the end of
replies. The header is now `# PocketClaw (%s)`. Nothing else changed:
`You are PocketClaw, a helpful AI assistant.` is untouched.

There is no response formatter and none was added. A model's reply reaches the
channel byte-for-byte as written, and
`TestFinalResponseIsDeliveredVerbatimIncludingEmoji` runs a real turn through
the bus to prove it, using a reply that contains the lobster. Stripping a
character from generated text would also strip it from text a user asked for.

The legitimate uses are unrelated and remain: `pkg/env.go`'s `Logo`, the
`cmd/picoclaw` terminal presentation, `/help` branding, workspace skill
metadata, documentation and assets.

### Core was rebuilt for vc55

This milestone changed `core/src`, which made the staged Core stale by design.
It was rebuilt through `./core/build-android-arm64.sh` and staged in its own
commit; the freshness guard passes and the installed Core carries the new
identity header.

### One runtime correction outside the repository

After vc55 was installed, a first physical test still produced a reply ending
with the mascot emoji. A read-only investigation traced every prompt part that
can reach the model and found the shipped Core and every repository default
already clean; the last remaining occurrence was a single decorative emoji in a
Markdown heading inside this device's own persisted long-term memory file, which
is loaded verbatim into the prompt. That one character was removed in place, in
the user's own runtime file, with no other line touched and no restart needed.

**No source change was made for this, and none should be.** Do not add code that
rewrites a user's memory file, and do not add an upgrade migration that edits
arbitrary user content. A clean install is unaffected: the shipped identity and
every seeded workspace default contain no mascot emoji.

The investigation also surfaced a separate upgrade defect — seeded workspace
templates are never refreshed on an existing install — recorded as deferred in
`TASKS.md`. It was not fixed here.

## Final Launcher Icon — PHYSICALLY ACCEPTED AND CLOSED

- Status: **PASS on a physical Android device (SM-A165F / Android 16), 2026-09-07
  as vc54. Merged to `develop` with `--no-ff`.** `main` untouched, no tags moved,
  no release created. Core was neither rebuilt nor modified for this milestone.
- Branch `feature/final-launcher-icon`, from `develop` at `c927243`, head
  `7df0bf7`. Retained, not deleted.

### Physical acceptance evidence

    versionName 0.2.0, versionCode 54, arm64
    APK  3716ffc75790668f4811b56408c9b2ee329d5827047931672b9358abecfa2c3c
    libpicoclaw.so     bf4fb01faf2741ccc402f3c07b4925e6cd4ebdd7d581d8775ab1744d50b8fc44
    libpicoclaw-web.so bd84eaba705f3d0ec0e71e8cfbefe5cf392668b4a53807b8b9bafb5d89422280

Core was built for vc52 from source `207c372` and reused byte-for-byte in vc53
and vc54, neither of which changed Go source.

| Observed | Result |
| --- | --- |
| The launcher icon on the device home screen is the flat APERTURE mark | PASS |

Accepted by direct visual inspection of the installed launcher icon. The
observation this milestone existed to make is the one the user made.

- The Android launcher is now the flat APERTURE mark on the Aperture canvas
  (`#0b1014`), replacing the glossy 3D mark. Every raster is derived from the
  one canonical geometry by `tool/generate_android_launcher_icons.py`, which
  imports `core/src/web/frontend/scripts/generate-brand-assets.py` read-only —
  there is no second copy of the mark to drift.
- Two defects were fixed alongside the artwork. The legacy pre-26 icons had a
  transparent background, so on API 24/25 — and `minSdk` is 24 — the launcher
  drew a floating mark with no tile; they are now opaque RGB tiles with no alpha
  channel at all. And `flutter_launcher_icons` still pointed its Android source
  at the 3D mark, so any future run would have reverted the launcher; Android
  generation is now disabled there with the reason recorded in `pubspec.yaml`.
- A `<monochrome>` layer was added for Android 13+ themed icons, drawn from the
  same geometry.
- `android:roundIcon` stays absent by decision; the corrected opaque legacy icon
  serves the API 25 launchers a round-icon family would have.
- Held by test as well as by the device: the mark's farthest ink sits 32.23dp
  from centre once the 16% adaptive inset is applied, inside Android's 33dp
  mask-safe radius. `test/unit/launcher_icon_test.dart` re-measures this from the
  PNG.
- Still open, deliberately: `assets/app_icon.png` and `assets/icon.ico` are the
  orange PicoClaw lobster and remain the Windows/macOS icon source and the
  desktop window icon. They are not Android launcher resources; they belong to
  the deferred namespace / branding migration recorded in `TASKS.md`.

## Status Dashboard v1 — PHYSICALLY VERIFIED AND CLOSED

- Status: **PASS on a physical Android device (SM-A165F / Android 16), 2026-09-07
  as vc53. Merged to `develop` with `--no-ff`.** `main` untouched, no tags moved,
  no release created.
- Branch `feature/status-dashboard-v1`, from `develop` at `a7d13b7`, head
  `85eb93a`. Retained, not deleted.
- Six commits: the Status implementation, the semantic audit corrections, the
  vc52 Core, the QR placement fix, the uptime contract, and the bottom-spacing
  polish.

### Physical acceptance evidence

Accepted across three device passes; vc53 is the artifact that closed it.

| Observed | Result |
| --- | --- |
| Status metrics render on the integrated tab-0 Dashboard | PASS |
| Telegram activity moves the counters correctly | PASS |
| Channels render with truthful Running/Stopped state | PASS |
| Model, provider and fallback count render correctly | PASS |
| Resources render Core RSS and cumulative CPU time | PASS |
| QR instructions render inside the QR access card | PASS |
| Gateway uptime renders as a human-readable duration | PASS |
| No large empty band below the Resources card | PASS |
| Navigation remains four destinations | PASS |

| Item | Value |
| --- | --- |
| versionName / versionCode | 0.2.0 / 53 |
| APK SHA-256 | `6469e9f103109505c9d05c9b47534e033a633d1da36c636621ecfa550196e857` |
| Core source fingerprint | `f4a3f913014034c974df39a63214c34fb66d4e8b08d848115b58db4723c12104` |
| `libpicoclaw.so` | `bf4fb01faf2741ccc402f3c07b4925e6cd4ebdd7d581d8775ab1744d50b8fc44` |
| `libpicoclaw-web.so` | `bd84eaba705f3d0ec0e71e8cfbefe5cf392668b4a53807b8b9bafb5d89422280` |

Core was built for vc52 from source `207c372` and reused byte-for-byte in vc53,
which changed only Dart. Three earlier device passes — vc51, vc52, vc53 — each
closed one review round: metrics, then uptime and QR placement, then spacing.

### What the architecture actually is

Status reads one authenticated snapshot per health poll. The gateway assembles
it inside the request from state that already exists — `activeTurnStates` ranged
by depth, session mailbox lengths, the channel manager's own maps, the agent's
resolved candidates — plus six atomics incremented at the two lifecycle points
that already classify the work. Nothing samples, buffers, retains or persists.

Deriving the counters from the runtime event bus was considered and rejected:
its subscriptions drop events under backpressure, so bus-derived counters would
undercount exactly when the numbers matter.

The payload is flat scalar DTOs in `pkg/status`, mapped field by field — the
same boundary `commands.SubagentInfo` draws for `/subagents`, drawn again
because Status has a wider audience than one chat window. A golden serialization
test fills the runtime with sensitive-looking values and proves none can reach
the JSON; it rejected a new top-level field on its first run after the uptime
work, which is the guard behaving correctly.

### Open, recorded, not fixed here

The Android gateway PID/auth token lives under the PocketClaw home, which
resolves to shared external storage when that storage mode is in use, and POSIX
0600 is not honoured there. Pre-existing; it already guards `/reload` and is
reused unchanged for read-only detailed Status. Recorded in `TASKS.md` for
Release Hardening. Do not weaken detailed-status authentication to work around
it.

## Telegram Command UX Phase A — PHYSICALLY VERIFIED AND CLOSED

- Status: **PASS on a physical Android device (SM-A165F / Android 16), 2026-09-06
  as vc46. Merged to `develop` with `--no-ff`.** `main` untouched, no tags moved,
  no release created.
- Branch `feature/telegram-command-ux`, from `develop` at `2f863d2`, head
  `896021f`. Retained, not deleted.
- Phase A **CLOSED** and still intact. Phase B was built, physically verified,
  and then abandoned by product decision — see below.

### Physical acceptance evidence

| Observed | Result |
| --- | --- |
| `/help` renders a readable product-oriented overview | PASS |
| `/switch` no longer dumps raw usage grammar | PASS |
| `/show` no longer dumps raw usage grammar | PASS |
| `/subagents` renders human-readable status, not a `%+v` struct | PASS |
| No `TurnID` / `SessionKey` / `ChatID` / `UserMessage` leak observed | PASS |
| `/context` remains readable and useful | PASS |

| Item | Value |
| --- | --- |
| Package / version | `com.lord1egypt.pocketclaw`, 0.2.0, code 46, arm64 |
| APK SHA-256 | `0d4304787dbe8e2226b146b247c5014ab0151939694fe4ca3bf3762a04f2fa34` |
| Core source fingerprint | `31d5277ecadf6f3f7e81de1d61e542da489e8a8d9e8b10fc50e9b404024016d7` |
| `libpicoclaw.so` | `c0ca79f377202fb9ec1ebc76b16529ec55f0a121291b11591817d5b973bc0e2f` |
| `libpicoclaw-web.so` | `f616ab7bd72f942dc650b2a6b16bda60f2f43b0d930e02b6147e4bb950377719` |

Installed with `install -r`; `firstInstallTime`, uid and `dataDir` unchanged, and
the installed permission list is identical before and after.

### What Phase A delivered

- **Raw CLI grammar is no longer the primary no-argument response.** A bare
  `/switch`, `/show` or `/list` used to answer with its parser grammar —
  `Usage: /switch [model to <name>|channel]`. Each now answers with a sentence
  naming its choices and a concrete example. This lives in a `Definition` field
  consumed by the executor, so it is presentation rather than a per-command
  special case. A genuinely wrong argument still gets the exact usage line,
  because there the grammar is the answer.
- **The `/subagents` privacy leak is fixed at the type boundary.** The command
  printed the agent's active-turn struct with `%+v`, which carried the user's
  own message, their session key and their chat id into Telegram along with a
  year 1 timestamp. The runtime now hands `pkg/commands` a `SubagentInfo` with a
  status, a duration and a nesting depth and nothing else, so those fields are
  not reachable from the formatting code at all. Turns are numbered rather than
  named because the only label a turn carries is the user's prompt.
- **`/help` is a product overview.** It no longer prints `EffectiveUsage` for
  every command, which is where all the angle brackets and pipes lived.
- **Command descriptions were rewritten for people.** `RegisterCommands` already
  derives Telegram's native "/" menu from `Definition.Description`, so this
  reached both surfaces through the mechanism that already existed.

### What Phase A deliberately did not do

No interactive buttons, no callback handling, and **no `/model`**. Those need an
interaction subsystem PocketClaw does not have, and a temporary text-only model
selector would have created a second model-selection state to unpick later.

The command-to-tap gap that remains is **not** a Phase A failure. Phase A moved
the answer from grammar to prose; it did not change who does the work, and it
was never scoped to. That is Phase B.

## Telegram Model Command — PHYSICALLY VERIFIED AND CLOSED

- Status: **PASS on a physical Android device (SM-A165F / Android 16), 2026-09-07
  as vc50. Merged to `develop` with `--no-ff`.** `main` untouched, no tags moved,
  no release created.
- Branch `feature/telegram-interactive-menus`, from the Phase A merge at
  `0acac8e`, head `39450df`. Retained, not deleted.
- The milestone that shipped is **not** the one the branch was opened for. The
  interactive picker was built, physically verified as vc47 and vc48, and then
  removed; what merged is its replacement, a fixed informational `/model`.

### What shipped — `/model` as a pointer, not a picker

`/model` answers with one fixed sentence and nothing else:

    🤖 Model selection is managed from PocketClaw Settings.

It is informational by construction rather than by care: the handler discards
the command `Runtime`, so the model switcher and the current-model reader are
unreachable from it, and the answer is a constant that cannot name a model,
provider or endpoint. A handled command returns before `runAgentLoop`, so there
is no LLM call and no history entry.

### Physical acceptance evidence — vc50

| Observed | Result |
| --- | --- |
| `/model` returns the fixed informational response | PASS |
| No interactive picker, no buttons | PASS |
| No callback infrastructure reachable | PASS |
| No "Thinking…" lifecycle for `/model` | PASS |
| No model or provider identifier visible | PASS |
| Telegram does not change the selected model | PASS |
| Dashboard remains the canonical model-selection surface | PASS |
| Phase A `/help`, `NoArgsHelp`, `/subagents` privacy still intact | PASS |
| `/switch model to <name>` still works for advanced use | PASS |

| Item | Value |
| --- | --- |
| Package / version | `com.lord1egypt.pocketclaw`, 0.2.0, code 50, arm64 |
| APK SHA-256 | `71ac692f724ad2454e7c08e4b6a3103e22b4cae7263127f109ab6a36fffaf2eb` |
| Core source fingerprint | `16a784237c5cf67a585746b34aa0d5adc597339b6e7b2d34f4e5a089eba9025c` |
| `libpicoclaw.so` | `4cc375fe4aefc89cf9108b964ad1a6ab34a0d89ca5a36608449ab35590222d5c` |
| `libpicoclaw-web.so` | `e8767e774d7dece2c061bf6609e52714ea96475db5434707fdffc5db5f9ec0d9` |

### The interactive picker — ABANDONED BY PRODUCT DECISION

#### This was not a failed implementation

`/model` worked, and the physical interaction test passed. Configured-model
eligibility filtering, revalidation at tap time, opaque TTL'd callback handles
that carried no model name or credential, chat and sender binding, editing the
same message instead of appending to the conversation, explicit cancel, and
retirement of a displaced picker were all implemented, covered by tests that
were each shown to fail when their protection was removed, and confirmed by use
on the device.

#### What was rejected is the product semantics

Physical use exposed a scope mismatch that no test asserted, because nothing had
said which scope was correct:

- The **PocketClaw Dashboard** owns the configured default in `config.json`.
- The **picker** moved only the running `AgentInstance`.
- So a model configured in the Dashboard could be absent from the picker, and a
  model chosen from Telegram never became the configured default.

That is two sources of truth for one setting. The only fixes are to synchronize
runtime selection back into configuration or to make the Dashboard follow
runtime state — both large mechanisms for a small convenience, and both were
rejected.

#### Final product direction

**Model selection and configuration belong to the PocketClaw Dashboard.**
Telegram remains a conversation and control surface, not a model-configuration
surface. `/model` says so and does nothing else, which is what makes the
direction discoverable from the place people ask the question.
`/switch model to <name>` survives as an advanced compatibility command
that plainly moves the running agent; its `/help` description is now "Advanced
runtime controls", so chat does not present itself as the place to choose a
model. There is no replacement *picker*, no persistence, no synchronization
mechanism, and no callback or interactive-menu runtime retained for later use —
PocketClaw again has no interactive input path in any channel. A future
interactive Telegram surface is designed when a real product requirement asks
for one. See DECISIONS.md, "Model selection belongs to the Dashboard, not
to Telegram".

## Android Hardware Tool Cleanup — PHYSICALLY VERIFIED AND CLOSED

- Status: **PASS on a physical Android device (SM-A165F / Android 16), 2026-09-06
  as vc45. Merged to `develop` with `--no-ff`.** `main` untouched, no tags moved,
  no release created.
- Branch `feature/android-hardware-tool-cleanup`, from `develop` at `5c767f4`,
  head `14a88ba`. Retained, not deleted.
- Milestone **CLOSED**.

### Physical acceptance evidence

The user opened the PocketClaw Android Tool Library and confirmed the hardware
tools and their category are gone.

| Item | Value |
| --- | --- |
| Package / version | `com.lord1egypt.pocketclaw`, 0.2.0, code 45, arm64 |
| APK SHA-256 | `59ebc6d82ed2f463556e16b1bfe1295f5acff349e3343e7c6be6b57dcef56a63` |
| Core source fingerprint | `46fb536b9b42b4c775170550bd834971b1e3e2415732d1065eb8c9ddb567f4c4` |
| `libpicoclaw.so` | `e9b784a08cc42677b89bf97d3ce36867dfdf5bf4510678a4b9bc716375cc7136` |
| `libpicoclaw-web.so` | `5decd123338539d5c46f82b4c7caaaed2656ecec7e3d9a2054b67979f6b30040` |

Installed with `install -r`; `firstInstallTime`, uid and `dataDir` unchanged. The
installed permission list is byte-identical before and after — this milestone
adds none, and the APK declares nothing USB, serial or hardware related.

### Accepted behaviour

- `i2c`, `spi` and `serial` are absent from the Android Tool Library.
- A persisted config cannot turn them back on: the managed Gateway is launched
  with `PICOCLAW_TOOLS_I2C_ENABLED`, `PICOCLAW_TOOLS_SPI_ENABLED` and
  `PICOCLAW_TOOLS_SERIAL_ENABLED` set to `false`, and env is applied after the
  file. That covers a config imported from a Linux machine or hand-edited.
- They therefore never register into `ToolRegistry` and never reach the model's
  tool definitions.
- The bundled hardware skill is withheld from model-facing discovery on Android,
  including on an installation upgraded from an earlier PocketClaw that already
  has `workspace/skills/hardware/SKILL.md` on disk.
- The dormant upstream implementations and the embedded skill bytes remain in
  the tree and in the binary. This was product-surface cleanup, not size work.
- Every other platform is unchanged, and that is now asserted for linux, darwin,
  windows and freebsd rather than only for whichever platform the suite runs on.

### Why it was hidden rather than deleted

The tools are correct, dependency-free beyond an already-vendored
`golang.org/x/sys/unix`, and genuinely useful on the Linux boards upstream
targets. They cost no permission and no measurable startup or memory. What made
them wrong was the platform: an unrooted phone exposes no `/dev/i2c-*`,
`/dev/spidev*` or `/dev/tty*` to an app UID — verified on the device, where even
`adb shell` finds none — and the app declares no USB host support, so serial over
USB-OTG would need an Android-native implementation rather than these syscall
wrappers.

The more consequential half was the skill, not the Tool Library. The bundled
hardware skill was embedded in the Core binary and reached the system prompt on
every turn, advertising I2C and SPI capability on a device that has neither. It
is filtered at discovery rather than at onboarding, because an upgraded
installation already has the file and declining to copy it for new users would
have fixed nothing for existing ones.

## Chat Lifecycle Durability — PHYSICALLY VERIFIED AND CLOSED

- Status: **PASS on a physical Android device (SM-A165F / Android 16), 2026-09-06
  across vc42, vc43 and vc44. Merged to `develop` with `--no-ff`.** `main`
  untouched, no tags moved, no release created.
- Branch `feature/chat-lifecycle-durability`, from `develop` at `2312f35`,
  head `8024094`. Retained, not deleted.
- Milestone **CLOSED**.

### Physical acceptance evidence

| Area | Candidate | Result |
| --- | --- | --- |
| Telegram `/stop` cancellation | vc42 | PASS |
| Telegram FIFO ordering | vc42 | PASS |
| Multi-image + fallback preservation | vc42 | PASS |
| Safe user-facing provider errors | vc43 | PASS |
| Telegram live channel reconciliation | vc44 | PASS — typing indicator OFF **and** ON both applied live, with no Gateway restart |

Accepted artifacts: vc42 `181ef4c5…`, vc43 `d9da1af8…`, vc44
`d37a8661f1c45efe8a200f05a586589c582b2fae3220cae190fdf217e72c780b`
(`com.lord1egypt.pocketclaw` 0.2.0, code 44, arm64). vc44 Core:
`libpicoclaw.so` `726ea74e…`, `libpicoclaw-web.so` `41c6a69c…`, source
fingerprint `59082e67b86a060869bd55118a221d5ac4395b8d05e42e7d8fd822ff92a2247b`.
Installed with `install -r`; `firstInstallTime`, uid and `dataDir` unchanged
across every install.

### The four defects this closes

**`/stop` was queued behind the turn it existed to cancel.** Telegram is the only
channel using the independent response lifecycle, so a message arriving mid-turn
is retained in the session mailbox and the dispatcher moves on without reading
it. The stop handler is reached only on the steering path, which Telegram never
takes, so `/stop` during a long turn produced no reply at all until that turn
finished on its own. Control traffic no longer queues. Cleanup also had to move:
the typing indicator and the "Thinking…" placeholder are recorded per inbound
lifecycle, so neither was reachable through the `/stop` message's own context —
the acknowledgement is published on the cancelled turn's lifecycle instead.

**Configuring a fallback changed the request sent to the primary.** Seven images
succeeded against the primary alone and were rejected with HTTP 400 as soon as
fallbacks were configured. Per-candidate providers were registered under the
runtime `provider/model` key and only for the fallbacks, so a fallback naming the
same protocol and model id took ownership of the primary's key and the primary's
request went to the fallback's endpoint with its credentials. Registration is
keyed by model_list identity now.

**Provider response text reached the chat.** A single model failing with 401 put
the whole response into Telegram under an "Original error:" heading — raw JSON,
account message, billing link. Both the single-model and exhausted-chain paths
now go through one rule, and a 401 about money no longer blames the API key.

**Channel settings did not apply live.** Three independent faults, each
sufficient alone: `gateway.hot_reload` defaults to off so the config watcher was
never armed; the reconcile hash had been narrowed to the settings payload and
could not see any common `Channel` field including `typing`; and
`Channel.Typing.Enabled` had exactly one reader in the tree — IRC. All three are
fixed, with hot reload enabled at the Android host boundary rather than by
changing the Core default.

### Deferred, recorded and deliberately not implemented

- **Stale "Gateway restart required" banner after a successful hot reload.**
  `gateway.bootConfigSignature` is assigned only when the launcher starts or
  attaches to a gateway process; nothing refreshes it after an in-process
  reload, so the banner stays on although the change has already applied.
  Cosmetic, non-blocking, and a real subsystem change to fix — it needs a
  reload-completion signal from gateway to launcher that does not exist today.
- **`BaseChannel` typing inconsistency for non-Telegram channels.** The gate
  went into `Manager.StartTyping`, which today only Telegram reaches. Discord,
  Slack, Matrix and Feishu still start typing through `BaseChannel` without
  consulting `Typing.Enabled`, and their defaults leave it `false` — so gating
  them there would switch off an indicator nobody asked to lose. Needs a
  defaults decision first, not a code change first.
- **Android hardware tools cleanup.** i2c / spi / serial should be hidden or
  unregistered at the PocketClaw Android surface rather than deleted from
  upstream Core, which would diverge the vendored tree for no runtime benefit.

## Configured Model Discovery — PHYSICALLY VERIFIED AND CLOSED

- Status: **PASS on a physical Android device (SM-A165F / Android 16),
  2026-09-05 as vc41. Merged to `develop` with `--no-ff`.** `main` untouched,
  no tags moved, no release created.
- Branch `feature/configured-model-discovery`, from `develop` at `2b37af2`.
  Retained, not deleted.
- Milestone **CLOSED**.

### The defect this closes

The Fallback Models picker offered Azure, Cerebras, Anthropic, Groq, Ollama and
Volcengine to a user who had configured only OpenCode and Gemini. The picker was
not inventing a catalog: `config.DefaultConfig()` seeds `model_list` with thirty
keyless provider templates, the backend already marked every one of them
`status: "unconfigured"`, and the Fallback picker was the one selector that never
read that flag. The chat Default selector filtered on `available` and hid them.
Two selectors, two filters over one list, and they disagreed.

The fix was not a third filter. There is now one shared source.

### Delivered behaviour

- **Default Model and Fallback Models draw from one configured-provider model
  source** — `core/src/web/frontend/src/lib/configured-model-source.ts`. A
  structural test fails the suite if either selector re-implements the rule,
  imports the provider preset registry, or reads `common_models`.
- **Only actually configured provider instances participate.** A provider with
  no resolvable credentials contributes no group at all.
- **Live model discovery uses the configured provider's own API**, through the
  existing `POST /api/models/fetch`.
- **Discovery is isolated per provider.** Under `Promise.allSettled`, one
  provider timing out shows a retry on that group, keeps its configured entries,
  and leaves every other group's results intact.
- **Discovered models can be materialized on selection** through
  `POST /api/models/materialize`, which finds or creates the entry and applies a
  role in a single config write.
- **Model identity is provider-scoped** — normalized provider, normalized API
  base and model id together — so two providers exposing `deepseek-chat` stay
  distinct rather than one routing through the other's credential.
- **API credentials remain server-side.** The discovery request carries a
  `model_index` and never a key; the backend resolves the stored credential
  after verifying the provider and base match, and materialization copies the
  stored `SecureStrings` without decrypting it.
- **The unconfigured DefaultConfig provider templates never appear in a
  selector**, including when discovery fails — there is no consolation fallback
  to the global catalog.
- **Providers without model-list support still expose their explicitly
  configured entries**, rather than being dropped or padded from presets.
- **Fallback materialization and chain Save remain separate operations by
  design.** A discovered fallback is materialized first and the ordered chain is
  saved by the existing Save flow. If that save later fails the model remains a
  valid, reusable configured entry outside the chain. This is deliberate and is
  not described anywhere as atomic. Default-model materialization is atomic.

### Product decision: no dedicated Vision / Image Model selector

**PocketClaw does not expose a Vision / Image Model control.** The product-facing
model configuration is **Default Model and Fallback Models**, and nothing else. A
user who needs image support chooses a multimodal model as their Default Model.

A Vision surface was built on this branch and removed at the user's decision:
`bc7c3ff` and `c9c0db2`, reverted by `66d80ee`. **Dedicated Vision routing is not
a delivered product feature and must not be listed as one.**

Core's `agents.defaults.image_model`, `agents.defaults.image_model_fallbacks` and
`routeMediaTurn` predate this branch, are load-bearing upstream behaviour, and
were left byte-identical to `develop` — verified file by file at merge time. They
remain reachable by editing the config file. **They must not be removed merely
because the Dashboard does not expose them.**

### Verification

- Frontend: `tsc -b` clean, `eslint` clean, **391 tests passed**.
- Go: `web/backend/...`, `pkg/agent` and `pkg/config` suites pass.
- Core: source-freshness, fingerprint, staged-core, developer-path, runtime
  payload and Python payload guards pass. The Gradle arm64 release guard verified
  all eleven native payloads.

### Final accepted artifact

- Built 2026-09-05 through the canonical Gradle release path.
- Package/version: `com.lord1egypt.pocketclaw`, `0.2.0` (version code `41`),
  minSdk 24, targetSdk 36.
- Size: 64,211,166 bytes — the arm64 band.
- SHA-256: `4fd3c201bb9a07017a954ade90f5eb471b97d281e057a58fad4b7e18cf76e7ec`
- Staged Core:
  `libpicoclaw.so` `e05de9f687f98acfc33e6a05d2b82f05216258f18a2aeb3302b30a1c7e273a66`,
  `libpicoclaw-web.so` `fbcf467c269be121c02d889564fee582f3e888189b347b3e0005f1d7af67a593`.
  `libpicoclaw-web.so` returning to its exact pre-Vision byte count is the
  embedded frontend confirming that removal rather than the source tree claiming
  it.
- Installed with `install -r`; app data preserved (`firstInstallTime`, uid and
  `dataDir` unchanged).

### Known pre-existing test failure — not caused by this branch

`TestNoUserFacingWhatsAppSurface` fails over
`test/widgets/whats_new_page_test.dart`, unchanged from the `develop` baseline
and unrelated to this work. Recorded, not fixed.

## APERTURE Visual Redesign — PHYSICALLY / VISUALLY VERIFIED AND CLOSED

- Status: **PASS on a physical Android device (SM-A165F / Android 16),
  2026-09-05. Merged to `develop` with `--no-ff`.** `main` untouched, no tags
  moved, no release created.
- Branch `feature/visual-redesign-aperture`, from `develop` at `9877365`.
  Retained, not deleted.
- Two accepted implementation commits: `e51b1cb` (visual foundation) and
  `082cc57` (product polish). Reviewed physically as vc38 and vc39.
- Milestone **CLOSED**.

This closes the PocketClaw Visual Identity / UI-UX Redesign milestone the
What's New entry named as next. The design source is
`docs/design/VISUAL_IDENTITY_CLAUDE_A.md` on `design/claude-concept-a` at
`2907752`, which stays a reference-only branch and is not merged.

### The four findings this answered

None of them was a matter of taste, which is why each has a guard now.

The web dashboard was running unmodified shadcn defaults: every neutral at
chroma 0 and a near-white dark-mode primary, so there was no brand hue in the
token set at all. `main.dart` drew light mode from `PocketClawDesign` and dark
mode from `AppTheme`, two unrelated design systems, so toggling the theme did
not adjust PocketClaw — it swapped products. The web wordmark and every favicon
were still the PicoClaw lobster while Android shipped a different mark. And
neither mark had a flat, single-colour, 16px form, so neither could function as
one.

### The accepted production state

- One APERTURE visual system across both surfaces.
- A chromatic Graphite neutral family — every neutral carries a small chroma at
  hue 245, rising as the surface darkens. Chroma-0 grey is the default of every
  framework, which is exactly why it read as one.
- **Claw** (hue 208) is the single interactive accent: intent, selection, focus.
- **Signal** (hue 195) is reserved for live machine state — the gateway running
  and a response streaming, and nothing else.
- Danger for destructive, Warning for exposure and risk, Success for confirmed.
- One Flutter theme emitting both brightnesses from the same tokens. All six
  user-selectable modes are preserved, along with the enum order and the
  persisted preference index; they became six accents over one structure rather
  than six products. `obsidian` keeps its true-black canvas for AMOLED and
  `sakura` still resolves light in both slots.
- A vector-first PocketClaw identity: one geometry, under 2 KB, legible at
  16px, correct in monochrome and in both themes from a single asset.
- The visible PicoClaw wordmark and lobster are gone from every user-facing
  surface, and the favicon family, touch icon, PWA icons and web manifest all
  carry PocketClaw branding.
- The mobile menu renders a real hamburger with a 44px hit area and an honest
  accessible name, and the narrow toolbar collects its low-frequency utilities
  behind one labelled overflow control.
- The desktop sidebar is the persistent brand anchor. There is no second full
  PocketClaw wordmark in the top chrome.
- Chat is deliberately asymmetric: the user's turn is a contained bubble, the
  assistant's is an editorial block on a logical-start rail, with polished
  composer, code, table and tool-call presentation.
- Flutter and the Models / Web pages carry the same component language.
- Arabic RTL and the accessibility contract are preserved throughout.

### What was deliberately not renamed

Only user-visible branding was removed. `picoclaw-web`, the `localStorage`
keys, the Go package paths, the `/pico/*` routes, `PicoOwnerPrincipal`,
`PICOCLAW_DNS_SERVER`, `libpicoclaw.so` and `libpicoclaw-web.so` are
load-bearing and stay. Renaming the storage keys silently discards every user's
saved preference and needs its own migration and its own proof.

### Verification

- Frontend: `tsc -b` clean, `eslint` clean, **349 tests passed**.
- Flutter: `flutter analyze` no issues, **223 tests passed**.
- Core: source-freshness, fingerprint-input, staged-core, developer-path,
  runtime payload and Python payload guards all pass. The Gradle arm64 release
  guard verified all eleven native payloads, the appended Python standard
  library and the `pocketclaw_bootstrap` entry point.
- One pre-existing failure is recorded and **not** fixed here — see below.

### Final accepted artifact

- Built 2026-09-05 with the canonical command

      cd android && ./gradlew :app:assembleRelease \
        -Ptarget-platform=android-arm64 \
        -Pdart-defines=$(printf '%s' \
          'POCKETCLAW_ONBOARDING_BASE_URL=https://pocketclaw-telegram-setup-bot-83ai.vercel.app' \
          | base64 -w0)

- Size: 64,200,254 bytes — the ~64 MB arm64 band, so
  `-Ptarget-platform=android-arm64` was honoured. 166.9 MB of payload in
  `lib/arm64-v8a/` against 286 KB and 123 KB of plugin stubs.
- SHA-256: `5e576b541ddb8158ac4e94bdf0aeb3f770403f20721f34fc4f47337bac0ac636`
- Package/version: `com.lord1egypt.pocketclaw`, `0.2.0` (version code `39`),
  minSdk 24, targetSdk 36.
- Installed with `install -r`; app data preserved (`firstInstallTime`, uid and
  `dataDir` all unchanged).
- Embedded Core rebuilt for this candidate:
  `libpicoclaw.so` `b6fc9a8e64f396cfb5e3d49a97f8cae2a92da2ead0d78aa02748020693ed81a4`,
  `libpicoclaw-web.so` `e32264d884909ec03d598456a45497080cf689061ba20389b5363fb3c090fda6`,
  both stamped with source fingerprint `1e44c173…` and carrying zero
  developer-machine paths.

### Known pre-existing test failure — not caused by this branch

`TestNoUserFacingWhatsAppSurface` fails on
`test/widgets/whats_new_page_test.dart`, which lists `'WhatsApp'` among the
words the release notes must never advertise. It was independently reproduced
in a clean worktree of `develop` at `9877365` and that file is byte-unchanged
on this branch. It is recorded, not fixed: it belongs to whoever owns that
guard, and folding it into a visual merge would hide it.

## What's New — PHYSICALLY VERIFIED and CLOSED

- Status: **PASS on a physical Android device (SM-A165F / Android 16),
  2026-09-05. Merged to `develop` with `--no-ff`.** `main` untouched, no tags
  moved, no release.
- Branch `feature/whats-new`, from `develop` at `09790ac7`. Retained, not
  deleted.
- Milestone **CLOSED**. Next milestone is PocketClaw Visual Identity /
  UI-UX Redesign. **Not started** — nothing in this milestone implements, stubs
  or prepares it.

Release notes for 0.2.0, reachable from Settings, in all twelve app locales.

### Settings entry

A What's New control in the Settings header, immediately **before** About and
styled to match it. It carries a localized NEW badge while the notes are
unread. Tapping it pushes a full page — `Navigator.push`, never an
`AlertDialog`. The existing D-pad focus chain absorbed one new node and now
runs `What's New -> About -> public mode toggle`, with What's New at the head
of the chain as its own predecessor.

### Seen state

- Preference key: `pocketclaw.whats_new.last_seen_version`, through
  `SharedPreferences`.
- Release identity is the **versionName only**, read from
  `PackageInfo.fromPlatform().version`. `buildNumber` — the Android versionCode
  — is never consulted, because it is bumped for internal candidates that carry
  nothing new to read. Keying the badge on it would re-announce a release the
  user has already seen; vc36 and vc37 are exactly that case.
- The badge shows while the stored value differs from the versionName, so a
  first install with nothing stored shows it.
- The mark is written once the route is on screen — never on the way into
  Settings, and never before the push succeeds.
- Both the store and the version loader are injected into `ConfigPage`, so the
  badge is drivable in a test without a platform preference store and without
  global mutable state.

### Release data

Typed Dart in `lib/src/whats_new/whats_new_release.dart`, not JSON.
`WhatsNewRelease{version, sections}` over `WhatsNewSection{kind, bullets}`,
where each bullet is a `String Function(AppLocalizations)`. Structure lives in
Dart; every user-facing word resolves out of the ARB bundles, so a release note
cannot ship untranslated English prose.

Sections are **New / Improvements / Fixes**. 0.2.0 advertises Managed Runtime,
the bundled Git / GitHub CLI / curl / ripgrep / jq / SQLite, PocketClaw's
bundled Python 3.14 runtime, secure GitHub sign-in and Telegram integration;
then provider resilience, a more accurate Managed Runtime catalog and the
PocketClaw identity cleanup; then the stale process-record recovery and the
multi-entry channel-list fix. A test joins all rendered copy and fails on
`WhatsApp`, `BlueStacks`, `Auto-Start` or `versionCode`.

### Localization and RTL

Sixteen flat keys in all twelve `.arb` bundles — ar, de, en, es, fr, hi, id,
ja, ko, pt, ru, zh — with real translations, not English copies. Enforced by a
test asserting every non-English sentence key differs from English and that the
nine non-Latin locales' prose is not ASCII. No message takes a placeholder, and
parity is asserted per locale.

The page uses `EdgeInsetsDirectional` throughout with no hard-coded left/right
and no Arabic layout branch. Proved by geometry rather than inspection: the
bullet marker sits left of its text under `en` and right of it under `ar`.

### The Settings header truncation, found physically on vc36

vc36 passed the feature but exposed a layout defect. The header was a `Row`
whose title sat in an `Expanded`. A `Row` gives its inflexible children their
natural width first and the `Expanded` only what remains, so the title was last
in line for space. With the unseen NEW badge widening the What's New button
there was nothing left: on a 360px-wide phone the title was handed **0px**
against the 176px it needed and rendered `Setti...`. Opening What's New retired
the badge, freed the width and hid the defect.

It was never English-specific. Arabic needed 198px and was truncated even in
the seen state.

The header is now a **`Wrap`**. Title and actions each keep their natural width,
sit at opposite edges under `WrapAlignment.spaceBetween` while they share a
line, and drop to a second line when they cannot. The actions are a nested
`Wrap` so a long locale breaks between the two buttons instead of overflowing.
Nothing is measured against a specific language, so RTL keeps mirroring on its
own. Regression-guarded at 360x800 in four cases — English and Arabic, badge
visible and badge seen — asserting the title's rendered box is at least its
`getMaxIntrinsicWidth`. All four fail against the pre-fix layout.

### Verification

- `flutter analyze`: no issues.
- `flutter test`: **203 passed**.
- Physical: vc36 PASS for the feature, vc37 PASS for the header fix, both on
  SM-A165F / Android 16 as in-place upgrades with app data preserved.

### Final accepted artifact

- Path: `build/app/outputs/flutter-apk/app-release.apk` (ignored; not committed)
- Built 2026-09-05 with the canonical command

      cd android && ./gradlew :app:assembleRelease \
        -Ptarget-platform=android-arm64 \
        -Pdart-defines=$(printf '%s' \
          'POCKETCLAW_ONBOARDING_BASE_URL=https://pocketclaw-telegram-setup-bot-83ai.vercel.app' \
          | base64 -w0)

  with `JAVA_HOME=/home/lordegypt/PocketCLaw/.tooling/jdk-17` and
  `GRADLE_USER_HOME=/home/lordegypt/PocketClaw-App/.tooling/gradle-stage-a-clean`.
- Size: 64,543,774 bytes — the ~64 MB arm64 band, not the ~50 MB universal
  band, so `-Ptarget-platform=android-arm64` was honoured. 167.5 MB of payload
  sits in `lib/arm64-v8a/` against 286 KB and 123 KB of plugin stubs in
  `armeabi-v7a` and `x86_64`.
- SHA-256: `7e617eb7e4e6a0738bf9cc7ce3da56204953911b387fee8781aa59ad363424e1`
- Package/version: `com.lord1egypt.pocketclaw`, `0.2.0` (version code `37`),
  minSdk 24, targetSdk 36.
- Release guard PASS for all eleven required arm64 payloads, plus the appended
  Python standard library and the `pocketclaw_bootstrap` entry point.

vc37 is also the first physically accepted APK carrying the pt-BR/zh Dashboard
locale cleanup at `65dfc02`, which vc35 predated.

## Telegram Context Settings + Dashboard i18n — PHYSICALLY VERIFIED and CLOSED

- Status: **PASS on a physical Android device (SM-A165F / Android 16),
  2026-09-04. Merged to `develop` with `--no-ff`.** `main` untouched, no tags
  moved, no release.
- Branch `feature/telegram-context-settings`, from `develop` at `f83699d4`.
- Milestone **CLOSED**. Next milestone is What's New, on a new branch from the
  updated `develop`. Not started.

The Telegram bounded-context algorithm itself was merged earlier, at
`f83699d4b6faab73022349c4e48928653a0c05e3`. This milestone makes its limit
user-configurable, applies a change without a restart, and finishes the
embedded dashboard's localization.

### Context Memory settings

- Choices: 10, **15 (Recommended)**, 20, 25, Custom. Custom accepts 5–50.
- An explicit, localized Save button. It stays disabled until the value is both
  valid and changed, and one action is one save.
- Backend key: `agents.defaults.telegram_recent_context_messages`. Default
  remains **15**, unchanged from the bounded-context milestone.

### Live apply — no restart

Saving a new Telegram context limit takes effect on the **next turn**, with no
app restart and no Gateway restart. The config cache detects an atomic
same-size replacement by file identity rather than size and mtime, because
`SaveConfig` renames a temporary file over the target and two saves of
equal-length payloads can land inside one coarse timestamp tick.

Physically proved on the device with Custom = 17:

| history_total | limit applied |
| --- | --- |
| 26 | 17 |
| 28 | 17 |
| 30 | 17 |
| 32 | 17 |
| 34 | 17 |

### Dashboard localization

- i18next + react-i18next. The Flutter host passes its selected locale as
  `?lng=`, which wins over the cached value; i18next's own localStorage
  persistence remains the manual override.
- Twelve app locales resolve: `ar de en es fr hi id ja ko pt ru zh`. `pt` maps
  onto the existing `pt-BR` resource rather than duplicating it.
- The pre-existing dashboard-only resources `bn-IN` and `cs` are preserved and
  still reachable from the selector.
- **907/907 keys in all thirteen non-English bundles**, 0 missing, 0 extra, 0
  placeholder drift.
- Arabic is genuinely right-to-left: `lang` and `dir` come from `i18n.dir()`,
  never from a test for Arabic.
- One shared language selector replaces the three hand-written dropdowns in the
  app header, Launcher Setup and Launcher Login. Entries are endonyms, so they
  are not routed through the resource bundles.
- Localized: Models, Channels (incl. Telegram), Chat, Credentials, all `pages`
  namespaces, Tour, Launcher Setup and Launcher Login.
- pt-BR and zh predated the nine-locale work and had never been held to the
  English-copy rule; their remaining untranslated prose was cleaned up last.

### RTL sidebar

- The shared `Sidebar` derives its default side from `i18n.dir()`: **RTL →
  right, LTR → left**. An explicit `side` prop still wins.
- Live language switching moves the anchor with the drawer open. There is no
  second direction state, no observer and no Arabic-specific check.
- The sidebar's inner border is the logical `border-e`, correct on the
  content-facing edge in both directions.
- Physically verified on vc35.

### Physical candidate

| Field | Value |
| --- | --- |
| versionName | `0.2.0` |
| versionCode | **35** |
| Commit built | `1505e33` |
| APK | `build/app/outputs/flutter-apk/app-release.apk`, 64,530,162 bytes |
| APK SHA-256 | `fcec23a5430969fef892bb83ccc784bd35a1a33729fdd926de2869c355282077` |
| `libpicoclaw.so` | 37,683,553, `2a6c701b4050bf21d7947e8db990ccaafa349ba196acff6994a93f52681264d5` |
| `libpicoclaw-web.so` | 25,756,001, `7985589fdadccec24b671dee92de879270f92328ffa1c548770d198d4f944cb2` |
| Core source fingerprint | `1e44c17368f6433bb914a367670398840e3dd23525f63f4dc31e769049772c08` |

Developer paths in both binaries: 0. Installed with `adb install -r` over the
existing install; `firstInstallTime` and the app UID were unchanged, so the
user's data and config were preserved. **User physical acceptance PASS.**

### One thing the installed APK does not carry

vc35 was built at `1505e33`. The final commit, `65dfc02`, changed only locale
JSON and i18n tests — no Core, Flutter or runtime behaviour — so it was
deliberately not rebuilt or reinstalled. **The pt-BR and zh translation cleanup
is therefore in `develop` but not in the APK on the device.** A future build
picks it up; nothing regressed by leaving it.

## WhatsApp Self-Chat + Chat image attachment APK — PHYSICALLY VERIFIED

- Status: **PASS on a physical Android device (SM-A165F / Android 16),
  2026-09-01. Merged to `develop` with `--no-ff`.** `main` untouched, no tags
  moved, no release.
- Path: `build/app/outputs/flutter-apk/app-release.apk` (ignored; not committed)
- Built: 2026-09-01 with the canonical command

      cd android && ./gradlew :app:assembleRelease \
        -Ptarget-platform=android-arm64 \
        -Pdart-defines=$(printf '%s' \
          'POCKETCLAW_ONBOARDING_BASE_URL=https://pocketclaw-telegram-setup-bot-83ai.vercel.app' \
          | base64 -w0)

  with `JAVA_HOME=/home/lordegypt/PocketCLaw/.tooling/jdk-17` and
  `GRADLE_USER_HOME=/home/lordegypt/PocketClaw-App/.tooling/gradle-stage-a-clean`.
- Size: 64,297,534 bytes — the ~64 MB arm64 band this app has occupied since the
  Managed Runtime payloads landed, not the ~50 MB universal band, so
  `-Ptarget-platform=android-arm64` was honoured.
- SHA-256: `2b4e5b56420e847d6fb772deda3fc2f83598b6f4e9868fa6f4fc9d09bbbafe67`
- Package/version: `com.lord1egypt.pocketclaw`, `0.2.0` (version code `15`),
  minSdk 24, targetSdk 36.
- Release guard PASS for all eleven required arm64 payloads, plus the appended
  Python standard library (11,511,845 bytes) and the `pocketclaw_bootstrap`
  entry point.

| Packaged library | Size | SHA-256 |
| --- | --- | --- |
| `libdartjni.so` | 131,248 | `47dae44db1c6202d164c0bb2ff25cc661023ba2904a6679abad4f3dcf3fcb5cd` |
| `libpicoclaw.so` | 37,683,553 | `fe277e8d821f35fadf3f0c54dee27d89f20cc557fca00d9cfcc1e2c64a8b0a4f` |
| `libpicoclaw-web.so` | 25,166,177 | `8aef0661ae6c3ab5a2176255ce3d9f6820e9f09d0277fa0969fdf72263c9eb8d` |

- Core source fingerprint stamped:
  `9a03c38717281c5adfeab35ace622603941be45f327ad23a33a3a197b957699b`.
  Developer paths in both binaries: 0.
- Regression gate, all green on this tree: `go test`/`go vet`
  `-tags goolm,stdjson ./...`; `flutter analyze` clean and 161 Flutter tests;
  `:app:testReleaseUnitTest` 23 tests; console `tsc`, `eslint`, and 112 vitest
  tests.
- Both WhatsApp packages appear in the merged manifest's `<queries>`, which is
  what makes package detection answer anything but "neither" on Android 11+.
- The retired `whatsapp` and `whatsapp_native` names are absent from the
  embedded console bundle in every locale; `whatsapp_self_chat` is present in
  both Core binaries.
- Physical results are recorded in `SESSION_HANDOFF.md`. The first pass
  (versionCode 14) passed every step except automatic apply on
  Connect / Change / Disconnect; the recheck (versionCode 15) passed that plus
  the Chat-attachment and GitHub regressions. This is the current reference
  physically verified PocketClaw artifact.


## Secure GitHub auth — PHYSICAL PASS and merged, 2026-09-01

Branch `feature/secure-github-auth`. **Physically validated inside the installed
application, then merged to `develop`.** Not released, `main` untouched, no tags
moved.

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

### Physical acceptance — PASS, 2026-09-01

All twelve steps on SM-A165F / Android 16, with a real token against a private
test repository.

| Step | Result |
|---|---|
| Connect, validated through the bundled gh | **PASS** |
| Account resolved as `Lord1Egypt` | **PASS** |
| Safe Core restart completed | **PASS** |
| Test connection | **PASS** |
| Close and reopen; credential persists | **PASS** |
| `gh api user` from the managed runtime | **PASS** |
| Private `gh repo view` | **PASS** |
| Private HTTPS `git clone`, `fetch`, `pull` | **PASS** |
| Raw token absent from argv, logs, events, agent output, gh config, `.git/config` | **PASS** |
| Disconnect removes active auth after the safe restart | **PASS** |
| `adb install -r` over the existing install preserves the credential | **PASS** |

Two results are worth keeping for what they prove rather than for passing.

After Disconnect, `git fetch` failed with `could not read Username for
'https://github.com': terminal prompts disabled`. Git had no credential from any
other source: no helper, no `~/.gitconfig` entry, nothing in the repository's
own config, no cached username. Had `gh auth setup-git` ever run, or had a token
reached a remote URL, that fetch would have succeeded. The process-scoped
`http.extraheader` left with the old Core process, which is the whole design.

The reinstall preserved `firstInstallTime` (2026-08-26) while `lastUpdateTime`
moved, so Android replaced the package and kept the data directory. The card
still read Connected as `Lord1Egypt` with nothing re-entered, and
`gh api user --jq .login` still returned it: the Keystore key survived the
package replacement, as it must, since it is not part of the APK.

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

Branch `feature/python-lite-runtime`, from `develop` at `b46921e`.
**Not started. Architecture review only — nothing to be compiled or bundled
until that review is approved.**

The appeal is capability per megabyte: one interpreter buys scripting, parsing,
JSON, CSV, XML, regex, calculation, file transformation, SQLite scripting,
archives and automation logic.

The constraint is that PocketClaw must not become a Linux distribution. No
PRoot, no apt, no compiler toolchain, no Node or npm, no arbitrary executable
downloads, no pip by default, no native wheel compilation, no shell emulation.
v1 targets an interpreter plus a selected standard library under
PocketClaw-controlled execution.

Python must respect what Managed Runtime already proved on hardware:
**executables ship in the APK and run from `nativeLibraryDir`; writable
executable storage is not used.** Writable Python data may live app-private, and
stdlib resources may ship as non-executable assets.

The APK is currently ~55.6 MB. Exact size projections are required before any
inclusion, and a 100+ MB addition needs explicit approval. Full scope in
`TASKS.md`.

## Provider Resilience & Automatic Failover — PHYSICAL PASS

Branch `feature/provider-resilience-failover`, from `develop` at `0a0b3fa`.
**Physical validation PASSED** on the target ARM64 device on 2026-08-30 and
merged to `develop`. Not released; `main` untouched and no tags moved.

Implementation commits: `68443c1` (resilience), `7b67493` (fallback UI and
automatic gateway restart), `ebf49b4` (restart safety invariants), `812a003`
(resume white-screen recovery), `3446b0f` (diagnostic naming).

### Physical results

| Check | Result |
|---|---|
| Provider automatic failover | **PASS** |
| Fallback Models UI, ordered selection | **PASS** |
| Primary unavailable → configured fallback answered | **PASS** |
| Single user-visible final answer | **PASS** |
| Automatic gateway restart after model config | **PASS** |
| Automatic gateway restart after fallback config | **PASS** |
| Active-turn safety | **PASS** |
| White-screen resume recovery | **PASS** |
| No recovery loop; healthy pages untouched | **PASS** |

The observed restart sequence was: active request running → model configuration
saved → UI entered "Restarting Gateway" → the request finished → gateway
restarted → configuration became active. **The active request was not
interrupted.**

### What was built, and what deliberately was not

The existing `FallbackChain`, `CooldownTracker` and `ClassifyError` were reused
rather than rewritten. There is **no checkpoint subsystem, no semantic tool
fingerprinting and no side-effect classification framework**: the agent loop
already guaranteed that a provider retry does not rewind completed tool
execution, so that property is protected by tests plus one exact-`toolCallID`
result-reuse guard.

Delivered: single-candidate cooldown, `Retry-After` support, hard-quota
distinction from transient throttling, 502/503/504 classified by what each
actually means, a conservative fallback capability gate, streaming failover only
before first visible output, steering and cancellation preserved across retries,
a `provider.*` event family with emitter-level redaction, the Fallback Models UI,
and automatic safe gateway config apply.

**Two restart invariants hold absolutely.** A busy gateway is never force
restarted when the two-minute wait expires, and an unverified busy state is never
treated as idle. Both leave the configuration saved and unapplied, and the manual
Restart Gateway control remains available.

### Counts

Tools **18**. Skills **7/7** on an existing upgraded workspace, **6/6** on a
fresh install. The removed GitHub Skill stays removed, and `picoclaw-agent` stays
unseeded — the count is not a target.

## Lean Runtime Pack v2 — PHYSICAL PASS

Branch `feature/lean-runtime-pack-v2`, fix commit `7ebd254`, from `develop` at
`fa27ad2`. **Physical validation PASSED** on a real ARM64 device on 2026-08-30
and merged to `develop`. Not released. `main` untouched; `v0.2.0-rc1`,
`v0.2.0-rc2` and `phase2-milestone-d` not moved.

### Physical results

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

### Skills

- Existing upgraded workspace: **7/7**
- Fresh install: **6/6**

The two differ legitimately. Seeding only ever writes and never deletes, so a
device that already had the GitHub Skill keeps its seven; a fresh workspace gets
the six seeded skills, because `picoclaw-agent` is deliberately unseeded.

### Known limitation

git is built with its default compiled-in `SHELL_PATH` of `/bin/sh`, which
Android does not have. **Git features that depend on a shell — hooks in
particular, and git's `ENOEXEC` fallback — are not guaranteed on Android.**
Nothing on the clone, fetch or push path needs a shell: the transport helpers
are ELF executables. See `DECISIONS.md` for why overriding it is not possible
without breaking git's own cross-build.

Six bundled tools now ship. Full architecture in `RUNTIME.md`.

| Tool | Version | Installed | License |
|---|---|---:|---|
| git + git-remote-http | 2.51.0 | 6.21 MB | GPL-2.0-only |
| gh | 2.82.1 | 55.9 MB | MIT |
| curl (mbedTLS) | 8.11.1 | 1.30 MB | curl + Apache-2.0 |
| ripgrep | 14.1.1 | 4.27 MB | MIT / Unlicense |
| sqlite3 | 3.50.4 | 1.23 MB | public domain |
| jq (from v1) | 1.7.1 | 0.77 MB | MIT |

APK 34,727,724 -> 58,302,215 bytes. Catalog 44 -> 55 tools.

### Two things a reader should know

**git found its helpers only after it could find itself.** The v2 physical run
failed `git clone` with `unable to find remote helper for 'https'`. The cause was
not TLS or symlinks: git spawns `git remote-https` and resolves the literal name
`git` through PATH, and the helper directory did not contain it. Fixed by
declaring `git` as one of its own helpers; the payloads are unchanged.

**git's transport helper is presented by symlink.** Android cannot package a
file named `git-remote-https`, so the runtime builds a directory of symlinks to
the packaged payloads and points `GIT_EXEC_PATH` at it. Nothing is written into
app storage and executed. This was the **one platform assumption v2 rested on**,
and the device has now settled it: symlink execution works, confirmed end to end
by a real clone. The probe continues to report `symlink_exec` so a device that
behaves differently says so in its own Debug Logs.

**gh's 55.9 MB is a sanctioned exception, not a precedent.** The user accepted it
explicitly because GitHub capability is core to the agent. The size policy still
binds everything else: yq was measured at 11.25 MB and left out on that basis,
since jq already covers JSON.

### Verification

Automated: `go vet` and `go test` green across Core, `flutter analyze` clean,
Flutter tests passing, all seven bundled payloads extracted from the built
release APK hashing to their catalog pins, and the arm64 guard listing every one.

Physical: **PASS**, as recorded above.

## Managed Runtime Foundation — PHYSICAL PASS

Branch `feature/managed-runtime-foundation`, commit `ee236da`, based on
`v0.2.0-rc2` / `404ef44`. Not released. `main` untouched. `v0.2.0-rc1`,
`v0.2.0-rc2` and `phase2-milestone-d` not moved.

**Physical validation: PASS** on a real ARM64 Android device, 2026-08-30.

That run covered the runtime tool registering; jq 1.7.1 executing and processing
JSON; `sha256sum`, `grep`, `sed`, `tar`, `uname`, `df` and `ping` executing;
stderr captured; a non-zero exit code preserved; a timeout terminating a harmless
long-running command; runtime lifecycle events appearing in the logs; no secret
leakage observed; the runtime driven end to end through Telegram; Service and
Gateway Auto-Start still working; and no recurrence of the Gateway PID ownership
false positive.

### Physical runtime catalog

- **43 of 44** catalog tools available on the tested device.
- Unavailable: `traceroute`, correctly reported as such rather than assumed
  present. This is the resolver doing its job: availability is measured per
  device, never read from the catalog.
- Bundled: jq 1.7.1 (`libpocketclaw-jq.so`), resolved and executed from
  `nativeLibraryDir`.

### Writable-app-data probe: INCONCLUSIVE on the tested device

The probe could neither execute its staged copy nor observe a clean permission
refusal, so it reported `inconclusive` with its reason, which is the honest
outcome rather than a guess in either direction.

This changes nothing. **The architecture does not depend on writable executable
app storage.** Executable delivery remains exactly two routes, both read-only to
the app:

1. Android system executables in `/system/bin`.
2. APK payloads the package manager unpacks into `nativeLibraryDir`.

An inconclusive probe is therefore information, not a blocker: the bundled jq
payload executed from `nativeLibraryDir` on the same device, which is the path
the runtime actually uses.

### Counts

- Tools: **18**
- Skills: **7/7**

Skills 7/7 is the current expected value and **not a regression**. The
incomplete GitHub Skill was intentionally removed by the user. It must not be
restored and 8/8 must not be treated as the target.

The PocketClaw Managed Runtime gives the Agent a controlled, observable, verified
local tool environment: `core/src/pkg/pcruntime` plus a `runtime` agent tool.
Architecture, storage layout, observability contract and the tool catalog are
documented in `RUNTIME.md`.

### The constraint this milestone established

PocketClaw targets Android SDK 36, and an app targeting API 29+ cannot execute a
file in its own writable storage — `setExecutable` does not change that. The
earlier plan for an app-private `runtime/bin` is invalid and has been replaced.
Executables reach the device only through `/system/bin` or through APK payloads
the installer unpacks into `nativeLibraryDir`, so the runtime contains no
download-and-execute path at all. See `DECISIONS.md`.

### Runtime Pack v1

- Tier 1 catalogued as system-provided and probed per device; Android already
  ships toybox, so BusyBox is deliberately not bundled.
- jq 1.7.1 bundled as `libpocketclaw-jq.so`, cross-built from the pinned official
  release tarball, proving the APK/`nativeLibraryDir` packaging contract.
- `curl`, `wget`, `openssl`, `git` and `gh` are not shipped; `TASKS.md` records
  why each is hard.

### Agent tool count

17 -> 18. All 17 existing tools are unchanged; the addition is `runtime`.

### Verification

Automated: `go vet` and `go test` green across Core, `flutter analyze` clean,
Flutter tests passing, and the jq payload extracted from the built release APK
hashing to its catalog pin, proving Gradle packaging leaves it byte-identical.

Physical: PASS, as recorded above.

## v0.2.0-rc2 — Auto-Start and Gateway PID ownership (PHYSICAL PASS)

Release candidate 2, published as a GitHub pre-release. Not a production
release and not published to Google Play.

Physical validation: **PASS** on a real ARM64 Android device, 2026-08-30.
Physical reference APK SHA-256:
`182b85183156428aa93baf3113492484258a3a1eace95f7bccd9ed82035177a3`

That run covered fresh install, Service Auto-Start, Gateway Auto-Start, manual
Service stop and start, Gateway starting automatically after the Service, no
immediate Service resurrection, internal PocketClaw chat, Core startup, Skills
8/8, Tools 17, Core bound only to `127.0.0.1:18790` / `[::1]:18790`, a working
Dashboard, and no recurrence of the Gateway PID ownership false positive.

- Auto-Start Safe Rebuild (`75ac0d9`): **PHYSICAL PASS**
- Gateway PID ownership fix (`90194c8`): **PHYSICAL PASS**
- `feature/autostart-foundation`: **NOT MERGED**, reference only. The shipped
  work was rebuilt from `v0.2.0-rc1` rather than salvaged from that branch.

Project: PocketClaw  
Current Phase: Phase 2 — Independent Product Repository  
Current Milestone: Phase 2 Milestone D — Telegram Managed-Bot Onboarding.
**COMPLETE.** Physically verified end to end on a real Android device on
2026-08-26, including a live Telegram → PocketClaw → AI provider → Telegram
message round trip against the production onboarding service.
Milestone C — Provider Catalog + Easy API-Key Setup, the OpenCode completion,
and the self-contained source migration — is complete and PASSED
physical-device testing on 2026-08-25, merged to `develop` as `36bc88d` and
tagged `phase2-milestone-c`.
Git Branch: `fix/user-facing-log-privacy`. `feature/telegram-managed-onboarding`
was merged with a non-fast-forward merge and is retained intact.
Last verified milestone: Phase 2 Milestone D (tag `phase2-milestone-d`).
Milestone C (merge `36bc88d`, tag `phase2-milestone-c`) is the fallback
reference state, with Milestone B (merge `225be3c`, tag `phase2-milestone-b`)
retained below it.
Recovery Branch: `recovery/pocketclaw-clean-debrand` @ `f25d38e`, retained intact
Foundation Bootstrap Commit: `950d4a3`  
Origin: `https://github.com/Lord1Egypt/PocketClaw.git` (private)  
Upstream FUI Baseline: `d689c94c1b67f625f70ec4111a9aa3f01be9cbb3`  
PicoClaw Core: `v0.3.1`, source `2cf030d2fd3b871d7ec17e3be34c24688aac76da`,
vendored into this repository at `core/src/` and built from there — see
`core/README.md` and `UPSTREAM_BASELINE.md`  
Source-of-Truth: this repository. A clone contains all application and runtime
source; no external checkout is a build dependency. Proven by
`core/verify-no-external-source.sh`.  
Build Status: arm64 release APK built through the canonical Gradle path; the
release guard verified the arm64 native payload.
APK Status: the Milestone D APK
`b6fea5d8ec5c3c66ba8a1320b0a217afcca322e75b5b26cc4082bbbb08a57f94`
PASSED physical-device testing on 2026-08-26 and remains the verified
milestone reference artifact. The broader pre-release candidate
`f663d25a2fffb0ce969ad4a9ce3405c1e563b6263c7af37e90768eef471c621d`
then physically confirmed the neutral native Telegram shortcut, basename-only
callers, user-facing log debranding, terminal-control removal, and exactly-once
queue/drain behavior. A DEBUG export exposed two smaller defects; their
replacement candidate
`eacbbc86b99429f114aba9ba1dca57224122fa176f6b4d99edf350454423f9a8`
is BUILT, AUTOMATED PASS, and NOT yet physically verified.
Verified Core binaries (device-verified 2026-08-26, in the reference APK):
`libpicoclaw.so` 37,224,801
`33f8b4efbc88333747c5df30b3ddc6864c924b35dba99e9c8c3b91df3470e98a`;
`libpicoclaw-web.so` 24,641,889
`5400cb02322ece5c7035356595355bd3c116adfc6a5bb6b78f3e2bd22dbcb3bd`.
The DEBUG-cleanup candidate carries a NEW, not-yet-verified pair:
`libpicoclaw.so`
`5c09eb72a1faa6dd6f8a0e6b66bcbc028f070d3eff9b8c5b5f87716c04d763bc`;
`libpicoclaw-web.so`
`cb6b10cc2a951d2307959e9effbdbd052762fe827b9943c22a9cdcdd54b03b52`.
Current Blocker: physical DEBUG-export validation and the remaining broader
pre-release device sweep. The GitHub release is **on hold** until both pass.
Next Exact Action: install `eacbbc86...423f9a8`, export a new DEBUG log, and
confirm valid `53.616µs` plus no successful `/api/gateway/logs` or
`/api/gateway/status` poll noise. `main` remains deliberately at `100a51d`.

## Completed

- Phase 1 baseline APK and all physical-device checks.
- Created a separate PocketClaw directory without upstream Git history.
- Selectively adapted the Android/Flutter foundation, tests, tools, and Core
  packaging from the reviewed FUI baseline.
- Preserved the Android DNS and optional-feedback behaviors.
- Recorded provenance, upstream tracking, and third-party notices.
- Audited and recorded the FUI/Core MIT notices and component classification.
- Ran `flutter analyze` (clean), `flutter test` (28 passed), and focused Core
  Android DNS/model API tests (passed).
- Built and inspected the independent arm64 foundation APK. Its embedded Core
  hashes match the pinned replacement binaries; Firebase build values are absent.
- Completed a Git diff/stat/check review and credential scan. No credentials,
  signing files, Firebase config, generated APKs, or caches are eligible for
  commit; the standard Gradle Wrapper files are intentionally retained.
- Created the private GitHub repository `Lord1Egypt/PocketClaw`, pushed
  `main`, and created/pushed the `develop` integration branch.
- Physical-device verification confirmed the independent foundation APK,
  launch, Core/Gateway, active-network DNS, model discovery/manual model,
  AI requests, Telegram, restart/persistence, and optional feedback behavior.
- Physical-device verification confirmed ClawHub Skill Hub search: a `Crypto`
  query returned 20 results with metadata, URLs, and install actions. The
  prior registry-unavailable observation is classified as resolved by the
  Android DNS fix, not as an independent Skill Hub defect.
- Started Milestone B on `feature/pocketclaw-identity`: product naming,
  independent package identity, original visual direction, Android icon/splash
  treatments, design tokens, and the required audit records are complete.
- Passed Milestone B `flutter analyze`, all 28 Flutter tests, and focused
  pinned-Core Android DNS/model API tests. Built and inspected the new APK:
  package/label are `com.lord1egypt.pocketclaw`/PocketClaw, Android branding
  resources are bundled, the Core hashes match the pin, and no Firebase config
  values are present.
- Committed Milestone B as `34b0f6b` and pushed
  `feature/pocketclaw-identity` to the private `origin`; `develop` and `main`
  remain untouched pending the user's physical-device approval.
- Root-caused the Milestone B black-screen regression to the two branded Android
  `launch_background.xml` resources. A `layer-list` `<item android:color>` is
  not a valid drawable layer: Android must inflate a drawable-backed item before
  Flutter can replace `LaunchTheme`. The corrected resources use
  `@color/pocketclaw_splash_background` through `android:drawable`.
- Added a source-level regression test for both launch-background variants.
  `flutter analyze` is clean; all 29 Flutter tests and focused Core
  `pkg/androiddns`/`web/backend/api` regressions pass. A debug and a replacement
  arm64 release APK were built and the release's compiled layer-list was
  inspected to confirm the first item has a drawable reference.

- Milestone C: audited the provider architecture end to end and recorded it in
  `docs/PROVIDER_ARCHITECTURE.md`. Established that all AI provider
  configuration lives in the Core web console, not in Flutter, and that the
  provider catalog is already backend-owned by `pkg/providers`.
- Milestone C: extended the backend-owned catalog with `category` and
  `documentation_url`, added the xAI, Together AI, Fireworks AI, and
  Custom OpenAI-Compatible presets, and registered all four in the protocol
  switch so they actually dispatch at runtime.
- Milestone C: enabled Gemini model discovery with a dedicated fetch branch
  that uses `X-Goog-Api-Key` for the native base and Bearer for the
  OpenAI-compatible base, and broadened fetch error classification to cover
  rate limiting, provider outage, and a missing listing endpoint.
- Milestone C: replaced the Add Model form with a two-step provider-first flow
  — choose provider, paste API key, fetch or type a model, save — deriving the
  model alias automatically and moving base URL, alias, and optional keys into
  Advanced. Local and custom providers keep a visible base URL.
- Milestone C: removed runtime logo fetching from `cdn.simpleicons.org` and
  Google's favicon service; provider marks are now rendered locally.
- Milestone C: added 22 frontend tests (new vitest runner), 8 Go catalog tests,
  and 5 Go model-discovery tests. `flutter analyze` clean, 27 Flutter tests,
  Go suites for providers/config/api/androiddns/mqtt/onboard/commands/agent all
  pass, and the frontend type-checks and lints clean.

## Constraints

- Do not modify the Phase 1 workspace or its verified APK.
- Do not merge upstream repositories automatically.
- Preserve PicoClaw Core protocol/binary/environment identifiers and the
  compatible `Downloads/picoclaw` workspace path while product identity changes.
- Do not commit credentials, signing material, generated APKs, or caches.
- Do not merge `feature/provider-catalog` to `develop` before the user's
  physical-device approval, and do not touch `main`.
- Do not start Telegram QR/deep-link onboarding: it is the next milestone.
- Do not enable obfuscation or anti-reverse-engineering during active feature
  development; release hardening is a later pre-release milestone.

## Independent Foundation APK

- Path: `build/app/outputs/apk/release/app-release.apk` (ignored; not committed)
- Built: 2026-08-24 03:06:47 +03:00
- Size: 32,619,416 bytes
- SHA-256: `207a5e4623b6c6ae295a29a874c7a7d6ac9a17552e2ddb2511daa093b085fa4d`
- Package/version: `com.sipeed.picoclaw`, `0.1.3` (version code `3`)
- Label: `PicoClaw` (intentionally unchanged for this foundation milestone)
- Architecture: functional application/Core payload is `arm64-v8a`

## Milestone B PocketClaw APK

- Path: `build/app/outputs/apk/release/app-release.apk` (ignored; not committed)
- Built: 2026-08-24
- Size: 34,096,308 bytes
- SHA-256: `0e440d2804978e6f94550a9d0cab563328d03cd32312bd4e33ec1ecdfc6d4883`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (version code `3`)
- Label: `PocketClaw`
- Architecture: universal APK; the PicoClaw Core payload is `arm64-v8a` and
  its two pinned hashes match `UPSTREAM_BASELINE.md`.

## Milestone B Runtime-Regression Replacement APK

- Status: BLOCKED — physical-device retest pending.
- Path: `build/app/outputs/flutter-apk/app-release.apk` (ignored; not committed)
- Built: 2026-08-24 04:34:40 +03:00
- Size: 33,308,653 bytes
- SHA-256: `45be7269af920df4a36eb4eb37171770bbcfa242ed7c071da28874d9c27ebe9e`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (version code `3`)
- Label: `PocketClaw`
- Core payload hashes: unchanged — gateway
  `3b849072a7c2858b0d2c0db5cbcfa42b542353e834f4c473399eda571ab16f3d`, web
  `252b38c64cbc4dc52277c206ca1b069cc7c3bb97b8a9c276e23f8edc3aaf95e3`.

## Stage B Debranded APK — PHYSICALLY VERIFIED (previous reference)

- Status: PASS on a physical Android device, 2026-08-25. Not merged to `develop`.
- Path: `build/app/outputs/apk/release/app-release.apk` (ignored; not committed)
- Also copied to `build/app/outputs/flutter-apk/app-release.apk` (identical).
- Built: 2026-08-25 from `recovery/pocketclaw-clean-debrand` with
  `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64`,
  `JAVA_HOME=/home/lordegypt/PocketCLaw/.tooling/jdk-17`,
  `GRADLE_USER_HOME=.tooling/gradle-stage-a-clean`, Flutter 3.47.1 / Dart 3.13.1.
- Size: 34,119,837 bytes
- SHA-256: `2717f32e9580cd5b5ea5da70b2cb9fcf13f6f14451423addcb5686e0278a1de4`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (version code `3`)
- Label: PocketClaw; launchable `com.lord1egypt.pocketclaw.MainActivity`
- arm64 payload verified by the release guard: `libdartjni.so` (131,248),
  `libpicoclaw.so` (37,421,409), `libpicoclaw-web.so` (24,772,961).
- Embedded Core hashes: gateway
  `1f239a827c8562d6ac2ffdf63c1354ce0d28396cdab7d4f240f3866cbb525fed`, web
  `94bb6319bbac08e1aa0fa43e8093b4dd00bad512cb67ca94a6a57d666f4bc716`.
- Physical-device results (2026-08-25): install PASS, app launch PASS, Flutter
  first frame PASS, black-screen regression FIXED, Gateway/Core startup PASS,
  navigation PASS, PocketClaw branding PASS, PocketClaw workspace path PASS,
  QR/access page PASS, no abnormal device slowdown observed.
- This is the reference physically verified PocketClaw artifact. Compare any
  future build against it.

## Pre-release APK check (mandatory)

`packageRelease` now fails the build if `lib/arm64-v8a/` is missing
`libdartjni.so`, `libpicoclaw.so`, or `libpicoclaw-web.so`, and prints a
"Verified arm64-v8a native payload" line when it passes. If `libdartjni.so` is
reported missing, purge `~/.pub-cache/hosted/pub.dev/jni-*/android/.cxx/` and
rebuild — that cache is outside the project `build/` tree, so cleaning build
intermediates does not clear it.

`./gradlew :app:assembleRelease -Ptarget-platform=android-arm64` is the
canonical release path. Do not release a universal `flutter build apk --release`.

## Milestone B Final Cleanup APK — VERIFIED REFERENCE ARTIFACT

- Status: PASS on a physical Android device, 2026-08-25. Merged to `develop`.
- Path: `build/app/outputs/apk/release/app-release.apk` (ignored; not committed)
- Also copied to `build/app/outputs/flutter-apk/app-release.apk` (identical).
- Built: 2026-08-25 with the canonical command
  `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64`.
- Size: 34,119,477 bytes
- SHA-256: `ba4f067df9811bd0e4af713343bdba632abbf96a41e3a5b47cf154740f70a4b8`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (version code `3`)
- Label: PocketClaw; launchable `com.lord1egypt.pocketclaw.MainActivity`
- ABIs advertised: `arm64-v8a`, `armeabi-v7a`, `x86_64`. Flutter and Core
  payloads are `arm64-v8a`; the other ABIs carry plugin JNI libs only.
- Release guard PASS for all three required libraries:

| Packaged library | Size | SHA-256 |
| --- | --- | --- |
| `libdartjni.so` | 131,248 | `47dae44db1c6202d164c0bb2ff25cc661023ba2904a6679abad4f3dcf3fcb5cd` |
| `libpicoclaw.so` | 37,421,409 | `eb895f0892509b76242f572515c26f56530ec417bdedc0bb9ec1486f40bd9c88` |
| `libpicoclaw-web.so` | 24,772,961 | `6d282df06680869a0aca25a976b123bce8e793d2f08708e79386a1761195a5a3` |

- Contents of this cleanup: MQTT fresh default is `/pocketclaw` while any
  explicitly configured prefix (including the legacy `/picoclaw`) is preserved;
  `skills/picoclaw-agent` is no longer seeded into a fresh workspace while
  existing user copies are untouched; factual Sipeed hardware references are
  retained deliberately.
- Both Core binaries are stripped with 0 debug sections, and
  `PICOCLAW_DNS_SERVER` is verified present in the rebuilt gateway.
- Physical-device results (2026-08-25): install PASS, app launch / first frame
  PASS, no black screen, Gateway/Core lifecycle PASS, navigation PASS,
  PocketClaw branding PASS, workspace path PASS, QR/access page PASS, Skill Hub
  PASS, provider/model flow PASS, no abnormal slowdown.
- This is the current verified reference artifact. Compare any future
  regression against it before forming new hypotheses.

## Milestone C Provider Catalog APK — PHYSICALLY VERIFIED (superseded)

- Status: PASS on a physical Android device, 2026-08-25. This is the verified
  reference artifact, superseding Milestone B's `ba4f067d...70f70a4b8`.
  Not merged to `develop`: the OpenCode completion below rides on the same
  branch and must pass its own device test first.
- Path: `build/app/outputs/apk/release/app-release.apk` (ignored; not committed)
- Built: 2026-08-25 with the canonical command
  `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64`,
  `JAVA_HOME=/home/lordegypt/PocketCLaw/.tooling/jdk-17`,
  `GRADLE_USER_HOME=/home/lordegypt/PocketClaw-App/.tooling/gradle-stage-a-clean`.
- Size: 34,123,401 bytes
- SHA-256: `b3dd892bdea86e8dfe7d1c2eb87e89f4e2832b1d1dbe39fc1decf20dabce569b`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (version code `3`)
- Label: PocketClaw; launchable `com.lord1egypt.pocketclaw.MainActivity`
- ABIs advertised: `arm64-v8a`, `armeabi-v7a`, `x86_64`. Flutter and Core
  payloads are `arm64-v8a`; the other ABIs carry plugin JNI libs only.
- Release guard PASS for all three required libraries:

| Packaged library | Size | SHA-256 |
| --- | --- | --- |
| `libdartjni.so` | 131,248 | `47dae44db1c6202d164c0bb2ff25cc661023ba2904a6679abad4f3dcf3fcb5cd` |
| `libpicoclaw.so` | 37,421,409 | `cbe568af0d6e0a1e3e4e48f7ab53fa00300509dc04f5d6ee07d0465e5556468a` |
| `libpicoclaw-web.so` | 24,772,961 | `86e53457468c6c53f6c8814b4345fcfe1ec7026e3ded388d2ab305c10cb0a4cd` |

- `libdartjni.so` is byte-identical to the Milestone B verified build.
- Both Core binaries are stripped with 0 debug sections, and
  `PICOCLAW_DNS_SERVER` is verified present in the rebuilt gateway.
- Branding invariants hold in the rebuilt launcher: 0 `PicoClaw`, 0 `Sipeed`,
  31 `PocketClaw`. `google_app_id` is present but empty, which is the expected
  `cleanupFirebaseResources` outcome; no Firebase credential value ships.
- What to test on the device: the Add Provider picker opens and lists providers
  by category; selecting a cloud provider asks only for an API key and a model;
  Fetch Models succeeds against a real provider; a failed fetch still allows a
  manually typed model ID; Custom OpenAI-Compatible accepts a base URL, key, and
  model ID; an existing provider still opens with its stored values intact; and
  startup, DNS, Core lifecycle, Telegram, Skill Hub, workspace, MQTT, and
  branding are all unregressed.

## Milestone C OpenCode Completion APK — RETIRED, NEVER TESTED

- Status: RETIRED without ever being tested. Its functionality is contained in
  the physically verified source-migration APK `588bbec1...5ff3a90b`, which
  supersedes it. Kept here only as a record of what was built.
- Path: `build/app/outputs/apk/release/app-release.apk` (ignored; not committed)
- Built: 2026-08-25 with the canonical command
  `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64`,
  `JAVA_HOME=/home/lordegypt/PocketCLaw/.tooling/jdk-17`,
  `GRADLE_USER_HOME=/home/lordegypt/PocketClaw-App/.tooling/gradle-stage-a-clean`.
- Size: 34,129,765 bytes
- SHA-256: `785ccd94cfa351ee2996ac340f9a55e828a0c8f736bec67a3edac906a56058c6`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (version code `3`)
- Label: PocketClaw; launchable `com.lord1egypt.pocketclaw.MainActivity`
- ABIs advertised: `arm64-v8a`, `armeabi-v7a`, `x86_64`. Only `arm64-v8a`
  carries `libapp.so`, `libflutter.so`, and the two Core payloads; the other
  ABIs carry plugin JNI libs only.
- Release guard PASS for all three required libraries:

| Packaged library | Size | SHA-256 |
| --- | --- | --- |
| `libdartjni.so` | 131,248 | `47dae44db1c6202d164c0bb2ff25cc661023ba2904a6679abad4f3dcf3fcb5cd` |
| `libpicoclaw.so` | 37,421,409 | `e48e8af073d6e7dfdb46ba8268785780d1f900888b82dba41747ef9212e78938` |
| `libpicoclaw-web.so` | 24,772,961 | `5faaf82ccbcd2fbad27d7ffc336f240fd5c08a48a1abbb2bd4eff7c383fe2abf` |

- `libdartjni.so` is byte-identical to every verified build since Milestone B.
- Both Core binaries are stripped with 0 debug sections, and
  `PICOCLAW_DNS_SERVER` is verified present in the rebuilt gateway.
- Branding invariants hold in the rebuilt launcher: 0 `PicoClaw`, 0 `Sipeed`,
  31 `PocketClaw`. `google_app_id` is present but empty, the expected
  `cleanupFirebaseResources` outcome; no Firebase credential value ships.

### What to test on the device

The rest of the app is unchanged from the verified Milestone C build, so the
regression sweep can be brief. The new surface is the two OpenCode presets:

1. OpenCode Zen appears in Add Provider, asks only for an API key, and Fetch
   Models returns the live list from `https://opencode.ai/zen/v1/models`.
2. OpenCode Go does the same against `https://opencode.ai/zen/go/v1/models`.
3. Run one real inference on each protocol family, per provider, because the
   protocol is chosen per model and only a live request proves the route:
   - a Responses-family model (`gpt-*`, `*codex*`),
   - an Anthropic Messages-family model (`claude-*`),
   - a chat-completions-family model (`kimi-*`, `deepseek-*`, `glm-*`).
4. Confirm the model ID saved and sent is the bare ID (`kimi-k3`), not the
   namespaced OpenCode CLI form.

The one assumption that only a device can settle is the Messages
authentication form — see the OpenCode routing decision in `DECISIONS.md`.
If a `claude-*` model returns 401 while `gpt-*` and `kimi-*` succeed, that is
the bearer-versus-`X-API-Key` question, not a routing failure.

## Self-Contained Source Migration APK — PHYSICALLY VERIFIED REFERENCE ARTIFACT

- Status: PHYSICALLY VERIFIED on 2026-08-25. This is the current verified
  reference artifact, superseding Milestone C's `b3dd892b...bce569b`. Bisect or
  diff any future regression against it before forming new hypotheses. It
  carries the OpenCode completion, so `785ccd94...56058c6` is retired untested.
- Path: `build/app/outputs/apk/release/app-release.apk` (ignored; not committed)
- Built: 2026-08-25 with the canonical command
  `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64`,
  `JAVA_HOME=/home/lordegypt/PocketCLaw/.tooling/jdk-17`,
  `GRADLE_USER_HOME=/home/lordegypt/PocketClaw-App/.tooling/gradle-stage-a-clean`,
  after building Core from `core/src/` with `core/build-android-arm64.sh`.
- Size: 34,123,225 bytes
- SHA-256: `588bbec144fe0c84b8429f4f053a73b44b9b3e8d9f24e31dab04b2165ff3a90b`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (version code `3`)
- Label: PocketClaw; launchable `com.lord1egypt.pocketclaw.MainActivity`
- Release guard PASS for all three required libraries:

| Packaged library | Size | SHA-256 |
| --- | --- | --- |
| `libdartjni.so` | 131,248 | `47dae44db1c6202d164c0bb2ff25cc661023ba2904a6679abad4f3dcf3fcb5cd` |
| `libpicoclaw.so` | 37,224,801 | `cb9b2cdea1ccd7ddbbda723ddd3bed1d8c3a931638b1952dd767f62efb895818` |
| `libpicoclaw-web.so` | 24,641,889 | `b6b356f75eb348933e6cb1049890bb8be4d2bd20b605f55b23a5487656db9ba5` |

- `libdartjni.so` is byte-identical to every verified build since Milestone B.
- Both Core binaries are stripped, `ARM aarch64` PIE, and are the first built
  from repository-local source and the first built with `-trimpath`. They are
  roughly 197 KB and 131 KB smaller than the previous pair for that reason.
- Developer-machine paths in the packaged Core binaries: 0 and 0. The previous
  pair carried 2,501 and 1,346. `core/build-android-arm64.sh` fails the build
  if this regresses.
- `PICOCLAW_DNS_SERVER` is verified present in the rebuilt gateway.
- Branding invariants hold in the rebuilt launcher: 0 `PicoClaw`, 0 `Sipeed`,
  31 `PocketClaw` — identical to the verified Milestone C counts.
- Pre-build validation: Go suites 92 ok / 0 failed, frontend `vitest` 28
  passed, `pnpm lint` clean, `flutter analyze` no issues, `flutter test` 27
  passed.

### Physical-device results — 2026-08-25, all PASS

| Check | Result |
| --- | --- |
| Install / startup | PASS |
| Flutter first frame | PASS |
| Gateway/Core lifecycle | PASS |
| User-facing logs free of developer absolute paths | PASS |
| User-facing logs free of PicoClaw product branding | PASS |
| Provider catalog | PASS |
| OpenCode Zen preset | PASS |
| OpenCode Zen Fetch Models | PASS |
| OpenCode Zen real request/response | PASS |
| OpenCode Go preset | PASS |
| OpenCode Go Fetch Models | PASS |
| OpenCode Go real request/response | PASS |
| Gemini / provider regression | PASS |
| Skill Hub regression | PASS |
| Telegram regression | PASS |
| Workspace regression | PASS |
| No black screen | PASS |
| No abnormal slowdown | PASS |

Three results carry more weight than the rest.

The logs check is the first device confirmation of the `-trimpath` change. The
build-time assertion proved the strings were absent from the binaries; this
proves nothing surfaces them on the screen a user actually reads.

Skill Hub search and Fetch Models both working is the end-to-end proof that the
Android active-network DNS integration survived being rebuilt from a relocated
source tree. Those paths fail closed without working DNS, so a PASS on both
means the integration is intact — not merely present as a string in the binary.

The four OpenCode results are the first live confirmation of per-model protocol
routing. Zen and Go each returned a real response, so the routing table in
`pkg/providers/opencode_routing.go` is exercised rather than assumed.

One question stays open, and this PASS does not close it. `DECISIONS.md`
records that OpenCode's Anthropic Messages surface is sent both `X-API-Key` and
a bearer header because the correct form could not be established offline. That
is only settled by a `claude-*` model returning a real response, and the device
report does not say which model families were exercised. Treat the Messages
route as unconfirmed until a `claude-*` inference is observed; if one 401s while
`gpt-*` and `kimi-*` succeed, the header pair is the cause, not the routing.

## Post-Milestone-D device regressions — FIXED, AWAITING PHYSICAL DEVICE

Three defects found on the device after Milestone D closed, on branch
`fix/user-facing-log-privacy` (not merged, not tagged). All three are
pre-release blockers; the GitHub milestone release is on hold until the
replacement APK passes a device re-test.

### Bug 1 — the caller leaked the upstream module path

The Logs screen showed:

    WRN api github.com/sipeed/picoclaw/web/backend/api/gateway.go:298 >
    removed stale pid file for PID 12302

Root cause: `pkg/logger/logger.go` builds its zerolog logger with `.Caller()`,
which reports the path the **compiler** recorded. The earlier `-trimpath` work
did exactly what it was asked — it removed `/home/lordegypt/...` — but what
replaces an absolute path under `-trimpath` is the Go **module path**. So the
fix for one leak created another, and no check caught it because every existing
assertion looked for `/home/`.

Fixed at the structured layer, in `logger.init()`, by setting
`zerolog.CallerMarshalFunc` to `ShortCallerLocation`, which reduces any
recorded caller to `file.go:line`. This is the earliest point that sees caller
metadata, so every writer, every level, and every exported log file inherits
the short form. No message text is rewritten and no arbitrary file path inside
a message is touched.

    WRN api gateway.go:298 > removed stale pid file for PID 12302

`ShortCallerLocation` normalises `\` as well as `/`. `filepath.ToSlash` only
rewrites the *host* separator, so a Windows-recorded caller would have passed
straight through a Linux build — the test caught this.

Eight user-visible message strings that named the project were also reworded
(not blind-replaced; each was read and edited individually):
`pkg/pid/pidfile.go`, `web/backend/api/gateway.go`,
`web/backend/utils/runtime.go`, `web/backend/main.go`, and four provider
credential errors that told users to run a CLI command that does not exist on
Android.

### Bug 2 — one event rendered as hundreds

Not repeated backend emission, and not a lifecycle bug. The device screenshot
was decisive: every duplicate carried the **identical timestamp** `02:23:44`
and the identical PID, and the event counter read the full 500.

Root cause: `PicoClawService.lastLog` is a **sticky snapshot** of the most
recent line that never clears. `ServiceManager._syncNativeServiceStatus`, which
runs on a three-second timer, appended that snapshot on every tick. One warning
was therefore re-added every three seconds until it filled the 500-entry buffer
and **evicted the entire real log history** — the damage was not cosmetic.

`RemovePidFileIfPID` was audited and is correct: it returns true only after
actually reading, matching, and removing the file, so it cannot have fired
repeatedly for one PID.

Fixed by making the producer match the contract the consumer needs, rather than
by deduplicating text. `PicoClawService` now funnels every line through
`publishLog`, which appends to a bounded pending queue; `takeNewLogs` drains
it, so each line is handed out exactly once. This also fixes a second defect the
snapshot hid: lines emitted **between** polls used to be lost, because only the
most recent one was ever read.

### Bug 3 — the two surfaces disagreed about whether Telegram was connected

The embedded console correctly showed Connected with the bot handle, while the
native Settings card still read "Connect PocketClaw to Telegram" and opened a
page whose primary action was a brand-new pairing. Same class of defect as the
Milestone D UI bug: two surfaces, no shared source of truth.

Root cause: the native card's subtitle was a **hardcoded constant**
(`TelegramOnboardingStrings.introHeadline`) and its tap handler went straight
to onboarding. It never read any state. The console, by contrast, derived
`configured` from Core's own `detectConfiguredSecrets`.

Fixed by giving both surfaces one canonical source. `TelegramConnectionReader`
(`lib/src/telegram/telegram_connection_status.dart`) reads the persisted
`channel_list.telegram` entry — a non-empty `settings.token`, plus `allow_from`
for owner scoping — which is exactly the state Core reports to the console. No
"connected" boolean is stored anywhere, so nothing can drift. It fails closed:
unreadable or malformed configuration reports disconnected.

- `TelegramSettingsCard` renders from that state and re-reads after the flow
  returns and on app resume, so a change made in the console is picked up.
- `TelegramConnectedPage` is what an already-connected user now opens: bot
  handle, owner, Open Chat, Reconnect / Create New Bot, Advanced / Manual.
- `TelegramOnboardingLauncher.open` is state-aware; `startPairing` is the
  explicit pairing entry. The console's Connect and Reconnect buttons call
  `startPairing`, since both are deliberate user requests.
- Reconnect asks first, and the confirmation states plainly that the current
  bot keeps working until a new one is ready.

Replacement is already safe and was verified rather than assumed:
`TelegramConfigWriter.apply` is the only thing that mutates
`channel_list.telegram`, and the controller calls it only after the new token
has been received. A cancelled or expired pairing therefore never reaches the
writer and the existing bot stays configured.

The bot handle is not stored by Core, so a manually configured bot — or one
paired on another device — shows as Connected without a handle and hides Open
Chat, rather than inventing one.

14 tests in `test/widgets/telegram_state_sync_test.dart` cover cases A–G
against real configuration JSON and the real widgets. Four of them fail against
the old hardcoded card, which was confirmed before they were accepted.

### Tests — confirmed to fail against the old code

- `core/src/pkg/logger/caller_sanitize_test.go` — the exact device caller plus
  absolute, `.upstream`, Windows, already-short and degenerate inputs, and an
  end-to-end case that logs through the real logger into a real file and
  asserts the emitted `caller` field carries no module path and no separator.
- `test/unit/service_manager_log_stream_test.dart` — drives the real polling
  path. Against the old code 25 polls produced **25 copies** of one event; it
  now produces one. Also asserts a genuinely repeated line is *not* collapsed,
  that bursts between polls all arrive, and that no rendered line carries a
  module or developer path.

`ServiceManager` is a singleton, so these tests assert on the lines each test
appended rather than on the whole list.

### Verification

`flutter analyze` clean, 93/93 Flutter tests, 36/36 frontend, `tsc -b` clean,
`pnpm lint` clean, and Go tests green for `pkg/logger`, `pkg/pid`,
`pkg/providers/...` and `web/backend/...`.

Core rebuilt (`-trimpath` verified, zero developer paths in both binaries) and
`core/pocketclaw-core-v0.3.1.patch` regenerated — now 67 files.

Candidate APK `543c759b04b0e4c77dd7831435753aceac0b1e16a7a45fdec3fb37ed2e45479a`,
34,227,525 bytes, `com.lord1egypt.pocketclaw` 0.1.3 (3), guard PASS. Secret scan
clean: 0 tokens, 0 webhook/pairing secrets, 0 Redis credentials. The onboarding
endpoint and the Telegram onboarding UI are both present and unchanged.

### Known, pre-existing, not a regression

`libapp.so` contains one developer path:
`file:///home/lordegypt/PocketClaw-App/.dart_tool/flutter_build/dart_plugin_registrant.dart`.
It is the Flutter-generated plugin registrant's source URI baked into the Dart
AOT snapshot, it is present in the **device-verified** `b6fea5d8...` APK and in
every earlier one, and it reaches a user only inside a Dart stack trace, not
the Logs screen. Go's `-trimpath` has no Dart equivalent. Recorded rather than
fixed, because it is out of this fix's scope and needs its own decision.

`Run: picoclaw auth login --provider <name>` remains in `libpicoclaw.so`, in
`cmd/picoclaw/internal/auth/helpers.go`. It is printed by the CLI `auth list`
subcommand, which the Android app never invokes — the app runs the gateway —
and on desktop the binary genuinely is named `picoclaw`, so rewording it would
make the instruction wrong. Left accurate deliberately.

The compiler's embedded source-path table still contains
`github.com/sipeed/picoclaw/...` inside the binaries. That is unavoidable Go
metadata for stack traces and is explicitly permitted; what matters is that it
no longer reaches rendered caller metadata.

## Phase 2 Milestone D — Telegram Managed-Bot Onboarding — COMPLETE

**Status: COMPLETE. Physical end-to-end test PASSED on a real Android device,
2026-08-26.**

Automatic managed-bot onboarding is now the default Telegram setup path in
PocketClaw. Manual Bot Token entry remains available as the advanced fallback.

### Verified artifacts

| | |
| --- | --- |
| Verified APK | `b6fea5d8ec5c3c66ba8a1320b0a217afcca322e75b5b26cc4082bbbb08a57f94` |
| APK size | 34,220,929 bytes, `com.lord1egypt.pocketclaw` 0.1.3 (version code 3) |
| `libpicoclaw.so` | 37,224,801 bytes, `33f8b4efbc88333747c5df30b3ddc6864c924b35dba99e9c8c3b91df3470e98a` |
| `libpicoclaw-web.so` | 24,641,889 bytes, `5400cb02322ece5c7035356595355bd3c116adfc6a5bb6b78f3e2bd22dbcb3bd` |
| Official manager | `@PocketClawSetupBot` |
| Onboarding service | `https://pocketclaw-telegram-setup-bot-83ai.vercel.app` |
| Public service repository | `Lord1Egypt/PocketClaw-Telegram-Setup` (public, MIT) |

The Core pair above was rebuilt for the UI integration fix and is now
**physically verified with this APK**, superseding the Milestone C pair.

### Live architecture

    PocketClaw Android
      → PocketClaw Telegram Setup  (Vercel)
        → @PocketClawSetupBot      (Telegram manager, Bot Management Mode)
          → Telegram Managed Bots
            → Upstash Redis        (pairing state, REST, 600 s TTL)
              → automatic PocketClaw Telegram configuration

The service repository is independent and public. PocketClaw's Android and Core
source stays in this repository; the APK builds from nothing in the service
repo and carries only its public base URL.

### Telegram UI integration — verified on device

- Channels → Telegram opens the managed onboarding flow: PASS
- The legacy Bot Token form is no longer the primary first-run flow: PASS
- QR code displayed: PASS
- Open Telegram action: PASS
- Suggested bot username uses the PocketClaw prefix: PASS
- Manual setup remains available: PASS
- Advanced Settings exposes the legacy Telegram configuration: PASS

### Managed-bot end to end — verified on device

Pairing created; QR/deep link generated; the Telegram managed-bot creation
screen opened; the bot was created. Created identity: display name
**PocketClaw Agent**, username pattern `@pocketclaw_<random>_bot`.

Returning to PocketClaw detected the pairing automatically, the child bot token
was delivered automatically with no BotFather copy/paste, the owner was
configured automatically, and the Telegram configuration saved itself. The
connected state and bot username rendered correctly, and Open Chat opened the
Telegram conversation. All PASS.

### Real message end to end — verified on device

    Telegram user → PocketClaw bot → PocketClaw Core
      → configured AI provider → AI response → Telegram

PASS. Real messages were sent to the newly created PocketClaw Agent bot and AI
replies were received. Multiple consecutive messages were tested and
conversation context persisted across them. The first response was slightly
slower, consistent with initial channel/provider startup; subsequent replies
were prompt.

### Live service state

Manager authentication PASS as `@PocketClawSetupBot`; `can_manage_bots` true;
Bot Management Mode enabled; Telegram webhook registered; shared pairing
storage PASS on Upstash Redis REST with a 600-second pairing TTL; Create Test
Pairing PASS; privacy page PASS.

### Security — confirmed

Embedded in the APK: manager bot token NO, Telegram webhook secret NO, pairing
secret NO, Redis credentials NO. Child bot token included in the QR: NO. Raw
token shown in normal UI: NO. Raw token required from the user: NO. Manager
token committed to Git: NO. Production secrets committed to Git: NO.

No secret value is recorded in this repository, and none ever should be.

### Core regression observed during physical testing

PocketClaw startup and Flutter first frame PASS; Gateway/Core lifecycle PASS;
Telegram channel lifecycle PASS; PocketClaw branding PASS; no black screen;
no abnormal runtime slowdown; user-facing log path privacy intact.

Preserved automated regression coverage, not re-tested physically in this
round: Android DNS/model discovery, Provider Catalog, Gemini and other
providers, OpenCode Zen, OpenCode Go, Skill Hub, Workspace, MQTT.

## Milestone D UI Integration Fix — the change that made it reachable

### Root cause

PocketClaw shows Telegram on two different surfaces, and Milestone D wired the
managed-bot flow to the wrong one.

The app's four tabs are Dashboard, the embedded Core console in a WebView,
Logs, and native Settings (`lib/main.dart:288-299`). Milestone D added the
onboarding entry to the **native Settings** page
(`lib/src/ui/config_page.dart`, `_buildTelegramEntry`). The **Channels** list —
where a user naturally goes to add a channel — lives inside the Core web
console, is served from `core/src/web/frontend`, and knew nothing about
onboarding.

So the flow existed, was fully tested, and was unreachable from the path users
take. Every Milestone D test passed because every one of them entered through
the Flutter widget directly; none entered through Channels.

### Actual Telegram entry component

    WebView tab (lib/main.dart:290)
      → Core web console
        → routes/channels/$name.tsx        (TanStack route /channels/$name)
          → components/channels/channel-config-page.tsx
            → channel-forms/telegram-form.tsx      ← what the device showed

`telegram-form.tsx` renders exactly the fields reported from the device: Bot
Token, API Base URL, HTTP Proxy, allow_from, Typing Indicator, Streaming
Output, Placeholder Message. "Enable channel" and "Save" come from the
`channel-config-page.tsx` wrapper around it.

### The fix — one journey, one implementation

The pairing flow stays in Dart, where it is already written and tested; the
console renders the entry point and asks the host to run it. Nothing was
reimplemented in TypeScript.

- `lib/src/ui/telegram_onboarding_launcher.dart` (new) — the single way into
  onboarding. Both the native settings list and the console now call it, so
  there is one implementation rather than one per surface. It also remembers
  the paired bot's public `@username` in SharedPreferences, because Core does
  not report the handle back and the connected summary needs it for Open Chat.
- `lib/src/ui/webview/pocketclaw_host_bridge.dart` (new) — the host contract.
  Builds the injected `window.__pocketclawHost` script and parses messages
  coming back. It carries no secret: only whether an endpoint was compiled in,
  and the bot's public handle.
- `lib/src/ui/webview/webview_android.dart` — registers the `PocketClawHost`
  JavaScript channel, injects the contract on page load, runs the native flow
  on request, then re-injects and fires `pocketclaw:telegram-updated` so the
  console re-renders as connected instead of showing a stale state.
- `core/src/web/frontend/src/components/channels/channel-forms/telegram-panel.tsx`
  (new) — the surface the Channels route now renders.
- `core/src/web/frontend/src/lib/pocketclaw-host.ts` (new) — typed access to
  the host, returning null in an ordinary browser.
- `channel-config-page.tsx` — the `telegram` branch renders `TelegramPanel`
  instead of the bare `TelegramForm`.

### The three surfaces

| Condition | Surface |
| --- | --- |
| No token, host can pair | **Managed onboarding first** — Connect PocketClaw to Telegram, Open Telegram, status, then "Having trouble? / Advanced / Manual setup" |
| No token, no host or no endpoint | **Manual setup**, shown in full with a short explanation and nothing to expand |
| Token already set | **Connected** — bot handle, owner, Open Chat, Reconnect / Create New Bot, then Advanced Settings |

The legacy form is never deleted or altered. It moved behind a disclosure, and
in the manual-only case it is still the whole page. Every field the manual path
depends on is asserted present by test.

### Security review of the new bridge

- The injected script carries no token or secret, and the bot handle is emitted
  through `jsonEncode` so it cannot break out of its string literal.
- `openExternal` accepts only absolute `http`/`https` URLs. `javascript:`,
  `file:`, `intent:`, `content:` and relative paths are dropped, so the bridge
  cannot be turned into an arbitrary-launch primitive.
- Both injection and message handling are scoped to the console's own origin
  (`PocketClawHostBridge.isSameOrigin`). The WebView will follow an outbound
  link if a user taps one, and a third-party page holding the host object could
  otherwise drive the app. Unparseable input fails closed.

### Tests

The point of these is that they fail against the old wiring. Reverting
`channel-config-page.tsx` to render `TelegramForm` was tried, and all 8 web
tests failed; restoring the panel made them pass. They enter through
`ChannelConfigPage channelName="telegram"` — the same component the APK renders
— not through an isolated onboarding widget.

- `channel-config-page.telegram.test.tsx` — 8 tests: case 1 managed onboarding
  primary and the handoff firing, case 2 fallback for both "no endpoint" and
  "no host at all", case 3 connected summary with Open Chat, case 4 the legacy
  form revealed from both Advanced entries, plus a host that injects late.
- `test/unit/pocketclaw_host_bridge_test.dart` — 11 tests over the script
  payload, origin scoping, scheme rejection, and malformed input.
- Frontend suite 36/36, Flutter 88/88, `flutter analyze` clean, `tsc -b` clean,
  `pnpm lint` clean.

`jsdom` and `@testing-library/react` are new frontend dev dependencies. They
had to be added: the whole failure was that no test rendered the real route,
and there was no DOM renderer in the project to do it with.

### The APK

- Path: `build/app/outputs/flutter-apk/app-release.apk`
- Size: 34,220,929 bytes (~34 MB arm64 band)
- SHA-256: `b6fea5d8ec5c3c66ba8a1320b0a217afcca322e75b5b26cc4082bbbb08a57f94`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (3)
- Release guard PASS, all three libraries present.
- **Core WAS rebuilt** — `core/src/web/frontend` changed, so the console binary
  had to be regenerated. New pair: `libpicoclaw.so` 37,224,801
  `33f8b4ef...3470e98a`; `libpicoclaw-web.so` 24,641,889 `5400cb02...2dbcb3bd`.
  They replace the device-verified `cb9b2cde...`/`b6b356f7...`, so the Core half
  of this APK is no longer device-proven and the regression sweep matters.
  `-trimpath` verified: zero developer paths in either binary.
  `core/pocketclaw-core-v0.3.1.patch` regenerated (58 files).
- Endpoint present once in `libapp.so`; the new console strings present in
  `libpicoclaw-web.so` and absent from the previous build's copy.
- Secret scan over all 425,247 printable strings: 0 Telegram tokens, 0
  `TELEGRAM_MANAGER_BOT_TOKEN`/`TELEGRAM_WEBHOOK_SECRET`/`PAIRING_SECRET`, 0
  `KV_REST_API_*`/`UPSTASH_*`/`REDIS_URL`, 0 `upstash`, 0 `redis://`. All 23
  64-hex strings in `libapp.so` are google_fonts asset checksums.

### What to check on the device

1. Channels → Telegram opens **Connect PocketClaw to Telegram**, not Bot Token.
2. Open Telegram launches the native pairing screen with QR and live status.
3. Completing pairing returns to a **Connected** summary without a manual
   reload, showing the new `@pocketclaw_..._bot` handle.
4. Open Chat opens Telegram outside the app.
5. Advanced / Manual setup reveals the full legacy form, and saving from it
   still works.
6. The native Settings → Telegram entry still reaches the same flow.
7. Regression: startup, DNS, provider catalog, OpenCode, Skill Hub, workspace,
   MQTT, Core lifecycle, branding — the Core binaries are new.

## Milestone D Live-Endpoint APK — DEVICE-FAILED (UI never reachable)

This is the artifact to install for the end-to-end Telegram test. It is the
first PocketClaw build that carries a real onboarding endpoint.

- Status: FAILED on a physical device 2026-08-26. Everything asserted about
  this build was true — endpoint compiled in, guard passed, no secrets — and it
  still did not work, because none of those checks covered whether a user could
  reach the flow. Superseded by `b6fea5d8...08a57f94`.
- Path: `build/app/outputs/flutter-apk/app-release.apk` (also written to
  `build/app/outputs/apk/release/app-release.apk`; both ignored, not committed)
- Built: 2026-08-26 with the canonical command plus the supported dart-define
  mechanism, which the Flutter Gradle plugin forwards to `flutter assemble` as
  `--DartDefines`:

      cd android && ./gradlew :app:assembleRelease \
        -Ptarget-platform=android-arm64 \
        -Pdart-defines=$(printf '%s' \
          'POCKETCLAW_ONBOARDING_BASE_URL=https://pocketclaw-telegram-setup-bot-83ai.vercel.app' \
          | base64 -w0)

  with `JAVA_HOME=/home/lordegypt/PocketCLaw/.tooling/jdk-17` and
  `GRADLE_USER_HOME=/home/lordegypt/PocketClaw-App/.tooling/gradle-stage-a-clean`.
  `-Pdart-defines` takes a comma-separated list of base64-encoded `KEY=VALUE`
  pairs. It is the Gradle-path equivalent of `--dart-define`, so the endpoint
  does not require leaving the canonical release command.
- Size: 34,211,833 bytes — in the ~34 MB arm64 band, not the ~50 MB universal
  band, so `-Ptarget-platform=android-arm64` was honoured.
- SHA-256: `9a0f74070f0129b2180b4b3237fbfacaf001ee6c8808a26e128d7ae06bb1be7f`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (version code `3`),
  minSdk 24, targetSdk 36.
- Release guard PASS. The build printed "Verified arm64-v8a native payload in
  app-release.apk: lib/arm64-v8a/libdartjni.so, lib/arm64-v8a/libpicoclaw.so,
  lib/arm64-v8a/libpicoclaw-web.so", and `unzip -l` confirms all three are
  packaged (131,248 / 37,224,801 / 24,641,889 bytes).
- Both Core binaries are byte-identical to the device-verified pair
  (`libpicoclaw.so` 37,224,801 `cb9b2cde...fb895818`; `libpicoclaw-web.so`
  24,641,889 `b6b356f7...656db9ba5`). No Core source changed, so the Core half
  is already device-proven — and, decisively for the secret scan, those
  binaries were compiled before the service was deployed and therefore cannot
  contain any of its secrets.
- Endpoint present: `https://pocketclaw-telegram-setup-bot-83ai.vercel.app`
  appears exactly once, in `lib/arm64-v8a/libapp.so`, and in no other file in
  the APK. The previous candidate had zero occurrences, so the string is there
  because of the define and nothing else.
- No Milestone D feature behavior was changed to produce this build. The only
  difference from `7c34ab12...c4911178e` is that the define is now set.

### Secret scan — explicit result

Performed over the printable strings of every file in the APK (425,035 lines).

| Checked for | Result |
| --- | --- |
| Telegram bot token pattern `<digits>:<35 chars>` (manager or child) | **0 matches** |
| `TELEGRAM_MANAGER_BOT_TOKEN` | **0 matches** |
| `TELEGRAM_WEBHOOK_SECRET` | **0 matches** |
| `PAIRING_SECRET` | **0 matches** |
| `KV_REST_API_URL` / `KV_REST_API_TOKEN` | **0 matches** |
| `UPSTASH_REDIS_REST_URL` / `UPSTASH_REDIS_REST_TOKEN` / `REDIS_URL` | **0 matches** |
| `upstash` (any case) | **0 matches** |
| `redis://` or `rediss://` | **0 matches** |

The 64-hex scan — the shape of `openssl rand -hex 32`, used for both the
webhook secret and the pairing secret — returns 1,432 hits, and every one is
accounted for:

- 23 unique in `libapp.so`, all of them `google_fonts` 8.2.1 font-asset SHA-256
  checksums, each traced back to that package's `google_fonts_parts/*.dart`.
  Nothing in `libapp.so` is unexplained.
- 209 in `libpicoclaw.so` and 201 in `libpicoclaw-web.so`, inside binaries
  byte-identical to the pre-deployment device-verified pair.
- 0 in `libflutter.so`; the remaining hits are repeats across those files.

Every HTTPS host reachable from the Dart layer was enumerated as well:
`api.flutter.dev`, `docs.flutter.dev`, `fonts.gstatic.com`, `github.com`,
`pub.dev`, `t.me`, and the onboarding base URL. No credential-bearing host is
present.

### Regression validation run before this build

- `flutter analyze` — clean, no issues (Flutter 3.47.1, Dart 3.13.1).
- `flutter test` — 77/77 passed, including the Telegram onboarding config,
  stage, lifecycle, configuration-merge, and manual-fallback tests.
- Core was deliberately not rebuilt or revalidated: no Core source changed and
  the committed binaries hash-match the device-verified pair.

### Device checklist for this APK

Unchanged from the checklist recorded under the superseded candidate below.
Run it, plus the standing regression sweep, and report back before any merge.

## Milestone D Telegram Onboarding APK — SUPERSEDED CANDIDATE (no endpoint)

- Status: SUPERSEDED by `9a0f7407...6bb1be7f` above. Retained as the record of
  the build that proved the app ships no endpoint when the define is unset.
  It was built before the service existed and has no endpoint compiled in, so
  it cannot run the flow. Do not install it for the device test.
- Path: `build/app/outputs/apk/release/app-release.apk` (ignored; not committed)
- Built: 2026-08-26 with the canonical command
  `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64`,
  `JAVA_HOME=/home/lordegypt/PocketCLaw/.tooling/jdk-17`,
  `GRADLE_USER_HOME=/home/lordegypt/PocketClaw-App/.tooling/gradle-stage-a-clean`.
- Size: 34,211,793 bytes
- SHA-256: `7c34ab12b544e585981c46632a5246a3a3fe66da24a84fce2c0b831c4911178e`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (version code `3`)
- Release guard PASS for all three required libraries.
- Both Core binaries are byte-identical to the device-verified pair
  (`libpicoclaw.so` 37,224,801 `cb9b2cde...fb895818`; `libpicoclaw-web.so`
  24,641,889 `b6b356f7...656db9ba5`). Milestone D changed no Core source, so
  the Core half of this APK is already device-proven. `libapp.so` grew from
  6,751,112 to 6,947,720 bytes, which is the new Dart code.
- No secret is embedded: 0 occurrences of a manager token pattern, and 0
  occurrences of a baked-in onboarding endpoint. The endpoint is a build-time
  `--dart-define` that is unset in this build.

### Why this cannot be device-tested yet

The flow depends on a Telegram bot that PocketClaw owns and that has Bot
Management Mode enabled. That bot does not exist, so there is nothing to point
the app at. Inventing a token or falling back to another project's setup
service was not an option, so the app in this build reports that automatic
setup is unavailable and offers manual token entry.

What is verifiable today, and was verified:

- 62 service tests across six Go packages, all against a fake Telegram.
- 50 new Flutter tests covering every stage, the lifecycle handling, the
  configuration merge, and the manual fallback. 77 Flutter tests in total.
- Core regression: 92 packages ok, `pnpm lint` clean.
- `flutter analyze` clean, release APK built, build guard passed.

### What to test on the device, once the operator setup is done

1. Settings shows a Telegram entry; opening it offers Connect Telegram, not a
   token field.
2. Connect issues a pairing; Open Telegram lands on Telegram's creation screen
   with the name and username already filled in.
3. Confirming in Telegram and returning shows Bot created, then Connected, with
   the new `@pocketclaw_..._bot` username.
4. The QR path works from a second device.
5. Backgrounding PocketClaw mid-flow and returning resumes the same pairing.
6. The bot answers its owner in Telegram and its `/start` reply is
   PocketClaw-branded, with no PicoClaw, Hermes, or Sipeed wording.
7. `allow_from` contains the creating user's Telegram ID, and the bot ignores
   other users.
8. Letting a pairing expire shows the expiry with a working retry.
9. Manual setup still works from the same screen.
10. Regression: startup, DNS, provider catalog, OpenCode, Skill Hub, workspace,
    MQTT, Core lifecycle, branding.

## Telegram Manager Bot — OPERATOR STATE (2026-08-26)

Recorded because these facts are external to this repository and cannot be
derived from it.

| | |
| --- | --- |
| Official Telegram manager | `@PocketClawSetupBot` |
| Display name | PocketClaw Setup |
| Manager created | **YES**, by the project owner |
| Bot Management Mode enabled | **YES**, manually in the BotFather mini app |
| Managed-bot deep link manually tested | **YES** — Telegram opened the managed-bot creation flow against `@PocketClawSetupBot` |
| Live `getMe` → `can_manage_bots` verification | **PASS (2026-08-26)** — the deployed service reports `Connected as @PocketClawSetupBot`, `can_manage_bots = true` |
| Public service repository | `Lord1Egypt/PocketClaw-Telegram-Setup` (public, MIT) |
| Deployment clone | `Lord1Egypt/pocketclaw-telegram-setup-bot` (private), kept in sync with upstream |
| Service deployment | **LIVE** at `https://pocketclaw-telegram-setup-bot-83ai.vercel.app` |
| Telegram webhook | **REGISTERED** at `…/telegram/webhook` |
| Pairing storage | **CONNECTED** — Upstash Redis over REST, via the Vercel Storage integration |
| Test pairing | **PASS** — created and read back through the live service |

The manager bot token is **not** recorded here, in any other document, in the
repository, or in the APK. It is entered directly into Vercel's environment
variables.

**The previously issued token is considered exposed** — it appeared in a
screenshot — and must be revoked with `/revoke` in BotFather. The production
deployment uses the regenerated token and only that.

The live `can_manage_bots` check is now **done**. It was the last link that
neither the BotFather UI nor a manually-opened deep link could establish, since
only the running service makes that API assertion. The setup page's **Verify
Telegram** button performed it against the production token and returned
`can_manage_bots = true`.

Every server-side prerequisite for Milestone D is therefore satisfied. What
remains is entirely on the app side: rebuild the APK with
`--dart-define=POCKETCLAW_ONBOARDING_BASE_URL=https://pocketclaw-telegram-setup-bot-83ai.vercel.app`
and run the device checklist.

## CODEX SOL HANDOFF — PRE-RELEASE FIX (2026-08-26)

### Status and rollback

- Candidate status: **AUTOMATED PASS; PHYSICAL DEVICE PENDING; RELEASE
  BLOCKED**. Do not merge or create `v0.2.0-rc1` until the device checklist
  passes.
- Work branch: `fix/user-facing-log-privacy`.
- Exact starting commit: `e5b88ff1a4c8f76e321c07af97eff5ca23d59d78`.
- Pushed rollback branch: `checkpoint/pre-codex-sol-prerelease-fix`.
- Pushed annotated rollback tag:
  `pre-codex-sol-prerelease-fix-20260826`.
- Both rollback refs resolve to the starting commit. Historical
  `phase2-milestone-d` remains at
  `8f861bca1c82b43b306e95b14e277269260bbab0`; it was not moved.

### Physical findings that triggered this candidate

- **PRE-RELEASE BLOCKER — PHYSICAL DEVICE REPRODUCED:** "Telegram may send
  Thinking placeholder for message A and stall the final response indefinitely
  until message B arrives; message B then triggers/delivers the response
  belonging to A."
- A broader reproduction returned "The model returned an empty response",
  then accepted several later Telegram updates and emitted several
  `Thinking... 💭` placeholders without final responses before recovering.
- Native Settings disagreed with the correctly connected Core console. A later
  installation showed only the imported `gh` skill. Android Logs showed the
  upstream-branded PID message and ANSI/control/block glyphs.
- The basename-caller and queue/drain fixes at the starting commit had already
  passed physically and were preserved.

### Root causes and fixes

- Native Telegram state was a fragile second interpretation of Core's split
  config. Native Settings is now the neutral `Telegram / Manage Telegram
  connection` shortcut to the authoritative Core route `/channels/telegram`.
  Tapping it has no pairing callback; pairing begins only from an explicit Core
  page action.
- Core secure fields are split between `config.json` and `.security.yml`; raw
  native JSON writes are invalid because secure values are redacted and the
  security file wins. Managed/manual setup now writes through Core's
  loopback-only, per-process-authenticated, write-only
  `PUT /api/pocketclaw/android/telegram` bridge. It returns no credential.
  Failed/cancelled replacement pairing performs no write, so the old bot stays
  active until a new pairing succeeds.
- Telego's default `fasthttp` path had no whole-request deadline when the
  gateway context had none. A lost mobile connection could block one outbound
  operation indefinitely while other update handlers accepted messages and
  sent placeholders. Telegram now always uses a proxy-preserving
  `net/http.Client` with a 45-second deadline.
- `TelegramChannel.EditMessage` swallowed post-connect errors and Manager
  ignored exhausted final delivery. Edit errors now propagate, normal send is
  the fallback, final delivery is synchronous, and failures propagate. Failed
  placeholder edits remain correlated: successful fallback send deletes the
  stale placeholder (or edits if deletion fails); exhausted normal send makes
  one final correlated edit attempt.
- Placeholder, typing, and reaction state was keyed only by channel/chat, so
  close arrivals could overwrite one another. Every accepted update now gets a
  safe random process-local lifecycle ID. Same-session Telegram messages are
  independent FIFO inbound requests instead of steering. Trace fields contain
  only event, correlation ID, channel, durations/counts, and tool name — no
  content, user/chat ID, token, arguments, session key, or credential.
- No causal link was found between the exactly-once Android log queue and the
  Telegram stall. The queue/drain fix remains intact.
- Tool audit found Android `exec` already has a 60-second default timeout,
  kills process trees, collects output without a pipe-drain deadlock, and
  returns `(no output)` for empty output. Skill HTTP clients are bounded.
  Success/failure/timeout/empty-output tests pass; there is no evidence `gh`
  caused the stall. Safe lifecycle events will locate any future device stall.
- Android Core is a sticky foreground service with a partial wake lock;
  Flutter pause/resume does not stop it. No source evidence ties the incident
  to backgrounding, but foreground/background/screen-locked behavior still
  requires physical validation. No battery hack was added.
- Both Milestone D and current source bundle eight templates:
  `agent-browser`, `github`, `hardware`, `picoclaw-agent`, `skill-creator`,
  `summarize`, `tmux`, `weather`. `picoclaw-agent` is deliberately unseeded, so
  the intended fresh baseline is **seven**. Import writes only the new skill
  directory and cannot replace others. The exact device reason for "only gh"
  is unknowable without its filesystem, but a real repair gap existed: existing
  config skipped seed paths. `onboard ensure-workspace` now fills only missing
  embedded files at Core-console startup, never overwrites user files, never
  newly seeds `picoclaw-agent`, and preserves an existing user copy.
- Logs now store one sanitized plain-text representation before display and
  export. CSI/SGR (RGB/256 included), OSC, DCS/SOS/PM/APC, cursor/erase, CR,
  backspace, C0/C1, DEL, and orphaned CSI fragments are removed while Arabic,
  emoji, ordinary Unicode, tabs, and newlines survive. Android sets
  `NO_COLOR=1`, `TERM=dumb`, and prints plain `PocketClaw`. Stale PID text is
  `pid belongs to another process; ignoring stale pid file`. No global string
  replacement was used.

### Changed source areas

- Android host: `PicoClawMethodChannel.kt`, `PicoClawService.kt`.
- Flutter: `lib/main.dart`; Core channel/service/log sanitizer; Telegram config
  writer; config/onboarding/settings widgets. The obsolete native status reader
  and connected page were deleted.
- Core lifecycle: `pkg/bus/types.go`, agent lifecycle/mailbox/pipeline files,
  channel base/interfaces/manager, and Telegram transport.
- Core config/skills/log/UI: onboard helpers/command, Android bridge API, web
  startup/onboarding/banner/gateway, Telegram route test, DingTalk and Teams
  titles.
- Tests changed across Flutter, agent lifecycle, Android bridge, skills,
  onboarding, frontend route, logger, and log sanitization.
- `core/pocketclaw-core-v0.3.1.patch` was regenerated (93 changed files) and
  intentional divergence recorded in `UPSTREAM_TRACKING.md`.

### Validation

- `flutter analyze`: PASS, no issues. `flutter test`: PASS, 94 tests.
- Frontend: Vitest PASS (2 files / 36 tests), `pnpm exec tsc -b` PASS,
  `pnpm lint` PASS.
- Go PASS: logger, PID, providers, all web backend packages, agent, skills,
  androiddns, MQTT, onboard, commands, channels, Telegram, and tools. The final
  agent/channel/Telegram run included long timeout cases.
- Deterministic lifecycle tests prove: empty provider A finalizes explicitly
  while idle; close B/C arrivals finalize independently in FIFO order; failed
  edit sends normally and cleans the placeholder; exhausted send can retry the
  placeholder as final delivery.
- Canonical Core build used repository-local `core/src/`, root Make targets,
  `-trimpath`, `-s`, and `-w`; both binaries contain zero developer paths.

### Candidate artifact

- APK: `build/app/outputs/apk/release/app-release.apk`
- Size: `34,239,649` bytes
- SHA-256:
  `f663d25a2fffb0ce969ad4a9ce3405c1e563b6263c7af37e90768eef471c621d`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (3), minSdk 24,
  target/compile SDK 36.
- Onboarding URL occurs once:
  `https://pocketclaw-telegram-setup-bot-83ai.vercel.app`.
- Build guard PASS for the three required arm64 libraries.
- `libpicoclaw.so`: 37,224,801 bytes,
  `87653601023974156be1b1a255387ec3c93a0c154254a7b8d9e1012ac2e926e5`.
- `libpicoclaw-web.so`: 24,641,889 bytes,
  `a8328f1d66932ba8da544060905965278898744c04c2ece5a993717e361400c5`.
- Secret scan PASS: zero Telegram-token shapes, secret environment names,
  Redis URLs, private-key blocks, or full provider-key shapes. Two hits are
  only seven-character provider-format prefixes compiled into Core, not keys.
  Zero `.upstream` paths and zero raw full gateway caller strings.
- Known deferred artifact issue remains once in `libapp.so`:
  `file:///home/lordegypt/PocketClaw-App/.dart_tool/flutter_build/dart_plugin_registrant.dart`.
  It is not normal UI/log output. Keep under **FINAL RELEASE HARDENING**; do
  not rush obfuscation changes into this candidate.

### Deferred roadmap — record only

- Background & Battery page: optimization state, user-initiated settings,
  Samsung Sleeping/Deep Sleeping guidance, reliability information. PocketClaw
  cannot silently grant Unrestricted mode.
- Runtime / Statistics tab: local-by-default uptime, agent state, token counts,
  CPU/RAM/Core resources, request outcomes/latency, provider/model/channel
  state, restart count, and later charts.
- Manager-token rotation remains separate security hygiene. A token appeared
  in screenshots/chat; do not mark rotation complete without explicit external
  confirmation, and never retrieve or print it.

### Required next action

Install only the APK above. Run the full physical checklist, especially send
`تسلم` and send nothing else until its final reply arrives; then idle and send
another message repeatedly. Check foreground/background/locked screen,
placeholder/typing/streaming combinations, native Telegram navigation,
reconnect preservation, fresh/existing skills plus `gh`, clean exactly-once
Unicode logs/export, provider, Skill Hub, startup, and performance. Do not merge
or release on anything short of physical PASS.

## DEBUG log cleanup micro-pass — AUTOMATED PASS, PHYSICAL PENDING

Physical export from candidate `f663d25a...71c621d` contained valid Go timing
text such as `53.616µs` as `53.616�s`, and 185 routine successful requests to
the two UI polling endpoints. The prior basename caller, branding cleanup,
terminal sanitizer, and exactly-once queue/drain fixes had physically passed
and are unchanged.

The UTF-8 corruption was not in Go, Kotlin ingestion, the Dart sanitizer, the
in-memory queue, or the Logs UI. Android Export Logs used
`Uint8List.fromList(content.codeUnits)`, truncating Dart UTF-16 code units to
single bytes before the Kotlin MediaStore writer. `µ` became lone byte `B5`,
which is invalid UTF-8; Arabic and emoji were vulnerable for the same reason.
The Android export boundary now uses `utf8.encode`, while the rune-safe
terminal sanitizer remains unchanged. The real MethodChannel export transport
is tested by strict UTF-8 decode after sanitizing ANSI-wrapped Arabic, `µ`,
emoji, mixed-language text, and Unicode punctuation; it introduces no U+FFFD.

The polling noise was legitimate middleware output, not duplicate delivery:
DEBUG HTTP logging recorded each successful UI request to its own log and
status endpoints. The HTTP middleware now suppresses only expected `GET`
requests to exact paths `/api/gateway/logs` and `/api/gateway/status` with a
2xx response. Failures, redirects, unexpected methods, unknown routes, and all
other API requests retain normal DEBUG logging. No deduplication or endpoint
behavior changed.

Validation: `flutter analyze` clean; 97 Flutter tests; frontend 36 Vitest tests,
`tsc -b`, and lint; Go `pkg/logger`, `pkg/gateway`, `web/backend/api`, and
`web/backend/middleware` pass with the shipped `goolm,stdjson` tags. Core
provenance patch regenerated at 95 files. Canonical Core build passed
`-trimpath` with zero developer paths; the APK guard passed all three required
arm64 libraries. Artifact scan found zero Telegram token shapes, service secret
names, Redis/Upstash values, or raw full Go callers; the single known generated
Dart source URI remains deferred to final hardening.

Replacement candidate:

- APK: `build/app/outputs/apk/release/app-release.apk` and identical
  `build/app/outputs/flutter-apk/app-release.apk`
- Size: 34,239,073 bytes
- SHA-256: `eacbbc86b99429f114aba9ba1dca57224122fa176f6b4d99edf350454423f9a8`
- Package/version: `com.lord1egypt.pocketclaw` 0.1.3 (3), minSdk 24,
  target/compile SDK 36
- `libpicoclaw.so`: 37,224,801 bytes,
  `5c09eb72a1faa6dd6f8a0e6b66bcbc028f070d3eff9b8c5b5f87716c04d763bc`
- `libpicoclaw-web.so`: 24,641,889 bytes,
  `cb6b10cc2a951d2307959e9effbdbd052762fe827b9943c22a9cdcdd54b03b52`
- Onboarding endpoint occurs once in `libapp.so`; version unchanged.
- Status: **AUTOMATED PASS; PHYSICAL DEVICE PENDING; RELEASE BLOCKED**.

Install only this replacement. Export a DEBUG log after several minutes and
verify valid `µ`, Arabic, emoji, punctuation, zero newly introduced U+FFFD,
continued ANSI/control removal, and no routine successful polling noise. Failed
polls and real API requests must remain visible. Continue the broader physical
pre-release checklist before any merge or release.

## Web Console log parity / internal identifier visibility — AUTOMATED PASS, PHYSICAL PENDING

The native/export cleanup remains intact. The remaining Web Console startup
boxes came from a separate path: the captured Core gateway CLI treated
`--no-color` as “remove RGB” but still emitted a six-line Unicode block-art
banner, while the React Logs page deliberately parsed SGR into styled spans and
did not handle OSC, cursor/erase, carriage return, backspace, or other terminal
protocols. The gateway log ring also stored raw child output, so the Web API
had no user-visible normalization boundary.

The earliest safe Web boundary is now `LogBuffer.Append`: it stores only the
shared brand-safe plain-text representation. The native Dart boundary and the
browser's idempotent legacy/raw-line guard are locked to the same canonical
JSON fixtures. Valid Unicode, Arabic, emoji, punctuation, `µs`, tabs, and
newlines survive; terminal protocols are removed. Literal box drawing is not
globally stripped because it can be legitimate Unicode. Instead, the captured
gateway is always started with `--no-color`, whose banner is now the single
plain line `PocketClaw`.

The launcher startup event is now `Starting gateway process` and never prints
the `libpicoclaw.so` path. Exact successful `GET /pico/ws` connection/close
events (recorder 200/2xx or upgrade 101) are suppressed at HTTP middleware.
Failures and unexpected methods remain visible as `/internal realtime
connection`, so the compatibility route itself is not exposed. The real
`/pico/ws` endpoint, library filenames, module paths, environment variables,
and other compatibility identifiers were not renamed.

Regression results: `flutter analyze` clean; 99 Flutter tests; frontend 3 files
/ 37 tests, `tsc -b`, and lint; tagged Go `pkg/logger`, `pkg/gateway`,
`web/backend/api`, `web/backend/middleware`, and `cmd/picoclaw` all pass. The
real React Logs page test consumes the shared contract and proves Arabic,
emoji, and `53.616µs` render with no ANSI/control fragments, routine
`/pico/ws`, internal library path, or unintended PicoClaw/Sipeed branding,
while a 500 remains visible under neutral wording. Core provenance is now 108
files; its generator was made deletion-safe after the intentionally removed
ANSI renderer exposed a stale tracked-file assumption.

Replacement candidate:

- APK: `build/app/outputs/apk/release/app-release.apk` and identical
  `build/app/outputs/flutter-apk/app-release.apk`
- Size: 34,241,381 bytes
- SHA-256: `3e138b4a53a0389b76dbef045649d906fe2db785cd2af606826f7a0f87170adc`
- Package/version: `com.lord1egypt.pocketclaw` 0.1.3 (3), minSdk 24,
  target/compile SDK 36
- `libpicoclaw.so`: 37,224,801 bytes,
  `c9c348e9c637a7552e810396bfba5460ad4a3f65ab506d933ac5d05ea68d236e`
- `libpicoclaw-web.so`: 24,641,889 bytes,
  `c891ca033ededb6b8941d997fbc8e0a8d69eb18f44a6ed2176f878f65d34840a`
- Both Core libraries contain zero developer paths; packaged hashes match.
- Live onboarding endpoint occurs once in `libapp.so`; metadata unchanged.
- Build guard: PASS for `libdartjni.so`, `libpicoclaw.so`, and
  `libpicoclaw-web.so`.
- Status: **AUTOMATED PASS; PHYSICAL DEVICE PENDING; RELEASE BLOCKED**.

Install only this replacement. In Native Logs, Export Logs, and Core Web
Console Logs, verify zero normal visible `picoclaw`/`PicoClaw`/`sipeed`/`Sipeed`,
no terminal boxes or ANSI, intact Arabic/emoji/`µs`, no successful
`/pico/ws` noise, and visible neutral wording for genuine failures. Do not
merge or release before the complete standing physical checklist passes.

## Final enabled-channel brand micro-fix — AUTOMATED PASS, PHYSICAL PENDING

Physical validation of commit `3611ca1` substantially passed: Web Console
terminal/control boxes are gone, the PocketClaw banner and UTF-8/`µs` are
correct, the internal library path is absent, and all eight skills are
available with 17 tools loaded. One raw startup summary remained:
`✓ Channels enabled: [telegram pico]`.

`pico` is the exact internal singleton channel ID for the Core Web Console
transport. It owns the authenticated `/pico/ws` connection, browser chat
sessions, streaming/tool-feedback protocol, and `/pico/media` delivery. Its
config key, factory registration, token handling, channel identity, routes,
session IDs, and compatibility behavior remain unchanged.

Only the two user-facing startup/reload summary print sites now map exact
`config.ChannelPico` to display label `pocketclaw`. The internal slice is copied
and left untouched; there is no substring or global replacement. Targeted
coverage proves `[telegram pico]` renders as `[telegram pocketclaw]`, while the
original internal value remains `pico` and unrelated names are unchanged.

Tagged Go `pkg/gateway`, `pkg/logger`, `web/backend/api`,
`web/backend/middleware`, and `cmd/picoclaw` tests pass. Core provenance is 110
files. Canonical Core build has zero developer paths, and the APK guard passed.

Replacement candidate:

- APK size: 34,241,857 bytes
- APK SHA-256: `aab3c565bd6bec2eb714443756e16be5d2ebe8d4496d94e64a6b8ecac25b3582`
- `libpicoclaw.so`: 37,224,801 bytes,
  `10446d8156b0a33a920f31c568b25c9ae59f96ea5f8db576f9fe40dce71b45f1`
- `libpicoclaw-web.so`: 24,641,889 bytes,
  `683463dfee287b7cd88f592b3ec534e2f8458e3bace77ca9bfee4dd7ce5f5ce4`
- Packaged Core hashes match; live onboarding endpoint occurs once.
- Status: **AUTOMATED PASS; FINAL LABEL PHYSICAL CHECK PENDING; RELEASE BLOCKED**.

Next physical check: restart Core and confirm the Web Console line is exactly
`✓ Channels enabled: [telegram pocketclaw]` (order may follow configured
channel order), with every already-passed log and skill behavior preserved.

## Final user-visible caller brand fix — AUTOMATED PASS, PHYSICAL PENDING

Physical testing found the remaining structured WebSocket logger header as
`INF pico pico.go:1013 > ...` / `pico.go:1071`. The internal Go package,
source filename, `ChannelPico`, routes, media protocol, config keys, and channel
identity remain unchanged.

The shared user-visible normalization contract now recognizes only structured
logger-header fields. Exact component token `pico` displays as `realtime`, and
exact caller basename `pico.go` displays as `realtime.go`; the numeric line
suffix is captured unchanged. The mapping is implemented at the Go Web log
buffer boundary, Dart native/export boundary, and idempotent React Logs guard,
using the same canonical fixtures. It is not a source rename or substring
replacement: `picometer`, `picophone.go`, and `pico_client` remain unchanged.

Expected example:

`INF pico pico.go:1013 > WebSocket client connected`

becomes:

`INF realtime realtime.go:1013 > WebSocket client connected`

Validation: `flutter analyze` clean; 99 Flutter tests including actual stored
and exported representation; frontend 37 tests/tsc/lint including the real Logs
page; tagged Go API/middleware/logger/gateway/CLI suites including `LogBuffer`;
110-file Core provenance; zero developer paths; packaged hashes and live
endpoint verified; permanent APK guard passed.

Replacement candidate:

- APK size: 34,242,865 bytes
- APK SHA-256: `1eeca7c993d657f0e6e763d94691a054e584592d0989445454144ac15ad089f7`
- `libpicoclaw.so`: 37,224,801 bytes,
  `a379ae4531d1c64bf653d503f9f20fa627d497389e037346815525f6407eafb9`
- `libpicoclaw-web.so`: 24,641,889 bytes,
  `584dd9ae2e9c11e46d50f879020a07758ff612b744ae69448ac3f3cb71fa4a01`
- Status: **AUTOMATED PASS; PHYSICAL DEVICE FINAL GATE PENDING; RELEASE BLOCKED**.

Next physical check: confirm Native Logs, Export Logs, and Web Console Logs all
show `realtime realtime.go:<original line>` for these WebSocket events, with
genuine failures visible and every previously passed Unicode, control-cleanup,
exactly-once, polling, channel-label, skill, Telegram, and provider behavior
unchanged.

## Web Console Logs scroll/jitter micro-pass — AUTOMATED PASS, PHYSICAL PENDING

Physical testing of the preceding candidate confirmed the complete
user-visible log cleanup, but the Web Console Logs viewport moved between
one-second polls. Native Logs did not reproduce it.

The Web page had two layout timing defects. Following users were corrected with
a passive `useEffect`, so an appended row could paint at the old scroll offset
before the browser was snapped to the new bottom. Long lines were also hard
wrapped in JavaScript using a column count recomputed by a `ResizeObserver` on
the entire content element; every appended row changed that element's height,
causing another measurement and potentially rewriting all long-row text/layout.
Rows additionally used array-index keys rather than the event identity already
available from the incremental API.

The Logs page now records follow state from the viewport's scroll events and
applies bottom correction in `useLayoutEffect`, before paint, only when the user
was within 24 px of the bottom. A scrolled-up viewport receives no programmatic
scroll. Log IDs are `run_id:absolute_offset`; memoized rows use those IDs as
keys. JavaScript hard wrapping and the content resize observer were removed;
the unchanged sanitized string is rendered once and wraps natively with CSS.
Browser anchoring is disabled inside the explicit-policy log content.

The real Logs page DOM suite covers bottom following, scrolled-up preservation,
unchanged long Telegram-style row node/text identity with Arabic, emoji and
`53.616µs`, and repeated no-new-log rerenders. Frontend Vitest is 41/41, `tsc
-b` and lint pass. Tagged Go API/middleware tests and the Native/Export
exactly-once/sanitization regression pass. Core provenance is 113 files; both
Core binaries have zero developer paths; the permanent APK guard passed.

Replacement candidate:

- APK size: 34,239,873 bytes
- APK SHA-256: `be5d7cbb18c0378dad0a3d53d2d3a4e71001411121fa056f7a6606645070fc96`
- `libpicoclaw.so`: 37,224,801 bytes,
  `715cd790d1af3f295a50a87a64d5ac8b2fbc0454c8e2268f55896538c6143cf1`
- `libpicoclaw-web.so`: 24,641,889 bytes,
  `e3930ae24e5d7e7185a470af65caf3c8d46a97335daecc3590ab64f9f5f5da14`
- Status: **AUTOMATED PASS; WEB VIEWPORT PHYSICAL GATE PENDING; RELEASE
  BLOCKED**.

Next physical check: leave Web Console Logs untouched at the bottom through
multiple updates, then scroll upward and wait through multiple polls. Confirm
no shake, rewrap, or forced bottom jump while every already-passed log
sanitization and branding behavior remains intact.

## Telegram token log-redaction security pass — AUTOMATED PASS, PHYSICAL PENDING

Physical evidence narrowed the disclosure to Core Web Console DEBUG Logs;
Native Android Logs remained clean. Investigation confirmed **case A**: Telego
constructs a full Bot API URL, but PocketClaw's compatible third-party logger
called its masking function before `logMessage`, stdout writers, and the Web
backend `LogBuffer`. The full token did not enter persisted Web history through
this path. The old mask deliberately retained the bot-ID prefix plus the first
and last four secret characters, and those fragments did enter the Web stream
and stored ring. This is a fragment-disclosure defect, not evidence that the
complete credential was persisted or compromised; no credential was rotated
or modified.

The same pre-stdout logger boundary now removes the complete Bot API credential,
including normal/percent-encoded forms and bare credentials, without retaining
an ID, prefix, or suffix. Bearer/Basic Authorization values are also removed.
Before `LogBuffer.Append` persists a Web line, Telegram Bot API URLs—whether
raw, fully redacted, or using the historical partial mask—normalize to
`Telegram API call: <operation>`. Failures retain HTTP method, operation,
status/error, timeout, and latency text. The React normalizer provides an
idempotent legacy/raw guard. Native/Export source and queue code were not
changed.

Synthetic-only regressions cover `getMe`, `getUpdates`, `sendMessage`,
`editMessageText`, timeout/error/5xx cases, arbitrary methods and custom API
servers, percent encoding, bare tokens, Authorization credentials, public bot
metadata preservation, actual `LogBuffer` storage, and the real Logs page.
Go logger/Telegram/gateway/API/middleware/CLI suites pass; frontend Vitest is
42/42 with tsc/lint; the unchanged Native/Export log regression is 7/7. Core
provenance is 115 files; zero developer paths and the permanent APK guard pass.

Replacement candidate:

- APK size: 34,240,641 bytes
- APK SHA-256: `8257e9f091039f2c29332b8f14e2d397d36bbb735593596a4caa0e567c7050fe`
- `libpicoclaw.so`: 37,224,801 bytes,
  `0e914550208fc7a77ce9319b32554be802108207789f992098d2186dc8b555e9`
- `libpicoclaw-web.so`: 24,641,889 bytes,
  `7d7b254b04b33919b6ebbfeac9147a06a4472de1c4fe6354b6ecb6fe6d9898c7`
- Status: **AUTOMATED PASS; WEB CONSOLE PHYSICAL GATE PENDING; RELEASE
  BLOCKED**.

Next physical check: enable DEBUG, exercise Telegram polling and message/edit
calls plus a recoverable failure, and confirm Web Console Logs show operation
names and useful failure details with zero credential fragments. Reconfirm the
already-passed viewport stability and Native Logs behavior.

## Final legacy brand visibility sweep — AUTOMATED PASS, PHYSICAL PENDING

Physical validation confirmed every preceding logging, viewport, Telegram,
Unicode, and Skills fix, then identified remaining structured compatibility
details in normal Core Web Console startup logs. The shared Web pre-storage
normalizer now maps only exact `channel=pico` and `type=pico` fields to
`pocketclaw`, hides exact `/pico/` only when attached to that internal channel,
uses PocketClaw realtime wording for exact protocol lifecycle messages, and
replaces the compatibility PID path with a semantic gateway PID message.
Exact realtime-subsystem failure messages remain visible with neutral wording.

Runtime `ChannelPico`, config/serialized IDs, Go packages/files, `/pico` routes,
the actual `.picoclaw.pid`, libraries, environment variables, and provenance
were not changed. There is no blanket or substring replacement. Negative tests
prove `picometer`, `picophone.go`, `pico_client.go`, `topic=pico-test`,
`my-pico-notes.txt`, and `.picoclaw.pid.backup` remain unchanged. A
representative stored startup plus real Logs-page DOM test reports zero
unintended legacy occurrences while preserving the security warning.

Regression results: frontend Vitest 44/44, TypeScript, lint; tagged Go logger,
gateway, channel, Pico, Telegram, Skills, API, middleware, and CLI suites;
Flutter analyze and 99/99 tests. The 115-file Core provenance patch was
regenerated, both binaries contain zero developer paths, and the permanent APK
payload guard passed.

Replacement candidate:

- APK SHA-256: `309f6d7ac47206015e3c3c9a5d5f1cf1903b5fb8b2c07783d21131e67a5f3030`
- `libpicoclaw.so`: 37,224,801 bytes,
  `49f89ae22f5020425ff9346fd579705bd2a44e43aa42018985cf1702d3be656f`
- `libpicoclaw-web.so`: 24,641,889 bytes,
  `98f3fa9df6b89bb745181793da2351adc7f8086ea63b20e2507d46bfaaf08bae`
- Status: **AUTOMATED PASS; WEB CONSOLE PHYSICAL GATE PENDING; RELEASE
  BLOCKED**.

Next physical check: restart Core with DEBUG logging and confirm channel
initialization, security, webhook, realtime, and PID events use only the
semantic PocketClaw display while all previously passed behavior remains intact.

## Telegram DEBUG final cleanup — AUTOMATED PASS, PHYSICAL PENDING

Telego's raw successful response is correct: `Err: [<nil>]`. PocketClaw's
pre-stdout credential redactor also preserved it correctly. Corruption occurred
when the Web backend normalized captured stdout before `LogBuffer` storage: the
orphaned-CSI fallback accepted `<` as a parameter byte and `n` as a final byte,
removed `[<n`, and persisted the malformed remainder `il>]`. React was already
rendering that stored string as safe text, not HTML.

The orphaned fallback now recognizes only numeric/private-numeric CSI remnants;
real ESC/C1 CSI handling remains unchanged. Ordinary `<` and `>` text,
including Arabic and emoji, stays intact. Exact successful Telego nil fields
display semantically as `Err: none`; generic `<nil>` text remains unchanged.
The real Logs DOM proves `<tag>` remains text and creates no element.

At the pre-stdout Telego logger boundary, routine `getUpdates` request lines and
only exact successful empty responses are omitted. Failures have their separate
`Execution error getUpdates` line, and non-empty or unsuccessful responses stay
visible. Web pre-storage, React, and Native/Export normalizers enforce the same
idempotent policy for legacy/raw lines. Four repeated empty poll pairs add zero
stored entries. Telegram lifecycle/delivery/onboarding and native queue logic
were not changed.

Regression results: frontend Vitest 45/45, TypeScript, lint; tagged Go logger,
gateway, channels, Pico, Telegram, Skills, API, middleware, and CLI suites;
Flutter analyze and 101/101 tests. The 115-file Core patch was regenerated,
both Core binaries contain zero developer paths, and the permanent APK guard
passed.

Replacement candidate:

- APK size: 34,243,585 bytes
- APK SHA-256: `2c00720a44a2b2ac5de2c82c202c172129e778392eabf5434c36883a322da27c`
- `libpicoclaw.so`: `7c1d3918e30ff673e62a963822fcdd00e327805221cea1b963cb222d1138ef1f`
- `libpicoclaw-web.so`: `1611b6e102fbcff726213bef659cbb05509efc12e63592a373df64fe14d09256`
- Status: **AUTOMATED PASS; PHYSICAL GATE PENDING; RELEASE BLOCKED**.

Next physical check: leave Telegram polling in DEBUG for several minutes;
routine empty `getUpdates` must stay silent. Exercise a non-empty update and a
recoverable failure, verify useful details remain, and reconfirm every prior
physical pass.

## Agent DEBUG privacy, Telegram payload privacy, and default identity — AUTOMATED PASS, PHYSICAL PENDING

Physical Web DEBUG evidence exposed two source-level payload paths. First, the
freshly generated system prompt itself—not merely its log preview—still used
the legacy lowercase product identity in `ContextBuilder.getIdentity`. Fresh
PocketClaw defaults now send `# PocketClaw 🦞` and `You are PocketClaw, a
helpful AI assistant.` to the model. Existing user-authored prompt overlays and
all internal/upstream compatibility identities remain unchanged.

Second, normal Agent logs explicitly emitted a system-prompt preview, complete
message/tool JSON, raw reasoning text, complete tool-call argument previews,
and structured routing/session identifiers. The explicit payload logs and dead
raw formatters are removed. Normal logs retain model, iteration, message/tool
counts, prompt length, content/reasoning lengths, token usage, tool names,
status, duration, and failures. An exact-field pre-writer copy redacts session
keys and internal identifiers, omits raw-content fields, and does not mutate the
runtime maps or routing values.

Telego's `Response.String()` also included complete successful Telegram result
JSON (chat/user IDs, names, usernames, language and message bodies). Before
this pass that raw result entered stdout and Web `LogBuffer`; it was not a
frontend-only leak. The Telego adapter now converts request/response data to
operation/status metadata before any writer. Empty successful `getUpdates`
remains silent, non-empty updates retain count/type only, successful operations
retain operation plus `ok=true`, and failures retain safe operation/error-code
metadata. Backend, React, and Native/Export normalization provide idempotent
legacy/raw guards. Credentials remain fully redacted.

Regression results: tagged Go Agent/logger/Telegram/Pico/gateway/API/
middleware/CLI suites; frontend Vitest 46/46, TypeScript and lint; Flutter
analyze and 101/101 tests. Core provenance is 124 files, both binaries contain
zero developer paths, the live onboarding endpoint is packaged, and the
permanent APK guard passed.

Replacement candidate:

- APK size: 34,251,141 bytes
- APK SHA-256: `46ca983a380d1a1b69f71f01cf18b840b5007054d732d8430fed5d11cd2b908e`
- `libpicoclaw.so`: 37,224,801 bytes,
  `8ed15601f3312c034e21df55bcbaa980b5c1c98ff1b15caa1a404be3129be24a`
- `libpicoclaw-web.so`: 24,641,889 bytes,
  `2100100454a42b0c7517084a9b52a069ad0153476da6d9e3787582bc7f5fc1d7`
- Status: **AUTOMATED PASS; PHYSICAL GATE PENDING; RELEASE BLOCKED**.

Next physical check: generate a fresh realtime/Telegram conversation under
DEBUG and confirm only lifecycle/count/length/tool-name metadata appears; no
prompt, message, reasoning, tool arguments/schemas, session/internal IDs, or
Telegram payload values may appear. Reconfirm Skills 8/8, Tools 17+, Telegram
delivery, Web viewport stability, and all prior Unicode/branding passes.

## 2026-08-29 — v0.2.0-rc1 release candidate frozen

**PHYSICAL DEVICE: PASS.** The candidate
`5760247a17ccff68d188180875f1812c150dd7ae6de45e3f109d7ba07c2186b9`
(34,260,709 bytes) was validated by the user on a real ARM64 Android device.
That run cleared app/Core startup, Skills 8/8, Tools 17, Telegram owner-only
authorization and final delivery, the no-second-message stall fix, Web/realtime
owner authorization, password/session dashboard auth, Core Gateway remaining
loopback-only on 18790, Public Mode OFF and ON, live OFF→ON→OFF→ON rebinding
without a manual service restart, a real LAN URL of the `192.168.x.x:18800`
form, authenticated Dashboard access from a computer on the same LAN, Core
18790 staying off the LAN, QR/connect URL refresh, Telegram continuity across a
Public Mode change, Web Logs stability, UTF-8/Arabic/emoji/µs rendering, and
every privacy expectation from the preceding passes. The DEBUG log-cleanup
physical gate that previously blocked release is therefore CLEARED.

Source provenance for that artifact is proven, not assumed. The Core build
stamps `BuildTime` through `-ldflags`, so consecutive builds differ in exactly
64 bytes at identical length. Rebuilding this workspace's Core source with the
validated binaries' own timestamps pinned reproduced both of them byte for
byte:

- `libpicoclaw.so` 37,224,801 bytes,
  `4e8c23c70bd77fbdce96d04004dd13b3ba4cac8e1e03164296cc47f7ead1ffb6`
- `libpicoclaw-web.so` 24,707,425 bytes,
  `45427e0d48c53c7611625a8b621e4a4f565bfec11702d12e66c94ada2c900917`

Those exact binaries are what the repository now carries and what the RC APK
packages, so the released native payload is the physically validated payload.

Release-candidate build identity:

- Package: `com.lord1egypt.pocketclaw`
- Version: `0.2.0`, version code `4` (was `0.1.3`/`3`, the inherited FUI
  baseline; a candidate tagged `v0.2.0-rc1` must not report `0.1.3`)
- ABI: `arm64-v8a`; the permanent guard verified `libdartjni.so`,
  `libpicoclaw.so`, and `libpicoclaw-web.so` under `lib/arm64-v8a/`

Automated verification at the freeze: `flutter analyze` clean and 114/114
Flutter tests; frontend 46/46 Vitest, `tsc -b`, and lint clean; the **complete**
Go suite green under `-tags goolm,stdjson`, along with `go build ./...` and
`go vet ./...`. The `goolm` tag selects the pure-Go Olm implementation, so the
previously reported `olm/olm.h` host dependency is not required at all and is
no longer an accepted exception. Core provenance regenerated to 141 files and
reproduced the committed patch byte for byte; both binaries carry zero
developer paths; `core/verify-no-external-source.sh` passed.

One known, non-blocking condition: `go test -race ./web/backend/api` fails in
`TestStartGatewayLocked_UsesReloadedConfigForBootSignature`. The race is
between that test's own cleanup calling `cmd.Wait()` and the production monitor
goroutine's `cmd.Wait()` on the same `exec.Cmd` — a test-harness defect, not a
production data race; production calls `Wait` once. It reproduces identically on
`develop` at `8f861bc`, so it is pre-existing and not a regression from this
work. Every other test in that package passes under `-race`, as do the race
runs for `pkg/channels`, `pico`, `telegram`, `logger`, `netbind`, `config`,
`skills`, and the rest of `web/backend`. Fixing the shared Kill+Wait cleanup
pattern in `gateway_test.go` is deferred; it is out of scope for RC closure.
