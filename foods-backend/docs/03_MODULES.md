# Módulos

## Menú y producción

- `productos`: Menú y productos.
- `combos`: Combos y menús compuestos. Los modificadores de producto son una
  capacidad distinta y no se presentan bajo este nombre.
- `recetas`: Recetas y producción.

`combos` es parte de la operación esencial y se incluye desde Emprende.
Emprende también incluye `inventario` y `compras` para registrar mercadería,
reponer existencias y recibir compras; no incluye Recetas ni el acceso separado
de Kárdex. La habilitación no modifica los permisos asignados a los roles.
La disponibilidad efectiva cruza `organization_modules.active`, módulos del
plan de la suscripción y estado de desarrollo. Roles y catálogos de empresa
solo presentan ese alcance; Plataforma conserva el catálogo comercial completo.

Cada módulo puede contener domain, application, infrastructure y transport/http.
Expone contratos mínimos y no importa adaptadores internos de otro módulo.

Las primeras verticales serán Organizations, Menu, Service, Orders, Kitchen y Billing.
