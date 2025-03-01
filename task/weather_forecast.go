package task

import (
	"context"
	"log/slog"
	"time"

	"github.com/angas/solarplant-go/config"
	"github.com/angas/solarplant-go/database"
	"github.com/angas/solarplant-go/hours"
	"github.com/angas/solarplant-go/slice"
	"github.com/angas/solarplant-go/smhi"
)

func NewWeatcherForcastTask(logger *slog.Logger, db *database.Database, config config.AppConfigWeatcherForecast) func() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if needImmediateForcastUpdate(ctx, db) {
		logger.Info("need an immediate update of weather forecast")
		runForcastTask(logger, db, config)
	} else {
		logger.Debug("no need for immediate update of weather forecast")
	}

	return func() {
		runForcastTask(logger, db, config)
	}
}

func runForcastTask(logger *slog.Logger, db *database.Database, config config.AppConfigWeatcherForecast) {
	logger.Debug("running weather forecast task...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fc, err := smhi.Get(config.Longitude, config.Latitude)
	if err != nil {
		logger.Error("weather forecast task error", slog.Any("error", err))
	} else {
		if err = db.SaveForcast(ctx, slice.Map(fc, toWeatherForecastRow)); err != nil {
			logger.Error("weather forecast task error", slog.Any("error", err))
		}
	}

	logger.Info("weather forecast task done", slog.Int("noOfHoursUpdated", len(fc)))
}

func needImmediateForcastUpdate(ctx context.Context, db *database.Database) bool {
	dh := hours.FromNow().Add(1)
	if _, err := db.GetWeatcherForecast(ctx, dh); err != nil {
		return true
	}
	return false
}

func toWeatherForecastRow(ep smhi.WetherForecast) database.WeatherForecastRow {
	dh := hours.FromTime(ep.Hour)
	return database.WeatherForecastRow{
		When:          dh,
		CloudCover:    ep.CloudCover,
		Temperature:   ep.Temperature,
		Precipitation: ep.Precipitation,
	}
}
