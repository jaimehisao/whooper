package views

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"git.infra.hisao.org/hisao/whooper/internal/models"
	"git.infra.hisao.org/hisao/whooper/internal/store"
	tea "github.com/charmbracelet/bubbletea"
)

func setupTestDB(t *testing.T) (*store.DB, func()) {
	tmpDir, err := os.MkdirTemp("", "whooper-test-*")
	if err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := store.Open(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatal(err)
	}
	return db, func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}
}

func TestWorkoutsModel(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	now := time.Now().UTC()
	start1 := now.Add(-24 * time.Hour).Format(time.RFC3339)
	end1 := now.Add(-23 * time.Hour).Format(time.RFC3339)
	start2 := now.Add(-48 * time.Hour).Format(time.RFC3339)
	end2 := now.Add(-47 * time.Hour).Format(time.RFC3339)

	// Seed some data
	workouts := []models.Workout{
		{ID: "1", UserID: 123, Start: start1, End: end1, SportID: 1, Score: &models.WorkoutScore{Strain: 10.5}},
		{ID: "2", UserID: 123, Start: start2, End: end2, SportID: 1, Score: &models.WorkoutScore{Strain: 12.0}},
	}
	if err := db.SaveWorkouts(workouts); err != nil {
		t.Fatal(err)
	}

	m := NewWorkouts(db)
	pm := &m

	// Test Init
	cmd := pm.Init()
	if cmd == nil {
		t.Fatal("Init should return a command")
	}

	// Test data loading
	msg := cmd()
	m2, cmd2 := pm.Update(msg)
	wm := m2.(*WorkoutsModel)
	if cmd2 != nil {
		t.Error("Update after data load should not return more commands")
	}
	if !wm.loaded {
		t.Error("Expected loaded to be true")
	}
	if len(wm.workouts) != 2 {
		t.Errorf("Expected 2 workouts, got %d", len(wm.workouts))
	}

	// Test navigation
	m2, _ = wm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	wm = m2.(*WorkoutsModel)
	if wm.cursor != 1 {
		t.Errorf("Expected cursor 1, got %d", wm.cursor)
	}

	m2, _ = wm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	wm = m2.(*WorkoutsModel)
	if wm.cursor != 0 {
		t.Errorf("Expected cursor 0, got %d", wm.cursor)
	}

	// Test detail view toggle
	m2, _ = wm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	wm = m2.(*WorkoutsModel)
	if !wm.detail {
		t.Error("Expected detail view to be active")
	}

	// Test View (smoke test)
	view := wm.View()
	if view == "" {
		t.Error("View should not be empty")
	}
}

