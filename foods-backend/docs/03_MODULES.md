# Módulos

## Menú y producción

- `productos`: Menú y productos.
- `combos`: Combos y menús compuestos. Los modificadores de producto son una
  capacidad distinta y no se presentan bajo este nombre.
- `recetas`: Recetas y producción.

`combos` es parte de la operación esencial y se incluye desde Emprende, junto
con `whatsapp_bot` (Fudia Concierge). `inventario`, `compras`, `kardex`,
`recetas`, `reservas` y `costos` se incluyen desde Impulso. Los planes son
acumulativos; la matriz oficial está en `docs/01_PRODUCT_SCOPE.md` (raíz). La
habilitación no modifica los permisos asignados a los roles.

`whatsapp_bot` es un módulo sin pantalla de empresa (2026-11-22): habilita los
pedidos desde el QR, pero no es una casilla del menú de roles. La migración
000140 la retira de `roles.menu_access`; las acciones de tarifas de envío
(`organizations.read` / `organizations.manage`) se muestran en la pantalla
`delivery`.

`costos` tampoco tiene casilla propia desde el 2026-11-22: en el catálogo se
llama «Gastos» (categoría Operación, junto a Caja y turnos), sus gastos son la
pestaña «Gastos» de Caja y las acciones `expenses.*` se muestran en la pantalla
`caja`. La migración 000141 retira `costos` de `roles.menu_access` y agrega
`caja` solo a los roles que ya tenían `cash.read` (no otorga permisos nuevos).

Delivery no es un módulo de la empresa desde el 2026-11-23: lo administra
Fudia. `delivery`, `delivery_apps` y `repartidores` salen del catálogo de
Módulos; Delivery Fudia se activa por empresa en Plataforma › Empresa y la
opción «Delivery» del menú es solo de Plataforma (migración 000143 la retira de
`roles.menu_access`). Detalle en `15_DELIVERY_GEOGRAPHY.md`.

Gasto en efectivo y egreso de caja son un solo registro (migración 000142). Con
`costos` activo, `POST /v1/admin/cash-shifts/{id}/movements` de tipo `expense`
exige `categoryId` y, en una sola sentencia, crea el gasto (medio de efectivo
de la empresa, fecha de negocio del turno, `expenses.cash_shift_id`) y su
movimiento (`source_type='expense'`, `source_id` = gasto). Las categorías para el
turno salen de `GET /v1/admin/cash-expense-categories` (`cash.manage`).
`createExpense` rechaza medios que mueven efectivo (`cash_expense_requires_shift`).
Anular un gasto del turno bloquea el turno abierto y agrega un ingreso
`expense_void`; con el turno cerrado responde 409 `expense_cash_shift_closed`.
Los egresos manuales anteriores no se migran: son historial.
La disponibilidad efectiva cruza `organization_modules.active`, módulos del
plan de la suscripción y estado de desarrollo. Roles y catálogos de empresa
solo presentan ese alcance; Plataforma conserva el catálogo comercial completo.

Cada módulo puede contener domain, application, infrastructure y transport/http.
Expone contratos mínimos y no importa adaptadores internos de otro módulo.

Las primeras verticales serán Organizations, Menu, Service, Orders, Kitchen y Billing.
