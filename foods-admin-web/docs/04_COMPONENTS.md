# Componentes

Componentes pequeños y accesibles. Design system no conoce autenticación ni API.
Formularios usan React Hook Form y Zod. Catálogos siempre llegan del API.

Las tablas usan paginación, acciones iconográficas y scroll horizontal únicamente
cuando el contenido excede el ancho. La barra permanece oculta en reposo y se
muestra al pasar el cursor o enfocar el contenedor. Formularios CRUD usan modal
responsive, controles con la altura de `--control-height`, validación visible y
estados de envío.

## Catálogo homologado

Toda pantalla usa estas primitivas antes que cualquier solución local. Se
exportan desde `@/design-system` (salvo `useMoney` y `useCurrency`, desde
`@/providers`). Una variante nueva se agrega aquí y en el design system; nunca
dentro de un módulo.

### Botones

`Button kind`:

| Variante | Uso | Apariencia |
| --- | --- | --- |
| `primary` | La única acción dominante de la superficie | Sólido `primary-600`, realce teñido |
| `secondary` | Alternativas, «Reintentar», «Limpiar filtros», «Editar» | Blanco, texto `primary-700`, borde `primary-200` |
| `ghost` | «Cancelar», «Anterior», acciones neutras | Blanco, texto `ink-700`, borde `line` |
| `danger` | Solo el botón que confirma dentro de `ConfirmDialog` | Sólido `danger` |
| `danger-outline` | Disparadores destructivos: Cancelar pedido u orden, Anular, Cerrar turno, Quitar acceso | Blanco, texto `danger`, borde `danger-200` |

- No existe botón verde: el verde comunica un estado ya confirmado.
- Altura `--control-height`, radio de control, texto 600 de
  `--font-size-control`, icono de 18 px a la izquierda. La flecha final solo en
  «Continuar».
- `loading` deshabilita, anuncia `aria-busy` y la etiqueta pasa al gerundio del
  mismo verbo («Guardar» → «Guardando…»).
- Orden: acción segura a la izquierda, principal a la derecha. En móvil los
  botones de un pie ocupan el ancho disponible.
- Acciones por fila: solo `RowActionButton` (ver, editar, interruptor de
  estado, cancelar, quitar) con tooltip y nombre accesible.

### Modales

Comportamiento común: `Modal` sobre `Dialog` retiene y devuelve el foco, bloquea
el scroll del fondo y apila ventanas. **Escape y el clic en el fondo nunca
cierran un modal**; los únicos cierres son la X y la acción segura del pie.
La X nunca queda fuera del alcance del teclado, ni siquiera durante la carga.

```tsx
<Modal title="Nuevo gasto" icon="receipt" size="md" dirty={isDirty} busy={saving} onClose={close}>
  <ModalForm onSubmit={submit}>
    <ModalBody>…campos…</ModalBody>
    <ModalFooter><ModalCancel/><Button type="submit" loading={saving}>Guardar</Button></ModalFooter>
  </ModalForm>
</Modal>
```

| Tipo | Cuándo | Tamaño | Cierre y pie | Móvil |
| --- | --- | --- | --- | --- |
| `ConfirmDialog` | Desactivar, cancelar, anular, eliminar, descartar, quitar acceso | 400 | «Cancelar» (foco inicial) y confirmación `danger` o `primary` | Tarjeta centrada |
| Formulario | Altas y ediciones | `sm` hasta 4 campos · `md` hasta 2 columnas · `lg` secciones o líneas · `xl` tablas o asistentes | X y «Cancelar» (`ModalCancel`), principal «Guardar» | `sm`/`md` hoja inferior; `lg`/`xl` pantalla completa |
| Detalle | Consultar un registro | `md` o `lg` | Solo la X. Pie opcional para la acción del flujo; destructivas con `danger-outline` y `ConfirmDialog` | Pantalla completa (`lg`) u hoja |
| Vista previa | Ticket, QR, informe PDF | `sm` o `xl` | «Descargar» secundaria, «Imprimir» primaria | Igual que su tamaño |
| Asistente | Procesos de varios pasos | `xl` | Acciones contiguas a la derecha: «Anterior»/«Cancelar» antes de «Continuar»/«Guardar»; sin «Paso X de Y» si los pasos son visibles | Pantalla completa |
| Hoja | Configurar una opción dentro de una tarea (combo, resumen de comanda) | Contenido | Grip y X; inerte mientras está cerrada | Hoja inferior |
| Tarea completa | Comanda | 100 % | Un solo cierre; confirma si hay líneas | Pantalla completa |