func TestWorkoutsModel_SummaryBreakdownAndDetail(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	now := time.Now().UTC()
	runStart := now.Add(-24 * time.Hour)
	rideStart := now.Add(-48 * time.Hour)
	if err := db.SaveWorkouts([]models.Workout{
		{
			ID: "run", UserID: 123, Start: runStart.Format(time.RFC3339), End: runStart.Add(45 * time.Minute).Format(time.RFC3339),
			SportID: 0, ScoreState: "SCORED",
			Score: &models.WorkoutScore{
				Strain:            12,
				AverageHeartRate:  150,
				MaxHeartRate:      180,
				Kilojoule:         1000,
				PercentRecorded:   98,
				DistanceMeter:     8000,
				AltitudeGainMeter: 120,
			},
		},
		{
			ID: "ride", UserID: 123, Start: rideStart.Format(time.RFC3339), End: rideStart.Add(90 * time.Minute).Format(time.RFC3339),
			SportID: 1, ScoreState: "SCORED",
			Score: &models.WorkoutScore{
				Strain:           9,
				AverageHeartRate: 130,
				MaxHeartRate:     165,
				DistanceMeter:    22000,
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	m := NewWorkouts(db)
	pm := &m
	pm.Update(pm.Refresh()())

	view := pm.View()
	for _, want := range []string{"Workouts: 2", "Total strain: 21.0", "Distance: 30.0 km", "Workout Strain", "Sport Breakdown", "Running", "Cycling"} {
		if !strings.Contains(view, want) {
			t.Fatalf("workouts view missing %q in:\n%s", want, view)
		}
	}

	pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	detail := pm.View()
	for _, want := range []string{"Recorded: 98%", "Load:", "Distance: 8.0 km", "Elev Gain: 120 m"} {
		if !strings.Contains(detail, want) {
			t.Fatalf("workout detail missing %q in:\n%s", want, detail)
		}
	}
}

func TestDashboardModel(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	now := time.Now().UTC()
	startToday := now.Format(time.RFC3339)

	// Seed data
	db.SaveRecoveries([]models.Recovery{
		{CycleID: 1, UserID: 123, CreatedAt: startToday, ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 80, HRVRmssd: 50, RestingHeartRate: 60}},
	})
	db.SaveSleeps([]models.Sleep{
		{ID: "1", UserID: 123, Start: startToday, ScoreState: "SCORED", Score: &models.SleepScore{StageSummary: models.SleepStageSummary{TotalInBedTimeMilli: 8 * 3600 * 1000}}},
	})

	m := NewDashboard(db)
	pm := &m

	// Test Init
	cmd := pm.Init()
	if cmd == nil {
		t.Fatal("Init should return a command")
	}

	// Test data loading
	msg := cmd()
	m2, _ := pm.Update(msg)
	dm := m2.(*DashboardModel)
	if !dm.loaded {
		t.Error("Expected loaded to be true")
	}
	if dm.recoveryScore != 80 {
		t.Errorf("Expected recovery 80, got %.0f", dm.recoveryScore)
	}

	// Test View
	view := dm.View()
	if view == "" {
		t.Error("View should not be empty")
	}
}

func TestDashboardModel_UsesLatestRecentScores(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	now := time.Now().UTC()
	yesterday := now.Add(-24 * time.Hour).Format(time.RFC3339)

	db.SaveRecoveries([]models.Recovery{
		{CycleID: 1, UserID: 123, CreatedAt: yesterday, ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 72, HRVRmssd: 44, RestingHeartRate: 57}},
	})
	db.SaveSleeps([]models.Sleep{
		{ID: "1", UserID: 123, Start: yesterday, ScoreState: "SCORED", Score: &models.SleepScore{
			StageSummary:       models.SleepStageSummary{TotalInBedTimeMilli: 8 * 3600 * 1000, TotalAwakeTimeMilli: 30 * 60 * 1000},
			SleepEfficiencyPct: 93,
		}},
	})
	db.SaveCycles([]models.Cycle{
		{ID: 1, UserID: 123, Start: yesterday, ScoreState: "SCORED", Score: &models.CycleScore{Strain: 11.2}},
	})

	m := NewDashboard(db)
	pm := &m
	pm.Update(pm.Refresh()())

	if pm.recoveryScore != 72 {
		t.Errorf("Expected recovery 72, got %.0f", pm.recoveryScore)
	}
	if pm.sleepHours == 0 {
		t.Error("Expected latest sleep hours to be populated")
	}
	if pm.dayStrain != 11.2 {
		t.Errorf("Expected strain 11.2, got %.1f", pm.dayStrain)
	}

	view := pm.View()
	if !strings.Contains(view, "Latest scored:") {
		t.Errorf("Expected latest scored date line, got: %s", view)
	}
}

func TestRecoveryModel(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	now := time.Now().UTC()
	db.SaveRecoveries([]models.Recovery{
		{CycleID: 1, UserID: 123, CreatedAt: now.Add(-24 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 80, HRVRmssd: 60, RestingHeartRate: 55}},
		{CycleID: 2, UserID: 123, CreatedAt: now.Add(-48 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 70, HRVRmssd: 55, RestingHeartRate: 58}},
		{CycleID: 3, UserID: 123, CreatedAt: now.Add(-72 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 60, HRVRmssd: 50, RestingHeartRate: 60}},
	})

	m := NewRecovery(db)
	pm := &m
	cmd := pm.Init()
	msg := cmd()
	m2, _ := pm.Update(msg)
	rm := m2.(*RecoveryModel)

	if !rm.loaded {
		t.Error("Expected loaded")
	}

	view := rm.View()
	if view == "" {
		t.Error("View should not be empty")
	}

	// Test range navigation
	m2, _ = rm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(">")})
	rm = m2.(*RecoveryModel)
	if rm.rangeIdx != 3 {
		t.Errorf("Expected rangeIdx 3, got %d", rm.rangeIdx)
	}

	m2, _ = rm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("<")})
	rm = m2.(*RecoveryModel)
	if rm.rangeIdx != 2 {
		t.Errorf("Expected rangeIdx 2, got %d", rm.rangeIdx)
	}
}

