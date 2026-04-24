package api

import "git.infra.hisao.org/hisao/whooper/internal/models"

func (c *Client) GetRecoveries(start, end string) ([]models.Recovery, error) {
	return FetchAll[models.Recovery](c, "/v1/recovery", recoveryParams(start, end))
}

func (c *Client) ForEachRecovery(start, end string, fn func([]models.Recovery) error) error {
	return FetchPaginated(c, "/v1/recovery", recoveryParams(start, end), fn)
}

func recoveryParams(start, end string) map[string]string {
	params := map[string]string{"limit": "25"}
	if start != "" {
		params["start"] = start
	}
	if end != "" {
		params["end"] = end
	}
	return params
}
