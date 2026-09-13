# PocketClaw defect log

This log contains defects and deferred engineering work with reliable repository
evidence. It is intentionally not an invented inventory of every issue ever
found. The operating method has surfaced and helped resolve dozens of defects;
only reconstructable examples belong here.

## Open / deferred





### PC-DEF-012 — Broad dependency export surfaces need reachability evidence

- **Discovered:** H5A native/ELF audit, 2026-09-11.
- **Component:** Dart JNI plugin, embedded Python, and dependency-native ELF.
- **Severity:** Hardening review item; no demonstrated functional or security
  failure.
- **Description:** Arm64 `libdartjni.so` exports 313 symbols (214
  `globalEnv_*`, 42 Dart DL, seven Java/JNI, and 50 other); packaged Dart AOT
  contains 11 matching names. The Python executable exports 2,261 dynamic
  symbols. Datastore exports four required Java methods plus five C++ helpers.
  These are wider surfaces than the app-owned entry points, but FFI lookups,
  JNI name binding, statically linked modules, and upstream consumer contracts
  make blind visibility changes unsafe.
- **Evidence:** Dynamic-symbol and packaged-AOT comparison from the exact H4B
  APK; required Java/JNI symbols and Dart snapshot exports are asserted by the
  H5A audit tool.
- **Reason deferred:** Static counts do not establish that an export is safe to
  remove. Narrowing requires dependency-specific call/reachability evidence and
  runtime validation.
- **H5C disposition:** Unchanged again. H5C narrowed no export and produced no
  new reachability evidence; the packaged export counts in the production APK
  are identical to H5B's.
- **H5B disposition:** Evaluated and deliberately unchanged. H5B produced no
  call/reachability evidence for any of these surfaces, and narrowing a
  visibility surface without it is how a runtime `UnsatisfiedLinkError` ships.
  No export map was added and the packaged export counts are unchanged. The
  automated audit continues to assert the required boundary rather than hide
  the rest.
- **Target milestone:** A later dependency-focused milestone. Narrowing requires
  dependency-specific reachability plus runtime validation, or an explicit
  owner acceptance of the retained surface.
- **Status:** OPEN / FUTURE EVIDENCE REQUIRED.

### PC-DEF-002 — Web console listens on `0.0.0.0:18800`

- **Discovered:** vc59 machine validation, 2026-09-08.
- **Component:** Launcher/web console network exposure.
- **Severity:** Unrated security review item.
- **Description:** The Core gateway is loopback-only on 18790, while the web
  console was observed listening on all interfaces on port 18800.
- **Evidence:** `TASKS.md`, “Non-blocking security review”; historical
  `PROJECT_STATE.md` vc59 evidence.
- **Reason deferred:** It predated the milestone that found it and requires a
  product decision about cross-device console access.
- **Target milestone:** Secrets/configuration and exposure audit, before stable.
- **Audited 2026-09-12, and the observation is explained.** The product now has
  an explicit Public Mode whose stated purpose is LAN access to the
  password-protected dashboard. Reproducible evidence from the shipped
  `pkg/netbind`, driven with the same default-mode selection
  `openLauncherListeners` applies:

      PUBLIC OFF (no -public, no host)   bindHosts=[::1 127.0.0.1]
      PUBLIC ON  (-public, no host)      bindHosts=[:: 0.0.0.0]
      host override 127.0.0.1 + -public  bindHosts=[127.0.0.1]   (host wins)
      gateway (host=localhost)           bindHosts=[::1 127.0.0.1]  in BOTH states

  `0.0.0.0:18800` is Public Mode ON and nothing else. The Core gateway on 18790
  is loopback-only in both states and is not reachable by the launcher's public
  flag at all — `openGatewayListeners` always passes `netbind.DefaultLoopback`.
  Authentication is mandatory whenever Public Mode is on: the unauthenticated
  surface is only `POST /api/auth/{login,logout,setup}`, `GET /api/auth/status`
  and GET/HEAD of the login/setup SPA routes, `/assets/`, the favicons,
  `site.webmanifest` and `robots.txt`; there is no unauthenticated health,
  config or control endpoint. The realtime WebSocket requires a live session
  **and** an origin check, and auth-path canonicalization blocks
  `/assets/../` traversal. `/api/auth/setup` is unauthenticated only while no
  password exists and requires a session once one does. The Android
  local-auto-login grant cannot exist on Android at all, because the host passes
  `--no-browser` and `shouldEnableLocalAutoLogin` requires its absence. The
  empty default `allowed_cidrs` means the IP allowlist is a documented no-op and
  the password is the boundary.
- **Disposition:** the original wording — "the console was observed listening on
  all interfaces" — is **resolved as explained by design**. Its one remaining
  actionable residue is that "Public Mode off" is not reliably enforced across a
  service restart, which is tracked precisely as `PC-DEF-020` rather than left
  inside this entry.
- **Status:** RESOLVED AS EXPLAINED / superseded by `PC-DEF-020`.

### PC-DEF-003 — Restart-required banner can remain after hot reload

- **Discovered:** Telegram interactive-menu/model-selection closeout.
- **Component:** Dashboard/launcher state presentation.
- **Severity:** Low / cosmetic, as recorded at discovery.
- **Description:** `gateway.bootConfigSignature` is not refreshed after a
  successful in-process reload, so the Dashboard can continue showing
  “Gateway restart required.”
- **Evidence:** `TASKS.md` and `SESSION_HANDOFF.md` entries named “Stale
  Gateway restart required banner.”
- **Reason deferred:** Unrelated to the milestone and non-blocking.
- **Target milestone:** Dedicated dashboard state-correctness maintenance.
- **Status:** OPEN.

### PC-DEF-004 — `BaseChannel` typing defaults are inconsistent

- **Discovered:** Telegram interactive-menu/model-selection closeout.
- **Component:** Non-Telegram channel typing behavior.
- **Severity:** Unrated.
- **Description:** Non-Telegram channels do not share settled typing defaults.
- **Evidence:** Repeated open entries in `TASKS.md`; historical
  `PROJECT_STATE.md` explicitly says defaults must be decided before gating.
- **Reason deferred:** Product semantics were undecided and unrelated to the
  completed Telegram scope.
- **Target milestone:** Dedicated channel behavior milestone.
- **Status:** OPEN.

### PC-DEF-005 — Versioned non-destructive bootstrap updates are unresolved

- **Discovered:** vc55 investigation; architecture implemented during Bootstrap
  Architecture on 2026-09-08.
- **Component:** Workspace template lifecycle.
- **Severity:** Deferred architecture work.
- **Description:** Binary-owned capability guidance now updates safely, but no
  production behavior offers or merges improvements to a pristine historical
  user template. User edits and `MEMORY.md` must never be overwritten.
- **Evidence:** `TASKS.md`, “Bootstrap architecture implemented”; root
  `DECISIONS.md`, “Product guidance ships in the binary; workspace files belong
  to the user”; current release-gate pending list.
- **Reason deferred:** Safe product behavior requires a separate decision and
  cannot be inferred from file equality alone.
- **Target milestone:** Post-stable architecture unless explicitly reprioritized.
- **Status:** OPEN.

### PC-DEF-006 — Full APK reproducibility is not yet proven

- **Discovered:** H1/H1.5 F-Droid readiness audit.
- **Component:** Release build / Official F-Droid path.
- **Severity:** Blocks the target developer-signed F-Droid publication path.
- **Description:** Core, frontend, runtime recipes, and individual payloads have
  reproducibility evidence, but the final hardened APK has not been rebuilt
  twice and compared bit-for-bit.
- **Evidence:** [`FDROID_RELEASE.md`](FDROID_RELEASE.md), sections 3, 4, 6, and
  8; current release-gate pending list.
- **Reason deferred:** Final APK inputs are still changing during H3 and later
  hardening.
- **Target milestone:** Production artifact hardening/reproducibility proof.
- **Not closed by the exposure-audit closure re-run, 2026-09-13.** That audit
  built one APK and did not build and compare two independent final builds, so
  this acceptance criterion is untouched. One observation recorded without
  overstating it: the APK built at the re-run (`113a8382…`) and the one built at
  `PC-DEF-021` (`7155de0a…`) differ while their Dart AOT and R8 mapping are
  byte-identical. The earlier APK had been overwritten by `--clean`, so no
  byte-level comparison was possible; differing archive hashes with identical
  compiled inputs are consistent with packaging non-determinism, which is
  precisely what this defect is about. It gates the F-Droid path, not the
  GitHub / direct APK release.
- **Status:** OPEN.

### PC-DEF-007 — F-Droid builder compatibility and committed prebuilts remain open

- **Discovered:** H1.5D source-build audit.
- **Component:** Official F-Droid submission.
- **Severity:** Submission blocker, not an application runtime defect.
- **Description:** All eight Managed Runtime payloads can be built from source,
  but F-Droid builder acceptance of NDK/Rust/Go recipes and the repository's
  committed native prebuilts is unresolved.
- **Evidence:** [`FDROID_RELEASE.md`](FDROID_RELEASE.md), “Remaining F-Droid
  question” and summary.
- **Reason deferred:** It requires final build metadata and external F-Droid
  policy validation.
- **Target milestone:** F-Droid submission preparation.
- **Status:** OPEN.

## Resolved

### PC-DEF-023 — A third-party Google OAuth client secret is embedded in both Core binaries

- **Discovered:** final release exposure audit, 2026-09-12.
- **Component:** `core/src/pkg/auth/oauth.go`, `GoogleAntigravityOAuthConfig`.
- **Severity:** Third-party credential reuse and availability risk. Not a
  disclosure of any PocketClaw or user secret; not a release blocker.
