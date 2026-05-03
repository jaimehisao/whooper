package api

import "git.infra.hisao.org/hisao/whooper/internal/models"

func (c *Client) GetSleeps(start, end string) ([]models.Sleep, error) {
	return FetchAll[models.Sleep](c, "/v2/activity/sleep", sleepParams(start, end))
}

func (c *Client) ForEachSleep(start, end string, fn func([]models.Sleep) error) error {
	return FetchPaginated(c, "/v2/activity/sleep", sleepParams(start, end), fn)
}

func sleepParams(start, end string) map[string]string {
	params := map[string]string{"limit": "25"}
	if start != "" {
		params["start"] = start
	}
	if end != "" {
		params["end"] = end
	}
	return params
}
