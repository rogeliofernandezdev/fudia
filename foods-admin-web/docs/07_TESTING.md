# Pruebas

Probar validaciones, permisos, filtros, errores y vacíos. Validar 390 px, escritorio
y accesibilidad. Puertas: lint, typecheck, tests y build.

`tests/ux-refinement.test.mjs` verifica validación y conservación del borrador de
menús, cupos/recargos/fechas, catálogo paginado, transición directa de Cocina,
contraste del botón exitoso, geometría primaria única y uso de `Dialog` en todas
las ventanas. Incluye Tab/Shift+Tab, restauración de foco, ventanas superpuestas
y cierre exclusivamente explícito (autocierre solo en éxito).
