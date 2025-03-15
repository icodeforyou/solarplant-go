package www

import (
	"log/slog"
	"net/http"

	_ "embed"
)

func NewSysInfoHandler(logger *slog.Logger, tm *TemplateManager, sysInfo SysInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")

		if err := tm.ExecuteToWriter("sys_info.html", sysInfo, &w); err != nil {
			logger.Error("handling log request", slog.Any("error", err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
