# Sistema de diseño administrativo

## Propósito

Este documento es la autoridad visual de foods-admin-web y no depende de
documentos externos. La administración conserva la identidad Foods con mayor
densidad para tablas, formularios y análisis.

## Paleta

La autoridad de valores está en `src/styles/variables.css`. `globals.css` la
importa una sola vez, después de Tailwind, y el layout raíz carga esa base para
todas las rutas. Las hojas de componentes y módulos consumen `var(...)` y
conservan la propiedad de sus selectores.

La hoja se organiza en paleta, matices, transparencias, alias semánticos,
espaciado, dimensiones, tipografía, radios, bordes, sombras y movimiento.
`--space-12` representa el paso actual de 12 px; modificarlo actualiza todos
sus usos. Espaciado, dimensiones y tipografía tienen escalas independientes
para poder ajustar la densidad sin alterar el tamaño del texto o los iconos.
Los matices secundarios conservan los colores existentes; los nombres
semánticos (`primary`, `brand`, `danger`, etc.) son la API recomendada.
Las transparencias derivan de su color base y los alias apuntan al token
canónico: no se vuelve a escribir el mismo hexadecimal.

Los contratos compartidos incluyen `--radius-control`, `--radius-card`,
`--font-size-label`, `--font-size-control`, `--control-height` y los tokens
de animación del modal. Los valores particulares por contexto y la altura
táctil móvil también se declaran en esa hoja. Las condiciones de media queries
y container queries conservan sus medidas literales, porque CSS no admite
custom properties en esas condiciones.

Los selectores de búsqueda de Recetas y Países reutilizan
`createLookupSelectStyles`, con extensiones explícitas cuando corresponde.
Canvas y Mapbox obtienen los colores mediante `readCssToken`, porque esas
APIs necesitan un valor resuelto. Las fuentes se resuelven en `body`, donde
Next proporciona sus variables de fuente.

`tests/design-tokens.test.mjs` impide redefinir tokens fuera de esta hoja,
duplicar colores, introducir valores de diseño literales en las hojas de
componentes o usar referencias inexistentes o circulares. La prueba de tamaño
mínimo de texto resuelve las variables antes de revisar las medidas.

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
| digital-100 | #F2ECFF | fondo violeta suave para estados operativos |
| ink-950 | #101828 | texto principal |
| ink-600 | #475467 | texto secundario |
| cloud-50 | #F7F8FC | fondo |
| surface | #FFFFFF | paneles |
| warning | #B54708 | alertas y texto/borde de acciones de advertencia |
| warning-50 | #FFF3E8 | fondo suave y hover de acciones de advertencia |
| danger | #C9362B | errores y acciones destructivas |

Usar superficies mayormente neutras, azul para navegación y acciones, verde para
éxito y violeta para canales digitales.

La composición prioriza blanco y azul índigo, con verdes como acentos de
identidad y estados positivos. Repetir pocos colores con funciones estables:
una acción dominante de fondo sólido por contexto, acciones secundarias sobre
blanco con borde neutro y fondos tintados suaves para selección o estado. El
menta aporta frescura en detalles, no en grandes superficies que compitan con
la acción principal. Ámbar y rojo se reservan para advertencias y errores;
violeta para canales digitales, no para diferenciar acciones de cobro. Los
avisos informativos usan texto neutro e icono, sin apariencia de botón.

## Identidad operativa

`foods-admin-web` concentra la experiencia web operativa y administrativa. La navegación lateral usa azul oscuro `ops-800`, la selección azul operativo más luminoso, las acciones primarias y tablas usan `primary-600`, y los canales digitales usan violeta. El verde se reserva para éxito.

## Tipografía y forma

- Manrope; Geist Mono para códigos y montos técnicos.
- Radio de 14 px en tarjetas y 6 px en controles.
- Bordes #E4E7EC y sombras discretas.
- Cifras tabulares para KPI, montos y porcentajes.
- Tamaño mínimo de texto en toda la aplicación (módulos, `globals.css` y
  design system): 10 px. Los eyebrows y etiquetas auxiliares usan 10 px;
  campos y botones usan 13 px en escritorio; labels 12 px; cuerpo 14 px.
  Los controles usan 16 px en móvil para mejorar lectura y evitar zoom al enfocar.
  La prueba
  `tests/type-scale.test.mjs` bloquea cualquier `font-size` menor.
- Inputs, tablas, botones y estados provienen del design system.

## Layout

- Sidebar: Control, Operación, Carta y producción, Abastecimiento y Configuración.
- El catálogo general se denomina «Carta y productos» y usa el icono de
  cubiertos. «Menús y combos» identifica únicamente productos compuestos;
  «Almuerzo» no se usa como módulo porque excluiría desayuno, cena y otros
  momentos de venta. Los iconos representan el dominio de cada opción y el
  icono de tres líneas se reserva para abrir o cerrar la navegación.
- Topbar compacta; empresa, local y usuario aparecen una sola vez.
- El selector de empresa/local usa un popover compacto con cabecera y cierre X,
  campos `FormField`/`Select` y acción «Aplicar» azul `primary-600`; no emplea
  un botón verde propio. Etiquetas en caja normal, espaciado de 16 px y altura
  única de control. Su carga, error, vacío y adaptación móvil se especifican
  en `03_LAYOUTS.md`.
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
- El registro y la edición de producto usan un formulario con información adicional
  desplegable. La cabecera y el footer con «Cancelar» y «Guardar» permanecen
  visibles; solo los campos tienen desplazamiento, también en móvil desde 390 px.
  El envío requiere activar explícitamente «Guardar» mediante clic, toque o teclado
  con el botón enfocado. Enter desde los campos no guarda el producto y conserva
  su función en la descripción y los selectores. Una sola solicitud puede estar
  en curso, incluida la validación previa; ante un error se permite corregir y reintentar.

