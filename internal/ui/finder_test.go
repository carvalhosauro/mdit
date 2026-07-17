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

func TestFinder_OpenFilterEnterEsc(t *testing.T) {
	root, v := setupVault(t, map[string]string{
		"alpha.md": "# Alpha\n\n",
		"beta.md":  "# Beta\n\n",
		"gamma.md": "# Gamma\n\n",
	})
	app := newApp(t, v, filepath.Join(root, "alpha.md"))

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(80, 24))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("alpha.md"))
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlP})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		s := ansi.Strip(string(bts))
		return strings.Contains(s, "Notes") &&
			strings.Contains(s, "alpha") &&
			strings.Contains(s, "beta")
	}, teatest.WithDuration(3*time.Second))

	// Filter: '/' enters filter mode; Enter applies; then Enter opens.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	tm.Type("bet")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // apply filter
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		s := ansi.Strip(string(bts))
		return strings.Contains(s, "beta") && !strings.Contains(s, "gamma")
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // open selection
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		s := ansi.Strip(string(bts))
		return strings.Contains(s, "beta.md")
	}, teatest.WithDuration(3*time.Second))

	// Esc from a fresh finder leaves editor intact.
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlP})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(ansi.Strip(string(bts)), "Notes")
	}, teatest.WithDuration(2*time.Second))
	tm.Send(tea.KeyMsg{Type: tea.KeyEscape})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		s := ansi.Strip(string(bts))
		return strings.Contains(s, "beta.md") && !strings.Contains(s, "Notes")
	}, teatest.WithDuration(3*time.Second))

	waitQuit(t, tm)
}
