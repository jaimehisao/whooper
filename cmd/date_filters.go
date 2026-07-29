package cmd

import (
	"git.infra.hisao.org/hisao/whooper/internal/store"
)

func validateDateRange(from, to string) error {
	return store.ValidateDateRange(from, to)
}

func exportDateBounds(from, to string) (string, string, error) {
	return store.DateBounds(from, to)
}
