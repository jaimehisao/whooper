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
// For large datasets, use FetchPaginated to avoid high memory pressure.
func FetchAll[T any](c *Client, endpoint string, params map[string]string) ([]T, error) {
	var all []T
	err := FetchPaginated(c, endpoint, params, func(records []T) error {
		all = append(all, records...)
		return nil
	})
	return all, err
}

// FetchPaginated retrieves records from a paginated Whoop API endpoint and calls the
// provided function for each page of results. This is more memory-efficient for large datasets.
func FetchPaginated[T any](c *Client, endpoint string, params map[string]string, fn func([]T) error) error {
	nextToken := ""

	for page := 0; page < maxPages; page++ {
		req := c.R.R().SetQueryParams(params)
		if nextToken != "" {
			req.SetQueryParam("nextToken", nextToken)
		}

		resp, err := req.Get(endpoint)
		if err != nil {
			return fmt.Errorf("fetch %s: %w", endpoint, err)
		}
		if resp.StatusCode() != http.StatusOK {
			return fmt.Errorf("fetch %s: status %d: %s", endpoint, resp.StatusCode(), resp.String())
		}

		var pr paginatedResponse[T]
		if err := json.Unmarshal(resp.Body(), &pr); err != nil {
			return fmt.Errorf("decode %s: %w", endpoint, err)
		}

		if len(pr.Records) > 0 {
			if err := fn(pr.Records); err != nil {
				return err
			}
		}

		if pr.NextToken == "" || pr.NextToken == nextToken {
			break
		}
		nextToken = pr.NextToken
	}
	return nil
}
