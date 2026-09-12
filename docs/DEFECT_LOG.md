# PocketClaw defect log

This log contains defects and deferred engineering work with reliable repository
evidence. It is intentionally not an invented inventory of every issue ever
found. The operating method has surfaced and helped resolve dozens of defects;
only reconstructable examples belong here.

## Open / deferred

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
- **Status:** OPEN.

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
