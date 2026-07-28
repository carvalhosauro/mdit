package editor

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCopy_OSC52WritesSequence(t *testing.T) {
	var buf bytes.Buffer
	prev := osc52Writer
	osc52Writer = &buf
	t.Cleanup(func() { osc52Writer = prev })

	m := newEditor(t, "hello world", 40, 6)
	for i := 0; i < 5; i++ {
		m, _ = key(m, typeKey(tea.KeyShiftRight))
	}
	_, cmd := key(m, typeKey(tea.KeyCtrlW))
	if cmd == nil {
		t.Fatal("copy should return an OSC 52 command")
	}
	_ = cmd()
	got := buf.String()
	wantPayload := base64.StdEncoding.EncodeToString([]byte("hello"))
	if !strings.Contains(got, "\x1b]52;c;"+wantPayload+"\a") {
		t.Fatalf("OSC 52 sequence = %q, want payload %q", got, wantPayload)
	}
}
