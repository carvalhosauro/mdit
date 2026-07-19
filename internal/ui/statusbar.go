package ui

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"

	"github.com/carvalhosauro/mdit/internal/doc"
)

// renderStatusBar draws the bottom bar with a clear hierarchy: file name
// (bold) and dirty marker (accent) first, key hints dimmed, and on the right a
// word count, a transient flash ("✓ saved") or error, and the cursor position.
//
// Under width pressure things are dropped in order of least importance: the
// word count first, then the key hints, then the file name is truncated — the
// message and cursor position are never pushed off screen.
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
	words := fmt.Sprintf("%d words", wordCount(d))

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

	hints := "  " + barHints()
	showWords := true

	// rightCore (message + position) is protected; the word count is optional
	// and dropped before anything on the left.
	rightCore := pos
	if msg != "" {
		rightCore = msg + "  " + pos
	}

	widths := func(nameS, hintsS string, wordsOn bool) (int, int) {
		lw := runewidth.StringWidth(nameS+dirty) + runewidth.StringWidth(hintsS)
		rp := rightCore
		if wordsOn {
			rp = words + "  " + rightCore
		}
		return lw, runewidth.StringWidth(rp)
	}

	leftW, rightW := widths(name, hints, showWords)
	if leftW+rightW+1 > w { // drop word count first
		showWords = false
		leftW, rightW = widths(name, hints, false)
	}
	if leftW+rightW+1 > w { // then the hints
		hints = ""
		leftW, rightW = widths(name, "", false)
	}
	if leftW+rightW+1 > w { // finally truncate the name
		avail := w - rightW - 1 - runewidth.StringWidth(dirty)
		if avail < 0 {
			avail = 0
		}
		name = truncateCells(name, avail)
		leftW, rightW = widths(name, "", false)
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
	if showWords {
		b.WriteString(th.StatusBarHint.Render(words))
		b.WriteString(th.StatusBar.Render("  "))
	}
	if msg != "" {
		b.WriteString(msgStyle.Render(msg))
		b.WriteString(th.StatusBar.Render("  "))
	}
	b.WriteString(th.StatusBarHint.Render(pos))
	return b.String()
}

// wordCount counts whitespace-separated tokens across the whole document.
func wordCount(d *doc.Document) int {
	n := 0
	for i := 0; i < d.LineCount(); i++ {
		n += len(strings.Fields(d.Line(i)))
	}
	return n
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
