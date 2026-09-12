# OPERATING RECORD — PC-DEF-024 dead analytics deep-link removal

RECONSTRUCTED OPERATING RECORD. Evidence-based closeout, not the original prompt.

## Status and boundary

- **Status:** **RESOLVED.**
- **Working branch:** `feature/final-release-hardening`.
- **Starting commit:** `85bff2ff39996298e63ece3a9cb182d8d742888f` (PC-DEF-022 closeout).
- **Version/baseline:** `0.2.0+62`; accepted physical baseline vc62 /
  `lastAcceptedVersionCode=62`, untouched.
- **Core fingerprint:** `6f00359dc9e8bf7ee24f9d170754b2792a41fb880d9da4f34a8600dd8f99df00`
  before and after. Core was **not** rebuilt and the staged pair is untouched.

Android product source and tests only. `PC-DEF-023` is untouched — no OAuth
credential was replaced, decoded, re-encoded, documented as accepted, or its
provider removed; that needs the owner's product decision. `PC-DEF-006` and
`PC-DEF-012` are unchanged. Public Mode, the AAB policy, the resolved updater
route, Core native exports and the signing architecture were not altered. No
production signing, no production candidate, no device, no ADB, no merge, tag,
publication or release.

## The surface, proved before removal

Seven points, established against the tree and the last built artifact:

1. **Merged manifest contained it.** `android:scheme="um.placeholder"`, one
   occurrence.
2. **The filter was** `VIEW` + `DEFAULT` + `BROWSABLE` — the second intent
   filter on `MainActivity`, alongside the legitimate `MAIN`/`LAUNCHER` one.
3. **`MainActivity` is already exported** (`android:exported="true"`) by the
   normal launcher contract, so the filter added web-originated launch to an
   activity any app could already start.
4. **The SDK is not packaged.** No `com/umeng`, `com/uc/crashsdk` or
   `UMConfigure` in the DEX, and no `umeng` entry anywhere in the APK.
5. **`POCKETCLAW_ANALYTICS_PROVIDER` defaults to `"none"`**, which is what
   decides whether the dependency is pulled at all.
6. **No feature depended on the scheme.** Its only references were the plumbing
   itself: the Gradle `val`, the `buildConfigField`, the `manifestPlaceholders`
   entry, the manifest `<data>` element and the logging branch.
7. **The logging branch only logged.** `logIncomingIntent` returned early unless
   the scheme matched, then wrote the URI to logcat. It fed no navigation and no
   application state. Beside it in `onNewIntent`, `setIntent(intent)` is real
   intent handling that FlutterActivity and plugins depend on.

## Decision: removal, not a conditional

The repository had already decided this shape for the same integration. The
manifest's advertising-permission comment records it:

> An analytics build gets whatever the analytics SDK's own AAR manifest
> declares. If that SDK ever genuinely needs an app-level declaration, add it to
> the analytics build's own manifest, not to this one.

A filter that is only correct for a build PocketClaw does not ship should not
sit in the manifest it does. Making it conditional would have added complexity
to preserve dead code, so the filter is removed and the reasoning is recorded in
its place.

## What changed

    android/app/src/main/AndroidManifest.xml   VIEW/DEFAULT/BROWSABLE filter removed,
                                               replaced by the reasoning above
    android/app/build.gradle.kts               umengLinkScheme val, its
                                               POCKETCLAW_UMENG_LINK_SCHEME
                                               buildConfigField and its
                                               manifestPlaceholders entry removed
    .../kotlin/.../MainActivity.kt             logIncomingIntent and both call
                                               sites removed, with the TAG
                                               constant and android.util.Log
                                               import that existed only for it

`onCreate` was left as an override that only called `super`, so it and the
then-unused `Bundle` import went with it. **`setIntent(intent)` stays** in
`onNewIntent`: removing analytics logging must not remove real intent handling.