- **Description:** the Google Cloud Code Assist ("Antigravity") OAuth
  configuration carries a hardcoded client ID **and client secret**, stored
  base64-encoded and decoded at runtime by a local `decodeBase64` helper. The
  encoded form is present in both `libpocketclaw.so` and `libpocketclaw-web.so`;
  the decoded form is not, so a plain string scan for the credential's prefix
  finds nothing. The source comment states these are "the same client
  credentials used by the OpenCode antigravity plugin" — that is, a credential
  registered to another project's Google Cloud account, not PocketClaw's.
  `web/backend/api/oauth.go:552` reaches it, so it is live product surface, not
  dead code.
- **Assessment:** for an installed application this class of secret is not
  confidential — RFC 8252 and Google's own desktop-client model assume it cannot
  be kept — so shipping it does not leak anything that was ever protected. The
  real exposures are different: PocketClaw depends on a credential a third party
  can revoke at any time, which would break the provider for every user; and the
  base64 wrapper means the credential is invisible to routine secret scanning,
  including this audit's own pattern pass. It was found by entropy review.
- **Owner decision, 2026-09-13: do not ship Google Antigravity in v0.2.0.**
  PocketClaw stable will not depend on another project's OAuth client. The
  provider may return later under a PocketClaw-owned integration; see
  `DECISIONS.md`.
- **Fix applied, 2026-09-13 — the provider is removed from the product.**
  Surface by surface rather than by deleting one function:
  `GoogleAntigravityOAuthConfig`, both embedded credentials and the orphaned
  `decodeBase64`; the `googleapis.com` token-URL inference that produced the
  `google-antigravity` provider name; the credential-store alias; the provider
  implementation and its test; the facade type aliases and fetch wrappers; the
  factory construction arm; the product-facing catalogue entry; the keyless
  `model_list` template; the legacy-import protocol mapping; the OAuth API's
  constant, order, methods, labels, config arm, project-ID fetch and default
  model; the implicit-OAuth handling in `models.go` and `model_status.go`; the
  CLI login arm, `authLoginGoogleAntigravity`, `authModelsCmd` and the
  `auth models` subcommand that existed only to list Antigravity models; and on
  the frontend the credential card, the union-type member, the hook status and
  label, and the locale key in all fourteen bundles.

  **Retained deliberately.** `OAuthProviderConfig.ClientSecret` stays: the
  generic token exchange supports confidential clients and that is shared
  infrastructure. `canonicalProvider` keeps its trim/lower-case normalisation,
  which every credential-store caller goes through — only the alias went.
  Gemini is untouched and is a different provider entirely, with its own
  catalogue entry, API-key auth and `generativelanguage.googleapis.com` base.
  OpenAI OAuth, the Anthropic token flow, PKCE, state, the callback and session
  handling are unchanged.

  **Backend behaviour:** `antigravity` and `google-antigravity` fall through to
  the existing unsupported-provider error rather than being special-cased, and
  the provider is absent from the catalogue — not a hidden callable provider
  behind a removed UI.
- **Verification:** `core/src/web/backend/api/no_antigravity_test.go` pins that
  five spellings are rejected as unsupported, that the OAuth surface is OpenAI
  and Anthropic only across order/methods/labels, that no catalogue entry or
  alias mentions it, that Gemini and the `google` → `gemini` alias survive, and
  that neither the encoded nor the decoded third-party credential exists in any
  Go, TS, TSX, JSON, Dart or Kotlin source file. `NormalizeProvider` is
  deliberately *not* the absence assertion: it is a string normaliser that
  echoes an unknown id back and says nothing about registration, which a first
  attempt at this test got wrong.

  Both staged Core binaries carry **zero** occurrences of `google-antigravity`,
  `antigravity`, `Antigravity`, `Google Code Assist`, `antigravity.google` and
  every encoded or decoded credential marker, while Gemini's endpoint and
  display name are still present. The first rebuild still showed three
  `antigravity` strings in the gateway binary: they came from the **embedded**
  agent skill document, which advertised the provider and the removed
  `auth models` command to the agent. That is a shipped product surface, so it
  was corrected and the pair rebuilt.
- **Core impact:** `core/src` changed, so the pair was rebuilt and re-staged
  under the two-commit rule from build-input commit
  `54ff2525fa555744d017aae56c9a26e2049812e1`. Source fingerprint moved from
  `6f00359dc9e8bf7ee24f9d170754b2792a41fb880d9da4f34a8600dd8f99df00` to
  `bc35a598d3a836e0a0c95afc73314fe49a38877b985b5b0f15bab11460184fa9`.

      libpocketclaw.so       37,658,976  0a28bd5e1d6e33dc35b808039571b6683e4d47dec021941650f916641b859f6e
                                         build ID c657e80da54549a3bcc9a8bdba0a576d7b7273a0
      libpocketclaw-web.so   25,319,424  9ae1d2d9e7ac26d602db722649ebec4ac9cd982fa166c50ded303685d2abce50
                                         build ID 45355d87ba5ec042740675f82bc3d3940265318a
      BuildTime              2026-09-13T00:24:56+0000

  Byte-identical in three independent output roots, one with a cold Go cache;
  both private companions likewise — `libpocketclaw.so.debug` 14,525,160 bytes
  `709cf86397dcb365b545eed00a84aede20a20ce77b1663562b1803cc4f77e546` and
  `libpocketclaw-web.so.debug` 9,687,312 bytes
  `a8c8f3d421fdce710a709cb330c79b63fe9aa68d7e0e5d55d58d0dd152006ebe`. Native
  contract 22 PASS / 0 FAIL. No Managed Runtime payload was rebuilt. The support
  manifest still names the PC-DEF-024 audit APK `f580cadc…`, which no longer
  contains this pair — the established pre-artifact state, rebound at the next
  artifact build.
- **Status:** RESOLVED, 2026-09-13.
- **Re-confirmed by the exposure-audit closure re-run, 2026-09-13:** still live
  product surface; the encoded form appears once in each Core binary and the
  decoded form zero times; the upstream-origin comment is intact. An
  installed-app OAuth client class, so the real exposure remains third-party
  revocation/dependency rather than secrecy, and no PocketClaw or user secret is
  disclosed. Classification unchanged; **not a release blocker.**
- **Status:** OPEN.

### PC-DEF-024 — A dead analytics deep link is exported in the release manifest

- **Discovered:** final release exposure audit, 2026-09-12.
- **Component:** `android/app/src/main/AndroidManifest.xml`; `MainActivity`.
- **Severity:** Unnecessary externally reachable entry point. Low.
- **Description:** `MainActivity` is exported and carries a second intent filter
  with `VIEW` + `DEFAULT` + `BROWSABLE` on scheme
  `${POCKETCLAW_UMENG_LINK_SCHEME}`. In the shipped configuration
  `POCKETCLAW_UMENG_APP_KEY` is empty, so `build.gradle.kts:42-46` resolves the
  scheme to the literal `um.placeholder`, and the merged release manifest
  confirms `android:scheme="um.placeholder"` reaches the artifact. Any web page
  can therefore launch PocketClaw with `um.placeholder://…`, and
  `MainActivity.logIncomingIntent` writes the full attacker-supplied URI to
  logcat at INFO. The Umeng SDK it exists for is not packaged
  (`POCKETCLAW_UMENG_PACKAGED=false`, and H1.5 removed the SDK entirely).
- **Assessment:** `MainActivity` is already launchable by any app through its
  LAUNCHER filter, so the added capability is web-originated launch plus
  attacker-controlled text in a log other apps cannot read on current Android.
  No injection surface: the URI is logged and otherwise unused.
- **Fix applied, 2026-09-13 — removed, not made conditional.** The repository
  had already decided this shape for the same integration: the manifest's
  advertising-permission comment records that "an analytics build gets whatever
  the analytics SDK's own AAR manifest declares" and that an app-level
  declaration it genuinely needs belongs to that build's own manifest. A filter
  that is only correct for a build PocketClaw does not ship should not sit in
  the manifest it does, and making it conditional would have added complexity to
  preserve dead code.

  Removed: the `VIEW` + `DEFAULT` + `BROWSABLE` intent filter from
  `AndroidManifest.xml`; `umengLinkScheme`, its `POCKETCLAW_UMENG_LINK_SCHEME`
  `buildConfigField` and its `manifestPlaceholders` entry from
  `build.gradle.kts`; and `MainActivity.logIncomingIntent` with both its call
  sites, plus the `TAG` constant and `android.util.Log` import that existed only
  for it. `onCreate` became an override that only called `super`, so it and the
  then-unused `Bundle` import went too. `setIntent(intent)` in `onNewIntent`
  **stays** — FlutterActivity and plugins read `getIntent()`, and removing the
  logging must not remove real intent handling.

  `POCKETCLAW_UMENG_APP_KEY`, `_CHANNEL` and `_PACKAGED` are **kept**: they are
  consumed by `AnalyticsReporter.kt` and the two `meta-data` entries, so they are
  live plumbing rather than residue, and they are not an exported surface.
- **Verification:** the packaged merged manifest of a fresh LOCAL TEST APK
  (`f580cadc…`) contains **zero** occurrences of `um.placeholder`, `BROWSABLE`,
  `android:scheme` and `action.VIEW`, while `category.LAUNCHER` and
  `.MainActivity` are still present and `debuggable`/`testOnly` remain absent.

  Seven tests in `test/unit/android_release_contract_test.dart` guard it: the
  source manifest declares no scheme at all; no `BROWSABLE` or `VIEW` survives;
  the `MAIN`/`LAUNCHER` contract and the exported launcher activity remain; the
  Gradle link-scheme plumbing is gone so no stale placeholder can survive merge
  processing; the logging branch is gone while `setIntent` remains; the default
  provider still packages no SDK; and the **merged** release manifest carries
  none of it when one has been built. Assertions strip XML comments first and
  check declarations, because the comment documenting the removal necessarily
  names what was removed — running the merged-manifest assertion for real caught
  exactly that, since Gradle carries comments through and only aapt2 strips them.

  Mutation-tested: reintroducing the filter fails the contract.
