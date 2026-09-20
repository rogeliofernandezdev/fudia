# Componentes

Componentes pequeños y accesibles. Design system no conoce autenticación ni API.
Formularios usan React Hook Form y Zod. Catálogos siempre llegan del API.

Las tablas usan paginación, acciones textuales y scroll horizontal únicamente
cuando el contenido excede el ancho. La barra permanece oculta en reposo y se
muestra al pasar el cursor o enfocar el contenedor. Formularios CRUD usan modal
responsive, controles con la altura de `--control-height`, validación visible y
estados de envío.

## Contratos visuales obligatorios

- `FeedbackProvider`: única salida para éxito, error e información después de una
  acción remota. Diálogo centrado, accesible, cerrable y con movimiento reducido.
- `ConfirmDialog`: requerido antes de desactivar o ejecutar una acción sensible;
  muestra consecuencias, acción segura y estado pendiente.
- `NotificationPopover`: contador en campana, lista por prioridad, marca temporal
  y acceso al historial completo.
- `DataTable`: cabecera oscura, filas alternas, estados con texto e icono, acciones
  homogéneas y representación responsive. Las acciones por fila son botones de
  icono con la altura de `--control-height`, nombre accesible y tooltip; editar
  usa azul operativo y desactivar usa rojo semántico. No se presentan como
  enlaces de texto. La columna
  de acciones centra el grupo completo, cada pictograma se centra ópticamente en
  su caja y los tooltips se anclan al centro del botón.
- Toda columna de acciones usa exclusivamente `RowActionButton`, construido
  sobre `IconButton`: SVG de 18 px,
  contenedor cuadrado del tamaño de `--control-height`, alineación central,
  `aria-label` y tooltip descriptivo.
  No se permiten acciones textuales ni iconos generados de forma aislada por pantalla.
- El mapeo semántico es único en toda la aplicación: ver = ojo azul, editar =
  lápiz azul, desactivar = encendido rojo y quitar fila = equis roja. Las
  pantallas no pueden elegir localmente otro icono o color para estas acciones.
- `Pagination`: rango/total, selector de filas, páginas numeradas con elipsis y
  controles anterior/siguiente.
- Todas las tablas, incluidos productos, categorías, mesas, zonas y plantillas
  de gestión, consumen la única primitiva `Pagination` exportada por
  `src/components/ui.tsx`; no se permiten paginadores locales por pantalla.

Estos contratos toman de Bodegas la interacción, densidad y jerarquía, pero sus
colores siempre se resuelven con los tokens definidos en `02_DESIGN_SYSTEM.md`.

## Homologación transversal

Todas las rutas administrativas usan las mismas primitivas `Button`,
`IconButton`, `RowActionButton`, `Input`, `Select`, `Textarea`, `Status`, toolbar, tabla responsive
y paginación. Las acciones
principales combinan icono y texto; las acciones por registro son iconográficas,
con tooltip y nombre accesible. En pantallas estrechas, la tabla se convierte en
tarjetas estructuradas sin perder acciones, estado ni información clave.

No se permiten páginas que reutilicen por exportación otra ruta de negocio ni
variantes locales de controles, tablas o paginación. La referencia funcional es
Bodegas y la identidad cromática es exclusivamente Foods.

## Disponibilidad progresiva

El formulario de producto revela complejidad según el modo de la organización.
Un restaurante en modo `simple` nunca ve insumos, recetas ni unidades.

- El campo decisivo es el modo de control del producto, presentado con lenguaje
  de negocio: «Siempre disponible», «Cupo del día», «Descuenta un producto
  comprado» y «Receta de insumos». No se exponen los valores técnicos.
- En modo `simple` solo se ofrecen «Siempre disponible» y «Cupo del día». Las
  otras dos opciones aparecen únicamente cuando la organización usa inventario.
- Elegir un modo revela solo sus campos: el cupo muestra un número; el producto
  comprado muestra un selector de insumo; la receta muestra su lista de líneas.
  Nunca se muestran campos de un modo no elegido.
- El producto no tiene un campo de cantidad editable. La disponibilidad se
  presenta como dato derivado, de solo lectura, con su origen y el insumo que la
  limita cuando corresponde.
- La tabla muestra la disponibilidad como estado con texto e icono, no como
  número suelto: disponible, cupo restante, agotado o sin control.
- Marcar «agotado hoy» es una acción de operaciones, no del administrador, y se
  presenta como acción reversible del local, nunca como desactivación del catálogo.
