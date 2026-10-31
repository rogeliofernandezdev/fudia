# Componentes

Los niveles de Inventario usan `unitCountable` del API: mínimo y óptimo
con paso 1/teclado numérico si se cuentan piezas, o 0.001/decimal si se miden.
Los ceros finales del formato de base se omiten (`100.000` → `100`), sin redondear
fracciones históricas. El óptimo nunca queda por debajo del mínimo. Se retira
Punto de reorden del formulario, listado, borrador y payload de guardado; el campo
antiguo permanece en backend solo por compatibilidad, sin condicionar el óptimo.
Este diálogo corto usa `modal-form-compact`: 12 px sobre su sección inicial,
conservando márgenes laterales y pie compartidos. Los dos campos de cantidad
usan `FormField className="numeric-value"`: valor de 16 px / 700 y cifras
tabulares e icono neutro de 20 px separado 12 px del número. Ambos usan una
caja simple (`box`) para representar existencias; las etiquetas distinguen
el mínimo del óptimo. El icono mantiene presencia sin reducirse y el número
mantiene la jerarquía principal del campo.

Componentes pequeños y accesibles. Design system no conoce autenticación ni API.
Formularios usan React Hook Form y Zod. Catálogos siempre llegan del API.

Medios de pago recibe `catalogTotal` y `catalogLimit` del API y oculta «Nuevo
medio» al completar las cuatro plazas, incluidos inactivos, aunque el listado
esté filtrado o paginado. La edición sigue disponible. Su diálogo reutiliza
`ModalForm` + `ModalBody` + `ModalFooter`: solo el cuerpo se desplaza, sin el pie
heredado sticky ni márgenes negativos. Cabecera y acciones conservan los estilos
compartidos. La precarga viene del backend: Perú tiene Efectivo, Tarjeta, Yape y
Plin; los demás países, Efectivo y Tarjeta.

El modal de Locales muestra «Datos del local» solo en la cabecera. Los campos
generales comienzan directamente; `LocationMap` aporta su propia etiqueta
«Dirección del local» y recibe `addressError`, sin otro `FormField` que la
repita. El bloque fiscal conserva un único rótulo visible «Perfil fiscal» y
su sección mantiene el nombre accesible «Configuración fiscal».
Usa `ModalForm` + `ModalBody` + `ModalFooter`: el mapa y sus controles quedan
dentro del cuerpo desplazable y recortado, separados del pie. El formulario
no usa el footer sticky heredado que permitía superponer controles del mapa
sobre Cancelar o Guardar.

## PdfViewer

`@/shared/pdf-viewer` exporta un modal cliente reutilizable basado en el visor
oficial completo de Mozilla. Recibe `document: Blob | null`, `filename`, `title`
y `onClose`; admite `subtitle`, `size` (`sm`: 480 px, `lg`: 960 px), `loading`,
`error`, `onRetry`, `notice` y `details` opcionales. El zoom inicial ajusta la
página completa y conserva las dimensiones del PDF.
El llamador obtiene/genera el Blob con sus permisos y hooks actuales. El visor
no conoce el dominio ni recibe tokens o URLs privadas. Se monta al previsualizar
y se desmonta al cerrar. Las acciones de impresión y descarga viven únicamente
en el visor. Ver [contrato, actualización y validación](09_PDF_VIEWER.md).

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

### Iconos, indicadores y filtros

- `Icon name` — set único de Tabler (monocromo, trazo 1.75); nunca importar
  Tabler directamente en un módulo.
- `IconTile name tone size appearance` — icono sobre cuadrito tintado (`primary`, `brand`,
  `digital`, `teal`, `sky`, `peach`, `rose`, `warning`, `danger`, `neutral`). Por
  defecto `appearance="soft"`; `vivid` admite `primary`, `brand` y `danger` para
  acentos de Dashboard y flujos de Caja. No sustituye `Status` ni colorea botones.
- `StatCard icon tone label value meta href attention` — indicador con flecha ↗
  hacia el detalle; agrupar con `.stat-grid`.
- `Input icon` — campo con icono al inicio (correo, teléfono, dirección,
  documento, referencia).
- `Select chip` — filtro de barras de listados con radio estándar de campo; se tiñe cuando filtra.

### Detalles

- `DetailHero`, `DetailContact`, `DetailFacts`, `DetailTimeline` — organización
  de modales de detalle (identidad con estado, contacto, datos en dos columnas,
  progreso o movimientos con círculos). Combinar con `Tabs`.
- En Ventas, `sales-detail-overview` agrupa identidad y contacto en dos columnas
  (una en móvil); `sales-detail-content` relaciona las pestañas con su
  panel con las pastillas compartidas sobre una misma superficie `cloud-50`,
  sin otra tarjeta para el contenido. `sales-detail-facts` alinea etiquetas y valores en filas.
  El contenedor mantiene `radius-card` (14 px); pista y selección heredan
  `radius-pill` del componente compartido. Los paneles conservan sangría de 12 px.
  Las etiquetas tienen ancho natural en escritorio y tres columnas en móvil;
  no altera la navegación horizontal del resto de `Tabs`.
  Fecha y reloj no se separan en nodos sin alineación; cliente y mozo se conservan.