- **Core impact:** none. Android product source and tests only; the Core source
  fingerprint is unchanged at
  `6f00359dc9e8bf7ee24f9d170754b2792a41fb880d9da4f34a8600dd8f99df00`, Core was
  not rebuilt and the staged pair is untouched.
- **Status:** RESOLVED, 2026-09-13.
- **Re-confirmed by the exposure-audit closure re-run, 2026-09-13:**
  `android:scheme="um.placeholder"` is present in the fresh merged release
  manifest; `POCKETCLAW_ANALYTICS_PROVIDER` still defaults to `none` so the SDK
  is not packaged; `logIncomingIntent` still only writes the URI to logcat.
  `MainActivity` is already LAUNCHER-exported. Classification unchanged; **not a
  release blocker.**
- **Status:** OPEN.

### PC-DEF-022 — `/api/update` fetches and extracts an arbitrary URL with no provenance check

- **Discovered:** final release exposure audit, 2026-09-12.
- **Component:** `core/src/web/backend/api/update.go`; `core/src/pkg/updater`.
- **Severity:** Authenticated attack surface. Not a release blocker.
- **Description:** `POST /api/update` takes a caller-supplied `url`, hands it to
  `updater.UpdateSelfFromRelease`, which downloads the named asset, extracts the
  archive, `chmod 0755`s the binary it finds and calls `selfupdate.Apply` on the
  running executable. The SHA-256 it computes comes from the same
  caller-controlled release document, so it authenticates nothing; there is no
  host allowlist and no signature check. The route is registered unconditionally
  for every platform, including Android.
- **Mitigations that bound it:** the route is behind the dashboard session wall
  (it is absent from `isPublicLauncherDashboardPath`); archive extraction is
  guarded against path traversal in both the zip and tar paths
  (`updater.go:558-562`, `635-638`); and on Android `os.Executable()` is inside
  the read-only install directory, so the replace step cannot succeed. What
  remains is an authenticated arbitrary-URL fetch with archive extraction to a
  temporary directory. Nothing in PocketClaw's own UI calls it — no Flutter,
  Kotlin or dashboard code references `/api/update` — so it is inherited
  upstream desktop surface with no product use.
- **Interaction with `PC-DEF-020`:** in Public Mode the route is LAN-reachable.
  `PC-DEF-020` is now resolved, so Public Mode can no longer be active while the
  UI reports it off; the route is reachable only when the user has deliberately
  enabled LAN access.
- **Re-confirmed by the exposure-audit closure re-run, 2026-09-13:** still
  registered unconditionally, still absent from the unauthenticated allowlist,
  still takes a caller-supplied URL, both traversal guards intact, still called
  by no PocketClaw UI, and the Android apply step still cannot succeed against
  the read-only install directory. Classification unchanged; **not a release
  blocker for the GitHub APK path.**
- **Fix applied, 2026-09-13 — removed.** Securing an unused self-update
  subsystem would have been the wrong repair, so the exposed route is gone
  rather than hardened.

  `core/src/web/backend/api/update.go` is deleted and `router.go` no longer
  calls `registerUpdateRoutes`. No special response was invented: `embed.go`
  already answers an unknown `/api/` path with `http.NotFound`, so
  `POST /api/update` is now an ordinary 404 with no information-leaking envelope
  and no misleading fake success.

  **`pkg/updater` stays.** `cmd/picoclaw/main.go` registers its CLI update
  command, a legitimate non-HTTP consumer, and the library's archive-traversal
  guards and tests are untouched. The acceptance criterion was removal of the
  exposed product route, not maximum source deletion.

  Everything was re-proved before the change rather than taken from the audit
  report: the route was registered at `api/update.go:12`, it was absent from the
  unauthenticated allowlist so a live session was required, and a search across
  Dart, Kotlin, TypeScript and TSX found **zero** callers anywhere outside the
  backend package.
- **Verification:** `core/src/web/backend/api/no_update_route_test.go` — the
  route resolves to no registered pattern under five HTTP methods; an
  authenticated request driven through the routed mux, past any auth wall and so
  matching this route's actual threat model, returns 404 with no handler
  envelope; six plausible renames (`/api/updates`, `/api/self-update`,
  `/api/system/update`, `/api/upgrade`, `/api/download` and others) are checked
  so an arbitrary-download surface cannot reappear under a different name; the
  auth middleware is asserted not to name the path, so removal cannot have
  widened the unauthenticated surface; and the handler file's absence is
  asserted so a revert cannot restore dead code that reads as live product.
  Mutation-tested: restoring `update.go` and its registration fails them.

  The shipped binaries confirm it independently — `/api/update` occurs **zero**
  times in both `libpocketclaw.so` and `libpocketclaw-web.so`, and the web
  binary shrank by 132,864 bytes as the linker dropped the now-unreachable
  paths.

  No regression in the network or auth boundary: `web/backend`, `api`,
  `middleware`, `dashboardauth`, `launcherconfig`, `netbind`, `gateway` and
  `updater` all pass. Public Mode semantics, the dashboard session wall, the
  WebSocket session-plus-origin check and the gateway's loopback pin were not
  touched.
- **Core impact:** `core/src` changed, so the source fingerprint moved from
  `2692de41b2fe2487475911b62cec519193d581b25cf6d0ebe935fc63973229df` to
  `6f00359dc9e8bf7ee24f9d170754b2792a41fb880d9da4f34a8600dd8f99df00` and the
  pair was rebuilt and re-staged under the two-commit rule from build-input
  commit `a0be2a705c1b255c5bd2fe1d8c9f44c094019627`:

      libpocketclaw.so       37,724,640  7ebeebd1…  build ID 512ed36a…
      libpocketclaw-web.so   25,385,088  b682b76d…  build ID 4eeb1385…
      BuildTime              2026-09-12T23:25:34+0000

  Byte-identical in three independent output roots, one with a cold Go cache;
  both private companions likewise. Native contract 22 PASS / 0 FAIL. No Managed
  Runtime payload was rebuilt. The private support manifest's `apk` field still
  names the exposure-audit APK `113a8382…`, which no longer contains this pair —
  the established pre-artifact state, rebound at the next artifact build.
- **Status:** RESOLVED, 2026-09-13.

### PC-DEF-025 — `flutter test` has been failing since H5B and no gate runs it

- **Discovered:** final release exposure audit, 2026-09-12.
- **Component:** `test/unit/namespace_n3_native_identity_test.dart`;
  `tool/release_gate.py` Flutter coverage.
- **Severity:** Test and gate integrity. No product impact.
- **Description:** `flutter test` reports **480 passed, 1 failed**. The failure
  is "the build script still consumes the upstream artifact names", which
  asserts that `core/build-android-arm64.sh` contains the literal
  `build/picoclaw-android-arm64`. H5B made the output root overridable, so the
  script now builds that path from `CORE_BUILD_DIR` / `CORE_OUTPUT_ROOT` and the
  literal no longer appears. The assertion is stale; the behaviour it guards —
  that the install step still consumes the upstream artifact names — is intact.
- **Why it went unnoticed:** the release gate never runs the suite. It runs three
  named files only — `android_release_contract_test.dart`,
  `android_backup_exclusion_test.dart` and
  `android_runtime_secret_placement_test.dart`
  (`tool/release_gate.py:621-633`) — so a red Flutter suite has passed every
  gate since `aa24d9e`, through H5B, H5C, UI-1 and `PC-DEF-019`.
- **Evidence:** the literal is present in `core/build-android-arm64.sh` at
  `6c24f9a` and absent from `aa24d9e` onward; the test file has not been touched
  since `0f0332d`, which predates H5B.
- **Fix applied, 2026-09-13.** Both halves.

  **The assertion.** It now pins the *rename boundary* rather than a path: the
  install step must take upstream `picoclaw-android-arm64` and ship it as
  `libpocketclaw.so`, and `picoclaw-launcher-android-arm64` as
  `libpocketclaw-web.so`, matched as a regex over the real `install -m 0755`
  lines through `$CORE_OUTPUT_ROOT`. It additionally asserts the private-support
  step consumes the same two upstream names — a rename that missed it would
  archive symbols for the wrong binary — and asserts the *absence* of a
  hard-coded `build/` root, which is the shape that went stale. So the test now
  fails if the rename boundary breaks and also fails if someone reintroduces the
  fixed root, while surviving legitimate output-root relocation.

  Mutation-tested both ways against the real script: renaming the upstream
  artifact fails it, and hard-coding `build/picoclaw-android-arm64` fails it.
  The script was restored byte-for-byte afterwards.

  **The gate.** `tool/release_gate.py` now runs the **complete** Flutter suite
  as `flutter.suite`, using a deterministic `find_flutter()` that prefers the
  repository toolchain, then `FLUTTER_ROOT`, and only then `PATH` — a gate that
  answers differently depending on the caller's shell is not a gate. The suite
  runs **once**, through the JSON reporter, and the three named contract items
  (`a1.contracts`, `signing.production_contract`, `a2.placement_guards`) are
  derived from that single run rather than being the whole of it. They are kept
  because a record that says only "the suite passed" loses which guarantee was
  checked.

  A non-zero exit can never be reported as PASS, and exit 0 with no parsed
  results is a FAIL rather than a pass — a suite that did not run must not look
  like a suite that passed. Failure output is bounded and names the failing
  suite and test.
