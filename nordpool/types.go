package nordpool

import "time"

type nordpoolData struct {
	DeliveryDateCET      string                `json:"deliveryDateCET"`
	Version              int                   `json:"version"`
	UpdatedAt            string                `json:"updatedAt"`
	DeliveryAreas        []string              `json:"deliveryAreas"`
	Market               string                `json:"market"`
	MultiAreaEntries     []multiAreaEntry      `json:"multiAreaEntries"`
	BlockPriceAggregates []blockPriceAggregate `json:"blockPriceAggregates"`
	Currency             string                `json:"currency"`
	ExchangeRate         float64               `json:"exchangeRate"`
	AreaStates           []areaState           `json:"areaStates"`
	AreaAverages         []areaAverage         `json:"areaAverages"`
}

type multiAreaEntry struct {
	DeliveryStart time.Time          `json:"deliveryStart"`
	DeliveryEnd   time.Time          `json:"deliveryEnd"`
	EntryPerArea  map[string]float64 `json:"entryPerArea"`
}

type blockPriceAggregate struct {
	BlockName           string                     `json:"blockName"`
	DeliveryStart       time.Time                  `json:"deliveryStart"`
	DeliveryEnd         time.Time                  `json:"deliveryEnd"`
	AveragePricePerArea map[string]priceStatistics `json:"averagePricePerArea"`
}

type priceStatistics struct {
	Average float64 `json:"average"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
}

type areaState struct {
	State string   `json:"state"`
	Areas []string `json:"areas"`
}

type areaAverage struct {
	AreaCode string   `json:"areaCode"`
	Price    *float64 `json:"price"`
}
