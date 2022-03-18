package libbadger

import (
	"github.com/dgraph-io/badger/v3"
	"github.com/spf13/viper"
)

func ViperLoggerLevel() (out OutOption) {
	out.Option = func(out badger.Options) (badger.Options, error) {
		if viper.GetString("env") == "development" {
			return out.WithLoggingLevel(badger.DEBUG), nil
		} else {
			return out.WithLoggingLevel(badger.INFO), nil
		}
	}
	return
}