- **Verification:** `flutter analyze` clean; `flutter test` **490 passed, 0
  failed**. Proven against a real red suite: a deliberately failing test placed
  in a file none of the three named contracts covers made the gate exit 1 and
  report `490 passed, 1 failed — …/pc_def_025_probe_test.dart: deliberate
  failure outside the three named contract files`, while the three named items
  stayed PASS. That is precisely the scenario that went unnoticed for five
  milestones. The probe was removed. `tool/test_release_gate.py` grew from 24 to
  **35 tests**, covering full-suite success, a failure outside the named files,
  failing-test identity in the output, bounded output under 100 failures,
  non-zero exit never passing, exit-0-with-no-results failing, the command not
  being the three named files, a missing toolchain skipping rather than passing,
  a named contract failing when its own file fails, and PATH being last in
  resolution order.
- **Gate composition:** the source gate gains one item (25 → 26) and the full
  artifact gate 56 → 57. Measured at `3e3941f` before this change: **56 PASS / 0
  FAIL / 0 SKIPPED**, 56 items on a clean tree.
- **Core impact:** none. Tests, release tooling and documentation only; the
  source fingerprint is unchanged at
  `2692de41b2fe2487475911b62cec519193d581b25cf6d0ebe935fc63973229df` and Core
  was not rebuilt.
- **Status:** RESOLVED, 2026-09-13.

### PC-DEF-021 — The AAB embeds the private R8 mapping and native debug symbols

- **Discovered:** final release exposure audit, 2026-09-12.
- **Component:** Android App Bundle packaging; release-asset policy; release gate.
- **Severity:** **RELEASE BLOCKER for any path that publishes the AAB.** No
  impact on the APK or on installed devices.
- **Description:** `:app:bundleRelease` writes release-support material that the
  APK correctly excludes into `BUNDLE-METADATA/`:

      18,776,264  BUNDLE-METADATA/com.android.tools.build.debugsymbols/arm64-v8a/libflutter.so.sym
      13,630,085  BUNDLE-METADATA/com.android.tools.build.obfuscation/proguard.map
       6,763,120  BUNDLE-METADATA/com.android.tools.build.debugsymbols/arm64-v8a/libapp.so.sym
         159,016  .../debugsymbols/arm64-v8a/libdartjni.so.sym
         134,544  .../debugsymbols/x86_64/libdartjni.so.sym
         109,380  .../debugsymbols/armeabi-v7a/libdartjni.so.sym

  `proguard.map` is SHA-256
  `14d49fad46e773e32da69b7b2336b7a968808cd1130f0319f7806ca4d09c1beb` —
  byte-identical to the private `mapping.txt`. `libapp.so.sym` is native debug
  data for the obfuscated Dart AOT library. Roughly 39.5 MB of the bundle is
  material H3A/H3B/H4A/H4B exist to keep out of distributed artifacts. This is
  AGP's intended design — Play consumes `BUNDLE-METADATA/` for crash
  symbolication and strips it from delivered splits — so it is correct for a
  Play upload and wrong for anything else.
- **Why it is not theoretical:** attaching the AAB to a public GitHub
  pre-release is this project's established practice.
  `PocketClaw-v0.2.0-rc1.aab` and `PocketClaw-v0.2.0-rc2.aab` are published
  assets today. Repeating that for a hardened release would publish the complete
  Java/Kotlin deobfuscation map and the Dart AOT symbols.
- **Why no gate caught it:** `RELEASE_PROCESS.md` states this material stays
  "outside APK/AAB files", which AGP cannot satisfy for a bundle, and
  `artifact.r8_mapping_private` in `tool/release_gate.py:864` is a hardcoded
  `True` whose observation reads "absent from APK" — it verifies nothing and is
  scoped to the APK. The repository also has no AAB build or inspection path at
  all: `tool/build_hardened_android.py` only runs `:app:assembleRelease`, and
  neither the release gate nor the native audit accepts a bundle.
- **Fix applied, 2026-09-12.** All four parts of the plan, and the framing
  changed: the defect is not that AGP writes those entries, it is that
  PocketClaw had no way to say what an artifact was *for*. Contents cannot be
  judged without purpose, so purpose is now declared.

  **(1) Policy corrected.** `RELEASE_PROCESS.md` no longer claims the mapping
  and symbols stay "outside APK/AAB files", which AGP cannot satisfy for a
  bundle and was therefore a rule nothing could obey. It now says: absent from
  every APK, never a public release asset, and expected inside a Play-destined
  bundle's `BUNDLE-METADATA/`. It also says explicitly not to strip them —
  that would remove Play's ability to symbolicate a crash and fix nothing.

  **(2) Distribution classes.** `tool/artifact_policy.py` defines
  `public-release`, `play-upload` and `non-publish-audit`. An AAB may never be
  `public-release`, and the refusal does not depend on what the bundle contains:
  a bundle with no mapping at all is still forbidden, because the prohibition is
  about what the format is for. Detection reads archive contents, not the file
  extension, so renaming a bundle to `.apk` does not launder it — a test covers
  that. A new public release asset allowlist rejects `*.aab`, mapping and usage
  reports, `.debug`/`.sym`/`.dwarf` companions, `.symbols`, symbol and
  private-support archives, keystores and key files, and `.env`, while leaving
  the APK, checksums, notices and source archives permitted.

  **(3) Real gate checks.** `artifact.r8_mapping_private` reads the archive and
  reports what it scanned — `"N archive entries scanned, deobfuscation entries =
  0"` — instead of the hardcoded `True` whose observation read "absent from
  APK". A test asserts the vacuous form cannot come back and that the same code
  answers differently for two different archives. `--verify-bundle` inventories
  modules, ABIs, manifests, native entries and every `BUNDLE-METADATA/` entry by
  name, size, category and Play-acceptability; `--artifact-class` is required for
  any artifact phase and has no default, so an unclassified artifact fails
  closed rather than being assumed public-safe. `--release-assets` checks a
  proposed asset list.

  **(4) rc1/rc2 left in place**, deliberately, and recorded — see below.

  Unrelated private material still fails in **every** class including a Play
  upload. The `BUNDLE-METADATA/` exemption covers exactly two known AGP entry
  shapes, `obfuscation/proguard.map` and `debugsymbols/<abi>/<lib>.so.sym`. That
  narrowing came from this milestone's own test suite: the first implementation
  exempted the whole directory, so a keystore dropped beside the mapping passed.
  It now fails.

  A repository-owned hardened bundle path was added rather than left to an ad-hoc
  Gradle invocation: `tool/build_hardened_android.py --package bundle` runs
  `:app:bundleRelease` through the same hardening contract as the APK — same
  obfuscation, split-debug-info, controlled generated URI, R8, shrinking and
  arm64 target — sharing the Dart verification helpers instead of duplicating
  them. It requires `--artifact-class` and does not offer `public-release`.
- **Verification:** `tool/test_artifact_policy.py`, 32 tests, covering all
  twelve required cases: a clean APK passes and an APK carrying a mapping fails;
  an AAB classified public fails, including one with no metadata at all; a
  Play-upload AAB passes and its expected `proguard.map` and native debug
  metadata are reported as allowed rather than as leakage; a non-publish audit
  AAB passes with the classification notice; a missing or unrecognised class
  fails closed; a Play AAB carrying unrelated private material fails; the asset
  allowlist rejects `*.aab`, mapping files, `.debug`, symbol archives and
  keystore-like material while permitting ordinary assets; and
  `artifact.r8_mapping_private` is shown to be content-derived. Six of those run
  the real gate CLI end to end. `tool/test_release_gate.py` (23 tests) still
  passes unchanged.
- **Historical exposure, not remediated here:** `PocketClaw-v0.2.0-rc1.aab` and
  `PocketClaw-v0.2.0-rc2.aab` remain attached to their published pre-releases.
  They predate Dart obfuscation and R8 minification, so what they disclose is
  not the current hardened mapping, but they are the practice this policy
  retires. This milestone had no authority to mutate published releases, so
  nothing was deleted and no release history was rewritten. The exact owner
  action, if removal is wanted, is recorded in `RELEASE_PROCESS.md` along with
  what it does and does not achieve: it ends ongoing public availability and
  cannot revoke a copy already downloaded, and each asset shows a recorded
  download.
- **Core impact:** none. Only tooling, tests and documentation changed; the Core
  source fingerprint is unchanged at
  `2692de41b2fe2487475911b62cec519193d581b25cf6d0ebe935fc63973229df` and Core
  was not rebuilt.
- **Status:** RESOLVED, 2026-09-12.

### PC-DEF-020 — Public Mode OFF does not guarantee a loopback-only console

- **Discovered:** final release exposure audit, 2026-09-12.
- **Component:** Dashboard listener; Android Public Mode toggle; `launcher-config.json`.
- **Severity:** **RELEASE BLOCKER.** The product's stated network posture can
  differ from the listener it actually opens.
- **Description:** The dashboard's public/loopback decision has two persisted
  authorities that are never reconciled. Android stores the user's choice in
  SharedPreferences `public_mode` and passes `-public` only when it is on. When
  it is off the flag is absent, so `web/backend/main.go` takes
  `effectivePublic = launcherCfg.Public` — the `public` field of
  `launcher-config.json`. Nothing on the Android OFF path ever writes that file:
  `handleAndroidNetworkModeApply` and `launcherHTTPRuntime.ApplyPublicMode`
  rebind the live listener and update in-memory state only, and
  `PUT /api/system/launcher-config` is the single writer of the file. The
  console's own Config page does send `public`, so saving that page while LAN
  access is on persists `public: true`. After that, turning Public Mode off in
  the native UI rebinds the listener to loopback for the life of the process and
  leaves the file saying `true`; the next service start — app restart, service
  kill, device reboot — binds the console to all interfaces while the native
  toggle still reports OFF.
