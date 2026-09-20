# AGENTS

## Objetivo

Construir una plataforma SaaS multiempresa para restaurantes peruanos.

## Reglas globales

1. Leer `README.md` y todos los documentos de `docs/` antes de cambios relevantes.
2. Mantener un monolito modular en backend y vertical slices en frontend.
3. El contrato OpenAPI es la frontera oficial entre backend y clientes.
4. PostgreSQL genera todos los UUID persistidos con `gen_random_uuid()`.
5. Ningún secreto se versiona ni se expone mediante variables públicas.
6. La autorización definitiva se valida en backend, incluyendo empresa y local.
7. No hardcodear catálogos de negocio en frontend.
8. Datos remotos siempre presentan skeleton, error explícito y estado vacío.
9. Diseñar mobile-first, validar desde 390 px y usar áreas táctiles de 44 px.
10. Usar español claro en interfaz y nombres técnicos en inglés en el código.
11. No copiar marcas, logotipos ni piezas publicitarias de las referencias.
12. Ejecutar lint, typecheck, tests y build antes de cerrar una implementación.

## Identidad

Cada frontend mantiene su sistema visual completo en su propio
`docs/02_DESIGN_SYSTEM.md`. Esos documentos son la autoridad local: verde
restaurante, azul operativo, violeta digital y superficies claras. No heredar
la paleta de `bodegas` ni depender de documentos de otro proyecto.
