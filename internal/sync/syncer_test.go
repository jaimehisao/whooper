package sync

import (
	"errors"
	"sync"
	"testing"
	"time"
)

var errAssert = errors.New("assert error")

func TestNew(t *testing.T) {
	syncer := New(nil, nil, nil)
	if syncer == nil {
		t.Fatal("New() returned nil")
	}
	if syncer.onProgress != nil {
		t.Error("onProgress should be nil")
	}
}

func TestSyncerProgress(t *testing.T) {
	var progressCalls []struct {
		entity string
		count  int
	}
	var mu sync.Mutex

	progress := func(entity string, count int) {
		mu.Lock()
		progressCalls = append(progressCalls, struct {
			entity string
			count  int
		}{entity, count})
		mu.Unlock()
	}

	s := &Syncer{onProgress: progress}
	s.progress("cycles", 5)
	s.progress("recoveries", 3)

	mu.Lock()
	if len(progressCalls) != 2 {
		t.Errorf("progress called %d times, want 2", len(progressCalls))
	}
	if progressCalls[0].entity != "cycles" || progressCalls[0].count != 5 {
		t.Errorf("first call = (%s, %d), want (cycles, 5)", progressCalls[0].entity, progressCalls[0].count)
	}
	if progressCalls[1].entity != "recoveries" || progressCalls[1].count != 3 {
		t.Errorf("second call = (%s, %d), want (recoveries, 3)", progressCalls[1].entity, progressCalls[1].count)
	}
	mu.Unlock()
}

func TestSyncerProgressNil(t *testing.T) {
	s := &Syncer{onProgress: nil}
	s.progress("test", 1)
}

func TestGetSyncStartWithOverlapNil(t *testing.T) {
	result := GetSyncStartWithOverlap(nil)
	if result != "" {
		t.Errorf("GetSyncStartWithOverlap(nil) = %q, want empty", result)
	}
}

type mockDBForSync struct {
	syncState string
	syncErr   error
}

func (m *mockDBForSync) GetSyncState(entity string) (string, error) {
	return m.syncState, m.syncErr
}

func TestGetSyncStartWithOverlapEmpty(t *testing.T) {
	db := &mockDBForSync{syncState: ""}
	result := GetSyncStartWithOverlap(db)
	if result != "" {
		t.Errorf("GetSyncStartWithOverlap('') = %q, want empty", result)
	}
}

func TestGetSyncStartWithOverlapError(t *testing.T) {
	db := &mockDBForSync{syncErr: errAssert}
	result := GetSyncStartWithOverlap(db)
	if result != "" {
		t.Errorf("GetSyncStartWithOverlap(err) = %q, want empty", result)
	}
}

func TestGetSyncStartWithOverlapUsesOldestEntity(t *testing.T) {
	db := &mockDBForSyncMap{states: map[string]string{
		"cycles":     "2024-01-20T00:00:00Z",
		"recoveries": "2024-01-10T00:00:00Z",
		"sleeps":     "2024-01-18T00:00:00Z",
		"workouts":   "2024-01-19T00:00:00Z",
	}}
	result := GetSyncStartWithOverlap(db)
	expected := "2024-01-09T00:00:00Z"
	if result != expected {
		t.Errorf("GetSyncStartWithOverlap() = %q, want %q", result, expected)
	}
}

type mockDBForSyncMap struct {
	states map[string]string
}

func (m *mockDBForSyncMap) GetSyncState(entity string) (string, error) {
	return m.states[entity], nil
}

func TestGetSyncStartWithOverlapInvalidTime(t *testing.T) {
	db := &mockDBForSync{syncState: "not-a-time"}
	result := GetSyncStartWithOverlap(db)
	if result != "" {
		t.Errorf("GetSyncStartWithOverlap(invalid) = %q, want empty", result)
	}
}

func TestFormatSyncTime(t *testing.T) {
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	result := FormatSyncTime(ts)
	expected := "2024-01-15T10:30:00Z"
	if result != expected {
		t.Errorf("FormatSyncTime() = %q, want %q", result, expected)
	}
}

func TestSyncEntities(t *testing.T) {
	if len(syncEntities) != 4 {
		t.Errorf("len(syncEntities) = %d, want 4", len(syncEntities))
	}
}

func TestSyncEntitiesList(t *testing.T) {
	expected := []string{"cycles", "recoveries", "sleeps", "workouts"}
	for i, e := range syncEntities {
		if e != expected[i] {
			t.Errorf("syncEntities[%d] = %s, want %s", i, e, expected[i])
		}
	}
}

func TestGetSyncStartWithOverlap_DifferentDateFormats(t *testing.T) {
	tests := []struct {
		name      string
		syncState string
		wantEmpty bool
	}{
		{"RFC3339", "2024-01-15T00:00:00Z", false},
		{"ISO date", "2024-01-15", true},
		{"unix timestamp", "1705276800", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &mockDBForSync{syncState: tt.syncState}
			result := GetSyncStartWithOverlap(db)
			if tt.wantEmpty && result != "" {
				t.Errorf("GetSyncStartWithOverlap(%q) = %q, want empty", tt.syncState, result)
			}
			if !tt.wantEmpty && result == "" {
				t.Errorf("GetSyncStartWithOverlap(%q) = empty, want date", tt.syncState)
			}
		})
	}
}
