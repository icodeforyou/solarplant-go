package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
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
func New(ctx context.Context, path string) (*Database, error) {
	logger := slog.Default()

	sqlite.RegisterConnectionHook(func(conn sqlite.ExecQuerierContext, _ string) error {
		_, err := conn.ExecContext(ctx, initSQL, nil)
		return err
	})

	exist := exists(path)

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

	if !exist {
		logger.Info("database does not exist, creating tables")
		initWeatcherForecast(write)
		initEnergyPrice(write)
		initTimeSeries(write)
		initEnergyForecast(write)
		initPlanning(write)
		initFaSnapshot(write)
	} else {
		logger.Debug("database exists")
	}

	return &Database{logger: logger, read: read, write: write}, nil
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

func exists(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		} else {
			panic(fmt.Errorf("error when looking for database : %w", err))
		}
	}

	return !stat.IsDir()
}
