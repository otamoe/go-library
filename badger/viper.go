package libbadger

import (
	"github.com/dgraph-io/badger/v3"
	liblogger "github.com/otamoe/go-library/logger"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func ViperLoggerLevel() (out OutOption) {
	out.Option = func(out badger.Options) (badger.Options, error) {
		if viper.GetString("env") == "development" {
			liblogger.SetLevel("badger", zap.DebugLevel)
		} else {
			liblogger.SetLevel("badger", zap.InfoLevel)
		}
		return out, nil
	}
	return
}
