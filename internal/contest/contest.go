package contest

import (
	"time"

	"github.com/judge/competition/internal/store"
)

func RemainingSeconds(cfg *store.ContestConfig, student *store.Student, now time.Time) int {
	if student.StartedAt == nil {
		return cfg.DurationMinutes * 60
	}
	deadline := student.StartedAt.Add(time.Duration(cfg.DurationMinutes) * time.Minute)
	rem := int(deadline.Sub(now).Seconds())
	if rem < 0 {
		return 0
	}
	return rem
}

func IsExpired(cfg *store.ContestConfig, student *store.Student, now time.Time) bool {
	if student.StartedAt == nil {
		return false
	}
	return RemainingSeconds(cfg, student, now) <= 0
}

func CanCompete(cfg *store.ContestConfig, student *store.Student, now time.Time) error {
	if !cfg.IsActive {
		return store.ErrInactive
	}
	if student.StartedAt == nil {
		return nil
	}
	if IsExpired(cfg, student, now) {
		return store.ErrExpired
	}
	return nil
}
