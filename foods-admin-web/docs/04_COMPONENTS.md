# Componentes

Componentes pequeños y accesibles. Design system no conoce autenticación ni API.
Formularios usan React Hook Form y Zod. Catálogos siempre llegan del API.

Las tablas usan paginación, acciones iconográficas y scroll horizontal únicamente
cuando el contenido excede el ancho. La barra permanece oculta en reposo y se
muestra al pasar el cursor o enfocar el contenedor. Formularios CRUD usan modal
responsive, controles con la altura de `--control-height`, validación visible y
estados de envío.

## Contratos visuales obligatorios

- `FeedbackProvider`: única salida para éxito, error e información después de una
  acción remota. Diálogo centrado, accesible, cerrable y con movimiento reducido.
  Éxito/error se publican solo después de resolver la petición. Antes de mostrar
  la respuesta, cierra la pila de diálogos de acción mediante el contrato
  `onResponseClose` de `Dialog`, de hijo a padre; nunca queda un formulario detrás.
  Los cierres de respuesta no disparan confirmaciones de descarte. Las validaciones
  locales permanecen junto a los campos y no usan el diálogo de respuesta.
  Excepción: las transiciones Preparando/Listo de Cocina se confirman con el
  cambio de panel, sin modal de éxito. Los errores sí muestran el diálogo.
  Los avisos de éxito se cierran automáticamente (4,2 s por defecto) o mediante
  Aceptar/X. Errores e información requieren cierre explícito. No se cierran al
  pulsar el fondo ni con Escape.
- `Dialog`: comportamiento común de todos los modales. Mantiene el foco dentro
  de la ventana activa, soporta ventanas superpuestas y selectores con portal,
  bloquea el scroll del fondo y devuelve el foco al control que abrió la ventana.
  No agrega botones ni cierra con Escape; cada pantalla conserva su X/Cancelar.
  Cada propietario registra `onResponseClose` para liberar su estado, incluso
  mientras su mutación todavía está pendiente de refrescar datos. El propio
  diálogo de feedback no registra ese cierre.
  `data-dialog-initial-focus` identifica la acción inicial; en confirmaciones
  corresponde a Cancelar y en avisos a Aceptar.
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
  únicos vinculados al diálogo. Al abrir mantiene `modal-overlay-in` (200 ms)
  para el fondo y usa `confirm-scale-punch` (520 ms) para el panel: escala desde
  0.78 hasta 1.06, rebota a 0.98 y 1.012 y se asienta en 1 desde el centro.
  El efecto elástico no desplaza el panel. Solo se activa con
  `prefers-reduced-motion: no-preference`; movimiento reducido desactiva los
  efectos. La animación ocurre al montar el modal y no se repite al actualizar
  su contenido. El CSS pertenece al componente compartido.
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
  `src/design-system/page-header.tsx`; no se permiten paginadores locales por pantalla.

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
- El alta con «Porciones preparadas» exige «Cantidad disponible hoy» y la guarda
  junto al producto. La cantidad sigue perteneciendo al local/día, no al catálogo.
  La edición comercial no modifica cantidades; los ajustes posteriores se hacen
  desde Disponibilidad. Sin control e Inventario físico no muestran ese campo.
- El listado de Productos muestra nombre sin descripción y una columna
  Disponibles con el saldo real del local actual. Porciones usa el saldo del día;
  Inventario físico incluye su unidad; Sin control muestra «—». El control de
  cantidad y el estado activo del catálogo permanecen separados del saldo.
- Marcar «agotado hoy» es una acción de operaciones, no del administrador, y se
  presenta como acción reversible del local, nunca como desactivación del catálogo.

## Niveles de inventario

El ajuste de niveles de Inventario usa el mismo `crud-modal compact`, `FormField`
y formulario con footer interno que el resto de altas y ediciones. No añade CSS
local: los márgenes, la separación de acciones y la columna única en móvil
provienen del sistema compartido. Valida cantidades no negativas con RHF/Zod y
bloquea campos y cierres manuales mientras espera la respuesta del guardado.

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

## Loader de pantalla completa

`FullScreenLoader` cubre la pantalla durante la validación de sesión y las
transiciones de ruta. Reutiliza el `Logo` oficial y lo anima como algo que hierve,
en un ciclo de 4,8 s que nace desde abajo: el líquido (el logo a color con la
máscara ondulada `--loader-wave-mask`) sube desde el fondo con curva suave, hierve
a dos tercios de la altura con una oscilación senoidal leve y baja con calma para
volver a nacer. Una segunda capa más clara ondula en sentido contrario. Las
burbujas y el vapor se encienden solo cuando el líquido llega a su nivel y se
apagan al bajar; suben despacio, con aceleración suave, y se desvanecen al final.
No hay vibración ni cambios bruscos: las curvas se generan con 24 tramos por
ciclo. Todo usa tokens del design system. Con `prefers-reduced-motion` muestra el
logo completo, sin burbujas, vapor ni movimiento.
El texto inferior describe la espera («Preparando tu espacio»,
«Validando sesión») y el contenedor expone `role="status"` y `aria-busy`.
