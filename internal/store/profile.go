package store

import (
	"database/sql"
	"fmt"

	"git.infra.hisao.org/hisao/whooper/internal/models"
)

func (db *DB) SaveProfile(p models.Profile) error {
	_, err := db.Exec(`INSERT OR REPLACE INTO profile (user_id, email, first_name, last_name)
		VALUES (?, ?, ?, ?)`,
		p.UserID, p.Email, p.FirstName, p.LastName,
	)
	return err
}

func (db *DB) GetProfile() (models.Profile, error) {
	var p models.Profile
	err := db.QueryRow(`SELECT user_id, email, first_name, last_name FROM profile ORDER BY user_id LIMIT 1`).
		Scan(&p.UserID, &p.Email, &p.FirstName, &p.LastName)
	if err == sql.ErrNoRows {
		return p, fmt.Errorf("no profile found")
	}
	return p, err
}
