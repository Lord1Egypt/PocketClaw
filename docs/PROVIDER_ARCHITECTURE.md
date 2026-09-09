# PocketClaw Provider Architecture

Audit date: 2026-08-25
Milestone: Phase 2 Milestone C — Provider Catalog + Easy API-Key Setup
Audited against: PicoClaw Core `v0.3.1`, source commit
`2cf030d2fd3b871d7ec17e3be34c24688aac76da`, plus the PocketClaw changes in
`core/pocketclaw-core-v0.3.1.patch`.

This document records the provider architecture **as it exists before Milestone
C changes**, so that later work can be judged against a factual baseline. Every
claim below was read out of the source, not assumed.

## 1. Where provider configuration actually lives

The Flutter application does **not** contain any AI-provider configuration.
`lib/src/ui/config_page.dart` configures the Core *service* only: host, port,
binary path, launch arguments, theme, language, and the optional device
feedback switch. A repository-wide search for provider/API-key/model handling in
`lib/` returns only the Firebase/Umeng device-feedback code.

All AI provider and model configuration is served by the **embedded web
console**, which is the Core web frontend compiled into
`libpocketclaw-web.so` and displayed inside the Flutter WebView
(`lib/src/ui/webview_page.dart`).

Consequence for Milestone C: provider-catalog work is a **Core change** (Go
backend plus the React frontend), delivered through a rebuilt
`libpocketclaw-web.so`/`libpocketclaw.so`, not a Flutter change. Building a second
provider UI in Flutter would duplicate the console and is explicitly not the
approach taken.

## 2. Configuration schema

### 2.1 `ModelConfig` — `pkg/config/config.go:760`

One entry in `config.model_list` per configured model.

| Field | JSON | Purpose |
| --- | --- | --- |
| `ModelName` | `model_name` | User-facing alias. Required. Unique. |
| `Provider` | `provider` | Provider/protocol ID for routing. When empty it is inferred from `Model`. |
| `Model` | `model` | Model identifier, optionally provider-prefixed. Required. |
| `APIBase` | `api_base` | Endpoint override. Empty means "use the catalog default". |
| `Proxy` | `proxy` | HTTP proxy URL. |
| `Fallbacks` | `fallbacks` | Fallback model names for failover. |
| `AuthMethod` | `auth_method` | `oauth` or `token`; empty means API-key auth. |
| `ConnectMode` | `connect_mode` | `stdio`/`grpc`, CLI-bridge providers only. |
| `Workspace` | `workspace` | Working directory, CLI-bridge providers only. |
| `RPM` | `rpm` | Client-side rate limit. |
| `MaxTokensField` | `max_tokens_field` | e.g. `max_completion_tokens`. |
| `RequestTimeout` | `request_timeout` | Seconds. |
| `ThinkingLevel` | `thinking_level` | `off\|low\|medium\|high\|xhigh\|adaptive`. |
| `ToolSchemaTransform` | `tool_schema_transform` | Tool-schema compatibility shim. |
| `Streaming` | `streaming` | Per-entry streaming opt-in. |
| `ExtraBody` | `extra_body` | Extra request-body fields. |
| `CustomHeaders` | `custom_headers` | Extra HTTP headers on every request. |
| `APIKeys` | `api_keys` | `SecureStrings`; multiple keys for failover. |
| `Enabled` | `enabled` | Entry active flag. |

There is **no separate "provider" record**. A provider exists in the user's
configuration only as the `provider` field of one or more model entries. The
Milestone C "Add Provider" flow therefore creates a model entry; it does not
introduce a new persisted object.

### 2.2 Validation — `ModelConfig.Validate()` and
`web/backend/api/models.go:198 validateIncomingModelConfig`

`model_name` and `model` are required; `model` may not contain whitespace or a
leading slash; `provider` is required at the API layer and must satisfy
`providers.IsSupportedModelProvider`. Creation is additionally gated by
`createAllowedForProvider` (`models.go:158`), which special-cases `bedrock`
(always creatable, AWS credential chain resolves at runtime) and the CLI
bridges `claude-cli`/`codex-cli` (creatable only when the executable probe
succeeds). Editing an existing entry whose provider is no longer creatable is
still permitted, which is what preserves older configurations.

## 3. The provider catalog is already backend-owned

`pkg/providers/provider_metadata.go` holds `modelProviderOptionsByName`, a map
of 40 provider IDs to `ModelProviderOption`:

