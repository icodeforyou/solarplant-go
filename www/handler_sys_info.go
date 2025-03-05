package www

import (
	"log/slog"
	"net/http"
	"time"

	_ "embed"
)

func NewSysInfoHandler(logger *slog.Logger, tm *TemplateManager, sysInfo SysInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		t, err := time.Parse(time.RFC3339, sysInfo.BuildTime)
		if err != nil {
			logger.Warn("parsing build time", slog.Any("error", err))
		} else {
			sysInfo.BuildTime = t.UTC().Local().Format("2006-01-02")
		}

		if err := tm.ExecuteToWriter("sys_info.html", sysInfo, &w); err != nil {
			logger.Error("handling log request", slog.Any("error", err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
