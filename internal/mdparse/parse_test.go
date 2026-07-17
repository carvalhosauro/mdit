package mdparse

import (
	"strings"
	"testing"

	"github.com/yuin/goldmark/ast"
)

// assertCoverage is the central invariant check: the returned blocks must cover
// every line index in [0, n-1], be contiguous (each block starts where the
// previous ended +1), and never overlap. The editor's virtualization depends on
// this, so every Parse test runs it.
func assertCoverage(t *testing.T, res Result, n int) {
	t.Helper()
	if n == 0 {
		if len(res.Blocks) != 0 {
			t.Fatalf("expected no blocks for empty input, got %d", len(res.Blocks))
		}
		return
	}
	if len(res.Blocks) == 0 {
		t.Fatalf("expected blocks covering %d lines, got none", n)
	}
	if res.Blocks[0].Start != 0 {
		t.Fatalf("first block must start at line 0, started at %d", res.Blocks[0].Start)
	}
	if last := res.Blocks[len(res.Blocks)-1].End; last != n-1 {
		t.Fatalf("last block must end at line %d, ended at %d", n-1, last)
	}
	for i, b := range res.Blocks {
		if b.Start > b.End {
			t.Fatalf("block %d has inverted range [%d,%d]", i, b.Start, b.End)
		}
		if i > 0 {
			prev := res.Blocks[i-1]
			if b.Start != prev.End+1 {
				t.Fatalf("block %d starts at %d but previous ended at %d (gap or overlap)", i, b.Start, prev.End)
			}
		}
	}
}

func lines(s string) []string {
	// Mirror how the doc package splits: drop a single trailing newline.
	s = strings.TrimSuffix(s, "\n")
	return strings.Split(s, "\n")
}

func TestParseCoverageTableDriven(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"empty", ""},
		{"single blank line", "\n"},
		{"heading only", "# Title"},
		{"heading blank paragraph", "# Title\n\nhello world"},
		{"multiline paragraph", "first line\nsecond line\nthird line"},
		{"fence with fake heading", "```go\nx := 1\n# fake heading\n```"},
		{"unclosed fence", "```go\nx := 1"},
		{"empty fence", "```\n```"},
		{"nested list", "- a\n  - b\n  - c\n- d"},
		{"table", "| a | b |\n|---|---|\n| 1 | 2 |"},
		{"blockquote", "> quote\n> line two"},
		{"thematic break", "para\n\n---\n\nmore"},
		{"frontmatter", "---\ntitle: hi\ntags: [a]\n---\n\n# Body"},
		{"frontmatter interior parses as setext", "---\ntitle: hi\n---"},
		{"frontmatter then list", "---\nk: v\n---\n- item"},
		{"setext heading equals", "Title\n===\n\npara"},
		{"setext heading dashes", "Title\n---\n\npara"},
		{"everything", "---\nt: x\n---\n# H\n\ntext one\ntext two\n\n```\ncode\n```\n\n- l1\n  - l2\n\n> q\n\n---\n\n| a |\n|---|\n| 1 |"},
		{"trailing blanks", "text\n\n\n"},
		{"leading blanks", "\n\ntext"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ls := lines(tc.src)
			res := Parse(ls)
			assertCoverage(t, res, len(ls))
			if string(res.Source) != strings.Join(ls, "\n") {
				t.Errorf("Source mismatch:\n got %q\nwant %q", res.Source, strings.Join(ls, "\n"))
			}
		})
	}
}

// findBlock returns the first block whose range contains the given line.
func findBlock(res Result, line int) *Block {
	for i := range res.Blocks {
		if line >= res.Blocks[i].Start && line <= res.Blocks[i].End {
			return &res.Blocks[i]
		}
	}
	return nil
}

func TestParseHeadingParagraph(t *testing.T) {
	ls := lines("# Title\n\nfirst\nsecond")
	res := Parse(ls)
	assertCoverage(t, res, len(ls))

	if b := findBlock(res, 0); b == nil || b.Kind != Heading {
		t.Fatalf("line 0 expected Heading, got %+v", b)
	}
	if b := findBlock(res, 1); b == nil || b.Kind != Blank {
		t.Fatalf("line 1 expected Blank, got %+v", b)
	}
	// The paragraph must be ONE block spanning lines 2-3.
	p := findBlock(res, 2)
	if p == nil || p.Kind != Paragraph {
		t.Fatalf("line 2 expected Paragraph, got %+v", p)
	}
	if p.Start != 2 || p.End != 3 {
		t.Fatalf("paragraph should span [2,3], got [%d,%d]", p.Start, p.End)
	}
	if p.Node == nil {
		t.Fatalf("paragraph Node must not be nil")
	}
}

