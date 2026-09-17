package session

import "testing"

// TestProvidersDontCrash is a local smoke test: it runs every provider
// against whatever agent data actually exists on the machine running the
// test and just checks nothing panics or errors, logging what it found.
// It intentionally never fails just because an agent isn't installed here.
func TestProvidersDontCrash(t *testing.T) {
	for _, p := range All() {
		p := p
		t.Run(p.ID(), func(t *testing.T) {
			if !p.Detect() {
				t.Logf("%s: not installed on this machine", p.Name())
				return
			}
			sessions, err := p.ListSessions()
			if err != nil {
				t.Fatalf("%s: ListSessions error: %v", p.Name(), err)
			}
			t.Logf("%s: %d session(s)", p.Name(), len(sessions))
			for i, s := range sessions {
				if i >= 5 {
					t.Logf("  ... and %d more", len(sessions)-5)
					break
				}
				if len(s.Paths) == 0 {
					t.Errorf("%s: session %q has no Paths, DeleteSession would be a no-op", p.Name(), s.ID)
				}
				t.Logf("  - %q  project=%q  updated=%s  size=%d  paths=%v",
					s.Title, s.Project, s.UpdatedAt, s.SizeBytes, s.Paths)
			}
		})
	}
}
