# ADR: cantidad de proyectos

## Estado

Aceptado para iniciar.

## Decisión

Crear cuatro proyectos desplegables dentro de este workspace.

1. `foods-backend`.
2. `foods-operations-web`.
3. `foods-admin-web`.
4. `foods-infrastructure`.

## Alternativas descartadas

### Un proyecto por módulo de negocio

Eleva coordinación, observabilidad y consistencia transaccional antes de tener
volumen que lo justifique. Los módulos vivirán primero en un monolito modular.

### Un único frontend

Acopla despliegues y prioridades muy distintas: la caja y cocina necesitan
estabilidad y rapidez táctil; la administración necesita densidad analítica.

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
