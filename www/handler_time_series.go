package www

import (
	"log/slog"
	"net/http"

	_ "embed"

	"github.com/angas/solarplant-go/database"
	"github.com/angas/solarplant-go/hours"
	"github.com/angas/solarplant-go/www/maybe"
)

type templateRow struct {
	When                 hours.DateHour
	CloudCover           maybe.Maybe[uint8]
	Temperature          maybe.Maybe[float64]
	Precipitation        maybe.Maybe[float64]
	EnergyPrice          maybe.Maybe[float64]
	Production           maybe.Maybe[float64]
	ProductionEstimated  maybe.Maybe[float64]
	Consumption          maybe.Maybe[float64]
	ConsumptionEstimated maybe.Maybe[float64]
	BatteryLevel         maybe.Maybe[float64]
	BatteryNetLoad       maybe.Maybe[float64]
	GridExport           maybe.Maybe[float64]
	GridImport           maybe.Maybe[float64]
	CashFlow             maybe.Maybe[float64]
	Strategy             maybe.Maybe[string]
	ComparedToThisHour   int
}

func NewTimeSeriesHandler(logger *slog.Logger, db *database.Database, tm *TemplateManager, recentHours *database.RecentHours) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		var rows []templateRow
		thisHour := hours.FromNow()
		hour := hours.FromMidnight()
		if thisHour.Hour-hour.Hour < 12 {
			hour = thisHour.Sub(12)
		}

		for {
			recentHour, ok := recentHours.Get(hour)
			if !ok {
				break
			}
			hour = hour.Add(1)

			rows = append(rows, templateRow{
				When:                 recentHour.When,
				CloudCover:           maybe.Some(recentHour.Ts.CloudCover),
				Temperature:          maybe.Some(recentHour.Ts.Temperature),
				Precipitation:        maybe.Some(recentHour.Ts.Precipitation),
				EnergyPrice:          maybe.Some(recentHour.Ts.EnergyPrice),
				Production:           maybe.Some(recentHour.Ts.Production),
				ProductionEstimated:  maybe.Some(recentHour.Ts.ProductionEstimated),
				Consumption:          maybe.Some(recentHour.Ts.Consumption),
				ConsumptionEstimated: maybe.Some(recentHour.Ts.ConsumptionEstimated),
				GridExport:           maybe.Some(recentHour.Ts.GridExport),
				GridImport:           maybe.Some(recentHour.Ts.GridImport),
				BatteryLevel:         maybe.Some(recentHour.Ts.BatteryLevel),
				BatteryNetLoad:       maybe.Some(recentHour.Ts.BatteryNetLoad),
				CashFlow:             maybe.Some(recentHour.Ts.CashFlow),
				Strategy:             maybe.Some(recentHour.Ts.Strategy),
				ComparedToThisHour:   recentHour.When.Compare(thisHour),
			})
		}

		// Append forecast data as long as it has been planned
		if len(rows) > 0 {
			from := rows[len(rows)-1].When.Add(1)

			forecast, err := db.GetDetailedPlanningFrom(r.Context(), from)
			if err != nil {
				logger.Error("fetching detailed planning", slog.Any("error", err))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			for _, f := range forecast {
				row := templateRow{
					When:                 f.When,
					CloudCover:           maybe.SqlNull(uint8(f.CloudCover.Int16), f.CloudCover.Valid),
					Temperature:          maybe.SqlNull(f.Temperature.Float64, f.Temperature.Valid),
					Precipitation:        maybe.SqlNull(f.Precipitation.Float64, f.Precipitation.Valid),
					EnergyPrice:          maybe.SqlNull(f.EnergyPrice.Float64, f.EnergyPrice.Valid),
					Production:           maybe.None[float64](),
					ProductionEstimated:  maybe.SqlNull(f.ProductionEstimated.Float64, f.ProductionEstimated.Valid),
					Consumption:          maybe.None[float64](),
					ConsumptionEstimated: maybe.SqlNull(f.ConsumptionEstimated.Float64, f.ConsumptionEstimated.Valid),
					GridExport:           maybe.None[float64](),
					GridImport:           maybe.None[float64](),
					BatteryLevel:         maybe.None[float64](),
					BatteryNetLoad:       maybe.None[float64](),
					CashFlow:             maybe.None[float64](),
					Strategy:             maybe.Some(f.Strategy),
					ComparedToThisHour:   f.When.Compare(thisHour),
				}

				rows = append(rows, row)
			}
		}

		if err := tm.ExecuteToWriter("time_series.html", rows, &w); err != nil {
			logger.Error("handling time_series request", slog.Any("error", err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
