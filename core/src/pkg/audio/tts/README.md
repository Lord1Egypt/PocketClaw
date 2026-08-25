# TTS (Text-to-Speech)

This package handles speech synthesis for PicoClaw.

If you are new to TTS setup, the simplest workflow is:

1. Add a TTS-capable entry to `model_list`.
2. Point `voice.tts_model_name` at that entry.
3. Put the API key in `.security.yml`.

## Quick Recommendation

For most users, these are the best starting points:

| Provider | Why start here |
| --- | --- |
| [OpenAI](https://platform.openai.com/docs/guides/text-to-speech) | Best-supported path in PicoClaw today. The current TTS implementation is built around the OpenAI-compatible `/audio/speech` API shape, and OpenAI is the safest default. |
| [Xiaomi MiMo](https://platform.xiaomimimo.com) | A good second option if you want an OpenAI-compatible provider endpoint and are already using MiMo models in the rest of your stack. |

## How TTS Configuration Works

PicoClaw does not keep TTS API keys inside `voice`.

Instead:

- `voice.tts_model_name` selects a named entry from `model_list`.
- That `model_list` entry provides the provider, model ID, API base, and proxy settings.
- For providers that need model-specific TTS parameters, use `model_list[].extra_body`
  to pass fields such as `voice` and `response_format`.
- `.security.yml` stores the API key for the same named model entry.

This is the recommended and supported configuration pattern.

## Recommended Setup

### Option A: OpenAI

`config.json`

```json
{
  "voice": {
    "tts_model_name": "openai-tts"
  },
  "model_list": [
    {
      "model_name": "openai-tts",
      "model": "openai/tts-1"
    }
  ]
}
```

`.security.yml`

```yaml
model_list:
  openai-tts:
    api_keys:
      - "sk-openai-your-key"
```

### Option B: Xiaomi MiMo

`config.json`

```json
{
  "voice": {
    "tts_model_name": "mimo-tts"
  },
  "model_list": [
    {
      "model_name": "mimo-tts",
      "model": "mimo/mimo-v2-tts"
    }
  ]
}
```

`.security.yml`

```yaml
model_list:
  mimo-tts:
    api_keys:
      - "your-mimo-key"
```

If you use a custom MiMo endpoint, you can also set `api_base` explicitly. Otherwise PicoClaw will use the provider default.

### Option C: OpenRouter MAI Voice 2

Some OpenAI-compatible TTS routes require provider-specific request fields.
OpenRouter's `microsoft/mai-voice-2` is one example: it needs a model-specific
voice name and works best with `response_format: "mp3"`.

`config.json`

```json
{
  "voice": {
    "tts_model_name": "mai-voice-2"
  },
  "model_list": [
    {
      "model_name": "mai-voice-2",
      "provider": "openrouter",
      "model": "microsoft/mai-voice-2",
      "api_base": "https://openrouter.ai/api/v1",
      "extra_body": {
        "voice": "en-US-Harper:MAI-Voice-2",
        "response_format": "mp3"
      }
    }
  ]
}
```

`.security.yml`

```yaml
model_list:
  mai-voice-2:
    api_keys:
      - "sk-or-your-openrouter-key"
```

## What PicoClaw Sends Today

The current TTS runtime uses an OpenAI-compatible speech request with these defaults:

- Endpoint: `/audio/speech`
- Response format: `opus`
- Voice: `alloy`
- Model: taken from the selected `model_list` entry

These defaults can now be overridden per model through `model_list[].extra_body`.

That means:

- `openai/tts-1` works naturally.
- Other OpenAI-compatible providers can work if they accept the same request format.
- Provider-specific TTS models may need their own `voice` and `response_format` values.
- If a provider rejects `response_format`, PicoClaw retries once without that field.

## How PicoClaw Chooses a TTS Provider

`DetectTTS` resolves TTS in this order:

1. **Preferred path**: resolve `voice.tts_model_name` against `model_list`.
2. If a matching model entry exists and has an API key, PicoClaw creates an OpenAI-compatible TTS provider using that model's settings.
3. **Fallback path**: if `voice.tts_model_name` is not set or cannot be resolved, PicoClaw scans `model_list` for the first entry whose model string contains `tts` and has an API key.

Fallback scanning exists for compatibility. New configs should set `voice.tts_model_name` explicitly.

## Notes About API Base Handling

PicoClaw normalizes the configured base URL for TTS:

- For OpenAI, a base like `https://api.openai.com` or `https://api.openai.com/v1` becomes `https://api.openai.com/v1/audio/speech`.
- For other OpenAI-compatible providers, PicoClaw preserves the configured base path and ensures it ends with `/audio/speech`.
- If `api_base` is omitted, PicoClaw uses the provider default base when the model prefix is known.

## Common Mistakes

- Setting `voice.tts_model_name` to a name that does not exist in `model_list`.
- Adding a TTS model but forgetting to put its API key in `.security.yml`.
- Assuming PicoClaw will automatically infer provider-specific custom voices.
- Forgetting to set `model_list[].extra_body.voice` or `model_list[].extra_body.response_format` for TTS models that require them.
- Using a provider endpoint that is not compatible with the OpenAI `/audio/speech` request format.

## Minimal Checklist

Before testing `send_tts`, make sure:

- `voice.tts_model_name` matches a `model_list[].model_name`.
- The matching `.security.yml` entry contains a valid API key.
- The chosen provider supports an OpenAI-compatible speech synthesis endpoint.
- Your selected model is actually a TTS-capable model.
