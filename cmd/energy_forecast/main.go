package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/angas/solarplant-go/config"
	"github.com/angas/solarplant-go/database"
	"github.com/angas/solarplant-go/task"
	"github.com/lmittmann/tint"
)

func main() {
	w := os.Stdout
	logger := slog.New(tint.NewHandler(w, nil))
	slog.SetDefault(slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.RFC3339Nano,
		}),
	))

	config := config.Load()
	db, err := database.New(context.Background(), config.Database.Path)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	task.NewEnergyForecastTask(logger, db, config.EnergyForecast)()
}
