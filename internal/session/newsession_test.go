package session

import (
	"reflect"
	"testing"
)

// TestNewSessionArgv locks in the one real behavioral distinction in this
// file: Claude Code has a real --name flag (a true rename, never sent to
// the model), while Codex/Grok/Gemini have no such flag and fall back to
// sending the name as the session's first message instead.
func TestNewSessionArgv(t *testing.T) {
	cases := []struct {
		provider NewSessionStarter
		name     string
		want     []string
	}{
		{NewClaudeProvider(""), "", []string{"claude"}},
		{NewClaudeProvider(""), "my session", []string{"claude", "--name", "my session"}},
		{NewCodexProvider(""), "", []string{"codex"}},
		{NewCodexProvider(""), "my session", []string{"codex", "my session"}},
		{NewGrokProvider(""), "", []string{"grok"}},
		{NewGrokProvider(""), "my session", []string{"grok", "my session"}},
		{NewGeminiProvider(""), "", []string{"gemini"}},
		{NewGeminiProvider(""), "my session", []string{"gemini", "my session"}},
	}
	for _, c := range cases {
		argv, _, err := c.provider.NewSession(c.name)
		if err != nil {
			t.Errorf("%T.NewSession(%q) returned error: %v", c.provider, c.name, err)
			continue
		}
		if !reflect.DeepEqual(argv, c.want) {
			t.Errorf("%T.NewSession(%q) = %v, want %v", c.provider, c.name, argv, c.want)
		}
	}
}

// TestCopilotAndAntigravityHaveNoNewSession confirms the providers that
// genuinely can't start a session non-interactively from triAGe correctly
// don't implement the interface at all, rather than implementing it badly.
func TestCopilotAndAntigravityHaveNoNewSession(t *testing.T) {
	var copilot Provider = NewCopilotProvider("")
	if _, ok := copilot.(NewSessionStarter); ok {
		t.Error("CopilotProvider unexpectedly implements NewSessionStarter")
	}
	var antigravity Provider = NewAntigravityProvider("")
	if _, ok := antigravity.(NewSessionStarter); ok {
		t.Error("AntigravityProvider unexpectedly implements NewSessionStarter")
	}
}
