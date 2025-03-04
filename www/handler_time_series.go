package www

import (
	"database/sql"
	"log/slog"
	"net/http"
	"slices"

	_ "embed"

	"github.com/angas/solarplant-go/config"
	"github.com/angas/solarplant-go/database"
	"github.com/angas/solarplant-go/hours"
)

type templateRow struct {
	When                 hours.DateHour
	CloudCover           uint8
	Temperature          float64
	Precipitation        float64
	EnergyPrice          float64
	Production           sql.NullFloat64
	Consumption          sql.NullFloat64
	BatteryLevel         sql.NullFloat64
	BatteryNetLoad       sql.NullFloat64
	EstimatedConsumption sql.NullFloat64
	EstimatedProduction  sql.NullFloat64
	Strategy             string
}

func NewTimeSeriesHandler(logger *slog.Logger, config config.AppConfigApi, db *database.Database, tm *TemplateManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		from := hours.FromNow().Sub(intOrDefault(r.URL, "hours", 24))

		timeseries, err := db.GetTimeSeriesSinceHour(r.Context(), from)
		if err != nil {
			logger.Error("handling time_series request", slog.Any("error", err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		planning, err := db.GetPlanningFrom(r.Context(), from)
		if err != nil {
			logger.Error("handling time_series request", slog.Any("error", err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		energyForecast, err := db.GetEnergyForecastFrom(r.Context(), from)
		if err != nil {
			logger.Warn("handling time_series request", slog.Any("error", err))
		}

		var rows []templateRow
		for _, ts := range timeseries {
			strategy := ""
			idx := slices.IndexFunc(planning, func(p database.PlanningRow) bool { return p.When == ts.When })
			if idx != -1 {
				strategy = planning[idx].Strategy
			}

			estCons := sql.NullFloat64{Valid: false}
			idx = slices.IndexFunc(energyForecast, func(ef database.EnergyForecastRow) bool { return ef.When == ts.When })
			if idx != -1 {
				estCons = sql.NullFloat64{Valid: true, Float64: energyForecast[idx].Consumption}
			}

			estProd := sql.NullFloat64{Valid: false}
			idx = slices.IndexFunc(energyForecast, func(ef database.EnergyForecastRow) bool { return ef.When == ts.When })
			if idx != -1 {
				estProd = sql.NullFloat64{Valid: true, Float64: energyForecast[idx].Production}
			}

			rows = append(rows, templateRow{
				When:                 ts.When,
				CloudCover:           ts.CloudCover,
				Temperature:          ts.Temperature,
				Precipitation:        ts.Precipitation,
				EnergyPrice:          ts.EnergyPrice,
				Production:           sql.NullFloat64{Valid: true, Float64: ts.Production},
				Consumption:          sql.NullFloat64{Valid: true, Float64: ts.Consumption},
				BatteryLevel:         sql.NullFloat64{Valid: true, Float64: ts.BatteryLevel},
				BatteryNetLoad:       sql.NullFloat64{Valid: true, Float64: ts.BatteryNetLoad},
				Strategy:             strategy,
				EstimatedConsumption: estCons,
				EstimatedProduction:  estProd,
			})
		}

		// Append forecast data as long as it has been planned
		if len(rows) > 0 {
			from = rows[len(rows)-1].When.Add(1)

			energyPrice, err := db.GetEnergyPriceFrom(r.Context(), from)
			if err != nil {
				logger.Warn("handling time_series request", slog.Any("error", err))
			}

			weatherForecast, err := db.GetWeatherForecastFrom(r.Context(), from)
			if err != nil {
				logger.Warn("handling time_series request", slog.Any("error", err))
			}

			for _, ep := range energyPrice {
				cloudCover, temperature, precipitation := uint8(0), 0.0, 0.0
				idx := slices.IndexFunc(weatherForecast, func(wf database.WeatherForecastRow) bool { return wf.When == ep.When })
				if idx != -1 {
					cloudCover = weatherForecast[idx].CloudCover
					temperature = weatherForecast[idx].Temperature
					precipitation = weatherForecast[idx].Precipitation
				}

				estCons, estProd := sql.NullFloat64{Valid: false}, sql.NullFloat64{Valid: false}
				idx = slices.IndexFunc(energyForecast, func(ef database.EnergyForecastRow) bool { return ef.When == ep.When })
				if idx != -1 {
					estCons = sql.NullFloat64{Valid: true, Float64: energyForecast[idx].Consumption}
					estProd = sql.NullFloat64{Valid: true, Float64: energyForecast[idx].Production}
				}

				strategy := ""
				idx = slices.IndexFunc(planning, func(p database.PlanningRow) bool { return p.When == ep.When })
				if idx != -1 {
					strategy = planning[idx].Strategy
				} else {
					break // Remaining hours have not been planned yet
				}

				rows = append(rows, templateRow{
					When:                 ep.When,
					CloudCover:           cloudCover,
					Temperature:          temperature,
					Precipitation:        precipitation,
					EnergyPrice:          ep.Price,
					EstimatedConsumption: estCons,
					EstimatedProduction:  estProd,
					Strategy:             strategy,
				})
			}
		}

		if err := tm.ExecuteToWriter("time_series.html", rows, &w); err != nil {
			logger.Error("handling time_series request", slog.Any("error", err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
