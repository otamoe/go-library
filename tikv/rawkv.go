package libtikv

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	libutils "github.com/otamoe/go-library/utils"
	"github.com/spf13/viper"
	"github.com/tikv/client-go/v2/config"
	"github.com/tikv/client-go/v2/rawkv"
	pd "github.com/tikv/pd/client"
	"go.uber.org/fx"
)

type (
	InRawkvOptions struct {
		fx.In
		Options []pd.ClientOption `group:"tikvRawkvOptions"`
	}

	OutRawkvOption struct {
		fx.Out
		Option pd.ClientOption `group:"tikvRawkvOptions"`
	}
)

func RawtvOption(option pd.ClientOption) func() (out OutRawkvOption) {
	return func() (out OutRawkvOption) {
		out.Option = option
		return
	}
}

func Rawkv(ctx context.Context, lc fx.Lifecycle, v *viper.Viper, inRawkvOptions InRawkvOptions) (client *rawkv.Client, err error) {

	security := config.Security{
		ClusterVerifyCN: v.GetStringSlice("tikv.rawkv.tls.cn"),
	}

	rand := string(libutils.RandByte(8, libutils.RandAlphaNumber))

	// ca 文件
	security.ClusterSSLCA = path.Join(os.TempDir(), fmt.Sprintf("tikv-rawkv-tls-ca-%s", rand))
	if err = ioutil.WriteFile(security.ClusterSSLCA, []byte(v.GetString("tikv.rawkv.tls.ca")), 0755); err != nil {
		return
	}
	// cert 文件
	security.ClusterSSLCert = path.Join(os.TempDir(), fmt.Sprintf("tikv-rawkv-tls-cert-%s", rand))
	if err = ioutil.WriteFile(security.ClusterSSLCert, []byte(v.GetString("tikv.rawkv.tls.cert")), 0755); err != nil {
		return
	}

	// key 文件
	security.ClusterSSLKey = path.Join(os.TempDir(), fmt.Sprintf("tikv-rawkv-tls-key-%s", rand))
	if err = ioutil.WriteFile(security.ClusterSSLKey, []byte(v.GetString("tikv.rawkv.tls.key")), 0755); err != nil {
		return
	}

	if client, err = rawkv.NewClient(ctx, v.GetStringSlice("tikv.rawkv.pdAddress"), security, inRawkvOptions.Options...); err != nil {
		return
	}
	lc.Append(fx.Hook{
		OnStop: func(c context.Context) error {
			client.Close()
			os.Remove(security.ClusterSSLCA)
			os.Remove(security.ClusterSSLCert)
			os.Remove(security.ClusterSSLKey)
			return nil
		},
	})

	return
}