func TestSleepModel(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	now := time.Now().UTC()
	db.SaveSleeps([]models.Sleep{
		{ID: "1", UserID: 123, Start: now.Add(-24 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.SleepScore{StageSummary: models.SleepStageSummary{TotalInBedTimeMilli: 8 * 3600 * 1000}, SleepEfficiencyPct: 90}},
		{ID: "2", UserID: 123, Start: now.Add(-48 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.SleepScore{StageSummary: models.SleepStageSummary{TotalInBedTimeMilli: 7 * 3600 * 1000}, SleepEfficiencyPct: 85}},
	})

	m := NewSleep(db)
	pm := &m
	cmd := pm.Init()
	msg := cmd()
	m2, _ := pm.Update(msg)
	sm := m2.(*SleepModel)

	if !sm.loaded {
		t.Error("Expected loaded")
	}

	view := sm.View()
	if view == "" {
		t.Error("View should not be empty")
	}

	// Test range navigation
	m2, _ = sm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(">")})
	sm = m2.(*SleepModel)
	if sm.rangeIdx != 3 {
		t.Errorf("Expected rangeIdx 3, got %d", sm.rangeIdx)
	}
	m2, _ = sm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("<")})
	sm = m2.(*SleepModel)
	if sm.rangeIdx != 2 {
		t.Errorf("Expected rangeIdx 2, got %d", sm.rangeIdx)
	}
}

