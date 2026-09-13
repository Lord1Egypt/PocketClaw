# Provider compatibility matrix

Generated from `core/src/pkg/providers/provider_metadata.go` (the catalog) and
`core/src/pkg/providers/factory_provider.go` (the protocol dispatch), then
annotated with what physical testing actually established. PC-DEF-032.

**Status vocabulary**

| Status | Means |
| --- | --- |
| PROVEN WORKING | A real inference request completed against this provider. |
| PROVEN BROKEN | A real request failed, and the cause is identified. |
| PARTIALLY COMPATIBLE | Some surface verified (usually discovery), another failing or unverified. |
| UNTESTED | Wired and buildable; no request has been made against it in this milestone. |
| UNKNOWN | Not enough information to classify. |

A successful `/models` call is discovery, not inference, and is never recorded
as PROVEN WORKING on its own.

## Matrix

| Provider ID | Protocol | Inference endpoint | Auth header type | Discovery | Default base | Status |
| --- | --- | --- | --- | --- | --- | --- |
| `alibaba-coding` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | no | `https://coding-intl.dashscope.aliyuncs.com/v1` | UNTESTED |
| `alibaba-coding-anthropic` | Anthropic Messages (native HTTP) | POST {base}/messages | x-api-key | no | `https://coding-intl.dashscope.aliyuncs.com/apps/anthropic` | UNTESTED |
| `anthropic` | Anthropic (SDK) | POST {base}/messages | x-api-key | no | `https://api.anthropic.com/v1` | UNTESTED |
| `anthropic-messages` | Anthropic Messages (native HTTP) | POST {base}/messages | x-api-key | no | `https://api.anthropic.com/v1` | UNTESTED |
| `avian` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `https://api.avian.io/v1` | UNTESTED |
| `azure` | Azure OpenAI | POST {base}/openai/deployments/.../chat/completions | api-key header or Entra token | no | `—` | UNTESTED |
| `bedrock` | AWS Bedrock (SDK) | AWS SDK Converse | AWS credential chain (ambient) | no | `—` | UNTESTED |
| `cerebras` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `https://api.cerebras.ai/v1` | UNTESTED |
| `claude-cli` | Local CLI bridge | claude binary | the CLI's own login | no | `—` | UNTESTED |
| `codex-cli` | Local CLI bridge | codex binary | the CLI's own login | no | `—` | UNTESTED |
| `custom-openai` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `—` | UNTESTED |
| `deepseek` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `https://api.deepseek.com/v1` | UNTESTED |
| `elevenlabs` | Speech-to-text, not a chat provider | POST {base}/v1/speech-to-text | xi-api-key | no | `https://api.elevenlabs.io` | UNTESTED |
| `fireworks` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `https://api.fireworks.ai/inference/v1` | UNTESTED |
| `gemini` | Gemini native | POST {base}/models/{model}:generateContent | x-goog-api-key | yes | `https://generativelanguage.googleapis.com/v1beta` | PROVEN WORKING |
| `github-copilot` | GitHub Copilot bridge | local gRPC/HTTP bridge | Copilot session | no | `localhost:4321` | UNTESTED |
| `gpt4free` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `http://localhost:1337/v1` | UNTESTED |
| `groq` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `https://api.groq.com/openai/v1` | UNTESTED |
| `litellm` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `http://localhost:4000/v1` | UNTESTED |
| `lmstudio` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `http://localhost:1234/v1` | UNTESTED |
| `longcat` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `https://api.longcat.chat/openai` | UNTESTED |
| `mimo` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `https://api.xiaomimimo.com/v1` | UNTESTED |
| `minimax` | MiniMax native | provider-specific | Bearer | yes | `https://api.minimaxi.com/v1` | UNTESTED |
| `mistral` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `https://api.mistral.ai/v1` | UNTESTED |
| `modelscope` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `https://api-inference.modelscope.cn/v1` | UNTESTED |
| `moonshot` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `https://api.moonshot.cn/v1` | UNTESTED |
| `nearai` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `https://cloud-api.near.ai/v1` | UNTESTED |
| `novita` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `https://api.novita.ai/openai` | UNTESTED |
| `nvidia` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `https://integrate.api.nvidia.com/v1` | UNTESTED |
| `ollama` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `http://localhost:11434/v1` | UNTESTED |
| `openai` | OpenAI Responses API, or Chat Completions | POST {base}/responses (OAuth/token) or {base}/chat/completions | Bearer, or Codex OAuth | yes | `https://api.openai.com/v1` | UNTESTED |
| `opencode_go` | Mixed: Responses / Chat Completions / Anthropic Messages, chosen per model | POST {base}/responses \| /chat/completions \| /messages | Bearer (plus x-api-key on the Messages arm) **and a required `x-opencode-session`** | yes | `https://opencode.ai/zen/go/v1` | PROVEN BROKEN (service healthy; PocketClaw fixed in source, unverified) |
| `opencode_zen` | Mixed: Responses / Chat Completions / Anthropic Messages, chosen per model | POST {base}/responses \| /chat/completions \| /messages | Bearer (plus x-api-key on the Messages arm) and `x-opencode-session` | yes | `https://opencode.ai/zen/v1` | PARTIALLY COMPATIBLE |
| `openrouter` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `https://openrouter.ai/api/v1` | UNTESTED |
| `qwen-intl` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `https://dashscope-intl.aliyuncs.com/compatible-mode/v1` | UNTESTED |
| `qwen-portal` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `https://dashscope.aliyuncs.com/compatible-mode/v1` | UNTESTED |
| `qwen-us` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | no | `https://dashscope-us.aliyuncs.com/compatible-mode/v1` | UNTESTED |
| `shengsuanyun` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `https://router.shengsuanyun.com/api/v1` | UNTESTED |
| `siliconflow` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `https://api.siliconflow.cn/v1` | UNTESTED |
| `together` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `https://api.together.xyz/v1` | UNTESTED |
| `venice` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `https://api.venice.ai/api/v1` | UNTESTED |
| `vivgrid` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `https://api.vivgrid.com/v1` | UNTESTED |
| `vllm` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `http://localhost:8000/v1` | UNTESTED |
| `volcengine` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `https://ark.cn-beijing.volces.com/api/v3` | UNTESTED |
| `xai` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `https://api.x.ai/v1` | UNTESTED |
| `zai` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `https://api.z.ai/api/coding/paas/v4` | UNTESTED |
| `zhipu` | OpenAI Chat Completions | POST {base}/chat/completions | Bearer | yes | `https://open.bigmodel.cn/api/paas/v4` | UNTESTED |

