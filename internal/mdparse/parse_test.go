package mdparse

import (
	"strings"
	"testing"
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
		{"frontmatter then list", "---\nk: v\n---\n- item"},
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
