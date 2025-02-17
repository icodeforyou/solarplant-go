package nordpool

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func GetEnergyPrices(area string, page int16, currency string) ([]EnergyPrice, error) {
	url := fmt.Sprintf("%s/marketdata/page/%d?currency=%s", API_URL, page, currency)

	slog.Default().Info("Fetching energy prices from Nordpool...", "url", url)

	req, _ := http.NewRequest("GET", url, nil)
	client := http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error getting NordPool prices: %v", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading NordPool response body: %v", err)
	}

	var nordpool nordpool
	if err := json.Unmarshal(body, &nordpool); err != nil {
		return nil, fmt.Errorf("error unmarshaling NordPool json: %v", err)
	}

	result := make([]EnergyPrice, 0)
	for _, row := range nordpool.Data.Rows {
		for _, column := range row.Columns {
			if column.Name == area {
				result = append(result, EnergyPrice{
					Hour:  dateToTime(row.StartTime),
					Price: priceToFloat(column.Value),
				})
			}
		}
	}

	return result, nil
}

func priceToFloat(str string) float64 {
	noSpace := strings.ReplaceAll(str, " ", "")
	noComma := strings.ReplaceAll(noSpace, ",", ".")
	price, err := strconv.ParseFloat(noComma, 64)
	if err != nil {
		return 0
	}
	return price / 1e3
}

func dateToTime(str string) time.Time {
	date, err := time.Parse("2006-01-02T15:04:05", str)
	if err != nil {
		return time.Time{}
	}
	return date
}
