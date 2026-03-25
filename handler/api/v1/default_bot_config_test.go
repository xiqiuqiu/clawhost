package v1

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestApplyDefaultBotConfigAddsDefaultsToEmptyConfig(t *testing.T) {
	resetViperForDefaultBotConfigTest(t)
	viper.Set("bot_defaults.provider", "anthropic")
	viper.Set("bot_defaults.base_url", "https://api.anthropic.com")
	viper.Set("bot_defaults.api_key", "test-key")
	viper.Set("bot_defaults.model", "claude-sonnet-4-20250514")

	got, err := applyDefaultBotConfig(nil, loadDefaultBotConfig())
	if err != nil {
		t.Fatalf("applyDefaultBotConfig returned error: %v", err)
	}

	models, ok := got["models"].(map[string]interface{})
	if !ok {
		t.Fatalf("models section missing or wrong type: %#v", got["models"])
	}

	providers, ok := models["providers"].(map[string]interface{})
	if !ok {
		t.Fatalf("providers section missing or wrong type: %#v", models["providers"])
	}

	provider, ok := providers["anthropic"].(map[string]interface{})
	if !ok {
		t.Fatalf("default provider missing: %#v", providers["anthropic"])
	}

	if provider["apiKey"] != "test-key" {
		t.Fatalf("expected apiKey to be injected, got %#v", provider["apiKey"])
	}

	agents, ok := got["agents"].(map[string]interface{})
	if !ok {
		t.Fatalf("agents section missing or wrong type: %#v", got["agents"])
	}

	defaults, ok := agents["defaults"].(map[string]interface{})
	if !ok {
		t.Fatalf("agents.defaults missing or wrong type: %#v", agents["defaults"])
	}

	model, ok := defaults["model"].(map[string]interface{})
	if !ok {
		t.Fatalf("agents.defaults.model missing or wrong type: %#v", defaults["model"])
	}

	if model["primary"] != "anthropic/claude-sonnet-4-20250514" {
		t.Fatalf("expected primary model to be injected, got %#v", model["primary"])
	}
}

func TestApplyDefaultBotConfigPreservesExplicitModelConfig(t *testing.T) {
	resetViperForDefaultBotConfigTest(t)
	viper.Set("bot_defaults.provider", "anthropic")
	viper.Set("bot_defaults.base_url", "https://api.anthropic.com")
	viper.Set("bot_defaults.api_key", "default-key")
	viper.Set("bot_defaults.model", "claude-sonnet-4-20250514")

	input := map[string]interface{}{
		"models": map[string]interface{}{
			"providers": map[string]interface{}{
				"openai": map[string]interface{}{
					"baseUrl": "https://api.openai.com/v1",
					"apiKey":  "user-key",
					"models": []interface{}{
						map[string]interface{}{"id": "gpt-4.1"},
					},
				},
			},
		},
		"agents": map[string]interface{}{
			"defaults": map[string]interface{}{
				"model": map[string]interface{}{
					"primary": "openai/gpt-4.1",
				},
			},
		},
	}

	got, err := applyDefaultBotConfig(input, loadDefaultBotConfig())
	if err != nil {
		t.Fatalf("applyDefaultBotConfig returned error: %v", err)
	}

	models := got["models"].(map[string]interface{})
	providers := models["providers"].(map[string]interface{})
	if _, exists := providers["anthropic"]; exists {
		t.Fatalf("did not expect default provider to be injected when explicit providers exist: %#v", providers)
	}

	agents := got["agents"].(map[string]interface{})
	defaults := agents["defaults"].(map[string]interface{})
	model := defaults["model"].(map[string]interface{})
	if model["primary"] != "openai/gpt-4.1" {
		t.Fatalf("expected explicit primary model to remain unchanged, got %#v", model["primary"])
	}
}

func TestApplyDefaultBotConfigErrorsWhenDefaultsAreIncomplete(t *testing.T) {
	resetViperForDefaultBotConfigTest(t)
	viper.Set("bot_defaults.provider", "anthropic")
	viper.Set("bot_defaults.base_url", "https://api.anthropic.com")
	viper.Set("bot_defaults.model", "claude-sonnet-4-20250514")

	_, err := applyDefaultBotConfig(nil, loadDefaultBotConfig())
	if err == nil {
		t.Fatal("expected an error for incomplete default bot config, got nil")
	}

	if !strings.Contains(err.Error(), "api_key") {
		t.Fatalf("expected error to mention api_key, got %q", err.Error())
	}
}

func resetViperForDefaultBotConfigTest(t *testing.T) {
	t.Helper()
	viper.Reset()
	t.Cleanup(viper.Reset)
}
