# Sistema de diseño administrativo — FUDIA DS 1.1

Este documento es la autoridad visual de `foods-admin-web` y no depende de
documentos externos. La primera parte fija los fundamentos (tokens, tipografía,
color, forma, movimiento y plataforma) que toda pantalla consume; la segunda
recoge las especificaciones compartidas de componentes y pantallas. Cuando una
especificación de pantalla contradiga un fundamento, prevalece el fundamento y
la especificación se corrige.

Aprobación integral: **2026-11-24** (FUDIA DESIGN SYSTEM 1.0). Esta versión
reemplaza la identidad azul operativo + marino anterior: superficies blancas,
fondo gris claro, sombras suaves, tipografía Inter, acción y selección en
índigo corporativo y estados en superficies pastel. No se mantiene una segunda
identidad: cualquier resto del sistema anterior es un defecto por corregir.

## Homologación de composición — 2026-11-25

FUDIA DS **1.1** completa la adaptación del video de referencia autorizada por
el usuario: superficies blancas, composición limpia e índigo en lugar de los
rellenos negros de interacción. Sustituye las decisiones anteriores de cabecera
grafito y espaciado heredado. La matriz semántica pastel aprobada el 24 de
noviembre sigue vigente; el índigo no sustituye colores de estados.

- **Página:** 24 px arriba y a los lados, 32 px abajo. Hasta 820 px: 20 px
  arriba y 16 px laterales. El shell y Plataforma consumen los mismos roles
  `page-space`, `page-inset` y `page-bottom-space`.
- **Navegación:** 224 px en escritorio, 210 px en escritorio estrecho y drawer
  de hasta 320 px en móvil; enlaces de al menos 44 px. Las reglas base preceden
  a las adaptaciones. El modo contraído alinea sus 78 px con el contenido.
- **Formularios:** `ModalForm` no tiene padding ni scroll. `ModalBody`
  posee el único scroll y 24 px laterales e inferiores; en móvil, 16 px.
  En formularios deja 20 px tras la cabecera (16 px móvil y 8 px con teclado).
  El header, cuerpo, resumen y footer comparten `modal-inset`. No anidar
  contenedores que vuelvan a añadir sangría general. El pie es un hermano fijo del cuerpo,
  sin sticky, márgenes negativos ni desplazamiento del formulario completo.
- **Tablas:** cabecera índigo `table-header-background` (primary-700), texto
  blanco `table-header-foreground`; las columnas fijas usan esos mismos roles.
  Celdas a 16 px en escritorio, alineadas con los filtros, y 12 px en móvil.
- **Composición de formularios:** la referencia guía la jerarquía y el ritmo.
  El patrón aprobado a partir de la imagen tiene encabezado blanco, icono de
  apoyo y explicación breve debajo del título. No utiliza encabezados oscuros
  ni franjas sólidas de marca. El cuerpo y el pie permanecen claros. El grupo
  principal usa `modal-primary-background` (primary-50), mientras que los grupos
  secundarios continúan sobre blanco. El énfasis se completa con tipografía,
  proximidad e iconos; no se reduce a añadir una caja gris. Los títulos
  siempre preceden a sus campos, sin columnas laterales de encabezados.
  Se conserva el orden, los grupos y la distribución propios de cada formulario;
  la homologación corrige su espaciado, superficie, tipografía e iconos de apoyo.
  Productos conserva su composición aprobada: imagen a
  la izquierda, datos a la derecha y descripción visible. No se inventan
  resúmenes para formularios vacíos.
- **Combinación de color:** las familias se asignan a funciones mediante
  `getIdentityTone` y se comparten entre el icono del encabezado y `FormSection`:
  menta para caja, pagos y preparación; durazno para gastos y categorías;
  celeste para clientes y reservas; violeta para productos, accesos e integraciones;
  petróleo para contacto, proveedores y compras. El primer grupo de datos
  toma una superficie clara de su familia; los siguientes mantienen fondo blanco
  con título e icono de color. El prefijo de los importes usa índigo tenue.
  Los textos y símbolos conservan contraste AA. Esta identidad no representa
  un resultado: estados, avisos y acciones destructivas conservan sus familias
  semánticas aprobadas, con etiqueta explícita.
- **Jerarquía:** secciones de formulario de 16 px; grupos de campos con 16 px
  de separación y secciones con 20–24 px. Mantener 40 px de control de escritorio
  y 48 px táctiles; compactar la composición no reduce las áreas de interacción.
- **Superficies:** indicadores del Dashboard blancos; el significado vive en
  cifras, iconos y estados. Acciones de tabla sin cajas tintadas permanentes,
  hover índigo tenue y foco visible. Sombras de acción discretas, sin halo índigo.
  El contacto de un detalle se integra en su superficie, sin otra tarjeta.
