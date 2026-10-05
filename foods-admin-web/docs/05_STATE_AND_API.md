# Estado e integración

TanStack Query administra estado remoto por organización/local. Presentación no
contiene rutas HTTP ni DTO crudos. Toda dependencia muestra skeleton, error y vacío.

Las transiciones de Cocina invalidan comandas, pedidos y el detalle afectado.
La respuesta confirmada se refleja en el panel correspondiente sin modal de
éxito; durante el envío se bloquea la acción y un fallo conserva el aviso de error.

El wizard de Menús y combos valida cada paso antes de continuar y todos los pasos
antes de guardar. Explica el primer error y enfoca el campo afectado. Retroceder
no borra los datos ni guarda registros parciales. X/Cancelar solicitan confirmación
cuando hay cambios sin guardar; la recarga utiliza el aviso nativo del navegador.
No se repite «Paso X de Y» junto a la navegación de pasos.

## Autocompletes remotos

Los autocompletes de catálogos grandes no cargan el catálogo completo. La apertura inicial usa una página pequeña de hasta 10 opciones. Las búsquedas no consultan con menos de 3 caracteres; desde 3 caracteres delegan el filtro al backend y recorren la paginación necesaria para reunir todas las coincidencias. Las opciones ya seleccionadas en formularios multirregistro se excluyen en cliente sin alterar el resultado remoto.

## Performance de consultas en cliente

Las búsquedas de listados no disparan una petición por pulsación. Deben reutilizar
`useDebouncedValue` con una ventana aproximada de 300 ms y mantener el texto
visible separado del valor consultado.

TanStack Query conserva la caché por clave; las mutaciones invalidan únicamente
las familias afectadas. Los diálogos pesados que no forman parte del primer render
se cargan con `next/dynamic` cuando la acción realmente se abre.

El prefetch de datos se incorpora solo en rutas frecuentes y cuando la medición
demuestre que reduce latencia sin duplicar tráfico.

## Estado de productos

La tabla de Carta y productos permite activar un producto inactivo mediante
`PATCH /v1/admin/products/{id}/status`, enviando únicamente `active`. La operación
conserva la ficha comercial y el historial, exige `menu.manage` y refresca los
listados del catálogo, disponibilidad, comandas y selectores de abastecimiento.

## Estado de categorías

Las categorías inactivas ofrecen la acción Activar mediante
`PATCH /v1/admin/categories/{id}/status`. Solo cambia `active`, conserva nombre,
orden, uso y asociaciones, y exige `menu.manage`. La activación refresca
`categories`, `purchase-item-categories`, `order-categories` y `order-catalog`.
La desactivación sigue bloqueada cuando hay productos activos asociados.

## Entrega en Salón

Salón y Pedidos ofrecen «Marcar como entregado» cuando Cocina termina una
comanda. El estado `entregado` conserva la mesa ocupada y el enlace a cobro
mientras la cuenta está abierta (`completedAt` ausente). Tras pagar, «Liberar
mesa» registra el cierre. Las mutaciones refrescan Salón, Pedidos y POS para
mantener sincronizados entrega, saldo y ocupación.
