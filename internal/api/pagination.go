package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type paginatedResponse[T any] struct {
	Records   []T    `json:"records"`
	NextToken string `json:"next_token,omitempty"`
}

// FetchAll retrieves all records from a paginated Whoop API endpoint.
func FetchAll[T any](c *Client, endpoint string, params map[string]string) ([]T, error) {
	var all []T
	nextToken := ""

	for {
		req := c.R.R().SetQueryParams(params)
		if nextToken != "" {
			req.SetQueryParam("nextToken", nextToken)
		}

		resp, err := req.Get(endpoint)
		if err != nil {
			return nil, fmt.Errorf("fetch %s: %w", endpoint, err)
		}
		if resp.StatusCode() != http.StatusOK {
			return nil, fmt.Errorf("fetch %s: status %d: %s", endpoint, resp.StatusCode(), resp.String())
		}

		var page paginatedResponse[T]
		if err := json.Unmarshal(resp.Body(), &page); err != nil {
			return nil, fmt.Errorf("decode %s: %w", endpoint, err)
		}

		all = append(all, page.Records...)
		if page.NextToken == "" {
			break
		}
		nextToken = page.NextToken
	}
	return all, nil
}