### Botones

`Button kind`:

| Variante | Uso | Apariencia |
| --- | --- | --- |
| `primary` | La única acción dominante de la superficie | Sólido `primary-600`, realce teñido |
| `secondary` | Consultas como «Ver resumen de pago», alternativas, «Reintentar», «Limpiar filtros», «Editar» | Blanco, texto `primary-700`, borde `primary-200` |
| `ghost` | Salir sin guardar («Cancelar»), «Anterior», acciones neutras | Blanco, texto `ink-700`, borde `line` |
| `danger` | Solo el botón que confirma dentro de `ConfirmDialog` | Sólido `danger` |
| `danger-outline` | Disparadores destructivos: Cancelar pedido u orden, Anular, Cerrar turno, Quitar acceso | Fondo `danger-50`, texto `danger`, borde `danger-200`; hover `danger-100` / `danger` / `danger-600` |

- No existe botón verde: el verde comunica un estado ya confirmado.
- `Button action` centraliza acciones repetidas: `cancel` → neutro con X;
  `cancel-record` → rojo tenue con icono cancelar; `back` → neutro con flecha
  izquierda; `continue` → primario con flecha al final, a la derecha del texto.
  La intención fija variante e icono, sin interpretar la etiqueta visible.
  `ModalCancel` consume `action="cancel"`; no añadir colores locales.
- Cancelar una salida de formulario no anula datos persistidos: neutro. Anular
  pedido/orden: rojo tenue y `ConfirmDialog` posterior. Confirmar la anulación:
  rojo sólido. Cancelar dentro de esa confirmación es la salida segura y
  permanece neutro. Descartar cambios se confirma con una acción explícita
  «Descartar», no recoloreando el Cancelar del formulario.
- La intención decide la variante: una consulta sigue siendo secundaria aunque
  sea la única acción del footer. Guardar, registrar, enviar o confirmar el avance
  del flujo son primarias; salir sin guardar/Anterior son neutras. Máximo una primaria por
  superficie, sin inventarla en un detalle de lectura. «Ver resumen de pago» usa
  `eye` y abre el documento no fiscal existente, sin registrar otro cobro.
- Altura `--control-height`, radio de control en escritorio y `radius-pill` en
  móvil (≤600 px), texto 600 de
  `--font-size-control`, icono de 18 px a la izquierda. La flecha final solo en
  «Continuar».
- La forma móvil pertenece a `.button` en `globals.css`, no a los módulos.
  Campos y X conservan su geometría; todas las variantes tienen al menos 44 px
  de área táctil y mantienen el mismo orden de teclado.
- `loading` deshabilita, anuncia `aria-busy` y la etiqueta pasa al gerundio del
  mismo verbo («Guardar» → «Guardando…»). El `Spinner` compartido
  reemplaza al icono en su misma posición (al final en «Continuar») para que el
  botón no cambie de ancho; `aria-busy` explícito produce el mismo resultado.
  Todo botón cuya etiqueta cambia a gerundio usa `loading` con la misma
  condición; `disabled` queda para la validez del formulario. Estados completos:
  `02_DESIGN_SYSTEM.md` › Microinteracciones y estados.
- Orden: acción segura a la izquierda, principal a la derecha. En móvil los
  botones de un pie ocupan el ancho disponible.
- Acciones por fila: solo `RowActionButton` (ver, editar, interruptor de
  estado, cancelar, quitar) con tooltip y nombre accesible. La composición
  operativa de Disponibilidad usa el contrato específico descrito abajo.

### Modales

Comportamiento común: `Modal` sobre `Dialog` retiene y devuelve el foco, bloquea
el scroll del fondo y apila ventanas. **Escape y el clic en el fondo nunca
cierran un modal**; los únicos cierres son la X y la acción segura del pie.
La X nunca queda fuera del alcance del teclado, ni siquiera durante la carga.
Con el teclado virtual abierto en móvil (`(width<=600px) and (height<=560px)`,
posible por `interactive-widget=resizes-content`) la hoja ocupa todo el alto,
la cabecera se compacta a 52 px, el pie reduce su relleno y `modal-context` se
oculta; al cerrar el teclado todo vuelve a su tamaño.

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
azul `primary-700`, título blanco, cuerpo blanco y pie fijo `cloud-50`,
con los mismos márgenes horizontales de 24/16 px. Padding vertical de 8 px y altura
mínima de 64 px, no altura fija: los textos largos conservan su información.
Icono primario blanco de 20 px, centrado en un espacio de 44 px sin recuadro.
La X blanca de 20 px se centra en un cuadro visible de 32 px con contorno blanco
de 1 px, fondo transparente y sin sombra, dentro de un área de clic de 44 px;
no cambia de color con hover ni al guardar y mantiene foco blanco visible para teclado.
Con `busy`, queda deshabilitada sin reducir su opacidad: icono y contorno siguen blancos.
Todos los iconos de cabecera son blancos, también en altas, ediciones y errores
de carga. Las respuestas success/error y confirmaciones warning conservan su
componente independiente y su semántica.

