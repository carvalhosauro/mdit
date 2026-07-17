package ui_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/teatest"
)

// TestIntegration_MVPFlow exercises the end-to-end MVP path:
// open vault → edit → autocomplete wikilink → follow link → back → zen → save+quit.
func TestIntegration_MVPFlow(t *testing.T) {
	root, v := setupVault(t, map[string]string{
		"home.md": "start\n",
		"dest.md": "# Dest\n\n",
	})
	path := filepath.Join(root, "home.md")
	app := newApp(t, v, path)

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(80, 24))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("home.md"))
	}, teatest.WithDuration(2*time.Second))

	// Edit + autocomplete wikilink to dest.
	tm.Send(tea.KeyMsg{Type: tea.KeyEnd})
	tm.Type(" [[")
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(ansi.Strip(string(bts)), "[[ link")
	}, teatest.WithDuration(3*time.Second))
	tm.Type("dest")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	time.Sleep(150 * time.Millisecond)

	// Save so follow does not hit the dirty prompt.
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlS})
	time.Sleep(100 * time.Millisecond)

	// Step left onto [[dest]] and follow.
	for i := 0; i < 6; i++ {
		tm.Send(tea.KeyMsg{Type: tea.KeyLeft})
	}
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlCloseBracket})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(ansi.Strip(string(bts)), "dest.md")
	}, teatest.WithDuration(3*time.Second))

	// Back to home.
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlB})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(ansi.Strip(string(bts)), "home.md")
	}, teatest.WithDuration(3*time.Second))

	// Zen toggle.
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlE})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(ansi.Strip(string(bts)), "zen")
	}, teatest.WithDuration(2*time.Second))
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlE})

	// Quit (clean after earlier save; re-save if zen somehow dirtied — it shouldn't).
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlQ})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "[[dest]]") {
		t.Fatalf("expected saved wikilink, got %q", data)
	}
}
