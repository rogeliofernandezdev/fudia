# Foods Admin Web

Frontend Next.js administrativo de Fudia. Antes de implementar cambios, leer `AGENTS.md`
y los documentos de `docs/`.

## Estructura

- `src/app`: rutas, layouts y BFF.
- `src/modules`: verticales de negocio.
- `src/design-system`: primitivas visuales.
- `src/shared`: código transversal sin dominio.
- `src/shell`: navegación administrativa.
- `src/styles`: base global.
- `docs/visual-references`: referencias visuales conservadas fuera de la raíz.

No agregar features de negocio a una carpeta genérica `src/components` ni CSS de negocio a `src/app`.

## Ejecución

1. Copiar `.env.example` a `.env.local` y configurar `FOODS_API_URL`.
2. Ejecutar `npm install` y `npm run dev`.
3. Abrir `http://localhost:5175/login`.

`.env.local` es local y no se versiona. El navegador nunca recibe credenciales de PostgreSQL;
la sesión se intercambia con el backend mediante cookie `HttpOnly`.

## Verificación

Ejecutar `npm run lint`, `npm run typecheck`, `npm test` y `npm run build`.