- **Evidence:** `core/src/web/backend/main.go:550-556` (`if !explicitPublic {
  effectivePublic = launcherCfg.Public }`);
  `core/src/web/backend/api/android_bridge.go:154-212` (no config write);
  `core/src/web/backend/launcher_http_runtime.go:134-161` (no config write);
  `core/src/web/backend/api/launcher_config.go:84-96` (the only writer);
  `core/src/web/frontend/src/components/config/config-page.tsx:701-706` (sends
  `public`). Reproducible bind evidence from the shipped `pkg/netbind` with the
  same default-mode selection `openLauncherListeners` applies:

      PUBLIC OFF (no -public, no host)   bindHosts=[::1 127.0.0.1]
      PUBLIC ON  (-public, no host)      bindHosts=[:: 0.0.0.0]
      host override 127.0.0.1 + -public  bindHosts=[127.0.0.1]
      gateway (host=localhost)           bindHosts=[::1 127.0.0.1]  in BOTH states

  The dashboard password wall is unaffected in every state, so this is exposure
  of a password-protected surface, not an unauthenticated one. The Core gateway
  on 18790 stays loopback-only regardless: `openGatewayListeners` always passes
  `netbind.DefaultLoopback` and never sees the launcher's public flag.
- **Fix applied, 2026-09-12** — option (b), plus the display half of (a).
  `PocketClawService` now passes `-public=true` or `-public=false` rather than
  the flag or nothing, so `flag.Visit` always reports the decision as supplied
  and the persisted field is never consulted on Android. `main.go`'s inline
  resolution moved into `resolveLauncherPublicMode` and its `flag.Visit` block
  into `launcherExplicitFlags`, so the state matrix is testable rather than
  arguable. Desktop is unchanged by construction: with no flag supplied the
  stored field is still the authority, which is the only way Public Mode can be
  set where there is no native toggle.

  The Config page was the second half. Making the host authoritative for the
  listener left the page reading and writing the stored field directly, so it
  could display a value the running listener contradicts and saving it rewrote
  the stale value. Where the host owns the decision the page now reports the
  effective mode and persists that instead of the submitted one, which also
  repairs a file that had already drifted. `effectiveLauncherPublic` already
  encoded the precedence and had no product caller; it is wired up rather than
  duplicated, and extended to prefer a runtime rebind over the startup flag
  because `ApplyPublicMode` replaces the listeners without rewriting
  `serverPublic`. The frontend is untouched — the field is informational under a
  host-owned decision by virtue of what the API reports, not by a redesign of
  Settings.

  Contract: `-public=<bool>`, always supplied by the Android host. An explicit
  `false` and an omitted flag are distinguishable because `flag.Visit` reports
  only flags that were `Set`; a test pins that, since the whole fix is inert
  without it.
- **Verification:** the required state matrix is covered by
  `core/src/web/backend/public_mode_authority_test.go` (fresh install off;
  native on; native on with a stored true; native off with a stored true for
  every subsequent process; a stored true predating startup; a stored false with
  native on; host-override precedence in all three public states; the gateway
  loopback for every gateway host value; and the end-to-end case that a stale
  stored true with an explicit off opens loopback sockets and nothing else),
  `core/src/web/backend/api/launcher_config_authority_test.go` (the page reports
  the effective mode, follows a runtime rebind, cannot override a host-owned
  decision, repairs a stale stored true on save, reports off under an explicit
  host, and stays writable on desktop), and
  `test/unit/android_public_mode_authority_test.dart` (the host always states
  the decision and never emits a bare `-public`). The live ON→OFF rebind was
  already covered by
  `TestLauncherHTTPRuntimeAppliesPublicModeWithoutReplacingHandler` and is not
  duplicated. No change to authentication, session handling, the WebSocket
  origin check, the unauthenticated path allowlist, the gateway's loopback pin,
  or Public Mode ON semantics; the middleware, dashboardauth, api,
  launcherconfig, netbind and gateway suites all pass.
- **Core impact:** `core/src` changed, so the source fingerprint moved from
  `bd4a8629a2682e2f05aa3859a400be8a77fb4954ad14994e5703ccbe365d05ec` to
  `2692de41b2fe2487475911b62cec519193d581b25cf6d0ebe935fc63973229df` and the
  pair was rebuilt and re-staged under the two-commit rule from build-input
  commit `f8bc52a0757f7b0a9f6c0704d2a3586db929e33f`:

      libpocketclaw.so       37,724,640  602ce034…  build ID ed130bed…
      libpocketclaw-web.so   25,517,952  b5cce071…  build ID f61a369f…
      BuildTime              2026-09-12T18:54:26+0000

  Byte-identical in three independent output roots, one with a cold Go cache;
  both private companions likewise. Native contract 22 PASS / 0 FAIL for the
  pair. No Managed Runtime payload was rebuilt.
- **Status:** RESOLVED, 2026-09-12.

### PC-DEF-019 — Staged Core predates the guided-tour dashboard change

- **Discovered:** guided-tour hardening repair, 2026-09-12.
- **Component:** staged `libpocketclaw.so` / `libpocketclaw-web.so`.
- **Severity:** Release blocker for the next artifact build; no runtime defect.
- **Description:** `libpocketclaw-web.so` embeds the compiled dashboard, so
  everything under `core/src/web/frontend` is a Core build input. Repairing the
  tour moved the Core source fingerprint from
  `86369a32a9873715672f7867b31dcd72a7d19088c49cdb1df2b584c548ba4c73` to
  `bd4a8629a2682e2f05aa3859a400be8a77fb4954ad14994e5703ccbe365d05ec`, so the
  staged pair no longer matches the source it is supposed to be built from.
- **Evidence:** `cmd/corefingerprint` recomputes the new value; the test-class
  source gate reports `core.staged_freshness` and `build.reproducibility_tests`
  FAIL with 22 PASS / 2 FAIL / 1 SKIPPED. Every other gate is unaffected.
- **Reason deferred:** This repair had no authority to rebuild Core. The H5C
  production artifact and its evidence remain historically valid; they simply
  describe the previous dashboard.
- **Resolution:** Core was rebuilt and re-staged on 2026-09-12 from canonical
  build-input commit `ea43369289c8b6c618faa08f7b91355882fc050c` — the UI-1
  source commit — under the two-commit rule, so the staged pair landed in a
  following commit that changes no build input and the source fingerprint stayed
  `bd4a8629a2682e2f05aa3859a400be8a77fb4954ad14994e5703ccbe365d05ec`.

      libpocketclaw.so       37,724,640 bytes
                             f273b9ced85f4d00cb542df9c2f4c691b4151526cb0ac9c2c7612a1432d7230f
                             build ID 25e206ab402f8cd44a766bc03935468beebd8633
      libpocketclaw-web.so   25,517,952 bytes
                             900c43fcaad2094017c6959eed623d1e2499cfd560f01f2cff36dd34202b86b9
                             build ID 84afbe2439b779722b22c2b4c6aa1300cd3ef199
      BuildTime              2026-09-12T07:27:12+0000

  Byte-identical in three independent output roots, one with a cold Go cache.
  `core.staged_freshness` and `build.reproducibility_tests` are PASS; the whole
  `pkg/coresource` package passes, 46 tests. The embedded dashboard is identified
  by content rather than timestamp: UI-1's `__pocketclaw_tour_probe__` is present
  and the deleted docs-step copy is absent, both reversed in the binary staged at
  `ea43369`. The pair holds the H5B/H5C native contract at 22 PASS / 0 FAIL under
  the repository's own ELF audit logic, and both private-support companions were
  rebuilt and rebound. Only the Core pair changed; no Managed Runtime payload,
  export map or `PC-DEF-012` disposition was touched, and no APK or AAB was
  built. Evidence:
  [`docs/prompts/history/PC-DEF-019_CORE_REBUILD_RESTAGE.md`](prompts/history/PC-DEF-019_CORE_REBUILD_RESTAGE.md).
- **Status:** RESOLVED, 2026-09-12.

### PC-DEF-013 — Guided tour placed its card outside the viewport in RTL

- **Phase discovered:** Guided-tour read-only audit, 2026-09-12.
- **Component:** `core/src/web/frontend/src/components/tour/tour-guide.tsx`.
- **Problem/root cause:** Steps declared a physical placement (`"left"` /
  `"right"`). The sidebar anchors to the right edge in Arabic — see
  `sidebar-direction.test.tsx` — so the Models step computed
  `left = rect.right + 12` and put the card, and the only Next button, past the
  viewport edge. There was no flip and no clamp, so the step could not be
  advanced, dismissed or reached at all.
- **Resolution:** Placement is now logical (`start`/`end`/`above`/`below`),
  resolved against the document direction, and the finished rectangle is clamped
  inside the viewport with an 8px margin. The clamp is the guarantee: no layout
  can put a tour control out of reach.
- **Verification:** Measured in real Chrome and Brave at 1440x900. In Arabic the
  live target is `left 1193 / right 1432`; the pre-fix formula would have placed
  the card at `left 1444`, right edge `1764`, off-screen. It now renders at
  `861..1181`, fully inside, with the primary control hit-testable at its own
  centre. Both directions, every step, both browsers.
- **Commit:** the guided-tour hardening commit containing this record.
- **Status:** RESOLVED.

### PC-DEF-014 — Guided tour left focus on the control it spotlighted

- **Phase discovered:** Guided-tour read-only audit, 2026-09-12.
- **Component:** `tour-guide.tsx`; the visible symptom surfaced on
  `SidebarMenuButton`.
