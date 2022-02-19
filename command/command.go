package libcommand

import (
	"time"

	liblogger "github.com/otamoe/go-library/logger"
	"go.uber.org/fx"
)

type (
	Command struct {
	}
)

var logger = liblogger.Get("command")

func (command *Command) Command(name string, worker int, slowQuery time.Duration) *Name {
	return &Name{
		worker:    worker,
		slowQuery: slowQuery,
		name:      name,
		workerCH:  make(chan bool, worker),
	}
}

func New() fx.Option {
	return fx.Options(
		fx.Provide(NewCommand),
	)
}

func NewCommand() *Command {
	command := &Command{}
	return command
}
