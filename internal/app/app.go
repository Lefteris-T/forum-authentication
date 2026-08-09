package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"forum/internal/config"
)

const shutdownTimeout = 5 * time.Second

func Run(ctx context.Context, cfg config.Config) error {
	server := &http.Server{
		Addr: cfg.Address,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "Forum is running")
		}),
	}

	serverErrors := make(chan error, 1)

	go func() {
		serverErrors <- normalizeServerError(server.ListenAndServe())
	}()

	select {
	case err := <-serverErrors:
		return err

	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			shutdownTimeout,
		)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}

		err := <-serverErrors
		return err
	}
}

func normalizeServerError(err error) error {
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return err
}
