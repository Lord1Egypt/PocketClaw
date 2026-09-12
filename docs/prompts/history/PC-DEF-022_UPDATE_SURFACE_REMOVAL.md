# OPERATING RECORD — PC-DEF-022 update-surface removal

RECONSTRUCTED OPERATING RECORD. Evidence-based closeout, not the original prompt.

## Status and boundary

- **Status:** **RESOLVED.**
- **Working branch:** `feature/final-release-hardening`.
- **Starting commit:** `7dd93b7e6584698b1132a25902391c0bab976016` (exposure-audit closure).
- **Source / build-input commit:** `a0be2a705c1b255c5bd2fe1d8c9f44c094019627`.
- **Staging / closeout commit:** the following commit carrying the rebuilt Core
  pair and this record, touching no Core build input.
- **Version/baseline:** `0.2.0+62`; accepted physical baseline vc62 /
  `lastAcceptedVersionCode=62`, untouched.

`PC-DEF-023`, `PC-DEF-024`, `PC-DEF-006` and `PC-DEF-012` are untouched. OAuth
provider behaviour, the Umeng/deep-link manifest, native export surfaces, the AAB
release policy and Public Mode semantics were not modified. No production signing
material was requested, no production candidate was built, no device or ADB was
used, and nothing was merged, tagged, published or released.

## The surface, re-proved before removal

Not taken from the audit report. All six points re-established against the tree:

1. **Route.** `mux.HandleFunc("/api/update", h.handleUpdate)` at
   `core/src/web/backend/api/update.go:12`, registered unconditionally from
   `router.go`.
2. **Auth.** Absent from `isPublicLauncherDashboardPath`, so the dashboard
   session wall applied — an authenticated surface, not a public one.
3. **Flutter callers.** Zero.
4. **Dashboard/frontend callers.** Zero. Kotlin: zero. A search across Dart,
   Kotlin, TypeScript and TSX found **zero** callers anywhere outside the
   backend package.
5. **Updater package.** Inherited upstream code with two consumers:
   `cmd/picoclaw/main.go:142` (`updater.NewUpdateCommand`, the CLI subcommand)
   and this HTTP route.
6. **Android apply.** `selfupdate.Apply` targets `os.Executable()`, which on
   Android is inside the read-only install directory, so the replace step could
   not succeed.

What remained was therefore an authenticated arbitrary-URL fetch and
archive-extraction primitive on a route the product never uses.

## The decision

Securing an unused self-update subsystem would have been the wrong repair. The
exposed route is removed.

    core/src/web/backend/api/update.go     deleted (52 lines: route, types, handler)
    core/src/web/backend/api/router.go     registerUpdateRoutes call removed

**`pkg/updater` is kept.** It has a live non-HTTP consumer in the CLI update
command, and its archive-traversal guards and tests are untouched — the prompt's
acceptance criterion was removal of the exposed product route, not maximum
source deletion, and deleting a library with a real consumer to tidy up would
have been a different and riskier change.

## HTTP behaviour after the fix

No special response was invented. `web/backend/embed.go:50` already answers an
unknown `/api/` path with `http.NotFound` rather than falling back to the SPA
entry, so `POST /api/update` is now an ordinary **404** — the router's normal
answer for an unregistered API route, with no information-leaking envelope and
no misleading fake success.

## Tests

`core/src/web/backend/api/no_update_route_test.go`:

- the route resolves to **no registered pattern** under POST, GET, PUT, PATCH
  and DELETE;
- an authenticated request driven through the routed mux — past any auth wall,
  which is the threat model this route actually had — returns 404 and carries
  none of this package's JSON envelope, so no handler ran;
- six plausible renames are checked (`/api/updates`, `/api/self-update`,
  `/api/selfupdate`, `/api/system/update`, `/api/upgrade`, `/api/download`) so an
  arbitrary-download surface cannot reappear under another name;
- the auth middleware is asserted not to name `/api/update`, so the removal
  cannot have widened the unauthenticated surface;
- `update.go`'s absence and `router.go`'s missing call are both asserted, so a
  partial revert cannot restore the handler as dead code that still reads as
  live product.

**Mutation-tested:** restoring `update.go` and its registration fails them.

The shipped binaries confirm it independently: `/api/update` occurs **zero**
times in either `libpocketclaw.so` or `libpocketclaw-web.so`, and the web binary
shrank by **132,864 bytes** as the linker dropped the now-unreachable paths.

`pkg/updater`'s own tests still pass and both `path traversal detected` guards
remain in the library.

