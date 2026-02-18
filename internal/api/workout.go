package api

import "git.infra.hisao.org/hisao/whooper/internal/models"

func (c *Client) GetWorkouts(start, end string) ([]models.Workout, error) {
	params := map[string]string{"limit": "25"}
	if start != "" {
		params["start"] = start
	}
	if end != "" {
		params["end"] = end
	}
	return FetchAll[models.Workout](c, "/v1/activity/workout", params)
}
