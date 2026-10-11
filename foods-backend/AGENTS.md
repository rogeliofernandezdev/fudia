# AGENTS — Backend

Backend oficial en Go. Aplicar el `AGENTS.md` raíz.

Regla obligatoria: leer el `README.md` de este proyecto y todos los documentos
de `docs/` antes de implementar cambios relevantes.

## Arquitectura

- Monolito modular, arquitectura hexagonal y vertical slices.
- PostgreSQL, REST/OpenAPI y outbox para eventos confiables.
- `domain` no depende de infraestructura.
- Los adaptadores de módulos se componen únicamente en `cmd/api`.

## Persistencia

- PK: `id uuid PRIMARY KEY DEFAULT gen_random_uuid()`.
- PostgreSQL es la única autoridad que genera UUID persistidos.
- Toda consulta operativa aplica alcance de organización y local.
- Stock, caja, pagos y estados de pedido cambian mediante documentos o transiciones auditables.

## Dependencias durante la extracción modular

- Los casos de uso declaran interfaces Repository específicas; no importan pgx,
  HTTP ni adaptadores concretos.
- `transport/http` no ejecuta SQL ni construye repositorios.
- `internal/platform` no importa módulos de negocio ni `internal/legacy`.
- `internal/legacy/httpapi` es deuda pendiente, no una ubicación para nuevas
  verticales. Consultar `docs/10_BACKEND_REFACTOR.md` antes de migrar una.
- Conservar los límites transaccionales entre pedidos, stock y pagos; ejecutar
  `go test ./internal/architecture` junto con las puertas generales.
