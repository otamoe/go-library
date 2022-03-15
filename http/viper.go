package libhttp

import (
	"github.com/spf13/viper"
)

func ViperListenAddress() (out OutOption) {
	listenAddress := viper.GetString("http.listenAddress")
	if listenAddress != "" {
		out = WithListenAddress(listenAddress)()
	} else {
		out.Option = func(server *Server) error {
			return nil
		}
	}
	return
}
