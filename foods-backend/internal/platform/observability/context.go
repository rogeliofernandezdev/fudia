package observability

import "context"

type Metadata struct{ CorrelationID, Method, Path string }
type key struct{}

func With(ctx context.Context, m Metadata) context.Context { return context.WithValue(ctx, key{}, m) }
func From(ctx context.Context) Metadata                    { m, _ := ctx.Value(key{}).(Metadata); return m }
