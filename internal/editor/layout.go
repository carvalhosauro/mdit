package editor

import (
	"hash/fnv"
	"sort"
	"strings"

	"github.com/mattn/go-runewidth"

	"github.com/carvalhosauro/mdit/internal/mdparse"
	"github.com/carvalhosauro/mdit/internal/render"
)

// blockLayout is the materialized screen representation of a single markdown
// block: either the block's raw source lines (when the cursor sits inside it)
// or the styled render.Block output, already soft-wrapped to the current width.
type blockLayout struct {
	raw    bool     // true when the block is under the cursor (shown as raw text)
	lines  []string // screen-ready lines, each <= width cells
	height int      // == len(lines); duplicated for readable prefix-sum code
}

// cacheKey identifies a rendered (non-raw) block by the content it was rendered
// from and the width it was wrapped to. Keying on content hash (not block index)
// means editing one block never invalidates the cached renders of the others,
// and index shifts from inserting/removing lines do not matter.
type cacheKey struct {
	hash  uint64
	width int
}

func hash64(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return h.Sum64()
}

// effWidth returns the effective render width, never below 1.
func (m Model) effWidth() int {
	if m.width < 1 {
		return 1
	}
	return m.width
}

// blockIndexForLine returns the index of the block whose inclusive line range
// covers line. Blocks are contiguous, sorted, and cover every line, so a binary
// search on Start finds it.
func blockIndexForLine(blocks []mdparse.Block, line int) int {
	if len(blocks) == 0 {
		return 0
	}
	i := sort.Search(len(blocks), func(i int) bool { return blocks[i].Start > line }) - 1
	if i < 0 {
		i = 0
	}
	if i >= len(blocks) {
		i = len(blocks) - 1
	}
	return i
}

// recompute rebuilds the layout after any change to the document, cursor, or
// size. It reparses only when the document version or width changed since the
// last parse; the per-block layout array is rebuilt every call, but rendered
// (non-cursor) blocks come from a content-keyed cache, so the only real render
// work happens on a cache miss — i.e. the first time a block's text is seen at
// a given width. This is the resolution of the height/virtualization tension:
// heights are always available (needed for prefix sums), yet no block is
// re-rendered per keystroke.
func (m *Model) recompute() {
	if m.width < 1 {
		return // no usable width yet; layout deferred until SetSize
	}
	if m.cache == nil {
		m.cache = make(map[cacheKey][]string)
	}

	if m.blocks == nil || m.layoutVersion != m.doc.Version() || m.layoutWidth != m.effWidth() {
		m.result = mdparse.Parse(m.doc.Lines())
		m.blocks = m.result.Blocks
		m.layoutVersion = m.doc.Version()
		m.layoutWidth = m.effWidth()
	}

	m.normalizeCursor()
	m.cursorBlock = blockIndexForLine(m.blocks, m.cursor.Line)

	m.layouts = make([]blockLayout, len(m.blocks))
	m.prefix = make([]int, len(m.blocks)+1)
	for i := range m.blocks {
		var lines []string
		raw := !m.zen && i == m.cursorBlock
		if raw {
			lines = m.rawBlockLines(i)
		} else {
			lines = m.renderedLines(i)
		}
		m.layouts[i] = blockLayout{raw: raw, lines: lines, height: len(lines)}
		m.prefix[i+1] = m.prefix[i] + len(lines)
	}

	m.followCursor()
}

// normalizeCursor clamps the cursor to a valid buffer position; the document may
// have shrunk (undo, deletion) since the cursor was last set.
func (m *Model) normalizeCursor() {
	if m.cursor.Line < 0 {
		m.cursor.Line = 0
	}
	if m.cursor.Line > m.doc.LineCount()-1 {
		m.cursor.Line = m.doc.LineCount() - 1
	}
	n := runeLen(m.doc.Line(m.cursor.Line))
	if m.cursor.Col < 0 {
		m.cursor.Col = 0
	}
	if m.cursor.Col > n {
		m.cursor.Col = n
	}
}

// renderedLines returns the cached styled render of block i, rendering (and
// caching) it on a miss.
func (m *Model) renderedLines(i int) []string {
	b := m.blocks[i]
	text := strings.Join(m.rawRange(b.Start, b.End), "\n")
	key := cacheKey{hash: hash64(text), width: m.effWidth()}
	if v, ok := m.cache[key]; ok {
		return v
	}
	lines := m.renderBlock(m.result, i, render.Context{
		Width:    m.effWidth(),
		Theme:    m.theme,
		IsBroken: m.isBroken,
	})
	m.cache[key] = lines
	return lines
}

// rawBlockLines returns the raw source lines of block i, soft-wrapped by cells
// to the current width and styled with the theme's RawBlock style.
func (m *Model) rawBlockLines(i int) []string {
	b := m.blocks[i]
	w := m.effWidth()
	style := m.theme.RawBlock
	var out []string
	for ln := b.Start; ln <= b.End; ln++ {
		for _, wr := range wrapRaw(m.doc.Line(ln), w) {
			out = append(out, style.Render(wr.text))
		}
	}
	if len(out) == 0 {
		out = []string{style.Render("")}
	}
	return out
}

