package app

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"forum/internal/config"
)

func TestRunReturnsStartupError(t *testing.T) {
	cfg := config.Config{
		Address: ":99999",
	}

	err := Run(context.Background(), cfg)

	if err == nil {
		t.Fatal("Run() error = nil, want startup error")
	}
}

func TestRunShutsDownOnContextCancellation(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error: %v", err)
	}
	defer listener.Close()

	cfg := config.Config{
		Address: listener.Addr().String(),
	}

	listener.Close()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)

	go func() {
		done <- Run(ctx, cfg)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() returned error after shutdown: %v", err)
		}

	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not shut down after context cancellation")
	}
}

func TestServerClosedIsNotApplicationError(t *testing.T) {
	err := normalizeServerError(http.ErrServerClosed)

	if err != nil {
		t.Fatalf("normalizeServerError() = %v, want nil", err)
	}
}

func TestOtherServerErrorsAreReturned(t *testing.T) {
	expected := errors.New("server failed")

	err := normalizeServerError(expected)

	if !errors.Is(err, expected) {
		t.Fatalf("normalizeServerError() = %v, want %v", err, expected)
	}
}