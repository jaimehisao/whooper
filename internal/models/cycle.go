package models

type Cycle struct {
	ID             int         `json:"id"`
	UserID         int         `json:"user_id"`
	CreatedAt      string      `json:"created_at"`
	UpdatedAt      string      `json:"updated_at"`
	Start          string      `json:"start"`
	End            string      `json:"end"`
	TimezoneOffset string      `json:"timezone_offset"`
	Days           int         `json:"days"`
	ScoreState     string      `json:"score_state"`
	Score          *CycleScore `json:"score"`
}

type CycleScore struct {
	Strain           float64 `json:"strain"`
	Kilojoule        float64 `json:"kilojoule"`
	AverageHeartRate int     `json:"average_heart_rate"`
	MaxHeartRate     int     `json:"max_heart_rate"`
}
