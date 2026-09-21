# OPERATING RECORD — PC-DEF-023 third-party OAuth dependency removal

RECONSTRUCTED OPERATING RECORD. Evidence-based closeout, not the original prompt.

## Status and boundary

- **Status:** **RESOLVED.**
- **Working branch:** `feature/final-release-hardening`.
- **Starting commit:** `3bab1168b224fde6cc7308cdf5c9a2c9b3a9087e` (PC-DEF-024 closeout).
- **Source / build-input commit:** `54ff2525fa555744d017aae56c9a26e2049812e1`.
- **Staging / closeout commit:** the following commit carrying the rebuilt Core
  pair and this record, touching no Core build input.
- **Version/baseline:** `0.2.0+62`; accepted physical baseline vc62 /
  `lastAcceptedVersionCode=62`, untouched.

`PC-DEF-006` and `PC-DEF-012` are untouched. Gemini API-key support, Public
Mode, the AAB policy, the resolved updater route, the signing architecture and
native export surfaces were not modified. No production signing, no production
candidate, no device, no ADB, no merge, tag, publication or release.

## The decision, and why it is not a secrecy finding

Owner product decision: **Google Antigravity is not shipped in v0.2.0.**

The exposure audit classified the credential deliberately and this milestone
keeps that classification. It is **not** a secret disclosure. An installed-app
OAuth client cannot keep a secret — RFC 8252 and Google's own desktop-client
model do not treat one as confidential — so nothing that was ever protected was
published, and neither a PocketClaw nor a user credential was involved.

What blocks shipping is **ownership**. The client ID and secret belonged to
another project; the source comment recorded them as the same credentials that
project's plugin uses. A third party can revoke them at any time, and every
PocketClaw user's provider would stop working for a reason PocketClaw could
neither predict, detect in advance, nor fix. A stable release should not carry a
feature whose availability is someone else's decision.

Recorded as `PC-D014` in `docs/DECISIONS.md`. The provider may return under a
PocketClaw-owned OAuth client; restoring it with a third party's is exactly what
the decision forbids.

## Surfaces removed

Enumerated first, then removed one at a time — removing the OAuth config alone
would have left a registered, constructible, UI-advertised provider behind it.

    pkg/auth/oauth.go            GoogleAntigravityOAuthConfig, both embedded
                                 credentials, the orphaned decodeBase64, and the
                                 googleapis.com token-URL inference that produced
                                 the "google-antigravity" provider name
    pkg/auth/store.go            the antigravity -> google-antigravity alias
    pkg/providers/oauth/         antigravity_provider.go + test, deleted
    pkg/providers/oauth_facade.go        type aliases and fetch wrappers
    pkg/providers/factory_provider.go    the construction arm
    pkg/providers/provider_metadata.go   the product-facing catalogue entry
    pkg/config/defaults.go       the keyless model_list template
    pkg/config/config_old.go     the legacy-import protocol mapping
    pkg/config/config.go         the supported-provider comment
    web/backend/api/oauth.go     constant, order, methods, labels, config arm,
                                 project-ID fetch, default model
    web/backend/api/models.go    the implicit auth_method default
    web/backend/api/model_status.go      providerUsesImplicitOAuth, whose only
                                 provider this was
    cmd/.../auth/helpers.go      the login arm, authLoginGoogleAntigravity,
                                 authModelsCmd, isAntigravityModel
    cmd/.../auth/models.go       the "auth models" subcommand, deleted — it
                                 existed only to list Antigravity models
    cmd/.../auth/login.go        the --provider flag help
    frontend                     antigravity-credential-card.tsx deleted; the
                                 OAuthProvider union member; the hook's status
                                 and label; the locale key in all 14 bundles
    workspace/skills/            the embedded agent guidance that advertised the
                                 provider and the removed auth models command

## Surfaces intentionally retained

- **`OAuthProviderConfig.ClientSecret`** and the confidential-client branches in
  the token exchange. That is shared OAuth infrastructure; another provider may
  legitimately need it, and narrowing it was explicitly out of scope.
- **`canonicalProvider`**, minus the alias. Trim and lower-case normalisation is
  what the credential store needs and every caller goes through it; only the
  `antigravity` → `google-antigravity` mapping went.
- **Gemini, entirely.** A different provider with its own catalogue entry,
  API-key auth and `generativelanguage.googleapis.com` base. A test now pins it,
  including the `google` → `gemini` alias, because "Google Antigravity" and
  "Google Gemini" are the two things easiest to conflate here.
- **OpenAI OAuth and the Anthropic token flow**, plus PKCE, state, the callback
  and session handling — all unchanged.

## Provider list before and after

    OAuth surface before   openai, anthropic, google-antigravity
    OAuth surface after    openai, anthropic

    normalizeOAuthProvider("antigravity")         -> unsupported provider
    normalizeOAuthProvider("google-antigravity")  -> unsupported provider

The error is the router's existing unsupported-provider path, not a special case
written for this removal, and the provider is absent from the catalogue the
dashboard renders — so it is not a hidden callable provider behind a removed UI.

## Credential absence

**Source:** a walk over every `.go`, `.ts`, `.tsx`, `.json`, `.dart` and `.kt`
file finds neither the encoded nor the decoded client-secret prefix, neither the
encoded client-id prefix, nor `.apps.googleusercontent.com`. Asserted by test,
so a reintroduction fails rather than being noticed later.

