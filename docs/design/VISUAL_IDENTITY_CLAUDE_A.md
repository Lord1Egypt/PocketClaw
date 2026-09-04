# PocketClaw Visual Identity — Concept A

**Status:** proposal for design review. Nothing here is implemented.
**Branch:** `design/claude-concept-a`, from `develop` at `9877365`.
**Scope:** visual direction only. No functional change is proposed anywhere in
this document except three small, explicitly-flagged UI-structure fixes in §17
and §20.

---

## 1. Design concept name

# APERTURE

*The pocket that opens.*

The name is the working title for the system, not a user-facing string. It
comes from the one gesture the whole identity is built on: a closed container
that opens to let something out. That is what PocketClaw is — a private machine
you carry, that opens and acts.

---

## 2. Design philosophy

### What is actually wrong today

This is not a matter of taste. Four findings from the current source explain
precisely why the product reads as a generic open-source frontend.

**Finding 1 — the web dashboard is running unmodified shadcn defaults.**
`core/src/web/frontend/src/index.css` defines every neutral at **chroma 0**:
`--background: oklch(0.145 0 0)`, `--foreground: oklch(0.985 0 0)`,
`--card: oklch(0.205 0 0)`, and in dark mode `--primary: oklch(0.922 0 0)` —
a near-white primary. There is no brand hue anywhere in the token set. The five
chart colors are the stock shadcn rainbow. A visitor is not imagining the
resemblance to every other shadcn admin panel; they are looking at one.

**Finding 2 — light and dark mode come from two different design systems.**
`lib/main.dart:78-79`:

```dart
theme: PocketClawDesign.lightTheme(),                    // Material 3, seed #00B8D9, Inter
darkTheme: AppTheme.getTheme(service.currentThemeMode),  // FlexColorScheme, 1 of 6 modes
```

Light mode is generated from a cyan seed. Dark mode is one of six
FlexColorScheme themes (`carbon`, `slate`, `obsidian`, `ebony`, `nord`,
`sakura`) whose accents are neon cyan, amber, pure white, gold, frost blue and
pink. Nothing connects the light identity to the dark one. Toggling the theme
does not adjust PocketClaw; it swaps products.

**Finding 3 — there are two unrelated brand marks.**
`core/src/web/frontend/public/logo_with_text.png` is the PicoClaw lobster
mascot beside a **PICOCLAW** wordmark in navy and coral. Every favicon in
`public/` is the same lobster. Meanwhile `assets/branding/pocketclaw-mark.png`
is an entirely different thing: a 3D glossy blue claw rising from an open box.
The web says PicoClaw, Android says something else, and neither is a system.

**Finding 4 — the marks that do exist cannot function as marks.**
`pocketclaw-mark.png` is a 654 KB isometric render with gradients, specular
highlights and a glow. It has no flat form, no single-colour form, and no
legible 16 px form. It cannot be a favicon, a monochrome status glyph, an
`AppBar` leading icon, or a wordmark lockup.

### The three principles that follow

**1. Instrument, not dashboard.**
Competing agent products are cloud consoles: wide gradients, marketing
illustration, floating cards on empty black. PocketClaw's actual proposition is
the opposite — the agent runs *on your device*, under your key, in your
pocket. So it should read like a precision instrument you own: dense where
density earns its place, quiet everywhere else, every value legible, nothing
decorative that does not carry information.

**2. One accent, spent deliberately.**
A premium interface is recognised by its restraint. Aperture allows exactly one
brand accent for interactive intent and one signal colour for live machine
state. Everything else is neutral. A screen where four things are coloured is a
screen where nothing is emphasised.

**3. Neutrals carry the brand.**
The single highest-leverage change in this entire document is one line of
arithmetic: **give every grey a small chroma at the brand hue.** Chroma-0 grey
is the default of every framework on earth, which is exactly why it reads as
default. Tinted neutrals are what separates a designed dark mode from a
generated one, and they cost nothing — no new asset, no new dependency, no
layout change.

---

## 3. Visual personality

| Aperture is | Aperture is not |
| --- | --- |
| Machined, deliberate, quiet | Loud, playful, mascot-led |
| Dark-first, with a real light mode | Dark-only, or a light theme inverted badly |
| Dense where it informs | Dense everywhere |
| Confident in flat colour and hairlines | Dependent on gradients, glass or glow |
| Monospace for machine values | Monospace as decoration |
| Still until something happens | Animated for its own sake |

If Aperture had a physical referent it would be a field instrument: an
anodised aluminium case, a high-contrast readout, one illuminated indicator
that means the device is live, and a switch that gives a definite click.

Explicitly rejected, per brief: F-Droid utilitarianism, stock shadcn, empty
black voids, gradient washes, glassmorphism, RGB/gaming accents, crypto-console
density, and any control below the touch-target minimum.

---

## 4. Colour and token system

### 4.1 The hue spine

Three hues, and no others in the core system.

| Role | Hue (OKLCH) | Meaning |
| --- | --- | --- |
| **Graphite** | 245 | every neutral: canvas, surfaces, borders, text |
| **Claw** | 208 | brand accent — interactive intent, selection, focus |
| **Signal** | 195 | *live machine state only* — gateway running, agent streaming |

Claw at hue 208 is a deliberate continuation, not a reset: `PocketClawDesign`
already seeds Material 3 from `#00B8D9`, which lands near this hue. Aperture
keeps that lineage and disciplines it — the cyan stops being a general-purpose
colour and becomes the accent, while a distinct, brighter Signal cyan is
reserved so that "this thing is live right now" is never confused with "this
thing is clickable".

Graphite at hue 245 is the load-bearing decision. It is a blue-leaning
neutral, and its chroma **rises as it darkens**, which is how physical dark
materials behave and why the result reads as considered rather than generated.

### 4.2 Dark palette (the primary target)

