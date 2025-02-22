package task

import (
	"log/slog"

	"github.com/angas/solarplant-go/config"
	"github.com/angas/solarplant-go/database"
	"github.com/angas/solarplant-go/ferroamp"
	"github.com/angas/solarplant-go/types"
)

type Tasks struct {
	WeatherForecast func()
	EnergyForecast  func()
	EnergyPrice     func()
	TimeSeries      func()
	Planning        func()
}

func NewTasks(
	logger *slog.Logger,
	db *database.Database,
	epFetcher types.EnergyPriceFetcher,
	faData *ferroamp.FaInMemData,
	cnfg *config.AppConfig,
) *Tasks {
	return &Tasks{
		WeatherForecast: NewWeatcherForcastTask(logger, db, cnfg.WeatherForecast),
		EnergyForecast:  NewEnergyForecastTask(logger, db, cnfg.EnergyForecast),
		EnergyPrice:     NewEnergyPriceTask(logger, db, epFetcher),
		TimeSeries:      NewTimeSeriesTask(logger, db, faData),
		Planning:        NewPlanningTask(logger, db, cnfg, faData),
	}
}