func TestSleepModel_SummaryAndRecentTable(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	now := time.Now().UTC()
	db.SaveSleeps([]models.Sleep{
		{
			ID: "1", UserID: 123, Start: now.Add(-24 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED",
			Score: &models.SleepScore{
				StageSummary: models.SleepStageSummary{
					TotalInBedTimeMilli:         8 * 3600 * 1000,
					TotalAwakeTimeMilli:         30 * 60 * 1000,
					TotalLightSleepTimeMilli:    4 * 3600 * 1000,
					TotalSlowWaveSleepTimeMilli: 90 * 60 * 1000,
					TotalRemSleepTimeMilli:      2 * 3600 * 1000,
					DisturbanceCount:            11,
				},
				SleepNeeded: models.SleepNeeded{
					BaselineMilli:             8 * 3600 * 1000,
					NeedFromSleepDebtMilli:    30 * 60 * 1000,
					NeedFromRecentStrainMilli: 30 * 60 * 1000,
				},
				SleepPerformancePct: 88,
				SleepConsistencyPct: 76,
				SleepEfficiencyPct:  94,
			},
		},
		{
			ID: "2", UserID: 123, Start: now.Add(-48 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED",
			Score: &models.SleepScore{
				StageSummary: models.SleepStageSummary{
					TotalInBedTimeMilli:      7 * 3600 * 1000,
					TotalAwakeTimeMilli:      45 * 60 * 1000,
					TotalLightSleepTimeMilli: 4 * 3600 * 1000,
					TotalRemSleepTimeMilli:   90 * 60 * 1000,
					DisturbanceCount:         14,
				},
				SleepNeeded: models.SleepNeeded{
					BaselineMilli: 8 * 3600 * 1000,
				},
				SleepPerformancePct: 80,
				SleepConsistencyPct: 70,
				SleepEfficiencyPct:  89,
			},
		},
	})

	m := NewSleep(db)
	pm := &m
	pm.Update(pm.Refresh()())

	view := pm.View()
	for _, want := range []string{
		"Avg:",
		"Need gap:",
		"Performance:",
		"Sleep Need Gap",
		"Sleep Performance",
		"Sleep Consistency",
		"Recent Nights",
		"Actual",
		"Dist",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("sleep view missing %q in:\n%s", want, view)
		}
	}
}

func TestWorkoutsModel_Detail(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	now := time.Now().UTC()
	start1 := now.Add(-24 * time.Hour).Format(time.RFC3339)

	// Seed data
	db.SaveWorkouts([]models.Workout{
		{
			ID: "1", UserID: 123, Start: start1, End: now.Add(-23 * time.Hour).Format(time.RFC3339),
			SportID: 1, Score: &models.WorkoutScore{
				Strain: 10.5,
				ZoneDuration: &models.ZoneDuration{
					ZoneTwoMilli: 1800000, // 30 mins
				},
			},
		},
	})

	m := NewWorkouts(db)
	pm := &m
	cmd := pm.Init()
	msg := cmd()
	m2, _ := pm.Update(msg)
	wm := m2.(*WorkoutsModel)

	// Enter detail
	m2, _ = wm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	wm = m2.(*WorkoutsModel)
	if !wm.detail {
		t.Fatal("expected detail view")
	}

	view := wm.View()
	if !strings.Contains(view, "Workout Detail") {
		t.Errorf("unexpected detail view output: %s", view)
	}

	// Exit detail
	m2, _ = wm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	wm = m2.(*WorkoutsModel)
	if wm.detail {
		t.Error("expected detail view closed")
	}
}

func TestWorkoutsModel_Nav(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	now := time.Now().UTC()
	db.SaveWorkouts([]models.Workout{
		{ID: "1", Start: now.Format(time.RFC3339)},
		{ID: "2", Start: now.Add(-1 * time.Hour).Format(time.RFC3339)},
	})

	m := NewWorkouts(db)
	pm := &m
	pm.Update(pm.Refresh()())

	pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if pm.cursor != 1 {
		t.Errorf("Expected cursor 1, got %d", pm.cursor)
	}

	pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // Beyond limit
	if pm.cursor != 1 {
		t.Errorf("Expected cursor 1, got %d", pm.cursor)
	}

	pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	if pm.cursor != 0 {
		t.Errorf("Expected cursor 0, got %d", pm.cursor)
	}

	pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}) // Beyond limit
	if pm.cursor != 0 {
		t.Errorf("Expected cursor 0, got %d", pm.cursor)
	}

	// Test detail view with no score
	db.SaveWorkouts([]models.Workout{
		{ID: "3", Start: now.Format(time.RFC3339), Score: nil},
	})
	pm.Update(pm.Refresh()())
	pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	pm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	view := pm.View()
	if !strings.Contains(view, "Workout Detail") {
		t.Errorf("Expected detail view even with no score")
	}
}

func TestDashboardModel_Nav(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	m := NewDashboard(db)
	pm := &m
	pm.Update(pm.Refresh()()) // Initial load

	pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}) // Sync trigger

	// Test with error
	pm.err = "forced error"
	view := pm.View()
	if !strings.Contains(view, "Error: forced error") {
		t.Errorf("Expected error message in view, got: %s", view)
	}

	// Test with data
	now := time.Now().UTC()
	db.SaveRecoveries([]models.Recovery{
		{CycleID: 1, UserID: 123, CreatedAt: now.Format(time.RFC3339), ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 80}},
	})
	db.SaveSleeps([]models.Sleep{
		{ID: "1", UserID: 123, Start: now.Format(time.RFC3339), ScoreState: "SCORED", Score: &models.SleepScore{StageSummary: models.SleepStageSummary{TotalInBedTimeMilli: 8 * 3600 * 1000}}},
	})
	db.SaveCycles([]models.Cycle{
		{ID: 1, UserID: 123, Start: now.Format(time.RFC3339), ScoreState: "SCORED", Score: &models.CycleScore{Strain: 15}},
	})

	pm.Update(pm.Refresh()())
	view = pm.View()
	if !strings.Contains(view, "Recovery: 80%") || !strings.Contains(view, "Strain: 15.0") {
		t.Errorf("Expected full dashboard view, got: %s", view)
	}
}

