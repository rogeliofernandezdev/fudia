# Módulos

## Menú y producción

- `productos`: Menú y productos.
- `combos`: Combos y menús compuestos. Los modificadores de producto son una
  capacidad distinta y no se presentan bajo este nombre.
- `recetas`: Recetas y producción.

Cada módulo puede contener domain, application, infrastructure y transport/http.
Expone contratos mínimos y no importa adaptadores internos de otro módulo.

Las primeras verticales serán Organizations, Menu, Service, Orders, Kitchen y Billing.
