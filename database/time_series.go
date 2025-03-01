package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/angas/solarplant-go/convert"
	"github.com/angas/solarplant-go/hours"
)

type TimeSeriesRow struct {
	When               hours.DateHour
	CloudCover         uint8
	Temperature        float64
	Precipitation      float64
	EnergyPrice        float64
	Production         float64
	ProductionLifetime float64
	Consumption        float64
	BatteryLevel       float64
	BatteryNetLoad     float64
}

type TimeSeriesWithEstimationsRow struct {
	TimeSeriesRow
	EstimatedConsumption sql.NullFloat64
	EstimatedProduction  sql.NullFloat64
}

func (d *Database) SaveTimeSeries(ctx context.Context, row TimeSeriesRow) error {
	d.logger.Debug("saving time series",
		"hour", row.When,
		"cloud_cover", row.CloudCover,
		"temperature", row.Temperature,
		"precipitation", row.Precipitation,
		"energy_price", row.EnergyPrice,
		"production", row.Production,
		"production_lifetime", row.ProductionLifetime,
		"consumption", row.Consumption,
		"battery_level", row.BatteryLevel,
		"battery_net_load", row.BatteryNetLoad)

	_, err := d.write.ExecContext(ctx, `
		INSERT INTO time_series (
			date,
			hour,
			cloud_cover,
			temperature,
			precipitation,
			energy_price,
			production,
			production_lifetime,
			consumption,
			battery_level,
			battery_net_load
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		row.When.Date,
		row.When.Hour,
		row.CloudCover,
		convert.TwoDecimals(row.Temperature),
		convert.TwoDecimals(row.Precipitation),
		convert.TwoDecimals(row.EnergyPrice),
		convert.TwoDecimals(row.Production),
		convert.TwoDecimals(row.ProductionLifetime),
		convert.TwoDecimals(row.Consumption),
		convert.TwoDecimals(row.BatteryLevel),
		convert.TwoDecimals(row.BatteryNetLoad),
	)

	if err != nil {
		return fmt.Errorf("saving time series: %w", err)
	}

	return nil
}

func (d *Database) GetTimeSeriesForHour(ctx context.Context, dh hours.DateHour) ([]TimeSeriesRow, error) {
	rows, err := d.read.Query(`
		SELECT 
			date, 
			hour, 
			cloud_cover, 
			temperature, 
			precipitation, 
			energy_price, 
			production, 
			production_lifetime, 
			consumption, 
			battery_level, 
			battery_net_load
		FROM time_series
		WHERE date >= ? AND hour = ?
		ORDER BY date, hour ASC`,
		dh.Date, dh.Hour)
	if err != nil {
		return nil, fmt.Errorf("fetching time series for %s: %w", dh, err)
	}

	defer rows.Close()

	ts, err := scanTimeSeriesHours(rows)
	if err != nil {
		return ts, fmt.Errorf("scanning time series row: %w", err)
	}

	return ts, nil
}

func (d *Database) GetTimeSeriesSinceHour(dh hours.DateHour) ([]TimeSeriesRow, error) {
	rows, err := d.read.Query(`
		SELECT 
			date, 
			hour, 
			cloud_cover, 
			temperature, 
			precipitation, 
			energy_price, 
			production, 
			production_lifetime, 
			consumption, 
			battery_level, 
			battery_net_load
		FROM time_series
		WHERE date >= ? AND hour >= ?
		ORDER BY date, hour ASC`,
		dh.Date, dh.Hour, dh.Date)
	if err != nil {
		return nil, fmt.Errorf("fetching time series since %s: %w", dh, err)
	}

	defer rows.Close()

	ts, err := scanTimeSeriesHours(rows)
	if err != nil {
		return ts, fmt.Errorf("scanning time series row: %w", err)
	}

	return ts, nil
}

func (d *Database) GetTimeSeriesWithEstimationsHour(ctx context.Context, dh hours.DateHour) ([]TimeSeriesWithEstimationsRow, error) {
	rows, err := d.read.QueryContext(ctx, `
		SELECT 
			ts.date,
			ts.hour,
			ts.cloud_cover,
			ts.temperature,
			ts.precipitation,
			ts.energy_price,
			ts.consumption,
			ef.consumption AS estimated_consumption,
			ts.production,
			ts.production_lifetime,
			ef.production AS estimated_production,
			ts.battery_level,
			ts.battery_net_load
		FROM time_series ts
			LEFT OUTER JOIN energy_forecast ef ON ef.date = ts.date AND ef.hour = ts.hour
		WHERE (ts.date > ?)
			OR (ts.date = ? AND ts.hour >= ?)
		ORDER BY ts.date, ts.hour ASC`,
		dh.Date, dh.Date, dh.Hour)
	if err != nil {
		return nil, fmt.Errorf("fetching time series with estimations for %s: %w", dh, err)
	}

	defer rows.Close()

	var ts []TimeSeriesWithEstimationsRow
	for rows.Next() {
		var t TimeSeriesWithEstimationsRow
		err := rows.Scan(
			&t.When.Date,
			&t.When.Hour,
			&t.CloudCover,
			&t.Temperature,
			&t.Precipitation,
			&t.EnergyPrice,
			&t.Consumption,
			&t.EstimatedConsumption,
			&t.Production,
			&t.ProductionLifetime,
			&t.EstimatedProduction,
			&t.BatteryLevel,
			&t.BatteryNetLoad)
		if err != nil {
			return nil, fmt.Errorf("scanning time series with estimations row: %w", err)
		}

		ts = append(ts, t)
	}

	return ts, nil
}

func scanTimeSeriesHours(rows *sql.Rows) ([]TimeSeriesRow, error) {
	var ts []TimeSeriesRow
	for rows.Next() {
		var t TimeSeriesRow
		err := rows.Scan(
			&t.When.Date,
			&t.When.Hour,
			&t.CloudCover,
			&t.Temperature,
			&t.Precipitation,
			&t.EnergyPrice,
			&t.Production,
			&t.ProductionLifetime,
			&t.Consumption,
			&t.BatteryLevel,
			&t.BatteryNetLoad)
		if err != nil {
			return nil, err
		}

		ts = append(ts, t)
	}

	return ts, nil
}

func (d *Database) PurgeTimeSeries(ctx context.Context) error {
	return d.purge(ctx, "time_series")
}
