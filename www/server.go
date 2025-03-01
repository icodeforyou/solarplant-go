package www

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/angas/solarplant-go/config"
	"github.com/angas/solarplant-go/database"
	"github.com/angas/solarplant-go/ferroamp"
	"github.com/angas/solarplant-go/hours"
	"github.com/angas/solarplant-go/task"
)

type Server struct {
	logger *slog.Logger
	config config.AppConfigApi
	db     *database.Database
	fa     *ferroamp.FaInMemData
	hub    *Hub
	tm     *TemplateManager
}

//go:embed static
var embeddedStaticDir embed.FS

func StartServer(db *database.Database, tasks *task.Tasks, fa *ferroamp.FaInMemData, config config.AppConfigApi) *Server {
	logger := slog.Default().With("module", "www")
	tm, err := NewTemplateManager(logger, config.WwwDir)
	if err != nil {
		logger.Error("template manager initialization error", slog.Any("error", err))
	}

	s := &Server{
		logger: logger,
		config: config,
		db:     db,
		fa:     fa,
		hub:    NewHub(logger),
		tm:     tm,
	}

	go s.hub.Run()

	logReqMW := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.logger.Debug("http request",
				slog.String("method", r.Method),
				slog.String("url", r.URL.String()),
				slog.String("remoteAddr", r.RemoteAddr))
			next.ServeHTTP(w, r)
		})
	}

	http.Handle("/", staticFilesHandler(config.WwwDir))

	http.Handle("/time_series", logReqMW(NewTimeSeriesHandler(
		logger.With(slog.String("handler", "time_series")),
		s.config,
		s.db,
		s.tm,
		tasks.TimeSeriesTask)))

	http.Handle("/energy_price", logReqMW(NewEnergyPriceHandler(
		logger.With(slog.String("handler", "time_series")),
		s.config,
		s.db,
		s.tm,
		tasks.EnergyPriceTask)))

	http.Handle("/weather_forecast", logReqMW(NewWeatherForecastHandler(
		logger.With(slog.String("handler", "time_series")),
		s.config,
		s.db,
		s.tm,
		tasks.WeatherForecastTask)))

	http.Handle("/energy_forecast", logReqMW(NewEnergyForecastHandler(
		logger.With(slog.String("handler", "time_series")),
		s.config,
		s.db,
		s.tm,
		tasks.EnergyForecastTask)))

	http.Handle("/planning", logReqMW(NewPlanningHandler(
		logger.With(slog.String("handler", "time_series")),
		s.config,
		s.db,
		s.tm,
		tasks.PlanningTask)))

	http.Handle("/log", logReqMW(NewLogHandler(logger.With(
		slog.String("handler", "time_series")),
		s.config,
		s.db,
		s.tm)))

	http.Handle("/chart", logReqMW(NewChartHandler(
		logger.With(slog.String("handler", "time_series")),
		s.db)))

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		name := r.Header.Get("User-Agent")
		client, err := NewClient(s.hub, w, r, name)
		if err != nil {
			s.logger.Error("new websocket client failed", slog.Any("error", err))
			return
		}
		s.hub.Register <- client
		go client.WritePump()
	})

	return s
}

func (s *Server) Run(ctx context.Context) {
	s.logger.Info("staring server...", "port", s.config.Port)
	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", s.config.Port),
	}

	srvErrors := make(chan error, 1)

	go func() {
		srvErrors <- srv.ListenAndServe()
	}()

	ticker := time.NewTicker(time.Second * 2)
	defer ticker.Stop()

	// Keeping state to avoid spamming logs
	fetchPanningErrorState := false
	fetchEnergyPriceErrorState := false

	for {
		select {
		case err := <-srvErrors:
			if err != nil {
				s.logger.Error("server error", slog.Any("error", err))
			}

		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(ctx, time.Second*5)
			defer cancel()
			err := srv.Shutdown(shutdownCtx)
			if err != nil {
				s.logger.Error("server shutdown failed", slog.Any("error", err))
			}
			return

		case <-ticker.C:
			hour := hours.FromNow()
			// TODO: This is a hack to get the current hour's planning,
			// it should be done in a more efficient way.
			planning, err := s.db.GetPlanningForHour(ctx, hour)
			if err != nil {
				if !fetchPanningErrorState {
					fetchPanningErrorState = true
					s.logger.Warn("failed to get planning", slog.String("hour", hour.String()), slog.Any("error", err))
				}
				planning = database.PlanningRow{}
			} else {
				fetchPanningErrorState = false
			}

			// TODO: This is a hack to get the current hour's ,
			// it should be done in a more efficient way.
			price, err := s.db.GetEnergyPriceForHour(ctx, hour)
			if err != nil {
				if !fetchEnergyPriceErrorState {
					fetchEnergyPriceErrorState = true
					s.logger.Warn("failed to get energy price", slog.String("hour", hour.String()), slog.Any("error", err))
				}
				price = database.EnergyPriceRow{}
			} else {
				fetchEnergyPriceErrorState = false
			}

			data := RealTimeData{
				GridPower:    s.fa.GridPower(),
				SolarPower:   s.fa.SolarPower(),
				BatteryPower: s.fa.BatteryPower(),
				BatteryLevel: s.fa.BatteryLevel(),
				EnergyPrice:  price.Price,
				Strategy:     planning.Strategy, // TODO: If we can't get current strategy we should probably show a default value.
			}
			buf, err := s.tm.Execute("real_time_data.html", data)
			if err != nil {
				s.logger.Error("template execution failed", slog.Any("error", err))
				return
			}

			s.hub.Broadcast <- buf.Bytes()
		}
	}
}

func staticFilesHandler(extDir *string) http.Handler {
	if extDir != nil && *extDir != "" {
		staticDir := path.Join(*extDir, "static")
		if _, err := os.Stat(staticDir); err == nil {
			return http.FileServer(http.Dir(staticDir))
		}
	}

	fsys, err := fs.Sub(embeddedStaticDir, "static")
	if err != nil {
		log.Panic(err)
	}
	return http.FileServer(http.FS(fsys))
}
