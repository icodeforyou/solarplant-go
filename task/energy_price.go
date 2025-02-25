package task

import (
	"context"
	"log/slog"
	"time"

	"github.com/angas/solarplant-go/database"
	"github.com/angas/solarplant-go/hours"
	"github.com/angas/solarplant-go/types"
)

func NewEnergyPriceTask(logger *slog.Logger, db *database.Database, fetcher types.EnergyPriceFetcher) func() {
	if needImmediateEnergyPriceUpdate(db) {
		logger.Info("need an immediate update of energy prices")
		runEnergyPriceTask(logger, db, fetcher)
	} else {
		logger.Debug("no need for immediate update of energy prices")
	}

	return func() { runEnergyPriceTask(logger, db, fetcher) }
}

func runEnergyPriceTask(logger *slog.Logger, db *database.Database, fetcher types.EnergyPriceFetcher) {
	logger.Debug("running energy price task...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	eps, err := fetcher.GetEnergyPrices(ctx)
	if err != nil {
		logger.Error("error fetching energy prices", slog.Any("error", err))
		return
	}

	var rows []database.EnergyPriceRow = make([]database.EnergyPriceRow, 0, len(eps))
	for _, ep := range eps {
		logger.Debug("energy price", "hour", slog.Any("hour", ep.Hour), slog.Float64("price", ep.Price))
		rows = append(rows, database.EnergyPriceRow{When: ep.Hour, Price: ep.Price})
	}
	db.SaveEnergyPrices(rows)

	logger.Info("energy price task done", slog.Int("noOfHoursUpdated", len(rows)))
}

func needImmediateEnergyPriceUpdate(db *database.Database) bool {
	dh := hours.FromNow().Add(1)
	if _, err := db.GetEnergyPriceForHour(dh); err != nil {
		return true
	}
	return false
}
