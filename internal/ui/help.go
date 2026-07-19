package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// handleHelpKey closes the overlay on any key.
func (a *App) handleHelpKey(tea.KeyMsg) (tea.Model, tea.Cmd) {
	a.mode = modeEdit
	return a, nil
}

// viewHelp draws the keybinding overlay centered over the dimmed editor.
func (a *App) viewHelp() string {
	th := a.theme

	keyW := 0
	for _, r := range bindings {
		if w := lipgloss.Width(r.hint); w > keyW {
			keyW = w
		}
	}

	var b strings.Builder
	b.WriteString(th.MenuTitle.Render("Keys"))
	for _, r := range bindings {
		b.WriteByte('\n')
		pad := strings.Repeat(" ", keyW-lipgloss.Width(r.hint))
		b.WriteString(th.MenuSelected.Render(r.hint) + pad + "  " + th.Menu.Render(r.desc))
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
