package blockedit

import (
	"strings"
	"unicode/utf8"

	"github.com/carvalhosauro/mdit/internal/doc"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"
)

// Align is a GFM column alignment.
type Align int

const (
	AlignNone Align = iota
	AlignLeft
	AlignCenter
	AlignRight
)

// tableWidget is the in-memory GFM pipe-table editor.
type tableWidget struct {
	header []string
	align  []Align
	body   [][]string

	focusRow int // 0 = header; body starts at 1
	focusCol int
	cellCol  int // rune cursor within focused cell

	openCursor doc.Position
	blockStart int
}

// OpenTable builds a table widget from the block's raw lines.
// ok=false → caller falls back to lazy-raw (malformed / not a pipe table).
func OpenTable(rawLines []string, cursor doc.Position, blockStart int) (Widget, bool) {
	if len(rawLines) < 2 {
		return nil, false
	}
	header := splitPipeRow(rawLines[0])
	align, ok := parseSeparator(rawLines[1], len(header))
	if !ok || len(header) == 0 {
		return nil, false
	}
	ncols := len(header)
	body := make([][]string, 0, len(rawLines)-2)
	for _, line := range rawLines[2:] {
		cells := splitPipeRow(line)
		body = append(body, padCells(cells, ncols))
	}
	if len(body) == 0 {
		body = [][]string{make([]string, ncols)}
	}
	w := &tableWidget{
		header:     padCells(header, ncols),
		align:      align,
		body:       body,
		openCursor: cursor,
		blockStart: blockStart,
	}
	w.focusFromCursor(cursor, blockStart, rawLines)
	return w, true
}

func (w *tableWidget) Update(msg tea.Msg) (Widget, tea.Cmd, Signal) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return w, nil, Continue
	}
	switch key.Type {
	case tea.KeyEsc:
		return w, nil, Cancel
	case tea.KeyTab:
		w.moveFocus(1)
		return w, nil, Continue
	case tea.KeyShiftTab:
		w.moveFocus(-1)
		return w, nil, Continue
	case tea.KeyEnter:
		w.moveVertical(1)
		return w, nil, Continue
	case tea.KeyLeft:
		w.moveLeft()
		return w, nil, Continue
	case tea.KeyRight:
		w.moveRight()
		return w, nil, Continue
	case tea.KeyUp:
		w.moveVertical(-1)
		return w, nil, Continue
	case tea.KeyDown:
		w.moveVertical(1)
		return w, nil, Continue
	case tea.KeyBackspace:
		w.backspace()
		return w, nil, Continue
	case tea.KeyDelete:
		w.deleteForward()
		return w, nil, Continue
	case tea.KeyCtrlShiftDown:
		w.insertRowBelow()
		return w, nil, Continue
	case tea.KeyCtrlShiftUp:
		w.deleteFocusedRow()
		return w, nil, Continue
	case tea.KeyCtrlShiftRight:
		w.insertColRight()
		return w, nil, Continue
	case tea.KeyCtrlShiftLeft:
		w.deleteFocusedCol()
		return w, nil, Continue
	case tea.KeyCtrlL:
		w.cycleAlign()
		return w, nil, Continue
	case tea.KeySpace:
		w.insertRunes([]rune{' '})
		return w, nil, Continue
	case tea.KeyRunes:
		if len(key.Runes) > 0 {
			w.insertRunes(key.Runes)
		}
		return w, nil, Continue
	}
	return w, nil, Continue
}

func (w *tableWidget) Lines(width int) []string {
	return w.viewLines(width)
}

// Focus returns the current cell focus (row, col). row 0 is the header.
func (w *tableWidget) Focus() (row, col int) {
	return w.focusRow, w.focusCol
}

func (w *tableWidget) CommitLines() []string {
	ncols := len(w.header)
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
		minW := alignMinWidth(w.align[c])
		if widths[c] < minW {
			widths[c] = minW
		}
		if widths[c] < 3 {
			widths[c] = 3
		}
	}

	out := make([]string, 0, 2+len(w.body))
	out = append(out, formatPipeRow(w.header, widths))
	out = append(out, formatSeparator(w.align, widths))
	for _, row := range w.body {
		out = append(out, formatPipeRow(padCells(row, ncols), widths))
	}
	return out
}

