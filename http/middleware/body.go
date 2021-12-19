package middleware

import (
	"errors"
	"io"
	"net/http"
	"time"
)

type (
	Body struct {
		Limit    int64
		LowSpeed int
	}

	BodyReader struct {
		io.ReadCloser
		Remaining int64
		LowSpeed  int

		err error
		res http.ResponseWriter

		wasAborted bool
		sawEOF     bool
	}
)

var (
	ErrBodyTooLarge = errors.New(http.StatusText(http.StatusRequestEntityTooLarge))
	ErrBodyTimeout  = errors.New(http.StatusText(http.StatusRequestTimeout))
)

func (mbr *BodyReader) tooLarge() (n int, err error) {
	if !mbr.wasAborted {
		mbr.wasAborted = true
		mbr.err = ErrBodyTooLarge
		mbr.res.Header().Set("Connection", "close")
	}
	err = mbr.err
	return
}

func (mbr *BodyReader) lowSpeed() (n int, err error) {
	if !mbr.wasAborted {
		mbr.wasAborted = true
		mbr.err = ErrBodyTimeout
		mbr.res.Header().Set("connection", "close")
	}

	err = mbr.err
	return
}

func (mbr *BodyReader) Read(p []byte) (n int, err error) {
	toRead := mbr.Remaining
	if mbr.Remaining == 0 {
		if mbr.sawEOF {
			return mbr.tooLarge()
		}
		// The underlying io.Reader may not return (0, io.EOF)
		// at EOF if the requested size is 0, so read 1 byte
		// instead. The io.Reader docs are a bit ambiguous
		// about the return value of Read when 0 bytes are
		// requested, and {bytes,strings}.Reader gets it wrong
		// too (it returns (0, nil) even at EOF).
		toRead = 1
	}
	if int64(len(p)) > toRead {
		p = p[:toRead]
	}

	done := make(chan struct{})

	second := (len(p) / mbr.LowSpeed) + 5

	go func() {
		n, err = mbr.ReadCloser.Read(p)
		close(done)
	}()
	select {
	case <-done:
		// 读取完毕

		if mbr.Remaining == 0 {
			// If we had zero bytes to read Remaining (but hadn't seen EOF)
			// and we get a byte here, that means we went over our limit.
			if n > 0 {
				return mbr.tooLarge()
			}
			return 0, err
		}
		mbr.Remaining -= int64(n)
		if mbr.Remaining < 0 {
			mbr.Remaining = 0
		}
		return
	case <-time.After(time.Second * time.Duration(second)):
		// 读取超时
		return mbr.lowSpeed()
	}
}

func (body *Body) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil && r.Body != http.NoBody {
			r.Body = &BodyReader{
				res:        w,
				ReadCloser: r.Body,
				Remaining:  body.Limit,
				LowSpeed:   body.LowSpeed,
			}
		}
		next.ServeHTTP(w, r)
	})
}
