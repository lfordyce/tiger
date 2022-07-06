package cmd

import (
	"bytes"
	"io"
	"os"
	"sync"
)

const (
	// Max length of left-side progress bar text before trimming is forced
	maxLeftLength = 30
	// Amount of padding in chars between rendered progress
	// bar text and right-side terminal window edge.
	termPadding      = 1
	defaultTermWidth = 80
)

// A writer that syncs writes with a mutex and, if the output is a TTY, clears before newlines.
type consoleWriter struct {
	rawOut *os.File
	writer io.Writer
	isTTY  bool
	mutex  *sync.Mutex

	// Used for flicker-free persistent objects like the progressbars
	persistentText func()
}

func (w *consoleWriter) Write(p []byte) (n int, err error) {
	origLen := len(p)
	if w.isTTY {
		// Add a TTY code to erase till the end of line with each new line
		// TODO: check how cross-platform this is...
		p = bytes.ReplaceAll(p, []byte{'\n'}, []byte{'\x1b', '[', '0', 'K', '\n'})
	}

	w.mutex.Lock()
	n, err = w.writer.Write(p)
	if w.persistentText != nil {
		w.persistentText()
	}
	w.mutex.Unlock()

	if err != nil && n < origLen {
		return n, err
	}
	return origLen, err
}
