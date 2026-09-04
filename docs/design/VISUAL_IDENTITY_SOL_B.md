# PocketClaw Visual Identity 2.0 — SOL Concept B

Design basis: production `develop` at `9877365`. This is a visual and interaction
proposal, not a production implementation. It preserves all existing product,
runtime, networking, privacy, and session behavior.

## 1. Concept name

## GRIPLINE — The Quiet Instrument

PocketClaw is presented as a precise personal instrument: powerful machinery held
inside a calm, comprehensible shell. "Grip" belongs to the claw and to local
control. "Line" is the visual trace that ties the brand mark, navigation, cards,
focus, runtime state, and tool activity together.

The intended emotional sequence is:

1. **Composed at rest.** Warm neutral surfaces, crisp type, and restrained color.
2. **Explicit when addressed.** Selection, focus, and warnings never hide behind
   low-contrast decoration.
3. **Alive while working.** A short segmented line moves only on the component
   doing work; the whole screen does not glow or pulse.
4. **Settled when complete.** Motion resolves into a static state label and icon.

This is not a mascot-led identity, a glass dashboard, or a sci-fi control room.
It is a durable product language for an AI agent that can run real tools.

## 2. Core philosophy

### Power held, not performed

PocketClaw should show capability through structure and feedback, not spectacle.
The UI uses strong hierarchy, concise state language, legible technical values,
and deliberate density. Color answers a question—identity, action, state, or
warning—and is not used as ambient decoration.

### One identity, native composition

Flutter and Web share tokens, the mark, typography roles, icon weight, control
states, Gripline accents, and motion timings. Their layouts remain platform
appropriate:

- Flutter uses a thumb-reachable bottom instrument dock, full-width setting
  groups, Android sheets, and visible D-pad focus.
- Web uses a persistent desktop navigation frame, wide editorial reading space,
  keyboard shortcuts, hover feedback, and responsive workbenches.
- The narrow Android WebView uses the Web layout's mobile composition. It does
  not pretend to be an identical copy of the Flutter shell around it.

### Calm density

The system avoids both dashboard clutter and oversized marketing whitespace.
The default content width is driven by the task:

- prose/chat: `760–840px` reading column;
- forms: `720px` primary column, optional `280px` contextual rail on wide Web;
- libraries (Skills, Tools, Models): `1120–1280px` workbench;
- phone: full width with `16px` gutters, never a desktop card shrunk into a
  narrow viewport.

### Honest state

"Saved," "applied," "running," "connected," and "available" are different
states. Existing behavior remains authoritative. The design may improve how a
known state is displayed; it must not infer a state, promise a restart, expose a
credential, or imply a capability that the runtime did not report.

## 3. What makes PocketClaw visually unique

The identity has four repeatable signatures. They are a system, not isolated
decoration.

### 3.1 The Gripline

A `3px` logical-start rail identifies the current object. It contains one solid
section and two short separated terminals. The proportions echo the three masses
in the mark. It appears on:

- the selected desktop navigation item;
- the active Flutter destination;
- the current model route;
- the streaming agent message;
- the focused feature panel on keyboard/D-pad input;
- a tool-call row while that call is active.

Rules:

- Static selection uses one solid line plus two fixed terminals.
- Indeterminate work moves a `12px` tracer through the line once per `900ms`.
- Success, warning, and error replace motion with a static terminal glyph.
- The line mirrors to logical start in RTL.
- It never appears on every card at once.
- It never represents hidden reasoning. It only marks visible selection or a
  user-observable operation state.

### 3.2 The tension corner

Feature surfaces have three `14px` corners and a tighter `4px` block-start,
inline-end corner. The asymmetry suggests a held sheet or folded pocket without
turning every rectangle into a speech bubble. In RTL, inline-end moves, so the
corner mirrors automatically.

Use it on hero controls, major route cards, dialogs, and the composer deck. Plain
rows, chips, tooltips, and small controls keep simpler radii.

### 3.3 The pocket field

Related controls sit inside a slightly sunken field rather than in a stack of
independent floating cards. The pattern is:

`canvas → section field → row/feature surface → raised transient surface`.

Settings becomes a small number of coherent groups. Models becomes a routing
workbench. Chat becomes a reading surface plus an anchored composer. This is the
main structural change that stops the product reading as a generic admin console.

### 3.4 Coral is a grip, not a wash

Claw Coral is used for the mark's gripping terminal, the user-message edge,
attachment affordances, and a small number of decisive moments. It never becomes
a page background or arbitrary gradient. Tidal Teal carries product action and
agent state. Their roles remain stable across light and dark modes.

## 4. Brand / mark proposal

### 4.1 Name: the Pocket Hinge

Replace both the old lobster identity and the current detailed neon raster with a
new vector-first mark called the **Pocket Hinge**.

The glyph is built on a `24 × 24` grid from three flat masses:

1. a broad lower pocket/fold, open at the top;
2. two opposing articulated prongs that turn inward;
3. a square-round central aperture held in negative space.

The lower fold makes "Pocket" visible; the opposing prongs make "Claw" visible;
the held aperture is the agent/tool target. The silhouette is mechanical but not
aggressive, and it does not depend on eyes, a cartoon animal, circuitry, letters,
or a detailed 3D render.

### 4.2 Construction rules

- Safe area: `2/24` of the glyph on all sides.
- Minimum gap: `2/24`; no hairline detail.
- Full-color form: Ink pocket, Tidal Teal left prong, Claw Coral right terminal,
  canvas-colored aperture.
- One-color form: a single filled silhouette with the aperture and joint gaps
  knocked out.
- App icon: glyph centered at 68% of an Ink squircle; no wordmark.
- Favicon at `16px`: simplified glyph with the same aperture and two prongs;
  remove the inner fold seam.
- Notification icon: monochrome silhouette only, valid Android alpha mask.
- Sidebar: `28px` glyph plus a custom text lockup; never use the full app icon.
- Wordmark: `Pocket` semibold and `Claw` regular in the product typeface. No
  uppercase shouting, color split, or mascot attachment.
- Clear space around a standalone mark: at least the aperture width.

### 4.3 Required exported asset family

Implementation should create all variants from one reviewed vector master:

