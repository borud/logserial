// Package sqlitestore contains the sqlite storage code.
package sqlitestore

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite" // include driver

	"github.com/borud/logserial/pkg/model"
	"github.com/borud/logserial/pkg/store"
)

type sqliteStore struct {
	mu sync.RWMutex
	db *sqlx.DB
}

var pragmas = []string{
	"PRAGMA foreign_keys = ON",    // turn on foreign keys
	"PRAGMA cache_size = -200000", // cache size in kibibytes, approx 200Mb
	"PRAGMA journal_mode = WAL",   // turn on write-ahead journaling mode
	"PRAGMA secure_delete = OFF",  // we do not need to overwrite deleted data with zeroes
	"PRAGMA synchronous = NORMAL", // this is the appropriate setting for WAL
	"PRAGMA temp_store = MEMORY",  // store any temporary tables and indices in memory
}

// errors
var (
	ErrListCancelled = errors.New("list cancelled")
)

// New SQLite storage backend.
func New(dbSpec string) (store.Store, error) {
	db, err := openDB(dbSpec)
	if err != nil {
		return nil, err
	}

	// execute pragmas
	for _, pragma := range pragmas {
		_, err := db.Exec(pragma)
		if err != nil {
			return nil, fmt.Errorf("error while executing pragma [%s]: %w", pragma, err)
		}
	}

	return &sqliteStore{
		db: db,
	}, nil
}

// Close the sqliteStore.
func (s *sqliteStore) Close() error {
	return s.db.Close()
}

// openDB opens a database. If it does not exist it is created and the schema is
// populated.  If it is memory based the schema is always created.
func openDB(dbSpec string) (*sqlx.DB, error) {
	// If the file does not already exist or the database is an in-memory database
	// we need to create the schema.
	dbNeedsCreation := true
	if !strings.Contains(dbSpec, ":memory:") {
		_, err := os.Stat(dbSpec)
		dbNeedsCreation = os.IsNotExist(err)
	}

	db, err := sqlx.Open("sqlite", dbSpec)
	if err != nil {
		return nil, fmt.Errorf("unable to open database: %w", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	if dbNeedsCreation {
		err := createSchema(db)
		if err != nil {
			return nil, fmt.Errorf("unable to create schema: %w", err)
		}
		slog.Info("created database", "dbSpec", dbSpec)
	}

	return db, nil
}

func (s *sqliteStore) Log(msg model.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.NamedExec("INSERT INTO log (ts,device,msg) VALUES(:ts, :device, :msg)", msg)
	return err
}

func (s *sqliteStore) List(ctx context.Context, msgChan chan model.Message, since, until time.Time, device ...string) error {
	defer func() {
		close(msgChan)
		slog.Info("chan closed")
	}()

	// before we perform query, make sure the context has not been cancelled
	select {
	case <-ctx.Done():
		return ErrListCancelled
	default:
		// continue
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var rows *sqlx.Rows
	var err error

	if len(device) == 0 {
		rows, err = s.db.Queryx("SELECT * FROM log WHERE ts >= ? AND ts < ? ORDER BY ts DESC", since.UnixMilli(), until.UnixMilli())
	} else {
		rows, err = s.db.Queryx("SELECT * FROM log WHERE ts >= ? AND ts < ? AND device = ? ORDER BY ts DESC", since.UnixMilli(), until.UnixMilli(), device[0])
	}

	if err != nil {
		return err
	}

	defer rows.Close()

	var msg model.Message

	for rows.Next() {
		err := rows.StructScan(&msg)
		if err != nil {
			slog.Error("error scanning rows", "since", since, "until", until, "err", err)
			return err
		}

		select {
		case <-ctx.Done():
			slog.Info("list cancelled")
			return ErrListCancelled
		case msgChan <- msg:
		}
	}

	return nil
}
