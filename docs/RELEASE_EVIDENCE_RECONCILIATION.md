# Release-evidence reconciliation

This record reconciles two counts that looked like regressions against older
release reports, states the architectural verdict on the Telegram ownership
probe, and records the design recommendation for making one Telegram
"configured" state authoritative. Both count differences are explained by a
documented change of scope, not by a lost or weakened check. It is evidence for
the release record, not a change to any gate.

## 1. Zero-Pico allowlist: 19 → 22 entries

**Verdict: the count grew because the scanner's scope and review were widened by
one commit, and the three added exemptions are all read-only migration/legal
material. No active Pico identity was introduced.**

### Old scope and count

The older baseline (2026-09-13, and stable from `8a60ef5` through `9017bb2`)
scanned **19 allowlist entries, all in use**. At that point
`is_owned_production()` covered `lib/`, `android/app/src/main/`, `tool/*.py`,
`core/*.sh`, the PocketClaw-authored Go packages (`pcruntime`, `coresource`,
`pid`, `channels/pocketclaw`), the dashboard (`core/src/web/...`) and a small
set of named files.

### Current scope and count

At `d4e7107` (*fix(core): retire active Pico runtime identities*) the same guard
was pointed at **11 additional runtime-path-owner files** in the vendored tree —
files the shipped product actively mints paths in, so an upstream-derived
spelling is not provenance once it reaches the device filesystem:

```
core/src/pkg/channels/wecom/reqid_store.go
core/src/pkg/channels/mqtt/mqtt.go
core/src/pkg/channels/weixin/media.go
core/src/pkg/agent/context.go
core/src/pkg/agent/prompt.go
core/src/pkg/media/tempdir.go
core/src/pkg/mcp/manager.go
core/src/pkg/providers/factory_provider.go
core/src/pkg/skills/clawhub_registry.go
core/src/pkg/tools/integration/skills_install.go
core/src/pkg/utils/download.go
```

Reading those files surfaced Pico-family strings that were previously outside the
scanner. Each was either fixed or claimed. The three that were claimed became new
allowlist entries: **19 → 22**. Two pre-existing entries were also *narrowed* or
*widened in wording only* (the Core-binary-name pattern was anchored so it cannot
match `picoclaw_…` identifiers, and two upstream patterns gained additional
upstream artifact names); neither changed the entry count.

### Entries added

| # | Pattern (abridged) | Category | Location | Why it is not an active identity |
| --- | --- | --- | --- | --- |
| A | `LegacyTempDirName = "picoclaw_media"` / `` `picoclaw_media` `` | `legacy_migration` | `core/src/pkg/media/tempdir.go` | The pre-migration media cache name, declared only so `RetireLegacyTempDir` can delete what an older build created. Anchored to the `Legacy` declaration and prose quoting it, **not** the bare string, so a regression of the live `TempDirName` would be caught. |
| B | `.picoclaw", "wecom"` / `picoclaw-wecom-reqid-store` | `legacy_migration` | `core/src/pkg/channels/wecom/reqid_store.go` | The two pre-migration WeCom request-ID store paths. Read only when the canonical store is absent; the canonical PocketClaw path is written and the legacy one deleted. |
| C | `Copyright (c) 2026 PicoClaw contributors` | `legal` | `core/src/pkg/providers/factory_provider.go` | The upstream copyright attribution. The product branding and the runtime `User-Agent` in the same file are PocketClaw. |

All three categories are on the allowed list (migration/read compatibility,
delete/cleanup of legacy data, legal/provenance). None creates a Pico path, emits
Pico branding, writes Pico config/defaults, advertises a Pico identity or
registers a Pico runtime identity.

### Justification for the remaining occurrences

The other 19 entries were already present and reviewed in the older baseline.
`python3 tool/no_active_pico.py` reports **22 allowlist entries, all in use**,
and `--public` reports no Pico branding on public surfaces outside the
attribution section. Both exit 0.

### Why the number must not be forced back to 19

An added exemption that stops matching fails the guard ("remove it"), and every
entry present matches. Deleting a live entry (A, B or C) does not remove the
string it claims; it only turns a reviewed exemption into an unexplained active
occurrence. The count is correct for the wider scope.

## 2. Native ELF audit: 194 PASS → 188 PASS