| Asset                                         | Form                                       | Requirement                        |
| --------------------------------------------- | ------------------------------------------ | ---------------------------------- |
| `pocketclaw-mark.svg`                         | vector, currentColor-capable               | Web mark and source of truth       |
| `pocketclaw-lockup.svg`                       | vector mark + outlined/controlled wordmark | desktop header/sidebar only        |
| `favicon.svg`                                 | simplified vector                          | light/dark media-aware fills       |
| `favicon.ico`, `favicon-96x96.png`            | derived raster                             | browser fallback                   |
| `apple-touch-icon.png`                        | Ink squircle, full-color glyph             | opaque background                  |
| `web-app-manifest-192x192.png`, `512x512.png` | full-color icon                            | PWA install surfaces               |
| Android adaptive foreground/background        | separated vector/raster layers             | safe zone and monochrome icon      |
| `ic_stat_pocketclaw.xml`                      | one-color glyph                            | notification constraints           |
| desktop `.ico` and tray variants              | simplified glyph                           | 16/20/24/32px checked individually |

The mark is intentionally vector-like because recognition at `16–24px` is a
hard requirement. A large raster may be exported from it, but must never be the
design master.

## 5. Color / token system

### 5.1 Core palette

No decorative gradients are part of GRIPLINE. Tonal change comes from adjacent
surfaces, borders, and restrained shadows.

| Role             | Light     | Dark      | Use                                       |
| ---------------- | --------- | --------- | ----------------------------------------- |
| `canvas`         | `#F3F4EF` | `#0D1211` | app/page background                       |
| `surface`        | `#FFFFFF` | `#141B19` | main cards, message plane                 |
| `surface-raised` | `#FAFBF8` | `#1A2421` | dialogs, menus, composer                  |
| `surface-sunken` | `#E9ECE7` | `#090D0C` | grouped fields, code wells                |
| `ink`            | `#121918` | `#EDF4F0` | primary content                           |
| `ink-muted`      | `#56615E` | `#AAB7B2` | supporting text                           |
| `ink-faint`      | `#737E7A` | `#7F8E88` | tertiary text; never body copy            |
| `border`         | `#CED6D1` | `#31403B` | structural boundaries                     |
| `border-strong`  | `#9EAAA5` | `#52645D` | controls, hover boundaries                |
| `tidal`          | `#0C6861` | `#78D9C9` | primary action, agent identity            |
| `tidal-hover`    | `#095750` | `#91E4D5` | hover/high emphasis                       |
| `tidal-soft`     | `#D5ECE7` | `#183B35` | selection and informative fields          |
| `coral`          | `#C84632` | `#FF876F` | claw terminal, user edge, decisive accent |
| `coral-soft`     | `#F8DED7` | `#43251F` | attachment/user supporting surface        |
| `focus`          | `#146EF5` | `#75A7FF` | keyboard/D-pad focus only                 |

Verified representative WCAG contrast ratios:

- Light Ink on Canvas: `16.13:1`.
- Light Muted Ink on Canvas: `5.81:1`.
- White on light Tidal action: `6.63:1`.
- White on light Coral action: `4.80:1`.
- Dark Ink on dark Canvas: `16.91:1`.
- Dark Muted Ink on dark Canvas: `9.11:1`.
- Dark Canvas ink on dark Tidal action: `11.33:1`.

### 5.2 Semantic colors

| State   | Light / Dark foreground | Soft field            | Required companion         |
| ------- | ----------------------- | --------------------- | -------------------------- |
| success | `#247A4B` / `#72D79A`   | `#DDF1E4` / `#173624` | check icon + text          |
| warning | `#8A5700` / `#F5BE63`   | `#F7E8C8` / `#3A2C13` | triangle icon + text       |
| error   | `#B83B3B` / `#FF8B88`   | `#F6DEDD` / `#402020` | octagon/exclamation + text |
| info    | `#265F9E` / `#80B8F2`   | `#DDEAF6` / `#172D42` | info icon + text           |
| neutral | `#56615E` / `#AAB7B2`   | surface-sunken        | shape/icon + text          |

Status is never encoded by a colored dot alone. Use label + icon + color. For
compact status, the minimum is a distinct shape with an accessible name: circle
check (healthy), diamond clock (pending), square stop (stopped), octagon alert
(error).

### 5.3 Interaction tokens

- hover: mix `6%` current ink into the local surface; strengthen border one step;
- pressed: mix `11%` ink, translate at most `1px` with no scale shrink;
- selected: `tidal-soft` field + `tidal` Gripline + semibold label;
- focus: `2px focus` ring, `2px` canvas/surface separation gap;
- disabled: retain label at minimum `3:1` where practical, remove shadow, reduce
  surface contrast, keep status/reason text fully legible;
- text selection: `tidal-soft` background and Ink foreground;
- destructive confirmation: neutral dialog until the final destructive button;
  do not tint the entire dialog red.

## 6. Typography

### 6.1 Family

Use an open, locally bundled family strategy:

- UI Latin/Cyrillic: **Source Sans 3 Variable** (SIL OFL 1.1).
- Arabic: **Noto Sans Arabic Variable** (SIL OFL 1.1).
- Devanagari/Bengali/CJK/Korean/Japanese: matching Noto Sans script families,
  subset and loaded by locale.
- Code and machine values: **IBM Plex Mono** (SIL OFL 1.1), then platform mono
  fallbacks.

Locale-specific font selection is normal script support, not layout branching.
The same semantic sizes, weights, and spacing tokens apply. Never force Latin
letter spacing or uppercase onto Arabic, Devanagari, Bengali, or CJK text.
Flutter assets and Web WOFF2 files should be bundled; runtime font fetching is
not required.

### 6.2 Type scale

| Token          | Size / line                    | Weight | Use                               |
| -------------- | ------------------------------ | ------ | --------------------------------- |
| `display`      | `40/44` desktop, `32/36` phone | 650    | rare empty/onboarding statement   |
| `title-1`      | `28/34` desktop, `24/30` phone | 650    | page title                        |
| `title-2`      | `21/27`                        | 650    | major section                     |
| `title-3`      | `17/23`                        | 600    | card/field title                  |
| `body`         | `16/25`                        | 400    | chat and primary explanatory text |
| `body-compact` | `14/21`                        | 400    | settings, workbench rows          |
| `label`        | `13/18`                        | 600    | buttons, controls                 |
| `meta`         | `12/17`                        | 500    | timestamps and state support      |
| `code`         | `13/20`                        | 450    | code/logs; phone may remain 13px  |

