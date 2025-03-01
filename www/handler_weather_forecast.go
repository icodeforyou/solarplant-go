package www

import (
	"log/slog"
	"net/http"

	_ "embed"

	"github.com/angas/solarplant-go/config"
	"github.com/angas/solarplant-go/database"
	"github.com/angas/solarplant-go/hours"
)

func NewWeatherForecastHandler(logger *slog.Logger, config config.AppConfigApi, db *database.Database, tm *TemplateManager, task func()) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "text/html")
			from := hours.FromNow().Sub(intOrDefault(r.URL, "hours", 24))

			ts, err := db.GetWeatherForecastFrom(r.Context(), from)
			if err != nil {
				logger.Error("handling weather_forecast get request", slog.Any("error", err))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err := tm.ExecuteToWriter("weather_forecast.html", ts, &w); err != nil {
				logger.Error("handling weather_forecast get request", slog.Any("error", err))
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
