package ferroamp

import (
	"testing"
)

func TestMovingAverage(t *testing.T) {

	ma1 := NewMovingAverage(1)
	ma1.Add(1)
	got := ma1.Avg()
	want := 1.0
	if got != want {
		t.Errorf("got %f, wanted %f", got, want)
	}

	ma2 := NewMovingAverage(3)
	ma2.Add(1)
	ma2.Add(2)
	ma2.Add(3)
	got = ma2.Avg()
	want = 2.0
	if got != want {
		t.Errorf("got %f, wanted %f", got, want)
	}
	ma2.Add(4)
	got = ma2.Avg()
	want = 3.0
	if got != want {
		t.Errorf("got %f, wanted %f", got, want)
	}
}