- La pantalla se denomina «Disponibilidad de la carta» y pertenece a «Carta y
  producción», porque incluye productos, platos, bebidas, menús y combos; no
  se presenta como disponibilidad de un único menú. La disponibilidad cotidiana
  no se modifica dentro del wizard. Usa una vista operativa separada por
  local con una retícula de columnas fijas compartida por cabecera y filas:
  Producto, Estado, Control, Cupo de hoy, Vendidas, Restantes y Acciones. La
  cabecera es la estándar de tablas (`primary-600`, 10 px/800) y las filas son
  continuas, sin bordes de tarjeta ni etiquetas repetidas por fila; en móvil
  cada fila pasa a tarjeta y las etiquetas de columna aparecen mediante
  `data-label`. Las columnas no cambian según el tipo de control: cuando un
  valor no aplica se muestra «—». Control muestra solo el tipo («Porciones
  preparadas», «Inventario físico», «Sin control»); en inventario, la
  aclaración de que la existencia se actualiza desde Inventario y con las
  ventas vive en un tooltip, no en una franja. Restantes muestra las porciones
  restantes o la existencia física con su unidad. El estado lleva una
  aclaración debajo solo cuando aporta («Agotado manualmente», «Calculado
  automáticamente»). La barra de filtros contiene únicamente búsqueda y
  categoría. La acción manual es Disponible/Agotado con guardado explícito del
  cupo. El botón de actualización de cupo permanece visible para hacer
  descubrible la función, pero solo se habilita después de modificar la
  cantidad. Las acciones usan etiquetas visibles breves y específicas
  («Guardar», «Agotar hoy», «Reactivar») y conservan un `aria-label`
  descriptivo completo; «Guardar» nunca se parte en dos líneas. «Agotar hoy»
  usa `warning` para texto y borde y `warning-50` únicamente como fondo suave
  en hover; nunca usa rojo destructivo ni invierte a un relleno ámbar sólido.
  Los avisos operativos (fuera de horario, componentes obligatorios, cupo o
  stock agotado) ocupan una segunda fila completa como franja `warning-50` con
  icono/texto `warning` y acento lateral; no se presentan como texto suelto
  sobre fondo neutro. «Pocas
  unidades» es siempre un estado calculado por las unidades restantes, nunca
  una acción manual. La vista se
  pagina desde el API, permite buscar y filtrar por categoría, diferencia cupo,
  vendidos y restantes, y bloquea acciones manuales cuando el estado se deriva
  del horario o de componentes obligatorios de un menú. «Guardar» modifica únicamente
  el cupo escrito; «Agotar hoy» y «Reactivar» modifican únicamente el override manual
  y nunca persisten un cupo pendiente de guardar. El cupo mínimo editable es el mayor
  entre 1 y las porciones ya vendidas. Un usuario con `menu.read` pero sin `menu.manage`
  ve la pantalla en modo solo lectura: conserva filtros, estados y cantidades, pero no
  puede editar cupos ni ejecutar acciones.
- Cocina usa un KDS de tres carriles operativos: «Por preparar», «En preparación»
  y «Listos para entregar». Cada carril es un panel neutro `cloud-50/surface`
  con borde estándar; el color semántico se limita al acento de estado, icono,
  contador y etiquetas: azul para «Por preparar», violeta para «En preparación»,
  verde para «Listo» y ámbar exclusivamente para «Por vencer» o «Con demora». El cuerpo del carril permanece neutro y la cabecera completa usa un fondo semántico de contraste medio/alto: `primary-600` para «Por preparar», `digital-500` para «En preparación» y `brand-600` para «Listo». Azul y violeta usan título/descripción en `surface`; el verde `brand-600` usa `ink-950` para el título y `ink-600` para la descripción, evitando que el texto se pierda sobre el fondo menta. Icono y contador se apoyan en `surface` y conservan `brand-700` como acento. Esta diferencia cromática debe ser evidente incluso al escanear la pantalla rápidamente. No se añaden franjas superiores ni barras laterales de color. Los nombres de mesa se muestran en mayúsculas (`MESA 1`, `MESA 2`) para facilitar el escaneo; nombres de clientes y otros identificadores conservan su escritura original. Los botones que cambian el flujo son
  acciones primarias azules; el verde no se usa como CTA antes de confirmar el
  éxito. Las tarjetas son densas, con borde `line`, radio de 14 px y sin
  sombras o transformaciones decorativas que compitan con la información.
  `danger` no se usa para retrasos de cocina porque se reserva para errores y
  acciones destructivas. El tiempo nunca se muestra como minutos de cuatro cifras: se formatea como
  minutos, horas/minutos o días/horas según corresponda. La urgencia siempre
  incluye texto explícito («A tiempo», «Por vencer», «Con demora», «Listo»);
  nunca depende solo del color. El objetivo de preparación aparece como dato
  secundario. El skeleton reproduce cabecera, identidad, líneas de productos y
  acción de cada tarjeta, no bloques rectangulares genéricos.
- El formulario de Recetas usa un modal de edición compacto y estructurado en dos
  zonas: datos de la receta e insumos. La cabecera sigue el patrón estándar de
  58 px mínimo, icono de 32 px, título de 15 px y cierre con
  `--control-height`. El producto y el rendimiento son los datos principales;
  «Producto preparado» usa el autocomplete asíncrono compartido basado en
  `react-select/async`, con selección única, `cacheOptions` y el mismo patrón visual
  `react-select-container` usado en los demás formularios. Las opciones muestran solo
  el nombre del producto, sin SKU. Al abrir muestra como máximo 10 productos; con 1 o
  2 caracteres no consulta el backend y solicita completar al menos 3; desde 3
  caracteres consulta el API y reúne todas las coincidencias de la búsqueda, recorriendo
  la paginación del endpoint cuando sea necesario. No se reemplaza por un `<select>`
  nativo ni se descarga el catálogo completo antes de que el usuario busque.
  Las notas son opcionales y no dominan visualmente el formulario. La composición
  de insumos vive en una sección propia con cabecera azul `primary-600` en
  escritorio, columnas Insumo, Cantidad, Merma y acción homologada de quitar.
  Cada fila de Insumo usa el mismo patrón de autocomplete asíncrono: al abrir
  muestra hasta 10 artículos de inventario, con 1 o 2 caracteres no consulta,
  y desde 3 caracteres reúne todas las coincidencias paginadas del API. Las
  opciones ya usadas por otras filas se excluyen. «Agregar insumo» crea una
  fila vacía y no depende del tamaño del catálogo cargado, por lo que no existe
  un máximo artificial de 5, 10 o 100 insumos; el límite funcional es únicamente
  no repetir el mismo artículo dentro de una receta. «Agregar insumo» pertenece
  a la cabecera de esa sección, nunca al footer. El
  footer contiene únicamente «Cancelar» y «Guardar», con la misma altura. El
  modal no incluye selector de Estado: activar o desactivar una receta se realiza
  desde la tabla mediante la acción de fila correspondiente. En móvil, el modal
  ocupa la pantalla y cada insumo se reorganiza como una tarjeta de edición sin
  perder unidad, cantidad, merma ni acción de quitar. Cantidad y unidad forman
  un único control compuesto, con un solo borde y foco en el contenedor. La
  columna Acción reserva el ancho completo del control cuadrado y nunca recorta
  el botón de quitar. El formulario de Recetas no hereda padding ni iconos
  decorativos del footer CRUD genérico: su espaciado pertenece al módulo y cada
  acción muestra un único icono explícito.
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
  `--font-size-control`, anillo de foco azul compartido y placeholder alineado a
  la izquierda.
