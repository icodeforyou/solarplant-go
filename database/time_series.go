package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/angas/solarplant-go/hours"
)

type TimeSeriesRow struct {
	When                 hours.DateHour
	CloudCover           uint8
	Temperature          float64
	Precipitation        float64
	EnergyPrice          float64
	Production           float64
	ProductionEstimated  float64
	ProductionLifetime   float64
	Consumption          float64
	ConsumptionEstimated float64
	GridImport           float64
	GridExport           float64
	BatteryLevel         float64
	BatteryNetLoad       float64
	CashFlow             float64
	Strategy             string
}

func (d *Database) SaveTimeSeries(ctx context.Context, row TimeSeriesRow) error {
	d.logger.Debug("saving time series",
		"hour", row.When,
		"cloud_cover", row.CloudCover,
		"temperature", row.Temperature,
		"precipitation", row.Precipitation,
		"energy_price", row.EnergyPrice,
		"production", row.Production,
		"production_estimated", row.ProductionEstimated,
		"production_lifetime", row.ProductionLifetime,
		"consumption", row.Consumption,
		"consumption_estimated", row.ConsumptionEstimated,
		"grid_import", row.GridImport,
		"grid_export", row.GridExport,
		"battery_level", row.BatteryLevel,
		"battery_net_load", row.BatteryNetLoad,
		"cash_flow", row.CashFlow,
		"strategy", row.Strategy)

	_, err := d.write.ExecContext(ctx, `
		INSERT INTO time_series (
			date,
			hour,
			cloud_cover,
			temperature,
			precipitation,
			energy_price,
			production,
			production_estimated,
			production_lifetime,
			consumption,
			consumption_estimated,
			grid_import,
			grid_export,
			battery_level,
			battery_net_load,
			cash_flow,
			strategy
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		row.When.Date,
		row.When.Hour,
		row.CloudCover,
		row.Temperature,
		row.Precipitation,
		row.EnergyPrice,
		row.Production,
		row.ProductionEstimated,
		row.ProductionLifetime,
		row.Consumption,
		row.ConsumptionEstimated,
		row.GridImport,
		row.GridExport,
		row.BatteryLevel,
		row.BatteryNetLoad,
		row.CashFlow,
		row.Strategy,
	)

	if err != nil {
		return fmt.Errorf("saving time series: %w", err)
	}

	return nil
}

/** Returns time series entries from this date and hour and every following same hour */
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
			production_estimated,
			production_lifetime, 
			consumption, 
			consumption_estimated,
			grid_import,
			grid_export,
			battery_level, 
			battery_net_load,
			cash_flow,
			strategy
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

func (d *Database) GetTimeSeriesFrom(ctx context.Context, dh hours.DateHour) ([]TimeSeriesRow, error) {
	rows, err := d.read.QueryContext(ctx, `
		SELECT 
			date, 
			hour, 
			cloud_cover, 
			temperature, 
			precipitation, 
			energy_price, 
			production, 
			production_estimated,
			production_lifetime, 
			consumption, 
			consumption_estimated,
			grid_import,
			grid_export,
			battery_level, 
			battery_net_load,
			cash_flow,
			strategy
		FROM time_series
		WHERE (date >= ? AND hour >= ?) OR (date > ?)
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
			&t.ProductionEstimated,
			&t.ProductionLifetime,
			&t.Consumption,
			&t.ConsumptionEstimated,
			&t.GridImport,
			&t.GridExport,
			&t.BatteryLevel,
			&t.BatteryNetLoad,
			&t.CashFlow,
			&t.Strategy)
		if err != nil {
			return nil, err
		}

		ts = append(ts, t)
	}

	return ts, nil
}

func (d *Database) PurgeTimeSeries(ctx context.Context, retentionDays int) error {
	return d.purgeData(ctx, "time_series", retentionDays)
}
