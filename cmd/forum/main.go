// Command forum loads configuration and runs the HTTP server until shutdown.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"forum/internal/app"
	"forum/internal/config"
)

func main() {
	// Validate configuration before allocating database or server resources.
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(1)
	}

	// Translate process signals into context cancellation so the app package
	// remains independent of operating-system signal handling.
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	// Run blocks until startup fails or the signal-aware context is cancelled.
	if err := app.Run(ctx, cfg); err != nil {
		fmt.Fprintln(os.Stderr, "server error:", err)
		os.Exit(1)
	}
}