- **Plataforma:** cabecera blanca y navegación índigo tenue, con los mismos
  campos, botones y márgenes que la administración.

Chat público, billeteras, QR, PDF y colores semánticos conservan su identidad.
La homologación se verifica en rutas reales y formularios de Gastos, Clientes,
Compras/Proveedores, Productos y Usuarios/Roles, además de los componentes base.

## Dirección

fudIA es una herramienta de operación para restaurantes peruanos: el mozo con
el celular en plena hora punta, el cajero en escritorio, el administrador
revisando el cierre. La interfaz se juzga por lo rápido que se completa la
tarea, no por la decoración.

- **Modo Operar.** Familiaridad, densidad útil, una acción dominante por
  superficie y estados siempre visibles. La identidad vive en detalles
  precisos, no en adornos.
- **Índigo corporativo.** El tono real del producto, `primary-600 #4654CD`,
  representa interacción y selección: acción principal, selección, foco y
  navegación activa. No decora: no es color de estado ni de fondo.
- **Neutros de la especificación.** Superficies blancas, fondo general
  `cloud-50 #F3F4F6`, grafito `ink-700 #374151` para texto general.
- **Estados pastel.** Toda estado usa superficie pastel, texto oscuro de su
  familia e indicador de color. No existe la píldora sólida de la versión
  anterior; la etiqueta y el contexto completan el significado.
- **Sin elementos de plantilla.** Prohibido: eyebrows en mayúsculas sobre los
  títulos, numeración decorativa de secciones, barras laterales de color en
  tarjetas o avisos, texto con degradado, sombras de halo, tarjetas dentro de
  tarjetas y monoespaciada como disfraz de dato técnico.

## Tokens

`src/styles/variables.css` es la única fuente de valores y `globals.css` la
importa una vez, después de Tailwind. Los nombres describen un rol o un paso de
escala, nunca un valor en píxeles. Las hojas de componentes y módulos consumen
`var(...)`; `tests/design-tokens.test.mjs` impide valores literales (`px`,
hexadecimales, pesos numéricos), redefinir tokens fuera de la hoja, colores
duplicados, referencias inexistentes y ciclos.

La equivalencia con la especificación general es: la escala `brand-*` del
documento de especificación vive aquí como `primary-*` (acción y selección);
la escala verde de éxito vive como `brand-*`. Ningún componente consume
hexadecimales directamente.

Tailwind se mantiene. El bloque `@theme inline` de `variables.css` hace que sus
utilidades resuelvan a los mismos tokens: `bg-primary-600`, `text-ink-500`,
`p-4` (16 px), `gap-3`, `rounded-lg` (tarjeta), `rounded-sm` (control),
`shadow-sm`, `text-sm`, `font-semibold`, `tracking-tight`, `ease-out`.

## Color

