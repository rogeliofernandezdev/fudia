package domain

var Channels = []map[string]string{{"value": "salon", "label": "Salón"}, {"value": "recojo", "label": "Recojo en local"}, {"value": "delivery", "label": "Delivery"}, {"value": "whatsapp", "label": "WhatsApp"}}

// ServiceChannels son los tipos de atención con que se filtran Pedidos y Cocina.
// WhatsApp es un origen (pedido QR de mesa por Concierge), no un tipo de atención.
var ServiceChannels = []map[string]string{{"value": "salon", "label": "Salón"}, {"value": "recojo", "label": "Recojo en local"}, {"value": "delivery", "label": "Delivery"}}

// ServiceChannel agrupa el canal guardado en su tipo de atención sin reescribir
// el historial: los pedidos QR por WhatsApp ocupan mesa y se atienden en Salón.
func ServiceChannel(channel string) string {
	if channel == "whatsapp" {
		return "salon"
	}
	return NormalizeChannel(channel)
}

// NormalizeChannel conserva compatibilidad con pedidos y clientes anteriores.
// Mostrador ya no es una opción distinta de entrega; no reescribe el historial.
func NormalizeChannel(channel string) string {
	if channel == "mostrador" {
		return "recojo"
	}
	return channel
}
