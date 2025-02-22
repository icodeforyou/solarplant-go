package task

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/angas/solarplant-go/config"
	"github.com/angas/solarplant-go/convert"
	"github.com/angas/solarplant-go/database"
	"github.com/angas/solarplant-go/ferroamp"
	"github.com/angas/solarplant-go/hours"
	"github.com/angas/solarplant-go/optimize"
)

func NewPlanningTask(logger *slog.Logger, db *database.Database, cnfg *config.AppConfig, faInMem *ferroamp.FaInMemData) func() {
	log := logger.With("task", "planning")

	return func() {
		log.Debug("running planning task...")

		startHour := hours.FromNow().Add(1)

		optInput := optimize.Input{
			Battery: optimize.Battery{
				AppConfigBatterySpec: cnfg.BatterySpec,
				CurrentLevel:         faInMem.BatteryLevel(),
			},
			EnergyTax:    cnfg.EnergyPrice.Tax,
			GridMaxPower: cnfg.Planner.GridMaxPower,
			Forecast:     make([]optimize.Forecast, cnfg.Planner.HoursAhead),
		}

		for h := 0; h < int(cnfg.Planner.HoursAhead); h++ {
			hour := startHour.Add(h)

			ef, err := db.GetEnergyForecast(hour)
			if err != nil {
				if err == sql.ErrNoRows {
					log.Warn("can't plan upcoming hours, no energy forecast found", slog.String("hour", hour.String()))
				} else {
					log.Error("error getting energy forecast", slog.String("hour", hour.String()), slog.Any("error", err))
				}
				return
			}

			ep, err := db.GetEnergyPriceForHour(hour)
			if err != nil {
				if err == sql.ErrNoRows {
					log.Warn("can't plan upcoming hours, no energy price found", slog.String("hour", hour.String()))
				} else {
					log.Error("error getting energy price", slog.String("hour", hour.String()), slog.Any("error", err))
				}
				return
			}

			optInput.Forecast[h] = optimize.Forecast{
				EnergyPrice:   ep.Price,
				EnergyBalance: convert.TwoDecimals(ef.Production - ef.Consumption),
			}
		}

		log.Debug(fmt.Sprintf("planning for %d hours ahead", cnfg.Planner.HoursAhead),
			slog.Any("hour", startHour),
			slog.Float64("battLvl", optInput.Battery.CurrentLevel))

		optOutput := optimize.BestStrategies(optInput)

		if len(optOutput.Strategy) != int(cnfg.Planner.HoursAhead) {
			log.Error(fmt.Sprintf("didn't get strategies for %d hours ahead", cnfg.Planner.HoursAhead))
			return
		}

		for h := 0; h < int(cnfg.Planner.HoursAhead); h++ {
			dh := startHour.Add(h)
			oi := optInput.Forecast[h]
			ou := optOutput.Strategy[h]
			log.Debug(fmt.Sprintf("result for hour %s", dh),
				slog.Float64("price", oi.EnergyPrice),
				slog.Float64("balance", oi.EnergyBalance),
				slog.Any("strategy", ou))
			db.SavePanning(database.PlanningRow{
				When:     dh,
				Strategy: ou.String(),
			})
		}

		log.Debug("planning finished",
			slog.Float64("cost", optOutput.Cost),
			slog.Float64("battLvl", optOutput.BatteryLevel))
	}
}
