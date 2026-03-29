package initializers

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// NewCancellableContext returns a context.Context that is cancelled when
// SIGINT or SIGTERM is received. It forces shutdown after 10 seconds.
func NewCancellableContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		slog.Info("Received signal: " + sig.String())
		cancel()

		select {
		case <-time.After(10 * time.Second):
			slog.Warn("Timeout reached. Forcing shutdown...")
			os.Exit(1)
		}
	}()

	return ctx, cancel
}
