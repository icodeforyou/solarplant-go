package www

import (
	"log/slog"
	"net/http"
	"slices"

	_ "embed"

	"github.com/angas/solarplant-go/config"
	"github.com/angas/solarplant-go/database"
	"github.com/angas/solarplant-go/hours"
)

type templateRow struct {
	database.TimeSeriesWithEstimationsRow
	Strategy string
}

func NewTimeSeriesHandler(logger *slog.Logger, config config.AppConfigApi, db *database.Database, tm *TemplateManager, task func()) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "text/html")
			from := hours.FromNow().Sub(intOrDefault(r.URL, "hours", 24))

			ts, err := db.GetTimeSeriesWithEstimationsHour(r.Context(), from)
			if err != nil {
				logger.Error("handling time_series request", slog.Any("error", err))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			pls, err := db.GetPlanningFrom(r.Context(), from)
			if err != nil {
				logger.Error("handling time_series request", slog.Any("error", err))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			var rows []templateRow
			for _, row := range ts {
				strategy := ""
				idx := slices.IndexFunc(pls, func(p database.PlanningRow) bool { return p.When == row.When })
				if idx != -1 {
					strategy = pls[idx].Strategy
				}
				rows = append(rows, templateRow{row, strategy})
			}

			if err := tm.ExecuteToWriter("time_series.html", rows, &w); err != nil {
				logger.Error("handling time_series request", slog.Any("error", err))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		case http.MethodPost:
			go task()
			w.WriteHeader(http.StatusAccepted)

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}