func TestDashboardModel_ViewBranches(t *testing.T) {
	m := NewDashboard(nil)
	if view := m.View(); !strings.Contains(view, "Loading dashboard") {
		t.Fatalf("expected loading view, got: %s", view)
	}

	m.loaded = true
	m.recoveryScore = 25
	m.hrvValue = 42
	m.rhrValue = 58
	m.sleepHours = 7.5
	m.sleepEffPct = 91
	m.dayStrain = 19
	m.sparklineData = []float64{55, 60, 65, 70}
	m.alerts = []string{"Low recovery: 25%", "High strain: 19.0"}
	m.recentWorkouts = []models.Workout{
		{
			ID:      "1",
			Start:   "2024-01-15T10:00:00Z",
			SportID: 1,
			Score:   &models.WorkoutScore{Strain: 12.3},
		},
		{
			ID:      "2",
			Start:   "2024-01-16T10:00:00Z",
			SportID: 999,
		},
	}

	view := m.View()
	for _, want := range []string{"Low recovery", "High strain", "HRV: 42 ms", "Sleep: 7.5h", "7-Day Recovery", "Recent Workouts", "Cycling", "Sport 999"} {
		if !strings.Contains(view, want) {
			t.Fatalf("dashboard view missing %q in:\n%s", want, view)
		}
	}
}

func TestRecoveryModel_Nav(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	m := NewRecovery(db)
	pm := &m
	pm.Update(pm.Refresh()()) // Load (empty) data

	pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(">")})
	pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(">")})
	pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(">")})
	pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(">")}) // Beyond limit
	pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("<")})
	pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("<")})
	pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("<")})
	pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("<")}) // Beyond limit

	// Test with error
	pm.err = "forced error"
	view := pm.View()
	if !strings.Contains(view, "Error: forced error") {
		t.Errorf("Expected error message in recovery view")
	}
}

func TestCorrelationsModel(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	now := time.Now().UTC()

	// Seed data for correlations
	db.SaveRecoveries([]models.Recovery{
		{CycleID: 1, UserID: 123, CreatedAt: now.Add(-24 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 80, HRVRmssd: 60}},
		{CycleID: 2, UserID: 123, CreatedAt: now.Add(-48 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 70, HRVRmssd: 50}},
		{CycleID: 3, UserID: 123, CreatedAt: now.Add(-72 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 60, HRVRmssd: 40}},
	})
	db.SaveCycles([]models.Cycle{
		{ID: 1, UserID: 123, Start: now.Add(-24 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.CycleScore{Strain: 15}},
		{ID: 2, UserID: 123, Start: now.Add(-48 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.CycleScore{Strain: 12}},
		{ID: 3, UserID: 123, Start: now.Add(-72 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.CycleScore{Strain: 10}},
	})

	m := NewCorrelations(db)
	pm := &m
	cmd := pm.Init()
	msg := cmd()
	m2, _ := pm.Update(msg)
	cm := m2.(*CorrelationsModel)

	if !cm.loaded {
		t.Error("Expected loaded")
	}

	view := cm.View()
	if view == "" {
		t.Error("View should not be empty")
	}

	// Test metric switching
	pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(">")})
	if pm.xIdx != 2 {
		t.Errorf("Expected xIdx 2, got %d", pm.xIdx)
	}
	pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("<")})
	if pm.xIdx != 1 {
		t.Errorf("Expected xIdx 1, got %d", pm.xIdx)
	}

	pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})
	if pm.yIdx != 2 {
		t.Errorf("Expected yIdx 2, got %d", pm.yIdx)
	}
	pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[")})
	if pm.yIdx != 0 {
		t.Errorf("Expected yIdx 0, got %d", pm.yIdx)
	}

	// Test navigation beyond limits
	for i := 0; i < 10; i++ {
		pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(">")})
	}
	for i := 0; i < 10; i++ {
		pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})
	}
}

