package playback

import "context"

// Both capacity waits consume the same startup deadline and can be cancelled.
func acquireSessionSlot(ctx context.Context, slots chan struct{}) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case slots <- struct{}{}:
		if err := ctx.Err(); err != nil {
			<-slots
			return err
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
