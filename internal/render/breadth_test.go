package render_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/carvalhosauro/mdit/internal/mdparse"
	"github.com/carvalhosauro/mdit/internal/render"
	"github.com/carvalhosauro/mdit/internal/theme"
)

// #9: at a narrow width, long table cells must wrap (full text visible) instead
// of being hard-truncated with an ellipsis.
func TestTable_WrapsCellsInsteadOfEllipsis(t *testing.T) {
	src := strings.Join([]string{
		"| Feature | Description |",
		"| ------- | ----------- |",
		"| Alpha   | V2: chaves de sandbox reutilizaveis com texto longo |",
	}, "\n")
	res := mdparse.Parse(strings.Split(src, "\n"))
	lines := render.Block(res, 0, render.Context{Width: 40, Theme: theme.DefaultDark()})
	joined := ansi.Strip(strings.Join(lines, "\n"))
	if strings.Contains(joined, "…") {
		t.Fatalf("table must not ellipsis-truncate cells, got:\n%s", joined)
	}
	compact := strings.ReplaceAll(joined, "\n", "")
	compact = strings.ReplaceAll(compact, " ", "")
	compact = strings.ReplaceAll(compact, "│", "")
	if !strings.Contains(compact, "reutilizaveis") || !strings.Contains(compact, "textolongo") {
		t.Fatalf("full cell text must remain visible via wrap, got:\n%s", joined)
	}
}

func TestCallout_WarningStyled(t *testing.T) {
	src := "> [!warning] Careful\n> body of the callout"
	res := mdparse.Parse(strings.Split(src, "\n"))
	lines := render.Block(res, 0, render.Context{Width: 40, Theme: theme.DefaultDark()})
	joined := ansi.Strip(strings.Join(lines, "\n"))
	if !strings.Contains(joined, "⚠") && !strings.Contains(strings.ToLower(joined), "warning") {
		t.Fatalf("callout should show warning marker, got %q", joined)
	}
	if !strings.Contains(joined, "body of the callout") {
		t.Fatalf("callout body lost: %q", joined)
	}
	// Must carry distinct ANSI vs a plain blockquote.
	plain := render.Block(mdparse.Parse([]string{"> just a quote"}), 0, render.Context{Width: 40, Theme: theme.DefaultDark()})
	if lines[0] == plain[0] {
		t.Fatal("callout header should style differently from plain blockquote")
	}
}

func TestCallout_UnknownFallsBackToQuote(t *testing.T) {
	src := "> [!alien] hi\n> x"
	res := mdparse.Parse(strings.Split(src, "\n"))
	lines := render.Block(res, 0, render.Context{Width: 40, Theme: theme.DefaultDark()})
	joined := ansi.Strip(strings.Join(lines, "\n"))
	// Unknown type still renders content; marker may be generic quote bar.
	if !strings.Contains(joined, "hi") && !strings.Contains(joined, "x") {
		t.Fatalf("unknown callout must still show text, got %q", joined)
	}
}

func TestHighlight_DoubleEquals(t *testing.T) {
	res := mdparse.Parse([]string{"see ==marked== text"})
	lines := render.Block(res, 0, render.Context{Width: 40, Theme: theme.DefaultDark()})
	if !strings.Contains(ansi.Strip(lines[0]), "marked") {
		t.Fatalf("highlight text lost: %q", lines[0])
	}
	if !strings.Contains(lines[0], "\x1b[") {
		t.Fatalf("highlight should be styled, got %q", lines[0])
	}
	// Single = must not trigger.
	plain := render.Block(mdparse.Parse([]string{"see =marked= text"}), 0, render.Context{Width: 40, Theme: theme.DefaultDark()})
	if strings.Contains(ansi.Strip(plain[0]), "==") {
		t.Fatalf("unexpected: %q", plain[0])
	}
	if ansi.Strip(plain[0]) != "see =marked= text" {
		t.Fatalf("single equals should stay literal, got %q", ansi.Strip(plain[0]))
	}
}

func TestTypographer_CurlyQuotesVisualOnly(t *testing.T) {
	src := `"hello"`
	res := mdparse.Parse([]string{src})
	lines := render.Block(res, 0, render.Context{Width: 40, Theme: theme.DefaultDark()})
	got := ansi.Strip(lines[0])
	if got == `"hello"` {
		t.Fatalf("typographer should curl quotes in render, got %q", got)
	}
	if !strings.Contains(got, "hello") {
		t.Fatalf("text lost: %q", got)
	}
	// Source bytes in the parse input stay ASCII — typographer is render-side.
	if string(res.Source) != src {
		t.Fatalf("Source mutated: %q", res.Source)
	}
}