func (w *tableWidget) ExitCursor(sig Signal) doc.Position {
	if sig == Cancel {
		return w.openCursor
	}
	// Best-effort: start of the block after commit.
	return doc.Position{Line: w.blockStart, Col: 0}
}

func (w *tableWidget) WouldLeave(delta int) bool {
	next := w.focusRow + delta
	return next < 0 || next >= w.rowCount()
}

func (w *tableWidget) focusFromCursor(cursor doc.Position, blockStart int, raw []string) {
	rel := cursor.Line - blockStart
	if rel < 0 {
		rel = 0
	}
	// Map source line to focus row: 0 header, 1 sep → treat as header, ≥2 body.
	switch {
	case rel <= 1:
		w.focusRow = 0
	default:
		w.focusRow = rel - 1 // body row index + 1 (header is 0)
		if w.focusRow > len(w.body) {
			w.focusRow = len(w.body)
		}
	}
	w.focusCol = 0
	w.cellCol = utf8.RuneCountInString(w.cell(w.focusRow, w.focusCol))
	_ = raw
}

// splitPipeRow splits a GFM pipe row into trimmed cell strings.
func splitPipeRow(line string) []string {
	s := strings.TrimSpace(line)
	if s == "" {
		return nil
	}
	// Strip optional leading/trailing pipes.
	if strings.HasPrefix(s, "|") {
		s = s[1:]
	}
	if strings.HasSuffix(s, "|") {
		s = s[:len(s)-1]
	}
	parts := strings.Split(s, "|")
	cells := make([]string, len(parts))
	for i, p := range parts {
		cells[i] = strings.TrimSpace(p)
	}
	return cells
}

func parseSeparator(line string, ncols int) ([]Align, bool) {
	cells := splitPipeRow(line)
	if len(cells) == 0 {
		return nil, false
	}
	aligns := make([]Align, len(cells))
	for i, c := range cells {
		a, ok := parseAlignCell(c)
		if !ok {
			return nil, false
		}
		aligns[i] = a
	}
	if len(aligns) != ncols {
		// Allow mismatch only if we can pad/truncate to header width.
		aligns = padAlign(aligns, ncols)
	}
	return aligns, true
}

func parseAlignCell(c string) (Align, bool) {
	c = strings.TrimSpace(c)
	if c == "" {
		return AlignNone, false
	}
	left := strings.HasPrefix(c, ":")
	right := strings.HasSuffix(c, ":")
	core := c
	if left {
		core = core[1:]
	}
	if right && len(core) > 0 {
		core = core[:len(core)-1]
	}
	if core == "" || strings.Trim(core, "-") != "" {
		return AlignNone, false
	}
	switch {
	case left && right:
		return AlignCenter, true
	case left:
		return AlignLeft, true
	case right:
		return AlignRight, true
	default:
		return AlignNone, true
	}
}

func padCells(cells []string, n int) []string {
	out := make([]string, n)
	copy(out, cells)
	return out
}

func padAlign(a []Align, n int) []Align {
	out := make([]Align, n)
	copy(out, a)
	return out
}

func alignMinWidth(a Align) int {
	switch a {
	case AlignCenter:
		return 5 // :---:
	case AlignLeft, AlignRight:
		return 4 // :--- or ---:
	default:
		return 3 // ---
	}
}

func formatPipeRow(cells []string, widths []int) string {
	var b strings.Builder
	b.WriteByte('|')
	for i, cell := range cells {
		b.WriteByte(' ')
		b.WriteString(cell)
		pad := widths[i] - runewidth.StringWidth(cell)
		if pad < 0 {
			pad = 0
		}
		b.WriteString(strings.Repeat(" ", pad))
		b.WriteString(" |")
	}
	return b.String()
}

func formatSeparator(align []Align, widths []int) string {
	var b strings.Builder
	b.WriteByte('|')
	for i, a := range align {
		b.WriteByte(' ')
		w := widths[i]
		switch a {
		case AlignLeft:
			b.WriteByte(':')
			b.WriteString(strings.Repeat("-", max(3, w-1)))
		case AlignRight:
			b.WriteString(strings.Repeat("-", max(3, w-1)))
			b.WriteByte(':')
		case AlignCenter:
			b.WriteByte(':')
			b.WriteString(strings.Repeat("-", max(3, w-2)))
			b.WriteByte(':')
		default:
			b.WriteString(strings.Repeat("-", max(3, w)))
		}
		b.WriteString(" |")
	}
	return b.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
