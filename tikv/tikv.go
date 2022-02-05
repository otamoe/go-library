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
		fx.Provide(libviper.WithSetDefault("tikv.rawkv.pdAddress", []string{"127.0.0.1:2379"}, "tikv rawkv pb address")),
		fx.Provide(libviper.WithSetDefault("tikv.rawkv.tls.ca", "", "tikv rawkv tls ca")),
		fx.Provide(libviper.WithSetDefault("tikv.rawkv.tls.cert", "", "tikv rawkv tls cert")),
		fx.Provide(libviper.WithSetDefault("tikv.rawkv.tls.key", "", "tikv rawkv tls key")),
		fx.Provide(libviper.WithSetDefault("tikv.rawkv.tls.cn", []string{}, "tikv rawkv tls cn")),

		fx.Provide(libviper.WithSetDefault("tikv.txnkv.pdAddress", []string{"127.0.0.1:2379"}, "tikv txnkv pb address")),
		fx.Provide(libviper.WithSetDefault("tikv.txnkv.tls.ca", "", "tikv txnkv tls ca")),
		fx.Provide(libviper.WithSetDefault("tikv.txnkv.tls.cert", "", "tikv txnkv tls cert")),
		fx.Provide(libviper.WithSetDefault("tikv.txnkv.tls.key", "", "tikv txnkv tls key")),
		fx.Provide(libviper.WithSetDefault("tikv.txnkv.tls.cn", []string{}, "tikv txnkv tls cn")),
		fx.Provide(Rawkv),
		fx.Provide(Txnkv),
	)
}
