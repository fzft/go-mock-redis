package linenoise

import (
	"bytes"
	"fmt"
	"github.com/peterh/liner"
	"os"
)

var Line *LineNoise

type LineNoise struct {
	*liner.State
}

func (ln *LineNoise) HistoryLoad(filepath string) error {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}
	_, err = ln.ReadHistory(bytes.NewReader(content))
	return err
}

func (ln *LineNoise) HistorySave(filepath string) error {
	var buf bytes.Buffer
	_, err := ln.WriteHistory(&buf)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath, buf.Bytes(), 0644)
}

func (ln *LineNoise) ClearScreen() (string, error) {
	clearSeq := "\x1b[H\x1b[2J"
	_, err := fmt.Fprint(os.Stdout, clearSeq)
	if err != nil {
		// Handle error or ignore, based on requirements.
	}
}

func init() {
	Line = &LineNoise{liner.NewLiner()}
	Line.SetCtrlCAborts(true)
}
