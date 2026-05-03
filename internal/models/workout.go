package models

type Workout struct {
	ID         string        `json:"id"`
	UserID     int           `json:"user_id"`
	CreatedAt  string        `json:"created_at"`
	UpdatedAt  string        `json:"updated_at"`
	Start      string        `json:"start"`
	End        string        `json:"end"`
	SportID    int           `json:"sport_id"`
	ScoreState string        `json:"score_state"`
	Score      *WorkoutScore `json:"score"`
}

type WorkoutScore struct {
	Strain              float64       `json:"strain"`
	AverageHeartRate    int           `json:"average_heart_rate"`
	MaxHeartRate        int           `json:"max_heart_rate"`
	Kilojoule           float64       `json:"kilojoule"`
	PercentRecorded     float64       `json:"percent_recorded"`
	DistanceMeter       float64       `json:"distance_meter"`
	AltitudeGainMeter   float64       `json:"altitude_gain_meter"`
	AltitudeChangeMeter float64       `json:"altitude_change_meter"`
	ZoneDuration        *ZoneDuration `json:"zone_duration"`
}

type ZoneDuration struct {
	ZoneZeroMilli  int `json:"zone_zero_milli"`
	ZoneOneMilli   int `json:"zone_one_milli"`
	ZoneTwoMilli   int `json:"zone_two_milli"`
	ZoneThreeMilli int `json:"zone_three_milli"`
	ZoneFourMilli  int `json:"zone_four_milli"`
	ZoneFiveMilli  int `json:"zone_five_milli"`
}

// SportName maps sport IDs to human-readable names.
var SportName = map[int]string{
	-1: "Activity", 0: "Running", 1: "Cycling", 16: "Baseball",
	17: "Basketball", 18: "Rowing", 19: "Fencing", 20: "Field Hockey",
	21: "Football", 22: "Golf", 24: "Ice Hockey", 25: "Lacrosse",
	27: "Rugby", 28: "Sailing", 29: "Skiing", 30: "Soccer",
	31: "Softball", 32: "Squash", 33: "Swimming", 34: "Tennis",
	35: "Track & Field", 36: "Volleyball", 37: "Water Polo",
	38: "Wrestling", 39: "Boxing", 42: "Dance", 43: "Pilates",
	44: "Yoga", 45: "Weightlifting", 47: "Cross Country Skiing",
	48: "Functional Fitness", 49: "Duathlon", 51: "Gymnastics",
	52: "HIIT", 53: "Martial Arts", 55: "Meditation",
	56: "Other", 57: "Paddle Tennis", 58: "Climbing",
	59: "Surfing", 60: "Wakeboarding", 61: "Snowboarding",
	62: "Triathlon", 63: "Stretching", 64: "Spin",
	65: "Motocross", 66: "Caddying", 70: "Hiking",
	71: "Obstacle Course Racing", 73: "Diving",
	74: "Operations / Tactical", 75: "Athletics",
	76: "Wheelchair Pushing", 77: "Kayaking", 82: "Ski Ergometer",
	83: "Elliptical", 84: "Stairmaster", 85: "Pickleball",
	86: "Paddleboarding", 87: "Fishing", 88: "Hunting",
	89: "Skateboarding", 90: "Assault Bike",
	91: "Kickboxing", 92: "Horseback Riding",
	126: "Floor Hockey", 127: "Ice Bath", 128: "BMX",
	230: "Badminton", 231: "Racquetball", 232: "Disc Golf",
}