```
ID, DisplayName, IconSlug, Domain, DefaultAPIBase, EmptyAPIKeyAllowed,
CreateAllowed, DefaultModelAllowed, SupportsFetch, DefaultAuthMethod,
AuthMethodLocked, Local, Priority, CommonModels, Aliases, httpAPI (unexported)
```

`pkg/providers/provider_catalog.go` exposes it through `ModelProviderOptions()`
and the predicates `IsSupportedModelProvider`, `IsModelProviderFetchable`,
`IsCreatableModelProvider`, `IsDefaultModelProvider`.

The Web API returns the whole catalog in `GET /api/models` as
`provider_options`. The React layer
(`web/frontend/src/components/models/provider-registry.ts`) is a pure
projection of that payload: it defines **no** provider URLs, IDs, or semantics
of its own, and returns an empty catalog when the backend payload is missing.

**Finding:** the "do not scatter provider URLs across UI files" requirement is
already satisfied by the existing architecture. Milestone C extends the
backend-owned catalog rather than introducing a competing one.

### 3.1 Provider identity versus display name

Identity is the stable lowercase ID (`openai`, `gemini`, `anthropic`,
`deepseek`, `openrouter`, `ollama`, …). `NormalizeProvider`
(`pkg/providers/model_ref.go:31`) lowercases, trims, and resolves the alias map
built from every entry's `Aliases`. Display names are a separate
`DisplayName` field and are never used as identifiers. This requirement is
already met.

## 4. Protocol dispatch — the real constraint

`providers.CreateProviderFromConfig` (`pkg/providers/factory_provider.go:88`)
switches on the normalized protocol and **returns `unknown protocol %q` from its
`default:` branch**. The switch arms are:

| Arm | Implementation |
| --- | --- |
| `openai` | OpenAI-compatible HTTP, or Codex OAuth when `auth_method` is `oauth`/`token` |
| one shared arm: `litellm, lmstudio, gpt4free, openrouter, groq, zhipu, nvidia, venice, nearai, ollama, moonshot, shengsuanyun, siliconflow, deepseek, cerebras, vivgrid, volcengine, vllm, qwen-portal, qwen-intl, qwen-us, mistral, avian, longcat, modelscope, novita, alibaba-coding, zai, mimo` | OpenAI-compatible HTTP |
| `gemini` | Native Google `generativeLanguage` protocol |
| `minimax` | OpenAI-compatible plus forced `reasoning_split: true` |
| `anthropic` | Anthropic over the HTTP provider |
| `anthropic-messages`, `alibaba-coding-anthropic` | Native Anthropic Messages |
| `azure` | Azure OpenAI deployment URLs; Entra ID when no key |
| `bedrock` | AWS SDK credential chain |
| `antigravity`, `claude-cli`, `codex-cli`, `github-copilot` | OAuth/CLI/local bridges |

**This is the single most important architectural fact for Milestone C.**
Adding an entry to `modelProviderOptionsByName` alone is *not* sufficient: a new
OpenAI-compatible provider must also be added to the shared OpenAI-compatible
switch arm, or every request with it fails at runtime with `unknown protocol`.
Any preset added without that change would be a broken preset.

## 5. Model discovery

Endpoint: `POST /api/models/fetch` → `handleFetchModels`
(`web/backend/api/models.go:637`).

Request `{provider, api_key, api_base, model_index}`. Behavior:

1. Rejects providers where `IsModelProviderFetchable` is false.
2. When `api_key` is empty and `model_index` is given, reuses the stored key —
   but only if the normalized provider **and** the effective API base both match
   the stored entry (`lookupStoredAPIKey`, `models.go:709`). This is what lets
   the UI re-fetch without ever sending the secret back to the browser.
3. Falls back to `DefaultAPIBaseForProtocol` when `api_base` is empty, and
   errors if there is still no base.
4. 10-second context timeout.
5. Dispatches in `fetchUpstreamModels` (`models.go:738`):
   - `ollama` — strips a trailing `/v1`, then `GET {root}/api/tags`.
   - `nearai` — `GET {base}/model/list`, Bearer.
   - default — `GET {base}/models`, Bearer, accepting several response shapes.
6. Successful results are cached through `SaveCatalog`.

**Path handling is base-relative**, so a custom base such as
`https://generativelanguage.googleapis.com/v1beta/openai` correctly yields
`.../v1beta/openai/models`. Trailing slashes are trimmed exactly once by
`strings.TrimRight(apiBase, "/")`. This behavior must not regress.

