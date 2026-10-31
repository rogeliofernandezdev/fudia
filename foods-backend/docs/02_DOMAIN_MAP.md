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


Los combos consumen las existencias de sus componentes: Orders agrega las
cantidades de cada plato suelto y de todas las selecciones de combos antes de
solicitar el consumo transaccional a Inventory. Menu informa `remaining` por
componente y `quotaRemaining` por alternativa del menú, local y día operativo.
El cupo del menú es un límite adicional, no crea existencias independientes.
Una alternativa agotada no puede añadirse a un combo nuevo ni venderse; un
cupo explícito debe ser mayor que cero. `none` no limita cantidades mientras
no se marque agotado. Al editar, las reservas del propio pedido se reutilizan
una sola vez. El backend revalida existencias y cupos al guardar el pedido.
