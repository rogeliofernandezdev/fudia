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

## Dependencias modulares

- Los casos de uso declaran interfaces Repository específicas; no importan pgx,
  HTTP ni adaptadores concretos.
- `transport/http` no ejecuta SQL ni construye repositorios.
- `internal/platform` no importa módulos de negocio ni `internal/legacy`.
- `internal/transport/httpapi` compone rutas y autorización; no accede a PostgreSQL.
- Toda colaboración entre módulos debe importar únicamente el `contracts` publicado
  por el módulo propietario (se exceptúa `audit/application` por contener solo
  un puerto). No importar domain, application, infrastructure ni transport de otros módulos.
- Los contratos públicos no importan SQL, HTTP ni implementaciones, y CI protege
  este límite en `internal/architecture/intermodule_test.go`.
- Los movimientos por recepciones y devoluciones se aplican exclusivamente a través
  de `inventory/application.PurchaseLedger`; Purchasing no calcula costos promedio ni
  actualiza saldos directamente.

- El paquete `internal/legacy` fue retirado y no debe reintroducirse.
  Consultar `docs/10_BACKEND_REFACTOR.md` para las responsabilidades de cada módulo.
- Conservar los límites transaccionales entre pedidos, stock y pagos; ejecutar
  `go test ./internal/architecture` junto con las puertas generales.
