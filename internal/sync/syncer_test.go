package sync

import (
	"errors"
	"sync"
	"testing"
	"time"
)

var assertError = errors.New("assert error")

func TestNew(t *testing.T) {
	syncer := New(nil, nil, nil)
	if syncer == nil {
		t.Error("New() returned nil")
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
	db := &mockDBForSync{syncErr: assertError}
	result := GetSyncStartWithOverlap(db)
	if result != "" {
		t.Errorf("GetSyncStartWithOverlap(err) = %q, want empty", result)
	}
}

func TestGetSyncStartWithOverlapValid(t *testing.T) {
	db := &mockDBForSync{syncState: "2024-01-15T00:00:00Z"}
	result := GetSyncStartWithOverlap(db)
	expected := "2024-01-14T00:00:00Z"
	if result != expected {
		t.Errorf("GetSyncStartWithOverlap() = %q, want %q", result, expected)
	}
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

func TestSyncAll_NilClientDB(t *testing.T) {
	s := &Syncer{}
	err := s.SyncAll()
	if err == nil {
		t.Error("expected error with nil client")
	}
}

func TestSyncFrom_NilClientDB(t *testing.T) {
	s := &Syncer{}
	err := s.SyncFrom("")
	if err == nil {
		t.Error("expected error with nil client")
	}
}

func TestSyncFrom_Full(t *testing.T) {
	s := &Syncer{}
	err := s.SyncFrom("full")
	if err == nil {
		t.Error("expected error with nil client")
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