```css
/* Canvas and surfaces — note: never pure black, never chroma 0 */
--pc-bg:            oklch(0.17 0.012 245);  /* app canvas                */
--pc-surface-1:     oklch(0.21 0.014 245);  /* cards, sidebar            */
--pc-surface-2:     oklch(0.25 0.016 245);  /* popovers, sheets, raised  */
--pc-surface-3:     oklch(0.29 0.018 245);  /* inputs, hover, code wells */

/* Hairlines — depth is drawn, not shadowed */
--pc-border:        oklch(0.33 0.020 245);
--pc-border-strong: oklch(0.44 0.026 245);

/* Text */
--pc-text:          oklch(0.97 0.004 245);  /* primary   ~15:1 on bg     */
--pc-text-muted:    oklch(0.74 0.014 245);  /* secondary  ~7:1 on bg     */
--pc-text-faint:    oklch(0.60 0.016 245);  /* tertiary   ~4.6:1 on bg   */

/* Brand accent */
--pc-primary:       oklch(0.70 0.130 208);
--pc-primary-hover: oklch(0.76 0.130 208);
--pc-primary-fg:    oklch(0.16 0.020 245);  /* dark ink ON the accent    */
--pc-primary-soft:  oklch(0.70 0.130 208 / 0.14);
--pc-primary-line:  oklch(0.70 0.130 208 / 0.40);

/* Machine state */
--pc-signal:        oklch(0.78 0.130 195);  /* LIVE — running, streaming */
--pc-success:       oklch(0.72 0.150 158);
--pc-warning:       oklch(0.80 0.140 78);
--pc-danger:        oklch(0.64 0.200 25);
/* each status also has a -soft at 14% and a -line at 40% */
```

Two rules that make this a system rather than a list:

- **`--pc-primary-fg` is dark ink, not white.** The accent sits at L 0.70, so
  white text on it fails contrast. Dark ink on a bright accent is also what
  makes a primary button read as *illuminated* rather than *painted*.
- **`--pc-signal` may only ever be applied to machine state** — gateway
  running, a response streaming, a tool executing. It is never a button, never
  a link, never a heading. This is the rule that keeps the interface from
  drifting into a gaming aesthetic while still letting a live agent glow.

### 4.3 Light palette

The same three hues, chroma reduced roughly 30%, lightness inverted around the
scale. Light mode is a first-class target, not an inversion.

```css
--pc-bg:            oklch(0.985 0.003 245);
--pc-surface-1:     oklch(1.000 0     0  );  /* cards lift by being lighter */
--pc-surface-2:     oklch(0.975 0.004 245);
--pc-surface-3:     oklch(0.955 0.006 245);
--pc-border:        oklch(0.905 0.008 245);
--pc-border-strong: oklch(0.820 0.012 245);
--pc-text:          oklch(0.22  0.015 245);
--pc-text-muted:    oklch(0.46  0.018 245);
--pc-text-faint:    oklch(0.58  0.016 245);
--pc-primary:       oklch(0.55  0.135 208);  /* darkened for contrast on white */
--pc-primary-fg:    oklch(0.99  0.005 208);  /* light ink ON the accent        */
--pc-signal:        oklch(0.60  0.130 195);
```

Note the inversion of the ink rule: in light mode the accent is dark enough
that white ink is correct. `--pc-primary-fg` is a token precisely so neither
mode has to think about it.

### 4.4 Semantic layer

Product code never reaches for a raw ramp value. It uses semantic aliases, so
a future re-theme is a token edit rather than a search across the codebase:

```
--pc-nav-active-bg / --pc-nav-active-marker / --pc-nav-hover-bg
--pc-field-bg / --pc-field-border / --pc-field-border-focus
--pc-msg-user-bg / --pc-msg-assistant-bg / --pc-msg-tool-bg
--pc-badge-neutral / --pc-badge-live / --pc-badge-warn / --pc-badge-error
```

Existing shadcn variables (`--background`, `--card`, `--primary`, …) are
**remapped onto these**, not deleted. Every shadcn component keeps working
untouched; it simply stops being grey. This is what makes the change tractable.

### 4.5 Chart / data colours

The stock shadcn `--chart-1..5` rainbow is replaced with a sequence that shares
the hue spine: Claw 208 → Signal 195 → Teal 172 → Violet 285 → Amber 78. Ordered
so the first two — the ones actually used most — are on-brand.

---

## 5. Typography

**No paid fonts, no proprietary assets, no new runtime dependency.**

| Role | Family | Source |
| --- | --- | --- |
| UI | **Inter Variable** | already a dependency in both surfaces |
| Machine values | **system monospace stack** | zero cost, no download |

Inter is already installed on both sides — `@fontsource-variable/inter` in the
web frontend, `google_fonts` + `GoogleFonts.interTextTheme()` in Flutter. There
is nothing to add.

```css
--pc-font-ui:   "Inter Variable", Inter, system-ui, sans-serif;
--pc-font-mono: ui-monospace, "SF Mono", "JetBrains Mono", "Cascadia Mono",
                Menlo, Consolas, monospace;
```

*Optional later upgrade:* JetBrains Mono Variable (OFL, free) for a more
distinctive readout. Not required, and deliberately not a dependency of this
concept.

### The mono rule

Monospace is not decoration. It is applied to exactly one category: **values a
machine produced and a human may need to compare character by character.**

Use mono for: version strings, SHA-256 digests, ports, host addresses, file
paths, model identifiers, token counts, latency, log lines, code.
Never for: headings, labels, body copy, buttons, empty-state prose.

This single rule does more for the "technically sophisticated" character than
any amount of styling, because it makes the interface *look like it knows what
kind of data it is holding*.

### Scale

A 1.20 ratio, capped at seven sizes. Optical sizes are set in `rem` so OS text
scaling continues to work.

| Token | Size | Line | Weight | Use |
| --- | --- | --- | --- | --- |
| `display` | 1.75rem | 1.25 | 600 | release header, onboarding title |
| `title` | 1.375rem | 1.3 | 600 | page title |
| `heading` | 1.125rem | 1.4 | 600 | card / section heading |
| `body` | 0.9375rem | 1.55 | 400 | default |
| `label` | 0.875rem | 1.4 | 500 | form labels, nav, buttons |
| `caption` | 0.8125rem | 1.4 | 400 | helper text, timestamps |
| `micro` | 0.6875rem | 1.2 | 600, +0.06em tracking | badges, status chips |

Tracking is negative on the two largest sizes (`-0.02em`, `-0.01em`) and
positive only on `micro`. Uppercase is used **only** at `micro`.

---

## 6. Spacing, radius and depth

### Spacing — 4 px base

`2, 4, 8, 12, 16, 20, 24, 32, 40, 48, 64`

Two composition rules that are worth more than the scale itself:

- **Related things touch (8), unrelated things separate (24).** Most "generic"
  interfaces use 16 for everything, which flattens hierarchy into mush.
- **A card's internal padding is 16 on mobile, 20 on desktop.** Never less than
  16 — cramped padding is the single clearest tell of a developer-tool UI.

### Radius

