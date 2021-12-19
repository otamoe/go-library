package libhttp

import (
	"net/http"
	"time"
)

type (
	Server struct {
		*http.Server
		options *Options
	}
)

func (server *Server) Start() (err error) {
	t := time.NewTimer(server.options.StartTimeout)
	defer t.Stop()
	errc := make(chan error, 1)
	go func() {
		var e error
		if server.TLSConfig == nil {
			e = server.ListenAndServe()
		} else {
			e = server.ListenAndServeTLS("", "")
		}
		if e == http.ErrServerClosed {
			e = nil
		}
		errc <- e
	}()
	select {
	case <-t.C:
	case err = <-errc:
	}
	return
}