- **Problem/root cause:** Reported as a highlight the tour failed to clear. It
  was not: the audit proved every tour node unmounts on every close path and
  that the tour never touches the target's classes, attributes or inline style.
  The spotlight was `pointer-events-none`, so a click on the highlighted item
  passed through to the real `<Link>`; the app navigated and the anchor kept DOM
  focus. `sidebarMenuButtonVariants` carries `outline-hidden focus-visible:ring-2`,
  so that anchor then painted a persistent ring almost identical to the tour's
  own spotlight, and it outlived the tour.
- **Resolution:** Two explicit policies instead of an accident. The spotlight now
  captures the click and swallows it, so the tour's own buttons own progression.
  And the tour captures the previously focused element when it opens and restores
  it on every termination path — finish, skip, Escape, click-outside, unmount —
  blurring instead when the opener is gone.
- **Verification:** In Chrome and Brave, clicking the spotlight leaves the path
  at `/`, keeps the tour open, and leaves no focused `[data-tour]` element. After
  the tour closes: `0` tour nodes, `activeElement` is `BODY`, no focused tour
  target. Route-active styling is untouched and still correct.
- **Commit:** the guided-tour hardening commit containing this record.
- **Status:** RESOLVED.

### PC-DEF-015 — Guided tour declared a step for a control that does not exist

- **Phase discovered:** Guided-tour read-only audit, 2026-09-12.
- **Component:** `tour-guide.tsx` step table; `tour.docs.*` translations.
- **Problem/root cause:** The `docs` step targeted `[data-tour='docs-button']`.
  No component has ever rendered that attribute, and the console has no
  documentation control for it to point at — the translated copy described a
  button in the top-right corner that does not exist. The step highlighted
  nothing in every layout and every language, and never failed loudly because
  a missing target fell back to a centred card.
- **Resolution:** The step is removed rather than answered with a new control
  invented to satisfy the tour. The now-unreferenced `tour.docs.*` keys are
  removed from all 14 locales and the i18n assertion re-pointed at a live step.
  The tour is three steps; the counter reads `1 / 3` through `3 / 3`.
- **Verification:** A structural test walks the step table and fails if any
  declared selector names a `data-tour` attribute no component renders, so a
  renamed target now breaks CI instead of shipping. Confirmed in-browser: the
  counter reads `1 / 3` in Chrome and Brave.
- **Commit:** the guided-tour hardening commit containing this record.
- **Status:** RESOLVED.

### PC-DEF-016 — Guided tour never re-measured its target

- **Phase discovered:** Guided-tour read-only audit, 2026-09-12.
- **Component:** `tour-guide.tsx`.
- **Problem/root cause:** Geometry was read during the render a click produced
  and never again — the component had no effect, listener or observer of any
  kind. Scrolling, resizing, a breakpoint swap or the header finishing its first
  data load all left the spotlight stranded at coordinates that no longer meant
  anything. An all-zero rect from an unlaid-out element was also accepted as
  valid, producing a 16px spotlight at (-8,-8) that dimmed the whole screen from
  the corner.
- **Resolution:** A target is eligible only when connected, not `display:none`,
  not `visibility:hidden` and of non-zero size. Resolution retries on
  `requestAnimationFrame` under a finite 90-frame budget and then degrades to a
  centred card — never a fixed delay, never an unbounded wait. While a step is
  live, a `ResizeObserver`, capture-phase `scroll`, `resize` and a
  `MutationObserver` keep it synchronized, and the target is brought into view
  through its own scroll container with `scrollIntoView({block:"nearest"})`
  rather than scrolling the page. All of it is released on step change, close
  and unmount.
- **Verification:** Regression tests move the target, resize, remove it from the
  DOM and swap it for a replacement, asserting the overlay follows or degrades;
  a listener-balance test proves `scroll` and `resize` counts return to zero
  after close.
- **Commit:** the guided-tour hardening commit containing this record.
- **Status:** RESOLVED.

### PC-DEF-017 — Guided tour backdrop blocked the app with no way out

- **Phase discovered:** Guided-tour read-only audit, 2026-09-12.
- **Component:** `tour-guide.tsx`.
- **Problem/root cause:** The no-target backdrop was a full-screen
  `fixed inset-0` layer without `pointer-events-none`; it absorbed every click
  aimed at the application. Escape did nothing, clicking outside did nothing,
  and nothing in the app could reopen the tour once dismissed. On a large
  desktop screen a dimmed, blurred, click-dead page with one small card reads as
  a freeze.
- **Resolution:** The dimmer is an explicit dismissal surface: clicking it ends
  the tour. Escape ends it too. The spotlight still blocks its target, by
  design, but it is the only blocking region and the card is always reachable.
- **Verification:** Tests assert Escape and a dimmer click each leave zero tour
  nodes and record completion, and that the centred fallback is dismissible.
  Confirmed in Chrome and Brave.
- **Commit:** the guided-tour hardening commit containing this record.
- **Status:** RESOLVED.

### PC-DEF-018 — Guided tour state had no schema version

- **Phase discovered:** Guided-tour read-only audit, 2026-09-12.
- **Component:** `core/src/web/frontend/src/store/tour.ts`.
- **Problem/root cause:** `localStorage["pocketclaw-tour-state"]` held
  `{currentStep, isActive}` with no version, so a changed step list would strand
  users on a step that no longer exists and the tour could never be replayed
  deliberately. It also meant the reported "the tour comes back after updates"
  had no versioning explanation: the real cause is that the embedded WebView
  (`http://127.0.0.1:<port>`) and Public Mode (`http://<device-ip>:18800`) are
  different origins with independent storage, so a changed LAN IP presents a
  fresh origin. That is browser behaviour and is left alone.
- **Resolution:** State is `{version, currentStep, isActive}` with
  `TOUR_VERSION = 1` and a migration that reads an absent version as 0. Anyone
  who finished stays finished; a step this build no longer defines completes
  rather than stranding; a corrupt value falls back to the default instead of
  throwing; state written by a newer build is left alone. `localStorage` is
  probed and degrades to memory where it is unavailable.
- **Verification:** Migration tests cover completed state, a resumable
  unversioned step, the removed `docs` step, corrupt values and future versions.
  Confirmed in-browser: a stored completed state renders zero tour nodes, and
  legacy `{currentStep:"docs", isActive:true}` renders zero tour nodes rather
  than hanging.
- **Commit:** the guided-tour hardening commit containing this record.
- **Status:** RESOLVED.

### PC-DEF-009 — Managed Git HTTP helper carried a build-only RUNPATH

- **Phase discovered:** H5A native/ELF audit, 2026-09-11.
- **Component:** Managed Runtime `libpocketclaw-git-remote-http.so`.
- **Problem:** The packaged PIE carried `DT_RUNPATH`
  `/tmp/pocketclaw-runtime-build/deps/lib`, a build-host search path with no
  runtime purpose on Android.
- **Resolution:** H5B found the cause in git's own Makefile, which turns
  `CURLDIR` into `-Wl,-rpath,$CURLDIR/lib`. The recipe now passes
  `CURL_CFLAGS="-I$DEPS_PREFIX/include"` and an explicit `CURL_LDFLAGS` library
  list, with `-L$DEPS_PREFIX/lib` in `LDFLAGS`. No finished ELF was rewritten.
- **Evidence:** Candidate APK SHA-256
  `d4fe2c4a035051e3b6500d2a2fe9bdad639c97323c355b26a3f8ae6f215b9dd8`; the
  enforced audit reports `no_runtime_search_path` PASS for all 18 packaged
  entries, and the payload still resolves only `libz.so`, `libdl.so`,
  `libc.so`. `install_payload` now fails on any RPATH/RUNPATH, not on one known
  root.
- **Status:** RESOLVED. Confirmed on the Samsung SM-A165F in the H5B
  physical smoke and again under the enrolled production signer in H5C, whose
  packaged native payloads are byte-identical to the artifact that ran on the
  device.

### PC-DEF-010 — Three runtime payloads retained the neutral build root

- **Phase discovered:** H5A native/ELF audit, 2026-09-11.
- **Component:** Managed Runtime curl, Git HTTP helper, and Python payloads.
- **Problem:** curl and the Git HTTP helper each carried ten mbedTLS source
  paths under `/tmp/pocketclaw-runtime-build/`, and Python one CPython build
  root, despite an existing `-ffile-prefix-map`.
- **Resolution:** H5B added shared `-ffile-prefix-map` / `-fdebug-prefix-map` /
  `-fmacro-prefix-map` settings and widened ripgrep's `--remap-path-prefix` to
  the whole build root. Two cases needed more, because a prefix map cannot
  rewrite a string the build wrote into generated *source*: jq records its
  literal `CFLAGS` in `src/config_opts.inc`, and CPython compiles its
  configure-time `VPATH` into `getpath.c` as a C string literal. Both generated
  inputs are normalized before compilation, the CPython one only after the host
  build interpreter is complete.
- **Evidence:** `strings` over all ten payloads finds zero build roots,
  `/home/lordegypt` or checkout paths; the audit asserts `build_path_privacy`
  per entry, and `install_payload` fails the build if `$BUILD_ROOT` survives.
- **Status:** RESOLVED. Confirmed on the Samsung SM-A165F in the H5B
  physical smoke and again under the enrolled production signer in H5C, whose
  packaged native payloads are byte-identical to the artifact that ran on the
  device.

### PC-DEF-011 — Native private symbol companions are now preserved

