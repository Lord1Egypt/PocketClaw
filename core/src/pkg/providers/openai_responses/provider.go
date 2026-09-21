// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

// Package openairesponses implements the OpenAI Responses API over plain HTTP
// against a caller-supplied base URL and bearer API key.
//
// The Core already spoke the Responses protocol in two places, but neither was
// reusable for a third-party gateway: the Azure provider hardcodes Azure's
// deployment path, and the Codex provider hardcodes the ChatGPT backend URL
// along with Codex-specific headers and instructions. This package keeps the
// shared request translation (pkg/providers/openai_responses_common) and adds
// only the generic transport.
package openairesponses

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"

	"github.com/sipeed/picoclaw/pkg/providers/common"
	orc "github.com/sipeed/picoclaw/pkg/providers/openai_responses_common"
	"github.com/sipeed/picoclaw/pkg/providers/protocoltypes"
)

type (
	LLMResponse    = protocoltypes.LLMResponse
	Message        = protocoltypes.Message
	ToolDefinition = protocoltypes.ToolDefinition
)

const (
	responsesAPIPath      = "responses"
	defaultRequestTimeout = 120 * time.Second
)

// Provider posts to {apiBase}/responses with a bearer API key.
type Provider struct {
	apiKey        string
	apiBase       string
	httpClient    *http.Client
	userAgent     string
	customHeaders map[string]string

	// sessionHeader, when set, names the request header that carries this
	// conversation's identity. PC-DEF-032.
	sessionHeader string
}

// SetSessionHeader makes this provider send the conversation identity under the
// named header. See common.ApplySessionHeader.
func (p *Provider) SetSessionHeader(header string) {
	p.sessionHeader = header
}

// NewProvider creates a Responses provider for an arbitrary OpenAI-compatible
// Responses endpoint.
func NewProvider(
	apiKey, apiBase, proxy, userAgent string,
	requestTimeoutSeconds int,
	customHeaders map[string]string,
) *Provider {
	client := common.NewHTTPClient(proxy)
	client.Timeout = defaultRequestTimeout
	if requestTimeoutSeconds > 0 {
		client.Timeout = time.Duration(requestTimeoutSeconds) * time.Second
	}

	headers := make(map[string]string, len(customHeaders))
	for k, v := range customHeaders {
		headers[k] = v
	}

	return &Provider{
		apiKey:        strings.TrimSpace(apiKey),
		apiBase:       strings.TrimRight(strings.TrimSpace(apiBase), "/"),
		httpClient:    client,
		userAgent:     strings.TrimSpace(userAgent),
		customHeaders: headers,
	}
}

// Chat sends a Responses API request. The model is carried in the body, so the
// URL is the base plus a single "responses" segment and nothing else.
func (p *Provider) Chat(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (*LLMResponse, error) {
	if p.apiBase == "" {
		return nil, fmt.Errorf("API base not configured")
	}
	if p.apiKey == "" {
		return nil, fmt.Errorf("api_key is required for the responses protocol")
	}

	requestURL, err := url.JoinPath(p.apiBase, responsesAPIPath)
	if err != nil {
		return nil, fmt.Errorf("building responses request URL: %w", err)
	}

	input, instructions := orc.TranslateMessages(messages)

	requestBody := responses.ResponseNewParams{
		Model: model,
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: input,
		},
		Store: openai.Opt(false),
	}
	if instructions != "" {
		requestBody.Instructions = openai.Opt(instructions)
	}
	if len(tools) > 0 {
		enableWebSearch, _ := options["native_search"].(bool)
		requestBody.Tools = orc.TranslateTools(tools, enableWebSearch)
		requestBody.ToolChoice = responses.ResponseNewParamsToolChoiceUnion{
			OfToolChoiceMode: openai.Opt(responses.ToolChoiceOptionsAuto),
		}
	}
	if maxTokens, ok := common.AsInt(options["max_tokens"]); ok {
		requestBody.MaxOutputTokens = openai.Opt(int64(maxTokens))
	}
	if temperature, ok := common.AsFloat(options["temperature"]); ok {
		requestBody.Temperature = openai.Opt(temperature)
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling responses request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("creating responses request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	if p.userAgent != "" {
		req.Header.Set("User-Agent", p.userAgent)
	}
	common.ApplySessionHeader(req, p.sessionHeader, options)
	// Custom headers last, so an operator can still override anything above.
	for k, v := range p.customHeaders {
		if strings.TrimSpace(k) == "" {
			continue
		}
		req.Header.Set(k, v)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending responses request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, common.HandleErrorResponse(resp, p.apiBase)
	}

	return orc.ParseResponseBody(resp.Body)
}

// GetDefaultModel returns an empty string: the model is always supplied by the
// configuration for a generic Responses endpoint.
func (p *Provider) GetDefaultModel() string {
	return ""
}
