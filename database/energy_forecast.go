package database

import (
	"database/sql"
	"log/slog"

	"github.com/angas/solarplant-go/convert"
	"github.com/angas/solarplant-go/hours"
)

type EnergyForecastRow struct {
	When        hours.DateHour
	Production  float64
	Consumption float64
}

func (d *Database) SaveEnergyForecast(rows []EnergyForecastRow) {
	for _, row := range rows {
		_, err := d.write.Exec(`
		INSERT INTO energy_forecast (
			date,
			hour,
			production,
			consumption
		) VALUES (?, ?, ?, ?)
		ON CONFLICT(date, hour) DO UPDATE SET
			production = excluded.production,
			consumption = excluded.consumption;`,
			row.When.Date,
			row.When.Hour,
			convert.TwoDecimals(row.Production),
			convert.TwoDecimals(row.Consumption),
		)
		panicOnError(err, "saving energy forecast") // TODO: Handle this error properly instead of panicking
	}
}

func (d *Database) GetEnergyForecast(dh hours.DateHour) (EnergyForecastRow, error) {
	rows, err := d.read.Query(`
		SELECT
			date,
			hour,
			production,
			consumption
		FROM energy_forecast
		WHERE (date = ? AND hour = ?)`,
		dh.Date, dh.Hour)
	if err != nil {
		d.logger.Error("error when fetching energy forecast", slog.Any("error", err))
		return EnergyForecastRow{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		return EnergyForecastRow{}, sql.ErrNoRows
	}

	var row EnergyForecastRow
	err = rows.Scan(
		&row.When.Date,
		&row.When.Hour,
		&row.Production,
		&row.Consumption)
	if err != nil {
		return EnergyForecastRow{}, err
	}

	return row, nil
}

func (d *Database) GetEnergyForecastFrom(dh hours.DateHour) ([]EnergyForecastRow, error) {
	rows, err := d.read.Query(`
		SELECT
			date,
			hour,
			production,
			consumption
		FROM energy_forecast
		WHERE (date = ? AND hour >= ?) OR date > ?`,
		dh.Date, dh.Hour, dh.Date)
	if err != nil {
		d.logger.Error("error when fetching energy forecast from date-hour", slog.Any("error", err))
		return nil, err
	}
	defer rows.Close()

	var forecasts []EnergyForecastRow
	for rows.Next() {
		var row EnergyForecastRow
		err := rows.Scan(
			&row.When.Date,
			&row.When.Hour,
			&row.Production,
			&row.Consumption)
		if err != nil {
			return nil, err
		}
		forecasts = append(forecasts, row)
	}

	return forecasts, nil
}

func initEnergyForecast(db *sql.DB) {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS energy_forecast (
		date CHAR(10) NOT NULL,
		hour INTEGER NOT NULL,
		production REAL NOT NULL,
		consumption REAL NOT NULL,		
		created INTEGER(4) NOT NULL DEFAULT (strftime('%s','now')),
		updated INTEGER(4) NOT NULL DEFAULT (strftime('%s','now')),
		CONSTRAINT energy_forecast_pk PRIMARY KEY (date, hour));

		CREATE TRIGGER energy_forecast_updated AFTER UPDATE ON energy_forecast
		BEGIN
			UPDATE energy_forecast SET updated = (strftime('%s','now')) 
			WHERE rowid = NEW.rowid;
		END;`)
	if err != nil {
		slog.Info("error when creating energy forecast table", slog.Any("error", err))
	}
}
