# Sistema de diseño administrativo

## Propósito

Este documento es la autoridad visual de foods-admin-web y no depende de
documentos externos. La administración conserva la identidad Foods con mayor
densidad para tablas, formularios y análisis.

## Paleta

| Token | Valor | Uso |
| --- | --- | --- |
| primary-600 | #4654CD | botones primarios y encabezados de tabla |
| primary-700 | #3946B8 | hover de acción primaria |
| primary-100 | #EEF0FF | selección y fondos interactivos suaves |
| navigation-900 | #1A2151 | navegación lateral |
| navigation-text | #C9D0F2 | texto secundario de navegación |
| brand-700 | #16875F | texto y estado exitoso |
| brand-600 | #63DCAE | confirmación y estado exitoso |
| brand-100 | #E8F7F1 | fondos positivos |
| ops-800 | #1A2151 | alias de navegación lateral |
| ops-700 | #3946B8 | enlaces y hover operativo |
| ops-500 | #4654CD | foco y estado activo |
| digital-700 | #5421A8 | WhatsApp e integraciones |
| digital-500 | #7C3AED | acento digital |
| ink-950 | #101828 | texto principal |
| ink-600 | #475467 | texto secundario |
| cloud-50 | #F7F8FC | fondo |
| surface | #FFFFFF | paneles |
| warning | #B54708 | alertas y texto/borde de acciones de advertencia |
| warning-50 | #FFF3E8 | fondo suave y hover de acciones de advertencia |
| danger | #C9362B | errores y acciones destructivas |

Usar superficies mayormente neutras, azul para navegación y acciones, verde para
éxito y violeta para canales digitales.

## Paridad con Operaciones

`foods-admin-web` y `foods-operations-web` comparten exactamente los tokens de
identidad y su semántica. La navegación lateral usa azul oscuro `ops-800`, la selección
azul operativo más luminoso, las acciones primarias y tablas usan `primary-600`, y los
canales digitales usan violeta. El verde se reserva para éxito. Las diferencias entre aplicaciones
se limitan a densidad y composición, nunca a reinterpretar el significado del color.

## Tipografía y forma

- Manrope; Geist Mono para códigos y montos técnicos.
- Radio de 14 px en tarjetas y 6 px en controles.
- Bordes #E4E7EC y sombras discretas.
- Cifras tabulares para KPI, montos y porcentajes.
- Inputs, tablas, botones y estados provienen del design system.

## Layout

- Sidebar: Control, Operación, Carta y producción, Abastecimiento y Configuración.
- El catálogo general se denomina «Carta y productos» y usa el icono de
  cubiertos. «Menús y combos» identifica únicamente productos compuestos;
  «Almuerzo» no se usa como módulo porque excluiría desayuno, cena y otros
  momentos de venta. Los iconos representan el dominio de cada opción y el
  icono de tres líneas se reserva para abrir o cerrar la navegación.
- Topbar compacta; empresa, local y usuario aparecen una sola vez.
- Dashboard separa KPI ejecutivos de alertas operativas.
- Una pantalla de gestión no repite el mismo dato en varios lugares. Si el total
  ya aparece en el tab y el rango en la paginación, no se añaden tarjetas de
  resumen que lo repitan.
- No se muestran métricas acotadas a la página visible («visibles en esta
  página», «inactivos en esta página»): describen lo que el usuario ya ve o
  inducen conclusiones falsas sobre el total. Un KPI debe ser global y accionable.
- Gestión: título, acciones, filtros, listado, paginación y detalle.
- En móvil, tablas pasan a tarjetas o usan scroll controlado.
- Los productos pueden llevar una imagen opcional (foto del plato, bebida,
  etc.). La imagen se sube después de crear el producto; al editar, se puede
  cambiar o quitar. Acepta PNG, JPEG y WebP hasta 5 MB. Se almacena en
  `/uploads/products/{id}.{ext}` y se sirve desde el backend.
- El formulario de producto incluye campos opcionales: tiempo de
  preparación (minutos), alérgenos (tags seleccionables), plato destacado
  (switch) y precio de costo. Ninguno es obligatorio; el operador puede
  registrar un producto solo con nombre, precio y categoría.
