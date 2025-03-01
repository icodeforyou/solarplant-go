package task

import (
	"context"
	"log/slog"
	"time"

	"github.com/angas/solarplant-go/config"
	"github.com/angas/solarplant-go/database"
)

func NewMaintenanceTask(logger *slog.Logger, db *database.Database, cnfg *config.AppConfig) func() {
	return func() {
		logger.Debug("running maintenance task...")

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		if err := db.PurgeLog(ctx, cnfg.Logging.GetDbMaxEntries()); err != nil {
			logger.Error("log maintenance error", slog.Any("error", err))
		}

		if err := db.PurgeEnergyForecast(ctx); err != nil {
			logger.Error("energy_forecast maintenance error", slog.Any("error", err))
		}

		if err := db.PurgeEnergyPrice(ctx); err != nil {
			logger.Error("energy_price maintenance error", slog.Any("error", err))
		}

		if err := db.PurgeFaSnapshot(ctx); err != nil {
			logger.Error("fa_snapshot maintenance error", slog.Any("error", err))
		}

		if err := db.PurgePlanning(ctx); err != nil {
			logger.Error("planning maintenance error", slog.Any("error", err))
		}

		if err := db.PurgeTimeSeries(ctx); err != nil {
			logger.Error("time_series maintenance error", slog.Any("error", err))
		}

		if err := db.PurgeWeatherForecast(ctx); err != nil {
			logger.Error("weather_forecast maintenance error", slog.Any("error", err))
		}

		logger.Info("maintenance task done")
	}
}
