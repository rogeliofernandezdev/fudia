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
- Todo catálogo de negocio se obtiene del API.
- Cada selector CSS se declara una sola vez por contexto. No reescribir una
  clase en bloques posteriores ni depender de la cascada para corregirla;
  estados y breakpoints deben usar variantes explícitas.
- Los modales de solo lectura (detalle, vista previa) no llevan botón
  "Cerrar" en el footer: el botón X del header es el único cierre.

<!-- BEGIN:nextjs-agent-rules -->

# This is NOT the Next.js you know

This version has breaking changes — APIs, conventions, and file structure may all differ from your training data. Read the relevant guide in `node_modules/next/dist/docs/` (resolved from this file's directory; in monorepos the `next` package may not be visible from the repo root) before writing any code. Heed deprecation notices.

This block is written and re-added by `next dev` — verify at `node_modules/next/dist/server/lib/generate-agent-files.js`. Removing it from a diff only re-creates the uncommitted change; committing it with your work keeps the tree clean.

<!-- END:nextjs-agent-rules -->
