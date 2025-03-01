package types

import (
	"context"

	"github.com/angas/solarplant-go/hours"
)

type EnergyPrice struct {
	Hour  hours.DateHour
	Price float64 // Price in SEK per kWh excluding VAT
}

type EnergyPriceProvider interface {
	GetEnergyPrices(ctx context.Context) ([]EnergyPrice, error)
}
