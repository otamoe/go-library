package libcommand

import (
	"time"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

type (
	Command struct {
		logger *zap.Logger
	}
)

func (command *Command) Command(name string, worker int, slowQuery time.Duration) *Name {
	return &Name{
		logger:    command.logger.Named(name),
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

func NewCommand(logger *zap.Logger) *Command {
	command := &Command{
		logger: logger.Named("command"),
	}
	return command
}
