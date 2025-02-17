package www

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
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

func StartServer(logger *slog.Logger, config config.AppConfigApi, db *database.Database, tasks *task.Tasks, fa *ferroamp.FaInMemData) *Server {
	tm, err := NewTemplateManager(path.Join(config.WwwDir, "templates"))
	if err != nil {
		logger.Error("template manager failed", slog.Any("error", err))
	}
	err = tm.Watch()
	if err != nil {
		logger.Error("template watch failed", slog.Any("error", err))
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

	http.Handle("/", http.FileServer(http.Dir(path.Join(config.WwwDir, "static"))))
	http.HandleFunc("/time_series", NewTimeSeriesHandler(s.config, s.db, s.tm, tasks.TimeSeries))
	http.HandleFunc("/energy_price", NewEnergyPriceHandler(s.config, s.db, s.tm, tasks.EnergyPrice))
	http.HandleFunc("/weather_forecast", NewWeatherForecastHandler(s.config, s.db, s.tm, tasks.WeatherForecast))
	http.HandleFunc("/energy_forecast", NewEnergyForecastHandler(s.config, s.db, s.tm, tasks.EnergyForecast))
	http.HandleFunc("/planning", NewPlanningHandler(s.config, s.db, s.tm, tasks.Planning))
	http.HandleFunc("/chart", NewChartHandler(s.db))
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
			// TODO: This is a hack to get the current hour's energy price,
			// it should be done in a more efficient way.
			price, err := s.db.GetEnergyPriceForHour(hours.FromNow())
			if err != nil {
				s.logger.Error("failed to get energy price", slog.Any("error", err))
				price = database.EnergyPriceRow{}
			}
			planning, err := s.db.GetPlanningForHour(hours.FromNow())
			if err != nil {
				s.logger.Error("failed to get planning", slog.Any("error", err))
				planning = database.PlanningRow{}
			}

			data := RealTimeData{
				GridPower:    s.fa.GridPower(),
				SolarPower:   s.fa.SolarPower(),
				BatteryPower: s.fa.BatteryPower(),
				BatteryLevel: s.fa.BatteryLevel(),
				EnergyPrice:  price.Price,
				Strategy:     planning.Strategy,
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
