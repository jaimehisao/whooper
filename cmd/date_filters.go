package cmd

import (
	"time"

	"git.infra.hisao.org/hisao/whooper/internal/store"
)

func parseDateOnly(name, value string) (time.Time, error) {
	return store.ParseDateOnly(name, value)
}

func validateDateRange(from, to string) error {
	return store.ValidateDateRange(from, to)
}

func exportDateBounds(from, to string) (string, string, error) {
	return store.DateBounds(from, to)
}
