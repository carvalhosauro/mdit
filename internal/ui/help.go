package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// helpRows lists every binding, including the ones the status bar has no room
// for (back, follow-link) — the bar only advertises ^G.
var helpRows = []struct{ key, desc string }{
	{"^S", "save"},
	{"^P", "find note"},
	{"^E", "zen mode"},
	{"^B", "back to previous note"},
	{"^]", "follow wikilink under cursor"},
	{"[[", "wikilink autocomplete"},
	{"^Z / ^Y", "undo / redo"},
	{"^G", "this help"},
	{"^Q", "quit"},
}

// handleHelpKey closes the overlay on any key.
func (a *App) handleHelpKey(tea.KeyMsg) (tea.Model, tea.Cmd) {
	a.mode = modeEdit
	return a, nil
}

// viewHelp draws the keybinding overlay centered over the dimmed editor.
func (a *App) viewHelp() string {
	th := a.theme

	keyW := 0
	for _, r := range helpRows {
		if w := lipgloss.Width(r.key); w > keyW {
			keyW = w
		}
	}

	var b strings.Builder
	b.WriteString(th.MenuTitle.Render("Keys"))
	for _, r := range helpRows {
		b.WriteByte('\n')
		pad := strings.Repeat(" ", keyW-lipgloss.Width(r.key))
		b.WriteString(th.MenuSelected.Render(r.key) + pad + "  " + th.Menu.Render(r.desc))
	}
	b.WriteByte('\n')
	b.WriteString(th.Dim.Render("any key closes"))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.MenuBorder.GetForeground()).
		Padding(0, 2).
		Render(b.String())

	bg := a.dimContent(a.editor.View() + "\n" + a.renderStatusBar())
	return overlayCenter(bg, box, a.width, a.height)
}
