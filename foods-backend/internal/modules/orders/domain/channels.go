package domain

var Channels = []map[string]string{{"value": "salon", "label": "Salón"}, {"value": "recojo", "label": "Recojo en local"}, {"value": "delivery", "label": "Delivery"}, {"value": "whatsapp", "label": "WhatsApp"}}

// ServiceChannels son los tipos de atención con que se filtran Pedidos y Cocina.
// WhatsApp es un origen (pedido QR de mesa por Concierge), no un tipo de atención.
var ServiceChannels = []map[string]string{{"value": "salon", "label": "Salón"}, {"value": "recojo", "label": "Recojo en local"}, {"value": "delivery", "label": "Delivery"}}

// ChannelCounter es la venta por mostrador: un pedido independiente, sin mesa
// ni mozo, que se prepara como siempre y se cobra en caja. Se atiende como
// recojo en el local, por eso Pedidos y Cocina lo agrupan en esa pestaña; su
// origen se conserva en Order.Origin. Reutiliza el código histórico
// `mostrador`, que siempre identificó estas ventas.
const ChannelCounter = "mostrador"

// ManualChannels son los tipos que el cajero registra sin mesa desde Pedidos.
var ManualChannels = []map[string]string{{"value": ChannelCounter, "label": "Mostrador"}, {"value": "recojo", "label": "Recojo en local"}, {"value": "delivery", "label": "Delivery"}}

// Origen del pedido: la venta por mostrador se distingue del resto sin
// cambiar su tipo de atención.
const (
	OriginStandard = "standard"
	OriginCounter  = "counter"
)

// Origin deriva el origen del canal guardado.
func Origin(channel string) string {
	if channel == ChannelCounter {
		return OriginCounter
	}
	return OriginStandard
}

// ServiceChannel agrupa el canal guardado en su tipo de atención sin reescribir
// el historial: los pedidos QR por WhatsApp ocupan mesa y se atienden en Salón.
func ServiceChannel(channel string) string {
	if channel == "whatsapp" {
		return "salon"
	}
	return NormalizeChannel(channel)
}

// NormalizeChannel conserva compatibilidad de lectura y filtros: Mostrador se
// atiende como Recojo en local. No reescribe el historial.
func NormalizeChannel(channel string) string {
	if channel == ChannelCounter {
		return "recojo"
	}
	return channel
}

// TableRequirement centraliza la asociación de un pedido con una mesa: solo
// Salón ocupa una mesa tradicional; Mostrador, Recojo y Delivery nunca la usan
// y la venta por mostrador no exige elegir un mostrador físico.
// Devuelve el código y el mensaje del rechazo, o vacío si es válido.
func TableRequirement(channel, tableID, tableServiceType string) (string, string) {
	switch {
	case channel == "salon" && tableID == "":
		return "table_required", "Los pedidos de Salón deben estar asociados a una mesa."
	case channel != "salon" && tableID != "":
		return "table_not_allowed", "Solo los pedidos de Salón pueden ocupar una mesa."
	case tableID != "" && tableServiceType != "" && tableServiceType != "table":
		return "counter_not_table", "El mostrador no abre cuentas de mesa. Registra una venta por mostrador."
	}
	return "", ""
}