// rawRange returns the raw source lines in the inclusive range [start,end].
func (m *Model) rawRange(start, end int) []string {
	out := make([]string, 0, end-start+1)
	for ln := start; ln <= end; ln++ {
		out = append(out, m.doc.Line(ln))
	}
	return out
}

// wrapRow describes one screen row produced by soft-wrapping a raw line: its
// text plus the rune column in the original line where the row begins and how
// many runes it spans. The columns let the cursor mapping locate a rune column
// within its wrapped row.
type wrapRow struct {
	text     string
	startCol int
	runeLen  int
}

// wrapRaw soft-wraps a raw line into screen rows no wider than width cells,
// measuring with runewidth so wide (CJK) runes count as two cells and are never
// split. An empty line yields a single empty row. A single rune wider than the
// width still occupies its own row (it cannot be split); the width invariant is
// then unavoidably exceeded, which only happens at degenerate widths.
func wrapRaw(line string, width int) []wrapRow {
	if width < 1 {
		width = 1
	}
	runes := []rune(line)
	if len(runes) == 0 {
		return []wrapRow{{text: "", startCol: 0, runeLen: 0}}
	}
	var rows []wrapRow
	start := 0
	w := 0
	var b strings.Builder
	for i, r := range runes {
		rw := runewidth.RuneWidth(r)
		if w+rw > width && i > start {
			rows = append(rows, wrapRow{text: b.String(), startCol: start, runeLen: i - start})
			b.Reset()
			w = 0
			start = i
		}
		b.WriteRune(r)
		w += rw
	}
	rows = append(rows, wrapRow{text: b.String(), startCol: start, runeLen: len(runes) - start})
	return rows
}

// cursorLocation resolves the cursor's absolute screen row, the wrapped row it
// falls in, the cursor's rune index within that row, and its cell column within
// the row. It is the single source of truth shared by cursorScreenRowCol,
// followCursor, and the View cursor-drawing code.
func (m Model) cursorLocation() (screenRow int, wr wrapRow, idxInRow int, cellCol int) {
	if len(m.blocks) == 0 {
		return 0, wrapRow{}, 0, 0
	}
	w := m.effWidth()
	cb := m.cursorBlock
	b := m.blocks[cb]

	rowInBlock := 0
	for ln := b.Start; ln < m.cursor.Line; ln++ {
		rowInBlock += len(wrapRaw(m.doc.Line(ln), w))
	}

	lineRunes := []rune(m.doc.Line(m.cursor.Line))
	rows := wrapRaw(m.doc.Line(m.cursor.Line), w)
	for ri, row := range rows {
		end := row.startCol + row.runeLen
		if m.cursor.Col < end || ri == len(rows)-1 {
			idx := m.cursor.Col - row.startCol
			if idx < 0 {
				idx = 0
			}
			cell := 0
			for c := row.startCol; c < m.cursor.Col && c < len(lineRunes); c++ {
				cell += runewidth.RuneWidth(lineRunes[c])
			}
			return m.prefix[cb] + rowInBlock + ri, row, idx, cell
		}
	}
	return m.prefix[cb] + rowInBlock, wrapRow{}, 0, 0
}

// cursorScreenRowCol returns the cursor's absolute screen row and cell column.
func (m Model) cursorScreenRowCol() (int, int) {
	row, _, _, cell := m.cursorLocation()
	return row, cell
}

// followCursor adjusts the scroll offset so the cursor's screen row stays within
// the viewport, then clamps scroll to the valid range. In zen mode the user
// scrolls freely (cursor is not painted), so followCursor is a no-op.
func (m *Model) followCursor() {
	if m.zen {
		m.clampScroll()
		return
	}
	row, _ := m.cursorScreenRowCol()
	if row < m.scroll {
		m.scroll = row
	}
	if m.height > 0 && row >= m.scroll+m.height {
		m.scroll = row - m.height + 1
	}
	m.clampScroll()
}

// clampScroll keeps scroll within [0, maxScroll].
func (m *Model) clampScroll() {
	total := 0
	if len(m.prefix) > 0 {
		total = m.prefix[len(m.prefix)-1]
	}
	max := total - m.height
	if max < 0 {
		max = 0
	}
	if m.scroll > max {
		m.scroll = max
	}
	if m.scroll < 0 {
		m.scroll = 0
	}
}

// screenLine returns the materialized screen line at absolute row sr, or "" when
// sr is past the end of the document.
func (m Model) screenLine(sr int) string {
	total := 0
	if len(m.prefix) > 0 {
		total = m.prefix[len(m.prefix)-1]
	}
	if sr < 0 || sr >= total {
		return ""
	}
	bi := sort.Search(len(m.prefix), func(i int) bool { return m.prefix[i] > sr }) - 1
	if bi < 0 {
		bi = 0
	}
	row := sr - m.prefix[bi]
	lines := m.layouts[bi].lines
	if row < 0 || row >= len(lines) {
		return ""
	}
	return lines[row]
}

func runeLen(s string) int {
	return len([]rune(s))
}