`ModalFooter({children,note,status,statusLabel})` agrupa los botones contiguos
a la derecha, con 10 px de separación y la acción principal al final. No usar
`margin-right:auto` en Cancelar ni `space-between` entre las acciones. En móvil
las acciones adyacentes se reparten la fila por igual si caben. Un botón solo
en su fila, incluido Cancelar, ocupa el 100 % del ancho útil; apilados, todos
llenan su fila. Nunca reservar una media columna vacía. Los estados y notas usan otra fila
si hace falta. `status` contiene el `Status` real del registro, opcional; no
añadir estado ni resumen ficticio a formularios simples.

En móvil (≤600 px, viewport de referencia 427 × 933) las hojas ocupan el
ancho completo y se anclan abajo: los módulos usan `width:100%` o
`min(…,100%)`, nunca `calc(100vw - N)`, que deja tarjetas flotantes con
márgenes de 8–14 px y descuadra el margen de 16 px. En pantalla completa el pie
queda fijo abajo sin huecos: con `ModalForm`/`ModalBody` el cuerpo crece; un
`form` heredado sin `modal-form` recibe `margin-bottom:auto` en el bloque previo
al pie desde `modal.css`. Un `form` con `padding:0` propio debe usar `ModalForm`:
el form heredado compensa su padding con márgenes negativos en el pie y lo saca
16 px del modal.

`modal-action-layout` homologa pies con grupos anidados: ancho natural y
acciones contiguas en escritorio; flex-wrap con reparto igual y crecimiento
de una acción solitaria en móvil (≤600 px). `modal-action-subgroup` permite
que las secundarias participen de esas filas y `modal-action-main` asigna
fila completa a la principal cuando corresponde. No deducir la distribución
solo del número total de botones: Cancelar + Enviar comanda ya apilados deben
llenar dos filas completas. Botones directos de `ModalFooter` se reparten
el ancho mediante el estilo compartido existente. No aplica a confirmaciones,
feedback ni botones solo de icono.

`modal-inline-metadata` agrupa responsable y fecha fuera del header. En esta
franja etiqueta y valor tienen 14 px y mismo interlineado, con pesos 600/400;
el tamaño común no elimina la jerarquía de peso y color. `modal-inline-date`
alinea fecha/icono al extremo derecho, también si pasan a otra fila por falta
de espacio; `modal-inline-description` ocupa una fila completa para contexto
adicional. Reutilizado en Pedidos/Salón y cualquier modal con esa distribución.
Es una excepción explícita a las etiquetas de 12 px, no un cambio de campos
ni de `modal-record-data`. Su único dueño es `modal.css`.

`ModalWorkspace({children,summary})` es el patrón opcional para un detalle con
resumen real. Usa una sola columna: contenido desplazable y región `.modal-summary`
inferior a ancho completo, persistente en todos los tamaños. No usa panel lateral.
`.modal-summary-lines`, `.modal-summary-line` y `.modal-summary-total` son sus
estilos únicos, no duplicados por módulo. El detalle Pedidos/Salón lo usa en un
panel de hasta 760 px; los formularios sin resumen continúan con `ModalBody`.
Todo el fondo del resumen usa gris muy claro `cloud-50`; el estado abierto usa azul tenue.

`.modal-body` es el único scroll del modal y su barra fina está siempre visible
cuando hay desbordamiento; no combinarlo con `hover-scroll`. El cuerpo nunca desplaza en
horizontal (`overflow-x: hidden`): los tooltips ocultos de `FieldLabel`
(`field-hint`) se anclan al borde derecho del icono y crecen hacia la izquierda.
Los formularios usan `ModalForm` + `ModalBody` + `ModalFooter`; el `<form>`
directo con pie de márgenes negativos es heredado y no se usa en modales nuevos.

Checks, interruptores y radios propios dentro de modales (permisos del editor
de roles `.re-acc-item`, «Obligatorio» de la hoja de menús y combos
`.combo-toggle`, métodos de pago del POS `.pos-method`): el `<input>` nativo se
oculta con `position:absolute`, `opacity:0` y 1 × 1 px, pero su etiqueta declara
`position:relative` para ser su bloque contenedor. Si el input absoluto quedara
referido a `.crud-modal`, el foco al marcar o desmarcar desplazaría el panel
(con `overflow:hidden`): el pie sube y el modal se ve en blanco. Al marcar, el
modal conserva contenido y posición de scroll en escritorio y a 390 px; lo
cubren `tests/role-editor-scroll.test.mjs` y, para todo el CSS de `src/`,
`tests/hidden-input-anchor.test.mjs`.

