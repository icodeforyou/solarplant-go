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

		if err := db.Backup(ctx); err != nil {
			logger.Error("database backup error", slog.Any("error", err))
		}

		if err := db.PurgeBackups(ctx, cnfg.Database.GetBackupRetentionDays()); err != nil {
			logger.Error("backup maintenance error", slog.Any("error", err))
		}

		if err := db.PurgeLog(ctx, cnfg.Logging.GetDbMaxEntries()); err != nil {
			logger.Error("log maintenance error", slog.Any("error", err))
		}

		if err := db.PurgeEnergyForecast(ctx, cnfg.Database.GetDataRetentionDays()); err != nil {
			logger.Error("energy_forecast maintenance error", slog.Any("error", err))
		}

		if err := db.PurgeEnergyPrice(ctx, cnfg.Database.GetDataRetentionDays()); err != nil {
			logger.Error("energy_price maintenance error", slog.Any("error", err))
		}

		if err := db.PurgeFaSnapshot(ctx, cnfg.Database.GetDataRetentionDays()); err != nil {
			logger.Error("fa_snapshot maintenance error", slog.Any("error", err))
		}

		if err := db.PurgePlanning(ctx, cnfg.Database.GetDataRetentionDays()); err != nil {
			logger.Error("planning maintenance error", slog.Any("error", err))
		}

		if err := db.PurgeTimeSeries(ctx, cnfg.Database.GetDataRetentionDays()); err != nil {
			logger.Error("time_series maintenance error", slog.Any("error", err))
		}

		if err := db.PurgeWeatherForecast(ctx, cnfg.Database.GetDataRetentionDays()); err != nil {
			logger.Error("weather_forecast maintenance error", slog.Any("error", err))
		}

		logger.Info("maintenance task done")
	}
}
