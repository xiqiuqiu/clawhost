package v1

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type defaultBotConfig struct {
	Provider      string
	BaseURL       string
	APIKey        string
	Auth          string
	API           string
	AuthHeader    bool
	Model         string
	ModelName     string
	Reasoning     bool
	Input         []string
	ContextWindow int
	MaxTokens     int
	FallbackModel string
}

func loadDefaultBotConfig() defaultBotConfig {
	return defaultBotConfig{
		Provider:      strings.TrimSpace(strings.ToLower(viper.GetString("bot_defaults.provider"))),
		BaseURL:       strings.TrimSpace(viper.GetString("bot_defaults.base_url")),
		APIKey:        strings.TrimSpace(viper.GetString("bot_defaults.api_key")),
		Auth:          strings.TrimSpace(viper.GetString("bot_defaults.auth")),
		API:           strings.TrimSpace(viper.GetString("bot_defaults.api")),
		AuthHeader:    viper.GetBool("bot_defaults.auth_header"),
		Model:         strings.TrimSpace(viper.GetString("bot_defaults.model")),
		ModelName:     strings.TrimSpace(viper.GetString("bot_defaults.model_name")),
		Reasoning:     viper.GetBool("bot_defaults.reasoning"),
		Input:         append([]string(nil), viper.GetStringSlice("bot_defaults.input")...),
		ContextWindow: viper.GetInt("bot_defaults.context_window"),
		MaxTokens:     viper.GetInt("bot_defaults.max_tokens"),
		FallbackModel: strings.TrimSpace(viper.GetString("bot_defaults.fallback_model")),
	}
}

func applyDefaultBotConfig(input map[string]interface{}, defaults defaultBotConfig) (map[string]interface{}, error) {
	cfg := cloneConfigMap(input)

	needsProvider := !hasAnyProviders(cfg)
	needsPrimary := !hasPrimaryModel(cfg)

	if !needsProvider && !needsPrimary {
		return cfg, nil
	}
	if !defaults.isConfigured() {
		return cfg, nil
	}

	normalized, err := defaults.normalized()
	if err != nil {
		return nil, err
	}

	if needsProvider {
		models := ensureMap(cfg, "models")
		if strings.TrimSpace(asString(models["mode"])) == "" {
			models["mode"] = "merge"
		}
		providers := ensureMap(models, "providers")
		providers[normalized.Provider] = normalized.providerMap()
	}

	if !hasPrimaryModel(cfg) && providerExists(cfg, normalized.Provider) {
		agents := ensureMap(cfg, "agents")
		defaultsMap := ensureMap(agents, "defaults")
		modelMap := ensureMap(defaultsMap, "model")
		modelMap["primary"] = normalized.primaryModelRef()

		if normalized.FallbackModel != "" {
			modelAliases := ensureMap(defaultsMap, "models")
			if _, exists := modelAliases["fallback"]; !exists {
				modelAliases["fallback"] = map[string]interface{}{"alias": normalized.FallbackModel}
			}
		}
	}

	return cfg, nil
}

func (c defaultBotConfig) isConfigured() bool {
	return c.Provider != "" ||
		c.BaseURL != "" ||
		c.APIKey != "" ||
		c.Auth != "" ||
		c.API != "" ||
		c.Model != "" ||
		c.ModelName != "" ||
		c.FallbackModel != "" ||
		c.AuthHeader ||
		c.Reasoning ||
		len(c.Input) > 0 ||
		c.ContextWindow > 0 ||
		c.MaxTokens > 0
}

func (c defaultBotConfig) normalized() (defaultBotConfig, error) {
	out := c
	if out.Provider == "" {
		return out, fmt.Errorf("bot_defaults.provider is required")
	}
	if out.APIKey == "" {
		return out, fmt.Errorf("bot_defaults.api_key is required")
	}
	if out.Model == "" {
		return out, fmt.Errorf("bot_defaults.model is required")
	}
	if out.BaseURL == "" {
		out.BaseURL = defaultProviderBaseURL(out.Provider)
		if out.BaseURL == "" {
			return out, fmt.Errorf("bot_defaults.base_url is required for provider %q", out.Provider)
		}
	}
	if out.Auth == "" {
		out.Auth = "api-key"
	}
	if out.API == "" {
		out.API = defaultProviderAPI(out.Provider)
	}
	if out.ModelName == "" {
		out.ModelName = out.Model
	}
	if len(out.Input) == 0 {
		out.Input = []string{"text"}
	}
	if out.ContextWindow == 0 {
		out.ContextWindow = 200000
	}
	if out.MaxTokens == 0 {
		out.MaxTokens = 8192
	}
	return out, nil
}

func (c defaultBotConfig) providerMap() map[string]interface{} {
	return map[string]interface{}{
		"baseUrl":    c.BaseURL,
		"apiKey":     c.APIKey,
		"auth":       c.Auth,
		"authHeader": c.AuthHeader,
		"api":        c.API,
		"models": []map[string]interface{}{
			{
				"id":            c.Model,
				"name":          c.ModelName,
				"reasoning":     c.Reasoning,
				"input":         c.Input,
				"contextWindow": c.ContextWindow,
				"maxTokens":     c.MaxTokens,
			},
		},
	}
}

func (c defaultBotConfig) primaryModelRef() string {
	return fmt.Sprintf("%s/%s", c.Provider, c.Model)
}

func defaultProviderBaseURL(provider string) string {
	switch provider {
	case "anthropic":
		return "https://api.anthropic.com"
	case "openai":
		return "https://api.openai.com/v1"
	default:
		return ""
	}
}

func defaultProviderAPI(provider string) string {
	if provider == "openai" {
		return "openai-completions"
	}
	return "anthropic-messages"
}

func cloneConfigMap(input map[string]interface{}) map[string]interface{} {
	if len(input) == 0 {
		return make(map[string]interface{})
	}

	raw, err := json.Marshal(input)
	if err != nil {
		return make(map[string]interface{})
	}

	var cloned map[string]interface{}
	if err := json.Unmarshal(raw, &cloned); err != nil {
		return make(map[string]interface{})
	}
	return cloned
}

func ensureMap(parent map[string]interface{}, key string) map[string]interface{} {
	if existing, ok := parent[key].(map[string]interface{}); ok {
		return existing
	}
	next := make(map[string]interface{})
	parent[key] = next
	return next
}

func hasAnyProviders(cfg map[string]interface{}) bool {
	models, ok := cfg["models"].(map[string]interface{})
	if !ok {
		return false
	}
	providers, ok := models["providers"].(map[string]interface{})
	return ok && len(providers) > 0
}

func hasPrimaryModel(cfg map[string]interface{}) bool {
	agents, ok := cfg["agents"].(map[string]interface{})
	if !ok {
		return false
	}
	defaults, ok := agents["defaults"].(map[string]interface{})
	if !ok {
		return false
	}
	model, ok := defaults["model"].(map[string]interface{})
	if !ok {
		return false
	}
	return strings.TrimSpace(asString(model["primary"])) != ""
}

func providerExists(cfg map[string]interface{}, provider string) bool {
	models, ok := cfg["models"].(map[string]interface{})
	if !ok {
		return false
	}
	providers, ok := models["providers"].(map[string]interface{})
	if !ok {
		return false
	}
	_, exists := providers[provider]
	return exists
}

func asString(value interface{}) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}