En móvil (≤600 px) el editor de roles se comporta como la Comanda: tarea a
pantalla completa (100dvh, sin radio) con un solo desplazamiento, el del form.
El cuerpo y el acordeón no tienen scroll propio; el panel usa `overflow:clip`
(conserva el radio sin romper `sticky`). Pestañas Menú/Acciones y búsqueda
quedan fijas arriba al recorrer la lista (`top` compensa el padding del form) y
las pestañas ocultan sus contadores, como todas las pestañas móviles: el conteo
por grupo y los checks ya dicen qué está asignado. Escritorio y tablet conservan modal y contadores, con el mismo
solo desplazamiento: la barra de pestañas, filtro y búsqueda es `sticky` (`top`
compensa los 24 px del form) y el acordeón no tiene scroll propio. Validado a 390 × 844, 427 × 840 y 427 × 933 sin scroll horizontal.
Iconos de las pestañas (todos los anchos): Menú `layout`, Acciones `circleCheck`
(acciones permitidas; reemplaza la llave `key`).

`dl.modal-record-data` organiza los datos de consulta de Ventas, Cobros y el detalle de Menús y combos.
En escritorio es una franja de una fila: fondo `primary-50`, borde `primary-100`,
columnas ajustadas al contenido (`grid-auto-flow: column`, `grid-auto-columns:
auto`) separadas por líneas `primary-200`. El valor de la primera pareja
(identidad) va en 16 px bold `primary-700`; su `small` y los demás valores no cambian. En ≤620 px usa una lista de filas
etiqueta/valor: columna de etiqueta de 120 px, valor flexible, 12 px entre
columnas y 10 px entre filas, alineados a la primera línea del texto.
Los nombres largos envuelven; no se truncan. La lista móvil conserva el fondo de la franja.
La carga usa la misma retícula. El único dueño de este layout es `modal.css`.
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

`Select` con icono (incluido el que `FormField` deduce de la etiqueta) superpone el icono sobre un `<select>` de ancho completo con padding izquierdo: la lista nativa se abre alineada al borde del campo, sin recortarse desde el texto.

`ModalItemList` agrupa líneas de consumo sin encabezados de tabla ni borde exterior.
Cada línea usa `modal-item-row` y `role="listitem"`; su cantidad usa
`modal-item-quantity`, circular, con fondo `primary-100` y texto `primary-700`;
no tiene interacción ni representa un estado. No se repite un conteo de unidades
en el encabezado de productos. Los separadores entre líneas son
`line`. Las notas usan `modal-detail-notes` fuera de esa
lista, con encabezado propio, padding compacto y fondo `cloud-50`, sin borde.

Línea de producto compartida (`modal.css`), obligatoria en los detalles de
Pedidos/Salón, Ventas y POS para que se vean iguales:

| Pieza | Uso |
|---|---|
| `modal-item-info` | Contenedor que se disuelve en la retícula de `modal-item-row`: nombre e importe en la primera fila, el resto a todo el ancho debajo |
| `modal-item-name` / `modal-item-total` | Nombre 14 px / 600 `ink-950` e importe 14 px / 700 tabular, alineados a la cantidad de 32 px |
| `modal-item-meta` | «precio c/u · destino» en 12 px `ink-600`, separados con «·» |
| `ModalItemNote` | Nota del producto como cápsula `cloud-50` con borde `line` e icono `note` |
| `ModalItemComponents` | Elecciones de menús/combos y modificadores como carta: grupo en versalitas, filete vertical `line`, valor y `side` (destino si difiere, estado). En ≤600 px el filete pasa a separadores horizontales. `name` acepta nodos: el detalle de Menús y combos lista cada parte con sus opciones separadas por «·» y recargos en `small` |

Detalle de Menús y combos (patrón de Detalle de pedido): `Dialog` `size-md` con `ModalHeader` (icono `combo`, título = nombre del menú, sin franja ni subtítulo) y `ModalWorkspace`. Arriba, el `Status` real (Activo, Inactivo o «Fuera de fecha») y una línea con icono `calendar` de cuándo se vende («Se vende el …» o días · desde · hasta), con la descripción si existe. «Platos y opciones» (`h3` de 16 px) es una `ModalItemList`: cada parte lleva su `IconTile` (`sm`, mismo tono e icono que en el editor), el nombre de la parte en 12 px y sus platos en 14 px separados por «·», con el costo extra si aplica. El resumen fijo «Resumen del menú» muestra Partes y Opciones, con el Precio como importe principal. Es de lectura: sin `ModalFooter`; la X es el único cierre. No repite reglas de elección («Obligatorio · elige 1»). Error con `ModalError`; la carga usa `Skeleton` con la misma geometría.
La hoja «Arma tu menú o combo» (`ComboEditor`) tampoco lleva subtítulo («NUEVO MENÚ O COMBO») ni borde propio en el panel: el borde dejaba un filo claro alrededor de la cabecera marino. Reutiliza `crud-modal size-lg`, un solo `modal-body` y `ModalFooter`; no es un asistente, así que no lleva stepper, «Continuar» ni paso de revisión. En móvil es pantalla completa con el pie fijo (validado a 427 × 933, 427 × 840 y 390 × 844: margen de 16 px, sin desborde horizontal, controles de 44 px).