Technical IDs use `dir="ltr"` and `unicode-bidi: plaintext`/directional isolation,
not a whole LTR container. Model IDs, URLs, hashes, ports, tokens, and code remain
selectable. Truncation is middle or end based on value type, with the full value
available through copy/tooltip or an expanded row.

## 7. Shapes, radii, borders, and depth

### 7.1 Radius tokens

- `r-2 = 4px`: tension corner, tiny status markers.
- `r-1 = 6px`: tooltip, code controls, compact badges.
- `r-control = 8px`: inputs, buttons, nav rows.
- `r-card = 14px`: ordinary cards and fields.
- `r-feature = 18px`: composer, hero control, sheets/dialogs.
- `r-sheet = 24px`: mobile sheet top corners only.
- pill (`999px`): only tags, status capsules, switches, and circular icon buttons.

Large surfaces use the logical tension corner: block-start inline-end is `4px`,
the remaining corners use their token. Do not apply a `16px` radius to every
element.

### 7.2 Borders

- ordinary boundary: `1px border`;
- active/focus geometry: `2px`, never layout-shifting (outline or inset);
- separators start after the leading icon/title when rows belong to one group;
- no border plus shadow plus tinted background unless the surface is transient;
- code/log dividers use stronger neutral borders, not product accent.

### 7.3 Elevation

| Level | Treatment                       | Use                      |
| ----- | ------------------------------- | ------------------------ |
| 0     | no shadow                       | canvas, sunken fields    |
| 1     | `0 1px 0 rgba(18,25,24,.06)`    | cards, dock              |
| 2     | `0 8px 24px rgba(18,25,24,.10)` | sticky composer, popover |
| 3     | `0 18px 50px rgba(9,13,12,.18)` | dialog/sheet only        |

Dark mode uses borders and small ambient shadows; it does not create lighter
"glowing" edges. Blur/backdrop-filter is not required for identity and should be
avoided except for a modest opaque sticky-bar fallback.

### 7.4 Component recipes

| Component         | Concrete GRIPLINE recipe                                                                                                                                              |
| ----------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ordinary card     | Surface, `1px` Border, `r-card`, level-1 shadow at most; `16px` phone / `18px` desktop padding                                                                        |
| feature card      | Surface, tension corner, optional logical-start Gripline only when selected/active                                                                                    |
| primary button    | Tidal fill, verified contrasting label, `r-control`, `40px` desktop / `48px` touch height                                                                             |
| secondary button  | Raised Surface, strong border on hover, Ink label; no accent wash                                                                                                     |
| ghost/icon button | transparent at rest, `6%` Ink hover field; icon always has accessible name                                                                                            |
| input/textarea    | Surface, `1px` Border Strong, persistent label, `12px` inline padding; focus ring outside, error message below                                                        |
| segmented control | one Sunken container and equal/intrinsic segments; selected segment uses Surface + Gripline/weight, not color alone; wraps or becomes a select before labels truncate |
| badge             | `24px` minimum height, `r-1` or pill only for categorical metadata; icon/label required for state                                                                     |
| dialog            | max `560px`, level-3 depth, tension corner, labelled title/description, actions logical end and stacking on narrow widths                                             |
| side sheet        | `min(92vw, 480px)` Web and platform-appropriate Flutter width; logical-side entry, fixed heading/actions, scrolling body                                              |
| mobile sheet      | full width, `r-sheet` block-start corners, drag handle only when it corresponds to real dismissal behavior                                                            |
| tooltip           | Raised Surface, `r-1`, `12px` text, `240px` max width, `400ms` pointer delay and immediate keyboard display                                                           |
| loading           | structural skeleton matching final layout; spinner only inside the control that initiated a short action                                                              |
| empty state       | `40px` mark/semantic icon, `title-2`, one explanatory sentence, one primary recovery action; maximum `520px`                                                          |
| inline notice     | semantic soft field, `3px` logical-start rule, state icon, title and actionable copy; never color alone                                                               |

## 8. Motion

### 8.1 Timing

- `instant = 80ms`: pressed feedback, icon swap.
- `quick = 140ms`: hover, selection, tooltip exit.
- `standard = 220ms`: sheet, dialog, collapsible section.
- `emphasis = 320ms`: first empty-state entrance only.
- curves: enter `cubic-bezier(.2,.8,.2,1)`; exit `cubic-bezier(.4,0,1,1)`.

### 8.2 Motion vocabulary

- Selection slides the Gripline `6–10px` along the logical block axis while
  content cross-fades; no bouncing icons.
- Sheets translate from their actual logical side. Dialogs fade and rise `6px`.
- Streaming uses a tracer confined to the message Gripline and a static
  "Responding" label. Avoid three unrelated pulsing dots plus shimmer plus spin.
- Skeletons use a single low-contrast sweep. More than six skeleton rows switch
  to static placeholders to reduce visual noise.
- Completion changes icon and label once; no confetti.
- Error uses no shake. The field border and inline message appear together.

`prefers-reduced-motion: reduce` and Flutter accessibility motion settings make
all layout transitions immediate, remove tracers/shimmers, and use opacity no
longer than `80ms`. Progress remains understandable through text and icons.

## 9. Flutter direction

### 9.1 Main shell

Phone uses a `64–72dp` bottom **instrument dock** with four existing destinations:
Status, Web, Logs, Settings. Each item has a `48dp` minimum target and always shows
a localized label at normal text scale. Selected state uses a Tidal Gripline at
the top of the item, a filled icon variant, and semibold label—never a floating
white tile on a dark slab.

At tablet/desktop widths (`≥840dp`), the same destinations move to a `88dp`
logical-start rail with labels. This is a layout adaptation only; selected index,
unsaved-settings handling, and Web route handoff remain unchanged.