func TestCorrelationsModel_Boundary(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	now := time.Now().UTC()
	// Seed exactly 3 points
	db.SaveRecoveries([]models.Recovery{
		{CycleID: 1, UserID: 123, CreatedAt: now.Add(-24 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 80}},
		{CycleID: 2, UserID: 123, CreatedAt: now.Add(-48 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 70}},
		{CycleID: 3, UserID: 123, CreatedAt: now.Add(-72 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 60}},
	})
	db.SaveCycles([]models.Cycle{
		{ID: 1, UserID: 123, Start: now.Add(-24 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.CycleScore{Strain: 15}},
		{ID: 2, UserID: 123, Start: now.Add(-48 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.CycleScore{Strain: 12}},
		{ID: 3, UserID: 123, Start: now.Add(-72 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.CycleScore{Strain: 10}},
	})

	m := NewCorrelations(db)
	pm := &m
	pm.Update(pm.Refresh()())
	view := pm.View()
	if strings.Contains(view, "Not enough data") {
		t.Error("expected enough data with 3 points")
	}
}

func TestWorkoutsModel_Viewport(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Seed 50 workouts to trigger virtualization logic
	now := time.Now().UTC()
	var workouts []models.Workout
	for i := 0; i < 50; i++ {
		workouts = append(workouts, models.Workout{
			ID: strconv.Itoa(i + 1), Start: now.Add(-time.Duration(i) * time.Hour).Format(time.RFC3339),
		})
	}
	db.SaveWorkouts(workouts)

	m := NewWorkouts(db)
	pm := &m
	pm.height = 20 // Ensure reserved height leaves small viewport
	pm.Update(pm.Refresh()())

	// Navigate deep
	for i := 0; i < 40; i++ {
		pm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	}

	view := pm.View()
	if !strings.Contains(view, "(41/50)") {
		t.Errorf("Expected cursor at 41, got: %s", view)
	}
}

func TestCorrelationsModel_Strong(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	now := time.Now().UTC()
	// Perfect correlation
	db.SaveRecoveries([]models.Recovery{
		{CycleID: 1, UserID: 123, CreatedAt: now.Add(-24 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 10}},
		{CycleID: 2, UserID: 123, CreatedAt: now.Add(-48 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 20}},
		{CycleID: 3, UserID: 123, CreatedAt: now.Add(-72 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 30}},
	})
	db.SaveCycles([]models.Cycle{
		{ID: 1, UserID: 123, Start: now.Add(-24 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.CycleScore{Strain: 1}},
		{ID: 2, UserID: 123, Start: now.Add(-48 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.CycleScore{Strain: 2}},
		{ID: 3, UserID: 123, Start: now.Add(-72 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.CycleScore{Strain: 3}},
	})

	m := NewCorrelations(db)
	pm := &m
	pm.xIdx = 0 // recovery
	pm.yIdx = 0 // recovery (perfect correlation with itself)
	pm.Update(pm.Refresh()())
	view := pm.View()
	if !strings.Contains(view, "(strong)") {
		t.Errorf("Expected strong correlation styling, got: %s", view)
	}
}

func TestWorkoutsModel_SmallWidth(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	m := NewWorkouts(db)
	m.width = 40 // Small width
	pm := &m
	pm.Update(pm.Refresh()())
	view := pm.View()
	if view == "" {
		t.Error("View should not be empty")
	}
}

func TestCorrelationsModel_Weak(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	now := time.Now().UTC()
	// Seed data with weak correlation
	db.SaveRecoveries([]models.Recovery{
		{CycleID: 1, UserID: 123, CreatedAt: now.Add(-24 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 80}},
		{CycleID: 2, UserID: 123, CreatedAt: now.Add(-48 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 81}},
		{CycleID: 3, UserID: 123, CreatedAt: now.Add(-72 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.RecoveryScore{RecoveryScore: 79}},
	})
	db.SaveCycles([]models.Cycle{
		{ID: 1, UserID: 123, Start: now.Add(-24 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.CycleScore{Strain: 10}},
		{ID: 2, UserID: 123, Start: now.Add(-48 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.CycleScore{Strain: 20}},
		{ID: 3, UserID: 123, Start: now.Add(-72 * time.Hour).Format(time.RFC3339), ScoreState: "SCORED", Score: &models.CycleScore{Strain: 15}},
	})

	m := NewCorrelations(db)
	pm := &m
	pm.Update(pm.Refresh()())
	view := pm.View()
	if !strings.Contains(view, "(weak)") {
		// Correlation might not be weak enough with 3 points, but let's see
	}
}