Los módulos no redefinen estas piezas; solo añaden su propio contexto (por ejemplo
el destino compartido o el estado de preparación en Pedidos/Salón). Los cuerpos
desplazables con `tabIndex` no muestran contorno azul al recibir foco.
El resumen inferior usa `cloud-50` y un límite superior neutro de 1 px. Estos estilos tienen un solo dueño
en el design system, no se duplican por módulo.

Reglas:

- Datos: `dt` y `modal-data-label` usan 12 px / 600; `dd` y `modal-data-value`, 14 px / 400.
  Títulos de sección 16 px / 700, salvo `modal-section-title` + `modal-section-rule`
  en el cuerpo de Detalle de pedido, venta y cobro (13 px / 700, mayúsculas,
  `ink-800`, regla de 2 px `primary-200`), definidos solo en `modal.css`; el
  título de `modal-summary-heading` usa el mismo estilo y regla. Los datos informativos no heredan el aumento
  móvil a 16 px reservado a controles editables. Los tokens se definen en `variables.css` y los roles
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
- `busy`: bloquea X, «Cancelar» y campos mientras dura la petición; la X conserva color y opacidad.
- Producto reutiliza `ModalForm`, `ModalBody` y `ModalFooter`: cuerpo y pie son hermanos.
  El módulo no redefine padding del formulario/cuerpo ni márgenes o altura de los botones;
  así el pie conserva 24/16 px horizontales y 10 px entre acciones, también al guardar.
- Guardado en curso: pasar `busy` a `ModalHeader` (o `Modal`) añade `ModalBusy`, un velo sin texto que bloquea al instante y atenúa tras 160 ms; si la espera supera 1 s (`--delay-long-wait`) aparece un spinner central sin texto. El único «Guardando…» es el del botón (`loading`). `busyOverlay={false}` queda reservado a la Comanda. Tras guardar, `notify` va antes de cualquier recarga.
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
- `FormDisclosure({title,summary,open,onToggle})`: sección opcional plegable de
  un formulario (Producto › «Información adicional»). Encabezado de ancho completo y alto `--control-height` (44 px en táctil) con
  chevron azul que gira al abrir, título y resumen a la derecha («Opcional» o
  un estado real, p. ej. «Registrada»). Estilo único en `form-controls.css`;
  los módulos no redeclaran el toggle. Dentro de una sección «Opcional» los
  campos no repiten la marca `optional`.
- `LocationMap({latitude,longitude,address,onAddressChange,onChange,onPlaceChange,addressLabel,mapLabel})`
  (`@/design-system/location-map`, Mapbox): buscador con sugerencias, mapa con pin
  arrastrable y aviso «Ubicación definida». Lo usan Locales (etiqueta por defecto
  «Dirección del local») y Clientes › dirección de entrega (`addressLabel` acepta contenido, p. ej. con «Opcional»).
  `onPlaceChange` entrega `{district, city}` de la sugerencia elegida: en Perú Mapbox
  informa el distrito como `place` (o `locality`) y el departamento como `region`, sin
  el prefijo «Departamento de». Se importa directo, no desde el barrel, y carga su CSS.
- `ImageDropTile({src,file,alt,label,dropText,tapText,hint,fit,multiple,removeLabel,onFiles,onRemove})`
  (`@/design-system/image-drop-tile`, se importa directo y carga su CSS): único
  control para elegir imágenes. Recuadro punteado `line-strong` sobre `cloud-25`,
  icono `upload` en círculo `primary-100`, texto distinto para ratón y táctil
  (`hover: none`) y la regla de archivos en `ink-400`; acepta arrastrar y soltar
  (`primary-600` sobre `primary-50`). Con imagen la muestra (`cover` o `contain`);
  `file` presenta el archivo local con `FileImage`, que libera la URL temporal.
- **Estándar para toda imagen editable:** tocar la imagen la reemplaza y
  `onRemove` muestra `ImageRemoveButton`, una X circular de 24 px (área táctil de
  44 px) montada sobre la esquina superior derecha: fondo `surface`, borde
  `line-strong`, sombra `elevation-2` y hover rojo tenue (`danger-50`/`danger-600`).
  Es la única forma de quitar: no se agregan botones «Cambiar» ni papelera.
  `ImageRemoveButton` también se usa sola sobre fichas que no son un
  `ImageDropTile` (páginas de la carta); su contenedor lleva `position: relative`
  y no recorta el desborde. Lo usan Producto › Imagen y Mi restaurante › Logo y
  Carta virtual. Prohibido mostrar el `<input type="file">` nativo.
