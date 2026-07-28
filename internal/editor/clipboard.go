package editor

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// osc52Writer is the sink for OSC 52 clipboard sequences. Tests may replace it.
var osc52Writer io.Writer = os.Stdout

// copyOSC52 returns a command that asks the terminal to set the system
// clipboard via OSC 52. It is best-effort: terminals may ignore or strip it
// (especially over SSH with allowPaste disabled). The internal register is
// always updated independently of this command.
func copyOSC52(text string) tea.Cmd {
	if text == "" {
		return nil
	}
	payload := base64.StdEncoding.EncodeToString([]byte(text))
	seq := fmt.Sprintf("\x1b]52;c;%s\a", payload)
	return func() tea.Msg {
		_, _ = io.WriteString(osc52Writer, seq)
		return nil
	}
}
