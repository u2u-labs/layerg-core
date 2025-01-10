package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

// GET function with dynamic response unmarshaling and handling for nil params
func GET(ctx context.Context, endpoint string, token string, params map[string]string, responseStruct interface{}) error {
	baseURL, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Append query parameters if provided
	if params != nil {
		query := baseURL.Query()
		for key, value := range params {
			query.Add(key, value)
		}
		baseURL.RawQuery = query.Encode()
	}
	fmt.Print(baseURL.String())
	// Create GET request
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create GET request: %w", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("GET request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := ioutil.ReadAll(resp.Body)
		return errors.New("GET request failed with status " + resp.Status + ": " + string(respBody))
	}

	// Read and unmarshal response into provided struct
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read GET response body: %w", err)
	}
	if err := json.Unmarshal(respBody, &responseStruct); err != nil {
		return fmt.Errorf("failed to unmarshal GET response: %w", err)
	}

	return nil
}

// POST function with dynamic response unmarshaling and handling for nil body
func POST(ctx context.Context, endpoint string, token string, signature string, body interface{}, responseStruct interface{}) error {
	// If body is nil, set to empty JSON
	var jsonBody []byte
	var err error
	if body != nil {
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal POST body: %w", err)
		}
	} else {
		jsonBody = []byte("{}") // Representing an empty JSON object
	}

	// Create POST request
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create POST request: %w", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if signature != "" {
		req.Header.Set("Signature", signature)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("POST request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := ioutil.ReadAll(resp.Body)
		return errors.New("POST request failed with status " + resp.Status + ": " + string(respBody))
	}

	if responseStruct == nil {
		return nil
	}

	// Read and unmarshal response into provided struct
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read POST response body: %w", err)
	}
	if err := json.Unmarshal(respBody, &responseStruct); err != nil {
		return fmt.Errorf("failed to unmarshal POST response: %w", err)
	}

	return nil
}
