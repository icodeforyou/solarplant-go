package www

import (
	"net/http"

	_ "embed"

	"github.com/angas/solarplant-go/config"
	"github.com/angas/solarplant-go/database"
	"github.com/angas/solarplant-go/hours"
)

func NewPlanningHandler(config config.AppConfigApi, db *database.Database, tm *TemplateManager, task func()) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "text/html")
			from := hours.FromNow().Sub(intOrDefault(r.URL, "hours", 24))
			ts, err := db.GetPlanningFrom(from)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err := tm.ExecuteToWriter("planning.html", ts, &w); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		case http.MethodPost:
			task()
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}
