// Package editor implements the mdit inline-render editor widget: a bubbletea
// component that shows a markdown document with every block rendered in place,
// except the block under the cursor, which is shown as raw source so it can be
// edited. The layout is virtualized (only the visible screen slice is drawn) and
// rendered blocks are cached by content, so a keystroke re-renders at most the
// one block it touched.
package editor

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/carvalhosauro/mdit/internal/doc"
	"github.com/carvalhosauro/mdit/internal/mdparse"
	"github.com/carvalhosauro/mdit/internal/render"
	"github.com/carvalhosauro/mdit/internal/theme"
)

// FollowLinkMsg is emitted (via a tea.Cmd) when the user presses Ctrl+] on a
// wikilink; the UI layer resolves and opens Target.
type FollowLinkMsg struct{ Target string }

// AutocompleteMsg is emitted (via a tea.Cmd) right after the user types the
// second '[' of a "[[" so the UI can open the note-completion popup.
type AutocompleteMsg struct{ Query string }

// Model is the editor widget. The zero value is not usable; construct one with
// New.
type Model struct {
	doc      *doc.Document
	theme    theme.Theme
	isBroken func(string) bool

	width, height int

	cursor  doc.Position
	goalCol int  // remembered target column for vertical (up/down) motion
	scroll  int  // top screen row of the viewport
	zen     bool // when true, all blocks are rendered (no raw cursor block)

	// layout state, rebuilt by recompute.
	result        mdparse.Result
	blocks        []mdparse.Block
	layouts       []blockLayout
	prefix        []int
	cursorBlock   int
	layoutVersion int // doc.Version() the current parse reflects
	layoutWidth   int // width the current parse/layout reflects

	cache map[cacheKey][]string

	// renderBlock renders a non-cursor block; a field so tests can wrap it to
	// count render calls. Defaults to render.Block.
	renderBlock func(res mdparse.Result, i int, ctx render.Context) []string
}

// New constructs an editor Model over d with the given theme and broken-link
// predicate (isBroken may be nil). Call SetSize before View.
func New(d *doc.Document, th theme.Theme, isBroken func(string) bool) Model {
	m := Model{
		doc:           d,
		theme:         th,
		isBroken:      isBroken,
		cache:         make(map[cacheKey][]string),
		renderBlock:   render.Block,
		layoutVersion: -1,
	}
	m.recompute()
	return m
}

// SetSize sets the viewport dimensions and rebuilds the layout.
func (m *Model) SetSize(w, h int) {
	m.width, m.height = w, h
	m.recompute()
}

// Cursor returns the current cursor position (Col in runes).
func (m Model) Cursor() doc.Position { return m.cursor }

// Doc returns the underlying document.
func (m Model) Doc() *doc.Document { return m.doc }

// SetDoc swaps the document and resets cursor, scroll, and all layout history.
func (m *Model) SetDoc(d *doc.Document) {
	m.doc = d
	m.cursor = doc.Position{}
	m.goalCol = 0
	m.scroll = 0
	m.blocks = nil
	m.layouts = nil
	m.prefix = nil
	m.cache = make(map[cacheKey][]string)
	m.layoutVersion = -1
	m.recompute()
}

// SetCursor moves the cursor to p (clamped on next recompute) and rebuilds layout.
func (m *Model) SetCursor(p doc.Position) {
	m.cursor = p
	m.goalCol = p.Col
	m.recompute()
}

// InsertText inserts s at the cursor and rebuilds layout.
func (m *Model) InsertText(s string) {
	m.cursor = m.doc.Insert(m.cursor, s)
	m.goalCol = m.cursor.Col
	m.recompute()
}

// SetZen toggles read-only fully-rendered mode (no raw cursor block).
func (m *Model) SetZen(on bool) {
	m.zen = on
	m.recompute()
}

// Zen reports whether the editor is in zen (fully rendered) mode.
func (m Model) Zen() bool { return m.zen }

// Scroll returns the top screen row of the viewport.
func (m Model) Scroll() int { return m.scroll }

// SetScroll sets the viewport scroll offset (clamped).
func (m *Model) SetScroll(s int) {
	m.scroll = s
	m.clampScroll()
}

// Update handles incoming messages (currently key and window-size messages) and
// returns the updated model plus any command to run.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// View renders exactly height screen lines joined by "\n", drawing the cursor by
// inverting the cell it occupies.
func (m Model) View() string {
	if m.height < 1 {
		return ""
	}
	curRow, _ := m.cursorScreenRowCol()

	var b strings.Builder
	for i := 0; i < m.height; i++ {
		sr := m.scroll + i
		if i > 0 {
			b.WriteByte('\n')
		}
		if !m.zen && sr == curRow {
			b.WriteString(m.renderCursorRow())
			continue
		}
		b.WriteString(m.screenLine(sr))
	}
	return b.String()
}

// renderCursorRow rebuilds the cursor's screen row from raw text, styling the
// cell under the cursor with a reversed RawBlock style. The cursor block is
// always raw, so its row is a plain (single-style) segment and cell inversion is
// exact without ANSI surgery.
func (m Model) renderCursorRow() string {
	_, wr, idx, _ := m.cursorLocation()
	style := m.theme.RawBlock
	rev := style.Reverse(true)

	runes := []rune(wr.text)
	if idx >= len(runes) {
		// Cursor at end of the row: a trailing reversed space marks it.
		return style.Render(string(runes)) + rev.Render(" ")
	}
	left := string(runes[:idx])
	cur := string(runes[idx])
	right := string(runes[idx+1:])
	return style.Render(left) + rev.Render(cur) + style.Render(right)
}