- Producto separa `productType` (`prepared` o `retail`) de categoría y
  `quantityControl`. La categoría sigue siendo comercial y puede mezclar, por
  ejemplo, una limonada preparada y una gaseosa de reventa dentro de Bebidas.
  Los productos creados desde Carta y productos nacen como `prepared`; el
  atajo Inventario > Nuevo producto vendible clasifica automáticamente como
  `retail` sin pedir un paso adicional al usuario. El tipo no se repite como
  un campo de solo lectura porque ya está expresado por la opción seleccionada.
  El alta rápida sí exige categoría, pero el selector consulta únicamente
  categorías cuyo `productScope` sea `retail` o `both`; nunca muestra
  categorías exclusivas de platos preparados.
- El registro y la edición de producto usan un wizard de tres pasos dentro del
  mismo modal: Información, Operación y Presentación. Avanzar no persiste datos;
  el producto se envía una sola vez desde el último paso. Cada paso valida solo
  los campos que contiene, conserva el estado al retroceder y muestra un resumen
  antes de guardar. En móvil el wizard ocupa la pantalla completa y mantiene las
  acciones de navegación visibles.
- La pantalla se denomina «Disponibilidad de la carta» y pertenece a «Carta y
  producción», porque incluye productos, platos, bebidas, menús y combos; no
  se presenta como disponibilidad de un único menú. La disponibilidad cotidiana
  no se modifica dentro del wizard. Usa una vista
  operativa separada por local con tarjetas compactas, estado textual, cupo y
  filas operativas continuas en escritorio, sin bordes de tarjeta repetidos, y tarjetas apiladas únicamente en
  móvil, con una acción manual Disponible/Agotado y guardado explícito del cupo. El botón de actualización de cupo permanece visible para hacer descubrible la función, pero
  solo se habilita después de modificar la cantidad. En filas densas, las acciones usan
  etiquetas visibles breves y específicas («Guardar cupo», «Agotar hoy», «Reactivar») y conservan un
  `aria-label` descriptivo completo. «Agotar hoy» usa `warning` para texto y borde y
  `warning-50` únicamente como fondo suave en hover; nunca usa rojo destructivo ni
  invierte a un relleno ámbar sólido. Los avisos operativos de agotamiento dentro de
  la fila se muestran como una franja `warning-50`, con icono/texto `warning` y acento
  lateral de advertencia; no se presentan como texto suelto sobre fondo neutro. Las
  subcolumnas Control, Cupo, Vendidas y Restantes comparten alineación vertical estable
  y el nombre del tipo de control no se trunca cuando hay espacio suficiente. «Pocas
  unidades» es siempre un estado calculado por las unidades restantes, nunca
  una acción manual. La vista se
  pagina desde el API, permite buscar y filtrar por categoría, diferencia cupo,
  vendidos y restantes, y bloquea acciones manuales cuando el estado se deriva
  del horario o de componentes obligatorios de un menú.
- El producto solicita explícitamente el control de disponibilidad mediante
  tarjetas radio: «Siempre disponible» o «Cupo diario». «Siempre disponible»
  no muestra un contador; «Cupo diario» revela y exige un entero mayor que cero
  bajo la etiqueta «Cupo diario predeterminado». La interfaz nunca infiere esta
  intención únicamente porque el usuario dejó una cantidad vacía.
- Los menús y combos usan un wizard independiente de cuatro pasos: información,
  composición, disponibilidad y revisión. La plantilla «Menú del día» prepara
  Entrada, Plato principal, Postre y Bebida, pero cada parte puede editarse. En la interfaz,
  «Partes del menú» identifica esos momentos del menú. El listado no muestra un
  total global de «opciones», porque mezclar entradas, platos principales,
  bebidas y postres en una sola cifra es ambiguo. El detalle presenta cada parte
  con sus productos elegibles; así se ve claramente, por ejemplo, que existen
  dos platos principales sin contabilizar la bebida como si fuera otro plato. Las opciones se seleccionan
  entre productos existentes; no se duplican fichas ni controles de disponibilidad.
  En composición se muestra siempre obligatoriedad, mínimo/máximo y recargo por
  opción. La disponibilidad del menú se deriva de las partes obligatorias.
