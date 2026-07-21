package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"git.infra.hisao.org/hisao/whooper/internal/models"
)

func (c *Client) GetProfile() (*models.Profile, error) {
	resp, err := c.R.R().Get("/v2/user/profile/basic")
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, &StatusError{
			Endpoint: "/v2/user/profile/basic",
			Status:   resp.StatusCode(),
			Body:     resp.String(),
		}
	}
	var p models.Profile
	if err := json.Unmarshal(resp.Body(), &p); err != nil {
		return nil, fmt.Errorf("decode profile: %w", err)
	}
	return &p, nil
}
