package task

import (
	"context"
	"log/slog"
	"time"

	"github.com/angas/solarplant-go/config"
	"github.com/angas/solarplant-go/database"
)

func NewMaintenanceTask(logger *slog.Logger, db *database.Database, cnfg *config.AppConfig) func() {
	log := logger.With("task", "maintenance")

	return func() {
		log.Debug("running maintenance task...")

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		if err := db.PurgeLog(ctx, cnfg.Logging.GetDbMaxEntries()); err != nil {
			log.Error("error purging log", slog.Any("error", err))
		}

		if err := db.PurgeEnergyForecast(ctx); err != nil {
			log.Error("error purging energy_forecast", slog.Any("error", err))
		}

		if err := db.PurgeEnergyPrice(ctx); err != nil {
			log.Error("error purging energy_price", slog.Any("error", err))
		}

		if err := db.PurgeFaSnapshot(ctx); err != nil {
			log.Error("error purging fa_snapshot", slog.Any("error", err))
		}

		if err := db.PurgePlanning(ctx); err != nil {
			log.Error("error purging planning", slog.Any("error", err))
		}

		if err := db.PurgeTimeSeries(ctx); err != nil {
			log.Error("error purging time_series", slog.Any("error", err))
		}

		if err := db.PurgeWeatherForecast(ctx); err != nil {
			log.Error("error purging weather_forecast", slog.Any("error", err))
		}

		logger.Info("maintenance task done")
	}
}
