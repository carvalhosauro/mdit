package editor

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"github.com/carvalhosauro/mdit/internal/doc"
	"github.com/carvalhosauro/mdit/internal/mdparse"
	"github.com/carvalhosauro/mdit/internal/render"
	"github.com/carvalhosauro/mdit/internal/theme"
)

// TestMain forces a color profile so lipgloss emits ANSI even without a TTY;
// assertions strip ANSI so color does not affect them.
func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	lipgloss.SetHasDarkBackground(true)
	os.Exit(m.Run())
}

// fixture: heading + long paragraph + blank + 3-line table + blank + fence.
var fixtureLines = []string{
	"# Title",
	"This is a fairly long paragraph that wraps across lines.",
	"",
	"| A | B |",
	"| - | - |",
	"| 1 | 2 |",
	"",
	"```",
	"code line one",
	"```",
}

func newFixture(t *testing.T) Model {
	t.Helper()
	d := doc.NewFromString(strings.Join(fixtureLines, "\n"))
	m := New(d, theme.DefaultDark(), nil)
	m.SetSize(20, 6)
	return m
}

func stripLines(s string) []string {
	raw := strings.Split(s, "\n")
	out := make([]string, len(raw))
	for i, ln := range raw {
		out[i] = ansi.Strip(ln)
	}
	return out
}

func joinStrip(lines []string) string {
	var b strings.Builder
	for _, ln := range lines {
		b.WriteString(ansi.Strip(ln))
		b.WriteByte('\n')
	}
	return b.String()
}

// blockForLine returns the layout index whose range covers line.
func (m Model) testBlockForLine(line int) int {
	return blockIndexForLine(m.blocks, line)
}

func TestLayout_CursorBlockIsRaw(t *testing.T) {
	m := newFixture(t)
	// Cursor starts at line 0 -> heading block is raw and shows the '#'.
	cb := m.cursorBlock
	if !m.layouts[cb].raw {
		t.Fatalf("cursor block %d should be raw", cb)
	}
	if got := joinStrip(m.layouts[cb].lines); !strings.Contains(got, "# Title") {
		t.Fatalf("raw heading should contain '# Title', got %q", got)
	}
}

func TestLayout_NonCursorBlocksRendered(t *testing.T) {
	m := newFixture(t)
	// Move cursor into the paragraph (line 1) so the heading renders (no '#').
	m.cursorTo(doc.Position{Line: 1, Col: 0})
	headingBlock := m.testBlockForLine(0)
	if m.layouts[headingBlock].raw {
		t.Fatalf("heading block should be rendered, not raw, when cursor left it")
	}
	got := joinStrip(m.layouts[headingBlock].lines)
	if strings.Contains(got, "#") {
		t.Fatalf("rendered heading should not contain '#', got %q", got)
	}
	if !strings.Contains(got, "Title") {
		t.Fatalf("rendered heading should contain 'Title', got %q", got)
	}
	// Paragraph is now the raw cursor block.
	pb := m.testBlockForLine(1)
	if !m.layouts[pb].raw {
		t.Fatalf("paragraph block should be raw")
	}
}

func TestLayout_TableFullyRawWhenCursorInAnyRow(t *testing.T) {
	for _, line := range []int{3, 4, 5} {
		m := newFixture(t)
		m.cursorTo(doc.Position{Line: line, Col: 0})
		tb := m.testBlockForLine(line)
		if !m.layouts[tb].raw {
			t.Fatalf("table block should be raw when cursor at line %d", line)
		}
		got := joinStrip(m.layouts[tb].lines)
		for _, want := range []string{"| A | B |", "| - | - |", "| 1 | 2 |"} {
			if !strings.Contains(got, want) {
				t.Fatalf("raw table (cursor line %d) missing %q, got %q", line, want, got)
			}
		}
	}
}

func TestLayout_TableRenderedWhenCursorOutside(t *testing.T) {
	m := newFixture(t)
	// Cursor at heading; table is rendered.
	tb := m.testBlockForLine(4)
	if m.layouts[tb].raw {
		t.Fatalf("table should be rendered when cursor is elsewhere")
	}
	got := joinStrip(m.layouts[tb].lines)
	// Rendered GFM table uses a box separator row.
	if !strings.Contains(got, "┼") {
		t.Fatalf("rendered table should contain box separator, got %q", got)
	}
}

