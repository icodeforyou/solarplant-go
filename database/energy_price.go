package database

import (
	"database/sql"
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

func initEnergyPrice(db *sql.DB) {
	_, err := db.Exec(`CREATE TABLE energy_price (
	date CHAR(10) NOT NULL,
	hour INTEGER NOT NULL,
	price REAL,
	created INTEGER(4) NOT NULL DEFAULT (strftime('%s','now')),
	updated INTEGER(4) NOT NULL DEFAULT (strftime('%s','now')),
	CONSTRAINT energy_price_pk PRIMARY KEY (date, hour));

	CREATE TRIGGER energy_price_updated AFTER UPDATE ON energy_price
	BEGIN
		UPDATE energy_price SET updated = (strftime('%s','now'))
		WHERE rowid = NEW.rowid;
	END;`)
	panicOnError(err, "creating energy_price table")
}
