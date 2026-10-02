# Estado e integración

TanStack Query administra estado remoto por organización/local. Presentación no
contiene rutas HTTP ni DTO crudos. Toda dependencia muestra skeleton, error y vacío.

## Autocompletes remotos

Los autocompletes de catálogos grandes no cargan el catálogo completo. La apertura inicial usa una página pequeña de hasta 10 opciones. Las búsquedas no consultan con menos de 3 caracteres; desde 3 caracteres delegan el filtro al backend y recorren la paginación necesaria para reunir todas las coincidencias. Las opciones ya seleccionadas en formularios multirregistro se excluyen en cliente sin alterar el resultado remoto.

## Performance de consultas en cliente

Las búsquedas de listados no disparan una petición por pulsación. Deben reutilizar
`useDebouncedValue` con una ventana aproximada de 300 ms y mantener el texto
visible separado del valor consultado.

TanStack Query conserva la caché por clave; las mutaciones invalidan únicamente
las familias afectadas. Los diálogos pesados que no forman parte del primer render
se cargan con `next/dynamic` cuando la acción realmente se abre.

El prefetch de datos se incorpora solo en rutas frecuentes y cuando la medición
demuestre que reduce latencia sin duplicar tráfico.
