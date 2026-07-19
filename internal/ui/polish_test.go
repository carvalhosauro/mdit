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

// P4: following a broken wikilink offers to create the note, and confirming
// creates <target>.md in the current note's directory, then opens it.
func TestApp_CreateNoteFromBrokenLink(t *testing.T) {
	root, v := setupVault(t, map[string]string{
		"a.md": "[[new]]\n",
	})
	app := newApp(t, v, filepath.Join(root, "a.md"))

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(80, 24))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("new"))
	}, teatest.WithDuration(2*time.Second))

	// Cursor at 0,0 sits on [[new]].
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlCloseBracket})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(ansi.Strip(string(bts)), `Create note "new"`)
	}, teatest.WithDuration(3*time.Second))

	tm.Type("c")
	newPath := filepath.Join(root, "new.md")
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		_, err := os.Stat(newPath)
		return err == nil
	}, teatest.WithDuration(3*time.Second))

	waitQuit(t, tm)

	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("new note was not created in the note's directory: %v", err)
	}
}

// P3: an empty buffer shows a dim placeholder.
func TestApp_EmptyPlaceholder(t *testing.T) {
	root, v := setupVault(t, map[string]string{
		"empty.md": "",
	})
	app := newApp(t, v, filepath.Join(root, "empty.md"))

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(80, 24))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(ansi.Strip(string(bts)), "Start typing")
	}, teatest.WithDuration(2*time.Second))

	waitQuit(t, tm)
}

// P5: the status bar shows a live word count.
func TestApp_StatusBarWordCount(t *testing.T) {
	root, v := setupVault(t, map[string]string{
		"note.md": "alpha beta gamma\n",
	})
	app := newApp(t, v, filepath.Join(root, "note.md"))

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(80, 24))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(ansi.Strip(string(bts)), "3 words")
	}, teatest.WithDuration(2*time.Second))

	waitQuit(t, tm)
}
