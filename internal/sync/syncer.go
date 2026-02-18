package sync

import (
	"fmt"
	"time"

	"git.infra.hisao.org/hisao/whooper/internal/api"
	"git.infra.hisao.org/hisao/whooper/internal/store"
)

type ProgressFunc func(entity string, count int)

type Syncer struct {
	client *api.Client
	db     *store.DB
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
	if err := s.syncProfile(); err != nil {
		return fmt.Errorf("sync profile: %w", err)
	}

	start := s.getSyncStart()

	if err := s.syncCycles(start); err != nil {
		return fmt.Errorf("sync cycles: %w", err)
	}
	if err := s.syncRecoveries(start); err != nil {
		return fmt.Errorf("sync recoveries: %w", err)
	}
	if err := s.syncSleeps(start); err != nil {
		return fmt.Errorf("sync sleeps: %w", err)
	}
	if err := s.syncWorkouts(start); err != nil {
		return fmt.Errorf("sync workouts: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	for _, entity := range []string{"cycles", "recoveries", "sleeps", "workouts"} {
		_ = s.db.SetSyncState(entity, now)
	}
	return nil
}

func (s *Syncer) getSyncStart() string {
	last, err := s.db.GetSyncState("cycles")
	if err != nil || last == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, last)
	if err != nil {
		return ""
	}
	// 1-day overlap for retroactive updates
	return t.Add(-24 * time.Hour).Format(time.RFC3339)
}

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
