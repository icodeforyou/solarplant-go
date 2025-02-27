package task

import (
	"log/slog"

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

		db.SaveFaSnapshot(database.FaSnapshotRow{
			When: currHour,
			Data: *faInMem.CurrentState(),
		})

		prevHour, err := db.GetFaSnapshotForHour(currHour.Sub(1))
		if err != nil {
			logger.Error("error when fetching snapshot from previous hour", slog.Any("error", err))
		}

		if prevHour.IsZero() {
			logger.Info("don't save time series, no snapshot from previous hour...")
			return
		}

		fc, err := db.GetWeatcherForecast(currHour)
		if err != nil {
			logger.Error("error when fetching forecast hour", slog.Any("error", err))
			fc = database.WeatherForecastRow{}
		}

		ep, err := db.GetEnergyPriceForHour(currHour)
		if err != nil {
			logger.Error("error when fetching energy_price hour", slog.Any("error", err))
			ep = database.EnergyPriceRow{}
		}

		db.SaveTimeSeries(database.TimeSeriesRow{
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

		logger.Info("time series task done",
			slog.Float64("productionLastHour", faInMem.ProducedSince(prevHour.Data)),
			slog.Float64("consumtionLastHour", faInMem.ConsumedSince(prevHour.Data)),
			slog.Float64("productionLifetime", faInMem.ProductionLifetime()))
	}
}
