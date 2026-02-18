package api

import "git.infra.hisao.org/hisao/whooper/internal/models"

func (c *Client) GetSleeps(start, end string) ([]models.Sleep, error) {
	params := map[string]string{"limit": "25"}
	if start != "" {
		params["start"] = start
	}
	if end != "" {
		params["end"] = end
	}
	return FetchAll[models.Sleep](c, "/v1/activity/sleep", params)
}
