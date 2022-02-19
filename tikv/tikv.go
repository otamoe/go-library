package libtikv

import (
	libviper "github.com/otamoe/go-library/viper"
	"go.uber.org/fx"
)

// func init() {
// 	log.SetLevel(log.WarnLevel)
// 	log.SetOutput(zap.NewStdLog(iLog.Logger("tikv").Desugar()).Writer())
// }

func New() fx.Option {
	return fx.Options(
		fx.Provide(Rawkv),
		fx.Provide(Txnkv),
	)
}

func init() {
	libviper.SetDefault("tikv.rawkv.pdAddress", []string{"127.0.0.1:2379"}, "tikv rawkv pb address")
	libviper.SetDefault("tikv.rawkv.tls.ca", "", "tikv rawkv tls ca")
	libviper.SetDefault("tikv.rawkv.tls.cert", "", "tikv rawkv tls cert")
	libviper.SetDefault("tikv.rawkv.tls.key", "", "tikv rawkv tls key")
	libviper.SetDefault("tikv.rawkv.tls.cn", []string{}, "tikv rawkv tls cn")
	libviper.SetDefault("tikv.txnkv.pdAddress", []string{"127.0.0.1:2379"}, "tikv txnkv pb address")
	libviper.SetDefault("tikv.txnkv.tls.ca", "", "tikv txnkv tls ca")
	libviper.SetDefault("tikv.txnkv.tls.cert", "", "tikv txnkv tls cert")
	libviper.SetDefault("tikv.txnkv.tls.key", "", "tikv txnkv tls key")
	libviper.SetDefault("tikv.txnkv.tls.cn", []string{}, "tikv txnkv tls cn")
}
