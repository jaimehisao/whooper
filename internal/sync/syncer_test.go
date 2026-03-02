package sync

import (
	"sync"
	"testing"
)

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