```
--pc-radius-xs:   6px   /* badges, chips, micro-controls */
--pc-radius-sm:  10px   /* buttons, inputs, list rows    */
--pc-radius-md:  14px   /* cards, popovers               */
--pc-radius-lg:  20px   /* sheets, dialogs, message bubbles */
--pc-radius-full: 999px /* avatars, status dots, pills   */
```

The current values (`--radius: 0.625rem` web, `radiusSmall 10 / radiusMedium
16` Flutter) are already close. Aperture aligns them exactly so a card in the
Flutter shell and a card in the embedded WebView have the same corner. That
alignment is a large part of what will make the two surfaces read as one
product — the WebView sits *inside* the Flutter app, and today the corners
disagree.

### Depth — drawn, not shadowed

Dark-mode drop shadows are close to invisible; leaning on them is why dark
interfaces so often look flat. Aperture builds elevation from three stacked
devices instead:

1. **A lightness step.** Each level moves up one surface token.
2. **A hairline.** 1px `--pc-border`, with a slightly brighter top edge
   (`inset 0 1px 0 oklch(1 0 0 / 0.04)`) to suggest a lit upper bevel.
3. **A shadow, only above level 2.** Reserved for genuinely floating things —
   popovers, sheets, dialogs — where a real occlusion cue is warranted.

| Level | Use | Recipe |
| --- | --- | --- |
| 0 | canvas | `--pc-bg` |
| 1 | card, sidebar | `surface-1` + hairline |
| 2 | input, hover, code well | `surface-3` + hairline |
| 3 | popover, menu | `surface-2` + hairline + `0 8px 24px oklch(0 0 0 / 0.32)` |
| 4 | dialog, sheet | `surface-2` + hairline + `0 24px 64px oklch(0 0 0 / 0.44)` |

In light mode, levels 1–2 invert: cards get *lighter* than the canvas, and the
hairline does most of the separation.

---

## 7. Navigation model

PocketClaw has three navigation contexts and they should not pretend to be the
same shape.

| Context | Model | Rationale |
| --- | --- | --- |
| Flutter shell (phone) | 4-item bottom bar | thumb reach; already the IA |
| Flutter shell (tablet/desktop) | left rail, icon + label | horizontal space exists |
| Web dashboard (desktop) | persistent left sidebar, grouped | many destinations |
| Web dashboard (mobile) | off-canvas drawer behind a **menu** button | narrow WebView |

### The active-state device

One idea, used everywhere, in both technologies:

> **The active destination is marked by a 2px bracket on its inline-start
> edge, in `--pc-primary`, with a `--pc-primary-soft` row fill.**

Not a filled pill, not a bold weight change, not a background block. A bracket.
It is quiet, it is unmistakable, it scales from a 32px sidebar row to a 64px
bottom-bar item, and — because it is expressed with `border-inline-start` in
CSS and `EdgeInsetsDirectional` / `BorderDirectional` in Flutter — **it mirrors
in RTL for free**, with no locale branch anywhere.

The bracket is the same shape as the mark's opening (§15). That is the thread
that ties navigation, logo, focus ring and empty states into one language.

---

## 8. Flutter application design

### 8.1 The theme architecture problem, and the fix

Today (`lib/main.dart:78-79`) light mode comes from `PocketClawDesign` and dark
mode from `AppTheme.getTheme()`. These are two unrelated systems and it shows.

**Proposal:** a single `PocketClawTheme` that emits both brightnesses from the
Aperture tokens, with the existing six-mode selector **kept as a feature but
re-based**. The six modes stop being six different products and become six
*accent* choices over one structural system:

| Existing mode | Today | Under Aperture |
| --- | --- | --- |
| `carbon` | #111 + neon cyan | Graphite structure, **Claw 208** accent (default) |
| `slate` | Slate 950 + amber | Graphite structure, Amber 78 accent |
| `obsidian` | pure black + white | **True-black canvas variant**, Claw accent (OLED) |
| `ebony` | warm grey + gold | Warm-graphite structure (hue 60), Gold accent |
| `nord` | Polar Night + frost | Graphite structure, Frost 220 accent |
| `sakura` | pink, light-only | Light mode, Rose 350 accent |

No functionality is removed — the user keeps every choice they have today —
but the app is recognisably PocketClaw in all six. Structure, spacing, radii,
elevation, typography and the bracket motif are constant; only the accent hue
and canvas depth vary.

`obsidian` is worth keeping as a true-black variant specifically because AMOLED
phones benefit from it. It is the one sanctioned exception to "never pure
black".

### 8.2 Main shell

- `AdaptiveActionBar` keeps its adaptive rail/bar behaviour unchanged.
- Bottom bar: height 64 + safe area, `surface-1` with a top hairline. Inactive
  icons `--pc-text-faint`, active icon + label `--pc-primary` with a 2px
  bracket **above** the item (the bottom-bar rotation of the same motif).
- Icons stay Material outlined/filled pairs, already the pattern in
  `_buildNavButton`. Outlined = inactive, filled = active. No icon changes.
- `PageTransitionSwitcher` with `SharedAxisTransition` is retained; the vertical
  axis is correct for a bottom bar and is already RTL-safe.

> **Recommendation (small, user-visible, out of visual scope but found here):**
> the four nav tooltips in `lib/main.dart` are hardcoded English —
> `'Status'`, `'Web'`, `'Logs'`, `'Settings'`. In an app that ships twelve
> locales this is a localization defect. Flagged, not fixed here.

### 8.3 Settings

The current Settings page is a vertical stack of `Card`s with a header row.
Aperture keeps that structure exactly — it is sound — and changes only surface
treatment:

- **Header.** The `Wrap` shipped in the What's New milestone stays as-is. It is
  correct and it is what stops the title truncating. Title at `title` size,
  actions as ghost buttons with hairline borders.
- **Section grouping.** Cards gain an optional `micro`-size uppercase group
  label above them (`CONNECTION`, `AGENT`, `INTEGRATIONS`, `ABOUT`) in
  `--pc-text-faint`. This is the cheapest possible fix for a long settings
  list reading as an undifferentiated stack.
- **Card.** `surface-1`, `radius-md`, 1px hairline, zero elevation. Internal
  padding 16. Title at `heading`, description at `caption` in `--pc-text-muted`.