- El wizard de menús y combos no solicita categoría de producto. Cada alternativa
  conserva su propia categoría; el producto técnico que representa al menú se
  registra sin `categoryId` para no mezclar clasificación de platos con composición.
- Los alérgenos provienen del backend (`GET /v1/admin/allergens`) y se
  seleccionan mediante un control `react-select` asíncrono (AsyncSelect) con
  multi-select. El filtrado lo hace el backend con `?q=`, no el frontend. No
  se hardcodean alérgenos en el código.
- El control de alérgenos respeta el mismo diseño que los demás inputs:
  `--control-height` de altura, borde `#e4e7ec`, radio de 6 px, fuente de
  11 px, sin outline ni box-shadow al hacer focus, y placeholder alineado a
  la izquierda.
- Las mesas se registran en una tabla con filas editables y botón `+` para
  agregar múltiples mesas en un solo proceso. No se usa modal para crear
  mesas. El guardado envía todas las filas válidas en un solo `POST` batch.
- Las zonas (Terraza, Salón, Barra, etc.) se administran en un tab dentro de
  la página de Mesas, con su propio CRUD. El campo Zona al crear/editar mesas
  es un select que carga las zonas activas del API, no un input libre.
- La configuración presenta la jerarquía Empresa → Perfiles por país → Locales.
  País, moneda e impuesto se editan en el perfil fiscal de la empresa; cada local
  selecciona un perfil existente. Los tipos de cambio se gestionan en una vista
  separada con moneda origen, moneda destino, vigencia, fuente e historial. No se
  mezclan estos conceptos en un único formulario ni se duplican valores por local.
- El onboarding de plataforma se presenta como un wizard de cuatro pasos:
  Empresa → Fiscal → Local → Administrador. La confirmación final crea el tenant
  completo de forma transaccional; avanzar entre pasos no persiste datos parciales.

## Componentes y estados

- Usuarios y permisos se separan en dos pestañas. Un usuario puede tener varias asignaciones rol–local. El editor de rol separa claramente «Accesos al sistema» (opciones del menú) de «Permisos de acción» (operaciones dentro de cada pantalla); nunca se presentan como un único concepto. El Administrador `[*]` permanece protegido, los demás roles predeterminados permiten adaptar accesos y permisos conservando nombre y descripción, y los roles personalizados permiten editar toda su definición. La activación se realiza desde la tabla, nunca dentro del formulario.

- Área táctil mínima de 44 por 44 px en móvil.
- Skeleton con forma final: el de tablas reproduce cabecera y filas con
  celdas alineadas a las columnas reales (incluyendo icono, título y subtítulo
  en la primera celda), no barras o cuadrados planos.
- Estados con texto e icono; nunca solo color.
- Contraste WCAG AA y foco visible con ops-500.
- Alertas de éxito, error e información usan un diálogo global centrado, icono
  semántico, título, explicación breve y acción «Aceptar». El botón «Aceptar»
  usa `--control-height` como todos los demás botones. El comportamiento y
  jerarquía siguen el estándar de Bodegas, reinterpretado con tokens Foods.
- Confirmaciones destructivas son diálogos independientes: explican el impacto,
  conservan «Cancelar» como acción segura y bloquean la repetición durante el envío.
- En todo modal de registro o edición, el botón de guardado solo dice
  «Guardar» (sin sufijos como «producto», «categoría», «mesa» o «zona», y sin
  variantes como «Guardar cambios»). El estado ocupado dice «Guardando…».
  Esta regla aplica a productos, categorías, zonas y cualquier otro registro.
- Los botones «Guardar» y «Cancelar» de un mismo footer tienen idéntica altura
  (`--control-height`). El ícono de «Guardar» es un check (✓), no un disquete.
- La cabecera de los modales de registro es compacta: padding 11×16 px,
  altura mínima 58 px, título h2 a 15 px, ícono de título 32×32 px y botón
  cerrar 36×36 px. No usar cabeceras altas ni íconos grandes en formularios.