func TestLayout_PrefixSumsMatchHeights(t *testing.T) {
	m := newFixture(t)
	sum := 0
	for i := range m.layouts {
		if m.prefix[i] != sum {
			t.Fatalf("prefix[%d]=%d want %d", i, m.prefix[i], sum)
		}
		sum += m.layouts[i].height
		if m.layouts[i].height != len(m.layouts[i].lines) {
			t.Fatalf("block %d height %d != len(lines) %d", i, m.layouts[i].height, len(m.layouts[i].lines))
		}
	}
	if m.prefix[len(m.layouts)] != sum {
		t.Fatalf("final prefix %d want %d", m.prefix[len(m.layouts)], sum)
	}
}

func TestLayout_ViewReturnsExactlyHeightLines(t *testing.T) {
	m := newFixture(t)
	lines := strings.Split(m.View(), "\n")
	if len(lines) != 6 {
		t.Fatalf("View should return exactly 6 lines, got %d", len(lines))
	}
}

// build a 100-heading doc: 100 lines -> 100 blocks, each height 1.
func hundredBlockDoc() *doc.Document {
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = "# Heading " + itoa(i)
	}
	return doc.NewFromString(strings.Join(lines, "\n"))
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func TestLayout_Virtualization(t *testing.T) {
	d := hundredBlockDoc()
	m := New(d, theme.DefaultDark(), nil)

	count := 0
	m.renderBlock = func(res mdparse.Result, i int, ctx render.Context) []string {
		count++
		return render.Block(res, i, ctx)
	}
	m.SetSize(40, 6)

	// View is exactly height lines.
	if got := len(strings.Split(m.View(), "\n")); got != 6 {
		t.Fatalf("View should be 6 lines, got %d", got)
	}
	// 100 blocks, cursor on block 0 (raw) -> 99 rendered initially.
	if len(m.blocks) != 100 {
		t.Fatalf("expected 100 blocks, got %d", len(m.blocks))
	}
	// prefix sums == total heights (all headings height 1) == 100.
	if m.prefix[len(m.blocks)] != 100 {
		t.Fatalf("total height should be 100, got %d", m.prefix[len(m.blocks)])
	}

	// Move the cursor down to the last block; scroll must follow so the cursor
	// row is within the viewport. Each block is rendered at most once ever
	// (cache), so total render calls never exceed the block count.
	for i := 0; i < 100; i++ {
		m.cursorTo(doc.Position{Line: minInt(m.cursor.Line+1, 99), Col: 0})
	}
	if m.cursor.Line != 99 {
		t.Fatalf("cursor should be at last line, got %d", m.cursor.Line)
	}
	row, _ := m.cursorScreenRowCol()
	if row < m.scroll || row >= m.scroll+m.height {
		t.Fatalf("cursor row %d not within viewport [%d,%d)", row, m.scroll, m.scroll+m.height)
	}
	if count > len(m.blocks) {
		t.Fatalf("re-rendered blocks: %d render calls for %d blocks", count, len(m.blocks))
	}
	// View still exactly height lines at the bottom.
	if got := len(strings.Split(m.View(), "\n")); got != 6 {
		t.Fatalf("View at bottom should be 6 lines, got %d", got)
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestLayout_EditReflectsInViewAndUndoRestores(t *testing.T) {
	m := newFixture(t)
	before := m.View()
	// Insert a rune at the cursor (heading line).
	m.cursorTo(doc.Position{Line: 0, Col: 1})
	m.insertText("X")
	after := m.View()
	if before == after {
		t.Fatalf("view should change after edit")
	}
	if !strings.Contains(joinStrip(strings.Split(m.View(), "\n")), "#X") {
		t.Fatalf("edited heading should contain '#X', got %q", m.View())
	}
	// Undo restores content and cursor.
	m.undo()
	if strings.Contains(m.Doc().Line(0), "#X") {
		t.Fatalf("undo should remove inserted rune, line0=%q", m.Doc().Line(0))
	}
}

func TestLayout_EmptyDoc(t *testing.T) {
	d := doc.NewFromString("")
	m := New(d, theme.DefaultDark(), nil)
	m.SetSize(20, 6)
	lines := strings.Split(m.View(), "\n")
	if len(lines) != 6 {
		t.Fatalf("empty doc View should be 6 lines, got %d", len(lines))
	}
	if m.Doc().LineCount() != 1 {
		t.Fatalf("empty doc should have 1 line, got %d", m.Doc().LineCount())
	}
	row, col := m.cursorScreenRowCol()
	if row != 0 || col != 0 {
		t.Fatalf("empty doc cursor should be at 0,0 got %d,%d", row, col)
	}
}