- `SearchField` para buscar, `Select` para catálogos del API, `Tabs` para
  vistas o segmentos. Sin `<input>`, `<select>` ni pestañas locales.
- Validación: React Hook Form + Zod en `onSubmit` y revalidación `onChange`;
  foco en el primer campo con error. Mensajes: «Ingresa…» o «Selecciona…».
- Solo lectura con fondo `cloud-50` y borde punteado; deshabilitado con
  `cloud-100`.
- Foco de campos compuestos: `ds-input-shell` y `money-input` son dueños del
  único borde/anillo visible. El input/select interior no dibuja un segundo
  rectángulo, tampoco en `form-grid`; foco con error se representa en rojo sobre
  ese contenedor. Los campos independientes y los botones mantienen su foco.

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
  Excepción: las transiciones Preparando/Por entregar de Cocina se confirman con el
  cambio de panel, sin modal de éxito. Los errores sí muestran el diálogo.
  Los avisos de éxito se cierran automáticamente (4,2 s por defecto) o mediante
  Aceptar/X. Errores e información requieren cierre explícito. No se cierran al
  pulsar el fondo ni con Escape.
  La respuesta `success` presenta círculo verde restaurante `brand-600` y check
  blanco `surface` centrado (76/36 px). El panel reserva 64 px sobre el símbolo
  para separarlo de la X y 32 px al pie; título `ink-950` de 18 px / 700,
  explicación `ink-600` de 14 px. «Aceptar» es secundario neutro sobre blanco,
  con borde `line-strong`, sin check duplicado y con `radius-pill` en móvil.
  Se reutiliza la paleta Foods; las referencias externas solo orientan el
  movimiento y no autorizan colores nuevos. El panel se funde en 280 ms y
  permanece inmóvil. `feedback-icon.success:before` contiene la forma verde:
  círculo perfecto con `border-radius: 50%`, que empieza en `scale(1.3)`
  y se contrae hasta `scale(1)` con un rebote leve. Solo anima escala uniforme
  durante 520 ms (`duration-dialog`, `easing-spring`), sin desplazarlo ni
  deformarlo; no añade dependencias ni bucles. El SVG blanco es una capa independiente,
  centrada y sin estirar, que se funde en 200 ms desde los 280 ms.
  El símbolo se monta con la identidad de cada respuesta para reiniciar su
  entrada; los textos no tienen esperas propias.
  Movimiento reducido mantiene un fundido de 160 ms,
  con check y texto visibles sin esperas ni expansión. No simular una carga.
  Tokens y animaciones son propiedad de `providers/styles/feedback.css`;
  no cambian el autocierre, foco ni el callback posterior al cierre.
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
- `DataTable`: componente compartido (`src/design-system/data-table.tsx`), con
  composición Tailwind y estilos propios en `styles/data-table.css`; conserva
  la tabla nativa, el scroll horizontal y la navegación accesible en móvil.
  Cabecera oscura, filas alternas, estados con texto e icono y acciones
  homogéneas. Las acciones por fila son botones de
  icono con la altura de `--control-height`, nombre accesible y tooltip; editar
  usa azul operativo; activar/desactivar usa el interruptor de estado.
  Cancelar y quitar mantienen el rojo semántico. No se presentan como
  enlaces de texto. La columna
  de acciones centra el grupo completo y cada pictograma se centra ópticamente en
  su caja. Como la columna de acciones es la última, sus tooltips
  (`.standard-actions`/`.table-actions`) se anclan al borde derecho del botón y
  crecen hacia la izquierda: centrados, el tooltip oculto del último botón desbordaba
  la tabla y creaba scroll horizontal y un hueco a la derecha (visto en Menús y
  combos con nombres largos). Los demás tooltips siguen centrados.
- El historial de disponibilidad reutiliza `Modal size="xl"` y `ModalBody`: título
  de la consulta en cabecera, producto en el contexto y tabla de cinco columnas
  (fecha/hora, cambio, cantidad, motivo, usuario). Fecha y hora en líneas estables;
  el día operativo solo se muestra si difiere de la fecha local del evento.
  cupo anterior → nuevo y vendidas en su columna propia, sin repetir cifras en
  «Cambio». Solo el cuerpo y la tabla se desplazan; la X es el único cierre.
- Toda columna de acciones de gestión, salvo el listado operativo de
  Disponibilidad especificado abajo, usa exclusivamente `RowActionButton`, construido
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
con tooltip y nombre accesible. Las tablas informativas mantienen sus columnas
con desplazamiento horizontal accesible en pantallas estrechas; no se convierten
automáticamente en tarjetas.

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
- El alta con «Cupo del día» (valor técnico `portions`) exige «Cantidad disponible hoy» y la guarda
  junto al producto. La cantidad sigue perteneciendo al local/día, no al catálogo.
  La edición comercial no modifica cantidades; los ajustes posteriores se hacen
  desde Disponibilidad. Sin control e Inventario físico no muestran ese campo.