**Public Mode card.** The most consequential switch in the app; it should look
it. Two-line layout: title + one-line consequence in `caption`. When ON, a
`--pc-warning-soft` fill with a `--pc-warning` inline-start bracket — this is
network exposure, and warning colour is honest. When applying, the switch is
disabled and a 2px indeterminate bar sits under the card. The LAN address is
`--pc-font-mono`.

**Auto-start controls.** The existing two-line subtitle (preference state on
line 1, runtime state on line 2) is genuinely good design and stays. Aperture
only styles it: preference in `--pc-text-muted`, runtime as a status chip —
a 6px dot plus `micro` label, `--pc-signal` when running, `--pc-text-faint`
when stopped, `--pc-warning` when starting.

**Telegram settings.** Connection state as a status chip in the card header.
Connected shows the bot handle in mono. Primary action is the only filled
button on the card.

**Context Memory.** The 10/15/20/25/Custom choice becomes a segmented control:
one hairline-bordered track, the selected segment filled `--pc-primary-soft`
with `--pc-primary` label. "Recommended" is a `micro` badge on 15, never a
parenthetical in the label — that keeps the control readable in German and
Arabic where the parenthetical would wrap. Custom reveals a numeric field with
the 5–50 range shown as `caption` helper text, not as a validation error the
user has to trigger.

**GitHub.** Disconnected: hairline card, one filled Connect action.
Connected: `--pc-success` dot, `@login` in mono, Disconnect demoted to a
text button. The token itself is never rendered — unchanged, and correct.

**Models entry.** A row that opens the console. Trailing chevron uses
`Icons.chevron_right` under `Directionality`, so it mirrors in Arabic.

**What's New.** The badge becomes a `micro` pill in `--pc-primary` with
`--pc-primary-fg` ink. The page itself: release header at `display`, version in
mono, section headings at `heading` with a 2px `--pc-primary` bracket, bullets
with the existing 5px dot recoloured to `--pc-primary`. Section cards get
`--pc-success` / `--pc-primary` / `--pc-warning` bracket accents for
Fixes / New / Improvements respectively.

**About.** Stays a dialog. Version rows become a two-column definition list,
labels in `label`, values in **mono** — this is exactly the mono rule earning
its keep.

### 8.4 Controls

| Control | Treatment |
| --- | --- |
| Filled button | `--pc-primary` bg, `--pc-primary-fg` ink, `radius-sm`, height 44, weight 500 |
| Ghost button | transparent, hairline border, `--pc-text` label, hover `surface-3` |
| Text button | no border, `--pc-primary` label |
| Destructive | `--pc-danger` bg, only in confirmation dialogs |
| Input | `surface-3`, hairline, `radius-sm`, height 44; focus swaps border to `--pc-primary` and adds a 2px `--pc-primary-line` ring |
| Switch | track `surface-3` → `--pc-primary` when on; 44×44 hit area regardless of visual size |
| Card | `surface-1`, hairline, `radius-md`, elevation 0 |

### 8.5 Selected and focused states — the D-pad contract

PocketClaw supports D-pad/TV navigation through an explicit focus chain
(`FocusableButton`, `_whatsNewFocusNode`, `_aboutFocusNode`, …). Aperture must
not weaken it, and it is easy to weaken by accident.

**Focus and selection are visually distinct and must never be merged:**

- **Focused** (keyboard/D-pad position): a 2px `--pc-primary` ring at 2px
  offset, plus a one-step surface lift. Always drawn, never suppressed, and
  never conveyed by colour alone — the ring is a shape change.
- **Selected** (current value/route): the inline-start bracket + soft fill.

A control can be focused and not selected, selected and not focused, or both,
and all four combinations must be visually distinguishable. This is a hard
requirement, not a preference: on a TV the focus ring *is* the cursor.

---

## 9. Web dashboard design

### 9.1 Sidebar branding

Today `app-header.tsx:103` renders `/logo_with_text.png` — the PicoClaw lobster
and wordmark — inside a `w-36` box on `sm:` and above. Under Aperture this
becomes the PocketClaw lockup (§15): mark + wordmark as an inline SVG that
inherits `currentColor`, so it is correct in both themes with no second asset
and no flash of the wrong logo.

Placement moves from the header into the **sidebar header**, which is where a
product mark belongs in a sidebar layout, leaving the top bar for state and
actions. On mobile, where the sidebar is off-canvas, a compact mark-only
lockup stays in the header beside the menu button.

### 9.2 Desktop sidebar

- Width 264px, `surface-1`, hairline on the inline-end edge.
- Sidebar header: 56px, lockup, hairline beneath.
- Group labels at `micro` uppercase in `--pc-text-faint`, 20px above / 8px
  below. The existing collapsible groups (`navigation.chat`,
  `navigation.model_group`, `navigation.agent_group`, `navigation.services`)
  are kept exactly as they are.
- Rows: 36px, `radius-sm`, 10px gap, icon 18px. Hover `--pc-nav-hover-bg`.
  Active: `--pc-primary-soft` fill + 2px inline-start bracket + `--pc-primary`
  icon.
- Channel rows keep their per-channel icon masks — including `lark.svg`, which
  is a *platform* icon and not branding.
- `SidebarRail` (the desktop collapse handle) is retained; on desktop a
  panel-collapse metaphor is correct.

### 9.3 Top toolbar

The current header is doing too many jobs at once. Aperture re-orders it into
three zones with the gateway control as the clear centre of gravity:

`[ menu (mobile) | mark (mobile) ]  ·······  [ GATEWAY STATE ]  ·······  [ restart · language · theme · logout ]`

- The gateway pill is the toolbar's most important object: a status dot plus a
  label. Running = `--pc-signal` dot with a slow 2.4s pulse, "Running", and a
  Stop affordance revealed on hover/focus. Stopped = neutral dot and a filled
  Start button. Starting/restarting/stopping = amber dot, spinner, present
  tense label. This replaces today's green `bg-green-500` hardcode, which is
  outside the token system entirely.
- The "not connected" centre hint keeps its dashed-border treatment but adopts
  `--pc-danger-line` and `caption` size.
- Icon buttons go to 40×40 with 20px glyphs (from `size-8`/`size-4.5`).

### 9.4 Pages

