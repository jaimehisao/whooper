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
	R *resty.Client
}

func NewClient(tokenSource oauth2.TokenSource) *Client {
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
			return code == http.StatusTooManyRequests ||
				code == http.StatusInternalServerError ||
				code == http.StatusServiceUnavailable ||
				code == http.StatusBadGateway
		}).
		SetRetryAfter(func(_ *resty.Client, resp *resty.Response) (time.Duration, error) {
			if resp.StatusCode() == http.StatusTooManyRequests {
				if retryAfter := resp.Header().Get("Retry-After"); retryAfter != "" {
					if secs, err := strconv.Atoi(retryAfter); err == nil {
						if secs > maxRetryAfterSecs {
							secs = maxRetryAfterSecs
						}
						return time.Duration(secs) * time.Second, nil
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
	return &Client{R: r}
}
