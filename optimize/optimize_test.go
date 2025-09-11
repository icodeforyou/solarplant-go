package optimize

import (
	"math"
	"testing"

	"github.com/icodeforyou/solarplant-go/config"
)

// TODO: Write test for Input.SellPrice and Input.

func TestOptimizer(t *testing.T) {
	input := Input{
		GridMaxPower: 25.0,
		// TODO: Adapt test to include tax etc.
		EnergyTax:          0.0,
		EnergyTaxReduction: 0.0,
		GridBenefit:        0.0,
		Battery: Battery{
			CurrentLevel: 10.0,
			AppConfigBatterySpec: config.AppConfigBatterySpec{
				Capacity:         10.0,
				MinLevel:         10.0,
				MaxLevel:         100.0,
				MaxChargeRate:    3.0,
				MaxDischargeRate: 3.0,
				DegradationCost:  0.1,
			},
		},
		Forecast: []Forecast{
			{EnergyPrice: -2.0, EnergyBalance: 2.0},
			{EnergyPrice: 0.0, EnergyBalance: 2.0},
			{EnergyPrice: 2.0, EnergyBalance: -2.0},
		},
	}

	checkPermutation(t, input, []Strategy{StrategyCharge, StrategyPreserve, StrategyPreserve}, 2.3, 40.0)
	checkPermutation(t, input, []Strategy{StrategyCharge, StrategyPreserve, StrategyDischarge}, -3.4, 10.0)

	checkBestStrategy(t, input, []Strategy{StrategyCharge, StrategyPreserve, StrategyDischarge}, -3.4, 10.0)
}

func checkPermutation(t *testing.T, input Input, perm []Strategy, cost float64, battLvl float64) {
	c, b := costForPermutation(input, perm)
	if !almostEqual(c, cost) {
		t.Errorf("got const %f, wanted %f", c, cost)
	}
	if !almostEqual(b, battLvl) {
		t.Errorf("got battery level %f, wanted %f", b, battLvl)
	}
}

func checkBestStrategy(t *testing.T, input Input, strategies []Strategy, cost float64, battLvl float64) {
	output := BestStrategies(input)

	if len(output.Strategy) != len(strategies) {
		t.Errorf("got %d strategies, wanted %d", len(output.Strategy), len(strategies))
	}

	for i, s := range strategies {
		if output.Strategy[i] != s {
			t.Errorf("got strategy '%s', wanted '%s' at position %d", output.Strategy[i], s, i)
		}
	}

	if !almostEqual(output.Cost, cost) {
		t.Errorf("got const %f, wanted %f", output.Cost, cost)
	}

	if !almostEqual(output.BatteryLevel, battLvl) {
		t.Errorf("got battery level %f, wanted %f", output.BatteryLevel, battLvl)
	}
}

func almostEqual(f1 float64, f2 float64) bool {
	return math.Abs(f1-f2) < 1e-9
}