| Page | Direction |
| --- | --- |
| **Models** | Cards, not a table. Each: model name at `heading`, provider chip, identifier in mono. A `DEFAULT` badge on the active one. **Room is deliberately left for future routing roles** (§19) — the card already reserves a badge slot, so Default / Vision / Fallback can land later as sibling badges without a redesign. |
| **Credentials** | Locked rows. Secret masked to `••••••••`, never selectable, with a mono suffix hint. `--pc-success` dot when verified. |
| **Channels** | One card per channel, platform icon in a 40px `surface-3` tile, connection chip, one primary action. |
| **Telegram** | The channel card promoted to a page. Bot handle in mono, QR pairing in a bordered `surface-2` well with generous padding — a QR on a busy background is a scan failure. |
| **Hub / Skills / Tools** | A 3/2/1-column responsive grid of equal-height cards; icon, name, one-line description, install/enable state. This is the surface most at risk of looking like a package manager: fix it with generous padding, real spacing between rows, and never more than one accent per card. |
| **Config** | Two-column on desktop (label + control), single column below `md`. Grouped with `micro` section labels. Mono for paths, ports and hosts. |
| **Logs** | Full-bleed mono readout on `--pc-bg`, 1.5 line-height, level chips (`micro`) in the gutter, hairline row separators at 6% opacity. Filter bar sticky at top. This page should look *the most* like an instrument. |
| **Dialogs / sheets** | Elevation 4, `radius-lg`, 24px padding, title at `heading`. Sheets slide from the inline-end edge so they mirror in RTL. |
| **Language selector** | Keeps its current endonym list and `dir={i18n.dir(option.code)}` per option — that is already correct and stays. |

### 9.5 What reduces the "admin panel" feeling

Concretely, five changes carry most of the effect:

1. Tinted neutrals instead of chroma-0 grey (§4).
2. Cards instead of dense tables on Models, Channels and Skills.
3. `micro` uppercase group labels giving the sidebar and Config real structure.
4. One accent per screen region, with Signal reserved for live state.
5. Mono applied to machine values, so data *looks* like data.

---

## 10. Chat design

Chat is the product's primary surface and currently the least differentiated.

### 10.1 Conversation hierarchy

The strongest available move is to **stop styling user and assistant messages
symmetrically**. Two bubbles facing each other is a messaging app; PocketClaw
is an agent session.

- **User message.** A contained bubble, inline-end aligned, `surface-2`,
  `radius-lg` with the inline-end bottom corner tightened to `radius-sm`, max
  width 78ch. Compact.
- **Assistant message.** *Not* a bubble. Full measure, transparent background,
  starting at the inline-start edge with a 2px `--pc-primary-line` bracket down
  its full height. This makes the agent's output feel like a document rather
  than a chat reply, gives long answers, code and tables the width they need,
  and creates an obvious visual rhythm down the thread.
- **Tool execution.** A collapsed row: `surface-1`, hairline, `radius-sm`,
  tool name in mono, duration on the trailing edge, chevron to expand. Running
  = `--pc-signal` dot. Succeeded = `--pc-success`. Failed = `--pc-danger` and
  auto-expanded, because a failure the user has to click to see is a failure
  they will miss.
- **Reasoning / detail controls.** The existing visibility behaviour in
  `detail-visibility.ts` is **unchanged**. Aperture restyles the toggle and
  nothing else. No hidden reasoning content is exposed, and no default is
  altered.

### 10.2 Composer

A single rounded well pinned to the bottom, `surface-2`, hairline,
`radius-lg`, 12px padding, auto-growing 1→8 rows.

```
┌──────────────────────────────────────────────────────┐
│  Message PocketClaw…                                 │
│                                                      │
│  [＋]  [ model ▾ ]                        [ ⏎ Send ] │
└──────────────────────────────────────────────────────┘
```

- Focus lifts the border to `--pc-primary` and adds the 2px ring. The whole
  well responds, not just the textarea — a composer whose border does not react
  is the clearest "unfinished" tell in a chat UI.
- **Attachment** is a 40×40 ghost `＋` on the inline-start edge. It maps to the
  existing Android file-chooser path; the affordance is being made legible, not
  rewired.
- **Send / Stop occupy the same slot** and never shift layout. Idle: filled
  `--pc-primary` with a send glyph, disabled when empty. Streaming: the same
  footprint becomes a `--pc-signal`-bordered square-stop button. One position,
  two states, no reflow mid-response.
- The model selector sits in the composer footer, not in a distant toolbar —
  model choice is part of composing a message.
- `context-usage-ring` (already built) moves beside the model selector as a
  12px ring: `--pc-text-faint` under 60%, `--pc-warning` 60–85%,
  `--pc-danger` above. Existing component, better placed.

### 10.3 Empty state

Today `chat-empty-state.tsx` wraps its whole block in `opacity-70`, which drags
the heading and body text below comfortable contrast. Aperture removes the
opacity entirely and builds hierarchy from colour tokens instead — the same
visual softness, without the accessibility cost.

Structure: the PocketClaw mark at 48px in `--pc-text-faint`, a `title`
greeting, one `caption` line naming the active model in mono, then three
suggestion chips (`surface-1`, hairline, `radius-full`) that pre-fill the
composer. The existing three conditional variants — no models configured, no
default selected, not connected — are preserved exactly, each keeping its
distinct icon and its call to action, restyled to the token set.

### 10.4 Mobile chat

Composer pinned above the keyboard with safe-area padding. Assistant messages
go edge-to-edge with 16px gutters. The model selector collapses to a chip
showing only the short name. The session-history menu becomes a bottom sheet
rather than a dropdown.

---

## 11. Mobile design

Targets: Android phones (Flutter shell) and the narrow WebView the dashboard
renders inside. The WebView case is the harder one and is usually the one that
gets neglected.

- **Touch targets: 44×44 minimum, no exceptions.** Several current controls
  miss this — the sidebar trigger is `h-9 w-9` (36px) and the header icon
  buttons are `size-8` (32px). Visual size may stay small; the *hit area* must
  not.
- Single column below 640px. Two-column forms collapse to one.
- Page padding 16px; card padding 16px.
- Bottom-anchored primary actions in a `surface-1` bar with a top hairline and
  safe-area inset, so the thumb reaches them and the keyboard does not cover
  them.
- Dialogs become bottom sheets below 640px, with a grab handle.
- Tables become stacked cards. No horizontal scrolling of tabular data.
- Text scaling: every container sizes from content. Nothing is a fixed height
  that holds text — this is what breaks at 200% font scale.
- The Settings header `Wrap` from the What's New milestone is the pattern to
  copy for any header that pairs a title with actions.

---

## 12. Desktop design

Desktop must not be a stretched phone layout.

