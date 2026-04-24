package api

import "git.infra.hisao.org/hisao/whooper/internal/models"

func (c *Client) GetWorkouts(start, end string) ([]models.Workout, error) {
	return FetchAll[models.Workout](c, "/v1/activity/workout", workoutParams(start, end))
}

func (c *Client) ForEachWorkout(start, end string, fn func([]models.Workout) error) error {
	return FetchPaginated(c, "/v1/activity/workout", workoutParams(start, end), fn)
}

func workoutParams(start, end string) map[string]string {
	params := map[string]string{"limit": "25"}
	if start != "" {
		params["start"] = start
	}
	if end != "" {
		params["end"] = end
	}
	return params
}
