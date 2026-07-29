package models

type Recovery struct {
	CycleID    int            `json:"cycle_id"`
	SleepID    string         `json:"sleep_id"`
	UserID     int            `json:"user_id"`
	CreatedAt  string         `json:"created_at"`
	UpdatedAt  string         `json:"updated_at"`
	ScoreState string         `json:"score_state"`
	Score      *RecoveryScore `json:"score"`
}

type RecoveryScore struct {
	UserCalibrating  bool     `json:"user_calibrating"`
	RecoveryScore    float64  `json:"recovery_score"`
	RestingHeartRate float64  `json:"resting_heart_rate"`
	HRVRmssd         float64  `json:"hrv_rmssd_milli"`
	SpO2Percentage   *float64 `json:"spo2_percentage"`
	SkinTempCelsius  *float64 `json:"skin_temp_celsius"`
}

func (s *RecoveryScore) SpO2OrZero() float64 {
	if s == nil || s.SpO2Percentage == nil {
		return 0
	}
	return *s.SpO2Percentage
}

func (s *RecoveryScore) SkinTempOrZero() float64 {
	if s == nil || s.SkinTempCelsius == nil {
		return 0
	}
	return *s.SkinTempCelsius
}
