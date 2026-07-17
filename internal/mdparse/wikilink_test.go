package mdparse

import (
	"testing"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// collectWikiLinks walks the parsed AST and returns every WikiLink node found.
func collectWikiLinks(t *testing.T, src string) []*WikiLink {
	t.Helper()
	md := Markdown()
	root := md.Parser().Parse(text.NewReader([]byte(src)))
	var out []*WikiLink
	err := ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if wl, ok := n.(*WikiLink); ok {
			out = append(out, wl)
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		t.Fatalf("walk error: %v", err)
	}
	return out
}

func TestWikiLinkParseTargetOnly(t *testing.T) {
	wls := collectWikiLinks(t, "x [[nota]] y")
	if len(wls) != 1 {
		t.Fatalf("expected 1 wikilink, got %d", len(wls))
	}
	if wls[0].Target != "nota" {
		t.Errorf("Target = %q, want %q", wls[0].Target, "nota")
	}
	if wls[0].Alias != "" {
		t.Errorf("Alias = %q, want empty", wls[0].Alias)
	}
	if wls[0].Label() != "nota" {
		t.Errorf("Label() = %q, want %q", wls[0].Label(), "nota")
	}
}

func TestWikiLinkParseWithAlias(t *testing.T) {
	wls := collectWikiLinks(t, "[[nota|apelido]]")
	if len(wls) != 1 {
		t.Fatalf("expected 1 wikilink, got %d", len(wls))
	}
	if wls[0].Target != "nota" {
		t.Errorf("Target = %q, want %q", wls[0].Target, "nota")
	}
	if wls[0].Alias != "apelido" {
		t.Errorf("Alias = %q, want %q", wls[0].Alias, "apelido")
	}
	if wls[0].Label() != "apelido" {
		t.Errorf("Label() = %q, want %q", wls[0].Label(), "apelido")
	}
}

func TestWikiLinkMultiple(t *testing.T) {
	wls := collectWikiLinks(t, "[[a]] and [[b]]")
	if len(wls) != 2 {
		t.Fatalf("expected 2 wikilinks, got %d", len(wls))
	}
	if wls[0].Target != "a" || wls[1].Target != "b" {
		t.Errorf("targets = %q,%q want a,b", wls[0].Target, wls[1].Target)
	}
}

func TestWikiLinkNotALink(t *testing.T) {
	if wls := collectWikiLinks(t, "[not a link]"); len(wls) != 0 {
		t.Fatalf("expected 0 wikilinks for [not a link], got %d", len(wls))
	}
	// A regular markdown link must NOT become a wikilink.
	if wls := collectWikiLinks(t, "[text](http://x)"); len(wls) != 0 {
		t.Fatalf("expected 0 wikilinks for a regular link, got %d", len(wls))
	}
	// Single brackets are not wikilinks.
	if wls := collectWikiLinks(t, "[[unterminated"); len(wls) != 0 {
		t.Fatalf("expected 0 wikilinks for unterminated, got %d", len(wls))
	}
}

func TestWikiLinkNotParsedAsRegularLink(t *testing.T) {
	// [[x]] must be a WikiLink, never a regular ast.Link.
	md := Markdown()
	root := md.Parser().Parse(text.NewReader([]byte("[[page]](ignored)")))
	var links, wikis int
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n.(type) {
		case *WikiLink:
			wikis++
		case *ast.Link:
			links++
		}
		return ast.WalkContinue, nil
	})
	if wikis != 1 {
		t.Fatalf("expected 1 wikilink, got %d (links=%d)", wikis, links)
	}
}

func TestWikiLinkAt(t *testing.T) {
	cases := []struct {
		name    string
		line    string
		col     int
		wantTgt string
		wantOK  bool
	}{
		{"inside target", "x [[nota]] y", 5, "nota", true},
		{"on opening bracket", "x [[nota]] y", 2, "nota", true},
		{"on closing bracket", "x [[nota]] y", 9, "nota", true},
		{"before link", "x [[nota]] y", 0, "", false},
		{"after link", "x [[nota]] y", 11, "", false},
		{"alias returns target", "[[nota|apelido]]", 3, "nota", true},
		{"no link", "plain text", 3, "", false},
		{"second of two", "[[a]] [[bb]]", 8, "bb", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotTgt, gotOK := WikiLinkAt(tc.line, tc.col)
			if gotOK != tc.wantOK || gotTgt != tc.wantTgt {
				t.Errorf("WikiLinkAt(%q, %d) = (%q, %v), want (%q, %v)",
					tc.line, tc.col, gotTgt, gotOK, tc.wantTgt, tc.wantOK)
			}
		})
	}
}

func TestWikiLinkAtRuneColumns(t *testing.T) {
	// Column is measured in runes, not bytes: multibyte text before the link
	// must not shift the detected span.
	line := "café [[nota]]"
	// runes: c a f é space [ [ n o t a ] ]  -> '[' at rune index 5
	if tgt, ok := WikiLinkAt(line, 7); !ok || tgt != "nota" {
		t.Errorf("WikiLinkAt rune col 7 = (%q, %v), want (nota, true)", tgt, ok)
	}
	if _, ok := WikiLinkAt(line, 0); ok {
		t.Errorf("WikiLinkAt rune col 0 should be false")
	}
}
