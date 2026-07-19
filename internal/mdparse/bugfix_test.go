package mdparse

import (
	"unicode/utf8"

	"testing"
)

// TestWikiLinkAtIgnoresCodeSpan is the BG6 regression test: a [[...]] that falls
// inside an inline code span on the raw line must NOT be reported by WikiLinkAt,
// because the AST parser renders it as code (never a WikiLink). Matches outside
// code spans are still reported.
func TestWikiLinkAtIgnoresCodeSpan(t *testing.T) {
	// `[[note]]` — the whole line is a code span; ok must be false everywhere.
	inCode := "`[[note]]`"
	for col := 0; col <= utf8.RuneCountInString(inCode); col++ {
		if tgt, ok := WikiLinkAt(inCode, col); ok {
			t.Errorf("WikiLinkAt(%q, %d) = (%q, true), want ok=false inside code span", inCode, col, tgt)
		}
	}

	// x[[a]]y — no code span; the link is still detected.
	if tgt, ok := WikiLinkAt("x[[a]]y", 3); !ok || tgt != "a" {
		t.Errorf("WikiLinkAt(%q, 3) = (%q, %v), want (a, true)", "x[[a]]y", tgt, ok)
	}

	// a `[[x]]` [[y]] — [[x]] is inside backticks (skip), [[y]] is outside (found).
	mixed := "a `[[x]]` [[y]]"
	if tgt, ok := WikiLinkAt(mixed, 5); ok {
		t.Errorf("WikiLinkAt(%q, 5) = (%q, true), want ok=false for [[x]] inside code span", mixed, tgt)
	}
	if tgt, ok := WikiLinkAt(mixed, 12); !ok || tgt != "y" {
		t.Errorf("WikiLinkAt(%q, 12) = (%q, %v), want (y, true) for [[y]] outside code span", mixed, tgt, ok)
	}
}

// TestParseIndentedCodeKind is the BG7 regression test: a 4-space indented code
// block must classify as IndentedCode, distinct from a fenced code block's
// CodeFence.
func TestParseIndentedCodeKind(t *testing.T) {
	ls := lines("    indented code\n    line two")
	res := Parse(ls)
	assertCoverage(t, res, len(ls))

	b := findBlock(res, 0)
	if b == nil || b.Kind != IndentedCode {
		t.Fatalf("line 0 expected IndentedCode, got %+v", b)
	}
	if b.Start != 0 || b.End != 1 {
		t.Fatalf("indented code should span [0,1], got [%d,%d]", b.Start, b.End)
	}

	// A fenced code block must still classify as CodeFence, not IndentedCode.
	fenced := Parse(lines("```go\nx := 1\n```"))
	if fb := findBlock(fenced, 0); fb == nil || fb.Kind != CodeFence {
		t.Fatalf("fenced code must remain CodeFence, got %+v", fb)
	}
}
