package gateway

import (
	"context"
	"crypto/rand"
	"math/big"
	"time"
)

const minReplyDelay = 2500 * time.Millisecond
const maxReplyDelay = 5 * time.Second

// The model prepares the reply in Concierge. Only a new outgoing gateway send
// waits; idempotent lookups and delivery receipts do not repeat the delay.
func randomReplyDelay() (time.Duration, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(maxReplyDelay-minReplyDelay)+1))
	if err != nil {
		return 0, err
	}
	return minReplyDelay + time.Duration(n.Int64()), nil
}

func waitReplyDelay(ctx context.Context) error {
	duration, err := randomReplyDelay()
	if err != nil {
		return err
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
