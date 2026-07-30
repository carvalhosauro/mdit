package blockedit

import (
	"strings"
	"unicode/utf8"

	"github.com/carvalhosauro/mdit/internal/theme"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/wordwrap"
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

func (w *tableWidget) setFocus(row, col int, atEnd bool) {
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
	n := utf8.RuneCountInString(w.cell(row, col))
	if atEnd {
		w.cellCol = n
	} else {
		w.cellCol = 0
	}
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
	// Tab (forward) → caret at start; Shift+Tab (back) → caret at end.
	w.setFocus(idx/cols, idx%cols, delta < 0)
}

func (w *tableWidget) moveVertical(delta int) {
	rows := w.rowCount()
	next := w.focusRow + delta
	if next < 0 || next >= rows {
		return // stay; editor handles leave-block commit in S1.6
	}
	// Down → start; Up → end (symmetric with Tab / Shift+Tab).
	w.setFocus(next, w.focusCol, delta < 0)
}

func (w *tableWidget) moveLeft() {
	if w.cellCol > 0 {
		w.cellCol--
		return
	}
	if w.focusCol > 0 {
		w.setFocus(w.focusRow, w.focusCol-1, true) // enter from right → end
	} else if w.focusRow > 0 {
		w.setFocus(w.focusRow-1, w.colCount()-1, true)
	}
}

func (w *tableWidget) moveRight() {
	n := utf8.RuneCountInString(w.cell(w.focusRow, w.focusCol))
	if w.cellCol < n {
		w.cellCol++
		return
	}
	if w.focusCol+1 < w.colCount() {
		w.setFocus(w.focusRow, w.focusCol+1, false) // enter from left → start
		return
	}
	if w.focusRow+1 < w.rowCount() {
		w.setFocus(w.focusRow+1, 0, false)
	}
}

func (w *tableWidget) moveHome() {
	w.cellCol = 0
}

func (w *tableWidget) moveEnd() {
	w.cellCol = utf8.RuneCountInString(w.cell(w.focusRow, w.focusCol))
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

func (w *tableWidget) insertRowBelow() {
	ncols := w.colCount()
	empty := make([]string, ncols)
	if w.focusRow == 0 {
		// Insert as first body row.
		w.body = append([][]string{empty}, w.body...)
		w.setFocus(1, w.focusCol, false)
		return
	}
	// Insert below focused body row at body index focusRow.
	bi := w.focusRow
	if bi > len(w.body) {
		bi = len(w.body)
	}
	w.body = append(w.body[:bi], append([][]string{empty}, w.body[bi:]...)...)
	w.setFocus(w.focusRow+1, w.focusCol, false)
}

func (w *tableWidget) deleteFocusedRow() {
	if w.focusRow == 0 {
		return // header is immovable
	}
	bi := w.focusRow - 1
	if len(w.body) <= 1 {
		// Last body row: clear cells, keep the grid.
		w.body[0] = make([]string, w.colCount())
		w.setFocus(1, w.focusCol, false)
		return
	}
	w.body = append(w.body[:bi], w.body[bi+1:]...)
	if w.focusRow > len(w.body) {
		w.setFocus(len(w.body), w.focusCol, false)
	} else {
		w.setFocus(w.focusRow, w.focusCol, false)
	}
}

func (w *tableWidget) insertColRight() {
	at := w.focusCol + 1
	w.header = insertStringAt(w.header, at, "")
	w.align = insertAlignAt(w.align, at, AlignNone)
	for i := range w.body {
		w.body[i] = insertStringAt(w.body[i], at, "")
	}
	w.setFocus(w.focusRow, at, false)
}

func (w *tableWidget) deleteFocusedCol() {
	if w.colCount() <= 1 {
		return
	}
	c := w.focusCol
	w.header = removeStringAt(w.header, c)
	w.align = removeAlignAt(w.align, c)
	for i := range w.body {
		w.body[i] = removeStringAt(w.body[i], c)
	}
	if c >= w.colCount() {
		c = w.colCount() - 1
	}
	w.setFocus(w.focusRow, c, false)
}

func (w *tableWidget) cycleAlign() {
	if w.focusCol < 0 || w.focusCol >= len(w.align) {
		return
	}
	w.align[w.focusCol] = (w.align[w.focusCol] + 1) % 4
}

func insertStringAt(s []string, i int, v string) []string {
	if i < 0 {
		i = 0
	}
	if i > len(s) {
		i = len(s)
	}
	out := make([]string, 0, len(s)+1)
	out = append(out, s[:i]...)
	out = append(out, v)
	out = append(out, s[i:]...)
	return out
}

func removeStringAt(s []string, i int) []string {
	if i < 0 || i >= len(s) {
		return s
	}
	out := make([]string, 0, len(s)-1)
	out = append(out, s[:i]...)
	out = append(out, s[i+1:]...)
	return out
}

func insertAlignAt(a []Align, i int, v Align) []Align {
	if i < 0 {
		i = 0
	}
	if i > len(a) {
		i = len(a)
	}
	out := make([]Align, 0, len(a)+1)
	out = append(out, a[:i]...)
	out = append(out, v)
	out = append(out, a[i:]...)
	return out
}

func removeAlignAt(a []Align, i int) []Align {
	if i < 0 || i >= len(a) {
		return a
	}
	out := make([]Align, 0, len(a)-1)
	out = append(out, a[:i]...)
	out = append(out, a[i+1:]...)
	return out
}

func (w *tableWidget) viewLines(width int) []string {
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
	out = append(out, w.renderViewRow(w.header, widths, th.TableHeader, 0, th)...)
	out = append(out, viewSeparator(widths, th.Table))
	for i, row := range w.body {
		out = append(out, w.renderViewRow(row, widths, th.Table, i+1, th)...)
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

func (w *tableWidget) renderViewRow(cells []string, widths []int, style lipgloss.Style, row int, th theme.Theme) []string {
	wrapped := make([][]string, len(widths))
	maxLines := 1
	for i := range widths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		var lines []string
		if row == w.focusRow && i == w.focusCol {
			lines = paintFocusedCell(cell, widths[i], w.cellCol, th)
		} else {
			lines = wrapViewPlain(cell, widths[i])
			for j, ln := range lines {
				lines[j] = style.Render(runewidth.FillRight(ln, widths[i]))
			}
		}
		wrapped[i] = lines
		if len(lines) > maxLines {
			maxLines = len(lines)
		}
	}
	sep := " " + style.Render("│") + " "
	out := make([]string, maxLines)
	for r := 0; r < maxLines; r++ {
		parts := make([]string, len(widths))
		for c := range widths {
			if r < len(wrapped[c]) {
				parts[c] = wrapped[c][r]
			} else {
				parts[c] = style.Render(runewidth.FillRight("", widths[c]))
			}
		}
		out[r] = strings.Join(parts, sep)
	}
	return out
}

// paintFocusedCell renders a cell with Selection fill and a reverse caret at
// cellCol (rune index). Returns one or more screen lines (soft-wrapped).
func paintFocusedCell(cell string, width, cellCol int, th theme.Theme) []string {
	if width < 1 {
		width = 1
	}
	fill := th.Selection
	caret := th.Selection.Reverse(true)
	runes := []rune(cell)
	if cellCol < 0 {
		cellCol = 0
	}
	if cellCol > len(runes) {
		cellCol = len(runes)
	}

	type seg struct {
		text  string
		style lipgloss.Style
	}
	var segs []seg
	for i := 0; i < len(runes); i++ {
		if i == cellCol {
			segs = append(segs, seg{string(runes[i]), caret})
		} else {
			segs = append(segs, seg{string(runes[i]), fill})
		}
	}
	if cellCol >= len(runes) {
		segs = append(segs, seg{" ", caret})
	}

	contentW := 0
	for _, s := range segs {
		contentW += runewidth.StringWidth(s.text)
	}
	for contentW < width {
		segs = append(segs, seg{" ", fill})
		contentW++
	}

	var lines []string
	var b strings.Builder
	lineW := 0
	flush := func() {
		for lineW < width {
			b.WriteString(fill.Render(" "))
			lineW++
		}
		lines = append(lines, b.String())
		b.Reset()
		lineW = 0
	}
	for _, s := range segs {
		sw := runewidth.StringWidth(s.text)
		if lineW+sw > width && lineW > 0 {
			flush()
		}
		b.WriteString(s.style.Render(s.text))
		lineW += sw
	}
	if b.Len() > 0 || len(lines) == 0 {
		flush()
	}
	return lines
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
