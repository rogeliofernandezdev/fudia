# Arquitectura

Monolito modular en Go con arquitectura hexagonal y vertical slices.
El dominio no importa HTTP, PostgreSQL ni proveedores externos. Los módulos se
componen en cmd/api. OpenAPI es la frontera oficial con los clientes.

La implementación está en transición: ver `10_BACKEND_REFACTOR.md` para el
alcance extraído, las dependencias verificadas y las verticales pendientes.
`platform` contiene solo mecanismos compartidos; el código anterior pendiente
está explícitamente aislado en `internal/legacy/httpapi`.
