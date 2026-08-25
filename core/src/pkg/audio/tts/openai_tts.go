package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers/common"
)

type OpenAITTSProvider struct {
	apiKey         string
	apiBase        string
	voice          string
	model          string
	responseFormat string
	httpClient     *http.Client
}

type OpenAITTSOptions struct {
	Voice          string
	ResponseFormat string
}

type openAITTSAudioStream struct {
	io.ReadCloser
	fileExt     string
	contentType string
}

func (s *openAITTSAudioStream) AudioFileMeta() (string, string) {
	return s.fileExt, s.contentType
}

type openAITTSAPIError struct {
	statusCode int
	body       string
}

func (e *openAITTSAPIError) Error() string {
	return fmt.Sprintf("API error (status %d): %s", e.statusCode, e.body)
}

func NewOpenAITTSProvider(apiKey string, apiBase string, proxyURL string, model string) *OpenAITTSProvider {
	return NewOpenAITTSProviderWithOptions(apiKey, apiBase, proxyURL, model, OpenAITTSOptions{})
}

func NewOpenAITTSProviderWithOptions(
	apiKey string,
	apiBase string,
	proxyURL string,
	model string,
	options OpenAITTSOptions,
) *OpenAITTSProvider {
	// Normalize apiBase to avoid malformed endpoints like
	// "https://api.openai.com/audio/speech" when "/v1" is required.
	if apiBase == "" {
		apiBase = "https://api.openai.com/v1/audio/speech"
	} else {
		if u, err := url.Parse(apiBase); err == nil && u.Scheme != "" && u.Host != "" {
			path := u.Path
			if u.Host == "api.openai.com" {
				// For the official OpenAI host, ensure exactly one /v1 prefix and
				// that the path ends with /audio/speech.
				if path == "" || path == "/" || path == "/v1" {
					path = "/v1/audio/speech"
				} else {
					if !strings.HasPrefix(path, "/") {
						path = "/" + path
					}
					if !strings.HasPrefix(path, "/v1/") {
						path = "/v1" + strings.TrimSuffix(path, "/")
					}
					if !strings.HasSuffix(path, "/audio/speech") {
						path = strings.TrimSuffix(path, "/") + "/audio/speech"
					}
				}
			} else {
				// For non-OpenAI hosts (e.g., proxies), preserve the existing base
				// path and only ensure it ends with /audio/speech.
				if !strings.HasSuffix(path, "/audio/speech") {
					path = strings.TrimSuffix(path, "/") + "/audio/speech"
				}
			}
			u.Path = path
			apiBase = u.String()
		} else {
			// Fallback to the previous string-based behavior if parsing fails.
			if apiBase == "https://api.openai.com/v1" {
				apiBase = "https://api.openai.com/v1/audio/speech"
			} else if !strings.HasSuffix(apiBase, "/audio/speech") {
				// Just in case they provide openrouter base or standard base
				apiBase = strings.TrimSuffix(apiBase, "/") + "/audio/speech"
			}
		}
	}

	client := common.NewHTTPClient(proxyURL)
	client.Timeout = 60 * time.Second

	model = strings.TrimSpace(model)
	if model == "" {
		model = "tts-1"
	}

	voice := strings.TrimSpace(options.Voice)
	if voice == "" {
		voice = "alloy"
	}

	responseFormat := strings.TrimSpace(options.ResponseFormat)
	if responseFormat == "" {
		responseFormat = "opus"
	}

	return &OpenAITTSProvider{
		apiKey:         apiKey,
		apiBase:        apiBase,
		voice:          voice,
		model:          model,
		responseFormat: responseFormat,
		httpClient:     client,
	}
}

func (t *OpenAITTSProvider) Name() string {
	return "openai-tts"
}

func (t *OpenAITTSProvider) Synthesize(ctx context.Context, text string) (io.ReadCloser, error) {
	logger.DebugCF("voice-tts", "Starting TTS synthesis", map[string]any{"text_len": len(text)})

	responseFormat := t.responseFormat
	stream, err := t.doSpeechRequest(ctx, text, responseFormat)
	if err != nil {
		var apiErr *openAITTSAPIError
		if errors.As(err, &apiErr) && shouldRetryWithoutResponseFormat(apiErr.body) {
			logger.InfoCF("voice-tts", "Retrying TTS without response_format after provider rejection", map[string]any{
				"model": t.model,
			})
			responseFormat = ""
			stream, err = t.doSpeechRequest(ctx, text, responseFormat)
		}
		if err != nil {
			return nil, err
		}
	}

	fileExt, contentType := audioFileMetaForResponseFormat(responseFormat)
	return &openAITTSAudioStream{
		ReadCloser:  stream,
		fileExt:     fileExt,
		contentType: contentType,
	}, nil
}

func (t *OpenAITTSProvider) doSpeechRequest(
	ctx context.Context,
	text string,
	responseFormat string,
) (io.ReadCloser, error) {
	reqBody := map[string]any{
		"model": t.model,
		"input": text,
		"voice": t.voice,
	}
	if responseFormat != "" {
		reqBody["response_format"] = responseFormat
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.apiBase, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.apiKey)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			body = []byte(fmt.Sprintf("(failed to read error body: %v)", readErr))
		}
		return nil, &openAITTSAPIError{
			statusCode: resp.StatusCode,
			body:       string(body),
		}
	}

	return resp.Body, nil
}

func shouldRetryWithoutResponseFormat(body string) bool {
	lower := strings.ToLower(body)
	if !strings.Contains(lower, "response_format") {
		return false
	}
	return strings.Contains(lower, "invalid") || strings.Contains(lower, "unsupported")
}

func audioFileMetaForResponseFormat(responseFormat string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(responseFormat)) {
	case "", "mp3":
		return ".mp3", "audio/mpeg"
	case "wav":
		return ".wav", "audio/wav"
	default:
		return ".ogg", "audio/ogg"
	}
}
