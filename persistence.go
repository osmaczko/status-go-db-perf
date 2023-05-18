package main

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/mutecomm/go-sqlcipher" // We require go sqlcipher that overrides default implementation
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

func NewPersistence(path string, key string) (*Persistence, error) {
	db, err := openDB(path, key)
	return &Persistence{
		db: db,
	}, err
}

func (p *Persistence) Close() {
	p.db.Close()
}

func openDB(path string, key string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	// Disable concurrent access as not supported by the driver
	db.SetMaxOpenConns(1)

	if _, err = db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, err
	}
	keyString := fmt.Sprintf("PRAGMA key = '%s'", key)
	if _, err = db.Exec(keyString); err != nil {
		return nil, errors.New("failed to set key pragma")
	}

	if _, err = db.Exec(fmt.Sprintf("PRAGMA kdf_iter = '%d'", KdfIterationsNumber)); err != nil {
		return nil, err
	}

	// readers do not block writers and faster i/o operations
	// https://www.sqlite.org/draft/wal.html
	// must be set after db is encrypted
	var mode string
	err = db.QueryRow("PRAGMA journal_mode=WAL").Scan(&mode)
	if err != nil {
		return nil, err
	}
	if mode != WALMode && path != InMemoryPath {
		return nil, fmt.Errorf("unable to set journal_mode to WAL. actual mode %s", mode)
	}

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
	if _, err := p.db.Exec("INSERT INTO user_messages (id, seen) VALUES (?, ?)", id, 0); err != nil {
		return err
	}
	insertMessageIdx++
	return nil
}
