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

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := db.PurgeLog(ctx, cnfg.Logging.GetDbMaxEntries()); err != nil {
			log.Error("error purging log", slog.Any("error", err))
		}

		logger.Info("maintenance task done")
	}
}
