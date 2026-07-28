package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/carvalhosauro/mdit/internal/doc"
	"github.com/carvalhosauro/mdit/internal/theme"
)

// #9: zen must not ellipsis-truncate wide table cells.
func TestZen_WideTableNoEllipsis(t *testing.T) {
	table := strings.Join([]string{
		"# Tables",
		"",
		"| Feature | Description that is quite long indeed | Status |",
		"| ------- | ------------------------------------- | ------ |",
		"| Alpha   | V2: chaves de sandbox reutilizaveis com texto longo | OK |",
		"| Beta    | Checagem estado detalhada do fluxo completo | WIP |",
	}, "\n")
	a := NewApp(nil, doc.NewFromString(table), theme.DefaultDark())
	a.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	a.enterZen()

	view := ansi.Strip(a.View())
	if strings.Contains(view, "…") {
		t.Fatalf("zen truncated table with ellipsis:\n%s", view)
	}
	compact := strings.ReplaceAll(strings.ReplaceAll(view, "\n", ""), " ", "")
	if !strings.Contains(compact, "reutilizaveis") || !strings.Contains(compact, "textolongo") {
		t.Fatalf("zen must show full table cell text, got:\n%s", view)
	}
}
