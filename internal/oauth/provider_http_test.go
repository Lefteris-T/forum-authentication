package oauth

import (
	"strings"
	"testing"
)

func TestDecodeProviderJSONRejectsTrailingData(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "trailing invalid data",
			body: `{"access_token":"token"} invalid-data`,
		},
		{
			name: "second json value",
			body: `{"access_token":"token"} {"unexpected":true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var payload struct {
				AccessToken string `json:"access_token"`
			}

			err := decodeProviderJSON(strings.NewReader(tt.body), &payload)
			if err == nil {
				t.Fatal("decodeProviderJSON() error = nil, want error")
			}
		})
	}
}

func TestDecodeProviderJSONAllowsTrailingWhitespace(t *testing.T) {
	var payload struct {
		AccessToken string `json:"access_token"`
	}

	err := decodeProviderJSON(
		strings.NewReader("{\"access_token\":\"token\"}\n\t "),
		&payload,
	)
	if err != nil {
		t.Fatalf("decodeProviderJSON() error: %v", err)
	}

	if payload.AccessToken != "token" {
		t.Fatalf(
			"AccessToken = %q, want %q",
			payload.AccessToken,
			"token",
		)
	}
}
