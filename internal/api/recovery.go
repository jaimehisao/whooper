package api

import "git.infra.hisao.org/hisao/whooper/internal/models"

func (c *Client) GetRecoveries(start, end string) ([]models.Recovery, error) {
	params := map[string]string{"limit": "25"}
	if start != "" {
		params["start"] = start
	}
	if end != "" {
		params["end"] = end
	}
	return FetchAll[models.Recovery](c, "/v1/recovery", params)
}
