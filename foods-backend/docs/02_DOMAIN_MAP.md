# Mapa de dominio

Contextos: Identity, Organizations, Menu, Service, Orders, Kitchen, Billing,
EInvoicing, Inventory, Purchasing, Customers, Delivery, Messaging y Reporting.

Orders coordina el pedido; Kitchen administra su preparación; Billing administra
cuenta y pago. Ninguno modifica directamente las tablas del otro.

Menu define qué se vende y con qué modo de control; Inventory es la única
autoridad sobre existencias y sobre la disponibilidad calculada. Orders no
descuenta stock por sí mismo: registra la venta y solicita a Inventory el
movimiento de consumo, que puede no existir cuando el producto es `none` o
`manual`. Un restaurante en modo `simple` opera sin Inventory activo.
