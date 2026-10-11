# Arquitectura

Monolito modular en Go con arquitectura hexagonal y vertical slices.
El dominio no importa HTTP, PostgreSQL ni proveedores externos. Los módulos se
componen en cmd/api. OpenAPI es la frontera oficial con los clientes.

Todos los flujos del backend usan los módulos y repositorios descritos en
`10_BACKEND_REFACTOR.md`. `platform` contiene mecanismos compartidos y
`internal/transport/httpapi` compone las rutas y sus permisos. El paquete
`internal/legacy` fue retirado.
