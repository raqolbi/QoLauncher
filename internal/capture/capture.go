package capture

import (
	"bufio"
	"io"
	"sync"

	"github.com/raqolbi/qolauncher/internal/logwriter"
)

// Pipes captures stdout and stderr readers into the log writer concurrently.
func Pipes(stdout, stderr io.Reader, w *logwriter.Writer, wg *sync.WaitGroup) {
	if wg == nil {
		wg = &sync.WaitGroup{}
	}
	if stdout != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Stream(stdout, logwriter.StreamStdout, w)
		}()
	}
	if stderr != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Stream(stderr, logwriter.StreamStderr, w)
		}()
	}
}

// Stream reads r line-by-line and writes each line to w under stream name.
// Partial lines without trailing newline are flushed on EOF.
func Stream(r io.Reader, stream string, w *logwriter.Writer) {
	if r == nil || w == nil {
		return
	}

	reader := bufio.NewReader(r)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			msg := line
			if msg[len(msg)-1] == '\n' {
				msg = msg[:len(msg)-1]
			}
			_ = w.WriteLine(stream, msg)
		}
		if err != nil {
			return
		}
	}
}
