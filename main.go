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
	"github.com/angas/solarplant-go/elprisetjustnu"
	"github.com/angas/solarplant-go/ferroamp"
	"github.com/angas/solarplant-go/hours"
	"github.com/angas/solarplant-go/logging"
	"github.com/angas/solarplant-go/task"
	"github.com/angas/solarplant-go/www"
	"github.com/lmittmann/tint"
)

func main() {
	config := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consolHandler := tint.NewHandler(os.Stdout, &tint.Options{
		Level:      config.Logging.GetConsoleLevel(),
		TimeFormat: time.RFC3339,
	})

	db, err := database.New(ctx, slog.New(consolHandler), config.Database.Path, config.Database.GetRetentionDays())
	if err != nil {
		panic(err)
	}
	defer db.Close()

	dbHandler := logging.NewSQLiteHandler(db, config.Logging.GetDbLevel(), config.Logging.GetDbAttrsFormat())
	logger := slog.New(logging.NewMultiHandler(consolHandler, dbHandler))
	slog.SetDefault(logger)
	db.SetLogger(slog.Default())

	fa := ferroamp.New(
		config.Ferroamp.Host,
		config.Ferroamp.Port,
		config.Ferroamp.Username,
		config.Ferroamp.Password)

	faData := ferroamp.NewFaInMemData()
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

	tasks := task.NewTasks(
		logger,
		db,
		elprisetjustnu.New(config.EnergyPrice.Area),
		faData,
		config)
	tasks.Run()
	defer tasks.Stop()

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

	logger.Debug("restoring snapshot", slog.String("hour", snapshot.When.String()))
	return ferroamp.NewFaInMemData(&snapshot.Data)
}
