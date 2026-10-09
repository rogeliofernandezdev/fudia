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

## Puertas

Antes de integrar: `npm run lint`, `npm run typecheck`, `npm test` y `npm run build`.
Las pruebas de arquitectura bloquean imports a `@/components`, CSS suelto en `src/app`
y artefactos locales versionados.
