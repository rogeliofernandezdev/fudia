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

## Estándar visual obligatorio

La [matriz de estados aprobada](docs/02_DESIGN_SYSTEM.md#colores-de-operación-pedido-preparación-y-cuenta)
es la única referencia para pedido, preparación y cuenta en pantallas nuevas y existentes.
Usar `Status`, tokens compartidos y mapeos centrales: fondos tenues y texto oscuro,
sin reinterpretar colores por pantalla. Las reglas están en `AGENTS.md`.

Los modales también respetan la distribución compartida: franjas de datos en
línea con etiqueta/valor del mismo tamaño y fecha al extremo derecho; botones
móviles que llenan su fila, sin medias columnas vacías. Dos acciones comparten
el ancho cuando caben; apiladas, cada una ocupa el ancho completo. Contrato y
excepciones en `docs/04_COMPONENTS.md`, con autoridad en `docs/02_DESIGN_SYSTEM.md`.

El color de Cancelar depende del efecto: salir sin guardar es neutro; cancelar
un pedido u orden persistido es rojo tenue y requiere confirmación; confirmar
la anulación es rojo sólido. Cancelar dentro de esa confirmación sigue neutro.
Usar las intenciones compartidas de `Button`, sin colores locales. Continuar
es índigo con flecha al final. Referencia: `docs/02_DESIGN_SYSTEM.md`.

## Verificación

Ejecutar `npm run lint`, `npm run typecheck`, `npm test` y `npm run build`.
