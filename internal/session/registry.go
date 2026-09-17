package session

// All returns every known provider, regardless of whether it's actually
// installed on this machine. Callers should filter by Detect() before
// showing a provider that has zero presence, if they want to hide agents
// that were never installed.
func All() []Provider {
	return []Provider{
		NewClaudeProvider(),
		NewAntigravityProvider(),
		NewGeminiProvider(),
		NewCodexProvider(),
		NewCopilotProvider(),
	}
}
