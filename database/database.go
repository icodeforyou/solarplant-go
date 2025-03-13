package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/angas/solarplant-go/hours"
	sqlite "modernc.org/sqlite"
)

type Database struct {
	logger *slog.Logger
	read   *sql.DB
	write  *sql.DB
	path   string
}

const initSQL = `
	PRAGMA journal_mode = WAL;
	PRAGMA synchronous = NORMAL;
	PRAGMA temp_store = MEMORY;
	PRAGMA busy_timeout = 5000;
	PRAGMA automatic_index = true;
	PRAGMA foreign_keys = ON;
	PRAGMA analysis_limit = 1000;
	PRAGMA trusted_schema = OFF;
`

/**
 * A new database connection.
 * Inspired by: https://theitsolutions.io/blog/modernc.org-sqlite-with-go
 */
func New(ctx context.Context, path string) (*Database, error) {
	sqlite.RegisterConnectionHook(func(conn sqlite.ExecQuerierContext, _ string) error {
		_, err := conn.ExecContext(ctx, initSQL, nil)
		return err
	})

	read, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("error when opening database (read): %w", err)
	}
	read.SetMaxOpenConns(10) // readers can be concurrent
	read.SetConnMaxIdleTime(time.Minute)

	write, err := sql.Open("sqlite", path)
	if err != nil {
		read.Close()
		return nil, fmt.Errorf("error when opening database (write): %w", err)
	}
	write.SetMaxOpenConns(1) // only a single writer ever, no concurrency
	write.SetConnMaxIdleTime(time.Minute)

	err = migrate(ctx, write)
	if err != nil {
		panic(fmt.Errorf("database migration error: %w", err))
	}

	return &Database{
			logger: slog.Default().With(slog.String("module", "database")),
			read:   read,
			write:  write,
			path:   path,
		},
		nil
}

func (d *Database) SetLogger(logger *slog.Logger) {
	d.logger = logger
}

func (d *Database) Close() {
	d.read.Close()
	d.write.Close()
}

func (d *Database) purgeData(ctx context.Context, table string, retentionDays int) error {
	d.logger.Debug(fmt.Sprintf("purging table %s", table))
	duration := 24 * time.Hour * time.Duration(retentionDays)
	before := hours.FromTime(time.Now().Add(-duration))
	res, err := d.write.ExecContext(ctx, fmt.Sprintf(`
		DELETE FROM %s 
		WHERE (date = ? AND hour < ?) OR date < ?`, table),
		before.Date, before.Hour, before.Date)
	if err != nil {
		return fmt.Errorf("error when purging %s: %w", table, err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		d.logger.Warn("can't get rows affected by purge", slog.String("table", table), slog.Any("error", err))
	} else {
		d.logger.Debug(fmt.Sprintf("purged %d rows from %s", rows, table))
	}

	return nil
}
