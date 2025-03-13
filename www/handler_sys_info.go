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

		if len(sysInfo.BuildTime) >= 10 {
			t, err := time.Parse("2006-01-02", sysInfo.BuildTime[0:10])
			if err != nil {
				sysInfo.BuildTime = "[invalid build time]"
				logger.Warn("parsing build time", slog.Any("error", err))
			} else {
				sysInfo.BuildTime = t.UTC().Format("2006-01-02")
			}
		} else {
			sysInfo.BuildTime = "[no build time]"
			logger.Warn("build time not set")
		}

		if err := tm.ExecuteToWriter("sys_info.html", sysInfo, &w); err != nil {
			logger.Error("handling log request", slog.Any("error", err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