**Verdict: the two numbers describe two different scopes of the same tool. 194
includes the private native-support (debug symbols) manifest; 188 is the same
audit without it. Neither is a failure and no check was removed.**

### Arithmetic

`tool/native_elf_audit.py` builds its check list as:

- **180** per-entry checks: 18 packaged ELF entries × 10
  (`identity`, `load_alignment`, `nx_stack`, `no_wx_segments`, `no_textrel`,
  `no_runtime_search_path`, `no_debug_sections`, `no_static_symbol_table`,
  `build_path_privacy`, `relocation_hardening`).
- **+6** `required_exports` checks: `libapp.so`, three `libdartjni.so` (one per
  ABI) and the two Core executables (`libpocketclaw.so`,
  `libpocketclaw-web.so`).
- **+2** archive-level checks: `native.inventory` and
  `native.private_support_not_packaged`.
- **= 188** for an APK audited on its own.

Passing `--native-support-manifest` additionally runs `support_manifest_checks`,
which appends exactly **6** checks:

```
native.private_support_manifest
native.private_support_hashes
native.private_support_binding
native.private_support_symbolization
native.private_support_apk_binding
native.private_support_untracked
```

**188 + 6 = 194.**

### Which reports used which

- The **194** totals are the H5B / H5C / exposure-audit / production-candidate
  runs, which enforced the audit with `--enforce-target --native-support-manifest`
  against a **production/private** APK. The manifest binds that APK by SHA-256
  (`88ac3f39…`, 63,599,799 bytes) and verifies the private `.debug` companions.
- The **188** totals are the owner-round *verification* (development-signed) APKs.
  They do not bind the private support manifest — a dev APK has a different hash
  and signer — so the six support checks are absent. This is the correct scope
  for a non-publish audit.
- One historical line in `docs/prompts/history/EXPOSURE_AUDIT_RERUN.md` shows the
  command without the flag while reporting 194; the enforced invocation two
  paragraphs above it carries `--native-support-manifest`. It is a documentation
  abbreviation, not a second scope.

### Evidence from this sweep

```
$ python3 tool/native_elf_audit.py \
    --apk build/app/outputs/apk/release/app-release.apk \
    --manifest build/forensic/pc-def-052-native-audit.json
...
TOTAL: 188 PASS / 0 FAIL / 0 SKIP
```

18 entries were inventoried (`native.inventory PASS`). No expected ELF target is
missing: the packaged set is exactly `EXPECTED_ELF_ENTRIES` in the tool.

### Conclusion

The 194 → 188 difference is a scope difference, fully accounted for by the
private-support manifest's six checks. There is no missing native audit target.

## 3. Should the speculative `getUpdates` ownership probe remain?

**Verdict: keep it. It is not destructive, and the alternative cannot deliver
the one thing it exists for — a pre-commit rejection. One bounded guard is
recommended and deliberately not implemented in this pass.**

### What the probe actually is

`validateTelegramCredentials` issues `getMe`, then a non-destructive
`getWebhookInfo`, then one `getUpdates` carrying `{"limit":1,"timeout":0}` and
**no offset**. Telegram returns updates starting from the earliest unconfirmed
one and confirms an update only when a later `getUpdates` carries an offset
higher than its `update_id`, so this call returns pending updates without
confirming or dropping any. A negative offset would have done the opposite —
Telegram documents that it retrieves from the end of the queue and forgets all
earlier updates — which is why `offset=-1` was removed at `0535740`.

It is **not** a duplicate of the runtime's own first poll. The runtime's first
`getUpdates` is telego's, made short by `firstGetUpdatesRequest` setting
`timeout=0` on the first request only; the steady 30-second poll is unchanged.
Nothing in the live ownership path carries an offset parameter at all.

### Why not option B (ownership only through real runtime intake)

The probe's purpose is that a candidate owned by another service is rejected
**before any configuration mutation**, so the previously committed bot is
untouched and stays authoritative. Option B cannot do that: establishing
ownership through real intake means writing the candidate over the committed
configuration first and rolling back on failure. Rolling back a Telegram
credential is not free — the old token must be re-saved and re-applied, and the
managed service delivers a token exactly once — so the rollback window is a
window in which the install has neither bot live. The current shape trades that
for one short, non-confirming HTTP call.

