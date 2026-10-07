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

## Alta manual desde Pedidos

«Nuevo pedido» abre un formulario breve de datos, con Delivery inicialmente seleccionado.
Los tipos seleccionables (Delivery, Recojo y Mostrador) toman sus etiquetas de
`channelOptions` del API. Salón conserva su toma de comanda por mesa; WhatsApp
conserva el ingreso por Concierge y no se ofrece como origen manual.
Delivery exige cliente, teléfono válido y dirección; referencia, notas y costo
de envío completan la entrega. «Continuar» valida estos datos y abre `ComandaView`,
la misma toma de pedidos utilizada por Salón: carta, cantidades, elecciones,
notas y resumen. Delivery muestra cliente/dirección en lugar de mesa e incluye
el envío en el total. Recojo y Mostrador tampoco solicitan mesa. «Editar datos»
vuelve al formulario conservando productos, elecciones y notas. Avanzar o volver
no realiza peticiones de creación ni genera pedidos parciales. El catálogo
operativo es paginado y presenta la disponibilidad real del local.

La acción explícita «Registrar y enviar a cocina» hace POST al endpoint existente
`/v1/admin/orders` con `sendToKitchen=true`; no guarda al pulsar Enter desde un
campo. El servidor deriva empresa/local, recalcula precios y valida disponibilidad.
La petición bloquea campos, cierre y repetición; un fallo conserva el borrador.
X/Cancelar confirma el descarte si hubo cambios. El éxito cierra el formulario,
selecciona el canal creado, limpia filtros e invalida Pedidos, Cocina, POS,
Dashboard y disponibilidad. No se registran pagos desde este formulario: el cobro
sigue en POS. Delivery continúa con Listo → En camino → Entregado.

## Entrega en Salón

Salón y Pedidos ofrecen «Confirmar entrega» cuando Cocina termina una
comanda. El estado `entregado` conserva la mesa ocupada y el enlace a cobro
mientras la cuenta está abierta (`completedAt` ausente). La mesa se libera
automáticamente al cumplir entrega y pago completo, independientemente del
orden de estas acciones; no existe un botón adicional «Liberar mesa».
Los pagos parciales y los pedidos aún no entregados conservan la ocupación.
El backend registra el cierre en la misma transacción del último cobro o de
la entrega. Las mutaciones refrescan Salón, Pedidos y POS para mantener
sincronizados entrega, saldo y ocupación.
Los detalles abiertos de Salón y Pedidos se refrescan cada 10 segundos hasta
el cierre, para reflejar también los cobros registrados por otro usuario.
