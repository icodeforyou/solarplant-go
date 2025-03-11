package calc

func BuyPrice(kWh, price, energyTax, gridBenefit float64) float64 {
	return kWh * (price + energyTax - gridBenefit)
}

func SellPrice(kWh, price, energyTaxReduction float64) float64 {
	return kWh * (price + energyTaxReduction)
}

func CashFlow(gridImportKWh, gridExportKWh, price, tax, taxReduction, gridBenefit float64) float64 {
	netExp := gridExportKWh - gridImportKWh
	if netExp > 0 {
		return SellPrice(netExp, price, taxReduction)
	} else if netExp < 0 {
		return BuyPrice(-netExp, price, tax, gridBenefit) * -1
	}
	return 0
}
