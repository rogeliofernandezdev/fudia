package domain

type DashboardOperations struct {
	PendingBalance      string  `json:"pendingBalance"`
	UnpaidOrders        int     `json:"unpaidOrders"`
	PartialOrders       int     `json:"partialOrders"`
	TablesTotal         int     `json:"tablesTotal"`
	TablesOccupied      int     `json:"tablesOccupied"`
	KitchenConfirmed    int     `json:"kitchenConfirmed"`
	KitchenPreparing    int     `json:"kitchenPreparing"`
	ReadyOrders         int     `json:"readyOrders"`
	DeliveryPending     int     `json:"deliveryPending"`
	DeliveryInTransit   int     `json:"deliveryInTransit"`
	ActiveCashRegisters int     `json:"activeCashRegisters"`
	OpenCashShifts      int     `json:"openCashShifts"`
	CashBalance         *string `json:"cashBalance"`
	SoldOutProducts     int     `json:"soldOutProducts"`
	// Mesas de servicio ocupadas cuya cuenta ya se pidió (Por cobrar en Salón).
	TablesBillRequested int `json:"tablesBillRequested"`
	// Minutos desde el primer pedido de salón aún activo de cada mesa ocupada; nil sin mesas ocupadas.
	TableMinutesAverage *int            `json:"tableMinutesAverage"`
	OldestTable         *DashboardTable `json:"oldestTable"`
}
type DashboardTable struct {
	Name    string `json:"name"`
	Minutes int    `json:"minutes"`
}
type DashboardPeriod struct {
	Key  string
	Days int
}
type DashboardPeriodView struct {
	Key  string `json:"key"`
	Days int    `json:"days"`
	From string `json:"from"`
	To   string `json:"to"`
}
type DashboardPreviousView struct {
	SalesNet      string `json:"salesNet"`
	PaidOrders    int    `json:"paidOrders"`
	AverageTicket string `json:"averageTicket"`
	From          string `json:"from"`
	To            string `json:"to"`
}
type DashboardTrendPoint struct {
	Key      string `json:"key"`
	Current  string `json:"current"`
	Previous string `json:"previous"`
	Orders   int    `json:"orders"`
}
type DashboardTrend struct {
	Granularity string `json:"granularity"`
	// Now es el tramo en curso (hora HH u fecha de hoy): los tramos posteriores aún no ocurren.
	Now    string                `json:"now"`
	Points []DashboardTrendPoint `json:"points"`
}
type DashboardBreakdown struct {
	Value string `json:"value"`
	Label string `json:"label"`
	Total string `json:"total"`
	Count int    `json:"count"`
	// Solo en salesByChannel: pedidos que se terminaron de pagar en el periodo y su promedio.
	Orders        *int    `json:"orders,omitempty"`
	AverageTicket *string `json:"averageTicket,omitempty"`
	// Solo en salesByPaymentMethod: identidad de color igual que en Cobros. El efectivo afecta caja;
	// los demás medios llevan su posición entre los digitales del catálogo operativo de ventas.
	AffectsCash  *bool `json:"affectsCash,omitempty"`
	DigitalIndex *int  `json:"digitalIndex,omitempty"`
}
type DashboardCategorySale struct {
	Name    string `json:"name"`
	Qty     string `json:"qty"`
	Revenue string `json:"revenue"`
}
type DashboardInsights struct {
	Period               DashboardPeriodView     `json:"period"`
	Previous             DashboardPreviousView   `json:"previous"`
	Trend                DashboardTrend          `json:"trend"`
	SalesByChannel       []DashboardBreakdown    `json:"salesByChannel"`
	SalesByPaymentMethod []DashboardBreakdown    `json:"salesByPaymentMethod"`
	SalesByCategory      []DashboardCategorySale `json:"salesByCategory"`
}

func DashboardPeriodByKey(key string) (DashboardPeriod, bool) {
	switch key {
	case "", "today":
		return DashboardPeriod{Key: "today", Days: 1}, true
	case "7d":
		return DashboardPeriod{Key: "7d", Days: 7}, true
	case "30d":
		return DashboardPeriod{Key: "30d", Days: 30}, true
	}
	return DashboardPeriod{}, false
}
func (p DashboardPeriod) PreviousOffsets() (int, int) {
	if p.Days == 1 {
		return 7, 7
	}
	return 2*p.Days - 1, p.Days
}

type Dashboard struct {
	SalesNet           string              `json:"salesNet"`
	PaidOrders         int                 `json:"paidOrders"`
	AverageTicket      string              `json:"averageTicket"`
	OpenOrders         int                 `json:"openOrders"`
	CriticalStock      int                 `json:"criticalStock"`
	PurchasesToApprove int                 `json:"purchasesToApprove"`
	ReservationsToday  int                 `json:"reservationsToday"`
	KitchenPending     int                 `json:"kitchenPending"`
	HourlySales        []map[string]any    `json:"hourlySales"`
	TopProducts        []map[string]any    `json:"topProducts"`
	Operations         DashboardOperations `json:"operations"`
	BusinessDate       string              `json:"businessDate"`
	DashboardInsights
}
