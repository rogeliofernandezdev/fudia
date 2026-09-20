# Foods Admin Web

Frontend Next.js de administración. Antes de implementar, leer AGENTS.md y todos
los documentos de docs/. Verificar lint, typecheck, tests y build.

## Ejecución

1. Copiar `.env.example` a `.env.local` y configurar `FOODS_API_URL`.
2. Ejecutar `npm install` y `npm run dev`.
3. Abrir `http://localhost:5175/login`.

El navegador nunca recibe credenciales de PostgreSQL. La sesión se intercambia
con el backend mediante cookie `HttpOnly`.