- Las mesas se registran en una tabla con filas editables y botón `+` para
  agregar múltiples mesas en un solo proceso. No se usa modal para crear
  mesas. El guardado envía todas las filas válidas en un solo `POST` batch.
  **La edición de una mesa existente también ocurre en su misma fila de tabla**:
  Nombre, Zona y Asientos se convierten en controles del design system y la
  columna Acciones muestra Guardar/Cancelar. No se abre modal de edición ni se
  desplaza al usuario fuera del listado. Estado activo/inactivo se cambia solo
  mediante la acción de fila correspondiente; no se duplica como switch dentro
  de la edición. La configuración del QR permanece como una capacidad separada
  de la edición de datos básicos de la mesa.
- El QR impreso de cada mesa abre directamente Fudia Concierge en WhatsApp por
  medio de `/api/public/concierge/:qr`. No se muestra una pantalla pública
  intermedia con acciones duplicadas. `/table/:qr` existe únicamente como
  redirección técnica para conservar la compatibilidad de QR ya impresos.
- Las zonas (Terraza, Salón, Barra, etc.) se administran en un tab dentro de
  la página de Mesas, con su propio CRUD. El campo Zona al crear/editar mesas
  es un select que carga las zonas activas del API, no un input libre.
- La pantalla Empresa no agrega una tarjeta resumen que repita Razón social,
  Nombre comercial, Identificación fiscal o Zona horaria ya presentes en el
  formulario. Cada dato aparece una sola vez; el estado de la empresa puede
  vivir en la cabecera de la sección legal porque no se edita en ese formulario.
- Kárdex reutiliza exactamente el patrón de gestión de Inventario: `PageHeader`,
  un único panel `standardized-management`, barra compacta de filtros, tabla
  estándar y paginación compartida. La primera columna es Artículo y usa el mismo
  patrón visual de entidad que Inventario; Fecha y trazabilidad son atributos del
  movimiento. En móvil la tabla tiene tarjetas equivalentes mediante
  `management-cards`; nunca desaparece el contenido al ocultarse la tabla.
  No se agrega una segunda cabecera de resultados ni se repite el total si la
  paginación ya informa rango y total.
- Punto de venta (`/pos`) usa un único panel de gestión con búsqueda y filtro de
  estado. En escritorio presenta tabla estándar y en móvil tarjetas equivalentes;
  la acción de cobrar es iconográfica, usa `RowActionButton` y muestra tooltip,
  igual que el resto de acciones de fila. El turno de caja se presenta como
  contexto compacto y no como una segunda cabecera de página.
- El skeleton del POS conserva la geometría de cabecera, columnas, filas y
  acciones; el detalle reproduce resumen, productos y pagos. No se reemplaza
  por barras genéricas ni por un spinner aislado.
- Los modales de cobro y devolución usan la cabecera modal estándar, `FormField`
  con primitivas del design system y pie de acciones consistente. La mutación
  mantiene visible el contexto y muestra el estado ocupado en el botón que la
  originó. El selector de método de pago conserva una sola línea, un indicador
  de radio visible y foco accesible. El verde se reserva para cobros en efectivo
  confirmados o mensajes de éxito; selección y saldo usan azul operativo.
- El detalle de cobro es de consulta: se cierra únicamente con la `X` de la
  cabecera y no repite la acción general de cobro. Registrar un cobro se inicia
  desde la acción iconográfica de la tabla; la devolución permanece junto al
  pago concreto porque requiere ese contexto. Nunca se agrega un botón
  redundante «Cerrar» en el pie.
- «Cobrar» en Salón y «Cobrar saldo» en Pedidos abren directamente el formulario
  de pago del POS mediante `/pos?orderId=...`, no el detalle de consulta. La acción
  de cobro de la tabla usa ese mismo formulario. Antes de habilitarlo se consulta
  el saldo actual del pedido y se valida permiso, estado cobrable y turno asignado.
  La espera reproduce saldo, métodos, monto y acciones con skeleton. Falta de
  turno ofrece «Ir a Caja»; errores de pedido/caja ofrecen «Reintentar»; una cuenta
  pagada o cerrada no permite otro cobro. El formulario conserva «Cancelar» y
  «Registrar cobro», bloquea cierre/envíos mientras registra y guarda únicamente
  tras la acción explícita del usuario. El ojo de la tabla sigue abriendo el
  detalle sin añadir otro botón de cobro.
- La configuración presenta la jerarquía Empresa → Perfiles por país → Locales.
  País, moneda e impuesto se editan en el perfil fiscal de la empresa; cada local
  selecciona un perfil existente. Los tipos de cambio se gestionan en una vista
  separada con moneda origen, moneda destino, vigencia, fuente e historial. No se
  mezclan estos conceptos en un único formulario ni se duplican valores por local.
