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

	"github.com/gorilla/sessions"

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
	store  *sessions.CookieStore
}

//go:embed static
var embeddedStaticDir embed.FS

func StartServer(logger *slog.Logger, config config.AppConfigApi, db *database.Database, tasks *task.Tasks, fa *ferroamp.FaInMemData) *Server {
	tm, err := NewTemplateManager(config.WwwDir)
	if err != nil {
		logger.Error("template manager failed", slog.Any("error", err))
	}

	// Create a secure cookie store with a random key
	store := sessions.NewCookieStore([]byte(config.SessionKey))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   false, // Allow cookies over HTTP
		SameSite: http.SameSiteLaxMode,
	}

	s := &Server{
		logger: logger,
		config: config,
		db:     db,
		fa:     fa,
		hub:    NewHub(logger),
		tm:     tm,
		store:  store,
	}

	go s.hub.Run()

	http.HandleFunc("/login", s.handleLogin)
	http.HandleFunc("/logout", s.handleLogout)
	http.Handle("/", s.authMiddleware(staticFilesHandler(config.WwwDir)))
	http.Handle("/time_series", s.authMiddleware(http.HandlerFunc(NewTimeSeriesHandler(s.config, s.db, s.tm, tasks.TimeSeriesTask))))
	http.Handle("/energy_price", s.authMiddleware(NewEnergyPriceHandler(s.config, s.db, s.tm, tasks.EnergyPriceTask)))
	http.Handle("/weather_forecast", s.authMiddleware(NewWeatherForecastHandler(s.config, s.db, s.tm, tasks.WeatherForecastTask)))
	http.Handle("/energy_forecast", s.authMiddleware(NewEnergyForecastHandler(s.config, s.db, s.tm, tasks.EnergyForecastTask)))
	http.Handle("/planning", s.authMiddleware(NewPlanningHandler(s.config, s.db, s.tm, tasks.PlanningTask)))
	http.Handle("/log", s.authMiddleware(NewLogHandler(s.config, s.db, s.tm)))
	http.Handle("/chart", s.authMiddleware(NewChartHandler(s.db)))
	http.Handle("/ws", s.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.Header.Get("User-Agent")
		client, err := NewClient(s.hub, w, r, name)
		if err != nil {
			s.logger.Error("new websocket client failed", slog.Any("error", err))
			return
		}
		s.hub.Register <- client
		go client.WritePump()
	})))

	return s
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skippa auth för login-sidan
		if r.URL.Path == "/login" {
			next.ServeHTTP(w, r)
			return
		}

		session, _ := s.store.Get(r, "session-name")
		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		buf, err := s.tm.Execute("login.html", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(buf.Bytes())
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	s.logger.Info("login", "username", username, "password", password)
	s.logger.Info("admin user", "username", s.config.AdminUser, "password", s.config.AdminPassword)

	// Kontrollera mot konfigurerade användaruppgifter
	if username == s.config.AdminUser && password == s.config.AdminPassword {
		session, _ := s.store.Get(r, "session-name")
		session.Values["authenticated"] = true
		session.Save(r, w)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	buf, err := s.tm.Execute("login.html", map[string]interface{}{
		"Error": "Felaktigt användarnamn eller lösenord",
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(buf.Bytes())
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	session, _ := s.store.Get(r, "session-name")
	session.Values["authenticated"] = false
	session.Save(r, w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
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
			planning, err := s.db.GetPlanningForHour(hour)
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
			price, err := s.db.GetEnergyPriceForHour(hour)
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