Page changes use a restrained shared-axis transition (`220ms`, `8px`), disabled
for reduced motion. The current IndexedStack state retention is preserved.

### 9.2 Status / main control

Replace the oversized generic RUN presentation with an **Agent Core panel**:

- page title: "PocketClaw"; supporting label: "Agent Core";
- state row: shaped state icon, explicit Running/Starting/Stopped label, and a
  one-line explanation;
- primary start/stop control: `52dp` high full-width phone button, content-width
  desktop button; Stop is not red until the running state makes it destructive;
- endpoint shown in a selectable monospaced value well with Copy and QR actions;
- Local/Public Mode shown beside the endpoint with lock/globe icon and text;
- warning for unavailable LAN address stays inline with the affected endpoint;
- help hint is attached to the control it explains, not centered as orphan text.

The current start, stop, public mode, QR, endpoint, and runtime behavior is not
changed.

### 9.3 Settings information architecture

Use one scrolling page with four Pocket Fields. Each field has a section heading
outside and grouped rows inside:

1. **Agent:** Models, Context Memory.
2. **Connections:** Telegram, GitHub, Public Mode.
3. **Runtime:** Auto Start, gateway address/port, managed runtime/binary fields,
   advanced arguments where currently supported.
4. **PocketClaw:** Theme, language, What's New, About, device feedback where
   currently present.

Rows are `56–72dp` high depending on description. Navigation rows use a logical
end chevron; toggles stay logical end; status appears as icon + text before the
control. Avoid one floating card per setting.

Specific surfaces:

- **What's New:** version and unread lozenge in the row. The destination remains
  a full reading page. Release sections use a Gripline and plain bullets.
- **About:** Pocket Hinge at `48dp`, product/app version first, engine attribution
  in a distinct "Components" section. Attribution is factual, not product
  branding.
- **Telegram:** paper-plane service icon plus Connected/Not connected only when
  authoritative. The existing setup launcher remains the explicit entry point.
- **Context Memory:** short explanation, current number, wrapping segmented
  choices (`10/15/20/25/Custom`), and a separate custom input row. Do not save on
  mere typing. Existing optimistic/revert behavior stays.
- **GitHub:** account name and connection state, never token content. Connect is
  primary only while disconnected; Test and Disconnect become secondary and
  destructive-secondary actions.
- **Models:** opens the canonical Web route exactly as today. Show no locally
  cached model state in the Flutter row.
- **Public Mode:** lock/globe state language. Enabling uses an amber explanation
  field because network reachability changes; no new confirmation is proposed
  unless existing behavior already has one.
- **Auto Start:** service and gateway remain separate switches with explicit
  current runtime subtext. Their visual grouping must not imply a dependency the
  implementation does not enforce.

### 9.4 Telegram onboarding

Keep the existing state machine and return/resume behavior. Restyle it as a
compact vertical stage track:

- intro: service mark, two-sentence value statement, primary Open Telegram,
  secondary Set up manually;
- awaiting: QR in a high-contrast white well, visible "Waiting for confirmation"
  status, expiry if already available, and cancel/retry actions as currently
  supported;
- configuring: one determinate label per real stage, no invented percentages;
- connected: success glyph, bot identity only if available, Done action;
- expired/failed: non-color icon, precise recovery action, details without secret
  values.

### 9.5 Forms, buttons, and focus

- phone input/button height: minimum `48dp`; compact desktop: `40dp`;
- helper/error text remains below its field and participates in layout;
- switch row target is the full row where behavior already allows it;
- D-pad/keyboard focus: `3dp Focus Blue` outer ring + `2dp` surface gap, visible
  on cards, nav, chips, icon buttons, and links;
- focus never relies on hover, scale, or color fill alone;
- traversal order follows visual/logical order in LTR and RTL;
- Enter/center activates, Escape/Back dismisses the top transient surface;
- at `200%` text scaling, settings become taller rows and actions wrap below
  content rather than ellipsizing the setting name.

## 10. Web direction

### 10.1 Application frame

Desktop (`≥1024px`) uses a `248px` logical-start navigation frame and a separate
content plane. The top of the frame contains the `28px` Pocket Hinge and wordmark.
The gateway state module sits at the bottom of the navigation frame, so runtime
control is persistent without competing with every page title.

The content header is `64px`, non-glass, and contains page title, contextual
actions, language, and theme. Restart-required is an amber labeled action, not a
color-only icon. Global logout remains available only where current launcher
authentication provides it.

At `768–1023px`, navigation becomes a `72px` icon rail with tooltips and a clear
expand control. At `<768px`, it becomes the Menu drawer described in section 12.

### 10.2 Navigation grouping

Keep routes and capabilities; clarify their mental model:

- **Talk:** Chat.
- **Intelligence:** Models, Credentials.
- **Connections:** Channels, with configured/recent channels first only if that
  information is already available; Telegram remains a normal channel route.
- **Capabilities:** Hub, Skills, Tools.
- **System:** Config, Logs.

Groups are not individual floating accordions on desktop. Labels are quiet
`12px` section markers; only long channel lists collapse. Selected rows use a
logical-start Gripline, Tidal-soft field, filled icon, and current-page semantics.

### 10.3 Page archetypes

Avoid a universal grid of shadcn cards. Use four intentional archetypes:

- **Conversation:** Chat reading plane and composer.
- **Workbench:** Models, Skills, Tools, Hub—toolbar plus dense responsive content.
- **Configuration:** Channels, Credentials, Config—section index on wide desktop,
  single scrolling column on mobile.
- **Telemetry:** Logs—monospaced stream, filter bar, pause/clear actions.

Concrete page direction:

- **Channels / Telegram:** connection summary first, then grouped configuration.
  Secret fields never reveal stored secrets. Descriptions live with their field.
- **Credentials:** provider rows with authentication method, authoritative state,
  and one primary action. Device-code flows use a focused sheet with copyable
  code and clear waiting state.
- **Hub:** search is a workbench toolbar, not a giant marketing hero. Results use
  capability/source metadata and explicit install state.
- **Skills:** list is the default for scanability; optional grid is denser than
  marketplace marketing cards. Origin, enabled/installed state, and actions have
  stable columns on desktop and stacked metadata on mobile.
