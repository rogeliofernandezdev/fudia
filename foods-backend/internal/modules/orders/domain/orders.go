package domain

type OrderItemSelection struct {
	GroupID   string `json:"groupId"`
	GroupName string `json:"groupName"`
	ProductID string `json:"productId"`
	Name      string `json:"name"`
	Surcharge string `json:"surcharge"`
}
type OrderItem struct {
	ID         string               `json:"id"`
	ProductID  string               `json:"productId"`
	Name       string               `json:"name"`
	Qty        string               `json:"qty"`
	UnitPrice  string               `json:"unitPrice"`
	Note       string               `json:"note"`
	ItemType   string               `json:"itemType"`
	Selections []OrderItemSelection `json:"selections,omitempty"`
	Modifiers  []OrderItemModifier  `json:"modifiers,omitempty"`
}
type Order struct {
	ID            string             `json:"id"`
	Code          string             `json:"code"`
	Channel       string             `json:"channel"`
	// Origin es "counter" para la venta por mostrador y "standard" en otro caso.
	Origin        string             `json:"origin"`
	Status        string             `json:"status"`
	CompletedAt   string             `json:"completedAt,omitempty"`
	BillClosedAt  string             `json:"billClosedAt,omitempty"`
	AccountState  string             `json:"accountState"`
	ServiceItems  []OrderServiceItem `json:"serviceItems,omitempty"`
	CustomerID    string             `json:"customerId"`
	CustomerName  string             `json:"customerName"`
	CustomerPhone string             `json:"customerPhone"`
	Address       string             `json:"address"`
	Reference     string             `json:"reference"`
	// Punto de entrega fijado por WhatsApp (Concierge); nil si no hay pin.
	DeliveryLatitude   *float64    `json:"deliveryLatitude"`
	DeliveryLongitude  *float64    `json:"deliveryLongitude"`
	DeliveryDistanceKm *float64    `json:"deliveryDistanceKm"`
	TableID            string      `json:"tableId"`
	TableName          string      `json:"tableName"`
	WaiterID           string      `json:"waiterId"`
	WaiterName         string      `json:"waiterName"`
	CollectedByNames   []string    `json:"collectedByNames"`
	Notes              string      `json:"notes"`
	Subtotal           string      `json:"subtotal"`
	DeliveryFee        string      `json:"deliveryFee"`
	Total              string      `json:"total"`
	CreatedAt          string      `json:"createdAt"`
	UpdatedAt          string      `json:"updatedAt"`
	ItemCount          int         `json:"itemCount"`
	Items              []OrderItem `json:"items,omitempty"`
	PaidAmount         string      `json:"paidAmount,omitempty"`
	RemainingAmount    string      `json:"remainingAmount,omitempty"`
	PaymentStatus      string      `json:"paymentStatus,omitempty"`
}
type OrderItemSelectionInput struct {
	GroupID   string `json:"groupId"`
	ProductID string `json:"productId"`
}
type OrderItemInput struct {
	ID         string                    `json:"id"`
	ProductID  string                    `json:"productId"`
	Name       string                    `json:"name"`
	Qty        float64                   `json:"qty"`
	UnitPrice  float64                   `json:"unitPrice"`
	Note       string                    `json:"note"`
	Selections []OrderItemSelectionInput `json:"selections"`
	Modifiers  []OrderItemModifierInput  `json:"modifiers"`
	Reprice    bool                      `json:"reprice"`
}
type PreparedOrderSelection struct {
	GroupID   string
	GroupName string
	ProductID string
	Name      string
	Surcharge float64
}
type ComboOptionUsageKey struct {
	ComboProductID string
	GroupName      string
	ProductID      string
}
type PreparedOrderItem struct {
	ProductID  *string
	Name       string
	Qty        float64
	UnitPrice  float64
	Note       string
	ItemType   string
	Selections []PreparedOrderSelection
	Modifiers  []PreparedOrderModifier
}
type OrderInput struct {
	Channel       string           `json:"channel"`
	CustomerID    string           `json:"customerId"`
	CustomerName  string           `json:"customerName"`
	CustomerPhone string           `json:"customerPhone"`
	Address       string           `json:"address"`
	Reference     string           `json:"reference"`
	TableID       string           `json:"tableId"`
	Notes         string           `json:"notes"`
	DeliveryFee   float64          `json:"deliveryFee"`
	Items         []OrderItemInput `json:"items"`
	SendToKitchen bool             `json:"sendToKitchen"`
	// Punto de entrega fijado en el mapa; solo delivery. Ambos o ninguno.
	DeliveryLatitude  *float64 `json:"deliveryLatitude"`
	DeliveryLongitude *float64 `json:"deliveryLongitude"`
	// Calculada por el servidor desde el local; nunca se lee del cliente.
	DeliveryDistanceKm *float64 `json:"-"`
}
type OrderUpdateInput struct {
	CustomerName  string           `json:"customerName"`
	CustomerPhone string           `json:"customerPhone"`
	Address       string           `json:"address"`
	Reference     string           `json:"reference"`
	Notes         string           `json:"notes"`
	DeliveryFee   float64          `json:"deliveryFee"`
	Items         []OrderItemInput `json:"items"`
}
type OrderItemModifierInput struct {
	GroupID  string `json:"groupId"`
	OptionID string `json:"optionId"`
}
type OrderItemModifier struct {
	GroupID   string `json:"groupId"`
	GroupName string `json:"groupName"`
	OptionID  string `json:"optionId"`
	Name      string `json:"name"`
	Surcharge string `json:"surcharge"`
}
type PreparedOrderModifier struct {
	GroupID   string
	GroupName string
	OptionID  string
	Name      string
	Surcharge float64
}
type OrderServiceItem struct {
	DestinationLabel string `json:"destinationLabel"`
	StatusLabel      string `json:"statusLabel"`
	ID               string `json:"id"`
	OrderItemID      string `json:"orderItemId"`
	ProductID        string `json:"productId"`
	Name             string `json:"name"`
	Qty              string `json:"qty"`
	Destination      string `json:"destination"`
	Status           string `json:"status"`
}
type KitchenTicket struct {
	Destination   string      `json:"destination"`
	ID            string      `json:"id"`
	OrderID       string      `json:"orderId"`
	RoundNumber   int         `json:"roundNumber"`
	Code          string      `json:"code"`
	Channel       string      `json:"channel"`
	Origin        string      `json:"origin"`
	Status        string      `json:"status"`
	CustomerName  string      `json:"customerName"`
	TableName     string      `json:"tableName"`
	Notes         string      `json:"notes"`
	CreatedAt     string      `json:"createdAt"`
	UpdatedAt     string      `json:"updatedAt"`
	TargetMinutes *int        `json:"targetMinutes"`
	Items         []OrderItem `json:"items"`
}

