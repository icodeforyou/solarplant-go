package task

import (
	"context"
	"log/slog"
	"time"

	"github.com/angas/solarplant-go/database"
	"github.com/angas/solarplant-go/ferroamp"
	"github.com/angas/solarplant-go/hours"
)

func NewTimeSeriesTask(logger *slog.Logger, db *database.Database, faInMem *ferroamp.FaInMemData) func() {
	return func() {
		logger.Debug("running time series task...")

		// This task runs every hour, so we need to subtract one hour
		// to get the correct time for the snapshot. This is because
		// the snapshot contains accumulated values for the last hour and
		// this new hour has not yet been accumulated.
		currHour := hours.FromNow().Sub(1)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := db.SaveFaSnapshot(ctx, database.FaSnapshotRow{
			When: currHour,
			Data: *faInMem.CurrentState(),
		}); err != nil {
			logger.Error("time series task error", slog.Any("error", err))
		}

		prevHour, err := db.GetFaSnapshotForHour(ctx, currHour.Sub(1))
		if err != nil {
			logger.Error("planning task error, fetching snapshot from previous hour", slog.Any("error", err))
		}

		if prevHour.IsZero() {
			logger.Info("don't save time series, no snapshot from previous hour...")
			return
		}

		fc, err := db.GetWeatcherForecast(ctx, currHour)
		if err != nil {
			logger.Error("planning task error", slog.Any("error", err))
			fc = database.WeatherForecastRow{}
		}

		ep, err := db.GetEnergyPriceForHour(ctx, currHour)
		if err != nil {
			logger.Error("planning task error", slog.Any("error", err))
			ep = database.EnergyPriceRow{}
		}

		err = db.SaveTimeSeries(ctx, database.TimeSeriesRow{
			When:               currHour,
			CloudCover:         fc.CloudCover,
			Temperature:        fc.Temperature,
			Precipitation:      fc.Precipitation,
			EnergyPrice:        ep.Price,
			Production:         faInMem.ProducedSince(prevHour.Data),
			ProductionLifetime: faInMem.ProductionLifetime(),
			Consumption:        faInMem.ConsumedSince(prevHour.Data),
			BatteryLevel:       faInMem.BatteryLevel(),
			BatteryNetLoad:     faInMem.BatteryNetLoadSince(prevHour.Data),
		})
		if err != nil {
			logger.Error("time series task error", slog.Any("error", err))
		}

		logger.Info("time series task done",
			slog.Float64("productionLastHour", faInMem.ProducedSince(prevHour.Data)),
			slog.Float64("consumtionLastHour", faInMem.ConsumedSince(prevHour.Data)),
			slog.Float64("productionLifetime", faInMem.ProductionLifetime()))
	}
}
