package oauth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const maxProviderResponseSize = 1 << 20 // 1 MB

func getProviderJSON(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	accessToken string,
	accept string,
	dst any,
) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		endpoint,
		nil,
	)
	if err != nil {
		return fmt.Errorf("create provider api request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", accept)

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("provider api request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf(
			"provider api returned status %d",
			res.StatusCode,
		)
	}

	if err := decodeProviderJSON(res.Body, dst); err != nil {
		return err
	}

	return nil
}

func decodeProviderJSON(
	r io.Reader,
	dst any,
) error {
	limited := io.LimitReader(
		r,
		maxProviderResponseSize+1,
	)

	data, err := io.ReadAll(limited)
	if err != nil {
		return err
	}

	if len(data) > maxProviderResponseSize {
		return fmt.Errorf("provider response too large")
	}

	decoder := json.NewDecoder(bytes.NewReader(data))

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("provider response contains multiple JSON values")
		}

		return fmt.Errorf("provider response has trailing data: %w", err)
	}

	return nil
}
