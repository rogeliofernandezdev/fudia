package domain

import (
	"math"
	"testing"
)

func TestTipAmountUsesCurrencyPrecision(t *testing.T) {
	for _, c := range []struct {
		kind        string
		value, base float64
		decimals    int
		want        float64
		ok          bool
	}{
		{"percentage", 10, 59.97, 2, 6, true},
		{"percentage", 10, 59.97, 0, 6, true},
		{"fixed", 5, 100, 2, 5, true},
		{"fixed", 5, 100, 0, 5, true},
		{"fixed", 5.03, 100, 0, 0, false},
		{"fixed", 5.001, 100, 2, 0, false},
		{"percentage", 101, 100, 2, 0, false},
		{"fixed", math.Inf(1), 100, 2, 0, false},
		{"fixed", math.NaN(), 100, 2, 0, false},
		{"percentage", 0, 100, 2, 0, false},
		{"fixed", 5, 0, 2, 0, false},
		{"unknown", 5, 100, 2, 0, false},
	} {
		amount, ok := TipAmount(TipInput{Kind: c.kind, Value: c.value}, c.base, c.decimals)
		if ok != c.ok || (ok && math.Abs(amount-c.want) > .00001) {
			t.Errorf("%+v: got %v,%v", c, amount, ok)
		}
	}
}
