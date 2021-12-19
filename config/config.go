package libconfig

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var name = "otamoe"

func init() {
	// 设置 env 前缀
	SetEnvPrefix(name)

	// 环境类型
	SetDefault("env", "production", "Environment type  production, development, test")
}

func SetName(val string) {
	name = val
}

func GetName() string {
	return name
}

func SetDefault(key string, value interface{}, usage string) {
	if value == nil {
		panic("default value is nil")
	}

	key = strings.ToLower(key)

	// 默认值
	viper.SetDefault(key, value)

	// flag 值
	switch val := value.(type) {
	case time.Duration:
		pflag.Duration(key, val, usage)
	case net.IP:
		pflag.IP(key, val, usage)
	case net.IPMask:
		pflag.IPMask(key, val, usage)
	case net.IPNet:
		pflag.IPNet(key, val, usage)
	case string:
		pflag.String(key, val, usage)
	case bool:
		pflag.Bool(key, val, usage)
	case int:
		pflag.Int(key, val, usage)
	case int8:
		pflag.Int8(key, val, usage)
	case int16:
		pflag.Int16(key, val, usage)
	case int32:
		pflag.Int32(key, val, usage)
	case int64:
		pflag.Int64(key, val, usage)
	case uint:
		pflag.Uint(key, val, usage)
	case uint8:
		pflag.Uint8(key, val, usage)
	case uint16:
		pflag.Uint16(key, val, usage)
	case uint32:
		pflag.Uint32(key, val, usage)
	case uint64:
		pflag.Uint64(key, val, usage)
	case float32:
		pflag.Float32(key, val, usage)
	case float64:
		pflag.Float64(key, val, usage)
	case []byte:
		pflag.BytesBase64(key, val, usage)

	//...
	case []time.Duration:
		pflag.DurationSlice(key, val, usage)
	case []net.IP:
		pflag.IPSlice(key, val, usage)
	case []string:
		pflag.StringSlice(key, val, usage)
	case []bool:
		pflag.BoolSlice(key, val, usage)
	case []int:
		pflag.IntSlice(key, val, usage)
	case []int32:
		pflag.Int32Slice(key, val, usage)
	case []int64:
		pflag.Int64Slice(key, val, usage)
	case []uint:
		pflag.UintSlice(key, val, usage)
	case []float32:
		pflag.Float32Slice(key, val, usage)
	case []float64:
		pflag.Float64Slice(key, val, usage)
	default:
		panic(fmt.Sprintf("default value type %T", val))
	}
}

func SetEnvPrefix(val string) {
	viper.SetEnvPrefix(strings.ToUpper(val))
}

func Parse() {
	// 自动 绑定 env 环境
	viper.AutomaticEnv()

	// 解析 flag
	viper.BindPFlags(pflag.CommandLine)
}

func Get(key string) interface{} {
	return viper.Get(key)
}

func GetBool(key string) bool {
	return viper.GetBool(key)
}
func GetString(key string) string {
	return viper.GetString(key)
}
func GetDuration(key string) time.Duration {
	return viper.GetDuration(key)
}
func GetFloat64(key string) float64 {
	return viper.GetFloat64(key)
}

func GetInt(key string) int {
	return viper.GetInt(key)
}

func GetInt32(key string) int32 {
	return viper.GetInt32(key)
}

func GetInt64(key string) int64 {
	return viper.GetInt64(key)
}
func GetUint(key string) uint {
	return viper.GetUint(key)
}

func GetUint32(key string) uint32 {
	return viper.GetUint32(key)
}

func GetUint64(key string) uint64 {
	return viper.GetUint64(key)
}

func GetIntSlice(key string) []int {
	return viper.GetIntSlice(key)
}
func GetStringSlice(key string) []string {
	return viper.GetStringSlice(key)
}

func GetTime(key string) time.Time {
	return viper.GetTime(key)
}

func GetConfig() *viper.Viper {
	return viper.GetViper()
}