- Content max-width 1200px, centred, with a wider 1440px allowance for Logs.
- Chat measure capped at 78ch for prose regardless of window width. Full-width
  chat text is unreadable and is a common failure in agent consoles.
- Two-column forms above 1024px: label column 220px, control column fluid.
- Grids: 3 columns ≥1280px, 2 ≥900px, 1 below.
- Hover states exist and are meaningful — 120ms surface lift on rows and cards.
- Keyboard: visible focus everywhere, `⌘/Ctrl+K` command palette (`cmdk` is
  already a dependency), `Esc` closes the top layer.
- The sidebar is persistent, never a drawer, above 1024px.

---

## 13. RTL strategy

Arabic is a first-class target and the system is designed so that being correct
in Arabic is the *default*, not an extra pass.

**The rule: no physical direction in any product code.**

| Never | Always |
| --- | --- |
| `margin-left`, `padding-right` | `margin-inline-start`, `padding-inline-end` |
| `left: 0`, `text-align: left` | `inset-inline-start: 0`, `text-align: start` |
| `border-left` | `border-inline-start` |
| `EdgeInsets.only(left:)` | `EdgeInsetsDirectional.only(start:)` |
| `Alignment.centerLeft` | `AlignmentDirectional.centerStart` |
| `Icons.arrow_back` | `Icons.arrow_back` under `Directionality` (auto-mirrors) |

Because every Aperture device — the active-nav bracket, the assistant-message
rail, the composer attachment button, the card accent, sheet entry direction —
is specified in logical properties, **the entire system mirrors with no
Arabic-specific branch anywhere.** This is the reason the bracket was chosen
over, say, a left-anchored glow.

Already correct in the codebase and preserved as-is:
`sidebar.tsx:170` resolving `side` from `i18n.dir()`; the `ltr:`/`rtl:` variant
usage on `SidebarRail`; `dir={i18n.dir(option.code)}` per language option; the
RTL drawer mirroring shipped in the Dashboard i18n milestone; and the
`EdgeInsetsDirectional` layout of the What's New page.

**Deliberate LTR exceptions, kept:** model identifiers, host addresses, ports,
file paths, SHA digests and code blocks stay `dir="ltr"` even in Arabic —
`add-model-sheet.tsx:661` and `edit-model-sheet.tsx:595` already do this and
are right. A reversed hostname is a bug, not localization.

**Long-translation resilience.** German and Russian labels run 30–50% longer
than English. Every button, chip and nav row sizes to content with `Wrap`
(Flutter) or `flex-wrap` (web) at the container level, never a fixed width.
The What's New header defect is the canonical example of what happens
otherwise, and its `Wrap` fix is the pattern.

---

## 14. Accessibility

Not a review gate at the end — constraints the tokens already satisfy.

**Contrast.** Every pairing in §4 is chosen against WCAG AA.

| Pair | Ratio | Requirement |
| --- | --- | --- |
| `--pc-text` on `--pc-bg` | ~15:1 | AA 4.5 ✔ (AAA) |
| `--pc-text-muted` on `--pc-bg` | ~7:1 | AA 4.5 ✔ |
| `--pc-text-faint` on `--pc-bg` | ~4.6:1 | AA 4.5 ✔ |
| `--pc-primary-fg` on `--pc-primary` | ~9:1 | AA 4.5 ✔ |
| `--pc-primary` on `--pc-bg` | ~6.5:1 | AA non-text 3.0 ✔ |
| hairline on adjacent surface | ~3.1:1 | AA non-text 3.0 ✔ |

`--pc-text-faint` is the floor. Nothing renders below it, and the blanket
`opacity-70` in the chat empty state is removed rather than re-tuned.

**Never colour alone.** Every status carries a shape or a word as well as a
hue: the running dot has a "Running" label, the failed tool row has an icon and
auto-expands, the required field has an asterisk and a message. This also
covers the ~8% of users with colour-vision deficiency for whom the
success/warning distinction is otherwise invisible.

**Touch targets** 44×44 minimum. **Focus** always visible, 2px ring at 2px
offset, never `outline: none` without a replacement. **D-pad** the existing
focus chain is preserved exactly (§8.5). **Screen readers**: icon-only buttons
keep an accessible name; status dots are `aria-hidden` with the text label
carrying the meaning; live regions announce gateway transitions once, not on
every poll. **Text scaling** to 200% without clipping. **Reduced motion**:
`prefers-reduced-motion` collapses every transition to opacity-only and stops
the gateway pulse.

---

## 15. PocketClaw branding treatment

### 15.1 The idea worth keeping

`assets/branding/pocketclaw-mark.png` has a genuinely good *concept* buried in
a bad *execution*: a claw rising out of an open container. That is exactly what
the name says — **pocket** + **claw** — and no competitor owns it.

Aperture keeps the concept and rebuilds the execution.

### 15.2 The mark

A geometric mark on a 24×24 grid, 2-unit stroke, single path, one colour:

- An **open container** — a squared U, drawn in perspective-free flat
  elevation, open at the top.
- A **claw** — two mirrored chevron arms rising and angling outward from
  inside it.
- The negative space between the arms forms the **aperture**: the same bracket
  used for active nav, focus and the assistant rail.

Requirements it must meet, all of which the current PNG fails:

| Requirement | Why |
| --- | --- |
| Legible at 16px | favicon, tab, status bar |
| Single flat colour, `currentColor` | works in both themes from one asset |
| No gradient, no glow, no 3D | reproducible in print, embroidery, monochrome |
| SVG, under 2 KB | vs. the current 654 KB PNG |
| Square and safe-area balanced | Android adaptive icon masking |

### 15.3 Lockups

1. **Mark** — square, standalone. App icon, favicon, avatar, empty states.
2. **Horizontal lockup** — mark + "PocketClaw" in Inter SemiBold, cap-height
   matched, gap = one stroke unit. Sidebar header, desktop toolbar.
3. **Stacked lockup** — mark above wordmark. Splash, About, onboarding.

Wordmark is set in Inter SemiBold at `-0.02em`, one word, capital P and C. Never
`Pocketclaw`, `POCKETCLAW`, or `Pocket Claw`.

### 15.4 Application

| Surface | Asset |
| --- | --- |
| Android launcher | adaptive icon: mark foreground, `--pc-bg` background |
| Android splash | stacked lockup, centred, `--pc-bg` |
| Flutter About | stacked lockup |
| Web sidebar header | horizontal lockup, inline SVG, `currentColor` |
| Web mobile header | mark only |
| Favicon | mark, 16/32/48 in one `.ico` + an SVG favicon |
| Chat empty state | mark at 48px in `--pc-text-faint` |

