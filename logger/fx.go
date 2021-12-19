package liblogger

import (
	"context"
	"errors"
	"strings"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

type (
	FXLogger struct {
		*zap.Logger
	}
)

func NewFX(logger *zap.Logger, stop bool) (fxOption fx.Option) {
	if logger == nil {
		logger = GetLogger()
	}
	if logger == nil {
		return fx.Error(errors.New("logger is nil"))
	}
	return fx.Provide(func(lc fx.Lifecycle) (out *zap.Logger) {
		lc.Append(fx.Hook{
			OnStop: func(c context.Context) error {
				if stop {
					return logger.Sync()
				}
				return nil
			},
		})
		return out
	})
}

func WithFXLogger(logger *zap.Logger) fx.Option {
	if logger == nil {
		logger = GetLogger().Named("fx")
	}
	return fx.WithLogger(func() fxevent.Logger {
		return &FXLogger{logger}
	})
}

func (l *FXLogger) LogEvent(event fxevent.Event) {
	switch e := event.(type) {
	case *fxevent.OnStartExecuting:
		l.Info("HOOK OnStart executing", zap.String("FunctionName", e.FunctionName), zap.String("CallerName", e.CallerName))
	case *fxevent.OnStartExecuted:
		if e.Err != nil {
			l.Error("HOOK OnStart failed", zap.String("FunctionName", e.FunctionName), zap.String("CallerName", e.CallerName), zap.Duration("Runtime", e.Runtime), zap.Error(e.Err))
		} else {
			l.Info("HOOK OnStart successfully", zap.String("FunctionName", e.FunctionName), zap.String("CallerName", e.CallerName), zap.Duration("Runtime", e.Runtime))
		}
	case *fxevent.OnStopExecuting:
		l.Info("HOOK OnStop executing", zap.String("FunctionName", e.FunctionName), zap.String("CallerName", e.CallerName))
	case *fxevent.OnStopExecuted:
		if e.Err != nil {
			l.Error("HOOK OnStop failed", zap.String("FunctionName", e.FunctionName), zap.String("CallerName", e.CallerName), zap.Duration("Runtime", e.Runtime), zap.Error(e.Err))
		} else {
			l.Info("HOOK OnStop successfully", zap.String("FunctionName", e.FunctionName), zap.String("CallerName", e.CallerName), zap.Duration("Runtime", e.Runtime))
		}
	case *fxevent.Supplied:
		if e.Err != nil {
			l.Error("Failed to supply", zap.String("TypeName", e.TypeName), zap.Error(e.Err))
		} else {
			l.Info("SUPPLY", zap.String("TypeName", e.TypeName))
		}
	case *fxevent.Provided:
		for _, rtype := range e.OutputTypeNames {
			l.Info("PROVIDE", zap.String("rtype", rtype), zap.String("ConstructorName", e.ConstructorName))
		}
		if e.Err != nil {
			l.Error("after options were applied", zap.Error(e.Err))
		}
	case *fxevent.Invoking:
		l.Info("INVOKE", zap.String("FunctionName", e.FunctionName))
	case *fxevent.Invoked:
		if e.Err != nil {
			l.Error("fx.Invoke Failed", zap.String("FunctionName", e.FunctionName), zap.String("Trace", e.Trace), zap.Error(e.Err))
		}
	case *fxevent.Stopping:
		l.Info(strings.ToUpper(e.Signal.String()))
	case *fxevent.Stopped:
		if e.Err != nil {
			l.Error("Failed to stop cleanly", zap.Error(e.Err))
		}
	case *fxevent.RollingBack:
		l.Error("Start failed, rolling back", zap.Error(e.StartErr))
	case *fxevent.RolledBack:
		if e.Err != nil {
			l.Error("Couldn't roll back cleanly", zap.Error(e.Err))
		}
	case *fxevent.Started:
		if e.Err != nil {
			l.Error("Failed to start", zap.Error(e.Err))
		} else {
			l.Info("RUNNING")
		}
	case *fxevent.LoggerInitialized:
		if e.Err != nil {
			l.Error("Failed to initialize custom logger", zap.Error(e.Err))
		} else {
			l.Info("Initialized custom logger", zap.String("ConstructorName", e.ConstructorName))
		}
	}
}
