package ui

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"
)

// renderStatusBar draws the bottom bar with a clear hierarchy: file name
// (bold) and dirty marker (accent) first, key hints dimmed, and on the right a
// transient flash ("✓ saved"), an error, and the cursor position.
func (a *App) renderStatusBar() string {
	th := a.theme
	d := a.editor.Doc()
	cur := a.editor.Cursor()

	name := a.fileName()
	dirty := ""
	if d.Dirty() {
		dirty = " ●"
	}
	pos := fmt.Sprintf("%d:%d", cur.Line+1, cur.Col+1)

	msg := ""
	msgStyle := th.StatusBarOK
	if a.statusErr != "" {
		msg = a.statusErr
		msgStyle = th.StatusBarError
	} else if a.statusMsg != "" {
		msg = a.statusMsg
	}

	w := a.width
	if w < 1 {
		w = 1
	}

	rightPlain := pos
	if msg != "" {
		rightPlain = msg + "  " + pos
	}
	rightW := runewidth.StringWidth(rightPlain)

	// Left side: prefer dropping hints, then truncating the name, over ever
	// pushing the message/position off screen.
	hints := "  " + barHints()
	leftW := runewidth.StringWidth(name+dirty) + runewidth.StringWidth(hints)
	if leftW+rightW+1 > w {
		hints = ""
		leftW = runewidth.StringWidth(name + dirty)
	}
	if leftW+rightW+1 > w {
		avail := w - rightW - 1 - runewidth.StringWidth(dirty)
		if avail < 0 {
			avail = 0
		}
		name = truncateCells(name, avail)
		leftW = runewidth.StringWidth(name+dirty) + runewidth.StringWidth(hints)
	}

	pad := w - leftW - rightW
	if pad < 0 {
		pad = 0
	}

	var b strings.Builder
	b.WriteString(th.StatusBarName.Render(name))
	if dirty != "" {
		b.WriteString(th.StatusBarDirty.Render(dirty))
	}
	if hints != "" {
		b.WriteString(th.StatusBarHint.Render(hints))
	}
	b.WriteString(th.StatusBar.Render(strings.Repeat(" ", pad)))
	if msg != "" {
		b.WriteString(msgStyle.Render(msg))
		b.WriteString(th.StatusBar.Render("  "))
	}
	b.WriteString(th.StatusBarHint.Render(pos))
	return b.String()
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