**Do not** re-colour the mark per theme, place it on a gradient, add a glow,
rotate it, or stretch it. Clear space = one stroke unit on all sides.

---

## 16. PicoClaw visible-brand removal plan

Findings from an audit of `core/src/web/frontend/`, with the deliberate
boundary the brief requires: **user-visible surfaces only, no global rename.**

### 16.1 Must be replaced — user-visible

| # | Asset / file | Evidence | Action |
| --- | --- | --- | --- |
| 1 | `public/logo_with_text.png` | PicoClaw lobster + **PICOCLAW** wordmark, navy/coral. Rendered at `app-header.tsx:103` in a `w-36` box on `sm:` and above — **this is the desktop upper-left wordmark reported.** Its `alt` already says "PocketClaw logo", so today the alt text contradicts the image. | Replace with the PocketClaw horizontal lockup as inline SVG; delete the PNG. |
| 2 | `public/favicon.svg` (90 KB) | lobster | Replace with the mark, <2 KB |
| 3 | `public/favicon-96x96.png` | lobster — confirmed visually | Replace |
| 4 | `public/favicon.ico` | lobster | Replace |
| 5 | `public/apple-touch-icon.png` | lobster | Replace |
| 6 | `public/web-app-manifest-192x192.png` | lobster | Replace |
| 7 | `public/web-app-manifest-512x512.png` | lobster | Replace |
| 8 | `public/site.webmanifest` | `"name": "MyWebSite"`, `"short_name": "MySite"`, `theme_color: "#ffffff"` — placeholder boilerplate, never branded at all | Rewrite: `PocketClaw` / `PocketClaw`, `theme_color` = `--pc-bg`, `background_color` = `--pc-bg` |

### 16.2 Verified clean — no action

- `index.html` `<title>` is already `PocketClaw`.
- `header.logoAlt` is already "PocketClaw logo" in all twelve locale bundles
  (`en.json:148` and siblings).
- No `PICOCLAW` string appears in any user-visible i18n value.

### 16.3 Explicitly NOT renamed — internal and load-bearing

Per the brief, these stay. Renaming them is a separate, independently-proven
piece of work and **must not** ride along with a visual change:

- `package.json` `"name": "picoclaw-web"` — build identifier, never rendered.
- `localStorage` / state keys in `features/chat/state.ts`,
  `detail-visibility.ts`, `store/code-block.ts`, `store/tour.ts`,
  `hooks/use-highlight-theme.ts`, `hooks/use-credentials-page.ts` — renaming
  these silently discards every user's saved preference.
- `lib/plain-text-log.ts` and the log contract it implements.
- `libpicoclaw.so`, `libpicoclaw-web.so`, Go package paths, config keys, the
  channel/transport identifiers, and the OpenClaw importer's expectations.

**`public/lark.svg` is not branding.** It is the Lark/Feishu platform icon,
used as a CSS mask for a channel row in `use-sidebar-channels.ts:57-58`. It
stays.

### 16.4 Sequencing

Assets first (items 1–8 in one commit, no code change beyond the header's
`img` → inline SVG swap), then tokens, then components. Item 1 is the one the
user actually sees on desktop and should not wait for the rest of the redesign.

---

## 17. Mobile menu icon solution

### 17.1 Root cause — located precisely

The header already asks for a hamburger. `app-header.tsx:93-95`:

```tsx
<SidebarTrigger className="… sm:hidden [&>svg]:size-5">
  <IconMenu2 />
</SidebarTrigger>
```

But `ui/sidebar.tsx:259-284` renders:

```tsx
function SidebarTrigger({ className, onClick, ...props }) {
  return (
    <Button … {...props}>
      <IconLayoutSidebar />                        {/* ← hardcoded */}
      <span className="sr-only">{t("common.toggleSidebar")}</span>
    </Button>
  )
}
```

`children` arrives inside `...props`, but **JSX children written between the
tags override children passed via spread.** The `IconMenu2` the header supplies
is silently discarded, and `IconLayoutSidebar` — a panel/layout glyph — renders
in its place.

That is exactly the reported symptom: on mobile the control looks like a
screen-layout toggle rather than navigation. The original author's intent was
correct; the component swallowed it.

### 17.2 The fix

Three small changes, no behaviour change:

**a. Let the trigger accept an icon.**

```tsx
function SidebarTrigger({ className, onClick, children, ...props }) {
  const { toggleSidebar } = useSidebar()
  const { t } = useTranslation()
  return (
    <Button data-sidebar="trigger" data-slot="sidebar-trigger"
            variant="ghost" size="icon-sm" className={cn(className)}
            onClick={(e) => { onClick?.(e); toggleSidebar() }} {...props}>
      {children ?? <IconLayoutSidebar />}
      <span className="sr-only">{t("common.toggleSidebar")}</span>
    </Button>
  )
}
```

The default is unchanged, so every other call site keeps today's icon. The
header's existing `<IconMenu2 />` — three horizontal lines, the universal menu
affordance — now actually renders on mobile.

**b. Give it a real touch target.** `h-9 w-9` (36px) → `h-11 w-11` (44px) with
a 20px glyph. The visual weight barely changes; the hit area passes.

**c. Give it an honest accessible name.** The mobile control announces
"Open menu" / "Close menu" (a new `common.openMenu` / `common.closeMenu` pair
across the twelve bundles), while the desktop `SidebarRail` keeps
`common.toggleSidebar`. Today both say "toggle sidebar", which is wrong for a
navigation drawer.

### 17.3 Desktop stays as it is

On desktop the sidebar genuinely is a collapsible panel, so `SidebarRail` keeps
`IconLayoutSidebar`. **Mobile and desktop are not required to match** — the
semantics differ, so the icons should too. Mobile navigates; desktop collapses.

### 17.4 RTL and behaviour

Placement already follows `i18n.dir()` via `sidebar.tsx:170`, so the button sits
at the inline-start edge and the drawer enters from the correct side in Arabic.
A hamburger is direction-neutral and needs no mirroring. Open/close behaviour,
`setOpenMobile`, and the auto-close on nav selection are **untouched**.

---

## 18. Component examples

### Status chip

```
●  Running          ●  Stopped         ●  Starting…
```

6px dot + `micro` label, `radius-full`, `surface-1` fill, hairline.
Running `--pc-signal` with a 2.4s pulse; stopped `--pc-text-faint`, no
animation; starting `--pc-warning` with a spinner. The label is always present
— the dot never carries the meaning alone.

