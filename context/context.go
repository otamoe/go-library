package libcontext

import "context"

var Context context.Context

func init() {
	Context = context.Background()
}

func SetContext(ctx context.Context) {
	Context = ctx
}
func GetContext() context.Context {
	return Context
}
