package optimize

type Strategy int

const (
	StrategyDefault   Strategy = iota // Maximize self-consumption
	StrategyPreserve                  // Preserve battery level
	StrategyCharge                    // Buy power if produced power is not enough
	StrategyDischarge                 // Sell excess power to the grid from the battery
	strategyCount                     // Number of strategies
)

func (s Strategy) String() string {
	switch s {
	case StrategyDefault:
		return "default"
	case StrategyPreserve:
		return "preserve"
	case StrategyCharge:
		return "charge"
	case StrategyDischarge:
		return "discharge"
	default:
		return "unknown"
	}
}

func (s Strategy) IsValid() bool {
	return s >= StrategyDefault && s < strategyCount
}