- El listado de Productos muestra nombre sin descripción y una columna
  Disponibles con el saldo real del local actual. Cupo del día usa el saldo del día;
  Inventario físico incluye su unidad; Sin control muestra «—». El control de
  cantidad y el estado activo del catálogo permanecen separados del saldo.
- Marcar «agotado hoy» es una acción de operaciones, no del administrador, y se
  presenta como acción reversible del local, nunca como desactivación del catálogo.

### Listado operativo de Disponibilidad

Reutiliza `PageHeader`, `SearchField`, `Select`, `Input`, `IconButton`, `Button`,
`Status` y `Pagination`; su composición pertenece al módulo de disponibilidad.
Foto/icono y nombre forman la identidad, sin líneas de categoría ni control
(redundantes con el filtro y con Cupo de hoy); vendidas y restantes forman un
balance. Nombre y restantes usan 16 px/700; vendidas 14 px/400 y etiquetas
12 px/400. Los iconos son del
set compartido: plato, preparación, inventario, balance, calendario e historial.
Los estados mantienen los tonos establecidos en `02_DESIGN_SYSTEM.md`.

Guardar es un `Button` primario («Guardar», icono `save`) bajo el `Input` de
cupo, a su mismo ancho, y aparece solo cuando cambia el valor; su nombre
accesible identifica el producto. El campo conserva su ancho y oculta las
flechas nativas. El error se vincula al campo mediante
`aria-invalid`/`aria-describedby`; el mínimo es el mayor entre 1 y vendidas,
solo admite enteros. Guardar inválido o durante una mutación queda bloqueado.
Los cambios válidos abren el formulario de motivo existente antes del API.

Agotar hoy usa `Button kind="secondary"`, icono `circleMinus`, texto/borde
`warning` y hover `warning-50`; Reactivar usa `secondary`, `refresh` y azul.
El agotado manual es reversible y no usa el estilo destructivo del catálogo.
Historial usa `Button secondary` con `clock`, tooltip y nombre accesible
descriptivo: icono en escritorio, icono y texto en contenedor ≤1020 px.
En esa composición, dos acciones comparten el ancho y una sola lo llena.
La altura contextual se define en `variables.css`: 44 px en escritorio/tablet,
48 px en móvil. No crear una variante global de estado o botón por esta pantalla.

Sin control muestra la explicación «Sin límite de cantidad» en el espacio de
cupo y balance; los estados derivados identifican su cálculo automático.
Inventario muestra el origen de actualización y el saldo con unidad; null usa
«Sin dato». Los avisos contextuales son una píldora `warning-50` de 2–4 palabras en
la misma línea del estado y con su altura; las aclaraciones bajan a otra línea. El cupo no muestra texto
auxiliar; el botón Guardar visible ya indica el cambio pendiente. Carga, error,
vacío, lectura, paginación y acciones derivadas respetan el contrato existente.
El modal de historial conserva la tabla de cinco columnas descrita arriba;
esta composición no añade cambios a su layout.

## Buscador de artículos de inventario

`InventoryItemLookup` (Inventario) es el único selector de artículos en Registrar
ajuste, Transferir y el filtro del Kárdex. Busca en el servidor
(`inventory/products?q=`, 20 resultados por consulta) con `react-select/async` y
`createLookupSelectStyles`, de modo que ningún artículo queda fuera por la
paginación. Mantiene el artículo elegido completo para mostrar existencia, unidad
y costo promedio sin una segunda consulta. En Transferir excluye los artículos ya
elegidos en otras líneas; en móvil su ancho es fijo para que la tabla no desborde.

El ajuste de entrada pide **Costo unitario** por unidad base. Vacío usa el
promedio actual (el texto de ayuda lo muestra); si el artículo aún no tiene
promedio es obligatorio, y el backend repite esa validación. Las salidas no
muestran el campo. Los cuerpos de Registrar ajuste y Transferir usan
`grid-auto-rows:max-content` para que, al desplazarse, ninguna sección se
comprima.

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

`FullScreenLoader` cubre la pantalla durante la validación de sesión y la carga
inicial de la aplicación; entre pantallas del shell se usa `RouteSkeleton`
(`src/app/(admin)/loading.tsx`), que conserva el menú lateral y la barra superior. Reutiliza el `Logo` oficial y lo anima como algo que hierve,
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

