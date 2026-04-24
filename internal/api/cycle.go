package api

import "git.infra.hisao.org/hisao/whooper/internal/models"

func (c *Client) GetCycles(start, end string) ([]models.Cycle, error) {
	return FetchAll[models.Cycle](c, "/v1/cycle", cycleParams(start, end))
}

func (c *Client) ForEachCycle(start, end string, fn func([]models.Cycle) error) error {
	return FetchPaginated(c, "/v1/cycle", cycleParams(start, end), fn)
}

func cycleParams(start, end string) map[string]string {
	params := map[string]string{"limit": "25"}
	if start != "" {
		params["start"] = start
	}
	if end != "" {
		params["end"] = end
	}
	return params
}
