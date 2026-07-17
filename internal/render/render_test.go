package render_test

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/termenv"

	"github.com/carvalhosauro/mdit/internal/mdparse"
	"github.com/carvalhosauro/mdit/internal/render"
	"github.com/carvalhosauro/mdit/internal/theme"
)

// TestMain forces a color profile so lipgloss actually emits ANSI sequences even
// when tests run without a TTY. Golden assertions strip ANSI, so color does not
// affect them; the two smoke tests rely on the sequences being present.
func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	lipgloss.SetHasDarkBackground(true)
	os.Exit(m.Run())
}

const width = 40

func ctx() render.Context {
	return render.Context{Width: width, Theme: theme.DefaultDark()}
}

// renderDoc parses the given markdown text and renders block i, returning the
// screen lines.
func renderDoc(t *testing.T, text string, i int, c render.Context) []string {
	t.Helper()
	res := mdparse.Parse(strings.Split(text, "\n"))
	if i >= len(res.Blocks) {
		t.Fatalf("block %d out of range (%d blocks) for %q", i, len(res.Blocks), text)
	}
	return render.Block(res, i, c)
}

// stripped renders and strips ANSI from every line.
func stripped(t *testing.T, text string, i int, c render.Context) []string {
	t.Helper()
	lines := renderDoc(t, text, i, c)
	out := make([]string, len(lines))
	for j, l := range lines {
		out[j] = ansi.Strip(l)
	}
	return out
}

func eq(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("line count: got %d %q, want %d %q", len(got), got, len(want), want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("line %d: got %q, want %q\nfull got: %q", i, got[i], want[i], got)
		}
	}
}

func TestHeadingStripsHash(t *testing.T) {
	eq(t, stripped(t, "# Título", 0, ctx()), []string{"Título"})
}

func TestHeadingLevels(t *testing.T) {
	eq(t, stripped(t, "## Sub", 0, ctx()), []string{"Sub"})
	eq(t, stripped(t, "###### Deep", 0, ctx()), []string{"Deep"})
}

func TestParagraphWordWrap(t *testing.T) {
	src := "This is **bold** text with *italic* and `code` inside it that should wrap nicely."
	eq(t, stripped(t, src, 0, ctx()), []string{
		"This is bold text with italic and code",
		"inside it that should wrap nicely.",
	})
}

func TestNestedList(t *testing.T) {
	src := "- one\n- two\n  - nested a\n  - nested b"
	eq(t, stripped(t, src, 0, ctx()), []string{
		"• one",
		"• two",
		"  ◦ nested a",
		"  ◦ nested b",
	})
}

func TestOrderedList(t *testing.T) {
	src := "1. first\n2. second"
	eq(t, stripped(t, src, 0, ctx()), []string{
		"1. first",
		"2. second",
	})
}

func TestTaskList(t *testing.T) {
	src := "- [ ] todo\n- [x] done"
	eq(t, stripped(t, src, 0, ctx()), []string{
		"☐ todo",
		"☑ done",
	})
}

func TestBlockquote(t *testing.T) {
	eq(t, stripped(t, "> quoted text here", 0, ctx()), []string{
		"│ quoted text here",
	})
}

func TestFenceKeepsContent(t *testing.T) {
	src := "```go\nfunc main() {}\n```"
	lines := stripped(t, src, 0, ctx())
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "func main() {}") {
		t.Fatalf("fence content lost: %q", lines)
	}
}

func TestTableAligned(t *testing.T) {
	src := "| A | B |\n| - | - |\n| 1 | 2 |"
	eq(t, stripped(t, src, 0, ctx()), []string{
		"A │ B",
		"──┼──",
		"1 │ 2",
	})
}

func TestWikiLinkPlain(t *testing.T) {
	eq(t, stripped(t, "[[nota]]", 0, ctx()), []string{"nota"})
}

func TestWikiLinkAlias(t *testing.T) {
	eq(t, stripped(t, "[[nota|apelido]]", 0, ctx()), []string{"apelido"})
}

func TestThematicBreak(t *testing.T) {
	eq(t, stripped(t, "---\n\ntext", 0, ctx()), []string{strings.Repeat("─", width)})
}

func TestFrontmatter(t *testing.T) {
	src := "---\ntitle: Test\n---\n\nbody"
	eq(t, stripped(t, src, 0, ctx()), []string{
		"---",
		"title: Test",
		"---",
	})
}

func TestBlankBlock(t *testing.T) {
	// "a\n\nb" → block 1 is the blank line between two paragraphs.
	eq(t, stripped(t, "a\n\nb", 1, ctx()), []string{""})
}

// --- style smoke tests ---

func TestHeadingHasANSI(t *testing.T) {
	lines := renderDoc(t, "# Título", 0, ctx())
	if len(lines) != 1 || !strings.Contains(lines[0], "\x1b[") {
		t.Fatalf("expected ANSI in heading output, got %q", lines)
	}
}

func TestBrokenWikiLinkDiffers(t *testing.T) {
	ok := ctx()
	broken := ctx()
	broken.IsBroken = func(target string) bool { return true }
	okOut := render.Block(mdparse.Parse([]string{"[[nota]]"}), 0, ok)
	brOut := render.Block(mdparse.Parse([]string{"[[nota]]"}), 0, broken)
	if okOut[0] == brOut[0] {
		t.Fatalf("broken wikilink should render differently: %q", okOut[0])
	}
	// Both must strip to the same plain text.
	if ansi.Strip(okOut[0]) != "nota" || ansi.Strip(brOut[0]) != "nota" {
		t.Fatalf("wikilink text changed: ok=%q broken=%q", okOut[0], brOut[0])
	}
}

// --- width / height invariant across all fixture blocks ---

func TestInvariantWidthAndHeight(t *testing.T) {
	fixture := strings.Join([]string{
		"---",
		"title: Test",
		"---",
		"",
		"# Título",
		"",
		"This is **bold** text with *italic* and `code` inside it that should wrap nicely.",
		"",
		"- one",
		"- two",
		"  - nested a",
		"- [ ] todo",
		"- [x] done",
		"",
		"1. first",
		"2. second",
		"",
		"> quoted text that is quite long and definitely needs to wrap at forty columns wide",
		"",
		"```go",
		"func main() { fmt.Println(\"a very long line of code that exceeds forty columns for sure yes\") }",
		"```",
		"",
		"| Column One | Column Two | Column Three | Column Four |",
		"| - | - | - | - |",
		"| aaaaa | bbbbb | ccccc | ddddd |",
		"",
		"See [[nota|apelido]] and [[broken]] here.",
		"",
		"---",
		"",
		"plain text",
	}, "\n")

	res := mdparse.Parse(strings.Split(fixture, "\n"))
	c := ctx()
	c.IsBroken = func(target string) bool { return target == "broken" }

	for i, b := range res.Blocks {
		out := render.Block(res, i, c)
		if len(out) < 1 {
			t.Fatalf("block %d (kind %v) returned empty slice", i, b.Kind)
		}
		for j, line := range out {
			w := runewidth.StringWidth(ansi.Strip(line))
			if w > width {
				t.Fatalf("block %d (kind %v) line %d width %d > %d: %q",
					i, b.Kind, j, w, width, ansi.Strip(line))
			}
		}
	}
}
