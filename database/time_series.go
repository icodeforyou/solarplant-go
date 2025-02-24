package database

import (
	"database/sql"
	"log/slog"

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

func (d *Database) SaveTimeSeries(row TimeSeriesRow) {
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

	_, err := d.write.Exec(`
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
	panicOnError(err, "saving time series") // TODO: Handle this error properly instead of panicking
}

func (d *Database) GetTimeSeriesForHour(from hours.DateHour) ([]TimeSeriesRow, error) {
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
		from.Date, from.Hour)
	if err != nil {
		d.logger.Error("error when fetching time series", slog.Any("error", err))
		return nil, err
	}

	defer rows.Close()

	ts, err := scanTimeSeriesHours(rows)
	if err != nil {
		d.logger.Error("error when scanning time series row", slog.Any("error", err))
		return ts, err
	}

	return ts, nil
}

func (d *Database) GetTimeSeriesSinceHour(from hours.DateHour) ([]TimeSeriesRow, error) {
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
		from.Date, from.Hour, from.Date)
	if err != nil {
		d.logger.Error("error when fetching time series", slog.Any("error", err))
		return nil, err
	}

	defer rows.Close()

	ts, err := scanTimeSeriesHours(rows)
	if err != nil {
		d.logger.Error("error when scanning time series row", slog.Any("error", err))
		return ts, err
	}

	return ts, nil
}

func (d *Database) GetTimeSeriesWithEstimationsHour(from hours.DateHour) ([]TimeSeriesWithEstimationsRow, error) {
	rows, err := d.read.Query(`
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
		from.Date, from.Date, from.Hour)
	if err != nil {
		d.logger.Error("error when fetching time series with estimations", slog.Any("error", err))
		return nil, err
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
			d.logger.Error("error when scanning time series with estimations row", slog.Any("error", err))
			return nil, err
		}

		ts = append(ts, t)
	}

	return ts, nil
}

func initTimeSeries(db *sql.DB) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS time_series (
			date CHAR(10) NOT NULL,
			hour INTEGER NOT NULL,
			cloud_cover INTEGER NOT NULL,
			temperature REAL NOT NULL,
			precipitation REAL NOT NULL,
			energy_price REAL NOT NULL,
			consumption REAL NOT NULL,
			production REAL NOT NULL,
			production_lifetime REAL NOT NULL,
			battery_level REAL NOT NULL,
			battery_net_load REAL NOT NULL,
			CONSTRAINT time_series_pk PRIMARY KEY (date, hour)
		)`)
	if err != nil {
		slog.Info("error when creating time series table", slog.Any("error", err))
	}
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