## No network or auth regression

`web/backend`, `web/backend/api`, `web/backend/middleware`,
`web/backend/dashboardauth`, `web/backend/launcherconfig`, `pkg/netbind`,
`pkg/gateway` and `pkg/updater` all pass.

Public Mode OFF remains loopback-only and ON remains explicit LAN exposure —
their resolution and the bind plan were not touched. The dashboard
password/session wall, the WebSocket session-plus-origin boundary and the Core
gateway's unconditional loopback pin are unchanged, and the normal dashboard
APIs are unaffected: only one route was removed from the mux.

## Core rebuild and re-stage

`core/src` changed, so the two-commit rule applied.

    source fingerprint   2692de41b2fe2487475911b62cec519193d581b25cf6d0ebe935fc63973229df
                      →  6f00359dc9e8bf7ee24f9d170754b2792a41fb880d9da4f34a8600dd8f99df00
    build-input commit   a0be2a705c1b255c5bd2fe1d8c9f44c094019627
    BuildTime            2026-09-12T23:25:34+0000

    libpocketclaw.so       37,724,640  7ebeebd192577110bdfd2ee7cb8ddecc9b26dafeb7fe08d0d9f29f0a0f337c85
                                       build ID 512ed36a4ae1b2f0c37ed250102ea7da9ae01d1b
    libpocketclaw-web.so   25,385,088  b682b76dc27811219b4793f60034613521550cd603916fba7a0e51e5ff9950ec
                                       build ID 4eeb13853c4120befad82e36c49eac5a05471278

`libpocketclaw-web.so` is 132,864 bytes smaller than the pair it replaces, which
is the removed surface leaving the binary. `libpocketclaw.so` keeps its size —
the CLI updater consumer is still linked there, as intended.

Produced byte-identically in the canonical in-repository run and two further
independent output roots under different `CORE_BUILD_DIR`, `JNI_LIBS` and
`NATIVE_SYMBOL_ROOT` paths, the third with a cold `GOCACHE`. Both private
companions likewise:

    libpocketclaw.so.debug       14,545,360  6794d0123a1250386efc121421c328ee726bb4aa0aaa5e3a9b64fe2df3d7704a
    libpocketclaw-web.so.debug    9,692,184  2ee29a0bbfa96ff14de52ea93aa0bb1888f68f1dc9677b7646691fda1926fbf6

Both rebound to the new shipped hashes, sizes and build IDs; the archive stays
ignored and untracked, companions 0755 and the manifest 0600. **No Managed
Runtime payload was rebuilt** — `git diff` over `jniLibs/` shows exactly the two
Core files, and the eight runtime companions still carry their H5B mtimes.

**Stated honestly:** the private support manifest's `apk` field still names the
exposure-audit APK `113a8382…`, which no longer contains this Core pair. That is
the established pre-artifact state — `bind-apk` is a separate step against a
candidate artifact and none was built here — and it is rebound at the next
artifact build.

## Gates and tests

    Core Go suite (go test ./...)              98 packages ok, 0 failed
    pkg/coresource (freshness + reproducibility) 46 tests, 0 failed
    TestStagedCoreWasBuiltFromTheCurrentSource   PASS
    TestBundledPayloadsMatchTheirPinnedChecksums PASS
    TestStagedCoreEmbedsTheCurrentCatalog        PASS
    Native contract, Core pair                 22 PASS / 0 FAIL
    flutter analyze                            No issues found
    flutter test                              490 passed, 0 failed
    source gate, test class                    26 PASS / 0 FAIL / 0 SKIPPED
    source gate, production class              26 PASS / 0 FAIL / 0 SKIPPED
    release-tool suites                       139 tests across 8 files, all OK

The frontend suite has no structural coverage of this route — the dashboard
never called it, which is part of why it was removable — so it is reported
unchanged rather than claimed as evidence.

## Closeout conditions

1. `POST /api/update` is no longer a shipped PocketClaw API surface — removed
   from source and absent from both binaries.
2. Authenticated users cannot trigger an arbitrary-URL updater fetch or extract
   through the dashboard API — proved through the routed mux, past the auth wall.
3. Normal dashboard APIs remain functional — only one route left the mux, and
   every API suite passes.
4. Public Mode, auth, WebSocket and gateway boundaries unchanged.
5. Tests prove route removal, and are mutation-tested.
6. Core rebuilt and re-staged; fingerprint moved and is recorded.
7. All relevant suites and gates green.
