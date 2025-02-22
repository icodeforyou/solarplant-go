package database

import (
	"database/sql"
	"encoding/json"
	"log/slog"

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

func (d *Database) SaveFaSnapshot(row FaSnapshotRow) {
	d.logger.Debug("saving ferroamp snapshot", "hour", row.When)

	jsonData, err := json.Marshal(row.Data)
	panicOnError(err, "marshalling FaData to JSON")

	_, err = d.write.Exec(`
		INSERT INTO fa_snapshot (date, hour, data)
		VALUES (?, ?, ?)`,
		row.When.Date,
		row.When.Hour,
		jsonData,
	)
	panicOnError(err, "saving time series") // TODO: Handle this error properly instead of panicking
}

func (d *Database) GetFaSnapshotForHour(from hours.DateHour) (FaSnapshotRow, error) {
	row := d.read.QueryRow(`
		SELECT date, hour, data
		FROM fa_snapshot
		WHERE date = ? AND hour = ?`,
		from.Date, from.Hour)

	var jsonData string
	var fs FaSnapshotRow
	err := row.Scan(&fs.When.Date, &fs.When.Hour, &jsonData)
	if err == sql.ErrNoRows {
		return FaSnapshotRow{}, nil
	}
	if err != nil {
		return FaSnapshotRow{}, err
	}

	err = json.Unmarshal([]byte(jsonData), &fs.Data)
	if err != nil {
		d.logger.Error("error when unmarshalling FaData from JSON", slog.Any("error", err))
		return FaSnapshotRow{}, err
	}

	return fs, nil
}

func initFaSnapshot(db *sql.DB) {
	_, err := db.Exec(`
		CREATE TABLE fa_snapshot (
			date CHAR(10) NOT NULL,
			hour INTEGER NOT NULL,
			data TEXT NOT NULL,
			CONSTRAINT fa_snapshot_pk PRIMARY KEY (date, hour)
		)`)
	panicOnError(err, "creating fa_snapshot table")
}