| Rol | Tokens | Uso |
| --- | --- | --- |
| Superficie | `surface`, `cloud-25`, `cloud-50`, `cloud-100`, `cloud-150` | Paneles en blanco; fondo de página `cloud-50` (#F3F4F6); hover de superficies `cloud-25` (#F9FAFB); hundidos y deshabilitados en `cloud-100`; pista y estados neutros en `cloud-150` |
| Líneas | `line`, `line-strong` | `line` (#E5E7EB) divisores y paneles; `line-strong` (#D1D5DB) bordes de campos |
| Texto e iconos | `ink-950`, `ink-800`, `ink-700`, `ink-600`, `ink-500`, `ink-400`, `ink-300` | `ink-950` (#18181B) títulos e importes; `ink-700` (#374151) texto general y datos secundarios; `ink-600` secundario; `ink-500` auxiliar; `ink-400` marcadores; `ink-300` iconos deshabilitados |
| Índigo corporativo | `primary-25` … `primary-800` | `primary-600 #4654CD` acción, selección, foco y elemento activo; `primary-700` hover; `primary-800` pressed; `primary-50` fondos seleccionados |
| Información | `info-50/100/200/500/600/800` | Semántica azul (#3B82F6): confirmado, en camino y hitos informativos; distinta del índigo de acción |
| Éxito | `brand-50 … brand-800` | Verde de éxito (#10B981): confirmado, pagado, entregado, activo |
| Violeta digital | `digital-50 … digital-700` | IA/automatización e integraciones (#8B5CF6); nunca representa WhatsApp |
| WhatsApp | `chat-header`, `chat-outgoing` | Identidad propia del canal (#008069 sobre #D9FDD3); nunca violeta ni índigo |
| Petróleo | `teal-100/200/300/500/700` | Cuenta abierta y acento de identidad (#14B8A6) |
| Celeste identidad | `sky-100/200/500/800` | Acento de identidad (categorías, hitos de cuenta); no comparte tono con la información |
| Morado categórico | `purple-100/200/600` | Serie de gráfico y acento puntual; ya no es estado |
| Advertencia | `warning-50 … warning-800`, `note-100`, `note-ink` | Preparación y pendientes (#F59E0B); notas de cocina ámbar oscuro |
| Error | `danger-50 … danger-700` | Errores y acciones destructivas (#DC2626) |
| Pagos | `payment-yape-*`, `payment-plin-*` | Identidad protegida de billeteras (círculos con texto) |
| QR | `qr-ink` | Negro suave del código operativo |

Reglas:

- Una sola acción sólida índigo por superficie. Las secundarias van sobre
  blanco con borde `line-strong`.
- El índigo no es estado: los estados usan las familias semánticas pastel.
- Rojo y ámbar comunican significado (error, demora, pendiente); nunca
  decoran.
- Los importes y valores protagonistas van en negro suave (`ink-950`), con
  cifras tabulares.
- Los estados combinan color, texto e icono; nunca dependen solo del color.
- Transparencias por paso (`ink-950-alpha-8`, `primary-600-alpha-24`…), no
  porcentajes sueltos.

### Identidades protegidas

- **Chat público.** La paleta `--chat-*` está congelada en literales
  (`chat-header #008069`, canvas `#efeae2`, enviados `#d9fdd3`, recibidos
  blancos): la homologación del panel administrativo no altera su identidad.
  Solo se homologan espaciados, accesibilidad y responsive.
- **Yape y Plin.** Círculo lleno con el nombre en blanco, sin logotipo:
  `payment-yape-background #742284`, `payment-plin-background #00B8C8`, con
  derivados `*-soft` y `*-border` propios. Los valores reales del repositorio
  tienen prioridad sobre cualquier referencia externa.
- **WhatsApp y Telegram.** Mantienen su iconografía de marca (`whatsapp`) y el
  color propio de la paleta `--chat-*` (verde `chat-header #008069` con fondo
  suave `chat-outgoing #D9FDD3`): nunca violeta ni índigo.
- **Logo de Fudia.** Se conserva sin cambios (`logo-mark.png`); el wordmark
  usa el índigo sobre superficie clara.

## Tipografía

Una sola familia, **Inter** (`next/font`, variable `--font-inter`), con escala
fija por rol. Los importes, cantidades y porcentajes usan cifras tabulares
(`font-variant-numeric: tabular-nums`, clase `.tabular`), nunca monoespaciada.

| Token | Tamaño | Rol |
| --- | --- | --- |
| `--font-size-2xs` | 11 px | Micro: etiquetas de grupo y contadores diminutos |
| `--font-size-xs` (`--font-size-label`) | 12 px | Labels, metadatos, estados y cabecera de tabla |
| `--font-size-sm` (`--font-size-control`) | 13 px | Controles, celdas, descripción de página (16 px en táctil) |
| `--font-size-base` | 14 px | Cuerpo |
| `--font-size-md` | 16 px | Títulos de panel y estado vacío |
| `--font-size-lg` | 18 px | Títulos de modal e importes secundarios |
| `--font-size-xl` | 22 px | Título de página en móvil, KPI |
| `--font-size-2xl` | 28 px | Título de página, KPI de escritorio |
| `--font-size-3xl` / `--font-size-4xl` | 36 / 48 px | Cifras protagonistas (Ventas del dashboard) |

- Pesos estándar 400/500/600 (`--weight-regular`, `--weight-medium`,
  `--weight-semibold`); los alias `bold`/`heavy` resuelven a 600. Evitar
  negritas excesivas.
- Interlineado: `tight` 1.2, `snug` 1.35, `normal` 1.5, `relaxed` 1.65.
- Tracking: `tight` (−0.02 em) en títulos de 22 px o más, `snug` en 16–18 px,
  `caps` solo en texto en mayúsculas de 11 px (se transforma con CSS; el DOM
  conserva caja normal para lectores de pantalla).
- Tamaño mínimo 11 px (`tests/type-scale.test.mjs`). Títulos con
  `text-wrap: balance`; párrafos con medida máxima de 72 caracteres.
- Sin medios píxeles ni tamaños fuera de la escala.

## Espaciado y dimensiones

Base de 4 px con medios pasos hasta 14 px: `--space-px` (1), `--space-0-5`
(2), `--space-1` (4), `--space-1-5` (6), `--space-2` (8), `--space-2-5` (10),
`--space-3` (12), `--space-3-5` (14), `--space-4` (16), `--space-5` (20),
`--space-6` (24), `--space-7` (28), `--space-8` (32), `--space-10` (40),
`--space-11` (44), `--space-12` (48), `--space-16` (64), `--space-20` (80).
El índice equivale al de Tailwind (`p-4` = `--space-4`).

- Ritmo: grupos relacionados con 6–12 px, secciones separadas con 20–32 px.
- `--size-*` se reserva para dimensiones de layout (anchos de columna, alturas
  fijas, iconos), no para padding ni gap.
- `--control-height`: 40 px con puntero fino, 48 px en móvil (≤600 px) y en
  cualquier puntero táctil. Todo control interactivo de una barra o formulario
  usa esa altura.

## Forma

| Token | Valor | Uso |
| --- | --- | --- |
| `--radius-chip` | 4 px | Marcas pequeñas, barras de skeleton |
| `--radius-md` | 8 px | Controles compactos, cierre de modal, opciones |
| `--radius-control` | 10 px | Botones y campos de escritorio |
| `--radius-tile` | 10 px | Iconos contenidos e ítems internos |
| `--radius-card` | 16 px | Paneles, tarjetas |
| `--radius-dialog` | 20 px | Modales y confirmaciones |
| `--radius-sheet` | 24 px | Hojas inferiores en móvil |
| `--radius-pill` | 999 px | Estados, contadores, interruptores y botones de texto en móvil |

Una superficie se separa con borde **o** con elevación, no con ambos a la vez
salvo `--elevation-1`, que es un apoyo mínimo del panel.

## Elevación y anillos

Sombras suaves y discretas, una sola fuente de luz superior:

- `--elevation-1`: `0 1px 2px #00000005` — panel en reposo.
- `--elevation-2`: `0 2px 8px #00000008` — tarjeta interactiva, marca, logo.
- `--elevation-3`: `0 4px 16px #0000000D` — popover, menú, dropdown.
- `--elevation-4`: `0 24px 64px #00000026` — modal y drawer.
- `--elevation-5`: `0 12px 32px #0000001A` — toasts y elementos flotantes.
- `--elevation-up`: barras fijas inferiores; `--elevation-side`/`-side-end`:
  paneles laterales.
- `--elevation-primary`: realce teñido de índigo de la acción primaria y la
  página activa.
- `--elevation-primary-soft`: sombra amplia y suave bajo objetos de color
  pleno (el plato de la mesa en Salón).
- Anillos: `--ring-focus` (foco de campos: halo tenue junto al borde
  `primary-600`), `--ring-danger`, `--ring-selected`/`--ring-selected-soft`,
  `--ring-line`, `--ring-success`.

El foco de teclado es único: contorno de 2 px `primary-600` con 2 px de
separación en todo control (`:focus-visible`); los campos usan además
`--ring-focus`. En campos compuestos (`ds-input-shell`, buscadores, selectores
con icono y `money-input`) solo el contenedor dibuja borde y anillo mediante
`:focus-within`.

## Movimiento

El movimiento comunica un cambio de estado, nunca adorna. Sin rebotes ni
movimientos exagerados (`tests/design-tokens.test.mjs`).

| Token | Valor | Uso |
| --- | --- | --- |
| `--duration-press` | 120 ms | Respuesta al presionar |
| `--duration-fast` | 120 ms | Hover, color, borde, checkbox, radio, dropdown, tooltip |
| `--duration-motion` | 150 ms | Switch, pestañas, cambio de color |
| `--duration-base` | 200 ms | Sidebar, acordeón, velo de carga |
| `--duration-modal` | 180 ms | Modales y confirmaciones (escala inicial 0.98) |
| `--duration-drawer` | 220 ms | Hojas y drawers |
| `--duration-ambient-long` | 1500 ms | Skeleton |
| `--duration-ambient-short` | 800 ms | Spinner |

- Curvas: `--easing-out` para entradas y respuestas; `--easing-in-out` para
  movimiento en pantalla; `--easing-drawer` para hojas y drawers. Nunca
  `ease-in` en interfaz.
- Todo botón y botón de icono responde al presionar con
  `transform: scale(var(--press-scale))` (`.94` en controles de icono).
- Las pestañas responden con cambio de superficie y su indicador, sin escalado.
- Solo se animan `transform`, `opacity`, color y sombra; nunca alto, ancho o
  márgenes. No se usa `transition: all`.
- Las acciones repetidas cientos de veces no se animan.
- `prefers-reduced-motion: reduce` reduce todo a un fundido instantáneo.

## Microinteracciones y estados

Todo control responde en el mismo cuadro en que se toca y con los mismos
tokens. Orden de prioridad: **deshabilitado > en curso > error > foco >
presionado > hover > reposo**.

| Estado | Contrato | Tokens |
| --- | --- | --- |
| Hover | Solo con puntero fino; cambia fondo, borde o color un paso | `--transition-surface` (120 ms, `easing-out`) |
| Foco | Contorno de 2 px `primary-600` con 2 px de separación; campos añaden `--ring-focus` | `--ring-focus`, `--ring-danger` |
| Presionado | `scale(var(--press-scale))`; pestañas y acciones repetidas con superficie | `--transition-press` (120 ms) |
| En curso | `Button loading`: `ds-spinner` en el lugar del icono, etiqueta en gerundio, `aria-busy`, bloqueo de doble envío | `--opacity-busy` |
| Guardado sobre el formulario | Un solo indicador con texto: el botón que originó la acción. `ModalHeader busy` añade `ModalBusy` | `ModalBusy`, `--delay-long-wait` |
| Éxito | `FeedbackProvider` (respuesta success) o el `Status` confirmado; se notifica al instante y las recargas corren en segundo plano | Matriz de estados |
| Error | Borde `danger`, mensaje `field-error`/`form-error` con `feedback-reveal` | `--ring-danger` |
| Deshabilitado | `opacity: var(--opacity-disabled)` (.5), sin hover ni presión; campos con fondo `cloud-100` y texto `ink-400` | `--opacity-disabled` |

## Capas

`--z-raised` 1, `--z-sticky` 10, `--z-header` 20, `--z-popover` 30,
`--z-tour` 60, `--z-drawer` 90, `--z-task` 100, `--z-modal` 110,
`--z-sheet` 120, `--z-confirm` 130, `--z-toast`/`--z-feedback` 140,
`--z-tooltip` 150, `--z-loader` 200. No se escriben números sueltos.

## Iconos

Un único set en `design-system/icons.tsx`, dibujado con Tabler Icons
(`@tabler/icons-react`, licencia MIT): retícula de 24 × 24, trazo 1.75,
extremos y uniones redondeados. Las pantallas usan solo `Icon name="…"`;
cambiar un dibujo no toca las pantallas. `IconSize` limita los tamaños: 14 junto
a texto de 12–13 px, 16 en chips y pestañas, 18 en acciones, 20 en navegación,
24 en estados vacíos. El set es monocromo: el color de identidad lo aporta el
contenedor tintado (`IconTile`). No se usan emojis como iconos funcionales.

## Plataforma móvil

Mobile-first desde 390 px, validado en hardware real cuando el cambio afecta
gestos, teclado o áreas seguras.

- `viewport-fit=cover` y `themeColor` blanca; barras fijas y hojas suman
  `--safe-area-top` y `--safe-area-bottom`.
- Sin destello de toque, `touch-action: manipulation` y sin selección de texto
  al mantener presionado un botón.
- Todo `:hover` vive en `@media (hover:hover) and (pointer:fine)`.
- Áreas táctiles de 44 × 44 px. Los campos usan 16 px en táctil para que iOS
  no haga zoom.
- Alturas de pantalla con `dvh`; contenedores con scroll propio usan
  `overscroll-behavior: contain`.
- Teclado correcto: `inputMode="decimal"` en importes, `numeric` en códigos y
  sin `autoFocus` en pantallas táctiles.
- La barra superior mide `--shell-header` (64 px escritorio, 56 px móvil); las
  franjas fijas de filtros se anclan a ese token, nunca a un literal.

## Componentes base

- **Botones** (`globals.css`, única fuente): variantes `primary` (índigo
  sólido, hover `primary-700`, pressed `primary-800`), `secondary` (blanco,
  borde `line-strong`, texto `ink-950`), `tonal` (`primary-50` con texto
  `primary-700`), `ghost` (transparente, texto `ink-700`), `link` (índigo,
  subrayado en hover), `danger` (`danger-500`), `danger-outline` (borde
  `danger-200`, fondo `danger-50`) y `danger-subtle`. Las variantes `strong`
  de `secondary` y `ghost` usan el sólido índigo `primary-600` con hover
  `primary-700`: no existe superficie grafito en botones. Tamaños: XS 28, SM 32,
  MD (control) 40, LG 48. Peso 500–600, icono a la izquierda, foco de 2 px.
  `Button action` centraliza intenciones: `cancel` (neutro), `cancel-record`
  (rojo tenue), `back` (neutro), `continue` (índigo, flecha al final). En móvil
  (≤600 px) todos los botones de texto usan `radius-pill` y altura táctil de
  48 px; campos y cierre conservan su radio.
- **Estados**: `Status` en píldora pastel con punto o icono semántico y texto
  oscuro; `Tag` neutro para atributos. Ver la matriz abajo.
- **Encabezado de página**: título igual al nombre de la navegación,
  descripción de una línea, retorno opcional y como máximo una acción primaria
  y una secundaria. Sin eyebrow.
- **Modales**: `Modal` con tamaños `sm` 480, `md` 640, `lg` 880 y `xl` 1040.
  Superficie blanca, radio `--radius-dialog`, sombra `--elevation-4`, overlay
  negro al 40 % con desenfoque de 2 px. Cabecera blanca con título oscuro 18 px/600
  y X neutra; cierre de 32 px (44 px táctil en móvil). Los formularios tienen
  un icono de tarea de 20 px sobre una superficie índigo tenue de 40 px;
  contexto breve con `description` y cabecera de al menos 80 px, con divisor inferior.
  El contexto se oculta con el teclado abierto para recuperar altura. Los detalles de lectura
  conservan título y X, con cabecera de 64 px (56 px móvil).
  Cuerpo blanco desplazable y pie fijo blanco con divisor superior.
  `Cancelar` queda a la izquierda; alternativas y acción principal índigo a la
  derecha, conservando su orden en el DOM. La intención se declara con
  `Button action="cancel"`, sin inferirla del texto. En móvil, hoja inferior (radio
  `--radius-sheet`) o pantalla completa. Escape y el fondo nunca cierran un
  modal; con cambios sin guardar, cerrar pide confirmar el descarte.
  Visor PDF y opciones del chat público mantienen sus superficies protegidas.
- **Confirmaciones**: `ConfirmDialog` con icono semántico sobre superficie
  pastel, entrada sobria (escala 0.98, 180 ms) y confirmación destructiva roja.
- **Pestañas** (`Tabs` compartido): la activa se reconoce por fondo índigo
  pastel `primary-50`, texto `primary-700`, icono `primary-600` e **indicador
  inferior de 3 px**; las inactivas son transparentes con indicador `line`.
  Cuerpo de 44 px en escritorio y 52 px en móvil, contadores indigo y tono de
  familia para carriles y canales (`TabTone`: `primary`, `info`, `orange`,
  `green`, `violet`, `whatsapp`). Las franjas que filtran una lista larga usan la variante
  chips (`ds-tabs--chips`): píldoras blancas con borde, activa en índigo pastel.
- **Tablas**: contenedor blanco con radio de tarjeta; **encabezado índigo
  `table-header-background` (primary-700) con texto blanco** de 12 px/500 y 40 px de alto,
  decisión definitiva que reemplaza cualquier cabecera grafito anterior.
  Filas compactas de 48 px (56 px táctil), hover `primary-25`, seleccionadas
  `primary-50`, sin cebra ni bordes verticales, cifras tabulares, importes a
  la derecha y columna identificadora fija en móvil.
- **Interruptores**: MD 44 × 24 con perilla de 18; encendido en índigo
  `primary-600`, apagado en `ink-400`, 150 ms `ease-out`, área táctil ampliada
  en móvil.
- **Campos**: altura de control, radio `--radius-control`, borde `line-strong`
  (hover `ink-400`), foco `primary-600` + `--ring-focus`, deshabilitado
  `cloud-100`/`ink-400`, solo lectura hundido en `cloud-25`.
- **Estados vacíos y de error** (`states.tsx`): icono en contenedor de 56 px
  sobre `cloud-100`, título 16 px/600, descripción y acción opcional; el error
  usa la familia `danger` pastel.

### Jerarquía de datos en los modales de contenido

Etiquetas de datos 12 px/600, valores 14 px/400 y títulos de sección 16 px/600
(`modal-section-title`, caja normal, con `modal-section-rule` de 1 px `line`).
Los detalles (pedido, venta, cobro, turno, gasto, cliente) se organizan con
`DetailHero`, `DetailContact`, `Tabs`, `DetailFacts` y `DetailTimeline`. El
importe principal del resumen (`modal-summary-total`) es la única ficha hundida
(`cloud-25`) con etiqueta auxiliar `ink-500` e importe en negro suave 28 px/600;
pagada conserva el pastel `brand-50`. Franjas de lectura en línea:
`modal-inline-metadata` con etiqueta/valor de 14 px, responsable a la izquierda
y `modal-inline-date` al extremo derecho.

## Colores de operación: pedido, preparación y cuenta

Estándar aprobado y obligatorio para todas las pantallas, listados y modales
que presenten estos estados (matriz FUDIA DS 1.0, 2026-11-24). La semántica de
Pedidos y Salón se centraliza en `orderStatus` de `order-meta.ts`; Cocina/Barra
usa `serviceStatusTone`; la cuenta usa `accountTone`; Compras centraliza
`purchaseStatusMeta`. Reutilizar `Status` y los tokens de
`src/styles/variables.css`; ninguna pantalla redefine etiquetas del API ni
crea mapeos propios.

Reglas de aplicación:

- Toda estado usa la píldora pastel de su familia (fondo tenue, texto oscuro e
  indicador de color). La etiqueta y su contexto (pedido, preparación o
  cuenta) completan el significado; el color nunca es la única información.
- El índigo es acción y navegación: no es estado. Confirmado usa la familia
  información (`info-*`), distinta del índigo.
- Cambiar esta semántica requiere aprobación explícita y actualizar juntos
  esta matriz, tokens, mapeos y pruebas.

Matriz semántica (tono de `Status`):

| Flujo | Estado | Tono de `Status` | Significado |
| --- | --- | --- | --- |
| Pedido | Nuevo | `orange` — ámbar | Pendiente de validación por el local |
| Pedido | Confirmado | `blue` — información | Comanda enviada |
| Pedido / preparación | Preparando | `orange` — ámbar | En elaboración; coincide con el carril de Cocina |
| Pedido / preparación | Por entregar | `green` — éxito | Preparación terminada, falta servir |
| Pedido | En camino | `blue` — información | Delivery en traslado |
| Pedido / preparación | Entregado | `green` — éxito | Servido |
| Pedido | Cancelado | `red` — peligro | Cancelación |
| Preparación | Pendiente | `blue` — información | Aún sin preparar; coincide con el carril «Por preparar» |
| Cuenta | Cuenta abierta | `teal` — petróleo | Sigue admitiendo consumo; distinta de Confirmado |
| Cuenta | Por cobrar | `orange` — ámbar | Espera el pago |
| Cuenta | Pagada | `green` — éxito | Pago completo |
| Cuenta | Finalizada | `gray` — neutro | Atención terminada; distinta del verde de Pagada |
| Mesa (Salón) | Libre | `green` — éxito | Mesa sin pedido |
| Orden de compra | Borrador | `gray` — gris | Aún editable, sin aprobar |
| Orden de compra | Por aprobar / Recepción parcial | `orange` — ámbar | Espera aprobación o mercadería pendiente |
| Orden de compra | Aprobada | `blue` — información | Lista para recibir |
| Orden de compra | Recibida | `green` — éxito | Recepción completa |
| Orden de compra | Cancelada | `red` — peligro | Cancelación, igual que el pedido |
| Stock | Saludable | `green` — éxito | Existencias suficientes |
| Stock | Stock bajo | `orange` — ámbar | Por reponer |
| Stock | Sin stock | `red` — peligro | Agotado |
| Despacho a centrales | Buscando | `violet` — violeta | Oferta en curso |
| Despacho a centrales | Asignada / Recogido | `blue` — información | Central asignada y en traslado |
| Despacho a centrales | Entregado | `green` — éxito | Entrega completa |
| Despacho a centrales | Cancelado | `red` — peligro | Despacho cancelado |
| Comprobante fiscal | Emitiendo | `blue` — información | En proceso de emisión |
| Comprobante fiscal | Aceptado / Observado / Rechazado / Con error / Anulado | `green` / `orange` / `red` / `orange` / `gray` | Estados de la serie |
| Suscripción | Prueba / Activa / Vencida / Cancelada | `violet` / `green` / `red` / `gray` | Estados de plataforma |
| Pago detectado | Detectado / Asociado / Confirmado / Descartado | `orange` / `blue` / `green` / `gray` | Avisos de billeteras |
| Registros | Activo / Inactivo | `green` / `gray` | Cajas, turnos, medios y servicios |

El color representa un significado compartido, no un identificador universal:
ámbar es lo pendiente o en elaboración, azul información (confirmado, traslado),
verde éxito confirmado, rojo cancelación o error, petróleo la cuenta abierta,
violeta la oferta de despacho; el canal WhatsApp usa su verde propio
(`tone-whatsapp`). Cuenta abierta (`teal`) no comparte tono
con Confirmado (`blue`), ni Pagada (`green`) con Finalizada (`gray`).
`tests/operational-status-standard.test.mjs` verifica la paleta exacta de cada
tono (`cloud-100`, `info-bg`, `sky-100`, `violet-bg`, `success-bg`,
`warning-bg`, `teal-bg`, `danger-bg` y `cloud-150` para el gris enfatizado).

### Cocina (KDS)

Los carriles usan los alias de `variables.css`: `--kds-pending` (información),
`--kds-cooking` (advertencia), `--kds-ready` (éxito) y notas ámbar oscuro
(`--kds-note`/`--kds-note-ink`). La cabecera del carril conserva su relleno de
familia con texto e icono blancos; «Por vencer» es contorno rojo, «Con demora»
rojo sólido y el aviso dice «N con demora · M por vencer».

## Shell de navegación

- **Header**: blanco, borde inferior `line`, altura `--shell-header` (64/56),
  sticky, sin sombra. Buscador, selector de local, avisos y perfil conservan
  su comportamiento.
- **Sidebar**: blanco con borde derecho `line`, 224 px expandido (210 px ≤1100,
  drawer en ≤820). Ítems de 44 px con radio `--radius-tile`: texto `ink-600`,
  hover `cloud-100`, **activo `primary-600` con texto e icono blancos**, y
  grupo activo `primary-50`/`primary-700`. Etiquetas de grupo 11 px/600
  `ink-500`. Contadores en `danger-600`. En móvil: drawer con overlay al 40 %,
  cierre accesible y restauración del foco.
- **Header de tabla índigo**: toda tabla administrativa usa el encabezado
  `table-header-background` con texto blanco (decisión definitiva; los listados móviles
  derivados lo conservan).

## Especificaciones por pantalla

El detalle funcional de cada pantalla vive en sus módulos y pruebas; estas son
las reglas visuales compartidas vigentes con su ruta operativa:

| Pantalla | Ruta | Reglas compartidas |
| --- | --- | --- |
| Dashboard | `/dashboard` | Cifras protagonistas en `ink-950`, KPI con `StatCard` (28 px/600), señales operativas con chips de familia semántica y gráficos multicolor controlados (serie principal `primary-600`, categóricas teal/sky/morado/peach, histórico `ink-400`). |
| Salón | `/dining-room` | Plano de mesas: tarjeta blanca, casetón apenas teñido (`primary-100` al 50 %) y el plato como único color pleno (ocupado `primary-600`, libre punteado). Estados de mesa según la matriz; zonas en chips (`cmd-chip`) fijos bajo la barra en móvil (`top: var(--shell-header)`). |
| Pedidos | `/orders` | `DataTable` con encabezado índigo, pestañas de canal con tono de familia (WhatsApp en su verde propio) y detalle por capas (`DetailHero`/`DetailContact`/`Tabs`). |
| Cocina | `/kitchen` | Carriles según KDS (arriba), botones operativos de 48 px y temporizadores tabulares. |
| Cobros (`/pos`, antes «Punto de venta») | `/pos` | Importes principales en negro suave, resumen financiero en superficie clara (`cloud-25`), desglose hundido, métodos de pago con círculos Yape/Plin y confirmación índigo; el detalle de consulta es de solo lectura y el cobro conserva pantalla completa. |
| Caja | `/cash` | Turnos, pagos detectados, historial y gastos con estados de la matriz; importes a la derecha y encabezado índigo. |
| Ventas | `/sales` | Franja compacta del total sobre una superficie, periodos fijos en móvil bajo `--shell-header` y detalle por capas con pestañas en azul de familia. |
| Mesas | `/tables` | Tarjetas QR con código negro (`qr-ink`) sobre blanco y zona de silencio intacta. |
| Reservas | `/reservations` | Estados según la matriz (pendiente ámbar, confirmada información, sentada éxito, cancelada peligro). |
| Delivery | `/delivery` | Pedidos, proveedores, tarifas y entregas fallidas con estados del despacho (violeta/oferta, información/traslado, éxito/entrega, peligro/cancelado). |
| Carta y productos | `/products`, `/combos`, `/recipes`, `/availability` | Catálogo con encabezado índigo, filtros en chips y estados de disponibilidad (éxito/ámbar/peligro). |
| Inventario y compras | `/inventory`, `/stock-ledger`, `/purchases` | Stock según la matriz (éxito/ámbar/peligro), cantidades a la derecha y encabezado índigo. |
| Clientes | `/customers` | Detalle de solo lectura con `DetailHero`, `DetailContact` y `DetailFacts`. |
| Locales y Mi restaurante | `/locations`, `/settings/*` | Formularios por secciones con `FormSection` y mapa en el cuerpo desplazable. |
| Usuarios y roles | `/settings/users` | Tabla con encabezado índigo y permisos con tiles de identidad. |
| Facturación | `/receipts`, `/settings/billing` | Estados fiscales centralizados en `fiscalStatusView` (información/éxito/ámbar/peligro/neutro). |
| Plataforma | `/platform/*` | Onboarding, planes, suscripción, impuestos y canales WhatsApp con topbar de marca y estados de suscripción de la matriz. |
| Acceso | `/login` | Fondo gris claro, tarjeta blanca, franja de marca índigo y campos neutros. |

La comanda de Salón y el pedido manual comparten el catálogo con categorías de
acento estable por ID (azul índigo, celeste, violeta, petróleo) y «Guardar
comanda» en tono índigo tenue distinguible del envío primario. El panel del
reparto y el chat público conservan sus rutas públicas (`/delivery-panel`,
`/delivery-courier`, `/chat/[qr]`) con la identidad protegida correspondiente.

## Pruebas y verificación

Las puertas obligatorias (`npm run lint`, `npm run typecheck`, `npm test`,
`npm run build`) incluyen los contratos de tokens, tipografía, microinteracciones,
matriz de estados, propiedad CSS y arquitectura. Los previsualizadores manuales
de `tests/helpers/*-preview.mjs` permiten revisar capturas en desktop, tablet y
móvil (390 px incluido) tras `npm run build`.
