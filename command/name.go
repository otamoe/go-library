package libcommand

import (
	"context"
	"io"
	"os/exec"
	"time"

	"go.uber.org/zap"
)

type (
	Name struct {
		logger    *zap.Logger
		name      string
		worker    int
		slowQuery time.Duration
		workerCH  chan bool
	}
)

func (name *Name) Run(ctx context.Context, dir string, stdin io.Reader, stdout io.Writer, stderr io.Writer, args ...string) (run *Run) {
	run = &Run{
		name:   name,
		stdin:  stdin,
		stdout: &RunWriter{Writer: stdout},
		stderr: &RunWriter{Writer: stderr},
		wait:   make(chan struct{}),
		dir:    dir,
	}

	go func() {
		// 线程结束
		var err error
		defer close(run.wait)
		defer func() {
			run.err = err
		}()

		select {
		case name.workerCH <- true:
			// 写入线程 退出线程
			defer func() {
				<-name.workerCH
			}()
		case <-ctx.Done():
			// 已取消
			err = ctx.Err()
			return
		}

		cmd := exec.CommandContext(ctx, name.name, args...)
		cmd.Stdin = run.stdin
		cmd.Stdout = run.stdout
		cmd.Stderr = run.stderr
		cmd.Dir = run.dir

		now := time.Now()
		if err = cmd.Start(); err == nil {
			err = cmd.Wait()
		}

		if !run.kill {
			if err != nil && err.Error() == "exit status 1" && run.stderr.b.Len() == 0 {
				err = nil
			}
		}

		latency := time.Now().Sub(now)
		if err != nil {
			name.logger.Error(
				"error",
				zap.Strings("args", args),
				zap.String("dir", dir),
				zap.Duration("latency", latency),
				zap.String("stdout", run.stdout.b.String()),
				zap.String("stderr", run.stderr.b.String()),
			)
		} else if name.slowQuery != 0 && latency > name.slowQuery {
			name.logger.Warn(
				"slowQuery",
				zap.Strings("args", args),
				zap.String("dir", dir),
				zap.Duration("latency", latency),
				zap.String("stdout", run.stdout.b.String()),
				zap.String("stderr", run.stderr.b.String()),
			)
		} else {
			name.logger.Info(
				"info",
				zap.Strings("args", args),
				zap.String("dir", dir),
				zap.Duration("latency", latency),
				zap.String("stdout", run.stdout.b.String()),
				zap.String("stderr", run.stderr.b.String()),
			)
		}
	}()

	return
}
