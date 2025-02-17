package task

import (
	"context"
	"log/slog"
	"time"

	"github.com/angas/solarplant-go/database"
	"github.com/angas/solarplant-go/hours"
	"github.com/angas/solarplant-go/tibber"
)

func NewEnergyPriceTask(logger *slog.Logger, db *database.Database, tibber *tibber.Tibber) func() {
	if needImmediateEnergyPriceUpdate(db) {
		logger.Info("need an immediate update of energy prices")
		runTibberEnergyPriceTask(logger, db, tibber)
	} else {
		logger.Debug("no need for immediate update of energy prices")
	}

	return func() { runTibberEnergyPriceTask(logger, db, tibber) }
}

func runTibberEnergyPriceTask(logger *slog.Logger, db *database.Database, tibber *tibber.Tibber) {
	logger.Debug("running energy price task...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ep, err := tibber.GetEnergyPrices(ctx)
	if err != nil {
		logger.Error("error fetching tibber energy prices", slog.Any("error", err))
	} else {
		var rows []database.EnergyPriceRow = make([]database.EnergyPriceRow, 0, len(ep))
		for _, p := range ep {
			hour := hours.DateHour{Date: p.StartsAt.Format("2006-01-02"), Hour: uint8(p.StartsAt.Hour())}
			logger.Debug("energy price", "hour", slog.Any("hour", hour), slog.Float64("price", p.Energy), slog.Float64("tax", p.Tax))
			rows = append(rows, database.EnergyPriceRow{
				When:  hour,
				Price: p.Energy,
			})
		}
		db.SaveEnergyPrice(rows)
	}
}

func needImmediateEnergyPriceUpdate(db *database.Database) bool {
	dh := hours.FromNow().Add(1)
	if _, err := db.GetEnergyPriceForHour(dh); err != nil {
		return true
	}
	return false
}
