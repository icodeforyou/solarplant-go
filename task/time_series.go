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

		defer faInMem.TakeSnapshot()

		// This task runs every hour, so we need to subtract one hour
		// to get the correct time for the snapshot. This is because
		// the snapshot contains accumulated values for the last hour and
		// this new hour has not yet been accumulated.
		dh := hours.FromNow().Sub(1)

		db.SaveFaSnapshot(database.FaSnapshotRow{
			When: dh,
			Data: *faInMem.CurrentState(),
		})

		if !faInMem.HasSnapshot() {
			logger.Info("don't save time series, no snapshot from last hours...")
			return
		}

		fc, err := db.GetWeatcherForecast(dh)
		if err != nil {
			logger.Error("error when fetching forecast hour", slog.Any("error", err))
			fc = database.WeatherForecastRow{}
		}
		ep, err := db.GetEnergyPriceForHour(dh)
		if err != nil {
			logger.Error("error when fetching energy_price hour", slog.Any("error", err))
			ep = database.EnergyPriceRow{}
		}

		db.SaveTimeSeries(database.TimeSeriesRow{
			When:               dh,
			CloudCover:         fc.CloudCover,
			Temperature:        fc.Temperature,
			Precipitation:      fc.Precipitation,
			EnergyPrice:        ep.Price,
			Production:         faInMem.ProducedSinceSnapshot(),
			ProductionLifetime: faInMem.ProductionLifetime(),
			Consumption:        faInMem.ConsumedSinceSnapshot(),
			BatteryLevel:       faInMem.BatteryLevel(),
			BatteryNetLoad:     faInMem.BatteryNetLoadSinceSnapshot(),
		})

		logger.Debug("time series stored",
			slog.Float64("productionLastHour", faInMem.ProducedSinceSnapshot()),
			slog.Float64("consumtionLastHour", faInMem.ConsumedSinceSnapshot()),
			slog.Float64("productionLifetime", faInMem.ProductionLifetime()))
	}
}
