package libcommand

import (
	"bytes"
	"io"
)

type (
	Run struct {
		name   *Name
		stdin  io.Reader
		stdout *RunWriter
		stderr *RunWriter
		dir    string
		wait   chan struct{}
		err    error
		kill   bool
	}
	RunWriter struct {
		io.Writer
		b bytes.Buffer
	}
)

func (w *RunWriter) Write(p []byte) (n int, err error) {
	n, err = w.Writer.Write(p)
	if n > 0 {
		if _, err = w.b.Write(p[0:n]); err != nil {
			return
		}
	}
	return
}

func (run *Run) Err() (err error) {
	<-run.wait
	return run.err
}
func (run *Run) Wait() chan struct{} {
	return run.wait
}
