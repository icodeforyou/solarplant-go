package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/angas/solarplant-go/hours"
)

type PlanningRow struct {
	When     hours.DateHour
	Strategy string
}

type DetailedPlanningRow struct {
	PlanningRow
	EnergyPrice          sql.NullFloat64
	ProductionEstimated  sql.NullFloat64
	ConsumptionEstimated sql.NullFloat64
	CloudCover           sql.NullInt16
	Temperature          sql.NullFloat64
	Precipitation        sql.NullFloat64
}

func (d *Database) SavePanning(ctx context.Context, row PlanningRow) error {
	d.logger.Debug("saving planning",
		"hour", row.When,
		"strategy", row.Strategy)

	_, err := d.write.ExecContext(ctx, `
		INSERT INTO planning (date, hour, strategy)
		VALUES (?, ?, ?) 
		ON CONFLICT(date, hour) DO UPDATE SET strategy = excluded.strategy;`,
		row.When.Date,
		row.When.Hour,
		row.Strategy,
	)
	if err != nil {
		return fmt.Errorf("saving planning row: %w", err)
	}
	return nil
}

func (d *Database) GetPlanningFrom(ctx context.Context, dh hours.DateHour) ([]PlanningRow, error) {
	rows, err := d.read.QueryContext(ctx, `
		SELECT date, hour, strategy
		FROM planning
		WHERE (date > ?) OR (date = ? AND hour >= ?)
		ORDER BY date, hour ASC`,
		dh.Date, dh.Date, dh.Hour)
	if err != nil {
		return nil, fmt.Errorf("fetching planning from %s: %w", dh, err)
	}
	defer rows.Close()

	var res []PlanningRow
	for rows.Next() {
		var row PlanningRow
		err := rows.Scan(&row.When.Date, &row.When.Hour, &row.Strategy)
		if err != nil {
			return nil, err
		}
		res = append(res, row)
	}

	return res, nil
}

func (d *Database) GetPlanningForHour(ctx context.Context, dh hours.DateHour) (PlanningRow, error) {
	rows, err := d.read.QueryContext(ctx, `
		SELECT date, hour, strategy
		FROM planning
		WHERE date = ? AND hour = ?`,
		dh.Date, dh.Hour)
	if err != nil {
		return PlanningRow{}, fmt.Errorf("fetching planning for %s: %w", dh, err)
	}
	defer rows.Close()

	if !rows.Next() {
		return PlanningRow{}, sql.ErrNoRows
	}

	var row PlanningRow
	err = rows.Scan(&row.When.Date, &row.When.Hour, &row.Strategy)
	if err != nil {
		return PlanningRow{}, fmt.Errorf("scanning planning row: %w", err)
	}

	return row, nil
}

func (d *Database) GetDetailedPlanningFrom(ctx context.Context, dh hours.DateHour) ([]DetailedPlanningRow, error) {
	rows, err := d.read.QueryContext(ctx, `
		SELECT 
			pl.date, 
			pl.hour, 
			pl.strategy, 
			ep.price as energy_price, 
			ef.production as production_estimated,
			ef.consumption as consumption_estimated,	
			wf.cloud_cover,		
			wf.temperature,
			wf.precipitation
		FROM planning pl 
		LEFT OUTER JOIN energy_price ep ON ep.date = pl.date AND ep.hour = pl.hour
		LEFT OUTER JOIN energy_forecast ef ON ef.date = pl.date AND ef.hour = pl.hour
		LEFT OUTER JOIN weather_forecast wf ON wf.date = pl.date AND wf.hour = pl.hour
		WHERE (pl.date > ?) OR (pl.date = ? AND pl.hour >= ?)
		ORDER BY pl.date, pl.hour ASC`,
		dh.Date, dh.Date, dh.Hour)
	if err != nil {
		return nil, fmt.Errorf("fetching detailed planning from %s: %w", dh, err)
	}
	defer rows.Close()

	var res []DetailedPlanningRow
	for rows.Next() {
		var row DetailedPlanningRow
		err := rows.Scan(
			&row.When.Date,
			&row.When.Hour,
			&row.Strategy,
			&row.EnergyPrice,
			&row.ProductionEstimated,
			&row.ConsumptionEstimated,
			&row.CloudCover,
			&row.Temperature,
			&row.Precipitation)
		if err != nil {
			return nil, err
		}
		res = append(res, row)
	}

	return res, nil
}

func (d *Database) PurgePlanning(ctx context.Context) error {
	return d.purge(ctx, "planning")
}