- El onboarding de plataforma se presenta como un wizard de cinco pasos:
  Empresa → Plan y contrato → Fiscal → Primer local → Administrador. La confirmación final crea el tenant
  completo de forma transaccional; avanzar entre pasos no persiste datos parciales.

## Componentes y estados

- Usuarios y permisos se separan en dos pestañas. Un usuario puede tener varias asignaciones rol–local. El editor de rol separa claramente «Accesos al sistema» (opciones del menú) de «Permisos de acción» (operaciones dentro de cada pantalla); nunca se presentan como un único concepto. El Administrador de plataforma es el único `[*]`: ve todo el catálogo de módulos aunque estén desactivados, en desarrollo o planificados, y es el único que puede habilitar módulos para una empresa. El Administrador de empresa usa permisos explícitos y nunca recibe `*`. Los demás roles predeterminados permiten adaptar accesos y permisos conservando nombre y descripción, y los roles personalizados permiten editar toda su definición. La activación se realiza desde la tabla, nunca dentro del formulario.

- Área táctil mínima de 44 por 44 px en móvil.
- Skeleton con forma final: el de tablas reproduce cabecera y filas con
  celdas alineadas a las columnas reales (incluyendo icono, título y subtítulo
  en la primera celda), no barras o cuadrados planos.
- Estados con texto e icono; nunca solo color. Activo/Activa usa un check
  circular verde; Inactivo/Inactiva usa un signo menos circular neutro. Se
  construyen con `Status active={...}` para compartir semántica y geometría.
  Activar/Desactivar se maneja con un interruptor: verde (`brand-700`) y perilla
  derecha cuando el registro está activo; rojo (`danger-600`) y perilla izquierda
  cuando está inactivo.
  El tooltip indica la acción siguiente y el nombre accesible identifica el
  registro sin cambiar al alternar el estado. La desactivación conserva su
  confirmación. Check y símbolo de prohibición no se usan como estas acciones.
- Contraste WCAG AA y foco visible con ops-500.
- Alertas de éxito, error e información usan un diálogo global centrado, icono
  semántico, título, explicación breve y acción «Aceptar». El botón «Aceptar»
  usa `--control-height` como todos los demás botones. El comportamiento y
  jerarquía siguen el estándar de Bodegas, reinterpretado con tokens Foods.
- Excepción operativa: Cocina no muestra ese diálogo al pasar una comanda a
  Preparando/Listo; el cambio entre paneles confirma la operación. Los errores
  conservan el aviso. Éxito se cierra automáticamente o con Aceptar/X; errores e
  información requieren una acción explícita. Escape y el fondo no cierran los
  modales; los formularios conservan X/Cancelar. Todos reutilizan `Dialog` para
  retener y restaurar el foco, sin agregar nuevas formas de cierre.
- Confirmaciones destructivas son diálogos independientes: identifican el registro,
  conservan «Cancelar» como acción segura y bloquean la repetición durante el envío.
  La composición es simple: icono y título alineados verticalmente en una fila,
  frase y acciones dentro de una sola superficie, sin tarjetas internas ni divisores. La frase usa texto regular y
  negrita únicamente en el nombre del registro. Impacto e historial solo se
  muestran si aportan información necesaria para decidir; productos y categorías
  usan una pregunta breve sin explicaciones genéricas. Los botones no repiten
  el icono de advertencia; en móvil comparten el ancho y el área táctil estándar.
  La apertura usa un fondo gradual de 200 ms y un Scale Punch elástico de
  520 ms en el panel: crece desde el centro, supera levemente su tamaño final
  y se asienta con rebotes decrecientes. Se desactiva con movimiento reducido.
- En todo modal de registro o edición, el botón de guardado solo dice
  «Guardar» (sin sufijos como «producto», «categoría», «mesa» o «zona», y sin
  variantes como «Guardar cambios»). El estado ocupado dice «Guardando…».
  Esta regla aplica a productos, categorías, zonas y cualquier otro registro.
  Excepción: cuando el botón fija un estado del flujo distinto de la acción
  principal («Guardar borrador» frente a «Registrar y enviar a cocina» o a la
  aprobación de una orden de compra), el sufijo se conserva porque comunica
  que aún no hay efecto operativo.
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

### Registro de pedidos sin mesa

Pedidos ofrece «Nuevo pedido» únicamente con `orders.manage`. El modal comparte
cabecera compacta con icono, título y X, campos `FormField` y primitivas del design
system. Contacto y entrega forman una rejilla uniforme; dirección ocupa dos columnas
y referencia el ancho completo. La carta y el resumen se presentan en dos columnas
en escritorio y en una sola desde móvil, sin un wizard artificial ni textos redundantes.
La carta comparte catálogo, skeleton, error con Reintentar, vacío y paginación de Salón.
Los estilos del catálogo admiten ambos contenedores desde una sola definición,
sin copiar reglas ni aplicar el layout de pantalla completa de Salón al modal.
El resumen permite cambiar cantidad, quitar productos, configurar menús y añadir notas;
los importes se alinean en una franja neutra y el total usa azul primario.
Cabecera y footer permanecen visibles; el cuerpo tiene scroll vertical acotado.
En móvil el modal ocupa la pantalla, los controles conservan 44 px y las acciones
usan ancho completo. El footer tiene Cancelar secundario y una única acción
primaria con un icono: «Registrar y enviar a cocina». Durante el envío muestra
«Enviando…» y bloquea edición.

## Tablas y paginación

- Cabecera azul `primary-600`, texto blanco en mayúsculas, filas alternas sutiles y
  acciones iconográficas consistentes; el color semántico se reserva para estados.
- Escala tipográfica única para todas las tablas, definida en `globals.css` y
  nunca sobrescrita por módulo: cabecera y líneas secundarias usan
  `--font-size-label` (12 px); celdas, paginación y controles usan
  `--font-size-control` (13 px en escritorio). La cabecera conserva peso 800.
- La paginación informa el rango visible y total, permite 10, 20 o 50 filas y
  muestra páginas, elipsis, anterior y siguiente con estado activo inequívoco.
