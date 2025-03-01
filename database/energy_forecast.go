package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/angas/solarplant-go/convert"
	"github.com/angas/solarplant-go/hours"
)

type EnergyForecastRow struct {
	When        hours.DateHour
	Production  float64
	Consumption float64
}

func (d *Database) SaveEnergyForecast(ctx context.Context, rows []EnergyForecastRow) error {
	for _, row := range rows {
		d.logger.Debug("saving energy forecast",
			"hour", row.When,
			"production", row.Production,
			"consumption", row.Consumption)

		_, err := d.write.ExecContext(ctx, `
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
		if err != nil {
			return fmt.Errorf("saving energy forecast: %w", err)
		}
	}
	return nil
}

func (d *Database) GetEnergyForecast(ctx context.Context, dh hours.DateHour) (EnergyForecastRow, error) {
	rows, err := d.read.QueryContext(ctx, `
		SELECT
			date,
			hour,
			production,
			consumption
		FROM energy_forecast
		WHERE (date = ? AND hour = ?)`,
		dh.Date, dh.Hour)
	if err != nil {
		return EnergyForecastRow{}, fmt.Errorf("fetching energy forecast for %s: %w", dh, err)
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
		return EnergyForecastRow{}, fmt.Errorf("scanning energy forecast row: %w", err)
	}

	return row, nil
}

func (d *Database) GetEnergyForecastFrom(ctx context.Context, dh hours.DateHour) ([]EnergyForecastRow, error) {
	rows, err := d.read.QueryContext(ctx, `
		SELECT
			date,
			hour,
			production,
			consumption
		FROM energy_forecast
		WHERE (date = ? AND hour >= ?) OR date > ?`,
		dh.Date, dh.Hour, dh.Date)
	if err != nil {
		return nil, fmt.Errorf("fetching energy forecast from %s: %w", dh, err)
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

func (d *Database) PurgeEnergyForecast(ctx context.Context) error {
	return d.purge(ctx, "energy_forecast")
}
