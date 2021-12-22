package libhttp

import (
	"crypto/tls"
	"strings"

	"github.com/spf13/viper"
)

func ViperAddr(v *viper.Viper) (out OutOption) {
	addr := v.GetString("http.addr")
	if addr != "" {
		out = WithAddr(addr)()
	} else {
		out.Option = func(server *Server) error {
			return nil
		}
	}
	return
}

func ViperCertificates(v *viper.Viper) (out OutOption) {
	certificates := v.GetStringSlice("http.certificates")
	fcertificates := []string{}
	for _, c := range certificates {
		c = strings.TrimSpace(c)
		if len(c) != 0 {
			fcertificates = append(fcertificates, c)
		}
	}
	if len(fcertificates) == 0 {
		out.Option = func(server *Server) error {
			return nil
		}
	}

	out.Option = func(server *Server) (err error) {
		tlsCertificates := []tls.Certificate{}
		for _, c := range fcertificates {
			var certificate tls.Certificate
			if certificate, err = tls.X509KeyPair([]byte(c), []byte(c)); err != nil {
				return
			}
			tlsCertificates = append(tlsCertificates, certificate)
		}
		if server.TLSConfig == nil {
			server.TLSConfig = &tls.Config{
				MinVersion:   tls.VersionTLS12,
				Certificates: tlsCertificates,
			}
		} else {
			server.TLSConfig.Certificates = tlsCertificates
		}
		return nil
	}

	return
}