- Toda tabla de gestión incluye paginación, incluida la de categorías.
- La paginación es una primitiva única y compartida. No se replica su cálculo,
  marcado ni estilos dentro de los módulos de negocio.
- No conservar paginadores antiguos ni sus reglas (`catalog-pagination`, `pagination-meta`, estilos
  genéricos de `.management footer` o flechas generadas con pseudo-elementos).
  La geometría base de `Pagination` continúa en `globals.css`; los botones y sus
  estados tienen un único dueño en `pagination.css`, compartido por todas las
  tablas. No se corrigen conflictos con excepciones locales ni `!important`:
  Anterior/Siguiente activos conservan fondo primario y texto blanco;
  deshabilitados usan fondo y texto neutros, con radio y altura estándar.
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
- `--control-height` vale 38 px en escritorio y se redefine a 44 px en
  `@media(max-width:600px)` para conservar el área táctil. Un solo cambio de
  token reajusta toda la aplicación; no se permiten overrides `!important`
  por control ni excepciones arbitrarias por pantalla. Los dispositivos con
  puntero táctil también usan 44 px, independientemente del ancho.
- Los controles cuadrados de icono usan `--control-height` en ancho y alto para
  alinearse con los campos vecinos. Los inputs anidados dentro de un contenedor
  con borde usan `calc(var(--control-height) - 2px)`.
- El radio de los controles es 6 px de forma uniforme; 14 px se reserva para
  tarjetas y paneles.
- En escritorio, los campos usan 38 px de altura,
  radio de 6 px, borde neutro y texto de 13 px. En móvil aumentan a 44 px para
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
- Los iconos describen la función, no el rol de la persona: «Recetas»
  usa `cookingPot` por preparación/producción; «Disponibilidad» usa
  `availability`, un plato con confirmación de disponibilidad. No se usa un check
  genérico como icono principal de módulo.
- En Abastecimiento, Inventario usa `stock` porque representa existencia física;
  Kárdex usa `ledger` porque representa el historial valorizado de movimientos.
  Dos módulos vecinos no comparten icono si su función operativa es distinta.
- El hover modifica color y superficie sin desplazar ni escalar elementos.
- El drawer móvil bloquea el fondo, cierra con Escape, overlay o navegación y
  conserva scroll interno.
- Apertura y cierre duran entre 160 y 240 ms; movimiento reducido elimina la animación.

## Acción principal

En el detalle de mesa de Salón, la siguiente acción del flujo es dominante:
Confirmar entrega cuando todo está Listo, Cerrar cuenta cuando todo está Entregado
y Cobrar cuando la cuenta está Por cobrar. Comparten azul primario; Cobrar usa
`action-pay` con icono de tarjeta y hover `action-pay-hover`.
La entrega se confirma una sola vez para todo el pedido desde el pie.
El verde se reserva para los
estados confirmados, no para representar una acción pendiente. La cabecera distingue preparación y cuenta: `listo` se expresa como «Listo para entregar»; la cuenta usa Abierta / Por cobrar / Pagada. Los importes se comunican en Pagado y Saldo pendiente. Las tarjetas de Salón reutilizan ese mismo estado y color,
sin combinar «Listo» o «Entregado» con «por cobrar»; el saldo se consulta en el
resumen de cuenta del detalle. La franja superior es decorativa, mide 4 px y usa un
degradado de menta `brand-600` a azul índigo `primary-600`; no codifica estados.
No se agrega una instrucción genérica para liberar la mesa: el estado, el saldo
y las acciones disponibles ya expresan el avance. Quitar ese texto no modifica
los requisitos de cierre de cuenta, entrega y pago. En móvil, las acciones ocupan todo el ancho
y los estados siguen visibles. Esta variante conserva los tokens compartidos
de altura y radio de control.

El pie del detalle de mesa presenta una sola acción sólida azul, al final del
grupo: Cobrar si existe saldo cobrable; en otro caso, la siguiente acción real
del pedido. Editar es secundaria con fondo blanco y borde azul visible,
manteniendo la misma geometría de control. Cancelar pedido es una acción secundaria de contorno
rojo, sin relleno rojo dominante, y conserva su confirmación. Todos los iconos de acción son
outline de 18 px: tarjeta para Cobrar, plato/check para registrar la entrega,
recibo para Enviar comanda. La mesa se libera automáticamente cuando
la cuenta está cerrada, la entrega y el pago están completos, sin botón ni confirmación adicional. En
móvil se conserva el orden de lectura y de teclado, con botones de ancho
completo y área táctil de 44 px. Guardar bloquea las acciones y la navegación
a POS, sin cambiar la jerarquía de color.

El pie prioriza el grupo de acciones alineado a la derecha, sin notas explicativas
permanentes. Solo una restricción real, como devolver pagos antes de cancelar,
justifica una nota contextual: se presenta como texto neutro, sin aspecto de
botón, y en móvil va encima de las acciones. Los botones tienen ancho natural y la misma
altura en escritorio; en móvil mantienen ancho completo. Todo botón conserva un contorno visible
o relleno sólido, cursor interactivo, respuesta hover, foco y estado bloqueado.
Los datos y notas no reciben el contorno de acción, cursor de mano ni respuesta hover.

