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
	case tea.KeyEnter:
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

func (a *App) viewAutocomplete() string {
	ed := a.editor.View()
	popup := a.renderACPopup()
	bar := a.renderStatusBar()
	if popup == "" {
		return ed + "\n" + bar
	}
	return ed + "\n" + popup + "\n" + bar
}

func (a *App) renderACPopup() string {
	if len(a.acItems) == 0 {
		return lipgloss.NewStyle().Faint(true).Render("(no matches)")
	}
	const maxShow = 8
	start := 0
	if a.acIndex >= maxShow {
		start = a.acIndex - maxShow + 1
	}
	end := start + maxShow
	if end > len(a.acItems) {
		end = len(a.acItems)
	}
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("link:"))
	for i := start; i < end; i++ {
		b.WriteByte('\n')
		name := a.acItems[i]
		if i == a.acIndex {
			b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#89B4FA")).Render("> " + name))
		} else {
			b.WriteString("  " + name)
		}
	}
	return b.String()
}
