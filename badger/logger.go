package libbadger

import (
	"github.com/dgraph-io/badger/v3"
	goLog "github.com/ipfs/go-log/v2"
)

func Logger() (out OutOption) {
	out.Option = func(o badger.Options) (badger.Options, error) {
		return o.WithLogger(goLog.Logger("badger")), nil
	}
	return
}
