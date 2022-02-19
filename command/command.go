package libcommand

import (
	"time"

	goLog "github.com/ipfs/go-log/v2"
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
		logger:    goLog.Logger("command." + name).Desugar(),
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
	command := &Command{
		logger: goLog.Logger("command").Desugar(),
	}
	return command
}
