package domain

import "math"

func PreparedQuantityUsage(items []PreparedOrderItem) map[string]float64 {
	usage := map[string]float64{}
	for _, item := range items {
		if item.ItemType == "product" && item.ProductID != nil {
			usage[*item.ProductID] += item.Qty
			continue
		}
		if item.ItemType == "combo" {
			for _, selection := range item.Selections {
				usage[selection.ProductID] += item.Qty
			}
		}
	}
	return usage
}
func PreparedRecipeProductUsage(items []PreparedOrderItem) map[string]float64 {
	usage := map[string]float64{}
	for _, item := range items {
		if item.ProductID != nil {
			usage[*item.ProductID] += item.Qty
		}
		if item.ItemType == "combo" {
			for _, sel := range item.Selections {
				usage[sel.ProductID] += item.Qty
			}
		}
	}
	return usage
}
func QuantityUsageDelta(previous, next map[string]float64) map[string]float64 {
	ids := map[string]bool{}
	for id := range previous {
		ids[id] = true
	}
	for id := range next {
		ids[id] = true
	}
	delta := map[string]float64{}
	for id := range ids {
		change := next[id] - previous[id]
		if math.Abs(change) > 0.000001 {
			delta[id] = change
		}
	}
	return delta
}
