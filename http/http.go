package libhttp

import (
	"net/http"
	"net/url"

	libviper "github.com/otamoe/go-library/viper"
	"go.uber.org/fx"
)

func New() fx.Option {
	return fx.Options(
		fx.Provide(libviper.WithSetDefault("http.addr", ":8080", "HTTP addr")),
		fx.Provide(libviper.WithSetDefault("http.certificates", []string{}, "HTTP certificates")),

		fx.Provide(ViperAddr),
		fx.Provide(ViperCertificates),

		fx.Provide(NewServer),
	)
}

func Host(r *http.Request, defaultValue string) (host string) {
	if host = r.Header.Get("X-Forwarded-Host"); host != "" {

	} else if host = r.Host; host != "" {

	} else if host = r.Header.Get("X-Host"); host != "" {

	} else if host = r.URL.Host; host != "" {

	} else {
		host = defaultValue
	}

	if u, err := url.Parse("http://" + host); err == nil {
		return u.Hostname()
	}

	return defaultValue
}
