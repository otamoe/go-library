package libviper

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func init() {
	SetDefault("env", "production", "Environment type  production, development, test")
	SetDefault("help", false, "print help")
}

func Parse() (err error) {
	// 自动 绑定 env 环境
	viper.AutomaticEnv()

	// 解析 flag
	if err = viper.BindPFlags(pflag.CommandLine); err != nil {
		return
	}

	return
}

func PrintDefaults() (ok bool) {
	if viper.GetBool("help") {
		pflag.PrintDefaults()
		return true
	}
	return false
}

func SetDefault(name string, value interface{}, usage string) {
	viper.SetDefault(name, value)
	err := setDefaultPFlag(pflag.CommandLine, name, value, usage)
	if err != nil {
		panic(err)
	}
	return
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
