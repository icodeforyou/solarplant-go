package chartjs

import (
	"fmt"
	"math"
)

const NoOfHours = 24
const ColorYellow = "#ffc107d4"
const ColorRed = "#f44336d4"

func NewChart(title string) Chart {
	labels := make([]string, NoOfHours)
	for i := 0; i < NoOfHours; i++ {
		label := fmt.Sprintf("%02d:00", i)
		labels[i] = label
	}

	chart := Chart{
		Type: "line",
		Data: ChartData{
			Labels: labels,
			Datasets: []ChartDataset{
				{
					Data:        make([]*float64, NoOfHours),
					BorderWidth: 1,
					Tension:     0.4,
					Fill:        true,
					BorderColor: ColorYellow,
					YAxisID:     "YAxis1",
				},
				{
					Data:        make([]*float64, NoOfHours),
					BorderWidth: 1,
					Tension:     0.4,
					Fill:        true,
					BorderColor: ColorRed,
					YAxisID:     "YAxis2",
				},
			},
		},
		Options: ChartOptions{
			Responsive: true,
			Plugins: ChartPlugins{
				Legend: ChartLegend{Display: false},
				Title:  ChartTitle{Display: false},
			},
			Scales: map[string]ChartScale{
				"YAxis1": {
					Type:     "linear",
					Display:  true,
					Position: "left",
					Title:    ChartScaleTitle{Display: true, Text: "", Color: ColorYellow}},
				"YAxis2": {
					Type:     "linear",
					Display:  true,
					Position: "right",
					Title:    ChartScaleTitle{Display: true, Text: "", Color: ColorRed}},
			},
		},
	}

	if title != "" {
		chart.Options.Plugins.Title = ChartTitle{Display: true, Text: title}
	}

	return chart
}

func (cs ChartScale) WithTitle(title string) ChartScale {
	cs.Title.Text = title
	return cs
}

func (cs ChartScale) WithMinAndMax(min, max float64) ChartScale {
	cs.Min = &min
	cs.Max = &max
	return cs
}

func FixedFloat64(num float64, precision int) *float64 {
	p := math.Pow(10, float64(precision))
	rounded := math.Round(num * p)
	result := rounded / p
	return &result
}
