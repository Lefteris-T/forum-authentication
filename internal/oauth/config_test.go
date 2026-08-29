package oauth

import (
	"testing"
	"time"
)

func TestDefaultHTTPClientHasTimeout(t *testing.T) {
	client := DefaultHTTPClient()

	if client == nil {
		t.Fatal("DefaultHTTPClient() returned nil")
	}

	if client.Timeout <= 0 {
		t.Fatalf(
			"client.Timeout = %v, want positive timeout",
			client.Timeout,
		)
	}

	if client.Timeout != 10*time.Second {
		t.Fatalf(
			"client.Timeout = %v, want %v",
			client.Timeout,
			10*time.Second,
		)
	}
}
