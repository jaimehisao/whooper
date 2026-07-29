package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"golang.org/x/oauth2"
)

const baseURL = "https://api.prod.whoop.com/developer"

const maxRetryAfterSecs = 60

type Client struct {
	R           *resty.Client
	tokenSource oauth2.TokenSource
}

func NewClient(tokenSource oauth2.TokenSource) *Client {
	c := &Client{tokenSource: tokenSource}
	r := resty.New().
		SetBaseURL(baseURL).
		SetTimeout(30 * time.Second).
		SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second).
		SetRetryMaxWaitTime(30 * time.Second).
		AddRetryCondition(func(resp *resty.Response, err error) bool {
			if err != nil {
				return true
			}
			code := resp.StatusCode()
			if code == http.StatusUnauthorized {
				// Retry once after Token() refresh (Attempt is 1-based before retry).
				return resp.Request.Attempt <= 1
			}
			return code == http.StatusTooManyRequests ||
				code == http.StatusInternalServerError ||
				code == http.StatusServiceUnavailable ||
				code == http.StatusBadGateway
		}).
		SetRetryAfter(func(_ *resty.Client, resp *resty.Response) (time.Duration, error) {
			if resp.StatusCode() == http.StatusUnauthorized {
				// Force token refresh before retry.
				if c.tokenSource != nil {
					_, _ = c.tokenSource.Token()
				}
				return 0, nil
			}
			if resp.StatusCode() == http.StatusTooManyRequests {
				if retryAfter := resp.Header().Get("Retry-After"); retryAfter != "" {
					if secs, err := strconv.Atoi(retryAfter); err == nil {
						if secs > maxRetryAfterSecs {
							secs = maxRetryAfterSecs
						}
						return time.Duration(secs) * time.Second, nil
					}
					if when, err := http.ParseTime(retryAfter); err == nil {
						d := time.Until(when)
						if d < 0 {
							d = 0
						}
						if d > time.Duration(maxRetryAfterSecs)*time.Second {
							d = time.Duration(maxRetryAfterSecs) * time.Second
						}
						return d, nil
					}
				}
			}
			return 0, nil
		}).
		OnBeforeRequest(func(_ *resty.Client, req *resty.Request) error {
			tok, err := tokenSource.Token()
			if err != nil {
				return err
			}
			req.SetAuthToken(tok.AccessToken)
			return nil
		})
	c.R = r
	return c
}
