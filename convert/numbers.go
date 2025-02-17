package convert

import (
	"math"
)

func TwoDecimals(number float64) float64 {
	return RoundFloat64(number, 2)
}

func RoundFloat64(number float64, decimals int) float64 {
	return math.Round(number*math.Pow10(int(decimals))) / math.Pow10(int(decimals))
}

func MJ2Kwh(mj float64) float64 {
	return mj * 0.0002777778 / 1e6
}

func OctasToPercentage(octas float64) float64 {
	return math.Round((octas / 8) * 100)
}

func DegToRad(deg float64) float64 {
	return deg * math.Pi / 180.0
}
