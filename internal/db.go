package internal

import (
	"database/sql"
	"log"
)

type Backend struct {
	Db         *sql.DB
	AppriseURL string
}

func InitDB(path string) *sql.DB {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		log.Fatalf("DB open error: %v", err)
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS records (
			email TEXT PRIMARY KEY,
			apprise_key TEXT NOT NULL,
			tags TEXT
		);`)
	if err != nil {
		log.Fatalf("DB schema error: %v", err)
	}
	return db
}

func (b *Backend) GetRecord(email string) (*Record, error) {
	var rec Record
	err := b.Db.QueryRow(
		"SELECT email, apprise_key, tags FROM records WHERE email = ?",
		email,
	).Scan(&rec.Email, &rec.Key, &rec.Tags)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}
