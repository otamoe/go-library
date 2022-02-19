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

	"github.com/tikv/client-go/v2/txnkv"
	pd "github.com/tikv/pd/client"
	"go.uber.org/fx"
)

type (
	InTxnkvOptions struct {
		fx.In
		Options []pd.ClientOption `group:"tikvTxnkvOptions"`
	}

	OutTxnkvOption struct {
		fx.Out
		Option pd.ClientOption `group:"tikvTxnkvOptions"`
	}
)

func TxntvOption(option pd.ClientOption) func() (out OutTxnkvOption) {
	return func() (out OutTxnkvOption) {
		out.Option = option
		return
	}
}

func Txnkv(ctx context.Context, lc fx.Lifecycle, inTxnkvOptions InTxnkvOptions) (client *txnkv.Client, err error) {
	security := config.Security{
		ClusterVerifyCN: viper.GetStringSlice("tikv.txnkv.tls.cn"),
	}

	rand := string(libutils.RandByte(8, libutils.RandAlphaNumber))

	// ca 文件
	security.ClusterSSLCA = path.Join(os.TempDir(), fmt.Sprintf("tikv-txnkv-tls-ca-%s", rand))
	if err = ioutil.WriteFile(security.ClusterSSLCA, []byte(viper.GetString("tikv.txnkv.tls.ca")), 0755); err != nil {
		return
	}
	// cert 文件
	security.ClusterSSLCert = path.Join(os.TempDir(), fmt.Sprintf("tikv-txnkv-tls-cert-%s", rand))
	if err = ioutil.WriteFile(security.ClusterSSLCert, []byte(viper.GetString("tikv.txnkv.tls.cert")), 0755); err != nil {
		return
	}

	// key 文件
	security.ClusterSSLKey = path.Join(os.TempDir(), fmt.Sprintf("tikv-txnkv-tls-key-%s", rand))
	if err = ioutil.WriteFile(security.ClusterSSLKey, []byte(viper.GetString("tikv.txnkv.tls.key")), 0755); err != nil {
		return
	}

	cfg := config.GetGlobalConfig()
	cfg.Security = security

	config.StoreGlobalConfig(cfg)

	if client, err = txnkv.NewClient(viper.GetStringSlice("tikv.txnkv.pdAddress")); err != nil {
		return
	}

	lc.Append(fx.Hook{
		OnStop: func(c context.Context) error {
			os.Remove(security.ClusterSSLCA)
			os.Remove(security.ClusterSSLCert)
			os.Remove(security.ClusterSSLKey)
			client.Close()
			return nil
		},
	})

	return
}
