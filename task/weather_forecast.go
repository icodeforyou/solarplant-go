package task

import (
	"log/slog"

	"github.com/angas/solarplant-go/config"
	"github.com/angas/solarplant-go/database"
	"github.com/angas/solarplant-go/hours"
	"github.com/angas/solarplant-go/slice"
	"github.com/angas/solarplant-go/smhi"
)

func NewWeatcherForcastTask(logger *slog.Logger, db *database.Database, config config.AppConfigWeatcherForecast) func() {
	if needImmediateForcastUpdate(db) {
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

	fc, err := smhi.Get(config.Longitude, config.Latitude)
	if err != nil {
		logger.Error("error fetching weather forecast", slog.Any("error", err))
	} else {
		db.SaveForcast(slice.Map(fc, toWeatherForecastRow))
	}
}

func needImmediateForcastUpdate(db *database.Database) bool {
	dh := hours.FromNow().Add(1)
	if _, err := db.GetWeatcherForecast(dh); err != nil {
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
