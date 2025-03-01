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

func (d *Database) PurgePlanning(ctx context.Context) error {
	return d.purge(ctx, "planning")
}