- **Tools:** provider groups become execution-capability rows. Enablement state is
  labeled; web search settings remain a tab/section without changing behavior.
- **Config:** human-readable settings first and raw configuration as a clearly
  technical secondary view. Restart/apply state stays visible near Save.
- **Logs:** near-black neutral well in both themes, minimum `13px` code text,
  line-height `20px`, level icon/label, wrap toggle, search/filter, and no colored
  full-row backgrounds except selected search result.
- **Language selector:** globe + current language name at desktop; globe icon with
  accessible name at narrow widths. Menu items show native language names, check
  current choice, and have `44px` rows.

### 10.4 Wide displays

Content does not stretch line lengths. Above `1440px`, workbenches may add a
`280px` contextual inspector for the selected item; Chat remains centered and
uses surrounding space for subtle rails or an existing detail popover—not new
data. Above `1920px`, outer margins grow; core controls do not become wider.

### 10.5 Responsive composition matrix

| Width / environment                                | Navigation                                        | Content behavior                                                                       |
| -------------------------------------------------- | ------------------------------------------------- | -------------------------------------------------------------------------------------- |
| `320–359px` narrow WebView                         | visible hamburger + localized Menu; `88vw` drawer | `12px` gutters, stacked actions, two-row composer metadata, no icon-only Menu          |
| `360–479px` common phone                           | same Menu/drawer                                  | `16px` target gutters where space permits; user message max `88%`; forms/actions stack |
| `480–767px` large phone / narrow WebView landscape | same Menu/drawer                                  | two-column layout only for short metadata; primary reading/forms remain single column  |
| `768–1023px` tablet                                | `72px` rail, expandable when space permits        | workbench may use two columns; forms remain max `720px`; sheets stay side-aware        |
| `1024–1439px` desktop                              | expanded `248px` navigation                       | workbench up to `1120px`; Chat `840px`; contextual actions in page header              |
| `1440–1919px` wide desktop                         | expanded navigation                               | workbench up to `1280px`; optional existing-data inspector; reading width unchanged    |
| `≥1920px` very wide desktop                        | expanded navigation                               | larger outer margins only; controls and prose do not stretch                           |

Height and input method also participate: a landscape phone with a software
keyboard prioritizes message history plus composer; a pointer tablet may use the
rail but keeps touch-size targets. Breakpoints never assume English-length labels.

## 11. Chat direction

Chat is the hero surface and the clearest expression of GRIPLINE.

### 11.1 Message hierarchy

- **Agent messages:** editorial, full reading-column width, no bubble. A `28px`
  Pocket Hinge sits in a narrow logical-start identity rail. During streaming,
  that rail becomes the active Gripline. Body text is `16/26`.
- **User messages:** compact surface aligned logical end, maximum `78%` desktop
  and `88%` phone, Coral logical-end edge, `14px` card radius with the tension
  corner. Long user code/URLs still wrap or scroll safely.
- **System/notices:** centered only for session boundaries; otherwise inline
  status fields with icon and label.
- Timestamps/actions appear on focus, hover, or message menu but remain reachable
  by keyboard and touch. Hover is never the sole access path.

### 11.2 Long technical content

- prose width max `76ch`;
- headings use the type scale, not giant marketing sizes;
- paragraph spacing `12px`, section spacing `24px`;
- lists retain visible nesting and `24px` minimum logical indent;
- tables sit in a labelled horizontal scroll region on narrow screens, preserve
  row/column headers, and do not shrink below readable text;
- URLs and model IDs use `overflow-wrap:anywhere` only in prose; code retains
  fidelity in a scroll/wrap-controlled well;
- copy feedback is text + check icon and is announced politely.

### 11.3 Code

Code uses Surface Sunken with a neutral strong border. Header contains language,
Copy, Wrap, and Collapse; buttons are at least `36px` desktop / `44px` touch.
Line numbers are optional at very narrow widths and remain non-selectable. Syntax
colors are tested in both themes and are not reused as status colors. A code block
never forces the whole page wider than the viewport.

### 11.4 Tool-call presentation: the Execution Ledger

Tool calls are visible work, not chat bubbles and not chain-of-thought.

Each call is one ledger row:

- tool icon and human-readable tool name;
- explicit state: Queued / Running / Complete / Failed, based only on available
  protocol state;
- a Gripline tracer only while running;
- collapsed one-line summary when one is already provided;
- expandable Arguments and Result wells containing the data already shown by the
  product;
- retry/copy actions only if existing behavior supports them;
- duration only when real timing data exists.

Multiple calls become a single grouped ledger under the relevant agent message.
Failures stay expanded enough to show the error and recovery path. Hidden model
reasoning, private prompts, and unprovided intermediate thoughts are never
displayed or inferred.

### 11.5 Composer

The composer is an anchored **control deck**, not a floating glowing pill:

- max width `900px`, Surface Raised, `1px` border, tension corner, level-2 shadow;
- multiline input starts at `52px`, grows to `200px`, then scrolls;
- attachment action at logical start and Send at logical end;
- while streaming, Send becomes a square Stop control with label/tooltip and the
  same target size; it does not move to a new location;
- secondary deck row contains model selector, context indicator, and attachment
  state; it wraps into two rows on narrow screens;
- selected model shows a readable alias, with full model ID available in its menu;
- context is a labelled meter (`Context 42%`), not an unexplained tiny ring. At
  higher thresholds it adds warning icon/text, not only a new color;
- disabled send includes an accessible reason when no model or connection is
  available.

Attachments appear above the input as removable file tiles: thumbnail where safe
and already available, otherwise type icon, filename, size if known, and a `44px`
remove action. Image routing/security behavior is unchanged.

### 11.6 Empty and streaming states

Empty Chat shows the mark at `40px`, "What should we work on?", one sentence about
local control, and 3–4 compact example tasks derived from actual capabilities.
It occupies the useful center, not an enormous blank hero.

Streaming shows the active Gripline, "Responding", and stable partial content.
Do not animate the entire message opacity. Preserve scroll position when the user
has moved upward; use a "Jump to latest" control instead of forced autoscroll.

### 11.7 Mobile behavior

