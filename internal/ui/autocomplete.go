package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

func (a *App) openAutocomplete() {
	a.mode = modeAutocomplete
	a.acActive = true
	a.acIndex = 0
	a.refreshAutocomplete(a.autocompleteQuery())
}

func (a *App) autocompleteQuery() string {
	line := a.editor.Doc().Line(a.editor.Cursor().Line)
	col := a.editor.Cursor().Col
	runes := []rune(line)
	if col > len(runes) {
		col = len(runes)
	}
	before := string(runes[:col])
	i := strings.LastIndex(before, "[[")
	if i < 0 {
		return ""
	}
	// i is byte index; take substring after [[
	return before[i+len("[["):]
}

func (a *App) refreshAutocomplete(query string) {
	a.acItems = nil
	if a.vault == nil {
		return
	}
	notes := a.vault.List()
	names := make([]string, len(notes))
	for i, n := range notes {
		names[i] = n.Name
	}
	if query == "" {
		a.acItems = append([]string(nil), names...)
	} else {
		ranks := fuzzy.Find(query, names)
		a.acItems = make([]string, len(ranks))
		for i, r := range ranks {
			a.acItems[i] = names[r.Index]
		}
	}
	if len(a.acItems) == 0 {
		a.acIndex = 0
		return
	}
	if a.acIndex >= len(a.acItems) {
		a.acIndex = len(a.acItems) - 1
	}
	if a.acIndex < 0 {
		a.acIndex = 0
	}
}

func (a *App) handleAutocompleteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		a.mode = modeEdit
		a.acActive = false
		return a, nil
	case tea.KeyUp:
		if a.acIndex > 0 {
			a.acIndex--
		}
		return a, nil
	case tea.KeyDown:
		if a.acIndex < len(a.acItems)-1 {
			a.acIndex++
		}
		return a, nil
	case tea.KeyEnter, tea.KeyTab:
		return a.acceptAutocomplete()
	}

	var cmd tea.Cmd
	a.editor, cmd = a.editor.Update(msg)
	a.refreshAutocomplete(a.autocompleteQuery())

	line := a.editor.Doc().Line(a.editor.Cursor().Line)
	col := a.editor.Cursor().Col
	runes := []rune(line)
	if col > len(runes) {
		col = len(runes)
	}
	before := string(runes[:col])
	if !strings.Contains(before, "[[") {
		a.mode = modeEdit
		a.acActive = false
	}
	return a, cmd
}

// acceptAutocomplete replaces the typed query with the selected note name and
// closes the popup. Tab and Enter both accept (terminal completion reflex).
func (a *App) acceptAutocomplete() (tea.Model, tea.Cmd) {
	if len(a.acItems) == 0 {
		a.mode = modeEdit
		a.acActive = false
		return a, nil
	}
	name := a.acItems[a.acIndex]
	q := a.autocompleteQuery()
	m := a.editor
	for range []rune(q) {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	}
	m.InsertText(name + "]]")
	a.editor = m
	a.mode = modeEdit
	a.acActive = false
	return a, nil
}

// viewAutocomplete draws the editor with the completion popup anchored at the
// cursor — the popover originates from its trigger, and the total line count
// stays exactly the terminal height (no layout jump).
func (a *App) viewAutocomplete() string {
	bg := a.editor.View() + "\n" + a.renderStatusBar()
	popup := a.renderACPopup()

	row, col := a.editor.CursorViewportRowCol()
	ph := lipgloss.Height(popup)
	pw := lipgloss.Width(popup)
	edH := a.height - 1

	// Below the cursor line by default; flip above when there is no room.
	y := row + 1
	if y+ph > edH && row-ph >= 0 {
		y = row - ph
	}
	x := col
	if a.width > 0 && x+pw > a.width {
		x = a.width - pw
	}
	if x < 0 {
		x = 0
	}
	return overlayAt(bg, popup, x, y)
}

func (a *App) renderACPopup() string {
	th := a.theme
	var b strings.Builder
	b.WriteString(th.MenuTitle.Render("[[ link"))
	if len(a.acItems) == 0 {
		b.WriteByte('\n')
		b.WriteString(th.Dim.Render("no matches"))
	} else {
		const maxShow = 8
		start := 0
		if a.acIndex >= maxShow {
			start = a.acIndex - maxShow + 1
		}
		end := start + maxShow
		if end > len(a.acItems) {
			end = len(a.acItems)
		}
		for i := start; i < end; i++ {
			b.WriteByte('\n')
			name := a.acItems[i]
			if i == a.acIndex {
				b.WriteString(th.MenuSelected.Render("▸ " + name))
			} else {
				b.WriteString(th.Menu.Render("  " + name))
			}
		}
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.MenuBorder.GetForeground()).
		Padding(0, 1).
		Render(b.String())
}
