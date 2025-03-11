package www

import (
	"context"
	"log/slog"

	"github.com/angas/solarplant-go/calc"
	"github.com/angas/solarplant-go/config"
	"github.com/angas/solarplant-go/database"
	"github.com/angas/solarplant-go/ferroamp"
	"github.com/angas/solarplant-go/hours"
	"github.com/angas/solarplant-go/www/maybe"
)

type RealTimeData struct {
	GridPower          maybe.Maybe[float64]
	SolarPower         maybe.Maybe[float64]
	BatteryPower       maybe.Maybe[float64]
	BatteryLevel       maybe.Maybe[float64]
	EnergyPrice        maybe.Maybe[float64]
	GridImportThisHour maybe.Maybe[float64]
	GridExportThisHour maybe.Maybe[float64]
	CashFlowThisHour   maybe.Maybe[float64]
}

type RealTimeManager struct {
	db           *database.Database
	logger       *slog.Logger
	faInMem      *ferroamp.FaInMemData
	recentHours  *database.RecentHours
	config       config.AppConfigEnergyPrice
	energyPrices map[hours.DateHour]float64
}

func NewRealTimeManager(
	db *database.Database,
	faInMem *ferroamp.FaInMemData,
	recentHours *database.RecentHours,
	config config.AppConfigEnergyPrice) *RealTimeManager {
	return &RealTimeManager{
		db:          db,
		logger:      slog.Default().With("module", "real_time_manager"),
		faInMem:     faInMem,
		recentHours: recentHours,
		config:      config,
	}
}

func (m *RealTimeManager) Get(ctx context.Context) (RealTimeData, error) {
	rtd := RealTimeData{}
	thisHour := hours.FromNow()
	midnight := hours.FromMidnight()

	ep, ok := m.energyPrices[thisHour]
	if !ok {
		eps, err := m.db.GetEnergyPriceFrom(ctx, midnight)
		if err != nil {
			m.logger.Error("error getting energy prices", slog.Any("error", err))
		} else {
			m.energyPrices = make(map[hours.DateHour]float64)
			for _, ep := range eps {
				m.energyPrices[ep.When] = ep.Price
				if ep.When == thisHour {
					rtd.EnergyPrice = maybe.Some(ep.Price)
					break
				}
			}
		}
	} else {
		rtd.EnergyPrice = maybe.Some(ep)
	}

	recentHour, ok := m.recentHours.Get(thisHour.Sub(1))
	if ok {
		imp := m.faInMem.ImportedSince(recentHour.Fa.Data)
		exp := m.faInMem.ExportedSince(recentHour.Fa.Data)
		rtd.GridImportThisHour = maybe.Some(imp)
		rtd.GridExportThisHour = maybe.Some(exp)
		rtd.CashFlowThisHour = maybe.Some(calc.CashFlow(imp, exp, ep, m.config.Tax, m.config.TaxReduction, m.config.GridBenefit))
	}

	rtd.GridPower = maybe.Some(m.faInMem.GridPower())
	rtd.SolarPower = maybe.Some(m.faInMem.SolarPower())
	rtd.BatteryPower = maybe.Some(m.faInMem.BatteryPower())
	rtd.BatteryLevel = maybe.Some(m.faInMem.BatteryLevel())

	return rtd, nil
}