El detalle de mesa usa un modal compacto de hasta 680 px, sin altura mínima
artificial. «Mesa 04» es el título; cliente y tiempo de apertura aparecen debajo.
No muestra el código técnico del pedido ni un eyebrow «Mesa activa». El cuerpo
reúne productos y «Resumen de cuenta» en una sola columna desplazable, tanto en
escritorio como en móvil: evita una tarjeta lateral que deje vacío bajo los productos.
Cantidad, producto e importe usan una rejilla común; los nombres largos se parten
sin desplazar importes. La cantidad de cada línea usa un círculo de 32 px,
fondo `primary-100`, texto `primary-700` y cifras tabulares, centrado con el
nombre y el precio unitario. Es un dato de lectura, sin borde de botón ni
respuesta hover. Su skeleton conserva el mismo círculo. La cantidad total de
unidades aparece una sola vez junto al título de productos; no se duplica con
un conteo de líneas ni con un eyebrow «Detalle». Cabecera y acciones permanecen
visibles al desplazar pedidos largos. La cuenta es una franja compacta de fondo
`cloud-50`, borde estándar y radio de panel, sin un título visual adicional:
Subtotal, Pagado y Saldo
comparten columnas con etiqueta arriba e importe debajo. Delivery añade una
columna solo cuando existe. La última columna alinea el importe a la derecha y
lo destaca con `primary-700` sobre `primary-100`, sin apariencia interactiva.
Las etiquetas usan 13 px y peso 600; los importes secundarios 18 px y el saldo
24 px, todos con cifras tabulares. Subtotal y Pagado comparten el mismo eje de
lectura, sin tarjetas independientes. En móvil, Subtotal, Delivery (si existe), Pagado
y Saldo usan filas completas: etiqueta a la izquierda e importe a la derecha,
con una única alineación vertical de importes. El saldo conserva su fondo tintado y
mayor jerarquía tipográfica; no se mezclan columnas verticales con filas horizontales.
«Resumen de cuenta» conserva su encabezado accesible. Pagado en cero permanece neutro;
los pagos positivos y el importe de una cuenta completamente pagada usan
`brand-700`; la cuenta pagada cambia el fondo destacado a `brand-100`.
El skeleton reproduce estas mismas superficies y tamaños de importes.
Las notas generales no son advertencias:
usan superficies neutras y conservan el texto completo.

La autoría se presenta como información de lectura, no como botones: «Mozo» y
«Cobrado por», con etiquetas neutras, nombres completos que admiten salto de línea
y un espaciado compartido. Salón y Pedidos usan la misma franja en sus detalles.
Las tarjetas de mesa y las filas de Pedidos conservan estos nombres también en
móvil. No se muestra un cobrador antes de registrar pagos. Las acciones de atención
no aparecen para otro mozo; el acceso de cobro mantiene su permiso independiente.

Pedidos reutiliza esta misma estructura del detalle de Salón: fecha bajo el
título, un único conteo de unidades junto a Productos y cantidades circulares.
«Registrado» muestra siempre la fecha y hora del registro en la zona del local,
nunca un tiempo relativo como «Hace 17 h». La tabla y el detalle reutilizan el
formateador regional compartido. En móvil, la fecha aparece bajo la identidad del
pedido al ocultarse su columna, con salto de línea para no recortar el contenido.
No conserva bloques antiguos de «Registrado», «Consumo» o «Detalle» sin estilos.
Los productos y el total pertenecen al cuerpo desplazable; cabecera y acciones
permanecen visibles. Cuando solo se muestra Total del pedido, su fila ocupa
todo el ancho, con etiqueta a la izquierda e importe a la derecha. Delivery
añade Subtotal y Delivery únicamente si aplica. Las acciones tienen ancho
natural en escritorio y ancho completo en móvil, dentro del grupo compartido.
Los textos de interfaz se conservan en UTF-8 y Unicode NFC; no se reparan
tildes mediante reemplazos indiscriminados sobre nombres recibidos del API.

El botón primario usa `primary-600` (`#4654CD`), `--control-height`,
`--font-size-control` y `--radius-control`, igual que los demás controles.
Su jerarquía procede del color, no de una altura distinta. Los avisos exitosos
conservan verde `brand-600` y usan texto `ink-950`; blanco sobre menta no tiene
contraste suficiente. El hover verde oscuro usa texto blanco.

## Acceso administrativo

El login usa una composición centrada y compacta: tarjeta única con línea de
acento índigo, marca centrada dentro de la tarjeta como único encabezado (el
`h1` existe solo para lectores de pantalla; no hay título ni texto de ayuda
visibles), campos con las primitivas `FormField` e `Input` del design system,
acción principal a todo el ancho con icono de candado e indicadores de
confianza al pie. La página redefine `--control-height` a 44 px (48 px en
móvil) en lugar de fijar alturas literales; radio, tipografía y foco de los
campos son los estándar. En móvil reduce únicamente el padding; no cambia el
orden ni oculta información funcional. La referencia define la geometría, pero
la identidad, textos, iconos y colores son exclusivamente Foods.

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

El dashboard usa términos cotidianos del restaurante: «Ventas del día»,
«Pedidos cobrados», «Promedio por pedido», «Pedidos en atención»,
«Pendientes por revisar» y «Productos vendidos hoy». No muestra «Ventas netas»
ni «Ticket promedio». La tarjeta de ventas usa la nota «Cobrado hoy», sin
referencias a devoluciones ni cambiar el cálculo. Los nombres técnicos `salesNet` y
`averageTicket` se conservan en el contrato de API.

El resumen administrativo consolida datos reales de la empresa y local activos:
cuatro indicadores económicos/operativos y paneles compactos de Atención, Cocina,
Delivery y Caja. No duplica cada contador como una tarjeta independiente. Distingue
«Pedidos cobrados» (pago completo con cobro hoy) de «Saldo por cobrar» (pedidos sin
pagar o parcialmente pagados, incluso de días anteriores). El promedio corresponde
al importe total de los pedidos completamente cobrados. «Ventas del día» refleja
los movimientos de cobro de hoy, también pagos parciales.
La fecha procede del backend en la zona del local. La consulta se actualiza cada
30 segundos y su clave incluye empresa/local. Las secciones respetan los módulos
contratados y los enlaces usan las mismas reglas de menú y permisos del shell.
El efectivo de cajas ciegas no autorizado muestra «Importe reservado», nunca cero.
Los pendientes de disponibilidad/abastecimiento no repiten las colas de cocina.
La carga reproduce indicadores, paneles operativos, gráfico, pendientes y productos;
el error es recuperable, no reemplaza datos por ceros. En 390 px los paneles se
apilan, los importes permanecen alineados y los accesos tienen áreas de 44 px.

Ventas ofrece búsqueda, «Fecha de inicio» y «Fecha de fin» mediante controles
homologados, alineados en escritorio y apilados en móvil. El rango es opcional:
admite una sola fecha e incluye todo el día final según la zona del local,
filtrando la fecha de registro mostrada en la tabla. Cambiar un filtro reinicia
la paginación. «Limpiar fechas» conserva la búsqueda. Un rango invertido muestra
error junto al campo y no consulta el API; el vacío filtrado distingue que no
hay coincidencias de que todavía no existen ventas.

