---
version: 1
slug: "ders-presentation-order-detail-dialog-tsx-252d9682"
primary_target: "foods-admin-web/src/modules/operations/orders/presentation/order-detail-dialog.tsx"
related_targets: ["foods-admin-web/src/design-system/modal.tsx","foods-admin-web/src/design-system/styles/modal.css"]
---

# Modal operativo compartido

Alcance: estructura visual reutilizable de modales; detalle de Pedidos y Salón como primer consumidor del resumen persistente. Modo Operar. Sin cambios en permisos, estados, endpoints, textos de negocio ni comanda en desarrollo.

## Direction contract

THESIS: consultar productos, responsables y cuenta sin buscar el saldo al final de una lista. Se abandona la pila de tarjetas altas del detalle anterior.

OWN-WORLD: Manrope, superficies claras, cabecera compacta índigo de ancho completo, sin franja decorativa ni degradado. Estados tenues según la matriz obligatoria de foods-admin-web/docs/02_DESIGN_SYSTEM.md: cuenta abierta celeste, confirmado índigo, pagada verde y finalizada gris. Controles existentes de Fudia. Sin nueva identidad.

STORY: identificar mesa y estados; recorrer productos, cantidades y notas; comprobar cuenta y ejecutar únicamente acciones autorizadas.

FIRST VIEWPORT: cabecera breve con icono, título y X. Primero responsable y luego fecha en el cuerpo, antes de los productos. Una sola columna en todos los tamaños: productos y notas desplazables; resumen inferior persistente neutro cloud-50 con estado de cuenta e importes, antes de las acciones. Preparación y cuenta consumen sus mapeos centrales y la matriz aprobada, sin reinterpretar colores. Sin panel lateral de cuenta.

FORM: composición local definida directamente, code-led, sin concept-seed por tratarse de un componente acotado del sistema existente. Firma funcional: la cuenta conserva su posición al recorrer una comanda extensa. Formularios normales comparten cabecera y pie, sin columna de cuenta innecesaria.

FINISH: pruebas de permisos y estados, capturas ilustrativas de escritorio/móvil y documentación en la autoridad local `foods-admin-web/docs/02_DESIGN_SYSTEM.md`. Sin crear DESIGN.md ni convertir una decisión rechazada en estándar.

Salida concreta: capturas del componente real con datos ilustrativos, escritorio y 390px; inspección acotada y documentación en docs/02_DESIGN_SYSTEM.md y docs/04_COMPONENTS.md. No se crea ni sustituye una identidad global en DESIGN.md.