- `12–16px` page gutters; messages never rely on side-by-side metadata;
- composer accounts for safe-area and virtual keyboard, and remains part of the
  viewport without covering the last message;
- message actions open from a clear More button or long-press with an accessible
  alternative;
- model/context row wraps and never horizontally scrolls as a hidden toolbar;
- tool ledger and code stay within the message width;
- the visible Menu control remains available at the page header and is not
  confused with session history.

## 12. Mobile navigation

### 12.1 The Menu control

At `<768px`, replace the ambiguous panel-only trigger with an explicit button:

`[hamburger icon]  Menu`

Specifications:

- `44px` minimum height, target width approximately `88px` in English; padding,
  not a fixed width, accommodates translation;
- label uses localized `common.menu`; English value is "Menu" and is not forced
  uppercase in scripts where that is inappropriate;
- `aria-label="Open menu"` when closed and `aria-expanded`; drawer close control
  is an X with `aria-label="Close menu"`;
- button is at logical start: left in LTR, right in RTL;
- drawer opens from the same logical side and keeps existing open/close behavior;
- hamburger is three equal horizontal lines, not a sidebar/panel glyph;
- mark-only brand sits at the opposite side only when header width permits;
- `44px` row targets and visible section labels inside the drawer;
- focus moves to the drawer heading on open, is trapped while open, Escape closes,
  and focus returns to Menu;
- selecting a route closes the drawer exactly as current mobile navigation does;
- backdrop click behavior remains current unless accessibility testing requires a
  bug fix during implementation.

On very narrow widths (`<360px`) the action area may collapse language/theme into
an overflow menu, but the visible Menu button never collapses to icon-only.

### 12.2 Drawer composition

Width is `min(88vw, 360px)`. Header contains Pocket Hinge, PocketClaw wordmark,
and Close. Body contains the same grouped routes as desktop. Footer contains
gateway state/action and language/theme; critical state is not hidden above the
scroll region. Safe-area insets are respected.

## 13. Desktop navigation

Desktop keeps a separate collapse affordance because it controls density, not
main-menu discovery:

- expanded `248px`; collapsed `72px`;
- collapse button is a rail-with-arrow glyph on the navigation edge with tooltip
  "Collapse navigation" / "Expand navigation";
- keyboard shortcut remains only if already supported and is documented in the
  tooltip; do not overload the mobile Menu label;
- selected item Gripline sits on logical start and spans `24px` vertically;
- icon `20px`, row height `40px`, group gap `20px`;
- labels wrap only where meaningful; route names do not ellipsize under ordinary
  translated strings in expanded mode;
- collapsed mode shows accessible tooltips and retains group separators;
- channel overflow uses one explicit "More channels" row; it does not hide the
  selected channel when collapsed.

## 14. Models UX

Models is a routing workbench rather than a catalog of generic cards.

### 14.1 Current screen structure

1. **Routing** Pocket Field at top.
   - `Default model` route card with provider, model alias/ID, auth/availability
     state where known, Change action, and Default label.
   - `Fallback models` ordered chain directly below. Each row has order number,
     drag handle where existing behavior supports reorder, provider/model, and
     remove. Explain that order matters.
2. **Configured models** grouped by provider in compact sections.
3. Toolbar: Add model, provider/catalog discovery, search/filter if supported.

Setting a default remains the existing explicit action and apply/restart path. A
visual optimistic state must not claim the runtime applied a model until the
existing save/apply response confirms it.

### 14.2 Provider/model rows

- provider mark or stable letter tile, never runtime-fetched favicons;
- friendly label first; technical model ID in mono and directional isolation;
- type/auth/status badges are text plus icon;
- Edit, Test, Delete live in an overflow/action area with `44px` mobile targets;
- default model has Gripline and a labelled pin, not a star alone;
- destructive delete confirms the exact model ID and effect on routing;
- sheet on phone, side sheet up to `480px` on desktop, dialog only for small
  confirmations/tests.

### 14.3 Future Vision / Image routing accommodation

Do **not** add a field, config key, placeholder card, or fake capability now.
Design the routing field as an ordered list of role-based `RoutingRoleCard`
components rather than hard-coding a single hero card. A future release can insert
`Vision / Image model` between Default and Fallback without changing the page
grammar:

`Default model → [future Vision / Image model] → Fallback models`

The future role would use the same provider/model picker and capability/status
language. It must only appear when its backend/config behavior exists. The current
concept does not route attachments, infer model vision support, or alter fallback
behavior.

## 15. RTL / localization

Arabic is a first-class layout mode, and all current locales remain supported.

- Use CSS logical properties (`padding-inline`, `margin-inline`, `inset-inline`,
  `border-inline-start`) and Flutter `EdgeInsetsDirectional`, `AlignmentDirectional`,
  `BorderRadiusDirectional` or a custom logical ShapeBorder.
- Navigation and drawer move to logical start in RTL. The tension corner and
  Gripline mirror with them.
- Directional arrows mirror only when they mean previous/next, enter/exit, or
  spatial movement. Play, refresh, power, check, attachment, external brand marks,
  and media controls do not mirror.
- A chevron meaning "open details" points toward the destination's logical
  direction and is verified in both modes.
- Do not use hard-coded `left/right` for message alignment, drawer side, focus
  rail, tooltips, or sheet placement.
- User/agent hierarchy uses logical start/end; it does not assume user is always
  physically right.
- Model IDs, URLs, versions, code, hashes, IPs, and ports use `dir=ltr` plus bidi
  isolation on the value itself. Punctuation outside stays in paragraph direction.
- Controls use intrinsic size and wrapping. No fixed label widths. Primary action
  groups wrap or stack before text truncates.
- Test pseudo-localized strings at `1.4×` length, Arabic at `200%` text scale, and
  mixed strings such as `نموذج llama-3.2/vision:latest`.
- Uppercase is restricted to real acronyms. Section labels use weight/size, not
  transformed case.
- Icons with text keep logical `8px` spacing regardless of direction.

## 16. Accessibility

### 16.1 Visual

- Target WCAG 2.2 AA: `4.5:1` normal text, `3:1` large text and UI boundaries;
  critical body combinations above are already specified higher.
