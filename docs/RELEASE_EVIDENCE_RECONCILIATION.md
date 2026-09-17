# Release-evidence reconciliation

This record reconciles two counts that looked like regressions against older
release reports. Both differences are explained by a documented change of scope,
not by a lost or weakened check. It is evidence for the release record, not a
change to any gate.

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
