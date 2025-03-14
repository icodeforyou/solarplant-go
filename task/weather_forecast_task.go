package task

import (
	"context"
	"log/slog"
	"time"

	"github.com/angas/solarplant-go/config"
	"github.com/angas/solarplant-go/database"
	"github.com/angas/solarplant-go/hours"
	"github.com/angas/solarplant-go/smhi"
)

func NewWeatherForecastTask(logger *slog.Logger, db *database.Database, config config.AppConfigWeatherForecast) func() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if needImmediateForecastUpdate(ctx, db) {
		logger.Info("need an immediate update of weather forecast")
		runForecastTask(logger, db, config)
	} else {
		logger.Debug("no need for immediate update of weather forecast")
	}

	return func() {
		runForecastTask(logger, db, config)
	}
}

func runForecastTask(logger *slog.Logger, db *database.Database, config config.AppConfigWeatherForecast) {
	logger.Debug("running weather forecast task...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fc, err := smhi.Get(config.Longitude, config.Latitude)
	if err != nil {
		logger.Error("weather forecast task error", slog.Any("error", err))
	} else {
		rows := make([]database.WeatherForecastRow, len(fc))
		for i, ep := range fc {
			rows[i] = database.WeatherForecastRow{
				When:          hours.FromTime(ep.Hour),
				CloudCover:    ep.CloudCover,
				Temperature:   ep.Temperature,
				Precipitation: ep.Precipitation,
			}
		}
		if err = db.SaveForecast(ctx, rows); err != nil {
			logger.Error("weather forecast task error", slog.Any("error", err))
		}
	}

	logger.Info("weather forecast task done", slog.Int("noOfHoursUpdated", len(fc)))
}

func needImmediateForecastUpdate(ctx context.Context, db *database.Database) bool {
	dh := hours.FromNow().Add(12)
	if _, err := db.GetWeatherForecast(ctx, dh); err != nil {
		return true
	}
	return false
}