- **Phase discovered:** H5A native/ELF audit, 2026-09-11.
- **Component:** PocketClaw Core and Managed Runtime build recipes.
- **Problem:** All packaged payloads were stripped, correctly, but no recipe
  preserved a symbol-capable precursor, so a native crash address from a
  shipped build could not be resolved.
- **Resolution:** H5B builds every owned payload with debug information, strips
  the shipped copy, and derives a `.debug` companion from the same link through
  `tool/native_support.py`. Core drops Go's `-s -w` and strips the installed
  copy instead, which is what makes a Go companion possible; Python's companion
  comes from the interpreter before its standard library is appended, because
  that append is why the shipped file cannot be stripped.
- **Evidence:** Ignored `build/private-symbols/native/android-arm64/` holds ten
  companions and a 0600 manifest binding each to its shipped hash, size and
  build ID, plus a resolved representative function, and bound to the exact
  candidate APK. The audit's five `native.private_support_*` checks pass and
  nothing is tracked by Git. ripgrep's entry point is a qualified result and is
  recorded as such in the H5B operating record.
- **Status:** RESOLVED. Confirmed on the Samsung SM-A165F in the H5B
  physical smoke and again under the enrolled production signer in H5C, whose
  packaged native payloads are byte-identical to the artifact that ran on the
  device.

### PC-DEF-008 — Dart intermediate strip boundary verified

- **Phase discovered:** Post-H3B packaged-DWARF inspection.
- **Component:** Dart AOT intermediate / Android native-library packaging.
- **Problem:** Flutter warned that `gen_snapshot` emitted unobfuscated DWARF,
  raising the question whether source-level debug data reached the APK.
- **Resolution/conclusion:** H5A rechecked the exact H4B APK. Its 5,702,536-byte
  `libapp.so` is byte-identical to the H3/H4 Dart AOT evidence and is stripped:
  it has no `.debug_*`, `.zdebug_*`, `.symtab`, `.strtab`, source path, or
  application-name exposure. Its only dynamic exports are the three Flutter
  snapshot symbols; its 45-byte `.eh_frame` is unwind metadata. The ignored
  intermediate may contain DWARF, while the required private Dart split-debug
  file remains external.
- **Verification:** APK SHA-256
  `14ba7d138a4092aefe264c7e2af6240c97fc1b782ded69918cbf545351eb5eb2`;
  packaged `libapp.so` SHA-256
  `c7b2a885ff843a20c57097a0d16ba07c728bd64cf17455a1ce61e6f463a5ae77`;
  `readelf`, `file`, `strings`, and the H5A automated audit agree.
  `gen_snapshot --strip` would not reduce distributed exposure because AGP
  already produces the desired packaged result; enabling it could interfere
  with the established external symbol/reproducibility contract without a
  demonstrated release benefit.
- **Commit:** H5A audit/closeout commit containing this record.
- **Status:** RESOLVED / VERIFIED NON-BLOCKING. Future Flutter/AGP changes must
  retain the packaged-DWARF regression check.

### PC-DEF-R018 — Runtime payload epoch was derived from `HEAD`

- **Phase discovered:** H5B review of the inherited native implementation.
- **Component:** `runtime/android-build-env.sh`.
- **Problem/root cause:** `SOURCE_DATE_EPOCH` defaulted to
  `git show -s --format=%ct HEAD`. `libpocketclaw-python.so` embeds that date
  literally, so any commit — documentation included — changed the bytes the
  Core catalog had just pinned. The commit recording a checksum would have
  invalidated it, and the catalog could never be reproduced from the tree
  carrying it. `core/resolve-build-time.sh` documents this exact failure for
  Core and solves it by path scoping, which cannot help here because the
  recipes are their own build input.
- **Resolution:** Pin `RUNTIME_EPOCH=1789157892` as a build input alongside the
  tarball checksums, still overridable by an explicit `SOURCE_DATE_EPOCH`,
  which is now validated as Unix seconds.
- **Verification:** A regression test asserts no `HEAD`-derived derivation
  remains, that sourcing the script resolves to the pinned value, and that the
  staged Python payload actually contains that epoch's UTC date. The pinned
  value equals the one the payloads were built with, so no byte moved.
- **Commit:** H5B source commit `aa24d9e`.
- **Status:** RESOLVED.

### PC-DEF-R017 — Private companion path was taken from an unvalidated argument

- **Phase discovered:** H5B review of the inherited native implementation.
- **Component:** `tool/native_support.py`.
- **Problem/root cause:** The companion path was built directly from
  `--logical-name`. A name containing a separator or `..` would have written
  outside the private root, and the manifest's relative `supportPath` would
  then have been wrong about where the file is. The audit reads that name back
  out of the manifest, so the value is not purely internal.
- **Resolution:** Constrain the logical name to a plain file name and require
  the resolved path to stay directly inside the private root.
- **Verification:** A focused test rejects `../escape`, `nested/name.so`,
  `/absolute`, empty, `.` and `..`, and accepts a real payload name. The audit
  side has its own test that a manifest naming a support file outside its root
  fails `native.private_support_hashes`.
- **Commit:** H5B source commit `aa24d9e`.
- **Status:** RESOLVED.

### PC-DEF-R016 — A malformed private manifest raised instead of failing closed

- **Phase discovered:** H5B review of the inherited native implementation.
- **Component:** `tool/native_support.py`, `tool/native_elf_audit.py`.
- **Problem/root cause:** Both tools assumed a well-formed manifest. A truncated
  or hand-edited file made `update_manifest` raise an opaque `KeyError`, and the
  audit crashed with an unhandled `CalledProcessError` when a `supportPath`
  named something `readelf` cannot parse. An audit must report on whatever the
  private root actually contains.
- **Resolution:** `update_manifest` reports a malformed manifest as an
  actionable error and leaves the file untouched; the audit treats a malformed
  manifest, a non-object artifact list and an unparsable support file as
  findings.
- **Verification:** Tests cover truncated JSON, a non-list `artifacts`, a list
  of non-objects, a top-level array, a non-ELF support file and a non-object
  `symbolization`; every case fails closed and the malformed file is unchanged.
- **Commit:** H5B source commit `aa24d9e`.
- **Status:** RESOLVED.

### PC-DEF-R015 — Private symbols were archived before the release checks ran

- **Phase discovered:** H5B review of the inherited native implementation.
- **Component:** `runtime/android-build-env.sh`, `core/build-android-arm64.sh`.
- **Problem/root cause:** Both recipes captured the companion and wrote its
  manifest entry immediately after stripping, before the build-path privacy,
  RUNPATH, ABI and page-alignment checks. A build rejected by any of those would
  have left a support file and a manifest entry describing bytes that were
  never adopted.
- **Resolution:** Move the capture to the end of both recipes, after every
  check.
- **Verification:** A test asserts the capture appears after the `-trimpath`
  and source-fingerprint guards in the Core recipe, and the full third-root
  rebuild produced identical payloads and companions under the new order.
- **Commit:** H5B source commit `aa24d9e`.
- **Status:** RESOLVED.

### PC-DEF-R014 — ABI check could fail by SIGPIPE rather than by machine type

- **Phase discovered:** H5B third-root reproducibility build.
- **Component:** `runtime/android-build-env.sh`, `install_payload`.
- **Problem/root cause:** The check piped `llvm-readelf -h` into `grep -q` under
  `pipefail`. `grep -q` exits on its match, which can leave the reader writing
  into a closed pipe; the resulting SIGPIPE fails the pipeline for a reason
  unrelated to the machine type. It misfired once on a python payload whose
  bytes were provably correct and byte-identical to two other roots. This
  predates H5B and was fixed because it blocked the milestone.
- **Resolution:** Capture the header into a variable and match it with `case`.
- **Verification:** The same payload re-ran through `install_payload` and passed
  with the identical SHA-256 it had already produced in three roots.
- **Commit:** H5B closeout commit containing this record.
- **Status:** RESOLVED.

### PC-DEF-R013 — Release manifest still listed H4B as pending after validation

- **Phase discovered:** H4B production artifact inspection.
- **Component:** Release-gate status metadata.
- **Problem/root cause:** All 30 production artifact checks passed, but the
  manifest's static pending-hardening list still said R8 production-signed
  validation was pending. That completed-milestone text would make a valid H4B
  closeout internally contradictory.
- **Resolution:** Remove only the completed H4B item. Keep the independently
  open APK reproducibility and bootstrap-strategy items unchanged.
- **Verification:** A focused regression test requires the H4B item to be
  absent and the F-Droid reproducibility item to remain. The exact existing APK
  then passes the production artifact gate again without a rebuild.
- **Commit:** H4B closeout commit containing this record.
- **Status:** RESOLVED.

### PC-DEF-R012 — H4B helper selected the repository root as the Gradle project

- **Phase discovered:** H4B owner production-signing validation.
- **Component:** Temporary owner-local signing helper.
- **Problem/root cause:** The first helper invoked `android/gradlew` while its
  working directory remained the repository root. A Gradle wrapper locates its
  distribution but does not make its own directory the project root, so
  `:app:validateReleaseSigning` failed before assembly because the repository
  root is not a Gradle build.
- **Resolution:** Every helper Gradle invocation uses the canonical Android
  project explicitly with `android/gradlew -p android`. Before the first hidden
  prompt, the helper now verifies the Android settings and app build files and
  executes that exact validation route without signing variables, requiring
  the expected missing-material failure from `:app:validateReleaseSigning`.
- **Verification:** Twelve focused structural/order assertions and an EOF dry
  run proved the repository root is not selected, the Android project root is
  canonical, the validation task reaches `:app`, and all non-secret checks run
  before either password prompt. The corrected owner run then completed the
  production build and the exact APK passed 30/30 production artifact checks.
- **Commit:** H4B closeout commit containing this record.
- **Status:** RESOLVED.

