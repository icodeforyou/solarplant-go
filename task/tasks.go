package task

import (
	"context"
	"log/slog"

	"github.com/angas/solarplant-go/config"
	"github.com/angas/solarplant-go/database"
	"github.com/angas/solarplant-go/ferroamp"
	"github.com/angas/solarplant-go/types"
	"github.com/robfig/cron/v3"
)

type Tasks struct {
	cron                *cron.Cron
	cnfg                *config.AppConfig
	WeatherForecastTask func()
	EnergyForecastTask  func()
	EnergyPriceTask     func()
	TimeSeriesTask      func()
	PlanningTask        func()
	MaintenanceTask     func()
}

func NewTasks(
	db *database.Database,
	energyPriceProviders []types.EnergyPriceProvider,
	faData *ferroamp.FaInMemData,
	cnfg *config.AppConfig,
) *Tasks {
	logger := slog.Default().With("module", "tasks")
	return &Tasks{
		cron:                cron.New(),
		cnfg:                cnfg,
		WeatherForecastTask: NewWeatcherForcastTask(logger.With(slog.String("task", "weather_forecast")), db, cnfg.WeatherForecast),
		EnergyForecastTask:  NewEnergyForecastTask(logger.With(slog.String("task", "energy_forecast")), db, cnfg.EnergyForecast),
		EnergyPriceTask:     NewEnergyPriceTask(logger.With(slog.String("task", "energy_price")), db, energyPriceProviders),
		TimeSeriesTask:      NewTimeSeriesTask(logger.With(slog.String("task", "time_series")), db, faData),
		PlanningTask:        NewPlanningTask(logger.With(slog.String("task", "planning")), db, cnfg, faData),
		MaintenanceTask:     NewMaintenanceTask(logger.With(slog.String("task", "maintenance")), db, cnfg),
	}
}

func (t *Tasks) Run() {
	_, err := t.cron.AddFunc(t.cnfg.WeatherForecast.RunAt, t.WeatherForecastTask)
	if err != nil {
		panic(err)
	}
	_, err = t.cron.AddFunc(t.cnfg.EnergyForecast.RunAt, t.EnergyForecastTask)
	if err != nil {
		panic(err)
	}
	_, err = t.cron.AddFunc(t.cnfg.EnergyPrice.RunAt, t.EnergyPriceTask)
	if err != nil {
		panic(err)
	}
	_, err = t.cron.AddFunc("@hourly", t.TimeSeriesTask)
	if err != nil {
		panic(err)
	}
	_, err = t.cron.AddFunc(t.cnfg.Planner.RunAt, t.PlanningTask)
	if err != nil {
		panic(err)
	}
	_, err = t.cron.AddFunc("30 2 * * *", t.MaintenanceTask)
	if err != nil {
		panic(err)
	}
	t.cron.Start()
}

func (t *Tasks) Stop() context.Context {
	return t.cron.Stop()
}
