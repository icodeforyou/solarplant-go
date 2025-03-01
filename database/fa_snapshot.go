package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/angas/solarplant-go/ferroamp"
	"github.com/angas/solarplant-go/hours"
)

type FaSnapshotRow struct {
	When hours.DateHour
	Data ferroamp.FaData
}

func (r FaSnapshotRow) IsZero() bool {
	return r.When.IsZero()
}

func (d *Database) SaveFaSnapshot(ctx context.Context, row FaSnapshotRow) error {
	d.logger.Debug("saving ferroamp snapshot", "hour", row.When)

	data, err := json.Marshal(row.Data)
	if err != nil {
		return fmt.Errorf("marshalling ferroamp snapshot to JSON: %w", err)
	}

	_, err = d.write.ExecContext(ctx, `
		INSERT INTO fa_snapshot (date, hour, data)
		VALUES (?, ?, ?)`,
		row.When.Date,
		row.When.Hour,
		data,
	)
	if err != nil {
		return fmt.Errorf("saving ferroamp snapshot: %w", err)
	}

	return nil
}

func (d *Database) GetFaSnapshotForHour(ctx context.Context, dh hours.DateHour) (FaSnapshotRow, error) {
	row := d.read.QueryRowContext(ctx, `
		SELECT date, hour, data
		FROM fa_snapshot
		WHERE date = ? AND hour = ?`,
		dh.Date, dh.Hour)

	var jsonData string
	var fs FaSnapshotRow
	err := row.Scan(&fs.When.Date, &fs.When.Hour, &jsonData)
	if err == sql.ErrNoRows {
		return FaSnapshotRow{}, nil
	}
	if err != nil {
		return FaSnapshotRow{}, fmt.Errorf("fetching ferroamp snapshot for %s: %w", dh, err)
	}

	err = json.Unmarshal([]byte(jsonData), &fs.Data)
	if err != nil {
		return FaSnapshotRow{}, fmt.Errorf("unmarshalling ferroamp snapshot from JSON: %w", err)
	}

	return fs, nil
}

func (d *Database) PurgeFaSnapshot(ctx context.Context) error {
	return d.purge(ctx, "fa_snapshot")
}
