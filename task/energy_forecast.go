package task

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/angas/solarplant-go/config"
	"github.com/angas/solarplant-go/convert"
	"github.com/angas/solarplant-go/database"
	"github.com/angas/solarplant-go/hours"
)

type historyAverage struct {
	Production float64
	/** Compensated for battery charging */
	Consumption float64
	CloudCover  float64
	Temperature float64
}

func NewEnergyForecastTask(logger *slog.Logger, db *database.Database, config config.AppConfigEnergyForecast) func() {
	return func() {
		runEnergyForecastTask(logger.With("task", "energy-forecast-task"), db, config)
	}
}

func runEnergyForecastTask(logger *slog.Logger, db *database.Database, config config.AppConfigEnergyForecast) {
	logger.Debug("running energy forecast task...")
	hour := hours.FromNow()
	rows := make([]database.EnergyForecastRow, 0, int(config.HoursAhead))

	for i := 0; i < int(config.HoursAhead); i++ {
		hour = hour.Add(1)

		forecast, err := db.GetWeatcherForecast(hour)
		if err != nil {
			if err == sql.ErrNoRows {
				logger.Warn("energy forecast problem, forecast not found", "hour", hour.String())
			} else {
				logger.Error("energy forecast error, can't fetch forecast", slog.Any("error", err))
			}
		}

		avg, err := calcHistoryAverage(db, config, hour)
		if err != nil {
			logger.Error("energy forecast error, can't fetch history average", slog.Any("error", err))
		}

		// Normalize the production based on average cloud cover during the historical hours
		avgProduction := avg.Production + avg.Production*config.CloudCoverImpact*avg.CloudCover/8.0
		// Adjust the estimated production based on the forecasted cloud cover
		estProduction := avgProduction - avgProduction*config.CloudCoverImpact*float64(forecast.CloudCover)/8.0

		row := database.EnergyForecastRow{
			When:        hour,
			Production:  convert.TwoDecimals(estProduction),
			Consumption: convert.TwoDecimals(avg.Consumption),
		}

		rows = append(rows, row)
	}

	db.SaveEnergyForecast(rows)
}

func calcHistoryAverage(db *database.Database, config config.AppConfigEnergyForecast, hour hours.DateHour) (historyAverage, error) {
	hour = hour.Sub(24 * config.HistoricalDays)

	tsh, err := db.GetTimeSeriesForHour(hour)
	avg := historyAverage{}
	if err != nil {
		return avg, err
	}

	if (len(tsh)) == 0 {
		return avg, fmt.Errorf("no historical data found for %s", hour)
	}

	for _, h := range tsh {
		avg.Production += h.Production
		avg.Consumption += h.Consumption + h.BatteryNetLoad
		avg.Temperature += h.Temperature
		avg.CloudCover += float64(h.CloudCover)
	}

	count := float64(len(tsh))
	avg.Production = avg.Production / count
	avg.Consumption = avg.Consumption / count
	avg.Temperature = avg.Temperature / count
	avg.CloudCover = avg.CloudCover / count

	return avg, nil
}
