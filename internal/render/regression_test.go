package render_test

import (
	"strings"
	"testing"

	"github.com/carvalhosauro/mdit/internal/mdparse"
)

// TestIndentedCodeRendersAsCode is the BG7 render regression test: an indented
// code block (Kind IndentedCode) must render as styled code preserving its
// content, not as raw/Other text.
func TestIndentedCodeRendersAsCode(t *testing.T) {
	src := "    indented code\n    line two"
	res := mdparse.Parse(strings.Split(src, "\n"))

	b := res.Blocks[0]
	if b.Kind != mdparse.IndentedCode {
		t.Fatalf("expected block 0 to be IndentedCode, got kind %d", int(b.Kind))
	}

	// Content is preserved (goldmark strips the 4-space indent).
	eq(t, stripped(t, src, 0, ctx()), []string{
		"indented code",
		"line two",
	})

	// It renders with styling (ANSI escapes present), like a code block.
	lines := renderDoc(t, src, 0, ctx())
	if len(lines) == 0 || !strings.Contains(strings.Join(lines, ""), "\x1b[") {
		t.Fatalf("expected styled (ANSI) code output, got %q", lines)
	}
}
