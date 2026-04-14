package router

import "context"

type contextKey string

const paramsKey contextKey = "params"

func withParams(ctx context.Context, params map[string]string) context.Context {
	return context.WithValue(ctx, paramsKey, params)
}

// Params retrieves path parameters from the request context.
func Params(ctx context.Context) map[string]string {
	params, ok := ctx.Value(paramsKey).(map[string]string)
	if !ok {
		return make(map[string]string)
	}
	return params
}
