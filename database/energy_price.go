package database

import (
	"context"
	"log/slog"

	"github.com/angas/solarplant-go/convert"
	"github.com/angas/solarplant-go/hours"
)

type EnergyPriceRow struct {
	When  hours.DateHour
	Price float64
}

func (d *Database) SaveEnergyPrices(rows []EnergyPriceRow) {
	for _, row := range rows {
		_, err := d.write.Exec(`
			INSERT INTO energy_price (date, hour, price) VALUES (?, ?, ?)
			ON CONFLICT(date, hour) DO UPDATE SET price = excluded.price`,
			row.When.Date,
			row.When.Hour,
			convert.RoundFloat64(row.Price, 4))
		panicOnError(err, "saving energy prices") // TODO: Handle this error properly instead of panicking
	}
}

func (d *Database) GetEnergyPriceForHour(dh hours.DateHour) (EnergyPriceRow, error) {
	row := d.read.QueryRow(`SELECT
		date, hour, price
		FROM energy_price
		WHERE date = ? AND hour = ?`,
		dh.Date, dh.Hour)

	var ep EnergyPriceRow
	err := row.Scan(&ep.When.Date, &ep.When.Hour, &ep.Price)
	if err != nil {
		d.logger.Error("error when scanning energy price row", slog.Any("error", err))
		return EnergyPriceRow{}, err
	}

	return ep, nil
}

func (d *Database) GetEnergyPriceFrom(dh hours.DateHour) ([]EnergyPriceRow, error) {
	rows, err := d.read.Query(`SELECT
		date, hour, price
		FROM energy_price
		WHERE (date = ? AND hour >= ?) OR date > ?
		ORDER BY date, hour ASC`,
		dh.Date, dh.Hour, dh.Date)
	if err != nil {
		d.logger.Error("error when fetching energy price", slog.Any("error", err))
		return nil, err
	}

	defer rows.Close()

	var energyPrices []EnergyPriceRow
	for rows.Next() {
		var ep EnergyPriceRow
		err := rows.Scan(&ep.When.Date, &ep.When.Hour, &ep.Price)
		if err != nil {
			d.logger.Error("error when scanning energy price row", slog.Any("error", err))
			return nil, err
		}

		energyPrices = append(energyPrices, ep)
	}

	return energyPrices, nil
}

func (d *Database) PurgeEnergyPrice(ctx context.Context) error {
	return d.purge(ctx, "energy_price")
}
