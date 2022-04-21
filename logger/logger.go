package liblogger

import (
	"io"
	"os"
	"regexp"
	"strings"
	"sync"

	libviper "github.com/otamoe/go-library/viper"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

var mux sync.Mutex

// levels
var levels = make(map[string]zap.AtomicLevel)

// loggers
var loggers = make(map[string]*zap.Logger)

// core 接口
var core = &interfaceCore{}

// writers
var writers = &interfaceWriters{}

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
	libviper.SetDefault("logger.write.filename", "", "logger write path")
	libviper.SetDefault("logger.write.maxSize", 4, "logger write max size  mb")
	libviper.SetDefault("logger.write.maxBackups", 32, "logger write max backups")
	libviper.SetDefault("logger.write.maxAge", false, "logger write max max age")
	libviper.SetDefault("logger.write.compress", false, "logger write max compress")

	// 配置信息
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	enc := zapcore.NewJSONEncoder(cfg.EncoderConfig)

	writers.writers = append(writers.writers, os.Stderr)
	level := zap.NewAtomicLevelAt(zapcore.Level(zap.DebugLevel))
	core.Core = zapcore.NewCore(enc, writers, level)
	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}

	logger = logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return core
	}))
	loggers[""] = logger
	levels[""] = level
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
	if expr == "*" {
		expr = ".*"
	}
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

func AddWriter(w ...io.Writer) {
	writers.writers = append(writers.writers, w...)
}

func SetWriter(w ...io.Writer) {
	writers.Sync()
	writers.Close()
	writers.writers = w
}

func WriterSync() (err error) {
	return writers.Sync()
}

func WriterClose() (err error) {
	return writers.Sync()
}

func Viper() {
	if viper.GetString("env") == "development" {
		SetLevelRegex("*", zapcore.DebugLevel)
	} else {
		SetLevelRegex("*", zapcore.InfoLevel)
	}
	for _, s := range viper.GetStringSlice("logger.level") {
		i := strings.LastIndex(s, "=")
		if i == -1 {
			SetLevelRegex(s, zapcore.InfoLevel)
			continue
		}

		var l zapcore.Level = zapcore.InfoLevel
		if err := l.Set(s[i+1:]); err == nil {
			SetLevelRegex(s[0:i], l)
		}
	}

	if filename := viper.GetString("logger.write.filename"); filename != "" {
		w := &lumberjack.Logger{
			Filename:   filename,
			MaxSize:    viper.GetInt("logger.write.maxSize"),
			MaxBackups: viper.GetInt("logger.write.maxBackups"),
			Compress:   viper.GetBool("logger.write.compress"),
			MaxAge:     viper.GetInt("logger.write.maxAge"),
		}
		AddWriter(w)
	}

}
