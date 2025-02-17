package optimize

// Recursive function to generate all permutations of strategies
// for a given number of hours.
func permute(hours int, current []Strategy) [][]Strategy {
	if hours == 0 {
		return [][]Strategy{append([]Strategy{}, current...)}
	}

	var result [][]Strategy
	for s := 0; s < int(strategyCount); s++ {
		result = append(result, permute(hours-1, append(current, Strategy(s)))...)
	}

	return result
}