### PC-DEF-R011 — H4A output assertion mistook R8 removal for missing obfuscation

- **Phase discovered:** H4A local-test validation.
- **Component:** Canonical hardened-build R8 evidence checker.
- **Problem/root cause:** The first H4A post-build assertion required at least
  four exact internal class entries in `mapping.txt`. R8 correctly renamed
  three probe classes and removed or folded three others, so the Gradle build
  and APK were valid but the new assertion reported only three mappings.
- **Resolution:** Account for each probe through its exact renamed mapping or
  through `usage.txt`/nested mapping evidence of removal or folding, and require
  every original clear DEX descriptor to be absent. Manifest components remain
  a separate exact-name preservation check.
- **Verification:** The already-built fresh APK passes the corrected output
  inspection and 30/30 local-test artifact gates. Focused fixtures cover both
  renaming and removal/folding, missing mapping/usage output, packaged mapping,
  blanket rules, and disabled minification/resource shrinking.
- **Commit:** H4A closeout commit containing this record.
- **Status:** RESOLVED.

### PC-DEF-R009 — Cached Dart AOT could outlive its deleted split debug info

- **Phase discovered:** H3B owner production-signing validation.
- **Component:** Canonical Dart-hardened Android build helper / Flutter 3.47.1
  incremental build cache.
- **Problem/root cause:** The helper deleted the expected private DWARF before
  the build, but `:app:clean` did not invalidate `.dart_tool/flutter_build`.
  Flutter reused cached `app.so` because signing does not change Dart AOT inputs
  and the external split-debug-info file is not a tracked cache output. The APK
  assembled correctly while the required private symbol file was not recreated.
- **Resolution:** Clear only Flutter's generated `.dart_tool/flutter_build`
  cache before every hardened assembly so `gen_snapshot` must regenerate AOT
  and private DWARF as one pair. Refuse a symlinked cache path.
- **Verification:** The first diagnostic APK was production-signed and carried
  H3A-identical AOT but had no symbol file; its gate was 23 PASS / 2 FAIL / 0
  SKIP. After the fix, the owner rerun produced APK SHA-256
  `ceef6640d8abd9d084c3ff37d8e903aaf3c82b287de65ec15a37d91124bdebe6`
  with H3A-identical AOT and DWARF. The production artifact gate passed 25 / 25.
  Focused tests cover stale-cache removal, package-config preservation, symlink
  refusal, and cwd-independent symbol resolution.
- **Commit:** H3B closeout commit containing this record.
- **Status:** RESOLVED.

### PC-DEF-R010 — Owner signing helper prompted before Java preflight

- **Phase discovered:** H3B owner production-signing validation.
- **Component:** Temporary owner-local signing helper.
- **Problem/root cause:** The first temporary helper collected both owner
  passwords before checking `JAVA_HOME` and Java availability. Its trap still
  cleared the environment and no value was printed or stored, but the secret
  prompts occurred before all non-secret prerequisites had passed.
- **Resolution:** The corrected external helper validates JDK 17, Python,
  Gradle, repository/helper paths, and keystore presence before its first hidden
  prompt. The durable signing policy now requires this ordering for every future
  owner-secret helper.
- **Verification:** A missing-Java dry run exited before any prompt; a
  correctly configured EOF-only dry run completed all non-secret checks and did
  not begin a build. The subsequent owner rerun completed the production build,
  and the helper's exit trap cleared all four signing variables.
- **Commit:** H3B closeout commit containing this record.
- **Status:** RESOLVED.

### PC-DEF-R008 — Dart snapshot exposed an absolute generated-source URI

- **Phase discovered:** H2 artifact inspection; resolved in H3A.
- **Component:** Flutter/Dart release artifact and Gradle build path.
- **Problem/root cause:** `libapp.so` embedded
  `file:///home/lordegypt/PocketClaw-App/.dart_tool/flutter_build/dart_plugin_registrant.dart`.
  Flutter 3.47.1's Gradle plugin reads `filesystem-roots` and
  `filesystem-scheme` but does not forward those task fields to `flutter
  assemble`; direct and extra-frontend trials therefore left the absolute URI
  unchanged. The generated registrant also sits outside every package URI root
  in Pub's normal package config.
- **Resolution:** The canonical helper adds a deterministic generated-only
  package mapping before Gradle configuration. Flutter's own
  `toPackageUriForWorkspace` path then emits
  `package:pocketclaw_generated/dart_plugin_registrant.dart`. Gradle rejects a
  hardened compile without that exact mapping.
- **Verification:** Two clean local-test builds with different split-info roots
  produced identical Dart AOT SHA-256
  `c7b2a885ff843a20c57097a0d16ba07c728bd64cf17455a1ce61e6f463a5ae77`;
  the H3A artifact gate records `artifact.dart_snapshot_paths` PASS and 25 PASS
  / 0 FAIL / 0 SKIPPED overall.
- **Commit:** H3A closeout commit containing this record.
- **Status:** RESOLVED; H3B subsequently validated the same contract under the
  enrolled production signer.

### PC-DEF-R001 — Keystore helper could report a blank fingerprint as success

- **Phase discovered:** H2 signer enrollment.
- **Component:** `tool/create_release_keystore.sh`.
- **Problem/root cause:** The helper discarded `keytool` stderr, hiding its
  password prompt and integrity warning. The remaining pipeline found no digest
  but still exited successfully through `tr`.
- **Resolution:** Preserve stderr, pass passwords by environment-variable name,
  share one fingerprint implementation, and validate the output shape.
- **Verification:** `tool/test_create_release_keystore.py` creates a disposable
  keystore, verifies the digest, and proves the passwordless path fails with
  empty stdout.
- **Commit:** `78d33fd5b179dd52c7cd8118a5d23ec19c8368ec`.
- **Status:** RESOLVED.

### PC-DEF-R002 — Signing pending-state text outlived completed work

- **Phase discovered:** H2 closeout.
- **Component:** Release-gate state and signing documentation.
- **Problem/root cause:** After enrollment, the gate correctly narrowed “key not
  created” to “no artifact signed”; after private validation, that second state
  also became stale.
- **Resolution:** Remove the resolved pending item and update the authoritative
  H2 state without changing signing logic.
- **Verification:** Production artifact gate: 20 PASS, 0 FAIL, 1 expected SKIP;
  production source gate: 21/21 PASS.
- **Commits:** `78d33fd`, `7f19309`, `0be6afb`.
- **Status:** RESOLVED.

### PC-DEF-R003 — `gh` source recipe broke after canonical environment import

- **Phase discovered:** H1.5 source-build proof.
- **Component:** Managed Runtime `gh` build recipe.
- **Problem/root cause:** The Android DNS resolver gained a `canonicalenv`
  import, while the vendoring guard permitted standard-library imports only.
- **Resolution:** Vendor the stdlib-only `canonicalenv` leaf beside the resolver
  and allow exactly that import.
- **Verification:** Two source builds produced identical adopted bytes; manifest
  checksum and packaged payload match.
- **Commits:** `6f117d3`, adopted by `51ed822`–`fb38c7d`.
- **Status:** RESOLVED.

### PC-DEF-R004 — Python payload embedded wall-clock ZIP timestamps

- **Phase discovered:** H1.5 source-build proof.
- **Component:** `runtime/python-lite-stdlib.py`.
- **Problem/root cause:** Appended standard-library ZIP entries used wall-clock
  timestamps, making each payload different.
- **Resolution:** Use `SOURCE_DATE_EPOCH`, fixed permissions, and a sorted walk.
- **Verification:** Consecutive pinned-epoch builds were byte-identical; adopted
  payload, manifest checksum, and packaged payload match.
- **Commits:** `6f117d3`, adopted by `51ed822`–`fb38c7d`.
- **Status:** RESOLVED.

### PC-DEF-R005 — Canonical APK packaged Firebase/GMS and fetched fonts at runtime

- **Phase discovered:** H1/H1.5 F-Droid audit.
- **Component:** Flutter dependencies, Android manifest, and app typography.
- **Problem/root cause:** Proprietary SDK dependencies were packaged regardless
  of runtime use, and `google_fonts` defaulted to network fetching.
- **Resolution:** Remove Firebase/GMS from the canonical build; bundle Inter and
  Fira Code with their license texts and remove `google_fonts`.
- **Verification:** Source/artifact gates find no Firebase/GMS/AdMob/measurement
  surface; font contract tests prove local packaged assets.
- **Commits:** `612ce96` and `6f117d3`.
- **Status:** RESOLVED.

### PC-DEF-R006 — Artifact-only Core provenance check had no comparison value

- **Phase discovered:** vc62 Zero-Pico artifact validation.
- **Component:** `tool/release_gate.py`.
- **Problem/root cause:** `artifact.core_provenance_pair` read a fingerprint fact
  populated only by source mode, so correct artifact-only verification failed.
- **Resolution:** Resolve and cache the Core fingerprint on demand for both
  source and artifact paths.
- **Verification:** vc62 artifact gate and later H2 production artifact gate
  both pass the provenance-pair check.
- **Commit:** `84080a5`.
- **Status:** RESOLVED.

### PC-DEF-R007 — H2 closeout initially counted the summary as a gate item

- **Phase discovered:** H2 documentation closeout.
- **Component:** Evidence reporting.
- **Problem/root cause:** A temporary count included the final `PASS — ...`
  summary line in addition to the named checks.
- **Resolution:** Count only named gate rows and correct all recorded totals.
- **Verification:** 20 named PASS rows, 0 FAIL rows, 1 named SKIPPED row.
- **Commit:** `0be6afbd92209953d918d5c0516662bc54f0d081`.
- **Status:** RESOLVED.