### 5.1 Gemini discovery is not supported in the baseline

`gemini` has no `SupportsFetch` in the catalog, so `POST /api/models/fetch`
rejects it with `does not support model listing`. It could not simply be
flipped on: the default fetch branch sends `Authorization: Bearer`, whereas the
Gemini provider authenticates with the `X-Goog-Api-Key` header
(`pkg/providers/httpapi/gemini_provider.go:212`), and Google's native list
response is `{"models":[{"name":"models/<id>"}]}`, not the OpenAI shape.
Enabling Gemini discovery requires a dedicated fetch branch.

### 5.2 Error classification present in the baseline

PocketClaw already added `describeModelFetchNetworkError` and
`describeModelFetchHTTPError` (in `core/pocketclaw-core-v0.3.1.patch`), which
classify DNS failure, connection refused, timeout, generic network failure, and
HTTP 401/403 as authentication failure. Rate limiting, provider outage, and a
missing discovery endpoint are **not** yet distinguished.

### 5.3 Android DNS

Discovery runs inside the Core process, so it inherits the verified Android
active-network DNS bridge (`POCKETCLAW_DNS_SERVER`; the Core still accepts the
upstream `PICOCLAW_DNS_SERVER` as legacy input). No separate resolver path
exists in the fetch code, and none may be introduced.

## 6. API key storage and exposure

- **In config:** `APIKeys SecureStrings` (`pkg/config/config_struct.go:111`).
  A `SecureString` keeps a resolved value and a raw persisted form which may be
  plaintext, `file://…`, or `enc://…`.
- **At rest:** `SaveConfig` encrypts plaintext API keys to `enc://` ciphertext,
  but only when `PICOCLAW_KEY_PASSPHRASE` — an upstream name with no canonical
  alias — is set together with an SSH key;
  `file://` references are left as references. With no passphrase configured —
  **which is the normal Android situation** — keys are written as plaintext
  into the workspace config file, and `enc://` values from another environment
  cannot be loaded at all.
- **Over the API:** `GET /api/models` returns `maskAPIKey(...)`
  (`models.go:277`, `models.go:616`) — at most the first 3 and last 4
  characters, never less than 40% hidden, `****` for keys of 8 characters or
  fewer.
- **On update:** `PUT /api/models/{index}` preserves the stored key when
  `api_key` is omitted or empty (`models.go:362`), so the browser never has to
  round-trip the secret to change an unrelated field.
- **In logs:** no logging of key material was found on the model paths.

**Assessment:** the transport and UI-exposure posture is already sound and must
not be weakened. The at-rest weakness on Android (plaintext, because no
passphrase exists on the device) is real but is a platform-key-management
problem — Android Keystore or an equivalent — that touches config loading,
migration of existing files, and the Core's secret resolver. It is recorded
here and deferred to its own controlled milestone; Milestone C does not change
storage.

## 7. Custom / OpenAI-compatible endpoints in the baseline

There is no `custom` provider ID. A self-hosted or unlisted OpenAI-compatible
endpoint is configured today by selecting provider `openai` and typing an
`api_base`. That works — the `openai` arm uses `cfg.APIBase` whenever it is set
— but it is undiscoverable, and such entries are then displayed and grouped as
"OpenAI", which is misleading.

## 8. Baseline UX and why it is too technical

`web/frontend/src/components/models/add-model-sheet.tsx` presents, in the
*normal* (non-advanced) part of the form:

1. **Model Alias** — required free text the user must invent.
2. **Provider** — a combobox with edit-distance suggestions.
3. **Model Identifier** — required free text.
4. **API Key**.
5. **API Base URL** — always visible, for every provider.

plus an Advanced section with proxy, auth method, connect mode, workspace,
timeout, RPM, thinking level, max-tokens field, tool-schema transform,
streaming, extra body, and custom headers.

So the shortest path to a working cloud provider is five fields, two of which
(alias, model identifier) require knowledge the user does not have before
fetching models, and one of which (base URL) the catalog already knows. That is
the specific problem Milestone C addresses.

### 8.1 Remote logo fetching

`web/frontend/src/components/models/provider-icon.tsx` loads provider logos at
runtime from `https://cdn.simpleicons.org/<slug>` and, on failure, from
`https://www.google.com/s2/favicons?domain=<domain>`. On a mobile device this
performs third-party network requests that disclose which AI providers the user
has configured, and it degrades to a broken/blank icon whenever the device is
offline. This is removed in Milestone C in favor of local text marks.

