# ADR: cantidad de proyectos

## Estado

Aceptado y actualizado tras la consolidación de frontends.

## Decisión

Mantener cuatro proyectos desplegables dentro de este workspace.

1. `foods-backend`.
2. `foods-admin-web`.
3. `fudia-concierge`.
4. `foods-infrastructure`.

`foods-admin-web` concentra POS, mesas, pedidos, comandas, caja y las funciones
administrativas. Las rutas y permisos por rol separan cada modo de trabajo sin
duplicar autenticación, contratos, componentes ni sistema visual.

## Alternativas descartadas

### Un proyecto por módulo de negocio

Eleva coordinación, observabilidad y consistencia transaccional antes de tener
volumen que lo justifique. Los módulos vivirán primero en un monolito modular.

### Dos frontends para operación y administración

Duplican autenticación, navegación, sistema visual, integración con OpenAPI y
mantenimiento responsive. La separación necesaria se resuelve mediante rutas,
layouts, permisos y vertical slices dentro de `foods-admin-web`.

### Aplicaciones separadas para POS, KDS y mozos

Comparten sesión, caché, design system y gran parte del modelo operativo. Una
PWA con rutas y permisos por rol cubre esos modos con menor duplicación.

### Aplicación móvil nativa desde el inicio

No aporta suficiente valor frente a una PWA instalable. Se reconsiderará si se
requieren periféricos o capacidades offline que el navegador no pueda ofrecer.

## Criterios de extracción futura

Un módulo podrá convertirse en servicio independiente si presenta escalado
claramente diferente, aislamiento regulatorio, equipo autónomo o despliegues
frecuentes que comprometan el resto. La primera candidata sería mensajería o
facturación electrónica, nunca las transacciones centrales de pedido y stock.
