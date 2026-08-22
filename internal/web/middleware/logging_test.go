package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestLoggingLogsSafeContext(t *testing.T) {
	var logs bytes.Buffer

	logger := log.New(
		&logs,
		"",
		0,
	)

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.WriteHeader(http.StatusNoContent)
	})

	h := RequestLogging(
		logger,
		next,
	)

	req := httptest.NewRequest(
		http.MethodGet,
		"/posts/42",
		nil,
	)

	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	output := logs.String()

	if !strings.Contains(output, "method=GET") {
		t.Fatal("method was not logged")
	}

	if !strings.Contains(output, "path=/posts/42") {
		t.Fatal("path was not logged")
	}

	if !strings.Contains(output, "status=204") {
		t.Fatal("status was not logged")
	}
}
func TestRequestLoggingDoesNotLogSecrets(t *testing.T) {
	var logs bytes.Buffer

	logger := log.New(
		&logs,
		"",
		0,
	)

	next := http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		w.WriteHeader(http.StatusOK)
	})

	h := RequestLogging(
		logger,
		next,
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/login?token=super-secret",
		strings.NewReader(
			"email=a@example.com&password=my-secret-password",
		),
	)

	req.Header.Set(
		"Authorization",
		"Bearer secret-token",
	)

	req.Header.Set(
		"Cookie",
		"session=secret-session-id",
	)

	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	output := logs.String()

	secrets := []string{
		"super-secret",
		"my-secret-password",
		"secret-token",
		"secret-session-id",
	}

	for _, secret := range secrets {
		if strings.Contains(output, secret) {
			t.Fatalf(
				"secret %q leaked to logs",
				secret,
			)
		}
	}
}