Anatomía de la cabecera: icono blanco de 20 px sin superficie tintada, **título
de 18 px con la acción concreta** («Nuevo gasto», «Editar zona», «Mesa 04») y X.
Nunca añadir información debajo del título: sin subtítulo, eyebrow, fecha,
responsable ni estado dentro de la cabecera.

`ModalHeader` es la cabecera reutilizable tanto de `Modal` como de detalles
remotos que necesitan `Dialog` directamente. La cabecera reserva una fila para
icono, título y X. `subtitle`, `meta` y `details` se renderizan en `.modal-context`,
fuera de la cabecera; no duplicar información ya presente en el cuerpo.
En detalles operativos, responsables y fecha se ubican en el cuerpo. Cabecera
índigo `primary-700`, título blanco, cuerpo blanco y pie fijo `cloud-50`,
con los mismos márgenes horizontales de 24/16 px. Padding vertical de 8 px y altura
mínima de 64 px, no altura fija: los textos largos conservan su información.
Icono primario blanco de 20 px, centrado en un espacio de 44 px sin recuadro.
La X blanca de 20 px se centra en un cuadro visible de 32 px con contorno blanco
de 1 px, fondo transparente y sin sombra, dentro de un área de clic de 44 px;
no cambia de color con hover y mantiene foco blanco visible para teclado.
Todos los iconos de cabecera son blancos, también en altas, ediciones y errores
de carga. Las respuestas success/error y confirmaciones warning conservan su
componente independiente y su semántica.

`ModalFooter({children,note,status,statusLabel})` agrupa los botones contiguos
a la derecha, con 10 px de separación y la acción principal al final. No usar
`margin-right:auto` en Cancelar ni `space-between` entre las acciones. En móvil
las acciones adyacentes se reparten la fila; los estados y notas usan otra fila
si hace falta. `status` contiene el `Status` real del registro, opcional; no
añadir estado ni resumen ficticio a formularios simples.