- `44 × 44px` Web and `48 × 48dp` Flutter touch targets for primary/mobile
  controls; compact pointer-only controls remain at least `32px` with expanded
  hit target when safe.
- Focus Blue is deliberately separate from Tidal selection. Focus remains visible
  on a selected item.
- Status always has text and icon/shape. Charts/meters have numeric labels.
- Dark mode is a remapped palette, not light mode inverted. Syntax and semantic
  palettes are audited separately.

### 16.2 Input and navigation

- All actions are reachable by keyboard and Android D-pad in logical order.
- No positive `tabindex`. Skip link targets main content in Web.
- Drawer/dialog/sheet traps focus, labels its heading, closes with Escape/Back,
  and returns focus to its invoker.
- Tooltips supplement visible/accessibility labels and are not required to
  understand mobile icon buttons.
- Drag reordering has Move up/Move down keyboard and screen-reader alternatives.
- Hover-only message actions have visible focus and touch equivalents.

### 16.3 Semantics and announcements

- Use one `h1` per Web page, ordered headings in long assistant answers, landmark
  labels for navigation/main/composer/log stream.
- Streaming container uses restrained `aria-live`; do not announce every token.
  Announce start, tool-state changes, completion, and errors at meaningful
  boundaries.
- Save/apply, copied, connection, and upload state use polite live regions; errors
  are associated with the affected field.
- The mark's image alt is "PocketClaw" only when it is the sole product label;
  beside the wordmark it is decorative.
- Inputs have persistent labels, not placeholder-only labels.

### 16.4 Scaling and motion

- Web supports browser zoom to `200%` without lost actions or two-dimensional page
  scrolling at `320 CSS px`.
- Flutter supports system text scaling; rows grow and action clusters stack.
- Reduced-motion behavior is defined in section 8. No required information is
  communicated only during an animation.

## 17. Visible PicoClaw branding removal plan

The production source at the audited base still has a concrete Web identity leak:
`AppHeader` renders `/logo_with_text.png`, and that image visibly contains the old
PICOCLAW lobster wordmark. The browser icon family also uses the old lobster.

Implementation must use a targeted, user-visible asset replacement—not a global
rename.

| Production surface                          | Current source                                                                                   | Required treatment                                                                                                           |
| ------------------------------------------- | ------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------- |
| desktop Web header                          | `core/src/web/frontend/public/logo_with_text.png`, referenced by `src/components/app-header.tsx` | remove reference and asset; render Pocket Hinge + PocketClaw lockup                                                          |
| mobile Menu drawer                          | currently no coherent drawer brand block                                                         | add Pocket Hinge + PocketClaw; no upstream mark                                                                              |
| page title                                  | `core/src/web/frontend/index.html`                                                               | retain PocketClaw and verify launcher/OAuth pages                                                                            |
| SVG favicon                                 | `core/src/web/frontend/public/favicon.svg`                                                       | replace embedded old raster with simplified Pocket Hinge vector                                                              |
| ICO/PNG favicons                            | `favicon.ico`, `favicon-96x96.png`                                                               | regenerate from Pocket Hinge master                                                                                          |
| PWA/touch icons                             | `apple-touch-icon.png`, `web-app-manifest-192x192.png`, `web-app-manifest-512x512.png`           | replace old lobster exports                                                                                                  |
| PWA metadata                                | `site.webmanifest`                                                                               | set `name`/`short_name`, theme/background colors, accessible PocketClaw identity; verify icon entries                        |
| built embedded dashboard                    | `core/src/web/backend/dist/*`                                                                    | rebuild from reviewed frontend so source and embedded runtime contain the same new assets; do not hand-edit minified bundles |
| launcher/login/setup/OAuth browser surfaces | frontend routes and backend completion templates                                                 | inspect titles, visible marks, alt text, and headings; PocketClaw product identity only                                      |
| Flutter launcher/splash                     | current detailed PocketClaw raster and Android resources                                         | replace with Pocket Hinge adaptive/full-color exports; retain PocketClaw label                                               |
| Android notification                        | `ic_stat_pocketclaw.xml`                                                                         | replace glyph only after 24dp alpha-mask review                                                                              |
| desktop tray/process icon                   | `assets/icon.ico`, `assets/app_icon.png` fallbacks                                               | replace user-visible old/detailed identity with Pocket Hinge size-specific exports                                           |
| image semantics                             | Web `header.logoAlt` locale strings and Flutter semantics                                        | say PocketClaw, or mark decorative beside visible wordmark                                                                   |

Release audit:

1. Search source and built user-visible assets for old wordmarks/marks.
2. Decode/visually inspect every shipped icon size from the reviewed exports.
3. Verify browser tab, install prompt, saved-to-home-screen, Web header, drawer,
   splash, launcher, notification, and tray identity.
4. Check cache invalidation/service-worker behavior if applicable so an old icon
   is not retained after upgrade.
5. Preserve legal attribution in licenses/notices and factual engine attribution
   in About.

Explicitly retain compatibility-sensitive internals unless separately migrated:
`libpicoclaw.so`, `libpicoclaw-web.so`, Go module/package names, native service
class names, MethodChannel values, `PICOCLAW_*` environment keys, old storage
keys, DOM compatibility attributes, OAuth postMessage types, existing routes,
workspace/config paths, and legacy configured protocol identifiers. No global
search-and-replace is allowed.

## 18. Mobile Menu solution

The production `AppHeader` passes a hamburger child to `SidebarTrigger`, but the
current `SidebarTrigger` implementation renders its own panel/sidebar glyph and
does not render that child. It also exposes only a visually hidden "toggle
sidebar" label. That explains why the control reads as a sidebar toggle instead
of main navigation.

Implementation direction:

1. Split the abstractions into `MobileMenuButton` and `DesktopNavCollapseButton`.
2. `MobileMenuButton` directly renders a true hamburger plus visible localized
   Menu label; minimum `44px` height, `aria-controls`, and `aria-expanded`.
3. It calls the existing `toggleSidebar()` / `setOpenMobile()` path. Drawer route
   click continues calling `setOpenMobile(false)`.
