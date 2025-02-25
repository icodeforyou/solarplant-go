package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	sqlite "modernc.org/sqlite"
)

type Database struct {
	logger *slog.Logger
	read   *sql.DB
	write  *sql.DB
}

const initSQL = `
	PRAGMA journal_mode = WAL;
	PRAGMA synchronous = NORMAL;
	PRAGMA temp_store = MEMORY;
	PRAGMA mmap_size = 30000000000; -- 30GB
	PRAGMA busy_timeout = 5000;
	PRAGMA automatic_index = true;
	PRAGMA foreign_keys = ON;
	PRAGMA analysis_limit = 1000;
	PRAGMA trusted_schema = OFF;
`

/**
 * New creates a new database connection.
 * Inspired by: https://theitsolutions.io/blog/modernc.org-sqlite-with-go
 */
func New(ctx context.Context, logger *slog.Logger, path string) (*Database, error) {
	sqlite.RegisterConnectionHook(func(conn sqlite.ExecQuerierContext, _ string) error {
		_, err := conn.ExecContext(ctx, initSQL, nil)
		return err
	})

	read, err := sql.Open("sqlite", path)
	if err != nil {
		logger.Error("error when opening database (read)", slog.Any("error", err))
		return nil, err
	}
	read.SetMaxOpenConns(10) // readers can be concurrent
	read.SetConnMaxIdleTime(time.Minute)

	write, err := sql.Open("sqlite", path)
	if err != nil {
		logger.Error("error when opening database (write)", slog.Any("error", err))
		return nil, err
	}
	write.SetMaxOpenConns(1) // only a single writer ever, no concurrency
	write.SetConnMaxIdleTime(time.Minute)

	err = migrate(ctx, write)
	panicOnError(err, "migrating")

	return &Database{logger: logger, read: read, write: write}, nil
}

func (d *Database) SetLogger(logger *slog.Logger) {
	d.logger = logger
}

func (d *Database) Close() {
	d.read.Close()
	d.write.Close()
}

func panicOnError(err error, action string) {
	if err != nil {
		panic(fmt.Errorf("database error when %s: %w", action, err))
	}
}