### Card

```
┌────────────────────────────────────────┐  surface-1, radius-md, hairline
│  Public Mode                    [ ⬤ ]  │  heading + switch (44px hit area)
│  Exposes the Dashboard on your LAN.    │  caption / --pc-text-muted
│  ▏192.168.1.42:8080                    │  bracket + mono value
└────────────────────────────────────────┘
```

### Navigation row (active)

```
▌  ◈  Chat                                 ← 2px --pc-primary bracket,
                                             --pc-primary-soft fill,
                                             --pc-primary icon + label
```

`border-inline-start` — mirrors in Arabic with no branch.

### Assistant message

```
▏ PocketClaw
▏
▏ Here is the summary you asked for…
▏
▏ ┌────────────────────────────────┐
▏ │ ⚙ read_file        124ms   ▸  │   collapsed tool row
▏ └────────────────────────────────┘
```

Full measure, no bubble, 2px `--pc-primary-line` rail down the inline-start
edge.

### Empty state

```
            ◈                    mark, 48px, --pc-text-faint
     Ready when you are          title
   Using claude-opus-4.6         caption, model name in mono

  [ Summarise a file ]  [ Check gateway ]  [ Run a command ]
```

No `opacity` wrapper — hierarchy comes from tokens.

### Loading

Skeletons, not spinners, for content that has a known shape: `surface-3`
blocks at the final dimensions with a 1.6s shimmer, so nothing reflows on
arrival. Spinners only for indeterminate actions inside a button.

---

## 19. Implementation phases

Each phase is independently shippable and independently revertible. No phase
depends on a later one.

| Phase | Scope | Risk |
| --- | --- | --- |
| **0 — Review** | This document. Agree concept, palette, mark. | none |
| **1 — Brand assets** | §16 items 1–8. New SVG mark and lockups. Header `img` → inline SVG. `site.webmanifest`. **Removes the visible PICOCLAW wordmark.** | very low — assets + one element |
| **2 — Web tokens** | Rewrite `index.css` `:root` / `.dark` to Aperture. Remap existing shadcn vars onto `--pc-*`. No component edits. | low — every component inherits |
| **3 — Menu icon** | §17 a/b/c. `SidebarTrigger` children, 44px target, `openMenu`/`closeMenu` strings ×12. | very low, high user value |
| **4 — Flutter tokens** | Unify `PocketClawDesign` and `AppTheme` into one theme emitting both brightnesses; re-base the six modes as accents. | medium — touches theme selection; needs a widget-test pass |
| **5 — Web shell** | Sidebar, toolbar, gateway pill, nav bracket, group labels. | low |
| **6 — Chat** | Assistant rail, composer well, send/stop slot, tool rows, empty state. | medium — the most-used surface |
| **7 — Flutter Settings** | Cards, segmented Context Memory control, status chips, What's New page. | low |
| **8 — Web pages** | Models, Credentials, Channels, Hub, Skills, Tools, Config, Logs. | low, but broad |
| **9 — Polish** | Motion, skeletons, reduced-motion, focus audit, RTL sweep, 200% text-scale sweep. | low |

Recommended order if only part is taken: **1 → 3 → 2**. Those three remove the
competitor's wordmark, fix a genuine usability defect, and de-genericise every
web surface at once, for very little code.

---

## 20. Risks, and what must not change functionally

### Functional freeze — verified untouched by every proposal above

Telegram behaviour and the bounded-context algorithm · Context Memory limits
and live apply · the config cache's file-identity key · the model/provider
architecture · Managed Runtime · GitHub auth and the Keystore path · the Python
runtime · session behaviour · the security model · gateway networking and
lifecycle · **What's New seen-state logic** (`versionName` only, written on
open) · the D-pad focus chain · `detail-visibility` semantics.

Aperture is a token, asset and styling proposal. Where it touches structure it
says so explicitly, and there are exactly three such places: §17a
(`SidebarTrigger` renders `children`), §8.1 (one theme source instead of two),
and §9.3 (toolbar element order). None changes what any control *does*.

### Risks

| # | Risk | Mitigation |
| --- | --- | --- |
| 1 | **Phase 4 touches theme selection.** Six existing modes are user-visible state; re-basing them could change what a saved preference resolves to. | Keep the enum, the persisted key and all six names. Re-base colours only. Widget-test each mode before/after. |
| 2 | **Dark-mode contrast regression.** Tinted neutrals are new; a wrong chroma can quietly drop text below AA. | The §14 table is the acceptance test. Verify computed ratios, not appearance. |
| 3 | **RTL regression.** Any new physical property reintroduces a mirroring bug. | Lint against `margin-left`/`padding-right`/`EdgeInsets.only(left:)` in changed files. Keep the existing Arabic tests. |
| 4 | **Long translations.** New chips and segmented controls are prime overflow candidates in German, Russian and Arabic. | Everything sizes to content. Test German and Arabic at 360px. The What's New header defect is the precedent. |
| 5 | **Touch-target changes shift layout.** 32/36px → 44px in a dense toolbar can wrap a row. | Grow hit area via padding/`::before`, not always visual size. |
| 6 | **Chat is the highest-traffic surface** (Phase 6), so a regression there is the most costly. | Ship after 1–5. Keep the streaming path, `detail-visibility` and the file-chooser bridge untouched. |
| 7 | **Asset replacement can break the Android adaptive icon** if safe areas are wrong. | Design the mark inside the 66% adaptive safe zone; check the round, squircle and square masks. |
| 8 | **Scope creep into the internal rename.** The tempting next step after removing the wordmark is renaming `picoclaw-web` and the storage keys. | Explicitly out of scope (§16.3). Renaming storage keys discards user preferences and needs its own migration and its own proof. |
| 9 | **`libpicoclaw-web.so` must be rebuilt** for any web change to reach the device — the console is compiled into the Core binary. | Any web-surface phase requires a Core rebuild + physical verification. Not needed for the Flutter-only phases (4, 7). |

### Explicitly not implemented

**Vision / Image model routing is not in this concept.** The Models design in
§9.4 deliberately reserves a badge slot on each model card so that a future
Default / Vision / Fallback role can be expressed as sibling badges without
another redesign. That is conceptual room only. No routing control, no role
selector, no data model, and no string is proposed here.

---

*Concept A. Independent proposal. Not implemented, not merged.*
