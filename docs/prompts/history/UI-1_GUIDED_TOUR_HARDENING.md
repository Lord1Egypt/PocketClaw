# OPERATING RECORD — UI-1 guided tour hardening

An isolated UI defect repair, authorized between H5C and the final release
exposure audit. It does not advance the release roadmap.

## Status and boundary

- **Status:** CLOSED.
- **Working branch:** `feature/final-release-hardening`.
- **Starting commit:** `f5d027dc1ca1500c75337453a0cf72be1142a746` (H5C closeout).
- **Ending commit:** the commit containing this record.
- **Version/baseline:** `0.2.0+62`; the accepted physical baseline remains
  vc62 / 62, untouched.
- **Next release milestone:** unchanged — final release exposure audit.

No native work, no signing material, no Core or Managed Runtime rebuild, no
APK/AAB build, no device access, no version bump, no merge, tag or publish.

## What was repaired

Six defects the read-only audit proved, recorded as `PC-DEF-013` through
`PC-DEF-018` and all RESOLVED. The full root causes are in
[`DEFECT_LOG.md`](../../DEFECT_LOG.md); the short form:

| Defect | Repair |
| --- | --- |
| `PC-DEF-013` RTL placement | logical `start`/`end`/`above`/`below` resolved against document direction, plus an unconditional viewport clamp |
| `PC-DEF-014` focus left on the target | the spotlight blocks its target; focus is captured on open and restored on every termination path |
| `PC-DEF-015` dead `docs-button` step | step removed, `tour.docs.*` removed from all 14 locales, structural test added |
| `PC-DEF-016` no re-measure | eligibility check, bounded rAF resolution, ResizeObserver + scroll + resize + MutationObserver, scoped `scrollIntoView` |
| `PC-DEF-017` interaction trap | dimmer dismisses, Escape dismisses, card always reachable |
| `PC-DEF-018` no state version | `{version, currentStep, isActive}` with a migration that preserves completion |

`PC-DEF-019` is opened, not resolved: see below.

## The correction the audit forced

The symptom was reported as a teardown leak — the Models item still highlighted
after the tour closed. It was not. The audit proved every tour node unmounts on
every close path, and that the tour never writes a class, attribute or inline
style onto its target. The real chain was: a `pointer-events-none` spotlight let
the click reach the real nav link, the app navigated, the anchor kept DOM focus,
and `outline-hidden focus-visible:ring-2` on `SidebarMenuButton` painted a ring
that outlived the tour and looked exactly like the spotlight's own.

Fixing this as a teardown bug would have changed nothing. It is fixed as
click-through and focus ownership.

## Chosen interaction policy

The spotlight **captures and swallows** clicks on its target; the tour's own
buttons own progression. The dimmer around it dismisses the tour, as does
Escape. That combination is what makes the stuck ring structurally impossible:
the tour can no longer cause a navigation or move focus onto an application
control.

The `models` step's copy still reads "Click the 'Models' menu on the left …".
It now describes where the control is rather than an action the tour expects,
and it carries a physical direction in 14 translations. Rewording it is product
copy work and was left out of a defect repair; it is noted here so the decision
is visible rather than silent.

## Files changed

| File | Change |
| --- | --- |
| `core/src/web/frontend/src/components/tour/tour-guide.tsx` | rewritten: logical placement, clamping, eligibility, bounded resolution, live re-measure, focus ownership, dismissal |
| `core/src/web/frontend/src/components/tour/tour-steps.ts` | new: step table and target helpers, kept out of the component file so fast refresh stays clean and the table is testable |
| `core/src/web/frontend/src/store/tour.ts` | versioned state and migration; storage degrades to memory when `localStorage` is unavailable |
| `core/src/web/frontend/src/components/tour/tour-guide.test.tsx` | new: 27 regression tests |
| `core/src/web/frontend/src/i18n/locales/*.json` (14) | `tour.docs.*` removed |
| `core/src/web/frontend/src/i18n/i18n.test.ts` | assertion re-pointed from the removed step to `tour.gateway.title` |

No other file changed. No native binary, signing input, Core payload or
Managed Runtime payload was touched.

## Verification

**Automated.** Full frontend suite **418 tests in 25 files, all passing**,
including the 27 new tour tests and the 138 tests in the four suites nearest
this change (`header-responsive`, `sidebar-direction`, and the two i18n suites).
`tsc --noEmit` clean, `eslint` clean on the changed files, `prettier --check`
clean on them.

**Real browsers.** Headless Chrome 153 and Brave 153 (Chromium), driven over
the DevTools protocol at 1440x900, against the identical source served by vite.
A throwaway profile under `C:\temp` was used in each; no browser-wide setting
was changed and no cache was cleared as a fix. Both browsers, both directions:

| Direction | Step 1 | Step 2 (anchored) | Step 3 (anchored) |
| --- | --- | --- | --- |
| LTR | `560..880` | `259..579` | `809..1129` |
| RTL | `560..880` | `861..1181` | `301..621` |

Every card inside the 1440-wide viewport; the primary control hit-testable at
its own centre at every step; after close `0` tour nodes, `activeElement` `BODY`,
no focused `[data-tour]` element; Escape closes and records completion; a click
on the spotlight leaves the path at `/` and the tour open.

The RTL fix was measured against the defect rather than asserted. With the live
Arabic layout the Models target sits at `left 1193 / right 1432`; the pre-fix
formula would have placed the card at `left 1444`, right edge `1764`, against a
1440 viewport — off-screen, exactly the reported hang. It now renders at
`861..1181`.

Persistence, in-browser: a stored completed state renders zero tour nodes, and
legacy unversioned `{currentStep:"docs", isActive:true}` also renders zero
rather than stranding.

The embedded/mobile layout, breakpoint transitions and drawer states are covered
by the jsdom suite; no device was touched.

## PC-DEF-019 — the consequence this repair could not resolve

`libpocketclaw-web.so` embeds the compiled dashboard, so everything under
`core/src/web/frontend` is a Core build input. This repair therefore moved the
Core source fingerprint:

    before  86369a32a9873715672f7867b31dcd72a7d19088c49cdb1df2b584c548ba4c73
    after   bd4a8629a2682e2f05aa3859a400be8a77fb4954ad14994e5703ccbe365d05ec

The staged Core pair now predates its source. The test-class source gate reports
**22 PASS / 2 FAIL / 1 SKIPPED**, the two failures being `core.staged_freshness`
and `build.reproducibility_tests`, both from that one cause. Every other gate
passes.

This repair had no authority to rebuild Core, so the staleness is recorded
rather than papered over. The H5C production artifact and all its evidence
remain historically valid; they describe the previous dashboard. **Core must be
rebuilt and re-staged before the next artifact build**, under a milestone that
carries that authority, following the two-commit rule H5B established: the
source commit sets the canonical build time, and the staged pair lands in a
following commit that touches no build input.

## Not done, deliberately

- No tour dependency was introduced.
- No export, breakpoint, sidebar or header behaviour was changed.
- Route-active styling on the Models item is correct and was left alone.
- The `models` step copy was not reworded.
- No account or cloud synchronization of tour state; WebView and browser origins
  stay independent, which is browser behaviour, not something to work around.