- **No duplicar controles de estado dentro del modal de edición.** La
  activación y desactivación de cualquier registro (producto, categoría, etc.)
  se maneja exclusivamente desde el botón de acción de la tabla. Los
  formularios de edición no incluyen switches de estado activo/inactivo.
- Notificaciones viven en un popover compacto desde la campana, con contador,
  categoría visual, tiempo y enlace al historial; no usan mensajes flotantes aislados.
- El éxito de un registro se comunica mediante la alerta global después de que la
  API confirme la operación. Nunca se anticipa éxito ni se pierde el error remoto.

## Tablas y paginación

- Cabecera azul `primary-600`, texto blanco en mayúsculas, filas alternas sutiles y
  acciones textuales consistentes; el color semántico se reserva para estados.
- La paginación informa el rango visible y total, permite 10, 20 o 50 filas y
  muestra páginas, elipsis, anterior y siguiente con estado activo inequívoco.
- Toda tabla de gestión incluye paginación, incluida la de categorías.
- La paginación es una primitiva única y compartida. No se replica su cálculo,
  marcado ni estilos dentro de los módulos de negocio.
- El botón «Nuevo» va alineado a la derecha de los tabs de producto/categoría,
  no en el encabezado de página. En móvil se apila debajo de los tabs.
- El scroll horizontal solo aparece si hay desborde y se revela al pasar el cursor
  o enfocar el contenedor. En móvil se prioriza tarjeta o scroll controlado.
- Todos los formularios usan los mismos labels, inputs, selects, áreas de texto,
  foco azul operativo y estados ocupado/deshabilitado.
- **Altura única de control.** Todo control interactivo de una misma barra o
  formulario mide exactamente lo mismo: buscadores, selects, botones de filtro,
  botones de acción, botones de paginación y botones de modal usan
  `--control-height` (38 px en escritorio, 44 px en móvil). Ningún botón usa
  alturas fijas distintas; un valor literal fuera de `--control-height` se
  considera defecto.
  botones de icono, acciones de fila y controles de paginación. La altura se
  declara siempre con el token `--control-height` y nunca con un valor fijo.
- `--control-height` vale 36 px en escritorio y se redefine a 44 px en
  `@media(max-width:600px)` para conservar el área táctil. Un solo cambio de
  token reajusta toda la aplicación; no se permiten overrides `!important`
  por control ni excepciones por pantalla.
- Los controles cuadrados de icono usan `--control-height` en ancho y alto para
  alinearse con los campos vecinos. Los inputs anidados dentro de un contenedor
  con borde usan `calc(var(--control-height) - 2px)`.
- El radio de los controles es 6 px de forma uniforme; 14 px se reserva para
  tarjetas y paneles.
- En escritorio, los campos siguen la densidad de Bodegas: 36 px de altura,
  radio de 6 px, borde neutro y texto de 11 px. En móvil aumentan a 44 px para
  conservar el área táctil. El foco usa anillo azul Foods y el error rojo semántico.
- Los inputs compuestos con prefijo (montos, «S/») declaran el borde en el
  contenedor, no en cada parte; el prefijo y el input interno no duplican ni
  omiten bordes. El foco se aplica al contenedor con `:focus-within`.

## Responsive

Toda pantalla se valida desde 390 px. En móvil se reorganizan filtros, acciones
y detalle; no se limita a apilar columnas de escritorio.

## Configuración financiera

- Moneda e impuesto se presentan en dos paneles gemelos de la rejilla estándar,
  con igual cabecera, padding, columnas y altura visual. Una franja inferior
  resume el cálculo sin competir con los campos editables.
- La moneda procede del catálogo del API. Símbolo y decimales son derivados y
  no se editan manualmente. El selector expone el catálogo ISO 4217 activo
  completo; ningún frontend conserva una lista local o parcial.
- El país se selecciona antes de la moneda, pertenece a la empresa y usa Perú
  como valor inicial. Cambiarlo sugiere la moneda oficial, pero no modifica la
  tasa fiscal sin confirmación del administrador.
