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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if needImmediateEnergyPriceUpdate(ctx, db) {
		logger.Info("need an immediate update of energy prices")
		runEnergyPriceTask(logger, db, fetcher)
	} else {
		logger.Debug("no need for immediate update of energy prices")
	}

	return func() { runEnergyPriceTask(logger, db, fetcher) }
}

func runEnergyPriceTask(logger *slog.Logger, db *database.Database, fetcher types.EnergyPriceFetcher) {
	logger.Debug("running energy price task...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	eps, err := fetcher.GetEnergyPrices(ctx)
	if err != nil {
		logger.Error("energy price task error, fetching energy prices", slog.Any("error", err))
		return
	}

	var rows []database.EnergyPriceRow = make([]database.EnergyPriceRow, 0, len(eps))
	for _, ep := range eps {
		logger.Debug("energy price", slog.String("hour", ep.Hour.String()), slog.Float64("price", ep.Price))
		rows = append(rows, database.EnergyPriceRow{When: ep.Hour, Price: ep.Price})
	}
	err = db.SaveEnergyPrices(ctx, rows)
	if err != nil {
		logger.Error("energy price task error", slog.Any("error", err))
		return
	}

	logger.Info("energy price task done", slog.Int("noOfHoursUpdated", len(rows)))
}

func needImmediateEnergyPriceUpdate(ctx context.Context, db *database.Database) bool {
	dh := hours.FromNow().Add(1)
	if _, err := db.GetEnergyPriceForHour(ctx, dh); err != nil {
		return true
	}
	return false
}
