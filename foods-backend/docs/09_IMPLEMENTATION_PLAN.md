# Plan

## Implementado en la fundación

- Pool PostgreSQL configurado exclusivamente por entorno y con comprobación inicial.
- Migración multiempresa para identidad, locales, menú, inventario y compras.
- Sesiones opacas con hash en base de datos y cookie `HttpOnly`.
- Dashboard administrativo y consulta/registro de productos delimitados por sesión.
- CRUD de categorías y productos con permisos, paginación, desactivación y auditoría.
- Configuración monetaria y fiscal por empresa, expuesta mediante OpenAPI.
- Contrato OpenAPI 0.2 y errores estables.

## Siguiente secuencia

Permisos detallados; categorías y recetas; documentos de stock; compras y
recepciones; pedidos y comandas; cobro, impresión y comprobantes; WhatsApp y reportes.
