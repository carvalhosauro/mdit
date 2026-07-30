package blockedit

import (
	"strings"
	"unicode/utf8"

	"github.com/carvalhosauro/mdit/internal/theme"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/termenv"
)

func (w *tableWidget) rowCount() int {
	return 1 + len(w.body) // header + body
}

func (w *tableWidget) colCount() int {
	return len(w.header)
}

func (w *tableWidget) cell(row, col int) string {
	if row == 0 {
		if col < 0 || col >= len(w.header) {
			return ""
		}
		return w.header[col]
	}
	bi := row - 1
	if bi < 0 || bi >= len(w.body) || col < 0 || col >= len(w.body[bi]) {
		return ""
	}
	return w.body[bi][col]
}

func (w *tableWidget) setCell(row, col int, s string) {
	if row == 0 {
		if col >= 0 && col < len(w.header) {
			w.header[col] = s
		}
		return
	}
	bi := row - 1
	if bi >= 0 && bi < len(w.body) && col >= 0 && col < len(w.body[bi]) {
		w.body[bi][col] = s
	}
}

func (w *tableWidget) clampCellCol() {
	n := utf8.RuneCountInString(w.cell(w.focusRow, w.focusCol))
	if w.cellCol < 0 {
		w.cellCol = 0
	}
	if w.cellCol > n {
		w.cellCol = n
	}
}

func (w *tableWidget) setFocus(row, col int) {
	rows := w.rowCount()
	cols := w.colCount()
	if rows == 0 || cols == 0 {
		return
	}
	if row < 0 {
		row = 0
	}
	if row >= rows {
		row = rows - 1
	}
	if col < 0 {
		col = 0
	}
	if col >= cols {
		col = cols - 1
	}
	w.focusRow = row
	w.focusCol = col
	w.cellCol = utf8.RuneCountInString(w.cell(row, col))
}

func (w *tableWidget) moveFocus(delta int) {
	rows := w.rowCount()
	cols := w.colCount()
	if rows == 0 || cols == 0 {
		return
	}
	idx := w.focusRow*cols + w.focusCol + delta
	total := rows * cols
	idx = ((idx % total) + total) % total
	w.setFocus(idx/cols, idx%cols)
}

func (w *tableWidget) moveVertical(delta int) {
	rows := w.rowCount()
	next := w.focusRow + delta
	if next < 0 || next >= rows {
		return // stay; editor handles leave-block commit in S1.6
	}
	w.setFocus(next, w.focusCol)
}

func (w *tableWidget) moveLeft() {
	if w.cellCol > 0 {
		w.cellCol--
		return
	}
	if w.focusCol > 0 {
		w.setFocus(w.focusRow, w.focusCol-1)
	} else if w.focusRow > 0 {
		w.setFocus(w.focusRow-1, w.colCount()-1)
	}
}

func (w *tableWidget) moveRight() {
	n := utf8.RuneCountInString(w.cell(w.focusRow, w.focusCol))
	if w.cellCol < n {
		w.cellCol++
		return
	}
	if w.focusCol+1 < w.colCount() {
		w.setFocus(w.focusRow, w.focusCol+1)
		w.cellCol = 0
		return
	}
	if w.focusRow+1 < w.rowCount() {
		w.setFocus(w.focusRow+1, 0)
		w.cellCol = 0
	}
}

func (w *tableWidget) insertRunes(rs []rune) {
	cur := []rune(w.cell(w.focusRow, w.focusCol))
	w.clampCellCol()
	out := make([]rune, 0, len(cur)+len(rs))
	out = append(out, cur[:w.cellCol]...)
	out = append(out, rs...)
	out = append(out, cur[w.cellCol:]...)
	w.setCell(w.focusRow, w.focusCol, string(out))
	w.cellCol += len(rs)
}

func (w *tableWidget) backspace() {
	cur := []rune(w.cell(w.focusRow, w.focusCol))
	w.clampCellCol()
	if w.cellCol == 0 {
		return
	}
	out := append(append([]rune{}, cur[:w.cellCol-1]...), cur[w.cellCol:]...)
	w.setCell(w.focusRow, w.focusCol, string(out))
	w.cellCol--
}