## 9. Provider classification for Milestone C

Against the requested candidate list, judged strictly on what this pinned Core
can actually execute:

**SUPPORTED NOW** — present in the catalog with a working switch arm:
OpenAI, Anthropic (and Anthropic Messages), Google Gemini (native protocol),
DeepSeek, OpenRouter, Qwen/DashScope (`qwen-portal`, `qwen-intl`, `qwen-us`),
Moonshot/Kimi, Groq, Mistral, NVIDIA NIM, Cerebras, Ollama, LM Studio.

**OPENAI-COMPATIBLE — addable as catalog entry plus switch-arm registration:**
xAI (`https://api.x.ai/v1`), Together AI (`https://api.together.xyz/v1`),
Fireworks AI (`https://api.fireworks.ai/inference/v1`), and a first-class
Custom OpenAI-Compatible entry. All four are Bearer-auth OpenAI-shaped APIs
with a standard `GET {base}/models` listing.

**REQUIRES CORE ADAPTER** — in the catalog but not a plain HTTP key flow, so
they stay out of the simplified "paste an API key" path: Azure OpenAI
(deployment URLs, Entra ID), AWS Bedrock (AWS credential chain), GitHub Copilot
(local gRPC bridge), Google Code Assist/`antigravity` (locked OAuth),
Claude CLI and Codex CLI (local executables).

**MIXED PROTOCOL — shipped 2026-08-25:** OpenCode Zen
(`https://opencode.ai/zen/v1`) and OpenCode Go
(`https://opencode.ai/zen/go/v1`). Initially deferred for lack of verified
endpoints; the product owner supplied the official documentation, and both now
ship. They are not plain OpenAI-compatible providers — see §11.

## 11. OpenCode Zen and Go: per-model protocol routing

Both gateways front three request protocols behind a single base URL and a
single API key:

| Protocol | Endpoint | Provider used |
| --- | --- | --- |
| OpenAI Responses | `{base}/responses` | `pkg/providers/openai_responses` (new) |
| OpenAI chat completions | `{base}/chat/completions` | existing HTTP provider |
| Anthropic Messages | `{base}/messages` | `pkg/providers/anthropic_messages` |

Model discovery is OpenAI-shaped for both: `GET {base}/models`, Bearer auth,
parse `data[].id`. That goes through the existing shared fetch path unchanged.

Because the protocol is a property of the model rather than of the provider,
`CreateProviderFromConfig` cannot pick a transport from the provider ID alone.
It delegates to `pkg/providers/opencode_routing.go`, which owns:

- `ClassifyOpenCodeModel(model) (OpenCodeProtocol, known bool)` — longest
  matching model-ID family prefix; `known` is false when the fallback applies.
- `NormalizeOpenCodeModelID(model)` — strips the `opencode-go/` style CLI
  namespace so the bare ID reaches the wire.
- `IsOpenCodeProvider(provider)`.

Routing is family-based, not an enumerated model list: OpenCode changes its
lineup often, and Fetch Models already returns the authoritative live list.
An unrecognized model remains configurable and is still attempted using chat
completions, with a warning that names the model, the chosen protocol, and what
a 404 would imply. `known == false` is what stops any caller from treating a
fallback as a confirmed route.

Two consequences worth remembering:

- `common.NormalizeBaseURL` strips and re-appends `/v1`. That is a no-op for
  both OpenCode bases, but a regression there would silently turn
  `.../zen/go/v1` into `.../zen/v1`, so the round trip is asserted directly.
- The Messages route sends both `X-API-Key` and `Authorization: Bearer` via
  `anthropicmessages.WithBearerAuth()`, because OpenCode issues one account key
  for every surface and the expected form for its Messages endpoint is not
  documented in anything available here. The option is off by default, so
  Anthropic's own endpoint is unaffected.

## 12. Constraints carried into the implementation

- The backend catalog stays the single source of truth; the frontend keeps
  projecting it.
- Every new preset must be registered in **both** the catalog map and the
  protocol switch.
- Existing `model_list` entries must keep working unchanged, including entries
  that use provider `openai` with a custom `api_base`.
- Base-relative fetch URL construction must not regress.
- No API key may be logged, committed, or written into any document.
- The Android DNS bridge, MQTT `/pocketclaw` default, workspace path, Skill
  Hub, Telegram, and the arm64 release guard are untouched.
