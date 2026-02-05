package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client holds the configuration for the Revos API client
type Client struct {
	APIURL     string
	Token      string
	HTTPClient *http.Client
}

// NewClient creates a new Revos API client
func NewClient(apiURL, token string) *Client {
	return &Client{
		APIURL: apiURL,
		Token:  token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CubeOverlay represents the overlay resource from the API
type CubeOverlay struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	OrganizationID string          `json:"organizationId"`
	Data           json.RawMessage `json:"data"` // Keeping as RawMessage to support dynamic structure
	CreatedBy      string          `json:"createdBy"`
	CreatedAt      string          `json:"createdAt"`
	UpdatedAt      string          `json:"updatedAt"`
}

// OverlayPayload is used for Create and Update
type OverlayPayload struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Data        json.RawMessage `json:"data"`
}

func (c *Client) request(method, path string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	url := fmt.Sprintf("%s%s", c.APIURL, path)
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// GetOverlay retrieves an overlay by ID
func (c *Client) GetOverlay(id string) (*CubeOverlay, error) {
	body, err := c.request("GET", fmt.Sprintf("/cube-overlays/%s", id), nil)
	if err != nil {
		return nil, err
	}

	var overlay CubeOverlay
	// Handle API wrapper { "data": ... } if present, based on CLI code
	// CLI says: 
	// const json = (await response.json()) as { data?: T } | T;
    // if (typeof json === "object" && json !== null && "data" in json) ...
	
	// We'll try to unmarshal into a wrapper first
	var wrapper struct {
		Data *CubeOverlay `json:"data"`
	}
	// Also try unmarshalling directly
	// Or just unmarshal into a generic map to check for "data" key
	
	// Let's stick to a simple heuristic: if it looks like {data: ...}
	if err := json.Unmarshal(body, &wrapper); err == nil && wrapper.Data != nil && wrapper.Data.ID != "" {
		return wrapper.Data, nil
	}
	
	if err := json.Unmarshal(body, &overlay); err != nil {
		return nil, fmt.Errorf("failed to unmarshal overlay: %w", err)
	}
	
	return &overlay, nil
}

// CreateOverlay creates a new overlay
func (c *Client) CreateOverlay(payload OverlayPayload) (*CubeOverlay, error) {
	body, err := c.request("POST", "/cube-overlays", payload)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Data *CubeOverlay `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapper); err == nil && wrapper.Data != nil {
		return wrapper.Data, nil
	}
	
	var overlay CubeOverlay
	if err := json.Unmarshal(body, &overlay); err != nil {
		return nil, fmt.Errorf("failed to unmarshal overlay: %w", err)
	}
	return &overlay, nil
}

// UpdateOverlay updates an existing overlay
func (c *Client) UpdateOverlay(id string, payload OverlayPayload) (*CubeOverlay, error) {
	body, err := c.request("PATCH", fmt.Sprintf("/cube-overlays/%s", id), payload)
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Data *CubeOverlay `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapper); err == nil && wrapper.Data != nil {
		return wrapper.Data, nil
	}

	var overlay CubeOverlay
	if err := json.Unmarshal(body, &overlay); err != nil {
		return nil, fmt.Errorf("failed to unmarshal overlay: %w", err)
	}
	return &overlay, nil
}

// DeleteOverlay deletes an overlay
func (c *Client) DeleteOverlay(id string) error {
	_, err := c.request("DELETE", fmt.Sprintf("/cube-overlays/%s", id), nil)
	return err
}

// ListOverlays retrieves all overlays
func (c *Client) ListOverlays() ([]CubeOverlay, error) {
	body, err := c.request("GET", "/cube-overlays", nil)
	if err != nil {
		return nil, err
	}

	// Try wrapper format first
	var wrapper struct {
		Data []CubeOverlay `json:"data"`
	}
	if err := json.Unmarshal(body, &wrapper); err == nil && wrapper.Data != nil {
		return wrapper.Data, nil
	}

	// Try direct array
	var overlays []CubeOverlay
	if err := json.Unmarshal(body, &overlays); err != nil {
		return nil, fmt.Errorf("failed to unmarshal overlays: %w", err)
	}
	return overlays, nil
}

// GetOverlayByName retrieves an overlay by its name
func (c *Client) GetOverlayByName(name string) (*CubeOverlay, error) {
	overlays, err := c.ListOverlays()
	if err != nil {
		return nil, err
	}

	for _, overlay := range overlays {
		if overlay.Name == name {
			return &overlay, nil
		}
	}
	return nil, fmt.Errorf("overlay with name %q not found", name)
}
