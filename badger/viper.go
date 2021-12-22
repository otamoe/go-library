package libbadger

import (
	"strings"

	"github.com/dgraph-io/badger/v3"
	"github.com/spf13/viper"
)

func ViperValueDir(v *viper.Viper) (out OutOption) {
	valueDir := v.GetString("badger.valueDir")
	valueDir = strings.Replace(valueDir, "{name}", v.GetString("name"), -1)
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
func ViperIndexDir(v *viper.Viper) (out OutOption) {
	indexDir := v.GetString("badger.indexDir")
	indexDir = strings.Replace(indexDir, "{name}", v.GetString("name"), -1)
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
func ViperLoggerLevel(v *viper.Viper) (out OutOption) {
	out.Option = func(out badger.Options) (badger.Options, error) {
		if v.GetString("env") == "development" {
			return out.WithLoggingLevel(badger.DEBUG), nil
		} else {
			return out.WithLoggingLevel(badger.INFO), nil
		}
	}
	return
}
