package session

// All returns every known provider, regardless of whether it's actually
// installed on this machine. Callers should filter by Detect() before
// showing a provider that has zero presence, if they want to hide agents
// that were never installed.
//
// overrides maps a provider ID (see Provider.ID) to a config directory that
// replaces its built-in default, for agents installed somewhere else. A nil
// or empty map just uses every provider's default.
func All(overrides map[string]string) []Provider {
	return []Provider{
		NewClaudeProvider(overrides["claude"]),
		NewAntigravityProvider(overrides["antigravity"]),
		NewGeminiProvider(overrides["gemini"]),
		NewCodexProvider(overrides["codex"]),
		NewCopilotProvider(overrides["copilot"]),
		NewGrokProvider(overrides["grok"]),
	}
}
