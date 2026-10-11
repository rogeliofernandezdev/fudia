// Package contracts contains stable public integration primitives of Subscriptions.
package contracts

import "errors"

// ErrNotFound is the public classification sentinel shared with caller adapters.
var ErrNotFound = errors.New("subscription record not found")
