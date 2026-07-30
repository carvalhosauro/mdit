package ui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestApp_TableWidgetStatusHint(t *testing.T) {
	root, v := setupVault(t, map[string]string{
		"t.md": "# T\n\n| A | B |\n| --- | --- |\n| 1 | 2 |\n",
	})
	app := newApp(t, v, filepath.Join(root, "t.md"))

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(100, 24))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(ansi.Strip(string(bts)), "A")
	}, teatest.WithDuration(2*time.Second))

	// Move onto the table (heading → blank → table).
	for i := 0; i < 3; i++ {
		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	}
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		s := ansi.Strip(string(bts))
		return strings.Contains(s, "table") && strings.Contains(s, "esc cancel")
	}, teatest.WithDuration(3*time.Second))

	waitQuit(t, tm)
}

func TestApp_TableWidgetEditLeaveSaveSmoke(t *testing.T) {
	root, v := setupVault(t, map[string]string{
		"t.md": "| A | B |\n| --- | --- |\n| 1 | 2 |\n",
	})
	path := filepath.Join(root, "t.md")
	app := newApp(t, v, path)

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(80, 24))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(ansi.Strip(string(bts)), "A")
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.Type("Z")
	tm.Send(tea.KeyMsg{Type: tea.KeyUp}) // commit + leave
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlS})

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(ansi.Strip(string(bts)), "saved")
	}, teatest.WithDuration(3*time.Second))

	waitQuit(t, tm)

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "ZA") {
		t.Fatalf("saved file should contain edited cell ZA (typed at start), got %q", string(b))
	}
}
