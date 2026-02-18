package api

import "git.infra.hisao.org/hisao/whooper/internal/models"

func (c *Client) GetCycles(start, end string) ([]models.Cycle, error) {
	params := map[string]string{"limit": "25"}
	if start != "" {
		params["start"] = start
	}
	if end != "" {
		params["end"] = end
	}
	return FetchAll[models.Cycle](c, "/v1/cycle", params)
}
