package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/angas/solarplant-go/config"
	"github.com/angas/solarplant-go/database"
	"github.com/angas/solarplant-go/elprisetjustnu"
	"github.com/angas/solarplant-go/ferroamp"
	"github.com/angas/solarplant-go/logging"
	"github.com/angas/solarplant-go/nordpool"
	"github.com/angas/solarplant-go/task"
	"github.com/angas/solarplant-go/types"
	"github.com/angas/solarplant-go/www"
	"github.com/lmittmann/tint"
)

var (
	CommitHash = "n/a"
	BuildTime  = "n/a"
)

func main() {
	config := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consolHandler := tint.NewHandler(os.Stdout, &tint.Options{
		Level:      config.Logging.GetConsoleLevel(),
		TimeFormat: time.RFC3339,
	})

	db, err := database.New(ctx, config.Database.Path, config.Database.GetRetentionDays())
	if err != nil {
		panic(err)
	}
	defer db.Close()

	recentHours := database.NewRecentHours(db)
	if err := recentHours.Reload(ctx); err != nil {
		panic(err)
	}

	logger := slog.New(logging.NewMultiHandler(
		consolHandler,
		logging.NewSQLiteHandler(db, config.Logging.GetDbLevel(), config.Logging.GetDbAttrsFormat())))
	slog.SetDefault(logger)

	// Now we can use the logger to log database operations into the database itself
	db.SetLogger(logger.With("module", "database"))

	fa := ferroamp.New(
		config.Ferroamp.Host,
		config.Ferroamp.Port,
		config.Ferroamp.Username,
		config.Ferroamp.Password)

	faInMem := ferroamp.NewFaInMemData()
	fa.OnEhubMessage = faInMem.SetEHub
	fa.OnSsoMessage = faInMem.SetSso
	fa.OnEsmMessage = faInMem.SetEsm
	fa.OnEsoMessage = faInMem.SetEso
	fa.OnControlResponse = func(msg *ferroamp.ControlResponseMessage) {
		if msg.Status == "ack" {
			logger.Info("control request succeeded (ack)", slog.String("transId", msg.TransId), slog.String("message", msg.Message))
		} else {
			logger.Warn("control request failed (nak)", slog.String("transId", msg.TransId), slog.String("message", msg.Message))
		}
	}

	if isDevMode() {
		logger.Info("dev mode, skipping ferroamp connection")
	} else {
		if err := fa.Connect(); err != nil {
			logger.Error("ferroamp disconnected, terminating...", slog.Any("error", err))
			os.Exit(1)
		}
		defer fa.Disconnect()
	}

	energyPriceProviders := []types.EnergyPriceProvider{
		elprisetjustnu.New(config.EnergyPrice.Area), // Primary provider
		nordpool.New(config.EnergyPrice.Area),       // Secondary provider
	}

	tasks := task.NewTasks(db, energyPriceProviders, faInMem, recentHours, config)
	if isDevMode() {
		logger.Info("dev mode, skipping task scheduling")
	} else {
		tasks.Run()
		defer tasks.Stop()
	}

	regulatorStrategy := task.BatteryRegulatorStrategy{
		Interval:        time.Second * 10,
		UpdateThreshold: 0.1,
		GridMaxPower:    config.Planner.GridMaxPower,
	}
	batteryRegulator := task.NewBatteryRegulator(logger, db, config.BatterySpec, faInMem, regulatorStrategy)
	if isDevMode() {
		logger.Info("dev mode, skipping battery regulator")
	} else {
		batteryRegulator.Run(ctx)
	}

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

	sysInfo := www.SysInfo{CommitHash: CommitHash, BuildTime: BuildTime, RuntimeVersion: runtime.Version()}
	server := www.StartServer(db, tasks, faInMem, recentHours, config, sysInfo)
	server.Run(ctx)
}

func isDevMode() bool {
	return strings.EqualFold(os.Getenv("APP_ENV"), "development")
}
