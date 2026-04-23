package store

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"git.infra.hisao.org/hisao/whooper/internal/models"
)

func benchDB(b *testing.B) *DB {
	b.Helper()
	dbPath := filepath.Join(b.TempDir(), "bench.db")
	db, err := Open(dbPath)
	if err != nil {
		b.Fatalf("Open benchmark DB: %v", err)
	}
	b.Cleanup(func() { _ = db.Close() })
	return db
}

func makeRecoveries(n int) []models.Recovery {
	out := make([]models.Recovery, 0, n)
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range n {
		ts := base.AddDate(0, 0, i).Format(time.RFC3339)
		out = append(out, models.Recovery{
			CycleID:    i + 1,
			SleepID:    i + 1,
			UserID:     1,
			CreatedAt:  ts,
			UpdatedAt:  ts,
			ScoreState: "SCORED",
			Score: &models.RecoveryScore{
				RecoveryScore:    60 + float64(i%40),
				HRVRmssd:         30 + float64(i%50),
				RestingHeartRate: 45 + float64(i%15),
			},
		})
	}
	return out
}

func makeCycles(n int) []models.Cycle {
	out := make([]models.Cycle, 0, n)
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range n {
		start := base.AddDate(0, 0, i)
		out = append(out, models.Cycle{
			ID:         i + 1,
			UserID:     1,
			CreatedAt:  start.Format(time.RFC3339),
			UpdatedAt:  start.Format(time.RFC3339),
			Start:      start.Format(time.RFC3339),
			End:        start.Add(90 * time.Minute).Format(time.RFC3339),
			ScoreState: "SCORED",
			Score: &models.CycleScore{
				Strain: float64(8 + (i % 12)),
			},
		})
	}
	return out
}

func BenchmarkSaveRecoveries1k(b *testing.B) {
	db := benchDB(b)
	recoveries := makeRecoveries(1000)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := db.SaveRecoveries(recoveries); err != nil {
			b.Fatalf("SaveRecoveries error: %v", err)
		}
	}
}

func BenchmarkListRecoveries30d(b *testing.B) {
	db := benchDB(b)
	recoveries := makeRecoveries(5000)
	if err := db.SaveRecoveries(recoveries); err != nil {
		b.Fatalf("seed SaveRecoveries error: %v", err)
	}

	from := "2024-01-01T00:00:00Z"
	to := "2024-01-31T23:59:59Z"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rows, err := db.ListRecoveries(from, to)
		if err != nil {
			b.Fatalf("ListRecoveries error: %v", err)
		}
		if len(rows) == 0 {
			b.Fatal("ListRecoveries returned no rows")
		}
	}
}

func BenchmarkGetRecoveryTrend(b *testing.B) {
	db := benchDB(b)
	recoveries := makeRecoveries(5000)
	if err := db.SaveRecoveries(recoveries); err != nil {
		b.Fatalf("seed SaveRecoveries error: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, err := db.GetRecoveryTrend("", "")
		if err != nil {
			b.Fatalf("GetRecoveryTrend error: %v", err)
		}
		if len(rows) == 0 {
			b.Fatal("GetRecoveryTrend returned no rows")
		}
	}
}

func BenchmarkGetCorrelationDataRecoveryHRV(b *testing.B) {
	db := benchDB(b)
	recoveries := makeRecoveries(5000)
	if err := db.SaveRecoveries(recoveries); err != nil {
		b.Fatalf("seed SaveRecoveries error: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, err := db.GetCorrelationData("recovery", "hrv")
		if err != nil {
			b.Fatalf("GetCorrelationData error: %v", err)
		}
		if len(rows) == 0 {
			b.Fatal("GetCorrelationData returned no rows")
		}
	}
}

func BenchmarkGetCorrelationDataRecoveryStrain(b *testing.B) {
	db := benchDB(b)
	recoveries := makeRecoveries(5000)
	cycles := makeCycles(5000)
	if err := db.SaveRecoveries(recoveries); err != nil {
		b.Fatalf("seed SaveRecoveries error: %v", err)
	}
	if err := db.SaveCycles(cycles); err != nil {
		b.Fatalf("seed SaveCycles error: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, err := db.GetCorrelationData("recovery", "strain")
		if err != nil {
			b.Fatalf("GetCorrelationData cross-table error: %v", err)
		}
		if len(rows) == 0 {
			b.Fatal("GetCorrelationData cross-table returned no rows")
		}
	}
}

func BenchmarkSetAndGetSyncState(b *testing.B) {
	db := benchDB(b)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		stamp := fmt.Sprintf("2024-01-01T00:00:%02dZ", i%60)
		if err := db.SetSyncState("cycles", stamp); err != nil {
			b.Fatalf("SetSyncState error: %v", err)
		}
		if _, err := db.GetSyncState("cycles"); err != nil {
			b.Fatalf("GetSyncState error: %v", err)
		}
	}
}