### The residual, and the recommended guard

The probe is a real `getUpdates` consumer, and Telegram allows one. If
PocketClaw's own runtime is already long-polling the **same** bot, the probe
collides with itself: Telegram answers one of the two with 409, and the
runtime's 409 handling is terminal with no retry, so the running channel would
be revoked and retired while the candidate is rejected as `bot_in_use`.

Reachability is narrow. `validateTelegramCredentials` runs only from
`writeTelegramCredentialsContext`, whose two callers both deliver a token from
the managed pairing service, and a replacement (bot B ≠ bot A) cannot collide.
It requires the service to hand back a bot this install is already polling,
which is behaviour outside this repository.

**Recommended, not implemented:** skip step 3 when the candidate token is
identical to the currently committed token *and* the gateway reports Telegram
running. A token already committed and polling has already proved ownership, so
the probe buys nothing there and is the only thing that can cause the collision.
It is left out of this pass because the collision has not been demonstrated —
committing a fix for an unproven defect is the thing the review rule forbids —
and because the condition needs a device test of its own.

### The other residual: a conflict that appears after a passing probe

A bot that is free at probe time can be claimed by another service before the
runtime's first real intake. This is surfaced, not hidden: the runtime 409 is
terminal, the generation is retired, readiness reports `telegram_conflict` with
`webhook_active` or `bot_in_use` and generation 0, and both the managed-connect
flow and the connected card show an actionable, subtype-specific message. Note
what is *not* true of this case: the candidate passed validation, so it was
legitimately committed and the previous bot's token is gone. "The previous bot
stays recoverable" is a contract about *rejected* candidates, and it holds
exactly there.

## 4. Telegram "configured" authority — design recommendation for a later pass

**Not changed in this pass, and not required by either runtime-integrity fix.**
PC-DEF-071 made the contradiction honest; it did not merge the definitions.
This is the recommendation for doing that deliberately, later.

### The problem, stated once

Three places answer "is Telegram configured?" and they do not agree:

| Answerer | Definition | Source |
| --- | --- | --- |
| `channel-config-page.tsx` `isConfigured` | the `token` field of the **in-memory edit buffer** is non-empty | unsaved UI state |
| `telegram_readiness.go` `telegramIsConfigured` | `channel.Enabled` **and** a token **on disk** | persisted config |
| `telegramOwnerMissing` | the above **and** exactly one positive numeric owner | persisted config |

The frontend picks its whole surface (`connected` / `managed-onboarding` /
`manual-only`) from the first, then renders authoritative state from the second.
They disagree for a channel holding a token with `enabled: false`, and for the
edit buffer before a save — which is why the connected card could show a
starting spinner above "No bot is configured."

`clearTelegramCredentials` already clears token, owner and `enabled` together
precisely to avoid leaving that state behind, and its comment says why: *"a
disabled channel that still holds a token and an owner reads as connected to
every surface that asks."* That is the invariant; it is just not enforced
anywhere but in that one function.

### Recommendation

1. **Make Core the single authority.** Extend the readiness payload with the
   setup facts the page needs to choose a surface — `configured`, `enabled`,
   `has_token`, `owner_count` — so one backend-derived object answers both "which
   surface" and "what state". Readiness is already polled by both surfaces and
   already carries no identity, so this adds a field set, not an endpoint.
2. **Make the frontend's `configured` a rendering of that field, never a
   derivation from the edit buffer.** The edit buffer stays what it is — unsaved
   input — and drives the *form*, not the *surface*. This is the actual fix: the
   surface stops being chosen by something the user is still typing.
3. **Keep `isTelegramStartingState` as the stage list.** PC-DEF-071's positive
   naming is what makes a state added later render honestly by default; that
   property should survive the refactor rather than be replaced by a new
   else-chain.
4. **Add a Core-side invariant test** that a channel with a token and
   `enabled: false` is reported as not configured by *every* accessor, so the
   state `clearTelegramCredentials` guards against cannot be reintroduced by a
   different writer.

### Why later rather than now

It changes a payload contract that two surfaces and the Flutter client read, and
the 14-locale copy that hangs off it. Neither runtime-integrity fix in this pass
depends on it, and doing it alongside them would put a UI-contract change inside
a release-hardening pass whose whole point is that each change is provable on its
own.
