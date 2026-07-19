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

	"github.com/carvalhosauro/mdit/internal/ui"
)

func TestWikilink_FollowBackBrokenAutocomplete(t *testing.T) {
	root, v := setupVault(t, map[string]string{
		"a.md": "[[b]] and [[nope]]\n",
		"b.md": "# B Note\n\n",
	})
	app := newApp(t, v, filepath.Join(root, "a.md"))

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(80, 24))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("[[b]]"))
	}, teatest.WithDuration(2*time.Second))

	// Cursor at 0,0 is on [[b]].
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlCloseBracket})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(ansi.Strip(string(bts)), "b.md")
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlB})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(ansi.Strip(string(bts)), "a.md")
	}, teatest.WithDuration(3*time.Second))

	// Move onto [[nope]]: "[[b]] and [[nope]]" — col ~12.
	for i := 0; i < 12; i++ {
		tm.Send(tea.KeyMsg{Type: tea.KeyRight})
	}
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlCloseBracket})
	// P4: a broken link now offers to create the note instead of flashing an
	// error.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(ansi.Strip(string(bts)), `Create note "nope"`)
	}, teatest.WithDuration(3*time.Second))
	tm.Send(tea.KeyMsg{Type: tea.KeyEscape}) // cancel the create prompt

	waitQuit(t, tm)
}

func TestWikilink_AutocompleteInserts(t *testing.T) {
	root, v := setupVault(t, map[string]string{
		"a.md": "",
		"b.md": "# B\n",
	})
	app := newApp(t, v, filepath.Join(root, "a.md"))
	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(80, 24))

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("a.md"))
	}, teatest.WithDuration(2*time.Second))

	tm.Type("[[")
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(ansi.Strip(string(bts)), "[[ link")
	}, teatest.WithDuration(3*time.Second))

	tm.Type("b")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	time.Sleep(200 * time.Millisecond)
	if err := tm.Quit(); err != nil {
		t.Fatal(err)
	}
	fm := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second))
	app2 := fm.(*ui.App)
	if !strings.Contains(app2.Doc().Content(), "[[b]]") {
		t.Fatalf("got %q", app2.Doc().Content())
	}
}
