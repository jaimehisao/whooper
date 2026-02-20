package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const maxPages = 10000

type paginatedResponse[T any] struct {
	Records   []T    `json:"records"`
	NextToken string `json:"next_token,omitempty"`
}

// FetchAll retrieves all records from a paginated Whoop API endpoint.
func FetchAll[T any](c *Client, endpoint string, params map[string]string) ([]T, error) {
	var all []T
	nextToken := ""

	for page := 0; page < maxPages; page++ {
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

		var pr paginatedResponse[T]
		if err := json.Unmarshal(resp.Body(), &pr); err != nil {
			return nil, fmt.Errorf("decode %s: %w", endpoint, err)
		}

		all = append(all, pr.Records...)
		if pr.NextToken == "" || pr.NextToken == nextToken {
			break
		}
		nextToken = pr.NextToken
	}
	return all, nil
}
