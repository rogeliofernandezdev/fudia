// Package dispatch delivers stored documents to the worker. InProcess runs the
// send in a goroutine of the API process; the Cloud Tasks adapter replaces it
// in production, and the scheduled sweep recovers anything either one misses.
package dispatch

import (
	"context"
	"log/slog"
	"time"
)

type InProcess struct {
	process func(context.Context, string) error
}

func NewInProcess(process func(context.Context, string) error) InProcess {
	return InProcess{process: process}
}

func (d InProcess) Dispatch(documentID string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := d.process(ctx, documentID); err != nil {
			slog.Error("einvoicing_dispatch_failed", "document_id", documentID, "error", err.Error())
		}
	}()
}
