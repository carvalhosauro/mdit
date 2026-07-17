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

	"github.com/carvalhosauro/mdit/internal/doc"
	"github.com/carvalhosauro/mdit/internal/theme"
	"github.com/carvalhosauro/mdit/internal/ui"
	"github.com/carvalhosauro/mdit/internal/vault"
)

func setupVault(t *testing.T, files map[string]string) (string, *vault.Vault) {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	v, err := vault.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	return root, v
}

func newApp(t *testing.T, v *vault.Vault, path string) *ui.App {
	t.Helper()
	d, err := doc.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return ui.NewApp(v, d, theme.DefaultDark())
}

func TestApp_OpenShowsRenderedHeading(t *testing.T) {
	root, v := setupVault(t, map[string]string{
		"note.md": "# Hello World\n\nbody paragraph here\n",
	})
	app := newApp(t, v, filepath.Join(root, "note.md"))

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(80, 24))
	// Leave the heading block so it re-renders without the raw '#' prefix.
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		stripped := ansi.Strip(string(bts))
		return strings.Contains(stripped, "Hello World") &&
			!strings.Contains(stripped, "# Hello World")
	}, teatest.WithDuration(3*time.Second))

	if err := tm.Quit(); err != nil {
		t.Fatal(err)
	}
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestApp_TypeAndSave(t *testing.T) {
	root, v := setupVault(t, map[string]string{
		"note.md": "# Title\n\n",
	})
	path := filepath.Join(root, "note.md")
	app := newApp(t, v, path)

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(80, 24))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Title"))
	}, teatest.WithDuration(2*time.Second))

	// Move to end of first line (raw heading) and append unique text, then save.
	tm.Send(tea.KeyMsg{Type: tea.KeyEnd})
	tm.Type(" SAVED")
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlS})

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		data, err := os.ReadFile(path)
		if err != nil {
			return false
		}
		return strings.Contains(string(data), "SAVED")
	}, teatest.WithDuration(3*time.Second))

	if err := tm.Quit(); err != nil {
		t.Fatal(err)
	}
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestApp_QuitDirtyDiscard(t *testing.T) {
	root, v := setupVault(t, map[string]string{
		"note.md": "# Keep\n\n",
	})
	path := filepath.Join(root, "note.md")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	app := newApp(t, v, path)

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(80, 24))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Keep"))
	}, teatest.WithDuration(2*time.Second))

	tm.Type("DIRTY")
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlQ})

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Unsaved changes"))
	}, teatest.WithDuration(2*time.Second))

	tm.Type("d")
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("file was modified on discard:\nbefore=%q\nafter=%q", before, after)
	}
}

func TestApp_QuitSaveThroughConflict(t *testing.T) {
	root, v := setupVault(t, map[string]string{
		"note.md": "# Orig\n\n",
	})
	path := filepath.Join(root, "note.md")
	app := newApp(t, v, path)

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(80, 24))
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Orig"))
	}, teatest.WithDuration(2*time.Second))

	tm.Type("LOCAL")
	// External change bumps mtime so the next Save returns ErrExternalChange.
	time.Sleep(10 * time.Millisecond)
	if err := os.WriteFile(path, []byte("# External\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlQ})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Unsaved changes"))
	}, teatest.WithDuration(2*time.Second))

	tm.Type("s")
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("File changed on disk"))
	}, teatest.WithDuration(2*time.Second))

	tm.Type("o")
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "LOCAL") {
		t.Fatalf("overwrite did not keep local edits: %q", data)
	}
}
