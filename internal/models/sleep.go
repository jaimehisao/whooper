package models

type Sleep struct {
	ID         int         `json:"id"`
	UserID     int         `json:"user_id"`
	CreatedAt  string      `json:"created_at"`
	UpdatedAt  string      `json:"updated_at"`
	Start      string      `json:"start"`
	End        string      `json:"end"`
	Nap        bool        `json:"nap"`
	ScoreState string      `json:"score_state"`
	Score      *SleepScore `json:"score"`
}

type SleepScore struct {
	StageSummary          SleepStageSummary `json:"stage_summary"`
	SleepNeeded           SleepNeeded       `json:"sleep_needed"`
	RespiratoryRate       float64           `json:"respiratory_rate"`
	SleepPerformancePct   float64           `json:"sleep_performance_percentage"`
	SleepConsistencyPct   float64           `json:"sleep_consistency_percentage"`
	SleepEfficiencyPct    float64           `json:"sleep_efficiency_percentage"`
}

type SleepStageSummary struct {
	TotalInBedTimeMilli         int `json:"total_in_bed_time_milli"`
	TotalAwakeTimeMilli         int `json:"total_awake_time_milli"`
	TotalNoDataTimeMilli        int `json:"total_no_data_time_milli"`
	TotalLightSleepTimeMilli    int `json:"total_light_sleep_time_milli"`
	TotalSlowWaveSleepTimeMilli int `json:"total_slow_wave_sleep_time_milli"`
	TotalRemSleepTimeMilli      int `json:"total_rem_sleep_time_milli"`
	SleepCycleCount             int `json:"sleep_cycle_count"`
	DisturbanceCount            int `json:"disturbance_count"`
}

type SleepNeeded struct {
	BaselineMilli             int `json:"baseline_sleep_needed_milli"`
	NeedFromSleepDebtMilli    int `json:"need_from_sleep_debt_milli"`
	NeedFromRecentStrainMilli int `json:"need_from_recent_strain_milli"`
	NeedFromRecentNapMilli    int `json:"need_from_recent_nap_milli"`
}
