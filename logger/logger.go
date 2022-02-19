package liblogger

import (
	"regexp"
	"strings"
	"sync"

	libviper "github.com/otamoe/go-library/viper"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var mux sync.Mutex

// levels
var levels = make(map[string]zap.AtomicLevel)

// loggers
var loggers = make(map[string]*zap.Logger)

// core 接口
var core = &interfaceCore{}

//  regexLevel 匹配设置 的 level
var regexLevels = make([]regexLevel, 0)

type (
	regexLevel struct {
		regex *regexp.Regexp
		level zapcore.Level
	}

	interfaceCore struct {
		zapcore.Core
	}
)

func init() {
	libviper.SetDefault("logger.level", []string{}, "logger level")

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.Level = zap.NewAtomicLevelAt(zapcore.Level(zap.DebugLevel))

	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	core.Core = logger.Core()
	logger = logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return core
	}))
	loggers[""] = logger
	levels[""] = cfg.Level
}

func New() fx.Option {
	return fx.Options(
		fx.WithLogger(FxLogger),
	)
}

// 获得 logger
func Get(name string) (log *zap.Logger) {
	mux.Lock()
	defer mux.Unlock()
	var ok bool
	if log, ok = loggers[name]; !ok {
		var level zap.AtomicLevel
		if level, ok = levels[name]; !ok {
			level = zap.NewAtomicLevelAt(zapcore.Level(getLevel(name)))
			levels[name] = level
		}

		log = zap.New(core).WithOptions(zap.IncreaseLevel(level), zap.AddCaller())
		if name != "" {
			log = log.Named(name)
		}
		loggers[name] = log
	}
	return log
}

// 设置 level 级别 按照 正则匹配name
func SetLevelRegex(expr string, level zapcore.Level) {
	regex, err := regexp.Compile(expr)
	if err != nil {
		return
	}
	mux.Lock()
	defer mux.Unlock()
	regexLevels = append(regexLevels, regexLevel{regex, level})
	for key, val := range levels {
		if regex.MatchString(key) {
			val.SetLevel(level)
		}
	}
	return
}

// 设置 level
func SetLevel(name string, level zapcore.Level) {
	SetLevelRegex("^"+regexp.QuoteMeta(name)+"$", level)
}

// 读取 当前 name 的 level
func getLevel(name string) zapcore.Level {
	level := zapcore.InfoLevel
	for _, r := range regexLevels {
		if r.regex.MatchString(name) {
			level = r.level
		}
	}
	return level
}

func SetCore(c zapcore.Core) {
	core.Core = c
}

func Core() (c zapcore.Core) {
	return core.Core
}

func Viper() {
	for _, s := range viper.GetStringSlice("logger.level") {
		i := strings.LastIndex(s, "=")
		if i == -1 {
			SetLevelRegex(s, zapcore.InfoLevel)
			continue
		}

		var l zapcore.Level = zap.InfoLevel
		l.Set(s[i+1:])
		SetLevelRegex(s[0:i], l)
	}
}
