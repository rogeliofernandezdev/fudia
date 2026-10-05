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
  muestra el registro afectado, acción segura y estado pendiente. Su presentación
  es compacta: icono y título en una misma fila, centrados verticalmente, seguidos
  del mensaje directo y acciones, sin recuadros internos ni divisores. El icono
  conserva su tamaño y el título puede partirse en líneas sin desplazarlo. `description` admite texto o JSX para destacar únicamente el
  nombre con `<strong>` dentro de una frase. Productos y categorías preguntan
  «¿Deseas desactivar el producto/la categoría “nombre”?», sin explicaciones
  genéricas de disponibilidad ni historial. `subject`, `note` y `children` son
  opcionales; `subject` también se presenta como texto, con etiqueta opcional.
  Los campos omitidos no dejan bloques vacíos y las notas no imponen un icono.
  Los botones mantienen `--control-height`; en móvil comparten el ancho disponible.
  Cancelar recibe el foco inicial. Título y descripción tienen identificadores
  únicos vinculados al diálogo.
- `NotificationPopover`: contador en campana, lista por prioridad, marca temporal
  y acceso al historial completo.
- `DataTable`: cabecera oscura, filas alternas, estados con texto e icono, acciones
  homogéneas y representación responsive. Las acciones por fila son botones de
  icono con la altura de `--control-height`, nombre accesible y tooltip; editar
  usa azul operativo; activar/desactivar usa el interruptor de estado.
  Cancelar y quitar mantienen el rojo semántico. No se presentan como
  enlaces de texto. La columna
  de acciones centra el grupo completo, cada pictograma se centra ópticamente en
  su caja y los tooltips se anclan al centro del botón.
- Toda columna de acciones usa exclusivamente `RowActionButton`, construido
  sobre `IconButton`: SVG de 18 px (28 px para el interruptor de estado),
  contenedor cuadrado del tamaño de `--control-height`, alineación central,
  `aria-label` y tooltip descriptivo.
  No se permiten acciones textuales ni iconos generados de forma aislada por pantalla.
- El mapeo semántico es único en toda la aplicación: ver = ojo azul, editar =
  lápiz azul, activar/desactivar = interruptor binario, cancelar = equis circular roja
  y quitar fila = equis roja. Las
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

## Estado activo e inactivo

`Status` recibe `active` para representar estados binarios con texto e icono:
activo usa un check circular verde; inactivo, un signo menos circular neutro.
No se infiere el estado a partir del color ni del texto. Los otros estados
conservan su representación por `tone`. Los SVG son decorativos y el texto
visible comunica el estado a lectores de pantalla.

Activar y Desactivar se presentan como un interruptor en la columna Acciones,
con `role="switch"` y `aria-checked` derivado del estado confirmado: encendido
(verde `brand-700`, perilla derecha) para un registro activo; apagado
(rojo `danger-600`, perilla izquierda) para uno inactivo. El interruptor muestra
el estado actual, nunca el estado que tendrá después de pulsarlo. El tooltip comunica la acción siguiente: Activar o
Desactivar. `stateLabel` identifica el registro con un nombre accesible estable.

La caja táctil conserva `--control-height`; el SVG mide 28 px, centrado y sin
recuadro de color. Se opera con clic, toque, Espacio o Enter. No anticipa éxito:
se actualiza tras la respuesta del backend y el refresco del listado. Desactivar
conserva la confirmación existente. Durante una mutación se respetan `disabled`
y `aria-busy`. Cancelar una reserva usa `action="cancel"`, nunca un interruptor.

Referencias de interacción revisadas el 2026-10-05:
- [Lightspeed Restaurant O-Series: habilitar productos por local](https://o-series-support.lightspeedhq.com/hc/en-us/articles/31329473238299-Setting-up-products-and-prices-for-different-sites): interruptor por producto.
- [Lightspeed Restaurant K-Series: archivar y activar](https://k-series-support.lightspeedhq.com/hc/en-us/articles/1260804604870-Creating-and-editing-items): acciones textuales para retirar del catálogo.
- [Toast: administrar artículos](https://doc.toasttab.com/doc/platformguide/platformMenuManagerWorkingWithMenuItems.html): archivar/restaurar se distingue de disponibilidad.
- [WAI-ARIA: patrón switch](https://www.w3.org/WAI/ARIA/apg/patterns/switch/): estado binario y nombre estable.

El interruptor es una adaptación al modelo activo/inactivo de Fudia; no implica
que todos los proveedores usen el mismo control para retirar un producto.