`ModalWorkspace({children,summary})` es el patrón opcional para un detalle con
resumen real. Usa una sola columna: contenido desplazable y región `.modal-summary`
inferior a ancho completo, persistente en todos los tamaños. No usa panel lateral.
`.modal-summary-lines`, `.modal-summary-line` y `.modal-summary-total` son sus
estilos únicos, no duplicados por módulo. El detalle Pedidos/Salón lo usa en un
panel de hasta 760 px; los formularios sin resumen continúan con `ModalBody`.
Todo el fondo del resumen usa gris muy claro `cloud-50`; el estado abierto usa azul tenue.
En el detalle de pedidos, las acciones inferiores comparten el mismo fondo.
La cabecera del resumen agrupa título y estado, con separador inferior. Los
importes secundarios se distribuyen en columnas equilibradas y el saldo en una
región sin tarjeta, separada con `line` vertical también en móvil. A 390 px
Subtotal, Pagado y Saldo pendiente se mantienen en una fila.
`Status({icon,emphasis})` permite reforzar un estado de lectura sin hacerlo
parecer una acción: icono de la familia existente y peso 700. Cuenta abierta usa
`sky` destacado: fondo tenue `sky-100`, texto e icono `sky-800`, sin sombra ni hover.
Por cobrar usa ámbar, Pagada verde y Finalizada gris (`ink-800`/`cloud-150` con énfasis).
Todos los estados usan fondos tenues, sin variantes sólidas. Los estados de preparación y los estados binarios no cambian.
La paleta y asignación obligatorias se consultan únicamente en la
[matriz de estados](02_DESIGN_SYSTEM.md#colores-de-operación-pedido-preparación-y-cuenta);
no definir colores alternativos en cada consumidor.

`ModalItemList` agrupa líneas de consumo sin encabezados de tabla ni borde exterior.
Cada línea usa `modal-item-row` y `role="listitem"`; su cantidad usa
`modal-item-quantity`, circular, con fondo `primary-100` y texto `primary-700`;
no tiene interacción ni representa un estado. No se repite un conteo de unidades
en el encabezado de productos. Los separadores entre líneas son
`line`. Las notas usan `modal-detail-notes` fuera de esa
lista, con encabezado propio, padding compacto y fondo `cloud-50`, sin borde.
El resumen inferior usa `cloud-50` y un límite superior neutro de 1 px. Estos estilos tienen un solo dueño
en el design system, no se duplican por módulo.

Reglas:

- Datos: `dt` y `modal-data-label` usan 600; `dd` y `modal-data-value`, 400.
  Títulos de sección 700. Los tokens se definen en `variables.css` y los roles
  se aplican una sola vez en `modal.css`;
  no invertir etiqueta y valor en estilos locales. Los bloques heredados sin
  `dl` consumen `--modal-label-weight` y `--modal-value-weight`.
  «Mesa», «Cliente» o «Venta» identifican el dato mostrado, nunca «Cliente / Mesa»
  en un detalle. Totales, productos, estados y acciones conservan sus roles.
  Los valores de controles también son regulares dentro del modal. No cambia
  la composición de confirmaciones, respuestas ni documentos PDF.
- Cabecera y pie permanecen visibles; solo `ModalBody` se desplaza. En
  `ModalWorkspace` también permanece visible el resumen.
- Foco inicial en el primer campo (`data-dialog-initial-focus` cuando no es el
  primer control); en confirmaciones, «Cancelar».
- `dirty`: X y «Cancelar» abren «¿Descartar los cambios?». Obligatorio en
  formularios de más de 3 campos y en la comanda.
- `busy`: bloquea X, «Cancelar» y campos mientras dura la petición.
- Carga remota: `ModalLoading` dentro del `Modal` (conserva título y X); el
  error: `ModalError` con «Reintentar». `RemoteModalSkeleton` queda para
  cargas en las que aún no se conoce el título.
- Un modal no abre otro modal salvo `ConfirmDialog` o un alta rápida de
  catálogo (`sm`). Las capas usan los tokens `--z-*`.
- Antes de usar un modal, preferir edición en línea o un panel cuando la tarea
  no necesita interrumpir.

### Campos

- `FormField` siempre: etiqueta arriba (12 px, 600), control, y ayuda o error
  debajo (12 px). «Opcional» solo mediante `optional`; los obligatorios no
  llevan marca.
- `FormField` propaga `aria-invalid` y `aria-describedby` a `Input`,
  `Select`, `Textarea` y `MoneyInput`. El error de campo no usa
  `role="alert"`; el resumen del formulario se anuncia una sola vez.
- `MoneyInput symbol={useCurrency().symbol}`: un solo borde, prefijo, teclado
  decimal y cifras tabulares. Nunca `toFixed` + símbolo: los importes se
  muestran con `useMoney()`.
- `NumberStepper`: botones de 44 px en táctil, valor tabular y, en el mínimo,
  «Quitar» con papelera cuando la línea puede eliminarse.
  `appearance="soft"` ofrece una cápsula `cloud-100` sin borde ni divisiones,
  con botones de 44 px también en escritorio y foco visible sin recortar. Se usa
  en carta y resumen de comanda; la apariencia por defecto no cambia.
- `SearchField` para buscar, `Select` para catálogos del API, `Tabs` para
  vistas o segmentos. Sin `<input>`, `<select>` ni pestañas locales.
- Validación: React Hook Form + Zod en `onSubmit` y revalidación `onChange`;
  foco en el primer campo con error. Mensajes: «Ingresa…» o «Selecciona…».
- Solo lectura con fondo `cloud-50` y borde punteado; deshabilitado con
  `cloud-100`.

### Avisos y feedback

| Situación | Mecanismo |
| --- | --- |
| Validación de un campo | Texto bajo el campo |
| Resultado de una acción remota | `FeedbackProvider.notify` (éxito se cierra solo; error requiere «Aceptar», que es neutro) |
| Error al cargar datos | `ErrorState` con «Reintentar» en el lugar del contenido, nunca un aviso global |
| Restricción o advertencia contextual | `InlineNotice tone="info"\|"warning"\|"danger"\|"success"` |
| Acción destructiva o irreversible | `ConfirmDialog` |

Títulos de aviso: «No pudimos {verbo} {objeto}» para errores y «{Objeto}
{participio}» para éxitos («Gasto anulado»). Activar es inmediato; desactivar,
anular, cancelar, eliminar, descartar y quitar acceso piden confirmación.

### Glosario de verbos

| Verbo | Uso |
| --- | --- |
| Nuevo / Nueva {objeto} | Abrir un alta (nunca «Crear registro») |
| Guardar · Guardando… | Persistir un formulario |
| Registrar | Hechos operativos: cobro, pago, ingreso, recepción |
| Aplicar | Contexto y filtros |
| Cancelar | Cerrar un formulario sin guardar |
| Cancelar {objeto} | Anular un pedido u orden |
| Anular | Documentos contables y gastos |
| Desactivar / Activar | Registros de catálogo |
| Quitar | Desvincular una fila, archivo o integrante |
| Eliminar | Borrado real |
| Continuar / Anterior | Navegar en asistentes |
| Reintentar | Recuperar una carga |
| Limpiar filtros | Restablecer filtros |
| Aceptar | Cerrar un aviso |

### Encabezado de página

`PageHeader({title, description?, action?, back?, meta?})`: título igual al
nombre de la navegación, descripción de una línea, `back` hacia la ruta padre,
`meta` para un estado junto al título y `action` con una acción primaria y,
como máximo, una secundaria. Sin eyebrow y sin filtros dentro del encabezado.
En móvil la acción principal se reduce a un botón de icono de 44 px.

### Iconos

`Icon size` acepta `IconSize` (14, 16, 18, 20, 24, 28, 32, 36, 40, 48). Un
icono tiene un solo significado en toda la aplicación (ver
`02_DESIGN_SYSTEM.md` › Iconos).

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

## Primitivas homologadas (usar siempre antes que soluciones locales)

Todas se exportan desde `@/design-system` (salvo `useMoney`, desde `@/providers`).
Un módulo no debe definir su propio estado vacío, skeleton, buscador, pestañas,
pastilla de estado ni formateador de dinero.

- `EmptyState({icon,title,text,action})`: vacío accionable sobre `.catalog-state`.
  El texto distingue «Aún no hay …» de «Sin coincidencias» cuando hay filtros. La
  acción, si existe, es un `Button` con etiqueta específica («Nuevo producto»),
  nunca «Crear registro».
- `ErrorState({title,text,onRetry})`: error recuperable sobre
  `.catalog-state.error`; «Reintentar» es siempre `Button kind="secondary"` con
  icono `refresh`. No reemplaza datos por ceros.
- `Skeleton` y `TableSkeleton({columns,rows})`: un único brillo
  (`ds-skeleton-shimmer`). `TableSkeleton` recibe los mismos encabezados que la
  tabla real; la primera celda reproduce icono + nombre + detalle y la última,
  las acciones. Prohibido el texto «Cargando…» y las barras genéricas.
- `SearchField({placeholder,label})`: buscador de barra de filtros con lupa, a
  `--control-height`. Reemplaza `<input>` crudos dentro de `.toolbar`.
- `Tabs({items,value,onChange,label})`: control segmentado con `role="tablist"`,
  foco itinerante y flechas/Inicio/Fin; contador opcional por pestaña; 44 px en
  móvil. Reemplaza `.catalog-tabs` y variantes por módulo.
- `Status({tone})`: píldora suave tintada, sin borde, para preparación/pedido y cuenta.
  Tonos `green` (listo/activo/éxito), `blue` (confirmado), `sky` (cuenta abierta), `orange`
  (advertencia/por resolver), `red` (error/cancelación), `violet` (preparación/digital),
  `teal` (entregado) y `gray` (nuevo/neutro/finalizada).
  Reemplaza pastillas propias de cada módulo. La matriz oficial de pedido,
  preparación y cuenta está en `02_DESIGN_SYSTEM.md`; no se sobrescribe por pantalla.
- `useMoney()`: importe con símbolo, posición y decimales de la empresa y
  separadores del país del local; conserva el signo. Nunca `toFixed` + símbolo.

## Refinamiento de pestañas y modales compartidos

`Tabs` es también la única implementación de Productos/Categorías y Mesas/Zonas;
se retiró el marcado y CSS de `catalog-tabs`. La pista neutra no lleva una segunda
sombra o marco. Cada opción conserva `--control-height`, selección índigo suave,
texto azul y contador contrastado. Así la navegación no compite con «Nuevo …».
Las flechas, Inicio/Fin y foco itinerante permanecen; en móvil las opciones tienen
44 px y el desborde horizontal se desplaza sin envolver las etiquetas.

Los modales alinean cabecera, cuerpo y pie sobre un margen común de 24 px en
escritorio y 16 px en móvil. La X y el icono primario comparten una caja de 44 px
y glifos de 20 px; el icono primario no tiene fondo y la X tiene contorno blanco de 32 px,
sin relleno, sombra ni cambio de color con hover. Las notas del pie tienen
una fila propia, por encima de las acciones. Se preservan los tamaños,
el cierre explícito y los contratos de foco/respuesta. No hay franja de marca
en los modales: la cabecera índigo ocupa el ancho y el borde superior completos.

En móvil el fondo no añade padding superior a un panel de `100dvh`: eso recortaba
el pie de los modales completos. Las hojas compactas dejan margen superior seguro;
los modales completos suman el área segura a su cabecera y todos mantienen el
área segura inferior del pie. El contenido es el único bloque desplazable.
