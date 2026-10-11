// Package unitofwork defines transaction boundaries without exposing a database driver.
package unitofwork

import "context"

type Options struct{ ReadOnly, RepeatableRead bool }
type Transaction interface {
	Context() context.Context
	Commit(context.Context) error
	Rollback(context.Context) error
}
type Manager interface {
	Begin(context.Context, Options) (Transaction, error)
}
