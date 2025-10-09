package internal

import (
	"database/sql"
	"fmt"

	"github.com/fraddy91/smtprise/logger"
)

func InitDB(path string) *sql.DB {
	logger.Infof("Database path: %s", path)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		logger.Errorf("DB open error: %v", err)
		return nil
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS records (
		email TEXT NOT NULL,
		apprise_key TEXT NOT NULL,
		tags TEXT NOT NULL,
		mime_type TEXT NOT NULL,
		PRIMARY KEY (email, apprise_key, mime_type)
		);`)
	if err != nil {
		logger.Errorf("DB schema error: %v", err)
		return nil
	}

	return db
}

func (b *Backend) GetRecords(email string) ([]*Record, error) {
	rows, err := b.Db.Query(
		"SELECT email, apprise_key, tags, mime_type FROM records WHERE email = ?",
		email,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*Record
	for rows.Next() {
		var rec Record
		if err := rows.Scan(&rec.Email, &rec.Key, &rec.Tags, &rec.MimeType); err != nil {
			return nil, err
		}
		records = append(records, &rec)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

func (b *Backend) GetAllRecords() ([]Record, error) {
	rows, err := b.Db.Query("SELECT email, apprise_key, tags, mime_type FROM records ORDER BY email")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var recs []Record
	for rows.Next() {
		var r Record
		if err := rows.Scan(&r.Email, &r.Key, &r.Tags, &r.MimeType); err != nil {
			return nil, err
		}
		recs = append(recs, r)
	}
	return recs, nil
}

func (b *Backend) AddRecord(record *Record) error {
	_, err := b.Db.Exec(
		"INSERT OR REPLACE INTO records (email, apprise_key, tags, mime_type) VALUES (?, ?, ?, ?)",
		record.Email, record.Key, record.Tags, record.MimeType,
	)
	return err
}

func (b *Backend) UpdateRecord(field, value, email, mimeType string) error {
	allowedFields := map[string]bool{
		"apprise_key": true,
		"tags":        true,
		"mime_type":   true,
	}
	if !allowedFields[field] {
		return fmt.Errorf("invalid field name: %s", field)
	}
	query := "UPDATE records SET " + field + "=? WHERE email=? AND mime_type=?"
	_, err := b.Db.Exec(query, value, email, mimeType)
	return err
}

func (b *Backend) DeleteRecord(record *Record) error {
	_, err := b.Db.Exec(
		"DELETE FROM records WHERE email = ? AND apprise_key = ? and mime_type = ?",
		record.Email, record.Key, record.MimeType,
	)
	return err
}