- El impuesto se captura como porcentaje humano (por ejemplo, `18`), aunque el
  contrato lo persista como razón decimal (`0.18`).
- La vista previa aclara formato, cálculo y efecto de incluir el impuesto, y se
  advierte que los comprobantes emitidos no cambian retroactivamente.

## Menú

- Fondo `ops-800`, opción activa `ops-500` e iconos contenidos en una caja estable.
- El hover modifica color y superficie sin desplazar ni escalar elementos.
- El drawer móvil bloquea el fondo, cierra con Escape, overlay o navegación y
  conserva scroll interno.
- Apertura y cierre duran entre 160 y 240 ms; movimiento reducido elimina la animación.

## Acción principal

El botón primario azul `#4654CD` mide 48 px de alto, usa padding horizontal de 20 px,
texto de 13/20 px y radio de 10 px. Esta mayor presencia se reserva para la
acción dominante; botones secundarios e iconográficos mantienen su geometría.

## Acceso administrativo

El login usa una composición centrada y compacta: marca superior, tarjeta de un
solo formulario con línea de acento índigo, título y ayuda centrados, controles
de 44 px, acción principal a todo el ancho e indicadores de confianza al pie.
En móvil aumenta los controles a 48 px y reduce únicamente el padding; no cambia
el orden ni oculta información funcional. La referencia define la geometría,
pero la identidad, textos, iconos y colores son exclusivamente Foods.

## Integridad CSS

- Un selector se declara una sola vez dentro del mismo contexto CSS.
- Un selector tampoco se repite entre hojas importadas por el mismo contexto:
  cada clase tiene un único dueño. La prueba detecta duplicados tanto dentro
  de `globals.css` como entre `globals.css`, `navigation-state.css`,
  `loading.css` y `table-actions.css`.
- `table-actions.css` es el único dueño de los estilos de acciones
  iconográficas de tablas y tarjetas: `.table-actions`, `.table-actions button`,
  `.table-actions button.danger`, `.ds-icon-button`, `.ds-icon-button:hover`
  y `.standard-actions`. `globals.css` solo conserva reglas complementarias
  no duplicadas (como `.ds-icon-button svg`, `:focus-visible` y `:disabled`).
- Los tooltips reutilizan el patrón `[data-tooltip]` de `table-actions.css`;
  no se añaden reglas de tooltip en otras hojas.
- No se permiten bloques posteriores que reescriban una clase para corregir
  estilos anteriores.
- Las diferencias de estado, tamaño o dispositivo usan clases modificadoras,
  atributos o `@media` explícitos.
- La prueba `tests/css-duplicates.test.mjs` protege esta regla.
- Las dimensiones de control no se escriben en píxeles literales. Cualquier
  `height` o `min-height` de un input, select, botón o control de paginación usa
  `--control-height`; un valor fijo se considera defecto.

## Carga remota

- Toda lectura remota muestra un skeleton con la geometría aproximada del
  contenido final; tablas, tarjetas y formularios no usan un spinner aislado.
- El skeleton de tabla reproduce la rejilla de columnas: una fila de cabecera
  con barras cortas y filas de cuerpo con celdas alineadas, incluyendo icono,
  título y subtítulo en la primera columna. Nunca se renderiza como un bloque
  o barra única sin estructura.
- Toda mutación muestra estado ocupado en la acción que la originó, evita envíos
  duplicados y conserva visible el contexto de la pantalla.
- Después de cargar se presenta contenido, vacío accionable o error recuperable.
## Directorios maestros

Los módulos de directorio, como Clientes, usan la tabla estandarizada en escritorio y tarjetas equivalentes en móvil. Toda consulta remota debe incluir skeleton, error recuperable, vacío contextual y paginación compartida. Las acciones de fila son únicamente iconos homologados con tooltip; alta y edición usan las primitivas `Input`, `Select`, `Textarea`, `Button`, `Status` y `ConfirmDialog`. Los formularios extensos se agrupan por secciones visuales, sin añadir texto explicativo que no ayude a completar la tarea.
