package domain

// PaymentReady centraliza cuándo un pedido admite cobro. Salón exige la cuenta
// cerrada (la entrega se verifica aparte). La venta por mostrador se cobra en
// cualquier momento: preparación y pago son procesos independientes. Recojo y
// delivery se cobran cuando el pedido está listo, en camino o entregado.
func PaymentReady(channel, status string, completed, billClosed bool) bool {
	if completed || status == "cancelado" {
		return false
	}
	switch channel {
	case "salon":
		return billClosed
	case "mostrador":
		return true
	}
	return status == "listo" || status == "en_camino" || status == "entregado"
}
