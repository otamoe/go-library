package libbadger

import (
	"strings"

	"github.com/dgraph-io/badger/v3"
	"github.com/spf13/viper"
)

func ViperValueDir() (out OutOption) {
	valueDir := viper.GetString("badger.valueDir")
	valueDir = strings.Replace(valueDir, "{name}", viper.GetString("name"), -1)
	if valueDir != "" {
		out.Option = func(out badger.Options) (badger.Options, error) {
			return out.WithValueDir(valueDir), nil
		}
	} else {
		out.Option = func(out badger.Options) (badger.Options, error) {
			return out, nil
		}
	}
	return
}
func ViperIndexDir() (out OutOption) {
	indexDir := viper.GetString("badger.indexDir")
	indexDir = strings.Replace(indexDir, "{name}", viper.GetString("name"), -1)
	if indexDir != "" {
		out.Option = func(out badger.Options) (badger.Options, error) {
			return out.WithDir(indexDir), nil
		}
	} else {
		out.Option = func(out badger.Options) (badger.Options, error) {
			return out, nil
		}
	}
	return
}

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