**Binaries:** both staged Core binaries carry **zero** occurrences of
`google-antigravity`, `antigravity`, `Antigravity`, `Google Code Assist`,
`antigravity.google`, `R09DU1BYLU`, `GOCSPX-`, the encoded client-id prefix and
`apps.googleusercontent.com`. Gemini's `generativelanguage.googleapis.com` and
`Google Gemini` are still present, which is what makes the absence checks
meaningful rather than a blanket "no google" rule.

The first rebuild was **not** clean: three `antigravity` strings remained in
`libpocketclaw.so`. They came from `workspace/skills/picoclaw-agent/SKILL.md`,
which is `go:embed`-ed into the binary and still told the agent that
`antigravity` was an available provider and that `auth models` was a command.
That is a shipped product surface, so it was corrected and the pair rebuilt.
Checking the binary rather than trusting the source diff is what caught it.

## Tests

`core/src/web/backend/api/no_antigravity_test.go`:

- five spellings — including mixed case, padded and underscored — are rejected
  with the unsupported-provider error;
- the OAuth surface is OpenAI and Anthropic only, across order, methods and
  labels;
- no catalogue entry or alias mentions the provider;
- Gemini and its `google` alias survive;
- neither credential form exists in any source file.

`NormalizeProvider` is deliberately **not** the absence assertion. A first
attempt used it and failed: it is a string normaliser with an alias table that
echoes an unknown id straight back, so it says nothing about whether a provider
is registered. The catalogue is what the dashboard renders, so the catalogue is
what the test reads.

Tests that existed only for the removed provider were deleted:
`antigravity_provider_test.go`, the facade-signature test, the factory and
list-models provider cases, the CLI status alias test, `models_test.go`'s
subcommand test, and the credential-store alias tests. Shared tests were edited
rather than dropped. The store's trim/mixed-case normalisation test was **kept**
with its fixture retargeted to `openai`, because that behaviour survived and
only the alias went — deleting it would have lost real coverage.

## Core rebuild and re-stage

    source fingerprint   6f00359dc9e8bf7ee24f9d170754b2792a41fb880d9da4f34a8600dd8f99df00
                      →  bc35a598d3a836e0a0c95afc73314fe49a38877b985b5b0f15bab11460184fa9
    build-input commit   54ff2525fa555744d017aae56c9a26e2049812e1
    BuildTime            2026-09-13T00:24:56+0000

    libpocketclaw.so       37,658,976  0a28bd5e1d6e33dc35b808039571b6683e4d47dec021941650f916641b859f6e
                                       build ID c657e80da54549a3bcc9a8bdba0a576d7b7273a0
    libpocketclaw-web.so   25,319,424  9ae1d2d9e7ac26d602db722649ebec4ac9cd982fa166c50ded303685d2abce50
                                       build ID 45355d87ba5ec042740675f82bc3d3940265318a

Both shrank against the pair they replace — the gateway by 65,664 bytes and the
dashboard by 65,664 — as the provider, its fetch paths and its UI left the
binaries.

Produced byte-identically in the canonical in-repository run and two further
independent output roots under different `CORE_BUILD_DIR`, `JNI_LIBS` and
`NATIVE_SYMBOL_ROOT` paths, the third with a cold `GOCACHE`. Both private
companions likewise:

    libpocketclaw.so.debug       14,525,160  709cf86397dcb365b545eed00a84aede20a20ce77b1663562b1803cc4f77e546
    libpocketclaw-web.so.debug    9,687,312  a8c8f3d421fdce710a709cb330c79b63fe9aa68d7e0e5d55d58d0dd152006ebe

Native contract **22 PASS / 0 FAIL**. No Managed Runtime payload was rebuilt —
`git diff` over `jniLibs/` shows exactly the two Core files.

**Stated honestly:** the private support manifest's `apk` field still names the
PC-DEF-024 audit APK `f580cadc…`, which no longer contains this pair. That is
the established pre-artifact state — `bind-apk` runs against a candidate
artifact and none was built here — and it is rebound at the next artifact build.

## Tests and gates

    Core Go suite (go test ./...)              98 packages ok, 0 failed
    pkg/coresource                             46 tests, 0 failed
    TestStagedCoreWasBuiltFromTheCurrentSource  PASS
    TestBundledPayloadsMatchTheirPinnedChecksums / EmbedsTheCurrentCatalog  PASS
    Native contract, Core pair                 22 PASS / 0 FAIL
    flutter analyze                            No issues found
    flutter test                              497 passed, 0 failed
    frontend (vitest)                          25 files, 418 tests passed

Final source-gate totals are recorded in the owner report.

## Closeout conditions

1. The product no longer exposes Google Antigravity — OAuth surface, catalogue,
   factory, defaults, CLI, dashboard and embedded agent guidance.
2. The backend rejects both spellings with its ordinary unsupported-provider
   error.
3. The frontend does not advertise it; the card is deleted and the locale key is
   gone from all fourteen bundles.
4. Third-party credentials are absent from source, asserted by test.
5. Third-party credentials are absent from both staged Core binaries, verified
   by string inspection.
6. Gemini API-key functionality remains, asserted by test and visible in the
   binaries.
7. OpenAI OAuth and the Anthropic token flow remain.
8. Core is reproducibly rebuilt and re-staged across three roots.
9. All relevant tests and gates green.
