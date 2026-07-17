package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

const statusHints = "^S save ^P notes ^E zen ^Q quit"

func (a *App) renderStatusBar() string {
	d := a.editor.Doc()
	cur := a.editor.Cursor()

	name := a.fileName()
	dirty := ""
	if d.Dirty() {
		dirty = " [+]"
	}
	left := fmt.Sprintf("%s%s │ %d:%d │ %s", name, dirty, cur.Line+1, cur.Col+1, statusHints)

	right := ""
	rightStyle := a.theme.StatusBar
	if a.statusErr != "" {
		right = a.statusErr
		rightStyle = a.theme.StatusBarError
	}

	w := a.width
	if w < 1 {
		w = 1
	}

	leftW := runewidth.StringWidth(left)
	rightW := runewidth.StringWidth(right)
	pad := w - leftW - rightW
	if pad < 1 {
		// Prefer showing the left side; truncate left if needed.
		avail := w - rightW
		if avail < 0 {
			avail = 0
		}
		left = truncateCells(left, avail)
		leftW = runewidth.StringWidth(left)
		pad = w - leftW - rightW
		if pad < 0 {
			pad = 0
		}
	}

	line := left + strings.Repeat(" ", pad) + right
	style := a.theme.StatusBar.Width(w)
	if right != "" {
		// Paint left with StatusBar and right with error style, then join.
		leftStyled := a.theme.StatusBar.Render(left + strings.Repeat(" ", pad))
		rightStyled := rightStyle.Render(right)
		return lipgloss.JoinHorizontal(lipgloss.Top, leftStyled, rightStyled)
	}
	return style.Render(line)
}

func truncateCells(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= max {
		return s
	}
	var b strings.Builder
	w := 0
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if w+rw > max {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	return b.String()
}
