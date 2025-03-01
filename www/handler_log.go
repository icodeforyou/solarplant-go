package www

import (
	"log/slog"
	"net/http"
	"strconv"

	_ "embed"

	"github.com/angas/solarplant-go/config"
	"github.com/angas/solarplant-go/database"
)

func NewLogHandler(logger *slog.Logger, config config.AppConfigApi, db *database.Database, tm *TemplateManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "text/html")

		if page, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && page > 0 {
			pageSize := 25
			if ps, err := strconv.Atoi(r.URL.Query().Get("pageSize")); err == nil && ps > 0 {
				pageSize = ps
			}

			e, err := db.GetLogEntries(r.Context(), slog.LevelDebug, page, pageSize)
			if err != nil {
				logger.Error("handling log request", slog.Any("error", err))
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			data := struct {
				Page     int
				PageSize int
				Entries  []database.LogEntryRow
			}{
				Page:     page + 1,
				PageSize: pageSize,
				Entries:  e,
			}

			if err := tm.ExecuteToWriter("log_entries.html", data, &w); err != nil {
				logger.Error("handling log request", slog.Any("error", err))
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		} else {
			if err := tm.ExecuteToWriter("log.html", nil, &w); err != nil {
				logger.Error("handling log request", slog.Any("error", err))
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
	}
}
