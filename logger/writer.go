package liblogger

import (
	"io"

	"go.uber.org/zap/zapcore"
)

type (
	interfaceWriters struct {
		writers []io.Writer
	}
)

func (w *interfaceWriters) Write(p []byte) (n int, err error) {
	for _, writer := range w.writers {
		if err == nil {
			n, err = writer.Write(p)
		} else {
			writer.Write(p)
		}
	}
	return
}

func (w *interfaceWriters) Sync() (err error) {
	for _, writer := range w.writers {
		if val, ok := writer.(zapcore.WriteSyncer); ok {
			if e := val.Sync(); e != nil && err == nil {
				err = e
			}
		}
	}
	return
}

func (w *interfaceWriters) Close() (err error) {
	for _, writer := range w.writers {
		if val, ok := writer.(io.Closer); ok {
			if e := val.Close(); e != nil && err == nil {
				err = e
			}
		}
	}
	return
}
