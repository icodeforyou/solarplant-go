package database

import (
	"context"
	"errors"
	"log/slog"

	"github.com/angas/solarplant-go/hours"
)

type PlanningRow struct {
	When     hours.DateHour
	Strategy string
}

func (d *Database) SavePanning(row PlanningRow) {
	_, err := d.write.Exec(`
		INSERT INTO planning (date, hour, strategy)
		VALUES (?, ?, ?) 
		ON CONFLICT(date, hour) DO UPDATE SET strategy = excluded.strategy;`,
		row.When.Date,
		row.When.Hour,
		row.Strategy,
	)
	panicOnError(err, "saving planning row") // TODO: Handle this error properly instead of panicking
}

func (d *Database) GetPlanningFrom(from hours.DateHour) ([]PlanningRow, error) {
	rows, err := d.read.Query(`
		SELECT date, hour, strategy
		FROM planning
		WHERE (date > ?) OR (date = ? AND hour >= ?)
		ORDER BY date, hour ASC`,
		from.Date, from.Date, from.Hour)
	if err != nil {
		d.logger.Error("error when fetching planning from date-hour", slog.Any("error", err))
		return nil, err
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

func (d *Database) GetPlanningForHour(from hours.DateHour) (PlanningRow, error) {
	rows, err := d.read.Query(`
		SELECT date, hour, strategy
		FROM planning
		WHERE date = ? AND hour = ?`,
		from.Date, from.Hour)
	if err != nil {
		d.logger.Error("error when fetching planning for date-hour", slog.Any("error", err))
		return PlanningRow{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		return PlanningRow{}, errors.New("no planning row found")
	}

	var row PlanningRow
	err = rows.Scan(&row.When.Date, &row.When.Hour, &row.Strategy)
	if err != nil {
		return PlanningRow{}, err
	}

	return row, nil
}

func (d *Database) PurgePlanning(ctx context.Context) error {
	return d.purge(ctx, "planning")
}
