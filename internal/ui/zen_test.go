package ui_test

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestZen_ToggleReadOnlyCentered(t *testing.T) {
	root, v := setupVault(t, map[string]string{
		"note.md": "# ZenTitle\n\nbody text here\n",
	})
	app := newApp(t, v, filepath.Join(root, "note.md"))
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(100, 24))

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("ZenTitle"))
	}, teatest.WithDuration(2*time.Second))

	verBefore := app.Doc().Version()

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlE})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		s := ansi.Strip(string(bts))
		return strings.Contains(s, "zen") &&
			strings.Contains(s, "ZenTitle") &&
			!strings.Contains(s, "# ZenTitle")
	}, teatest.WithDuration(3*time.Second))

	tm.Type("x")
	time.Sleep(200 * time.Millisecond)
	if app.Doc().Version() != verBefore {
		t.Fatalf("typing in zen changed Version: %d -> %d", verBefore, app.Doc().Version())
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlE})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		s := ansi.Strip(string(bts))
		return strings.Contains(s, "# ZenTitle") || strings.Contains(s, "^S save")
	}, teatest.WithDuration(3*time.Second))

	waitQuit(t, tm)
}

func TestZen_FinderOpensAndReturns(t *testing.T) {
	root, v := setupVault(t, map[string]string{
		"alpha.md": "# Alpha\n\n",
		"beta.md":  "# Beta\n\n",
	})
	app := newApp(t, v, filepath.Join(root, "alpha.md"))
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(80, 24))

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Alpha"))
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlE})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(ansi.Strip(string(bts)), "zen")
	}, teatest.WithDuration(3*time.Second))

	// ^P from zen opens the finder.
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlP})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		s := ansi.Strip(string(bts))
		return strings.Contains(s, "Notes") && strings.Contains(s, "beta")
	}, teatest.WithDuration(3*time.Second))

	// Esc returns to zen (not raw edit).
	tm.Send(tea.KeyMsg{Type: tea.KeyEscape})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown}) // force a repaint
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		s := ansi.Strip(string(bts))
		return strings.Contains(s, "zen") && !strings.Contains(s, "Notes (")
	}, teatest.WithDuration(3*time.Second))

	// Open beta via finder and stay in zen.
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlP})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(ansi.Strip(string(bts)), "Notes")
	}, teatest.WithDuration(2*time.Second))
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	tm.Type("bet")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // apply filter
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // open
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		s := ansi.Strip(string(bts))
		return strings.Contains(s, "Beta") && strings.Contains(s, "zen") &&
			!strings.Contains(s, "# Beta")
	}, teatest.WithDuration(3*time.Second))

	waitQuit(t, tm)
}
