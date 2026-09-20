# Arquitectura

Monolito modular en Go con arquitectura hexagonal y vertical slices.
El dominio no importa HTTP, PostgreSQL ni proveedores externos. Los módulos se
componen en cmd/api. OpenAPI es la frontera oficial con los clientes.
