package ferroamp

type MovingAverage struct {
	window []float64
	size   int
	sum    float64
	index  int
	full   bool
}

func NewMovingAverage(size int) *MovingAverage {
	return &MovingAverage{
		window: make([]float64, size),
		size:   size,
	}
}

func (ma *MovingAverage) Add(value float64) {
	if ma.full {
		ma.sum -= ma.window[ma.index]
	}

	ma.window[ma.index] = value
	ma.sum += value
	ma.index = (ma.index + 1) % ma.size

	if ma.index == 0 {
		ma.full = true
	}
}

func (ma *MovingAverage) Avg() float64 {
	if ma.full {
		return ma.sum / float64(ma.size)
	}

	return ma.sum / float64(ma.index)
}

func (ma *MovingAverage) Reset() {
	ma.sum = 0
	ma.index = 0
	ma.full = false
	for i := range ma.window {
		ma.window[i] = 0
	}
}
