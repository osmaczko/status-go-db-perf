package main

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"

	sqlcipher "github.com/mutecomm/go-sqlcipher" // We require go sqlcipher that overrides default implementation
)

const (
	// WALMode for sqlite.
	WALMode      = "wal"
	InMemoryPath = ":memory:"

	KdfIterationsNumber = 256000
)

var insertMessageIdx = 0

type Persistence struct {
	db *sql.DB
}

func NewPersistence(path string, key string, maxOpenConns int, maxIdleConns int) (*Persistence, error) {
	db, err := openDB(path, key, maxOpenConns, maxIdleConns)
	return &Persistence{
		db: db,
	}, err
}

func (p *Persistence) Cleanup() {
	p.db.Close()
}

func openDB(path string, key string, maxOpenConns int, maxIdleConns int) (*sql.DB, error) {
	driverName := fmt.Sprintf("sqlcipher_with_extensions-%d", len(sql.Drivers()))
	sql.Register(driverName, &sqlcipher.SQLiteDriver{
		ConnectHook: func(conn *sqlcipher.SQLiteConn) error {
			if _, err := conn.Exec("PRAGMA foreign_keys=ON", []driver.Value{}); err != nil {
				return err
			}
			keyString := fmt.Sprintf("PRAGMA key = '%s'", key)
			if _, err := conn.Exec(keyString, []driver.Value{}); err != nil {
				return errors.New("failed to set key pragma")
			}

			if _, err := conn.Exec(fmt.Sprintf("PRAGMA kdf_iter = '%d'", KdfIterationsNumber), []driver.Value{}); err != nil {
				return err
			}

			// readers do not block writers and faster i/o operations
			if _, err := conn.Exec("PRAGMA journal_mode=WAL", []driver.Value{}); err != nil {
				return err
			}

			// workaround to mitigate the issue of "database is locked" errors during concurrent write operations
			if _, err := conn.Exec("PRAGMA busy_timeout=60000", []driver.Value{}); err != nil {
				return err
			}

			return nil
		},
	})

	db, err := sql.Open(driverName, path)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)

	return db, nil
}

func (p *Persistence) QueryUnseenMessages() ([]string, error) {
	rows, err := p.db.Query("SELECT id FROM user_messages WHERE seen=0")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ids, nil
}

func (p *Persistence) InsertUnseenMessage() error {
	id := fmt.Sprintf("msg-%d", insertMessageIdx)

	query := `
	INSERT INTO user_messages (
		id, whisper_timestamp, source, text, content_type,
		timestamp, chat_id, local_chat_id, clock_value, seen,
		replace_message, rtl, line_count, image_base64, audio_base64
	) VALUES (?, 0, "", "", 0, 0, "", "", 0, ?, "", 0, 0, "", "")`

	if _, err := p.db.Exec(query, id, false); err != nil {
		return err
	}

	insertMessageIdx++
	return nil
}
