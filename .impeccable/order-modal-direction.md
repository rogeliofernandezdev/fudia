# Modal operativo compartido

Alcance: estructura visual reutilizable de modales; detalle de Pedidos y Salón como primer consumidor del resumen persistente. Modo Operar. Sin cambios en permisos, estados, endpoints, textos de negocio ni comanda en desarrollo.

## Direction contract

THESIS: consultar productos, responsables y cuenta sin buscar el saldo al final de una lista. Se abandona la pila de tarjetas altas del detalle anterior.

OWN-WORLD: Manrope, superficies claras, cabecera compacta índigo de ancho completo, sin franja decorativa ni degradado. Estados tenues según la matriz obligatoria de foods-admin-web/docs/02_DESIGN_SYSTEM.md: cuenta abierta celeste, confirmado índigo, pagada verde y finalizada gris. Controles existentes de Fudia. Sin nueva identidad.

STORY: identificar mesa y estados; recorrer productos, cantidades y notas; comprobar cuenta y ejecutar únicamente acciones autorizadas.

FIRST VIEWPORT: ver `surfaces/ders-presentation-order-detail-dialog-tsx-252d9682.md`, único contrato de esta superficie. Cabecera breve, contexto en el cuerpo, una sola columna y resumen inferior persistente. El panel lateral de cuenta fue rechazado; no es un estándar aprobado.

FORM: composición local definida directamente, code-led, sin concept-seed por tratarse de un componente acotado del sistema existente. Firma funcional: la cuenta conserva su posición al recorrer una comanda extensa. Formularios normales comparten cabecera y pie, sin columna de cuenta innecesaria.

FINISH: pruebas y capturas ilustrativas con revisión acotada. Autoridad local: `foods-admin-web/docs/02_DESIGN_SYSTEM.md`. Sin crear DESIGN.md.

Salida concreta: capturas del componente real con datos ilustrativos, escritorio y 390px; inspección acotada y documentación en docs/02_DESIGN_SYSTEM.md y docs/04_COMPONENTS.md. No se crea ni sustituye una identidad global en DESIGN.md.
