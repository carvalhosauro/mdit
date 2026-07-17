package ui_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
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

func waitQuit(t *testing.T, tm *teatest.TestModel) {
	t.Helper()
	if err := tm.Quit(); err != nil {
		t.Fatal(err)
	}
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
