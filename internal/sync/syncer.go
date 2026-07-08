package sync

import (
	"fmt"
	"sync"
	"time"

	"git.infra.hisao.org/hisao/whooper/internal/api"
	"git.infra.hisao.org/hisao/whooper/internal/models"
	"git.infra.hisao.org/hisao/whooper/internal/store"
)

type ProgressFunc func(entity string, count int)

type Syncer struct {
	client     *api.Client
	db         *store.DB
	onProgress ProgressFunc
	writeMu    sync.Mutex // serialize SQLite writes across entity syncs
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
// If start is empty, uses each entity's last sync state (incremental sync).
// Pass a specific date or "full" for a full re-sync.
//
// Entity syncs fetch from the API concurrently but persist to SQLite serially.
// Each entity advances its own sync_state on success so a single failing entity
// does not force a full re-pull of the others on the next run.
func (s *Syncer) SyncFrom(start string) error {
	if err := s.syncProfile(); err != nil {
		return fmt.Errorf("sync profile: %w", err)
	}

	mode := start // "", "full", or an explicit RFC3339 / date start shared by all entities

	type entityResult struct {
		entity string
		err    error
	}

	results := make(chan entityResult, len(syncEntities))
	var wg sync.WaitGroup

	run := func(name string, fn func(string) error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			entityStart := mode
			switch mode {
			case "full":
				entityStart = ""
			case "":
				entityStart = GetEntitySyncStartWithOverlap(s.db, name)
			}
			err := fn(entityStart)
			if err == nil {
				now := time.Now().UTC().Format(time.RFC3339)
				s.writeMu.Lock()
				stateErr := s.db.SetSyncState(name, now)
				s.writeMu.Unlock()
				if stateErr != nil {
					err = fmt.Errorf("save sync state: %w", stateErr)
				}
			}
			results <- entityResult{name, err}
		}()
	}

	run("cycles", s.syncCycles)
	run("recoveries", s.syncRecoveries)
	run("sleeps", s.syncSleeps)
	run("workouts", s.syncWorkouts)

	wg.Wait()
	close(results)

	var errs []entityResult
	for r := range results {
		if r.err != nil {
			errs = append(errs, r)
		}
	}
	if len(errs) > 0 {
		msg := "sync errors:"
		for _, e := range errs {
			msg += fmt.Sprintf(" %s: %v;", e.entity, e.err)
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

// GetEntitySyncStartWithOverlap returns that entity's watermark minus one day.
func GetEntitySyncStartWithOverlap(db interface{ GetSyncState(string) (string, error) }, entity string) string {
	if db == nil {
		return ""
	}
	last, err := db.GetSyncState(entity)
	if err != nil || last == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, last)
	if err != nil {
		return ""
	}
	return t.Add(-24 * time.Hour).Format(time.RFC3339)
}

// GetSyncStartWithOverlap returns the earliest per-entity overlap start among
// entities that have a watermark. Kept for callers/tests that need a single
// shared incremental start. Returns empty if no entity has synced yet.
func GetSyncStartWithOverlap(db interface{ GetSyncState(string) (string, error) }) string {
	if db == nil {
		return ""
	}
	var earliest time.Time
	found := false
	for _, entity := range syncEntities {
		last, err := db.GetSyncState(entity)
		if err != nil || last == "" {
			continue
		}
		t, err := time.Parse(time.RFC3339, last)
		if err != nil {
			continue
		}
		if !found || t.Before(earliest) {
			earliest = t
			found = true
		}
	}
	if !found {
		return ""
	}
	return earliest.Add(-24 * time.Hour).Format(time.RFC3339)
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
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.db.SaveProfile(*p)
}

func (s *Syncer) syncCycles(start string) error {
	var total int
	err := s.client.ForEachCycle(start, "", func(cycles []models.Cycle) error {
		total += len(cycles)
		s.progress("cycles", total)
		s.writeMu.Lock()
		defer s.writeMu.Unlock()
		return s.db.SaveCycles(cycles)
	})
	return err
}

func (s *Syncer) syncRecoveries(start string) error {
	var total int
	err := s.client.ForEachRecovery(start, "", func(recoveries []models.Recovery) error {
		total += len(recoveries)
		s.progress("recoveries", total)
		s.writeMu.Lock()
		defer s.writeMu.Unlock()
		return s.db.SaveRecoveries(recoveries)
	})
	return err
}

func (s *Syncer) syncSleeps(start string) error {
	var total int
	err := s.client.ForEachSleep(start, "", func(sleeps []models.Sleep) error {
		total += len(sleeps)
		s.progress("sleeps", total)
		s.writeMu.Lock()
		defer s.writeMu.Unlock()
		return s.db.SaveSleeps(sleeps)
	})
	return err
}

func (s *Syncer) syncWorkouts(start string) error {
	var total int
	err := s.client.ForEachWorkout(start, "", func(workouts []models.Workout) error {
		total += len(workouts)
		s.progress("workouts", total)
		s.writeMu.Lock()
		defer s.writeMu.Unlock()
		return s.db.SaveWorkouts(workouts)
	})
	return err
}
