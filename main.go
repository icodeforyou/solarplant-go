package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/angas/solarplant-go/config"
	"github.com/angas/solarplant-go/database"
	"github.com/angas/solarplant-go/ferroamp"
	"github.com/angas/solarplant-go/hours"
	"github.com/angas/solarplant-go/task"
	"github.com/angas/solarplant-go/tibber"
	"github.com/angas/solarplant-go/www"
	"github.com/lmittmann/tint"
	"github.com/robfig/cron/v3"
)

func main() {
	config := config.Load()

	logHandler := tint.NewHandler(os.Stdout, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: time.RFC3339,
	})
	logger := slog.New(logHandler)
	slog.SetDefault(slog.New(logHandler))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := database.New(ctx, config.Database.Path)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	tibber := tibber.New(config.Tibber.ApiToken, config.Tibber.HomeId)

	fa := ferroamp.New(
		config.Ferroamp.Host,
		config.Ferroamp.Port,
		config.Ferroamp.Username,
		config.Ferroamp.Password)

	faData := newFaInMemData(logger, db)
	fa.OnEhubMessage = faData.SetEHub
	fa.OnSsoMessage = faData.SetSso
	fa.OnEsmMessage = faData.SetEsm
	fa.OnEsoMessage = faData.SetEso
	fa.OnControlResponse = func(msg *ferroamp.ControlResponseMessage) {
		if msg.Status == "ack" {
			logger.Info("control request succeeded (ack)", slog.String("transId", msg.TransId), slog.String("message", msg.Message))
		} else {
			logger.Warn("control request failed (nak)", slog.String("transId", msg.TransId), slog.String("message", msg.Message))
		}
	}

	if err := fa.Connect(); err != nil {
		logger.Error("ferroamp disconnected, terminating...", slog.Any("error", err))
		os.Exit(1)
	}
	defer fa.Disconnect()

	tasks := task.NewTasks(logger, db, tibber, faData, config)

	cron := cron.New()
	cron.AddFunc(config.WeatherForecast.RunAt, tasks.WeatherForecast)
	cron.AddFunc(config.EnergyForecast.RunAt, tasks.EnergyForecast)
	cron.AddFunc(config.EnergyPrice.RunAt, tasks.EnergyPrice)
	cron.AddFunc("@hourly", tasks.TimeSeries)
	cron.AddFunc(config.Planner.RunAt, tasks.Planning)
	cron.Start()
	defer cron.Stop()

	regulatorStrategy := task.BatteryRegulatorStrategy{
		Interval:        time.Second * 10,
		UpdateThreshold: 0.1,
		GridMaxPower:    config.Planner.GridMaxPower,
	}
	batteryRegulator := task.NewBatteryRegulator(logger, db, config.BatterySpec, faData, regulatorStrategy)
	batteryRegulator.Run(ctx)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.Info("terminating...")
				return
			case sig := <-sigCh:
				logger.Info("received signal", slog.Any("signal", sig))
				cancel()
			case batt := <-batteryRegulator.C:
				switch batt.Action {
				case task.ActionAuto:
					if err := fa.SetBatteryAuto(); err != nil {
						logger.Error("failed to set battery to auto mode", slog.Any("error", err))
					}
				case task.ActionCharge:
					if err := fa.SetBatteryLoad(-batt.Power); err != nil {
						logger.Error("failed to set battery to charge mode", slog.Any("error", err))
					}
				case task.ActionDischarge:
					if err := fa.SetBatteryLoad(batt.Power); err != nil {
						logger.Error("failed to set battery to discharge mode", slog.Any("error", err))
					}
				}
			}
		}
	}()

	server := www.StartServer(logger, config.Api, db, tasks, faData)
	server.Run(ctx)
}

/** Restore Ferroamp data from last hour snapshot or start from scratch */
func newFaInMemData(logger *slog.Logger, db *database.Database) *ferroamp.FaInMemData {
	now := time.Now().UTC()
	if now.Minute() == 59 && now.Second() > 58 {
		logger.Info("too close to the next snapshot, waiting for the next hour")
		time.Sleep(time.Second * 5)
	}
	from := hours.FromNow().Sub(1)
	snapshot, err := db.GetFaSnapshotForHour(from)
	if err != nil {
		logger.Error("failed to get snapshot", slog.Any("error", err), slog.String("hour", from.String()))
		return ferroamp.NewFaInMemData(nil)
	}

	if snapshot.IsZero() {
		logger.Info("no snapshot found, starting from scratch", slog.String("hour", from.String()))
		return ferroamp.NewFaInMemData(nil)
	}

	logger.Info("restoring snapshot", slog.String("hour", snapshot.When.String()))
	return ferroamp.NewFaInMemData(&snapshot.Data)
}