POS conserva un único `Dialog` al cargar y registrar un cobro: `POSPaymentEntry`
mantiene cabecera, backdrop y foco mientras alterna skeleton, error o `PaymentForm`.
El formulario se importa directamente para evitar un segundo montaje o una espera
vacía al descargar su código. Los medios provienen del catálogo; cuando
`affectsCash` está activo se exige «Efectivo recibido» mayor o igual al monto del
cobro y se muestra «Vuelto» calculado, también en pagos parciales. Recibido y vuelto
son ayudas locales: el API recibe solo el monto cobrado para no inflar Caja.
Cobro y devolución usan `ModalForm` → `ModalBody` + `ModalFooter`: el formulario
no añade padding, el cuerpo aplica el margen compartido y es el único scroll.
El pie queda al ras del panel, sin márgenes negativos ni estilos sticky propios;
conserva las acciones visibles en móvil.

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
  En Carta y productos / Categorías, la barra contiene búsqueda por nombre,
  uso y estado; no repite el título y la descripción de la pestaña. Filtra el
  catálogo completo obtenido del API antes de paginar, reinicia a la primera
  página al cambiar filtros y distingue catálogo vacío de falta de coincidencias.
- `Tabs({items,value,onChange,label})`: control segmentado con `role="tablist"`,
  foco itinerante y flechas/Inicio/Fin; icono obligatorio y contador opcional por pestaña; 44 px en
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
se retiró el marcado y CSS de `catalog-tabs`. Usar este componente para cualquier
navegación entre vistas equivalentes, sin recrear estilos por módulo. La pista
`cloud-150` tiene 4 px de margen interior y radio de cápsula, sin sombra ni marco.
Cada opción conserva `--control-height`; la seleccionada usa superficie blanca,
borde `primary-300` y texto `primary-700`. Los contadores usan fondo azul
`primary-600` y número blanco `surface` en todas las pestañas, incluido cero;
su tamaño de 12 px mantiene la etiqueta como contenido principal. El contador
es circular con una cifra y se amplía a cápsula cuando tiene varias cifras.
En móvil (≤600 px) ninguna pestaña muestra contador: la etiqueta usa ese ancho y
solo se parte con guion (`hyphens:auto`) si una palabra no cabe sola.
Cada `TabItem` exige `icon: IconName`: glifo semántico del set compartido de 16 px
antes del texto, sin recuadro propio ni colores locales. Nunca ocultarlo en móvil.
Productos usa `utensils`, Categorías `layers`, Usuarios `users`, Roles `lock`,
Menú de permisos `layout`, Acciones `key`; las demás vistas conservan su icono
semántico de operación. Esta regla aplica también a pestañas sin contador.
Las flechas, Inicio/Fin y foco itinerante permanecen. En móvil las opciones tienen
44 px y se presentan de dos en dos al mismo ancho; si hay más opciones, la pista
se desplaza horizontalmente con gesto o teclado, sin barra visible. `scroll-snap`
ajusta el inicio de cada par: no mostrar fragmentos de las pestañas vecinas al
terminar el desplazamiento. El último grupo impar contiene una pestaña y espacio
neutro, sin traer restos del grupo anterior. Una única pestaña ocupa todo el ancho.
Las etiquetas largas envuelven sin truncarse ni empujar los iconos o contadores.
`TabsBar({children,action})` alinea los tabs a la izquierda y las acciones de esa
vista a la derecha, centrados sobre el mismo eje en escritorio. En móvil, el
listado de tabs ocupa una fila completa y se desplaza si falta espacio, sin que
el botón le quite ancho. Las acciones ocupan la fila siguiente: una llena el
ancho; dos se reparten en columnas iguales y pueden envolver su texto. No poner «Nuevo …» en
`PageHeader` cuando la pantalla tiene tabs: esa acción pertenece a `TabsBar`.
Las acciones conservan su permiso, bloqueo, evento e icono originales. Si hay
varias, su grupo puede repartirse en filas en móvil, sin colocar las acciones
encima de la navegación.
`TabsBar sticky` se reserva para pestañas que filtran una lista larga (hoy, los
canales de Pedidos). En móvil la barra se disuelve (`display:contents`) y solo la
pista queda fija bajo la barra superior, a ancho completo con fondo `cloud-50`
como las zonas del Salón; las acciones quedan debajo y siguen el scroll. En
escritorio no cambia nada. Pestañas que separan secciones no usan `sticky`.

Los modales alinean cabecera, cuerpo y pie sobre un margen común de 24 px en
escritorio y 16 px en móvil. La X y el icono primario comparten una caja de 44 px
y glifos de 20 px; el icono primario no tiene fondo y la X tiene contorno blanco de 32 px,
sin relleno, sombra ni cambio de color con hover. Las notas del pie tienen
una fila propia, por encima de las acciones. Se preservan los tamaños,
el cierre explícito y los contratos de foco/respuesta. No hay franja de marca
en los modales: la cabecera marino ocupa el ancho y el borde superior completos.

En móvil el fondo no añade padding superior a un panel de `100dvh`: eso recortaba
el pie de los modales completos. Las hojas compactas dejan margen superior seguro;
los modales completos suman el área segura a su cabecera y todos mantienen el
área segura inferior del pie. El contenido es el único bloque desplazable.
