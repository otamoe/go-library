package libviper

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

type (
	InOptions struct {
		fx.In
		Options []Option `group:"viperOptions"`
	}

	OutOption struct {
		fx.Out
		Option Option `group:"viperOptions"`
	}

	Option func(v *viper.Viper) (err error)

	InSetDefaults struct {
		fx.In
		Options []SetDefault `group:"viperSetDefaults"`
	}
	OutSetDefault struct {
		fx.Out
		Option SetDefault `group:"viperSetDefaults"`
	}
	SetDefault struct {
		Name  string
		Value interface{}
		Usage string
	}
)

func PFlag(inSetDefaults InSetDefaults) (flagSet *pflag.FlagSet, err error) {
	flagSet = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
	for _, setDefault := range inSetDefaults.Options {
		if err = setDefaultPFlag(flagSet, setDefault.Name, setDefault.Value, setDefault.Usage); err != nil {
			return
		}
	}
	if err = flagSet.Parse(os.Args[1:]); err != nil {
		return
	}
	return
}

func Viper(flagSet *pflag.FlagSet, inOptions InOptions, inSetDefaults InSetDefaults) (v *viper.Viper, err error) {
	v = viper.New()

	for _, option := range inOptions.Options {
		if err = option(v); err != nil {
			return
		}
	}

	for _, setDefault := range inSetDefaults.Options {
		v.SetDefault(setDefault.Name, setDefault.Value)
	}

	// 自动 绑定 env 环境
	v.AutomaticEnv()

	// 解析 flag
	v.BindPFlags(flagSet)

	return
}

func WithSetDefault(name string, value interface{}, usage string) func() (out OutSetDefault) {
	return func() (out OutSetDefault) {
		out.Option.Name = name
		out.Option.Value = value
		out.Option.Usage = usage
		return
	}
}

func WithEnvPrefix(name string) func() (out OutOption) {
	return func() (out OutOption) {
		out.Option = func(v *viper.Viper) (err error) {
			v.SetEnvPrefix(strings.ToUpper(name))
			return
		}
		return
	}
}

func setDefaultPFlag(flagSet *pflag.FlagSet, name string, value interface{}, usage string) (err error) {
	if value == nil {
		err = errors.New("default value is nil")
		return
	}

	name = strings.ToLower(name)

	// flag 值
	switch val := value.(type) {
	case time.Duration:
		flagSet.Duration(name, val, usage)
	case net.IP:
		flagSet.IP(name, val, usage)
	case net.IPMask:
		flagSet.IPMask(name, val, usage)
	case net.IPNet:
		flagSet.IPNet(name, val, usage)
	case string:
		flagSet.String(name, val, usage)
	case bool:
		flagSet.Bool(name, val, usage)
	case int:
		flagSet.Int(name, val, usage)
	case int8:
		flagSet.Int8(name, val, usage)
	case int16:
		flagSet.Int16(name, val, usage)
	case int32:
		flagSet.Int32(name, val, usage)
	case int64:
		flagSet.Int64(name, val, usage)
	case uint:
		flagSet.Uint(name, val, usage)
	case uint8:
		flagSet.Uint8(name, val, usage)
	case uint16:
		flagSet.Uint16(name, val, usage)
	case uint32:
		flagSet.Uint32(name, val, usage)
	case uint64:
		flagSet.Uint64(name, val, usage)
	case float32:
		flagSet.Float32(name, val, usage)
	case float64:
		flagSet.Float64(name, val, usage)
	case []byte:
		flagSet.BytesBase64(name, val, usage)

	//...
	case []time.Duration:
		flagSet.DurationSlice(name, val, usage)
	case []net.IP:
		flagSet.IPSlice(name, val, usage)
	case []string:
		flagSet.StringSlice(name, val, usage)
	case []bool:
		flagSet.BoolSlice(name, val, usage)
	case []int:
		flagSet.IntSlice(name, val, usage)
	case []int32:
		flagSet.Int32Slice(name, val, usage)
	case []int64:
		flagSet.Int64Slice(name, val, usage)
	case []uint:
		flagSet.UintSlice(name, val, usage)
	case []float32:
		flagSet.Float32Slice(name, val, usage)
	case []float64:
		flagSet.Float64Slice(name, val, usage)
	default:
		err = fmt.Errorf("default value type %T", val)
		return
	}
	return
}
