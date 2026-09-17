package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// TelegramCredentialValidator proves a candidate token before it can replace
// the committed Telegram configuration.
type TelegramCredentialValidator func(
	ctx context.Context,
	token string,
	baseURL string,
	proxy string,
) error

var ErrTelegramCredentialsInvalid = errors.New("telegram credentials invalid")

const telegramCredentialValidationTimeout = 10 * time.Second

func (h *Handler) validateTelegramCredentials(
	ctx context.Context,
	token string,
	baseURL string,
	proxy string,
) error {
	validator := h.telegramCredentialValidator
	if validator == nil {
		validator = validateTelegramCredentials
	}
	return validator(ctx, token, baseURL, proxy)
}

// validateTelegramCredentials calls only getMe. It never logs or returns the
// request URL because that URL contains the candidate token.
func validateTelegramCredentials(
	ctx context.Context,
	token string,
	baseURL string,
	proxy string,
) error {
	apiRoot := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if apiRoot == "" {
		apiRoot = "https://api.telegram.org"
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if strings.TrimSpace(proxy) != "" {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			return errors.New("telegram credential validation unavailable")
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	} else if os.Getenv("HTTP_PROXY") != "" || os.Getenv("HTTPS_PROXY") != "" {
		transport.Proxy = http.ProxyFromEnvironment
	}

	validationCtx, cancel := context.WithTimeout(ctx, telegramCredentialValidationTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(
		validationCtx,
		http.MethodPost,
		apiRoot+"/bot"+token+"/getMe",
		bytes.NewReader([]byte("{}")),
	)
	if err != nil {
		return errors.New("telegram credential validation unavailable")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Transport: transport}).Do(req)
	if err != nil {
		return errors.New("telegram credential validation unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return ErrTelegramCredentialsInvalid
	}

	var body struct {
		OK        bool `json:"ok"`
		ErrorCode int  `json:"error_code"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&body); err != nil {
		return errors.New("telegram credential validation unavailable")
	}
	if body.ErrorCode == http.StatusUnauthorized {
		return ErrTelegramCredentialsInvalid
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices || !body.OK {
		return fmt.Errorf("telegram credential validation rejected")
	}
	return nil
}
