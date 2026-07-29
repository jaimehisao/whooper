package analysis

import (
	"fmt"
	"time"

	"git.infra.hisao.org/hisao/whooper/internal/store"
)

type WeeklyReport struct {
	WeekStart        string
	AvgRecovery      float64
	AvgHRV           float64
	AvgRHR           float64
	AvgSleepHours    float64
	AvgSleepEffPct   float64
	TotalStrain      float64
	WorkoutCount     int
	AvgWorkoutStrain float64
}

func GenerateWeeklyReport(db *store.DB, weekStart time.Time) (*WeeklyReport, error) {
	from := weekStart.Format("2006-01-02")
	// Inclusive end of the 7-day window (DateBounds expands to end-of-day).
	to := weekStart.Add(6 * 24 * time.Hour).Format("2006-01-02")
	// Include overnight sleeps that started the evening before the week.
	sleepFrom := weekStart.Add(-12 * time.Hour).Format(time.RFC3339)

	report := &WeeklyReport{WeekStart: from}

	recoveries, err := db.GetRecoveryTrend(from, to)
	if err != nil {
		return nil, fmt.Errorf("recovery trend: %w", err)
	}
	if len(recoveries) > 0 {
		var sumRec, sumHRV, sumRHR float64
		for _, r := range recoveries {
			sumRec += r.RecoveryScore
			sumHRV += r.HRV
			sumRHR += r.RHR
		}
		n := float64(len(recoveries))
		report.AvgRecovery = sumRec / n
		report.AvgHRV = sumHRV / n
		report.AvgRHR = sumRHR / n
	}

	sleeps, err := db.GetSleepTrend(sleepFrom, to)
	if err != nil {
		return nil, fmt.Errorf("sleep trend: %w", err)
	}
	if len(sleeps) > 0 {
		var sumDur, sumEff float64
		for _, s := range sleeps {
			sumDur += float64(s.DurationMilli) / 3600000.0
			sumEff += s.EfficiencyPct
		}
		n := float64(len(sleeps))
		report.AvgSleepHours = sumDur / n
		report.AvgSleepEffPct = sumEff / n
	}

	strains, err := db.GetStrainTrend(from, to)
	if err != nil {
		return nil, fmt.Errorf("strain trend: %w", err)
	}
	for _, s := range strains {
		report.TotalStrain += s.Strain
	}

	workouts, err := db.ListWorkouts(from, to)
	if err != nil {
		return nil, fmt.Errorf("list workouts: %w", err)
	}
	report.WorkoutCount = len(workouts)
	var sumStrain float64
	var scored int
	for _, w := range workouts {
		if w.Score != nil {
			sumStrain += w.Score.Strain
			scored++
		}
	}
	if scored > 0 {
		report.AvgWorkoutStrain = sumStrain / float64(scored)
	}

	return report, nil
}