Ventas incorpora la acción homologada de ojo con tooltip «Ver detalle de la
venta». Su tabla incluye «Medio de pago», con nombres reales del catálogo recibidos
en el listado; los pagos divididos muestran todos los medios, separados por « · »,
sin repetirlos ni hacer consultas por fila. Móvil conserva este dato a ancho completo
y el skeleton reproduce la misma columna y distribución. Sin cobros vigentes se
muestra «—», nunca un medio supuesto. Los pagos totalmente devueltos no se cuentan.
La acción homologada está disponible en cada fila y tarjeta móvil. El modal es
de consulta, con X como único cierre y sin cobrar, editar ni devolver.
Presenta cliente/mesa, canal, fecha,
productos con cantidad circular, precio unitario y total por línea, composición
y notas cuando existen, subtotal, delivery si aplica, total y pagos netos de
devoluciones. Los medios de pago usan nombres recibidos de la API. Si la venta
cambió después de abrir el historial, el detalle muestra su estado y saldo reales.
La carga reproduce esta estructura con skeleton; error ofrece Reintentar y los
productos/pagos vacíos tienen un mensaje explícito. El cuerpo tiene scroll y
la cabecera permanece visible. Desde 390 px conserva todos los datos y acciones.

Los módulos de directorio, como Clientes, usan la tabla estandarizada en escritorio y tarjetas equivalentes en móvil. Toda consulta remota debe incluir skeleton, error recuperable, vacío contextual y paginación compartida. Las acciones de fila son únicamente iconos homologados con tooltip; alta y edición usan las primitivas `Input`, `Select`, `Textarea`, `Button`, `Status` y `ConfirmDialog`. Los formularios extensos se agrupan por secciones visuales, sin añadir texto explicativo que no ayude a completar la tarea.


### Disponibilidad de módulos

- El catálogo distingue **Disponible**, **En desarrollo** y **Planificado**.
- `active` representa únicamente si un módulo **Disponible** está habilitado para una empresa; no representa su estado de desarrollo.
- El Administrador de plataforma ve todos los módulos en navegación y catálogo, incluso si están inactivos o no terminados.
- Los usuarios de empresa solo ven módulos disponibles, incluidos en su plan, activos para su organización y permitidos por su rol. Emprende incluye «Menús y combos».
- Usuarios y roles muestra únicamente accesos y acciones de módulos efectivos del plan. Los contadores y la vista previa cuentan esos mismos elementos; se omiten grupos vacíos. La empresa no recibe el catálogo comercial completo de Plataforma.
- Un módulo **En desarrollo** o **Planificado** nunca puede activarse para una empresa; el backend debe rechazar cualquier intento aunque el frontend falle.

## Onboarding comercial y suscripción

Existe un único onboarding de tenant en `/platform/onboarding`. La ruta
histórica de Configuración redirige a ese flujo y no mantiene una segunda
implementación. El alta se compone de Empresa → Plan y contrato → Fiscal →
Primer local → Administrador.

La zona horaria pertenece al paso Primer local y usa un autocomplete buscable por
ciudad o identificador IANA. Se sugiere desde `defaultTimezone` del catálogo del
país y permanece editable. Cambiar de país actualiza la sugerencia; retroceder o
refrescar catálogos conserva la elección del usuario. La empresa se inicializa
con esa misma zona y los demás locales conservan su configuración independiente.

Mientras se carga el contexto de `/platform/onboarding`, el skeleton reproduce
los cinco pasos, la cabecera, los campos del primer formulario y la acción
inferior. Reutiliza el shimmer de Plataforma, `--control-height` y la misma
rejilla responsive del wizard; no muestra texto genérico de carga.

El paso Plan y contrato consume el catálogo SaaS del backend: nunca hardcodea
precios ni paquetes. Muestra precio según ciclo, prueba, límites, módulos y
versión de condiciones, y exige registrar la aceptación antes de crear el
tenant. Plataforma administra el catálogo en `/platform/plans` y la
suscripción de la empresa activa en `/platform/subscription`.

El Administrador de plataforma accede a «Países y WhatsApp» directamente desde
el grupo Configuración del menú principal y también desde la navegación del área
de plataforma. La pantalla muestra la moneda predeterminada de cada país y
administra únicamente su número público, nombre visible y estado. El país se
elige mediante un autocomplete y su moneda queda asociada sin edición durante
el onboarding. `WHATSAPP_PHONE_ID`, los tokens y las demás credenciales de Meta
se inyectan exclusivamente mediante variables de entorno y nunca se exponen en
esta interfaz. Los usuarios de empresa nunca ven este acceso.

El Administrador de empresa con permiso `subscription.read` ve en Mi perfil
un resumen de solo lectura con plan, estado, precio/ciclo, renovación, prueba,
condiciones, uso frente a límites, módulos incluidos y último pago. Cambiar
plan, estado o registrar cobros sigue siendo una acción exclusiva de Plataforma.

### Puesta en marcha operativa

La ruta compone una única pantalla del módulo `setup`. La cabecera de bienvenida
reúne la acción «Iniciar recorrido guiado» y el avance esencial real. La hoja de
ruta usa pasos numerados sobre fondo neutro, selección azul suave y estados
textuales. El detalle tiene altura natural, instrucciones breves, un consejo
y una acción para abrir la sección con guía; no usa un panel lateral oscuro ni
reserva grandes espacios vacíos. En móvil la hoja de ruta tiene scroll horizontal
y las modalidades de atención se presentan como radios en filas.

El recorrido acompaña las pantallas reales con un panel contextual minimizable
y un contorno en el área explicada. «Anterior» y «Siguiente» navegan por las
opciones permitidas; el último paso vuelve a la revisión. Recorrer la guía no
marca requisitos como completos: la API conserva esa autoridad. La carga se
comparte por empresa y local, y se consulta de nuevo al regresar al resumen.


