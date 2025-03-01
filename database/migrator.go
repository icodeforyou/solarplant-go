package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
)

//go:embed migrations
var migrationsDir embed.FS

func migrate(ctx context.Context, db *sql.DB) error {
	var currVer int
	err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&currVer)
	if err != nil {
		return fmt.Errorf("get current version: %w", err)
	}

	files, err := migrationsDir.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}

	var sqlFiles []string
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".sql" {
			sqlFiles = append(sqlFiles, f.Name())
		}
	}

	slices.Sort(sqlFiles)

	re := regexp.MustCompile(`^(\d+)[-_]`)

	for _, name := range sqlFiles {
		matches := re.FindStringSubmatch(name)
		if len(matches) < 2 {
			return fmt.Errorf("parse version from migration file: %s", name)
		}
		nextVer, err := strconv.Atoi(matches[1])
		if err != nil {
			return fmt.Errorf("convert migration version from file %s: %w", name, err)
		}
		if nextVer <= currVer {
			continue // Skip migration if already applied
		}

		data, err := migrationsDir.ReadFile(path.Join("migrations", name))
		if err != nil {
			return fmt.Errorf("read migration file %s: %w", name, err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("start transaction for migration %d: %w", nextVer, err)
		}

		_, err = tx.ExecContext(ctx, string(data))
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %d: %w", nextVer, err)
		}

		_, err = tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d;", nextVer))
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("update database version for migration %d: %w", nextVer, err)
		}

		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("commit migration %d: %w", nextVer, err)
		}
	}

	return nil
}
