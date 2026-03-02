package sync

import (
	"fmt"
	"sync"
	"time"

	"git.infra.hisao.org/hisao/whooper/internal/api"
	"git.infra.hisao.org/hisao/whooper/internal/store"
)

type ProgressFunc func(entity string, count int)

type Syncer struct {
	client     *api.Client
	db         *store.DB
	onProgress ProgressFunc
}

func New(client *api.Client, db *store.DB, onProgress ProgressFunc) *Syncer {
	return &Syncer{client: client, db: db, onProgress: onProgress}
}

func (s *Syncer) progress(entity string, count int) {
	if s.onProgress != nil {
		s.onProgress(entity, count)
	}
}

// SyncAll performs an incremental sync of all data types.
// Uses 1-day overlap to catch retroactively updated scores.
func (s *Syncer) SyncAll() error {
	return s.SyncFrom("")
}

// SyncFrom syncs all data starting from the given date.
// If start is empty, uses the last sync state (incremental sync).
// Pass a specific date or "full" for a full re-sync.
func (s *Syncer) SyncFrom(start string) error {
	if err := s.syncProfile(); err != nil {
		return fmt.Errorf("sync profile: %w", err)
	}

	switch start {
	case "full":
		start = "" // empty start = fetch everything
	case "":
		start = s.getSyncStart() // incremental
	}

	type entityErr struct {
		entity string
		err    error
	}

	var mu sync.Mutex
	var errs []entityErr
	var wg sync.WaitGroup

	syncEntity := func(name string, fn func(string) error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := fn(start); err != nil {
				mu.Lock()
				errs = append(errs, entityErr{name, err})
				mu.Unlock()
			}
		}()
	}

	syncEntity("cycles", s.syncCycles)
	syncEntity("recoveries", s.syncRecoveries)
	syncEntity("sleeps", s.syncSleeps)
	syncEntity("workouts", s.syncWorkouts)

	wg.Wait()

	if len(errs) > 0 {
		msg := "sync errors:"
		for _, e := range errs {
			msg += fmt.Sprintf(" %s: %v;", e.entity, e.err)
		}
		return fmt.Errorf("%s", msg)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	for _, entity := range []string{"cycles", "recoveries", "sleeps", "workouts"} {
		if err := s.db.SetSyncState(entity, now); err != nil {
			return fmt.Errorf("save sync state for %s: %w", entity, err)
		}
	}
	return nil
}

func (s *Syncer) getSyncStart() string {
	return GetSyncStartWithOverlap(s.db)
}

func GetSyncStartWithOverlap(db interface{ GetSyncState(string) (string, error) }) string {
	if db == nil {
		return ""
	}
	last, err := db.GetSyncState("cycles")
	if err != nil || last == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, last)
	if err != nil {
		return ""
	}
	return t.Add(-24 * time.Hour).Format(time.RFC3339)
}

func FormatSyncTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

var syncEntities = []string{"cycles", "recoveries", "sleeps", "workouts"}

func (s *Syncer) syncProfile() error {
	p, err := s.client.GetProfile()
	if err != nil {
		return err
	}
	s.progress("profile", 1)
	return s.db.SaveProfile(*p)
}

func (s *Syncer) syncCycles(start string) error {
	cycles, err := s.client.GetCycles(start, "")
	if err != nil {
		return err
	}
	s.progress("cycles", len(cycles))
	return s.db.SaveCycles(cycles)
}

func (s *Syncer) syncRecoveries(start string) error {
	recoveries, err := s.client.GetRecoveries(start, "")
	if err != nil {
		return err
	}
	s.progress("recoveries", len(recoveries))
	return s.db.SaveRecoveries(recoveries)
}

func (s *Syncer) syncSleeps(start string) error {
	sleeps, err := s.client.GetSleeps(start, "")
	if err != nil {
		return err
	}
	s.progress("sleeps", len(sleeps))
	return s.db.SaveSleeps(sleeps)
}

func (s *Syncer) syncWorkouts(start string) error {
	workouts, err := s.client.GetWorkouts(start, "")
	if err != nil {
		return err
	}
	s.progress("workouts", len(workouts))
	return s.db.SaveWorkouts(workouts)
}