## Evidence

### `opencode_go` — service PROVEN WORKING, PocketClaw PROVEN BROKEN

Discovery PASS on device (~37 models) and re-confirmed here: `GET
https://opencode.ai/zen/go/v1/models` returns 200 with 38 ids including
`deepseek-v4.1-flash`.

Inference was then proven working **outside PocketClaw** by the owner, against
the same endpoint, key and model:

| Request | Result |
| --- | --- |
| no `x-opencode-session` | HTTP 400 `MissingSessionID` — "Request is missing x-opencode-session and cannot be routed efficiently" |
| `x-opencode-session: <uuid>` + `User-Agent: PocketClaw/0.2.0` | HTTP 200, assistant content `OK` |

So the service, the key, the model and the endpoint are healthy, and the
incompatibility was PocketClaw's: it sent no session header at all. This gateway
routes on conversation identity and refuses a request that does not carry one.

PocketClaw now derives a stable, opaque, per-conversation identifier and sends
it on all three transport arms. Not yet confirmed from the device, so this row
stays broken until it is.

### `opencode_zen` — PARTIALLY COMPATIBLE

Discovery PASS on device (~70 models). Two separate facts about this endpoint,
neither of which was the owner's failure:

1. It does not serve `deepseek-v4.1-flash`. `GET .../zen/v1/models` does not
   list it, and posting it answers `ModelError: Model deepseek-v4.1-flash is not
   supported`. That model belongs to OpenCode Go. This is a real hazard when a
   model id is carried across an endpoint change, and is guarded now, but it is
   not what broke the owner's chat.
2. It is the same service and account key as OpenCode Go, so it receives the
   same `x-opencode-session` header. The requirement is proven on Go only; an
   additive routing header cannot harm the Zen surface, and omitting it there
   would leave a second gateway broken for the same reason.

Zen's own model ids were not exercised against inference.

### `gemini` — PROVEN WORKING

A Gemini inference request completed during Samsung physical testing.

## What UNTESTED means here

Every row marked UNTESTED is wired through a protocol this build implements and
compiles against, and is covered by the provider unit tests. None of them has
had a request made against the real service in this milestone, because doing so
requires that provider's credential. Recording them as working would be a
claim the evidence does not support.

## Rows that are not chat providers

`elevenlabs` appears in the same catalog because the model list is where a
transcription credential is configured, but it has no chat surface: it is
reached through `pkg/audio/asr`, not through the provider factory, and
`validateIncomingModelConfig` restricts it to the one supported model id.

## The mixed-protocol gateways

OpenCode Zen and OpenCode Go share one base-URL shape, one key and three
request protocols, chosen per model in `opencode_routing.go`. Two consequences
this milestone confirmed:

1. **They require a conversation identity.** `x-opencode-session` is not
   optional on Go: without it every request is a 400, whatever the key or model.
   It must be stable per conversation, which means it cannot be generated per
   request, and it must not be the conversation's own key, which means it cannot
   be sent raw.
2. **The two endpoints do not serve the same inventory.** `deepseek-v4.1-flash`
   is in Go's list and not in Zen's. A model id is only meaningful together
   with the endpoint it was discovered from.
3. **The gateway validates the model before the credential.** An unsupported
   model produces a `ModelError` naming the model, which is exactly the detail
   the chat window used to discard — as is the `MissingSessionID` message that
   would have made this whole defect self-diagnosing.

