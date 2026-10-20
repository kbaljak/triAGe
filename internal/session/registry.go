package session

// IDs is every known provider ID, in the order they should be displayed.
var IDs = []string{"claude", "antigravity", "gemini", "codex", "copilot", "grok"}

// NewProvider constructs the provider for the given ID with the given home
// directory override (empty: use its built-in default). Returns nil for an
// unrecognized ID.
func NewProvider(id, override string) Provider {
	switch id {
	case "claude":
		return NewClaudeProvider(override)
	case "antigravity":
		return NewAntigravityProvider(override)
	case "gemini":
		return NewGeminiProvider(override)
	case "codex":
		return NewCodexProvider(override)
	case "copilot":
		return NewCopilotProvider(override)
	case "grok":
		return NewGrokProvider(override)
	}
	return nil
}

// All returns every known provider, regardless of whether it's actually
// installed on this machine. Callers should filter by Detect() before
// showing a provider that has zero presence, if they want to hide agents
// that were never installed.
//
// overrides maps a provider ID (see Provider.ID) to a config directory that
// replaces its built-in default, for agents installed somewhere else. A nil
// or empty map just uses every provider's default.
func All(overrides map[string]string) []Provider {
	providers := make([]Provider, 0, len(IDs))
	for _, id := range IDs {
		providers = append(providers, NewProvider(id, overrides[id]))
	}
	return providers
}
