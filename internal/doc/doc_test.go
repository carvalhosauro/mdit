package doc

import "testing"

func TestNewFromString(t *testing.T) {
	d := NewFromString("a\nb\nc")
	if d.LineCount() != 3 {
		t.Fatalf("LineCount() = %d, want 3", d.LineCount())
	}
	if d.Line(1) != "b" {
		t.Fatalf("Line(1) = %q, want b", d.Line(1))
	}
	if d.Path() != "" {
		t.Fatalf("Path() = %q, want empty", d.Path())
	}
}

func TestInsertSimple(t *testing.T) {
	d := NewFromString("hello")
	pos := d.Insert(Position{Line: 0, Col: 5}, " world")
	if pos != (Position{Line: 0, Col: 11}) {
		t.Fatalf("Insert() pos = %+v, want {0 11}", pos)
	}
	if d.Line(0) != "hello world" {
		t.Fatalf("Line(0) = %q", d.Line(0))
	}
}

func TestInsertMiddle(t *testing.T) {
	d := NewFromString("helloworld")
	pos := d.Insert(Position{Line: 0, Col: 5}, " ")
	if pos != (Position{Line: 0, Col: 6}) {
		t.Fatalf("Insert() pos = %+v, want {0 6}", pos)
	}
	if d.Line(0) != "hello world" {
		t.Fatalf("Line(0) = %q", d.Line(0))
	}
}

func TestInsertNewlineSplitsLine(t *testing.T) {
	d := NewFromString("helloworld")
	pos := d.Insert(Position{Line: 0, Col: 5}, "\n")
	if d.LineCount() != 2 {
		t.Fatalf("LineCount() = %d, want 2", d.LineCount())
	}
	if d.Line(0) != "hello" || d.Line(1) != "world" {
		t.Fatalf("lines = %v", d.Lines())
	}
	if pos != (Position{Line: 1, Col: 0}) {
		t.Fatalf("Insert() pos = %+v, want {1 0}", pos)
	}
}

func TestInsertMultilineText(t *testing.T) {
	d := NewFromString("ac")
	pos := d.Insert(Position{Line: 0, Col: 1}, "X\nY\nZ")
	if d.LineCount() != 3 {
		t.Fatalf("LineCount() = %d, want 3", d.LineCount())
	}
	want := []string{"aX", "Y", "Zc"}
	got := d.Lines()
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("lines = %v, want %v", got, want)
		}
	}
	if pos != (Position{Line: 2, Col: 1}) {
		t.Fatalf("Insert() pos = %+v, want {2 1}", pos)
	}
}

func TestDeleteRangeSameLine(t *testing.T) {
	d := NewFromString("hello world")
	pos := d.DeleteRange(Position{Line: 0, Col: 5}, Position{Line: 0, Col: 11})
	if d.Line(0) != "hello" {
		t.Fatalf("Line(0) = %q", d.Line(0))
	}
	if pos != (Position{Line: 0, Col: 5}) {
		t.Fatalf("DeleteRange() pos = %+v, want {0 5}", pos)
	}
}

func TestDeleteRangeCrossingLinesJoins(t *testing.T) {
	d := NewFromString("hello\nworld\nfoo")
	pos := d.DeleteRange(Position{Line: 0, Col: 3}, Position{Line: 2, Col: 1})
	if d.LineCount() != 1 {
		t.Fatalf("LineCount() = %d, want 1, lines=%v", d.LineCount(), d.Lines())
	}
	if d.Line(0) != "heloo" {
		t.Fatalf("Line(0) = %q, want heloo", d.Line(0))
	}
	if pos != (Position{Line: 0, Col: 3}) {
		t.Fatalf("DeleteRange() pos = %+v, want {0 3}", pos)
	}
}

func TestDeleteBackwardAtCol0JoinsWithPreviousLine(t *testing.T) {
	d := NewFromString("hello\nworld")
	pos := d.DeleteBackward(Position{Line: 1, Col: 0})
	if d.LineCount() != 1 {
		t.Fatalf("LineCount() = %d, want 1", d.LineCount())
	}
	if d.Line(0) != "helloworld" {
		t.Fatalf("Line(0) = %q", d.Line(0))
	}
	if pos != (Position{Line: 0, Col: 5}) {
		t.Fatalf("DeleteBackward() pos = %+v, want {0 5}", pos)
	}
}

