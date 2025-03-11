package task

import (
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/angas/solarplant-go/optimize"

	"github.com/angas/solarplant-go/config"
	"github.com/angas/solarplant-go/database"
	"github.com/angas/solarplant-go/ferroamp"
	"github.com/angas/solarplant-go/hours"
)

type BatteryRegulatorStrategy struct {
	// Time between each battery power update.
	Interval time.Duration

	// Minimum kw difference between the current battery power and
	// the new battery power before the new power is updated.
	UpdateThreshold float64

	// Maximum power from/to the grid in kW
	GridMaxPower float64
}

type BatteryRegulator struct {
	logger                *slog.Logger
	db                    *database.Database
	spec                  config.AppConfigBatterySpec
	faData                *ferroamp.FaInMemData
	strategy              BatteryRegulatorStrategy
	usingFallbackStrategy bool
	lastInstruction       BatteryInstruction
	C                     chan BatteryInstruction
}

const (
	ActionAuto      = "auto"
	ActionCharge    = "charge"
	ActionDischarge = "discharge"
)

type BatteryAction string

type BatteryInstruction struct {
	Action BatteryAction // charge, discharge, auto
	Power  float64       // power in kW
}

func NewBatteryRegulator(
	logger *slog.Logger,
	db *database.Database,
	bs config.AppConfigBatterySpec,
	faData *ferroamp.FaInMemData,
	strategy BatteryRegulatorStrategy) *BatteryRegulator {

	return &BatteryRegulator{
		logger:                logger,
		db:                    db,
		spec:                  bs,
		faData:                faData,
		strategy:              strategy,
		usingFallbackStrategy: false, // Keeping state to avoid spamming logs
		lastInstruction:       BatteryInstruction{},
		C:                     make(chan BatteryInstruction),
	}
}

func (br *BatteryRegulator) Run(ctx context.Context) {
	br.logger.Debug("starting battery regulator", slog.Any("interval", br.strategy.Interval))

	go func() {
		br.logger.Debug("waiting for system to stabilize")
		time.Sleep(time.Second * 60)
		ticker := time.NewTicker(br.strategy.Interval)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				br.adjustLoad(ctx)
			}
		}
	}()
}

func (br *BatteryRegulator) adjustLoad(ctx context.Context) {
	gridPwr := br.faData.GridPower()
	battLvl := br.faData.BatteryLevel()
	battPwr := br.faData.BatteryPower()
	battStatus := br.faData.BatteryStatuses()

	hour := hours.FromNow()
	planning, err := br.db.GetPlanning(ctx, hour)
	if err != nil {
		planning = database.PlanningRow{
			When:     hour,
			Strategy: optimize.StrategyDefault.String(),
		}
		if !br.usingFallbackStrategy {
			br.usingFallbackStrategy = true
			br.logger.Warn("failed to get planning for hour, using a fallback strategy",
				slog.String("hour", hour.String()),
				slog.String("strategy", planning.Strategy),
				slog.Any("error", err))
		}
	} else {
		if br.usingFallbackStrategy {
			br.usingFallbackStrategy = false
			br.logger.Info("recovered from fallback strategy, got planning for this hour",
				slog.String("hour", hour.String()),
				slog.String("strategy", planning.Strategy))
		}
	}

	sendAction := func(action BatteryAction, power float64) {
		bi := BatteryInstruction{Action: action, Power: power}
		diff := math.Abs(bi.Power - br.lastInstruction.Power)
		if bi.Action == br.lastInstruction.Action && diff < br.strategy.UpdateThreshold {
			return
		}
		// Fully charged, stop charging
		if bi.Action == ActionCharge && battLvl >= br.spec.MaxLevel {
			bi.Power = 0
		}
		// Fully discharged, stop discharging
		if bi.Action == ActionDischarge && battLvl <= br.spec.MinLevel {
			bi.Power = 0
		}

		br.logger.Debug("new battery regulation instruction",
			slog.Float64("gridPwr", gridPwr),
			slog.Float64("battLvl", battLvl),
			slog.Float64("battPwr", battPwr),
			slog.Any("battStatus", battStatus),
			slog.String("strategy", planning.Strategy),
			slog.Any("instruction", bi))

		br.lastInstruction = bi
		br.C <- bi
	}

	switch planning.Strategy {
	case optimize.StrategyDefault.String():
		sendAction(ActionAuto, 0)

	case optimize.StrategyPreserve.String():
		// Charge/discharge by 0 watts gives ESO faultCode = 8 that can be ignored
		sendAction(ActionCharge, 0)

	case optimize.StrategyCharge.String():
		freePwr := br.strategy.GridMaxPower - gridPwr - 1 // 1 kW safety zone
		newBattPwr := math.Min(br.spec.MaxChargeRate, freePwr)
		sendAction(ActionCharge, newBattPwr)

	case optimize.StrategyDischarge.String():
		freePwr := gridPwr - 1 // 1 kW safety zone
		newBattPwr := math.Max(br.spec.MaxDischargeRate, freePwr)
		sendAction(ActionDischarge, newBattPwr)

	default:
		br.logger.Error("unknown strategy", slog.Any("strategy", planning.Strategy))
	}
}
