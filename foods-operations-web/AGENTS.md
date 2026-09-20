# AGENTS — Operaciones web

PWA oficial para POS, salón, cocina y caja. Aplicar el `AGENTS.md` raíz.

Regla obligatoria: leer el `README.md` de este proyecto y todos los documentos
de `docs/` antes de implementar cambios relevantes.

La autoridad visual exclusiva de este proyecto es
`docs/02_DESIGN_SYSTEM.md`. No importar reglas visuales desde otro proyecto.

## Stack

- Next.js App Router, TypeScript estricto y Tailwind CSS.
- TanStack Query, React Hook Form y Zod.
- Vertical slices: `domain`, `application`, `infrastructure`, `presentation`.

## Reglas operativas

- Priorizar latencia percibida, targets táctiles y continuidad del servicio.
- Validar cada pantalla a 390 px, tablet horizontal y escritorio táctil.
- POS y KDS no comparten una composición visual aunque compartan tokens.
- No confirmar pedidos, pagos o anulaciones solo de forma optimista.
- Mostrar conexión y sincronización sin bloquear acciones locales seguras.
- Cada selector CSS se declara una sola vez por contexto. No reescribir una
  clase en bloques posteriores ni depender de la cascada para corregirla;
  estados y breakpoints deben usar variantes explícitas.

<!-- BEGIN:nextjs-agent-rules -->

# This is NOT the Next.js you know

This version has breaking changes — APIs, conventions, and file structure may all differ from your training data. Read the relevant guide in `node_modules/next/dist/docs/` (resolved from this file's directory; in monorepos the `next` package may not be visible from the repo root) before writing any code. Heed deprecation notices.

This block is written and re-added by `next dev` — verify at `node_modules/next/dist/server/lib/generate-agent-files.js`. Removing it from a diff only re-creates the uncommitted change; committing it with your work keeps the tree clean.

<!-- END:nextjs-agent-rules -->