func TestDeleteBackwardMidLine(t *testing.T) {
	d := NewFromString("hello")
	pos := d.DeleteBackward(Position{Line: 0, Col: 5})
	if d.Line(0) != "hell" {
		t.Fatalf("Line(0) = %q", d.Line(0))
	}
	if pos != (Position{Line: 0, Col: 4}) {
		t.Fatalf("pos = %+v", pos)
	}
}

func TestDeleteBackwardAtStartOfDocIsNoop(t *testing.T) {
	d := NewFromString("hello")
	pos := d.DeleteBackward(Position{Line: 0, Col: 0})
	if d.Line(0) != "hello" {
		t.Fatalf("Line(0) = %q, want unchanged", d.Line(0))
	}
	if pos != (Position{Line: 0, Col: 0}) {
		t.Fatalf("pos = %+v", pos)
	}
}

func TestDeleteForwardMidLine(t *testing.T) {
	d := NewFromString("hello")
	pos := d.DeleteForward(Position{Line: 0, Col: 0})
	if d.Line(0) != "ello" {
		t.Fatalf("Line(0) = %q", d.Line(0))
	}
	if pos != (Position{Line: 0, Col: 0}) {
		t.Fatalf("pos = %+v", pos)
	}
}

func TestDeleteForwardAtEndOfLineJoinsNext(t *testing.T) {
	d := NewFromString("hello\nworld")
	pos := d.DeleteForward(Position{Line: 0, Col: 5})
	if d.LineCount() != 1 {
		t.Fatalf("LineCount() = %d, want 1", d.LineCount())
	}
	if d.Line(0) != "helloworld" {
		t.Fatalf("Line(0) = %q", d.Line(0))
	}
	if pos != (Position{Line: 0, Col: 5}) {
		t.Fatalf("pos = %+v", pos)
	}
}

func TestDeleteForwardAtEndOfDocIsNoop(t *testing.T) {
	d := NewFromString("hello")
	pos := d.DeleteForward(Position{Line: 0, Col: 5})
	if d.Line(0) != "hello" {
		t.Fatalf("Line(0) = %q, want unchanged", d.Line(0))
	}
	if pos != (Position{Line: 0, Col: 5}) {
		t.Fatalf("pos = %+v", pos)
	}
}

func TestVersionIncrementsOnMutation(t *testing.T) {
	d := NewFromString("hello")
	v0 := d.Version()
	d.Insert(Position{Line: 0, Col: 0}, "x")
	if d.Version() != v0+1 {
		t.Fatalf("Version() = %d, want %d", d.Version(), v0+1)
	}
	d.DeleteBackward(Position{Line: 0, Col: 1})
	if d.Version() != v0+2 {
		t.Fatalf("Version() = %d, want %d", d.Version(), v0+2)
	}
}

func TestUnicodeColumnsCountRunes(t *testing.T) {
	d := NewFromString("héllo")
	if d.LineCount() != 1 {
		t.Fatalf("LineCount = %d", d.LineCount())
	}
	pos := d.Insert(Position{Line: 0, Col: 5}, "!")
	if pos != (Position{Line: 0, Col: 6}) {
		t.Fatalf("pos = %+v, want {0 6}", pos)
	}
	if d.Line(0) != "héllo!" {
		t.Fatalf("Line(0) = %q", d.Line(0))
	}

	d2 := NewFromString("héllo")
	p := d2.DeleteRange(Position{Line: 0, Col: 1}, Position{Line: 0, Col: 2})
	if d2.Line(0) != "hllo" {
		t.Fatalf("Line(0) = %q, want hllo", d2.Line(0))
	}
	if p != (Position{Line: 0, Col: 1}) {
		t.Fatalf("pos = %+v", p)
	}
}

func TestContent(t *testing.T) {
	d := NewFromString("a\nb")
	if d.Content() != "a\nb\n" {
		t.Fatalf("Content() = %q, want %q", d.Content(), "a\nb\n")
	}
}

func TestLinesReturnsCopy(t *testing.T) {
	d := NewFromString("a\nb")
	lines := d.Lines()
	lines[0] = "mutated"
	if d.Line(0) != "a" {
		t.Fatalf("Lines() copy mutation leaked into doc: %q", d.Line(0))
	}
}