func (w *tableWidget) deleteForward() {
	cur := []rune(w.cell(w.focusRow, w.focusCol))
	w.clampCellCol()
	if w.cellCol >= len(cur) {
		return
	}
	out := append(append([]rune{}, cur[:w.cellCol]...), cur[w.cellCol+1:]...)
	w.setCell(w.focusRow, w.focusCol, string(out))
}

func (w *tableWidget) viewLines(width int) []string {
	// Force a color profile so Reverse/background styles emit ANSI in tests
	// and non-TTY environments (matches editor/layout_test.go).
	lipgloss.SetColorProfile(termenv.TrueColor)
	th := theme.DefaultDark()
	ncols := w.colCount()
	if ncols == 0 {
		return []string{""}
	}
	if width < 1 {
		width = 1
	}

	widths := make([]int, ncols)
	for c := 0; c < ncols; c++ {
		widths[c] = runewidth.StringWidth(w.header[c])
		for _, row := range w.body {
			if c < len(row) {
				if ww := runewidth.StringWidth(row[c]); ww > widths[c] {
					widths[c] = ww
				}
			}
		}
		if widths[c] < 1 {
			widths[c] = 1
		}
	}
	widths = fitViewColumns(widths, width)

	var out []string
	out = append(out, renderViewRow(w.header, widths, th.TableHeader, 0, w.focusRow, w.focusCol, th)...)
	out = append(out, viewSeparator(widths, th.Table))
	for i, row := range w.body {
		out = append(out, renderViewRow(row, widths, th.Table, i+1, w.focusRow, w.focusCol, th)...)
	}
	if len(out) == 0 {
		out = []string{""}
	}
	return out
}

func fitViewColumns(widths []int, width int) []int {
	cols := len(widths)
	avail := width - 3*(cols-1)
	if avail < cols {
		avail = cols
	}
	sum := 0
	for i, w := range widths {
		if w < 1 {
			widths[i] = 1
			w = 1
		}
		sum += w
	}
	if sum <= avail {
		return widths
	}
	out := make([]int, cols)
	for i, w := range widths {
		scaled := w * avail / sum
		if scaled < 1 {
			scaled = 1
		}
		out[i] = scaled
	}
	return out
}

func renderViewRow(cells []string, widths []int, style lipgloss.Style, row, focusRow, focusCol int, th theme.Theme) []string {
	wrapped := make([][]string, len(widths))
	maxLines := 1
	for i := range widths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		lines := wrapViewPlain(cell, widths[i])
		for j, ln := range lines {
			lines[j] = runewidth.FillRight(ln, widths[i])
		}
		wrapped[i] = lines
		if len(lines) > maxLines {
			maxLines = len(lines)
		}
	}
	focusStyle := th.RawBlock.Reverse(true)
	sep := " " + style.Render("│") + " "
	out := make([]string, maxLines)
	for r := 0; r < maxLines; r++ {
		parts := make([]string, len(widths))
		for c := range widths {
			ln := runewidth.FillRight("", widths[c])
			if r < len(wrapped[c]) {
				ln = wrapped[c][r]
			}
			if row == focusRow && c == focusCol {
				parts[c] = focusStyle.Render(ln)
			} else {
				parts[c] = style.Render(ln)
			}
		}
		out[r] = strings.Join(parts, sep)
	}
	return out
}

func viewSeparator(widths []int, style lipgloss.Style) string {
	parts := make([]string, len(widths))
	for i, w := range widths {
		parts[i] = style.Render(strings.Repeat("─", w))
	}
	return strings.Join(parts, style.Render("─┼─"))
}

func wrapViewPlain(s string, width int) []string {
	if width < 1 {
		width = 1
	}
	if s == "" {
		return []string{""}
	}
	wrapped := wordwrap.String(s, width)
	var lines []string
	for _, ln := range strings.Split(wrapped, "\n") {
		if runewidth.StringWidth(ln) <= width {
			lines = append(lines, ln)
			continue
		}
		lines = append(lines, hardWrapView(ln, width)...)
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func hardWrapView(s string, width int) []string {
	var lines []string
	var cur strings.Builder
	curW := 0
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if curW+rw > width && curW > 0 {
			lines = append(lines, cur.String())
			cur.Reset()
			curW = 0
		}
		cur.WriteRune(r)
		curW += rw
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}
