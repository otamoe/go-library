package liblogger

import (
	"strings"

	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

type (
	fxLogger struct {
		*zap.Logger
	}
)

func FxLogger() fxevent.Logger {
	return &fxLogger{Logger: Get("fx")}
}

func (l *fxLogger) LogEvent(event fxevent.Event) {
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