**Kept deliberately:** `POCKETCLAW_UMENG_APP_KEY`, `_CHANNEL` and `_PACKAGED`.
They are consumed by `AnalyticsReporter.kt` and the two `meta-data` entries, so
they are live plumbing rather than residue — and unlike the deep link they are
not an exported surface. Removing them would have been scope the defect did not
ask for.

A second call site of `logIncomingIntent` in `onCreate` was found only because
the post-edit reference check looked for residual symbols rather than assuming
one call site; deleting the function while leaving that call would not have
compiled.

## Evidence from the packaged artifact

Fresh LOCAL TEST / NON-RELEASABLE APK, built through the canonical hardened
path. **This is audit evidence, not the production candidate.**

    path    build/app/outputs/apk/release/app-release.apk
    bytes   63,500,459
    sha256  f580cadc0ed0ba7ba243d374e5e1ca92db4d27f50a6e7a92524350182bbb175f

Its packaged merged manifest, read with `aapt2 dump xmltree`:

    um.placeholder        0
    BROWSABLE             0
    android:scheme        0
    action.VIEW           0
    category.LAUNCHER     1      (still launchable)
    .MainActivity         present
    android:debuggable    0
    android:testOnly      0

`artifact.package` `com.lord1egypt.pocketclaw`, `artifact.version` `0.2.0+62`,
`artifact.permissions` 13 as expected, `artifact.backup_exclusions` intact and
`artifact.native_hardening` clean — the surrounding hardening is unchanged.

## Tests

Seven new tests in `test/unit/android_release_contract_test.dart`:

1. the source manifest declares no `um.placeholder`, no
   `${POCKETCLAW_UMENG_LINK_SCHEME}`, and **no `android:scheme=` at all** — that
   filter declared the only one, so a new scheme is a deliberate act to review;
2. no `BROWSABLE` and no `VIEW` survives;
3. the `MAIN`/`LAUNCHER` contract, `.MainActivity` and `exported="true"` remain;
4. the Gradle link-scheme plumbing is gone, so no stale `manifestPlaceholder`
   can survive merge processing and reintroduce the scheme;
5. the analytics-only logging branch is gone while `setIntent(intent)` remains;
6. the default provider still packages no SDK;
7. the **merged** release manifest carries none of it, when one has been built.

Assertions strip XML comments and then check declarations. That is not
cosmetic: the comment documenting the removal necessarily names what was
removed, and running the merged-manifest assertion for real caught it — Gradle
carries comments through its merge, and only `aapt2` strips them when compiling
the binary manifest. The packaged artifact therefore has zero occurrences while
the intermediate XML still shows the prose.

**Mutation-tested:** reintroducing the filter into the manifest fails the
contract.

## Regression

    flutter analyze                          No issues found
    flutter test                             497 passed, 0 failed  (was 490; +7 new)
    android release contract tests            29 passed
    source gate, test class                   26 items, 25 PASS / 1 SKIPPED
                                              (repo.clean_worktree, work uncommitted)
    full artifact gate (LOCAL TEST APK,
      artifact-class public-release)          57 items, 56 PASS / 1 SKIPPED (same)
    flutter.suite inside the gate            497 passed, 0 failed

Final clean-tree totals are recorded in the owner report.

The private native support manifest was rebound to this fresh audit APK
(`f580cadc…`) through the established local-test process, so the artifact gate
ran against a correctly bound manifest. It is a **local-test** binding on a
**non-releasable** artifact and is not production evidence.

## Closeout conditions

1. `um.placeholder` absent from the default merged release manifest — zero in
   the packaged artifact.
2. The dead Umeng `BROWSABLE` filter is absent.
3. Normal `MAIN`/`LAUNCHER` behaviour remains, asserted in the same merged file
   so the check cannot pass on an empty manifest.
4. No shipping feature depended on the removed path — proved before removal.
5. The analytics-only URI logging is removed; the shared `setIntent` handling is
   preserved and asserted.
6. Seven tests guard against regression, and are mutation-tested.
7. Full relevant suite green.
8. Core fingerprint unchanged; no Core rebuild.
