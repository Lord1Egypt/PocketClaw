package auth

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/auth"
	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

const (
	supportedProvidersMsg = "supported providers: openai, anthropic"
	defaultAnthropicModel = "claude-sonnet-4.6"
)

func authLoginCmd(provider string, useDeviceCode bool, useOauth bool, noBrowser bool) error {
	switch provider {
	case "openai":
		return authLoginOpenAI(useDeviceCode, noBrowser)
	case "anthropic":
		return authLoginAnthropic(useOauth)
	default:
		return fmt.Errorf("unsupported provider: %s (%s)", provider, supportedProvidersMsg)
	}
}

func authLoginOpenAI(useDeviceCode bool, noBrowser bool) error {
	cfg := auth.OpenAIOAuthConfig()

	var cred *auth.AuthCredential
	var err error

	if useDeviceCode {
		cred, err = auth.LoginDeviceCode(cfg)
	} else {
		cred, err = auth.LoginBrowserWithOptions(cfg, auth.LoginBrowserOptions{NoBrowser: noBrowser})
	}

	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	if err = auth.SetCredential("openai", cred); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	appCfg, err := internal.LoadConfig()
	if err == nil {
		// Update or add openai in ModelList
		foundOpenAI := false
		for i := range appCfg.ModelList {
			if isOpenAIModel(appCfg.ModelList[i]) {
				appCfg.ModelList[i].AuthMethod = "oauth"
				foundOpenAI = true
				break
			}
		}

		// If no openai in ModelList, add it
		if !foundOpenAI {
			appCfg.ModelList = append(appCfg.ModelList, &config.ModelConfig{
				ModelName:  "gpt-5.4",
				Model:      "openai/gpt-5.4",
				AuthMethod: "oauth",
			})
		}

		// Update default model to use OpenAI
		appCfg.Agents.Defaults.ModelName = "gpt-5.4"

		if err = config.SaveConfig(internal.GetConfigPath(), appCfg); err != nil {
			return fmt.Errorf("could not update config: %w", err)
		}
	}

	fmt.Println("Login successful!")
	if cred.AccountID != "" {
		fmt.Printf("Account: %s\n", cred.AccountID)
	}
	fmt.Println("Default model set to: gpt-5.4")

	return nil
}

func authLoginAnthropic(useOauth bool) error {
	if useOauth {
		return authLoginAnthropicSetupToken()
	}

	fmt.Println("Anthropic login method:")
	fmt.Println("  1) Setup token (from `claude setup-token`) (Recommended)")
	fmt.Println("  2) API key (from console.anthropic.com)")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("Choose [1]: ")
		choice := "1"
		if scanner.Scan() {
			text := strings.TrimSpace(scanner.Text())
			if text != "" {
				choice = text
			}
		}

		switch choice {
		case "1":
			return authLoginAnthropicSetupToken()
		case "2":
			return authLoginPasteToken("anthropic")
		default:
			fmt.Printf("Invalid choice: %s. Please enter 1 or 2.\n", choice)
		}
	}
}

func authLoginAnthropicSetupToken() error {
	cred, err := auth.LoginSetupToken(os.Stdin)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	if err = auth.SetCredential("anthropic", cred); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	appCfg, err := internal.LoadConfig()
	if err == nil {
		found := false
		for i := range appCfg.ModelList {
			if isAnthropicModel(appCfg.ModelList[i]) {
				appCfg.ModelList[i].AuthMethod = "oauth"
				found = true
				break
			}
		}
		if !found {
			appCfg.ModelList = append(appCfg.ModelList, &config.ModelConfig{
				ModelName:  defaultAnthropicModel,
				Model:      "anthropic/" + defaultAnthropicModel,
				AuthMethod: "oauth",
			})
			// Only set default model if user has no default configured yet
			if appCfg.Agents.Defaults.GetModelName() == "" {
				appCfg.Agents.Defaults.ModelName = defaultAnthropicModel
			}
		}

		if err := config.SaveConfig(internal.GetConfigPath(), appCfg); err != nil {
			return fmt.Errorf("could not update config: %w", err)
		}
	}

	fmt.Println("Setup token saved for Anthropic!")

	return nil
}

func fetchGoogleUserEmail(accessToken string) (string, error) {
	req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading userinfo response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("userinfo request failed: %s", string(body))
	}

	var userInfo struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return "", err
	}
	return userInfo.Email, nil
}