var Statuses = []map[string]string{{"value": "nuevo", "label": "Nuevo"}, {"value": "confirmado", "label": "Confirmado"}, {"value": "preparando", "label": "Preparando"}, {"value": "listo", "label": "Por entregar"}, {"value": "en_camino", "label": "En camino"}, {"value": "entregado", "label": "Entregado"}, {"value": "cancelado", "label": "Cancelado"}}
var Transitions = map[string][]string{
	"nuevo":      {"confirmado", "cancelado"},
	"confirmado": {"cancelado"},
	"preparando": {},
	"listo":      {"en_camino", "entregado"},
	"en_camino":  {"entregado"},
	"entregado":  {},
	"cancelado":  {},
}

// CanManageOrderService: una mesa (salón o QR de WhatsApp) con responsable solo la
// atiende ese mozo; sin responsable, o fuera de salón, cualquiera con permiso.
func CanManageOrderService(userID, channel, waiterID string) bool {
	return (channel != "salon" && channel != "whatsapp") || waiterID == "" || (userID != "" && userID == waiterID)
}

// SendRecipient es un WhatsApp que pidió lo que se envía a Cocina; Key distingue
// cada ronda (o el pedido sin mesa) para avisar una sola vez.
type SendRecipient struct {
	ConversationID, Phone, Key string
}
func ValidOrderChannel(ch string) bool {
	return ch == "salon" || ch == "mostrador" || ch == "recojo" || ch == "delivery" || ch == "whatsapp"
}
func ValidOrderStatus(st string) bool {
	for _, s := range Statuses {
		if s["value"] == st {
			return true
		}
	}
	return false
}
func EditableOrderStatus(st string) bool {
	return st == "nuevo" || st == "confirmado"
}
func ValidKitchenTransition(current, next string) bool {
	return (current == "confirmado" && next == "preparando") ||
		(current == "preparando" && next == "listo")
}
func NormalizedKitchenRoundStatus(status string) string {
	switch status {
	case "confirmado", "preparando", "listo":
		return status
	default:
		return "confirmado"
	}
}
func PaymentStatus(total, paid float64) string {
	if paid <= 0.00001 {
		return "pending"
	}
	if paid+0.00001 >= total {
		return "paid"
	}
	return "partial"
}

type ComboOption struct {
	ProductID string
	Name      string
	Surcharge float64
	Available bool
}
type ComboGroup struct {
	ID       string
	Name     string
	Required bool
	Min      int
	Max      int
	Options  map[string]ComboOption
}
type ModifierOption struct {
	ID        string
	Name      string
	Surcharge float64
}
// OrderProductSnapshot es la vista de catálogo de un producto al preparar el
// pedido; se lee para todos los productos en una sola consulta.
type OrderProductSnapshot struct {
	IsCombo      bool
	Name         string
	Price        float64
	Available    bool
	HasModifiers bool
}
type ModifierGroup struct {
	ID       string
	Name     string
	Required bool
	Min      int
	Max      int
	Options  map[string]ModifierOption
}

type FloorTable struct {
	OccupiedSince string `json:"occupiedSince,omitempty"`
	ID            string `json:"id"`
	Name          string `json:"name"`
	Zone          string `json:"zone"`
	Seats         int    `json:"seats"`
	Order         *Order `json:"order"`
	PaidText      string `json:"-"`
}

// TableAccountIdleLimit marca una cuenta de mesa como «anterior sin cerrar»: si la
// comanda abierta no recibe rondas nuevas en este tiempo, un QR nuevo no le suma
// pedidos (el cliente pudo irse dejándola abierta) hasta que el personal la cobre,
// la cancele o agregue productos. Dentro de este tiempo, un grupo que escanea el
// mismo QR sigue pidiendo a la misma cuenta.
const TableAccountIdleLimitSeconds = 3 * 60 * 60
