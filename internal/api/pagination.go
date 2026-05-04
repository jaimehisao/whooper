package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const maxPages = 10000

type paginatedResponse[T any] struct {
	Records   []T    `json:"records"`
	NextToken string `json:"next_token,omitempty"`
}

type StatusError struct {
	Endpoint string
	Status   int
	Body     string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("fetch %s: status %d: %s", e.Endpoint, e.Status, e.Body)
}

func IsUnauthorized(err error) bool {
	var statusErr *StatusError
	return errors.As(err, &statusErr) && statusErr.Status == http.StatusUnauthorized
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
			return &StatusError{Endpoint: endpoint, Status: resp.StatusCode(), Body: resp.String()}
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
