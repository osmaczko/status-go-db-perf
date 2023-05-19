package main

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"log"
	"sync/atomic"

	sqlcipher "github.com/mutecomm/go-sqlcipher" // We require go sqlcipher that overrides default implementation
)

const (
	// WALMode for sqlite.
	WALMode      = "wal"
	InMemoryPath = ":memory:"

	KdfIterationsNumber = 256000
)

var insertMessageIdx = 0
var connectionsIdx int32

type Persistence struct {
	db *sql.DB
}

func NewPersistence(path string, key string) (*Persistence, error) {
	db, err := openDB(path, key)
	return &Persistence{
		db: db,
	}, err
}

func (p *Persistence) Cleanup() {
	p.db.Close()
}

func openDB(path string, key string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	// Workaround: casting the driver type breaks the database/sql abstraction
	// and may lead to compatibility issues in the future.
	// This method is used because the 'database/sql' package does not expose 'ConnectHook',
	// thereby making it impossible to individually configure each connection.
	// Consequently, the connections from the pool can't be properly decrypted, making them unusable.
	sqlcipherDriver, ok := db.Driver().(*sqlcipher.SQLiteDriver)
	if !ok {
		return nil, fmt.Errorf("unable to get sqlcipher driver")
	}
	sqlcipherDriver.ConnectHook = func(conn *sqlcipher.SQLiteConn) error {
		if _, err = conn.Exec("PRAGMA foreign_keys=ON", []driver.Value{}); err != nil {
			log.Println("Connection setup FAILED")
			return err
		}
		keyString := fmt.Sprintf("PRAGMA key = '%s'", key)
		if _, err = conn.Exec(keyString, []driver.Value{}); err != nil {
			log.Println("Connection setup FAILED")
			return errors.New("failed to set key pragma")
		}

		if _, err = conn.Exec(fmt.Sprintf("PRAGMA kdf_iter = '%d'", KdfIterationsNumber), []driver.Value{}); err != nil {
			log.Println("Connection setup FAILED")
			return err
		}

		log.Println("Connection setup: ", atomic.AddInt32(&connectionsIdx, 1))
		return nil
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