Después de crear la empresa, su Administrador entra a
`/settings/getting-started`. El asistente lee contadores reales de la API,
no simula datos ni duplica formularios: deriva al mantenimiento correspondiente
y comprueba los cambios al regresar. Son obligatorios el tipo de atención, una
categoría, un producto vendible y una caja; las mesas solo son obligatorias para
salón o modalidad mixta.

Los pasos opcionales se construyen desde los módulos activos de la suscripción.
Inventario, recetas y proveedores no aparecen cuando el plan no los incluye;
en particular, el plan básico nunca muestra Recetas. Mientras la puesta en
marcha siga pendiente, la navegación conserva un recordatorio visible y el
login del Administrador de empresa abre primero el asistente. Carga, error y
estado listo respetan los patrones visuales generales y el flujo funciona desde
390 px.


### Nombres de navegación

- Los nombres de navegación evitan repetir el nombre del grupo. Dentro de
  `CARTA Y PRODUCCIÓN`, el módulo se llama `Recetas`, no
  `Recetas y producción`. Por la misma regla, la opción se llama `Disponibilidad`,
  no `Disponibilidad de la carta`.
- Una opción debe nombrar las entidades que realmente administra. La pantalla
  de acceso usa `Usuarios y roles`: los permisos son atributos configurados
  dentro de cada rol y no necesitan repetirse en el nombre del módulo.


### Ruta inicial por rol

- Después del login nunca se redirige de forma fija a `/dashboard`.
- La entrada se calcula con el mismo catálogo que construye el sidebar y respeta,
  en este orden, módulo contratado, acceso de menú y permiso mínimo de lectura.
- Se abre la primera opción realmente accesible según el orden de navegación.
  Ejemplos predeterminados: Mesero → Pedidos, Cocinero → Cocina, Cajero → Punto
  de venta y Almacenero → Inventario.
- El logo de FUDIA y el cambio de local reutilizan exactamente la misma regla.
- Si ningún módulo satisface las tres condiciones, la cuenta entra a
  `/no-access`, donde se informa que debe revisarse su rol. No se muestra un
  Dashboard bloqueado como página inicial.


### Convención de URL

- Los segmentos públicos de las páginas se escriben en inglés; el contenido y
  las etiquetas visibles se mantienen en español.
- `src/shared/routing/page-routes.ts` es la única autoridad para construir
  enlaces internos. Una pantalla no debe escribir nuevamente una ruta literal.
- Las URL históricas en español redirigen a su equivalente canónico en inglés.
  Cada ruta canónica corresponde a una carpeta real de App Router, por lo que
  abrir un enlace directo o actualizar el navegador nunca depende de una
  navegación previa del cliente ni de un `rewrite`.
- Al renombrar una ruta se conserva temporalmente su alias en
  `legacyPageAliases`; no se duplican pantallas ni lógica de negocio.

### Datos de entrega y comanda compartida

Pedidos manuales separa datos de atención y selección de productos. El formulario
inicial es compacto: tipo de atención, cliente, teléfono y datos de entrega si
aplica. No incluye otra carta ni otro carrito. «Continuar» abre la misma comanda
que Salón, con el contexto del canal en lugar de mesa. «Editar datos» conserva
productos, cantidades, elecciones y notas; no repite indicadores de pasos.
Delivery desglosa Subtotal y Envío antes del Total. Solo «Registrar y enviar a
cocina» persiste el pedido. El cierre exige descarte cuando hay cambios y se
bloquea durante la petición. El formulario usa una columna en móvil desde 390 px;
la comanda conserva su resumen móvil y sus estilos compartidos.

### Atribución en Caja y turnos

En un turno abierto, «Cajero» identifica a quien abrió ese turno y quedó a cargo.
Al cerrar y abrir el siguiente turno, se muestra al nuevo abridor. El «Equipo
actual» se presenta por separado: asignar o retirar integrantes modifica los
permisos de operación, pero no transfiere la responsabilidad del turno.
En el historial de turnos cerrados, «Cerrado por» identifica a quien realizó el
cierre; el detalle conserva también «Abierto por» y ambas fechas. La autoría de
cada movimiento permanece en su propia fila. Asignar o retirar usuarios refresca
las cajas, el historial y el detalle inmediatamente. Punto de venta muestra al
mismo cajero del turno activo. El cambio de responsable se realiza cerrando el
turno y abriendo uno nuevo; no se infiere por la sesión que consulta la pantalla.

«Apertura» muestra fecha y hora reales, formateadas con el país y la zona horaria
del local; nunca se sustituye por el día operativo. El historial y el detalle
conservan apertura y cierre, con sus etiquetas explícitas y elementos `time`.
Mientras el turno está abierto no se inventa una fecha de cierre. La API entrega
instantes RFC 3339 en UTC; el frontend admite también los offsets antiguos de
PostgreSQL para mantener compatibilidad durante el despliegue.

## Entrega del pedido y cuenta de mesa

- Salón/Pedidos distinguen preparación y cuenta sin códigos técnicos ni instrucciones redundantes. El destino y estado de cada producto se presentan en una fila compacta y alineada.
- El pie muestra un único botón primario «Confirmar entrega» para todo el pedido cuando todos los productos pendientes están Listos. No se muestran botones de entrega individuales: destino y estado de cada producto son datos de lectura. Cocina/Barra preparan; el mozo confirma todo con una acción. Las rondas adicionales incluyen solo productos aún sin entregar y no repiten las entregas anteriores. No se muestran modales exitosos en estas transiciones operativas.
- «Agregar productos» reutiliza la comanda con solo líneas nuevas y aparece únicamente con cuenta Abierta. «Cerrar cuenta» es primaria y aparece solo después de entregar todos los productos de todas las rondas. Después se muestra el enlace a cobro solo con entrega completa, permiso y saldo; no se ofrecen más consumos ni reapertura.
- Cocina y Barra reutilizan el mismo KDS con selector de estación alimentado por backend, permisos independientes y las mismas primitivas visuales.
- En móvil (390 px) las filas permiten wrap sin desbordar; la entrega global y las demás acciones del pie conservan 44 px y ancho disponible. No se duplican clases de los estilos base.
