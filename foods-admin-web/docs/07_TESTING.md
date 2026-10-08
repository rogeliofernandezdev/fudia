# Pruebas

`plan-navigation.test.mjs` verifica Menús y combos en Emprende, ocultación de
módulos no incluidos incluso con rol de acceso total, separación de acceso y
lectura, catálogo de Plataforma y recarga de roles/permisos al cambiar el plan.

Probar validaciones, permisos, filtros, errores y vacíos. Validar 390 px, escritorio
y accesibilidad. Puertas: lint, typecheck, tests y build.

`pagination.test.mjs` protege la primitiva global en variantes completa, simple
y compacta: rango/total, límites de navegación, etiquetas e iconos, página activa
y colores compartidos. Impide volver a introducir paginadores antiguos, estilos
genéricos de pies de gestión o parches `!important` que sobrescriban la primitiva.

`tests/ux-refinement.test.mjs` verifica validación y conservación del borrador de
menús, cupos/recargos/fechas, catálogo paginado, transición directa de Cocina,
contraste del botón exitoso, geometría primaria única y uso de `Dialog` en todas
las ventanas. Incluye Tab/Shift+Tab, restauración de foco, ventanas superpuestas
y cierre exclusivamente explícito (autocierre solo en éxito).

El contrato de servicio separa preparación y cuenta. Probar destino remoto del producto, append sobre la misma cuenta Abierta sin líneas anteriores, botones por mozo, entrega completa antes de cierre/cobro, estados operativos sin modal exitoso y ajuste responsive desde 390 px. Los fixtures cobrables incluyen `billClosedAt`, estado Entregado y todos los productos entregados; Listo no equivale a Entregado. Rechazar cierre/cobro con Cocina, Barra o Entrega directa pendientes, incluso con estado agregado obsoleto; después del cierre no permitir consumo adicional ni con pago parcial.

Salón y Pedidos renderizan los controles reales para comprobar una única acción
«Confirmar entrega» en el pie, sin botones por producto. Un pedido mixto de Cocina,
Barra y Entrega directa debe tener todos sus productos pendientes Listos; un estado
agregado Listo no permite saltar productos en preparación. Probar mozo asignado,
lectura sin permisos, bloqueo durante envío, ronda adicional con entregas previas
y transición a «Cerrar cuenta» después de servir todo. La entrega envía una sola
petición de estado, refresca las vistas afectadas y conserva el modal de error,
sin modal exitoso.

`order-delivery.test.mjs` verifica que ambos detalles (Salón/Pedidos) compartan
la cancelación de Entrega directa sin servir, oculten la acción ante preparación,
entrega, pago o finalización, respeten al mozo asignado y la bloqueen mientras
se guarda. Las comandas históricas Listas no ofrecen cancelación.
