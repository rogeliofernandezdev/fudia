# AGENTS — Administración web

Frontend administrativo oficial. Aplicar el `AGENTS.md` raíz.

Regla obligatoria: leer el `README.md` de este proyecto y todos los documentos
de `docs/` antes de implementar cambios relevantes.

La autoridad visual exclusiva de este proyecto es
`docs/02_DESIGN_SYSTEM.md`. No importar reglas visuales desde otro proyecto.

## Stack

- Next.js App Router, TypeScript estricto y Tailwind CSS.
- TanStack Query, React Hook Form y Zod.
- Monolito modular por vertical slices.

## Reglas

- `src/app` se limita a routing, layouts y BFF; no aloja features ni CSS de negocio.
- Toda vertical nueva vive en `src/modules/<vertical>` y expone una API pública desde `index.ts`.
- `src/components` no se usa como contenedor genérico de features.
- CSS global/base vive en `src/styles`; CSS de negocio pertenece al módulo que lo consume.

- Navegación: Control, Venta, Cocina, Abastecimiento y Configuración.
- Separar KPI ejecutivos de señales operativas accionables.
- La presentación no construye URLs ni interpreta DTO crudos.
- Inputs y tablas provienen del design system.
- Los estados usan `Status` y la matriz obligatoria «Colores de operación:
  pedido, preparación y cuenta» de `docs/02_DESIGN_SYSTEM.md`. Fondos tenues,
  texto oscuro y contraste mínimo 4.5:1; sin colores locales ni variantes sólidas.
  Cuenta abierta (`sky`) no comparte tono con Confirmado (`blue`), ni Pagada
  (`green`) con Finalizada (`gray`). Reutilizar `orderStatus`, `serviceStatusTone`
  y `accountTone`; cualquier cambio semántico requiere aprobación y actualizar
  juntos documentación, tokens, mapeos y pruebas.
- Todo catálogo de negocio se obtiene del API.
- Cada selector CSS se declara una sola vez por contexto. No reescribir una
  clase en bloques posteriores ni depender de la cascada para corregirla;
  estados y breakpoints deben usar variantes explícitas.
- Modales de contenido: `ModalHeader` y `ModalFooter` compartidos; header solo
  con icono, título y X, nunca información adicional debajo del título. Acciones
  contiguas a la derecha, Cancelar antes de Guardar, separación de 10 px; sin
  márgenes automáticos que las distribuyan entre extremos. Estados reales en el
  resumen o footer, sin inventar estados para formularios. Se exceptúan
  `ConfirmDialog` y respuestas success/error/warning de `FeedbackProvider`.
- Los modales de solo lectura (detalle, vista previa) no llevan botón
  "Cerrar" en el footer: el botón X del header es el único cierre.
- Jerarquía obligatoria de modales: etiquetas de datos 600, valores 400 y
  títulos de sección 700. Usar los roles de `modal.css`; no invertir pesos
  por pantalla. Totales principales, estados, productos y acciones conservan
  su énfasis semántico. La etiqueta identifica el dato: «Mesa», «Cliente» o
  «Venta», nunca «Cliente / Mesa» en un detalle. No afecta los avisos ni el PDF.

<!-- BEGIN:nextjs-agent-rules -->

# This is NOT the Next.js you know

This version has breaking changes — APIs, conventions, and file structure may all differ from your training data. Read the relevant guide in `node_modules/next/dist/docs/` (resolved from this file's directory; in monorepos the `next` package may not be visible from the repo root) before writing any code. Heed deprecation notices.

This block is written and re-added by `next dev` — verify at `node_modules/next/dist/server/lib/generate-agent-files.js`. Removing it from a diff only re-creates the uncommitted change; committing it with your work keeps the tree clean.

<!-- END:nextjs-agent-rules -->
