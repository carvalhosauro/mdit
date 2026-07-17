package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const zenMaxWidth = 80

func (a *App) enterZen() (tea.Model, tea.Cmd) {
	a.zenSavedScroll = a.editor.Scroll()
	a.editor.SetZen(true)
	a.mode = modeZen
	a.layoutZenEditor()
	// Show the hint bar briefly, then hide it: zen is about immersion, a
	// permanent bar defeats the mode.
	a.zenBarVisible = true
	return a, a.flashTick(zenBarDuration)
}

func (a *App) exitZen() (tea.Model, tea.Cmd) {
	scroll := a.editor.Scroll()
	a.editor.SetZen(false)
	a.mode = modeEdit
	a.zenBarVisible = false
	a.layoutEditor()
	// Prefer scroll from zen reading position; fall back to pre-zen scroll.
	if scroll > 0 {
		a.editor.SetScroll(scroll)
	} else {
		a.editor.SetScroll(a.zenSavedScroll)
	}
	return a, nil
}

func (a *App) layoutZenEditor() {
	h := a.height - 1
	if h < 1 {
		h = 1
	}
	w := a.width
	if w > zenMaxWidth {
		w = zenMaxWidth
	}
	if w < 1 {
		w = 1
	}
	a.editor.SetSize(w, h)
}

func (a *App) handleZenKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlE, tea.KeyEscape:
		return a.exitZen()
	case tea.KeyCtrlQ:
		return a.doQuit()
	case tea.KeyCtrlS:
		return a.doSave()
	case tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown, tea.KeyHome, tea.KeyEnd:
		var cmd tea.Cmd
		a.editor, cmd = a.editor.Update(msg)
		return a, cmd
	}
	// Any other key (e.g. trying to type): resurface the hint bar so the way
	// out stays discoverable.
	a.zenBarVisible = true
	return a, a.flashTick(zenBarDuration)
}

func (a *App) viewZen() string {
	ed := a.editor.View()
	bar := a.renderZenBar()
	colW := a.width
	if colW > zenMaxWidth {
		colW = zenMaxWidth
	}
	lines := strings.Split(ed, "\n")
	padX := (a.width - colW) / 2
	if padX < 0 {
		padX = 0
	}
	var b strings.Builder
	for i, ln := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(strings.Repeat(" ", padX))
		b.WriteString(ln)
	}
	b.WriteByte('\n')
	if bar != "" {
		b.WriteString(strings.Repeat(" ", padX))
	}
	b.WriteString(bar)
	return b.String()
}

// renderZenBar shows feedback (save flash, errors) whenever present; the zen
// hint itself auto-hides. No full-width background — just quiet text.
func (a *App) renderZenBar() string {
	switch {
	case a.statusErr != "":
		return a.theme.StatusBarError.Render(" " + a.statusErr + " ")
	case a.statusMsg != "":
		return a.theme.StatusBarOK.Render(" " + a.statusMsg + " ")
	case a.zenBarVisible:
		return a.theme.Dim.Render("zen · ^E back")
	}
	return ""
}