func TestParseFenceIsSingleBlock(t *testing.T) {
	ls := lines("```go\nx := 1\n# fake heading\n```")
	res := Parse(ls)
	assertCoverage(t, res, len(ls))

	b := findBlock(res, 0)
	if b == nil || b.Kind != CodeFence {
		t.Fatalf("line 0 expected CodeFence, got %+v", b)
	}
	if b.Start != 0 || b.End != 3 {
		t.Fatalf("fence should span [0,3] (incl. delimiters), got [%d,%d]", b.Start, b.End)
	}
	// The '# fake heading' line must belong to the same fence, not a Heading.
	if fake := findBlock(res, 2); fake == nil || fake.Kind != CodeFence {
		t.Fatalf("line 2 (# fake heading) must be CodeFence, got %+v", fake)
	}
	// Exactly one block should cover the whole fence.
	count := 0
	for _, bl := range res.Blocks {
		if bl.Kind == CodeFence {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 CodeFence block, got %d", count)
	}
}

func TestParseNestedListSingleBlock(t *testing.T) {
	ls := lines("- a\n  - b\n  - c\n- d")
	res := Parse(ls)
	assertCoverage(t, res, len(ls))

	count := 0
	for _, b := range res.Blocks {
		if b.Kind == List {
			count++
			if b.Start != 0 || b.End != 3 {
				t.Fatalf("nested list should span [0,3], got [%d,%d]", b.Start, b.End)
			}
		}
	}
	if count != 1 {
		t.Fatalf("nested list must be ONE List block, got %d", count)
	}
}

func TestParseTable(t *testing.T) {
	ls := lines("| a | b |\n|---|---|\n| 1 | 2 |")
	res := Parse(ls)
	assertCoverage(t, res, len(ls))
	b := findBlock(res, 0)
	if b == nil || b.Kind != Table {
		t.Fatalf("expected Table, got %+v", b)
	}
	if b.Start != 0 || b.End != 2 {
		t.Fatalf("table should span [0,2], got [%d,%d]", b.Start, b.End)
	}
}

func TestParseBlockquote(t *testing.T) {
	ls := lines("> quote\n> line two")
	res := Parse(ls)
	assertCoverage(t, res, len(ls))
	b := findBlock(res, 0)
	if b == nil || b.Kind != Blockquote {
		t.Fatalf("expected Blockquote, got %+v", b)
	}
	if b.Start != 0 || b.End != 1 {
		t.Fatalf("blockquote should span [0,1], got [%d,%d]", b.Start, b.End)
	}
}

func TestParseThematicBreak(t *testing.T) {
	ls := lines("para\n\n---\n\nmore")
	res := Parse(ls)
	assertCoverage(t, res, len(ls))
	b := findBlock(res, 2)
	if b == nil || b.Kind != ThematicBreak {
		t.Fatalf("line 2 expected ThematicBreak, got %+v", b)
	}
	if b.Start != 2 || b.End != 2 {
		t.Fatalf("thematic break should be single line [2,2], got [%d,%d]", b.Start, b.End)
	}
}

func TestParseFrontmatter(t *testing.T) {
	ls := lines("---\ntitle: hi\ntags: [a]\n---\n\n# Body")
	res := Parse(ls)
	assertCoverage(t, res, len(ls))

	fm := findBlock(res, 0)
	if fm == nil || fm.Kind != Frontmatter {
		t.Fatalf("line 0 expected Frontmatter, got %+v", fm)
	}
	if fm.Start != 0 || fm.End != 3 {
		t.Fatalf("frontmatter should span [0,3] (--- .. ---), got [%d,%d]", fm.Start, fm.End)
	}
	if fm.Node != nil {
		t.Fatalf("frontmatter Node must be nil")
	}
	// The heading after frontmatter must be detected with correct offset ranges.
	h := findBlock(res, 5)
	if h == nil || h.Kind != Heading {
		t.Fatalf("line 5 expected Heading, got %+v", h)
	}
}

func TestParseFrontmatterNotClosed(t *testing.T) {
	// A leading '---' with no closing '---' is NOT frontmatter; the first line
	// is a thematic break (goldmark) and coverage must still hold.
	ls := lines("---\njust text\nmore text")
	res := Parse(ls)
	assertCoverage(t, res, len(ls))
	if b := findBlock(res, 0); b != nil && b.Kind == Frontmatter {
		t.Fatalf("unterminated --- must not be Frontmatter, got %+v", b)
	}
}

// TestParseNodeSegmentsAlignWithSource is the C1 regression test: when a
// document has frontmatter, goldmark must parse the FULL source so that every
// Block.Node's inline segments index correctly into Result.Source. Before the
// fix goldmark parsed only the frontmatter-stripped remainder, so segments were
// shifted left by the frontmatter prefix length and slicing them out of Source
// yielded silently wrong text.
func TestParseNodeSegmentsAlignWithSource(t *testing.T) {
	ls := lines("---\ntitle: hi\n---\n\nsee [[nota]] and **bold**")
	res := Parse(ls)
	assertCoverage(t, res, len(ls))

	// The paragraph is the last line.
	p := findBlock(res, 4)
	if p == nil || p.Kind != Paragraph || p.Node == nil {
		t.Fatalf("line 4 expected Paragraph with Node, got %+v", p)
	}

	var wl *WikiLink
	var emphText string
	err := ast.Walk(p.Node, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := node.(type) {
		case *WikiLink:
			wl = n
		case *ast.Emphasis:
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if tn, ok := c.(*ast.Text); ok {
					seg := tn.Segment
					emphText = string(res.Source[seg.Start:seg.Stop])
				}
			}
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}

	if wl == nil {
		t.Fatalf("expected a WikiLink inline in the paragraph")
	}
	// Slice the wikilink target segment out of Source: must be exactly "nota".
	got := string(res.Source[wl.Segment.Start:wl.Segment.Stop])
	if got != "nota" {
		t.Fatalf("WikiLink segment sliced from Source = %q, want %q", got, "nota")
	}
	if emphText != "bold" {
		t.Fatalf("Emphasis text sliced from Source = %q, want %q", emphText, "bold")
	}
}

// TestParseFrontmatterInteriorSetext guards the full-source-parsing trap: with
// goldmark parsing the whole document, the frontmatter interior ("title: hi"
// followed by "---") parses as a setext heading. The relabel step must absorb it
// entirely into a single Frontmatter block.
func TestParseFrontmatterInteriorSetext(t *testing.T) {
	ls := lines("---\ntitle: hi\n---")
	res := Parse(ls)
	assertCoverage(t, res, len(ls))
	if len(res.Blocks) != 1 {
		t.Fatalf("expected exactly 1 block, got %d: %+v", len(res.Blocks), res.Blocks)
	}
	fm := res.Blocks[0]
	if fm.Kind != Frontmatter || fm.Start != 0 || fm.End != 2 {
		t.Fatalf("expected Frontmatter block [0,2], got %+v", fm)
	}
	if fm.Node != nil {
		t.Fatalf("Frontmatter Node must be nil, got %v", fm.Node)
	}
}

// TestParseSetextHeadingEquals is the I1 regression test for '=' underlines: the
// text line plus its '===' underline must be ONE Heading block, not a heading
// followed by an Other/ThematicBreak gap block.
func TestParseSetextHeadingEquals(t *testing.T) {
	ls := lines("Title\n===\n\npara")
	res := Parse(ls)
	assertCoverage(t, res, len(ls))

	h := findBlock(res, 0)
	if h == nil || h.Kind != Heading {
		t.Fatalf("line 0 expected Heading, got %+v", h)
	}
	if h.Start != 0 || h.End != 1 {
		t.Fatalf("setext heading should span [0,1], got [%d,%d]", h.Start, h.End)
	}
	// The underline line must belong to the SAME heading block.
	if u := findBlock(res, 1); u == nil || u.Kind != Heading || u.Node != h.Node {
		t.Fatalf("line 1 (===) must be part of the Heading block, got %+v", u)
	}
	count := 0
	for _, b := range res.Blocks {
		if b.Kind == Heading {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 Heading block, got %d", count)
	}
}

// TestParseSetextHeadingDashes is the I1 regression test for '-' underlines,
// which are the dangerous case: '---' would otherwise be labeled ThematicBreak.
func TestParseSetextHeadingDashes(t *testing.T) {
	ls := lines("Title\n---\n\npara")
	res := Parse(ls)
	assertCoverage(t, res, len(ls))

	h := findBlock(res, 0)
	if h == nil || h.Kind != Heading {
		t.Fatalf("line 0 expected Heading, got %+v", h)
	}
	if h.Start != 0 || h.End != 1 {
		t.Fatalf("setext heading should span [0,1], got [%d,%d]", h.Start, h.End)
	}
	if u := findBlock(res, 1); u == nil || u.Kind != Heading {
		t.Fatalf("line 1 (---) must be Heading, not %+v", u)
	}
	// Must not be classified as a ThematicBreak.
	for _, b := range res.Blocks {
		if b.Kind == ThematicBreak {
			t.Fatalf("setext '---' underline must not be a ThematicBreak block: %+v", b)
		}
	}
}

func TestParseBlankNodesNil(t *testing.T) {
	ls := lines("# H\n\ntext")
	res := Parse(ls)
	assertCoverage(t, res, len(ls))
	b := findBlock(res, 1)
	if b == nil || b.Kind != Blank {
		t.Fatalf("line 1 expected Blank, got %+v", b)
	}
	if b.Node != nil {
		t.Fatalf("Blank Node must be nil")
	}
}
