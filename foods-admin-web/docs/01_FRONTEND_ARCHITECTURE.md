# Arquitectura frontend

`foods-admin-web` es un monolito modular Next.js organizado por vertical slices.

## Fronteras

- `src/app`: routing, layouts y BFF. No contiene lógica de negocio ni CSS de módulos.
- `src/modules/<vertical>`: código y estilos propiedad de una vertical de negocio.
- `src/design-system`: primitivas visuales reutilizables sin conocimiento del dominio.
- `src/shared`: utilidades y componentes genéricos sin ownership de negocio.
- `src/shell`: navegación y composición del área administrativa.
- `src/styles`: únicamente estilos globales/base.

Las rutas deben ser composition roots pequeños y consumir el `index.ts` público del módulo.
No se crea un contenedor genérico `src/components` para features de negocio.

## Estructura de un módulo

Cuando una vertical necesita capas separadas, usar:

```text
modules/<vertical>/
  domain/
  application/
  infrastructure/
  presentation/
  index.ts
```

Para subverticales grandes se permite agruparlas dentro del módulo, manteniendo ownership explícito.
El CSS específico se coloca junto a la presentación que lo usa o en `modules/<vertical>/styles`.

## Dependencias

`app -> modules/shared/shell`

`presentation -> application/domain/design-system/shared`

`application -> domain/infrastructure`

`infrastructure -> shared/api/domain`

El dominio no depende de React, Next.js ni HTTP. La presentación no debe crear contratos HTTP nuevos;
las integraciones nuevas se encapsulan en infrastructure antes de exponerse desde el módulo.

## CSS

No se permiten hojas CSS de negocio dentro de `src/app`. Los estilos globales viven en
`src/styles`; los del design system en `src/design-system/styles`; navegación en
`src/shell/styles`; y los de negocio dentro de su módulo.

Las hojas compartidas del design system (modal, paginación, acciones de tabla,
skeleton, pestañas, confirmación, controles, campo, skeleton de modal remoto y
loader) se registran una sola vez en `src/app/layout.tsx`, junto a `globals.css`
y `data-table.css`: los barrels se podan por ruta y varias pantallas reutilizan
esas clases sin importar el componente. Las hojas pesadas y específicas (mapa,
editor enriquecido) siguen con su componente.

## Rendimiento de carga

- Cada `index.ts` público (módulos, `@/design-system`, `@/providers`,
  `@/shared/management`) contiene solo re-exportaciones. `next.config.ts` los
  declara en `experimental.optimizePackageImports` (los módulos se descubren
  solos) y `package.json` marca `"sideEffects": ["*.css"]`. Así una página que
  importa `@/modules/operations` carga solo la vista que usa y no POS, Salón,
  Cocina y Caja a la vez. Un barrel con código propio rompe esta poda.
- Librerías pesadas que solo aparecen tras una acción se cargan bajo demanda:
  `next/dynamic` para componentes (editor de producto, buscador de combos,
  autocompletes de Recetas, diálogos de POS/Pedidos/Ventas) e `import()` para
  utilidades (`qrcode`, `jspdf`). El reemplazo de carga conserva la geometría
  final (por ejemplo `Skeleton` con `--control-height`).
- `src/app/(admin)/loading.tsx` muestra `RouteSkeleton` dentro del shell: habilita
  el prefetch parcial de rutas dinámicas y responde al toque al instante.
  `staleTimes.dynamic` (30 s) reutiliza el segmento de una pantalla ya visitada;
  los datos siguen gobernados por TanStack Query.
- Las imágenes de interfaz se sirven a su tamaño de uso (`images.unoptimized`):
  `logo-mark.png` (144 px) y `login-brand.png` (360 × 240). Los originales en alta
  resolución quedan solo para PDF e impresión de QR.
- La medición y el presupuesto viven en `docs/07_TESTING.md` › Rendimiento.

## Puertas

Antes de integrar: `npm run lint`, `npm run typecheck`, `npm test` y `npm run build`.
Las pruebas de arquitectura bloquean imports a `@/components`, CSS suelto en `src/app`
y artefactos locales versionados.
