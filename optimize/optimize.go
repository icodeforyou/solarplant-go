package optimize

import (
	"math"
	"sort"
)

type Forecast struct {
	EnergyPrice   float64 // Price of energy per kWh
	EnergyBalance float64 // Difference between produced and consumed power (kWh) not including the battery effect
}

type Input struct {
	Battery            Battery
	GridMaxPower       float64 // Maximum power to and from grid in kW
	EnergyTax          float64 // Energy tax in SEK/kWh including VAT (energiskatt)
	EnergyTaxReduction float64 // Energy tax reduction in SEK/kWh (skattereduktion)
	GridBenefit        float64 // Grid benefit in SEK/kWh (nätnytta)
	Forecast           []Forecast
}

func (i *Input) BuyPrice(price float64, kWh float64) float64 {
	return kWh * (price + i.EnergyTax - i.GridBenefit)
}

func (i *Input) SellPrice(price float64, kWh float64) float64 {
	return kWh * (price + i.EnergyTaxReduction)
}

type Output struct {
	Cost         float64    // Total cost of energy
	BatteryLevel float64    // Final battery level in percentage
	Strategy     []Strategy // Optimal strategy for each hour in the forecast
}

// BestStrategies calculates optimal charging/discharging
// based on energy prices and other constraints
func BestStrategies(input Input) Output {
	type permutationCost struct {
		Cost         float64
		BatteryLevel float64
		Permutation  []Strategy
	}

	// Generate all permutations of strategies
	// for the forecast length and then calculate
	// the cost for each permutation
	permutations := []permutationCost{}
	for _, p := range permute(len(input.Forecast), []Strategy{}) {
		cost, battLvl := costForPermutation(input, p)
		permutations = append(permutations, permutationCost{
			Cost:         cost,
			BatteryLevel: battLvl,
			Permutation:  p,
		})
	}

	// Sort permutations by cost, lowest first
	sort.Slice(permutations, func(p1, p2 int) bool {
		return permutations[p1].Cost < permutations[p2].Cost
	})

	return Output{
		Cost:         permutations[0].Cost,
		BatteryLevel: permutations[0].BatteryLevel,
		Strategy:     permutations[0].Permutation,
	}
}

// Calculates the total cost for a given permutation of strategies,
// i.e. how much money is spent (or earned) to/from the grid.
// Also returns new battery level in percentage.
func costForPermutation(input Input, permutation []Strategy) (float64, float64) {
	batt := input.Battery
	totCost := 0.0
	disqualified := false

	for hour, strategy := range permutation {
		price := input.Forecast[hour].EnergyPrice
		balance := input.Forecast[hour].EnergyBalance

		switch strategy {
		case StrategyDefault:
			battDiffKWh := batt.UpdateLevel(balance)
			buyKwh := max(0.0, battDiffKWh-balance)
			if buyKwh > 0 {
				totCost += input.BuyPrice(price, buyKwh)
			}
			sellKwh := max(0.0, balance-battDiffKWh)
			if sellKwh > 0 {
				totCost -= input.SellPrice(price, sellKwh)
			}
			totCost += batt.DegradationCost * math.Abs(battDiffKWh)

		case StrategyPreserve:
			if balance < 0 {
				totCost += input.BuyPrice(price, -balance)
			}
			if balance > 0 {
				totCost -= input.SellPrice(price, balance)
			}

		case StrategyCharge:
			if batt.AvailableCapacity() <= 0 {
				disqualified = true
				break
			}
			battDiffKWh := batt.UpdateLevel(batt.MaxChargeRate)
			buyKwh := max(0.0, battDiffKWh-balance)
			if buyKwh <= 0 {
				disqualified = true
				break
			}
			totCost += input.BuyPrice(price, buyKwh)
			totCost += batt.DegradationCost * math.Abs(battDiffKWh)

		case StrategyDischarge:
			if batt.RemainingCapacity() <= 0 {
				disqualified = true
				break
			}
			battDiffKWh := batt.UpdateLevel(-batt.MaxDischargeRate)
			sellKwh := max(0.0, balance-battDiffKWh)
			if sellKwh <= 0 {
				disqualified = true
				break
			}
			totCost -= input.SellPrice(price, sellKwh)
			totCost += batt.DegradationCost * math.Abs(battDiffKWh)
		}

		if disqualified {
			break // No need to continue if disqualified
		}
	}

	if disqualified {
		// Disqualified permutations are given infinite cost
		totCost = math.Inf(1)
	}

	return totCost, batt.CurrentLevel
}