4. `DesktopNavCollapseButton` renders the panel/rail glyph and its own collapse
   label. It is never shown on mobile.
5. Place Menu at logical start and let the existing direction resolver open the
   sheet on the matching side.
6. Add a visible Close action in the drawer while retaining focus trap, Escape,
   outside-click, and route-change behavior.
7. Add translations for `menu`, `openMenu`, and `closeMenu` to every shipped Web
   locale. Do not use the word "sidebar" in user-facing mobile copy.
8. Test LTR and RTL icon/label order, long translations, `320px` width, text zoom,
   keyboard focus return, and touch target geometry.

This is a presentation/semantics split around the current behavior, not a new
navigation state machine.

## 19. Implementation strategy

This proposal should be implemented in reviewable layers after design approval.

### Phase 0 — invariants and visual baselines

- Record existing route map, config/save/apply flows, service state transitions,
  session behavior, and component tests.
- Add semantic tests for product name, mobile Menu accessible name, drawer side,
  and old visible asset references.
- Inventory source assets and generated/built copies. Mark internal identifiers as
  explicitly protected.

### Phase 1 — identity package

- Produce/review Pocket Hinge SVG master, one-color form, lockup, and size tests.
- Export Web/PWA/Android/notification/tray assets from the master.
- Replace visible Web header and icon surfaces, Flutter launcher/splash, and tray
  assets with targeted edits.
- Rebuild embedded Web distribution through its normal pipeline.

### Phase 2 — shared tokens

- Web: replace generic shadcn color variables with semantic GRIPLINE tokens;
  keep primitives but change their recipes and page compositions.
- Flutter: consolidate the competing theme paths into one token-backed
  `ThemeExtension`/component theme family while preserving the current user theme
  behavior until a product decision explicitly changes it.
- Bundle script-aware open fonts and verify package/license impact.
- Add reusable `GripPanel`, `StateLabel`, `TechnicalValue`, and `FocusFrame`
  primitives in both stacks.

### Phase 3 — shell and navigation

- Implement Web desktop frame, mobile Menu split, responsive drawer, and language
  selector semantics without changing routes.
- Implement Flutter instrument dock/rail without changing destination indexes,
  IndexedStack retention, WebView URLs, or unsaved-change behavior.
- Validate RTL, `320px` Web, common phone widths, tablet, desktop, and wide desktop.

### Phase 4 — hero surfaces

- Chat first: message hierarchy, Execution Ledger, code/tables, composer, context
  label, empty/streaming states.
- Models second: routing field, provider sections, sheets/dialogs. Keep existing
  default/fallback data and save/apply behavior.
- No future Vision/Image routing logic in this phase.

### Phase 5 — remaining Web workbenches

- Channels/Telegram, Credentials, Hub, Skills, Tools, Config, Logs.
- Replace page-level generic card grids with the appropriate archetype while
  keeping every API call and route stable.

### Phase 6 — Flutter owned surfaces

- Status, Settings groups, What's New, About, Telegram onboarding, Context Memory,
  GitHub, Models link, Public Mode, Auto Start, forms and dialogs.
- Preserve current platform-channel calls, safe restart path, and lifecycle logic.

### Phase 7 — verification gates

- Token contrast audit in both themes; syntax/status palette audit.
- Web unit/component tests, localization parity, keyboard and screen-reader pass.
- Flutter analyzer/unit/widget tests, D-pad/focus and text-scale widget tests.
- Responsive matrix: `320`, `360`, `393`, `412`, `600`, `768`, `1024`, `1440`,
  and `1920 CSS px`; portrait/landscape where relevant.
- RTL matrix includes drawer side, Gripline, tension corner, sheets, composer,
  message alignment, mixed-direction values, and arrow semantics.
- Asset audit includes source and built distribution. Any old visible PicoClaw
  identity blocks release.

Prototype mapping: the isolated `docs/design/prototype/concept-b.html` demonstrates
the identity, responsive navigation, Chat hero surface, Execution Ledger, composer,
model routing field, light/dark behavior, and RTL mirroring. It contains no
production imports or wiring and is not an implementation source.

## 20. Risks / functional boundaries

### Design risks and mitigations

| Risk                                                        | Mitigation                                                                                       |
| ----------------------------------------------------------- | ------------------------------------------------------------------------------------------------ |
| Gripline becomes decoration everywhere                      | restrict it to current selection, focus, or active operation; one per hierarchy level            |
| asymmetric tension corner feels gimmicky                    | use only on feature/transient surfaces; plain rows remain plain                                  |
| warm neutral canvas looks muddy                             | keep primary content on white/near-black Surface and preserve specified contrast                 |
| Coral is mistaken for error                                 | errors always use the separate Error red plus octagon/exclamation; Coral is never a status color |
| custom mark loses clarity at 16px                           | separate favicon simplification, pixel-grid review, no fold seam below 24px                      |
| script-specific fonts increase bundle size                  | subset per shipped locale, preload only current UI family, retain system fallbacks               |
| responsive workbenches become horizontal mini-desktops      | switch structure at breakpoints; stack metadata/actions before truncating                        |
| richer tool UI leaks private model data                     | render only existing public tool-call fields; never infer or expose reasoning                    |
| token migration changes behavior through component rewrites | separate styling/token commits from API/state changes and retain behavior tests                  |
| generated Web distribution diverges from source             | build from source and audit both; never patch minified files manually                            |

### Frozen behavior

This concept does not authorize changes to:

- Telegram setup/configuration logic or lifecycle handling;
- Context Memory semantics, limits, bounded context, or persistence;
- live apply, restart deferral, or provider routing;
- Managed Runtime, Python runtime, GitHub authentication/storage, or credential
  visibility;
- gateway networking, binding, public-mode security, or authentication;
- What's New seen/read behavior;
- chat/session creation, persistence, streaming protocol, or scroll privacy;
- attachment privacy/security behavior;
- Core APIs, Go modules, native binary names, environment variables, storage keys,
  routes, or protocol identifiers;
- dedicated Vision / Image Model routing. The design only leaves a compositional
  insertion point for a future, separately specified implementation.

Any design detail that appears to require one of those changes must be removed or
deferred, not silently implemented as part of the visual system.