func authLoginPasteToken(provider string) error {
	cred, err := auth.LoginPasteToken(provider, os.Stdin)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	if err = auth.SetCredential(provider, cred); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	appCfg, err := internal.LoadConfig()
	if err == nil {
		switch provider {
		case "anthropic":
			// Update ModelList
			found := false
			for i := range appCfg.ModelList {
				if isAnthropicModel(appCfg.ModelList[i]) {
					appCfg.ModelList[i].AuthMethod = "token"
					found = true
					break
				}
			}
			if !found {
				appCfg.ModelList = append(appCfg.ModelList, &config.ModelConfig{
					ModelName:  defaultAnthropicModel,
					Model:      "anthropic/" + defaultAnthropicModel,
					AuthMethod: "token",
				})
				appCfg.Agents.Defaults.ModelName = defaultAnthropicModel
			}
		case "openai":
			// Update ModelList
			found := false
			for i := range appCfg.ModelList {
				if isOpenAIModel(appCfg.ModelList[i]) {
					appCfg.ModelList[i].AuthMethod = "token"
					found = true
					break
				}
			}
			if !found {
				appCfg.ModelList = append(appCfg.ModelList, &config.ModelConfig{
					ModelName:  "gpt-5.4",
					Model:      "openai/gpt-5.4",
					AuthMethod: "token",
				})
			}
			// Update default model
			appCfg.Agents.Defaults.ModelName = "gpt-5.4"
		}
		if err := config.SaveConfig(internal.GetConfigPath(), appCfg); err != nil {
			return fmt.Errorf("could not update config: %w", err)
		}
	}

	fmt.Printf("Token saved for %s!\n", provider)

	if appCfg != nil {
		fmt.Printf("Default model set to: %s\n", appCfg.Agents.Defaults.GetModelName())
	}

	return nil
}

func authLogoutCmd(provider string) error {
	if provider != "" {
		if err := auth.DeleteCredential(provider); err != nil {
			return fmt.Errorf("failed to remove credentials: %w", err)
		}

		appCfg, err := internal.LoadConfig()
		if err == nil {
			// Clear AuthMethod in ModelList
			for i := range appCfg.ModelList {
				switch provider {
				case "openai":
					if isOpenAIModel(appCfg.ModelList[i]) {
						appCfg.ModelList[i].AuthMethod = ""
					}
				case "anthropic":
					if isAnthropicModel(appCfg.ModelList[i]) {
						appCfg.ModelList[i].AuthMethod = ""
					}
				}
			}
			config.SaveConfig(internal.GetConfigPath(), appCfg)
		}

		fmt.Printf("Logged out from %s\n", provider)

		return nil
	}

	if err := auth.DeleteAllCredentials(); err != nil {
		return fmt.Errorf("failed to remove credentials: %w", err)
	}

	appCfg, err := internal.LoadConfig()
	if err == nil {
		// Clear all AuthMethods in ModelList
		for i := range appCfg.ModelList {
			appCfg.ModelList[i].AuthMethod = ""
		}
		config.SaveConfig(internal.GetConfigPath(), appCfg)
	}

	fmt.Println("Logged out from all providers")

	return nil
}

func authStatusCmd() error {
	store, err := auth.LoadStore()
	if err != nil {
		return fmt.Errorf("failed to load auth store: %w", err)
	}

	if len(store.Credentials) == 0 {
		fmt.Println("No authenticated providers.")
		fmt.Println("Run: picoclaw auth login --provider <name>")
		return nil
	}

	fmt.Println("\nAuthenticated Providers:")
	fmt.Println("------------------------")
	for provider, cred := range store.Credentials {
		status := "active"
		if cred.IsExpired() {
			status = "expired"
		} else if cred.NeedsRefresh() {
			status = "needs refresh"
		}

		fmt.Printf("  %s:\n", provider)
		fmt.Printf("    Method: %s\n", cred.AuthMethod)
		fmt.Printf("    Status: %s\n", status)
		if cred.AccountID != "" {
			fmt.Printf("    Account: %s\n", cred.AccountID)
		}
		if cred.Email != "" {
			fmt.Printf("    Email: %s\n", cred.Email)
		}
		if cred.ProjectID != "" {
			fmt.Printf("    Project: %s\n", cred.ProjectID)
		}
		if !cred.ExpiresAt.IsZero() {
			fmt.Printf("    Expires: %s\n", cred.ExpiresAt.Format("2006-01-02 15:04"))
		}

		if provider == "anthropic" && cred.AuthMethod == "oauth" {
			usage, err := auth.FetchAnthropicUsage(cred.AccessToken)
			if err != nil {
				fmt.Printf("    Usage: unavailable (%v)\n", err)
			} else {
				fmt.Printf("    Usage (5h):  %.1f%%\n", usage.FiveHourUtilization*100)
				fmt.Printf("    Usage (7d):  %.1f%%\n", usage.SevenDayUtilization*100)
			}
		}
	}

	return nil
}

// isOpenAIModel checks if a model config belongs to the OpenAI provider.
func isOpenAIModel(modelCfg *config.ModelConfig) bool {
	protocol, _ := providers.ExtractProtocol(modelCfg)
	return protocol == "openai"
}

// isAnthropicModel checks if a model config belongs to the Anthropic provider.
func isAnthropicModel(modelCfg *config.ModelConfig) bool {
	protocol, _ := providers.ExtractProtocol(modelCfg)
	return protocol == "anthropic"
}
