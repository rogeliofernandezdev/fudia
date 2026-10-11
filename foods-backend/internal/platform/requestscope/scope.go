// Package requestscope carries verified identity and tenant context across adapters.
package requestscope

import "context"

type Scope struct{ UserID, OrganizationID, LocationID, Name string }

// Key is exported only for the legacy adapter and its existing fixtures.
// New adapters should use With and From.
type Key struct{}

func With(ctx context.Context, s Scope) context.Context { return context.WithValue(ctx, Key{}, s) }
func From(ctx context.Context) (Scope, bool)            { s, ok := ctx.Value(Key{}).(Scope); return s, ok }
