# Sistema de diseño administrativo

Este documento es la autoridad visual de `foods-admin-web` y no depende de
documentos externos. La primera parte fija los fundamentos (tokens, tipografía,
color, forma, movimiento y plataforma) que toda pantalla consume; la segunda
recoge las especificaciones de cada pantalla. Cuando una especificación de
pantalla contradiga un fundamento, prevalece el fundamento y la especificación
se corrige.

## Dirección

fudIA es una herramienta de operación para restaurantes peruanos: el mozo con
el celular en plena hora punta, el cajero en escritorio, el administrador
revisando el cierre. La interfaz se juzga por lo rápido que se completa la
tarea, no por la decoración.

- **Modo Operar.** Familiaridad, densidad útil, una acción dominante por
  superficie y estados siempre visibles. La identidad vive en los detalles
  precisos (color, forma, movimiento), no en adornos.
- **Identidad cromática propia.** Azul operativo índigo para acción y
  selección, navegación índigo profundo, verde restaurante para éxito
  confirmado y la franja de marca, violeta para canales digitales. Neutros
  fríos teñidos de índigo en todo el sistema, nunca grises puros.
- **Firma de marca.** La franja existente se conserva en el acceso.
  Los modales no tienen franja ni degradado: la cabecera índigo
  ocupa el ancho completo y llega al borde superior del panel.
- **Sin elementos de plantilla.** Prohibido: eyebrows en mayúsculas sobre los
  títulos, numeración decorativa de secciones, barras laterales de color en
  tarjetas o avisos, texto con degradado, sombras de halo sin desplazamiento,
  tarjetas dentro de tarjetas y monoespaciada como disfraz de dato técnico.

## Tokens

`src/styles/variables.css` es la única fuente de valores y `globals.css` la
importa una vez, después de Tailwind. Los nombres describen un rol o un paso de
escala, nunca un valor en píxeles. Las hojas de componentes y módulos consumen
`var(...)`; `tests/design-tokens.test.mjs` impide valores literales (`px`,
hexadecimales, pesos numéricos), redefinir tokens fuera de la hoja, colores
duplicados, referencias inexistentes y ciclos.

Tailwind se mantiene. El bloque `@theme inline` de `variables.css` hace que sus
utilidades resuelvan a los mismos tokens: `bg-primary-600`, `text-ink-500`,
`p-4` (16 px), `gap-3`, `rounded-lg` (tarjeta), `rounded-sm` (control),
`shadow-sm`, `text-sm`, `font-semibold`, `tracking-tight`, `ease-out`. Usar
utilidades para composición y espaciado puntual; el CSS del módulo queda para la
personalización que una utilidad no expresa con claridad. No se escriben
valores arbitrarios (`p-[13px]`).

## Color

| Rol | Tokens | Uso |
| --- | --- | --- |
| Superficie | `surface`, `cloud-25`, `cloud-50`, `cloud-100`, `cloud-150` | Paneles en blanco; fondo de página `cloud-50`; superficies hundidas y filas alternas en `cloud-25`/`primary-25` |
| Líneas | `line`, `line-strong` | Bordes de 1 px; `line-strong` en hover de campos |
| Texto e iconos | `ink-950`, `ink-800`, `ink-700`, `ink-600`, `ink-500`, `ink-400`, `ink-300` | `ink-950` títulos y datos; `ink-600` texto secundario; `ink-500` metadatos; `ink-400` es el mínimo legible (4.5:1 sobre blanco); `ink-300` solo para iconos deshabilitados |
| Azul operativo | `primary-25` … `primary-800` | `primary-600` acción primaria, selección, foco y cabecera de tabla; `primary-700` hover y texto de acción; `primary-100`/`primary-50` fondos de selección |
| Navegación | `navigation-900`, `navigation-950`, `navigation-text` | Sidebar, cabecera de pantallas completas y backdrop |
| Verde restaurante | `brand-50` … `brand-800` | Éxito confirmado, activo, pagado; `brand-600` (menta) solo en la franja de marca y el carril «Listo» |
| Violeta digital | `digital-50` … `digital-700` | WhatsApp, integraciones, «En preparación» |
| Entrega completada | `teal-100`, `teal-700` | Azul petróleo para Entregado; Finalizada usa gris neutro para distinguirla de Pagada |
| Cuenta abierta | `sky-100`, `sky-800` | Celeste tenue, distinto del índigo de Confirmado |
| Advertencia | `warning-50` … `warning-800`, `note-100`, `note-ink` | Advertencias operativas; notas de cocina |
| Error | `danger-50` … `danger-700` | Errores y acciones destructivas |

Reglas:

- Una sola acción sólida azul por superficie. Las secundarias van sobre blanco.
- El verde nunca es una acción pendiente: se reserva para estados confirmados.
  No existe botón verde.
- Rojo y ámbar solo comunican error o advertencia; nunca decoran.
- Los estados combinan color, texto e icono; nunca dependen solo del color.
- Transparencias por paso (`ink-950-alpha-8`, `primary-600-alpha-24`…), no
  porcentajes sueltos.

## Tipografía

Una sola familia, Manrope, con escala fija por rol. No hay segunda familia: los
importes, cantidades y porcentajes usan Manrope con cifras tabulares
(`font-variant-numeric: tabular-nums`, clase `.tabular`), nunca monoespaciada.

| Token | Tamaño | Rol |
| --- | --- | --- |
| `--font-size-2xs` | 10 px | Cabeceras de tabla y etiquetas en mayúsculas espaciadas |
| `--font-size-xs` (`--font-size-label`) | 12 px | Labels, metadatos, estados |
| `--font-size-sm` (`--font-size-control`) | 13 px | Controles, celdas, descripción de página (16 px en táctil) |
| `--font-size-base` | 14 px | Cuerpo |
| `--font-size-md` | 16 px | Títulos de panel y estado vacío |
| `--font-size-lg` | 18 px | Títulos de modal, importes secundarios y subtítulos de sección |
| `--font-size-xl` | 22 px | Título de página en móvil, KPI |
| `--font-size-2xl` | 28 px | Título de página |
| `--font-size-3xl` / `--font-size-4xl` | 36 / 48 px | Solo cifra principal del dashboard |

- Pesos: `--weight-regular` 400, `--weight-medium` 500, `--weight-semibold`
  600, `--weight-bold` 700, `--weight-heavy` 800. Manrope no tiene pesos
  superiores a 800. Jerarquía habitual: título 800, encabezado de sección 700,
  botones y labels 600, cuerpo 400–500.
- Interlineado: `none` 1, `tight` 1.2, `snug` 1.35, `normal` 1.5, `relaxed` 1.65.
- Tracking: `--letter-spacing-tight` (−0.02 em) en títulos de 22 px o más,
  `snug` (−0.01 em) en títulos de 16–18 px, `caps` (0.08 em) solo en texto en
  mayúsculas de 10 px. El texto en mayúsculas se escribe en caja normal en el
  JSX y se transforma con CSS.
- Tamaño mínimo 10 px (`tests/type-scale.test.mjs`). Títulos con
  `text-wrap: balance`; párrafos con medida máxima de 72 caracteres.
- Sin medios píxeles ni tamaños fuera de la escala.

## Espaciado y dimensiones

Base de 4 px con medios pasos hasta 14 px: `--space-px` (1), `--space-0-5`
(2), `--space-1` (4), `--space-1-5` (6), `--space-2` (8), `--space-2-5` (10),
`--space-3` (12), `--space-3-5` (14), `--space-4` (16), `--space-5` (20),
`--space-6` (24), `--space-7` (28), `--space-8` (32), `--space-10` (40),
`--space-11` (44), `--space-12` (48), `--space-16` (64), `--space-20` (80).
El índice equivale al de Tailwind (`p-4` = `--space-4`).

- Ritmo: grupos relacionados con 6–12 px, secciones separadas con 20–32 px;
  más espacio encima de un encabezado que debajo.
- `--size-*` se reserva para dimensiones de layout (anchos de columna, alturas
  fijas, iconos), no para padding ni gap.
- `--control-height`: 38 px con puntero fino, 44 px en móvil (≤600 px) y en
  cualquier puntero táctil. Todo control interactivo de una barra o formulario
  usa esa altura.

## Forma

| Token | Valor | Uso |
| --- | --- | --- |
| `--radius-chip` | 4 px | Marcas pequeñas, barras de skeleton |
| `--radius-control` | 6 px | Botones y campos de escritorio |
| `--radius-tile` | 10 px | Iconos contenidos, ítems internos, avisos |
| `--radius-card` | 14 px | Paneles, tarjetas, modales |
| `--radius-sheet` | 20 px | Hojas inferiores en móvil |
| `--radius-pill` | 999 px | Estados, contadores, interruptores y botones de texto en móvil |

Una superficie se separa con borde **o** con elevación, no con ambos a la vez
salvo `--elevation-1`, que es un apoyo mínimo del panel.

## Elevación y anillos

Una sola fuente de luz superior y sombras teñidas de índigo:

- `--elevation-1`: panel en reposo.
- `--elevation-2`: tarjeta interactiva, marca, logo.
- `--elevation-3`: popover, menú, tooltip, barra fija.
- `--elevation-4`: modal y drawer.
- `--elevation-up`: barras fijas inferiores; `--elevation-side`/`-side-end`:
  paneles laterales.
- `--elevation-primary`: realce teñido de la acción primaria y la página activa.
- `--elevation-selected`: tarjeta seleccionada (contorno azul + realce).
- Anillos: `--ring-focus` (foco de campos: halo tenue de 3 px al 10 % junto al borde
  `primary-600`, para que no se lea como doble borde), `--ring-danger` (foco con error),
  `--ring-selected`/`--ring-selected-soft` (selección), `--ring-line`
  (contorno interior), `--ring-success`.

El foco de teclado es único: contorno de 2 px `primary-600` con 2 px de
separación en todo control (`:focus-visible`); los campos usan además
`--ring-focus`.

## Movimiento

El movimiento comunica un cambio de estado, nunca adorna.

| Token | Valor | Uso |
| --- | --- | --- |
| `--duration-press` | 120 ms | Respuesta al presionar |
| `--duration-fast` | 160 ms | Hover, color, borde |
| `--duration-base` | 200 ms | Popovers, pestañas, interruptores |
| `--duration-slow` | 280 ms | Modales, hojas, drawers |
| `--duration-dialog` | 520 ms | Scale Punch de confirmaciones |
| `--duration-ambient*` | 750–1600 ms | Skeleton y loader |

- Curvas: `--easing-out` (`cubic-bezier(.23,1,.32,1)`) para entradas y
  respuestas; `--easing-in-out` para movimiento en pantalla; `--easing-drawer`
  para hojas y drawers; `--easing-spring` solo para realces puntuales. Nunca
  `ease-in` en interfaz.
- Todo botón, botón de icono y paso de cantidad responde al presionar
  con `transform: scale(.97)` (`.94` en controles de icono).
- Las pestañas de operación responden con cambio de superficie, sin escalado:
  se usan repetidamente y la selección por teclado debe ser inmediata.
- Pestañas generales: `Tabs` compartido, pista `cloud-150` con margen interior
  de 4 px y `radius-pill`; selección blanca con borde `primary-300` y texto azul
  `primary-700`. Icono semántico obligatorio de 16 px antes de cada etiqueta,
  del set compartido y siempre visible, también en móvil. Contadores de 12 px,
  fondo índigo `primary-600` y número blanco, de menor tamaño que la etiqueta.
  En móvil: altura mínima de 44 px y dos pestañas completas por grupo, del mismo
  ancho. Desplazamiento horizontal con ajuste al inicio de cada par, sin fragmentos
  de pestañas vecinas. Si el número es impar, la última queda sola en su grupo.
  Etiquetas largas pueden envolver; nunca truncarlas. No convertir la selección en un CTA sólido.
  `TabsBar` mantiene las acciones de la vista alineadas a la derecha en escritorio.
  En móvil, tabs a ancho completo y acciones en una fila debajo, nunca encima en
  `PageHeader`. Una acción llena su fila; dos se reparten en columnas iguales.
- Los modales entran desde `scale(.98)` y 12 px abajo; en móvil la hoja sube
  desde el borde inferior. Nada aparece desde `scale(0)`.
- Solo se animan `transform`, `opacity`, color y sombra; nunca alto, ancho o
  márgenes. No se usa `transition: all`.
- Las acciones repetidas cientos de veces (teclado, agregar a la comanda,
  pasar de página) no se animan.
- `prefers-reduced-motion: reduce` reduce todo a un fundido instantáneo.

## Capas

`--z-raised` 1, `--z-sticky` 10, `--z-header` 20, `--z-popover` 30,
`--z-tour` 60, `--z-drawer` 90, `--z-task` 100 (pantalla completa como la
comanda), `--z-modal` 110, `--z-sheet` 120, `--z-confirm` 130,
`--z-feedback` 140, `--z-loader` 200. No se escriben números sueltos.

## Iconos

Un único set propio en `design-system/icons.tsx`: 24 × 24, trazo 1.7,
extremos y uniones redondeados, esquinas suaves. `IconSize` limita los tamaños: 14 junto a texto de
12–13 px, 16 en chips, pestañas y pasos de cantidad, 18 en acciones y botones
(por defecto en `Button` e `IconButton`), 20 en navegación, 24 en estados
vacíos, 28 en el interruptor de estado.

El set es **duotono**. Cada icono se dibuja en tres capas:

- **Línea**: la estructura, en `currentColor`. En superficies neutras usa
  `--icon-line` (`ink-800`).
- **Acento** (`icon-accent` en trazo, `icon-dot` sólido): un único detalle con
  significado (el punto del pin, el badajo de la campana, la cabeza del
  usuario, las líneas del comprobante), en `--icon-accent` (`primary-600`).
- **Cuerpo** (`icon-body`): la silueta cerrada. Se rellena con
  `--icon-fill-active` (16 % del color del icono) cuando el elemento está
  activo: página actual de la navegación, pestaña activa, `aria-current`,
  `aria-pressed` o `Icon filled`.

El duotono aplica en todo el sistema. El acento no se asigna por pantalla:
`--icon-accent-auto` lo deriva del color del icono con color relativo
OKLCH, subiendo la luminosidad (`--icon-accent-lift`) y, si el color es
neutro, dándole croma índigo (`--icon-accent-chroma`). Así, un icono en tinta
recibe detalle índigo, uno índigo recibe índigo claro, uno blanco sobre fondo
de color permanece blanco, y rojo, verde, ámbar, violeta, petróleo o celeste
conservan su tono, por lo que el acento nunca mezcla semánticas. Sobre la
navegación oscura el acento es `--icon-accent-inverse` (`primary-400`).
`Status` conserva el color exacto de la matriz: su icono es monocromo. Sin
soporte de color relativo el icono se ve monocromo. Los glifos de interfaz y
de estado (`alert`, `check`, `circleCheck`, `cancel`, chevrons, `close`,
`plus`, `minus`, `menu`, `refresh`, `undo`) no tienen acento.

Un icono tiene un solo significado en toda la aplicación: `receipt` para
comprobantes y ventas, `invoice` para documentos (orden de compra, contrato,
recepción), `dish` para platos y productos de la carta, `tables` para mesas,
`note` para avisos informativos, `alert` solo para advertencias y errores. No
se usan emojis ni glifos Unicode como iconos.

## Plataforma móvil

Mobile-first desde 390 px, validado en hardware real cuando el cambio afecta
gestos, teclado o áreas seguras.

- `viewport-fit=cover` y `themeColor` índigo; barras fijas y hojas suman
  `--safe-area-top` y `--safe-area-bottom`.
- Sin destello de toque (`-webkit-tap-highlight-color: transparent`),
  `touch-action: manipulation` en controles y sin selección de texto al
  mantener presionado un botón.
- Todo `:hover` vive en `@media (hover:hover) and (pointer:fine)`: en táctil
  el hover no queda pegado. La respuesta táctil es `:active`.
- Áreas táctiles de 44 × 44 px. Los campos usan 16 px en táctil para que iOS
  no haga zoom.
- Alturas de pantalla con `dvh`, nunca `vh`. Contenedores con scroll propio
  usan `overscroll-behavior: contain`.
- Teclado correcto: `inputMode="decimal"` en importes, `numeric` en códigos,
  `type="tel"` y `type="email"` en sus campos, y sin `autoFocus` que abra el
  teclado al entrar a una pantalla táctil.
- Superficies del navegador con la paleta: selección `primary-200`, cursor de
  texto `primary-600` y scrollbars finas `ink-400-alpha-40`. La Comanda es una
  excepción aprobada: carta, categorías y resumen usan desplazamiento nativo sin
  barra visible, con acceso explícito a todas las categorías y acciones persistentes.

## Componentes base

Los contratos de cada componente viven en `04_COMPONENTS.md` y las plantillas
de página en `03_LAYOUTS.md`. Resumen:

- **Botones**: `primary`, `secondary`, `ghost`, `danger` (solo confirmar en
  `ConfirmDialog`) y `danger-outline` (disparadores destructivos). Altura de
  control, radio de control en escritorio y `radius-pill` en móvil (≤600 px),
  texto 600, icono de 18 px a la izquierda. La X y los campos conservan su radio.
- **Encabezado de página**: título igual al nombre de la navegación,
  descripción de una línea, retorno opcional y como máximo una acción primaria y
  una secundaria. Sin eyebrow.
- **Modales**: `Modal` con tamaños `sm` 480, `md` 640, `lg` 880 y `xl` 1040.
  Cabecera índigo `primary-700`, título blanco de 18 px e icono blanco de 20 px
  centrado en un espacio de 44 px, sin recuadro. Altura mínima 64 px y padding vertical de 8 px;
  puede crecer si el título necesita varias líneas. La X blanca mide 20 px,
  dentro de un cuadro visible de 32 px con solo contorno blanco de 1 px y fondo
  transparente, sin sombra. El área de clic conserva 44 px también en escritorio.
  No cambia de color con hover; conserva foco blanco visible para teclado.
  El título nombra la acción. La cabecera solo contiene icono, título y X:
  nunca subtítulos, fechas, responsables ni estados debajo del título.
  `subtitle`, `meta` y `details`, si existen, se presentan fuera de la cabecera. Cuerpo
  desplazable; pie fijo. En móvil, hoja inferior (`sm`/`md`) o pantalla
  completa (`lg`/`xl`). Escape y el fondo nunca cierran un modal; con cambios
  sin guardar, cerrar pide confirmar el descarte. Todo modal de contenido usa
  `ModalHeader` y `ModalFooter`; se exceptúan respuestas success/error y
  confirmaciones warning (`ConfirmDialog`/`FeedbackProvider`). El pie usa
  `cloud-50`; acciones contiguas a la derecha con separación `space-2-5`
  (10 px), Cancelar antes de Guardar. Nunca empujar Cancelar al extremo izquierdo.
  En móvil ambas acciones comparten fila cuando caben, con anchos iguales. Un botón
  solo en su fila ocupa todo el ancho útil, también Cancelar; si se apilan,
  cada acción ocupa el 100 %, sin reservar una media columna vacía. El estado real del registro,
  si existe, se presenta con `Status` en el resumen o pie; los formularios
  sin estado no inventan uno.
  `ModalWorkspace` añade resumen persistente solo cuando el detalle lo necesita:
  inferior a ancho completo en todos los tamaños. Los formularios
  sin resumen mantienen su cuerpo normal; no se les impone una columna vacía.
- **Jerarquía de datos en todos los modales de contenido**: etiqueta 12 px / 600
  (seminegrita, `ink-700`, `font-size-label`), valor 14 px / 400 (regular,
  `ink-950`, `font-size-base`) y título de sección 16 px / 700 (`font-size-md`).
  Esta escala informativa no crece en móvil: solo los campos editables usan
  16 px para evitar el zoom del navegador. El título del modal conserva 18 px
  y el importe principal su rol destacado. La etiqueta identifica el concepto;
  el valor es su contenido, nunca al revés. `modal.css` es el dueño de los roles
  `dt`/`dd` y `modal-data-label`/`modal-data-value`, con
  `--modal-label-weight` y `--modal-value-weight`. Los estilos locales solo
  organizan el bloque. Los controles del formulario heredan el peso regular
  mediante `--control-value-weight`, sin cambiar la tipografía fuera del modal.
  Nombres de productos y medios de pago usan 14 px / 600, sus importes 14 px / 700;
  precio unitario y datos auxiliares 12 px. Así las secciones de 16 px siguen
  encabezando el bloque, como en el detalle de Pedidos. Totales principales,
  cantidades, estados y acciones conservan su énfasis explícito; no aplicar
  la regla a toda negrita indiscriminadamente.
  En detalles de pedidos y ventas, mostrar «Mesa» si hay mesa, «Cliente» si solo
  hay nombre y «Venta» en venta directa; nunca el rótulo combinado «Cliente / Mesa».
  Se exceptúan confirmaciones, feedback y contenido de PDF/tickets.
- **Franjas de lectura en línea**: `modal-inline-metadata` es la excepción
  explícita a la escala de etiquetas de 12 px: etiqueta y valor usan 14 px y
  el mismo interlineado, con pesos 600/400 y los colores de lectura existentes.
  «Mozo» y «Sin asignar» o el nombre real tienen el mismo tamaño. Responsable
  primero a la izquierda; fecha con icono al extremo derecho mediante
  `modal-inline-date`. Si no caben, la fecha pasa a otra fila y conserva el
  borde derecho, sin recortar nombres. No cambia formularios ni la retícula
  de consulta `modal-record-data`. El dueño de esta regla es `modal.css`.
- **Tablas**: cabecera `primary-600` con texto blanco de 10 px en mayúsculas,
  filas de 56 px, cifras tabulares, hover `primary-25` y tarjetas equivalentes
  en móvil.
- **Estados**: `Status` en píldora con icono o punto semántico y texto.

### Jerarquía de acciones y botones móviles

La variante se decide por la intención, no por cuántos botones haya:

| Variante | Cuándo usarla | Ejemplos |
| --- | --- | --- |
| `primary` | Completar la tarea principal o avanzar el flujo; como máximo una por superficie | Guardar, Registrar cobro, Enviar a cocina, Confirmar entrega, Imprimir en una vista previa |
| `secondary` | Consultar información o ejecutar una alternativa a la tarea dominante | Ver resumen de pago, Descargar PDF, Editar, Agregar productos, Reintentar |
| `ghost` | Volver o abandonar un formulario sin persistir | Anterior, Cancelar |
| `danger-outline` | Iniciar una acción destructiva con confirmación posterior | Anular, Cancelar pedido |
| `danger` | Confirmar la acción destructiva dentro de `ConfirmDialog` | Confirmar anulación |

#### Colores e intención de Cancelar

«Cancelar» no siempre significa anular un registro. El color depende del efecto
real, no del texto ni de la pantalla:

| Acción real | Variante / API | Fondo · texto · borde | Confirmación |
| --- | --- | --- | --- |
| Salir de un formulario o volver sin guardar | `action="cancel"` → `ghost` | `surface` · `ink-700` · `line` | Solo si hay cambios sin guardar, mediante descarte |
| Anular un pedido/orden ya existente | `action="cancel-record"` → `danger-outline` | `danger-50` · `danger` · `danger-200` | Obligatoria, identifica el registro |
| Confirmar la anulación irreversible | `kind="danger"` | `danger` · `surface` · transparente | Es la acción destructiva del diálogo de confirmación |
| Cancelar dentro de la confirmación, evitando la anulación | `action="cancel"` o `kind="ghost"` | `surface` · `ink-700` · `line` | No ejecuta ninguna mutación |

Así, cerrar «Datos del pedido» nunca usa azul secundario ni rojo de anulación.
En el detalle de pedido, «Cancelar» es rojo tenue porque inicia la anulación;
su contexto y nombre accesible identifican el pedido, y la confirmación usa
«Cancelar pedido». No usar rojo para la salida segura de una confirmación.
Al descartar cambios, Cancelar sigue siendo neutro; la acción explícita
«Descartar» dentro de la confirmación sí comunica la pérdida de datos.

El mapa `buttonActions` de `Button` fija variante, icono y posición:
`cancel` (X inicial, neutro), `cancel-record` (icono cancelar inicial, rojo tenue),
`back` (flecha izquierda inicial, neutro) y `continue` (índigo primario, chevron
a la derecha del texto: «Continuar →»). La intención prevalece sobre `kind` o
`icon` locales; nunca se deduce del contenido de `children`. El dueño de los
colores y sus hover es `globals.css`: neutro `cloud-50`/`ink-950`/`line-strong`,
rojo tenue `danger-100`/`danger`/`danger-600`, primario `primary-700`/`surface`.
Disabled conserva su familia y bloquea la interacción; no se recolorea como
otra acción. Los estados de negocio mantienen su matriz independiente.

Un detalle de solo lectura puede tener únicamente una acción secundaria: no se
convierte en primaria por ser la única. «Ver resumen de pago» usa `eye`, abre
el documento operativo existente y no cobra ni emite un comprobante fiscal.
No se usa «Ver ticket de pago» como etiqueta del botón.

Todos los `Button` de texto usan `radius-pill` en móvil (≤600 px), incluidos
primarios y secundarios, con la altura táctil compartida de al menos 44 px.
La regla vive en `globals.css`, no en cada pantalla. Campos y X del header
mantienen `radius-control`; no se redondea el contorno de cierre a un círculo.
El orden, foco, progreso y permisos no cambian al pasar a móvil.

El ancho depende de la fila real, no de una rejilla con huecos: dos acciones
que caben en la misma fila se reparten el ancho por igual; una acción sola
llena el 100 % del ancho útil. Al apilar acciones, todas llenan su fila.
Para pies con grupos anidados se usa `modal-action-layout` y
`modal-action-subgroup`; `modal-action-main` reserva una fila completa a la
principal cuando la distribución lo requiere, como en Pedidos/Salón.
Así, Cancelar + Enviar comanda apilados ocupan ambos el ancho completo; dos
secundarias que caben juntas comparten fila. En escritorio permanecen
contiguas a la derecha y con ancho natural. No afecta iconos, campos, X,
`ConfirmDialog` ni respuestas de `FeedbackProvider`.

### Colores de operación: pedido, preparación y cuenta

Estándar aprobado y obligatorio para todas las pantallas, listados y modales que
presenten estos estados. La semántica de Pedidos y Salón se centraliza en
`orderStatus` de `order-meta.ts`; Cocina/Barra usa `serviceStatusTone` y la cuenta
usa `accountTone` de `order-service-controls.tsx`, sin reinterpretar etiquetas del API.

Reglas de aplicación:

- Reutilizar `Status` y los tokens de `src/styles/variables.css`; no crear badges
  propios, colores hexadecimales locales ni mapeos distintos por pantalla.
- Siempre fondo tenue y texto/icono oscuro, con contraste mínimo 4.5:1.
  No usar relleno sólido con texto blanco, sombras ni hover en estados de lectura.
- `emphasis` refuerza icono y peso del texto, sin convertir el estado en botón
  ni alterar su familia de color. La variante gris enfatizada se especifica abajo.
- Cuenta abierta es celeste y Confirmado índigo. Pagada es verde y Finalizada
  gris: estas parejas deben seguir siendo visualmente distintas.
- Mostrar siempre la etiqueta y su contexto (pedido, preparación o cuenta):
  el color nunca es la única información del estado.
- Cambiar esta semántica requiere aprobación explícita y actualizar juntos esta
  matriz, tokens, mapeos y pruebas. No modificar estados ni flujos de negocio
  para adaptar el diseño.

Paleta exacta de `Status` (fondo / texto e icono):

| Tono | Fondo tenue | Texto e icono |
| --- | --- | --- |
| `gray` | `cloud-100` · `#F0F2F8` | `ink-500` · `#59617C` |
| `blue` — índigo | `primary-100` · `#EEF0FF` | `primary-700` · `#3946B8` |
| `sky` — celeste | `sky-100` · `#E0F2FE` | `sky-800` · `#075985` |
| `violet` | `digital-100` · `#F2ECFF` | `digital-700` · `#5421A8` |
| `green` | `brand-100` · `#E8F7F1` | `brand-800` · `#0E6646` |
| `orange` — ámbar | `warning-100` · `#FFE8CF` | `warning-800` · `#7A3B0D` |
| `teal` — azul petróleo | `teal-100` · `#E6F5F7` | `teal-700` · `#0B6B78` |
| `red` | `danger-50` · `#FFF1EF` | `danger-700` · `#A5281E` |
| `gray emphasis` | `cloud-150` · `#E9ECF4` | `ink-800` · `#222842` |

Matriz semántica completa:

| Flujo | Estado | Tono de `Status` | Significado |
| --- | --- | --- | --- |
| Pedido | Nuevo | `gray` — gris | Registrado, aún sin confirmar |
| Pedido | Confirmado | `blue` — índigo | Comanda enviada |
| Pedido / preparación | Preparando | `violet` — violeta | En elaboración; coincide con el panel de Cocina |
| Pedido / preparación | Listo para entregar / Listo | `green` — verde | Preparación terminada, falta servir |
| Pedido | En camino | `orange` — ámbar | Delivery en traslado |
| Pedido / preparación | Entregado | `teal` — azul petróleo | Servido; distinto de Listo y de Pagada |
| Pedido / preparación | Cancelado | `red` — rojo | Cancelación |
| Preparación | Pendiente | `orange` — ámbar | Aún sin preparar; distinto del celeste de Cuenta abierta |
| Entrega directa | Pendiente de entrega (`listo`) | `green` — verde | Disponible para servir; el API conserva esa etiqueta |
| Cuenta | Cuenta abierta | `sky` — celeste tenue | Sigue admitiendo consumo; distinto de Confirmado |
| Cuenta | Por cobrar | `orange` — ámbar | Espera el pago |
| Cuenta | Pagada | `green` — verde | Pago completo |
| Cuenta | Finalizada | `gray` — gris neutro | Atención terminada; visualmente distinta del verde de Pagada |

El color representa un significado compartido, no un identificador universal:
verde es un hito cumplido, ámbar una resolución pendiente y azul petróleo una
entrega completada. La cuenta finalizada usa gris neutro. El nombre y la región distinguen los flujos.
El pedido tiene siete tonos distintos. Preparación y cuenta conservan su ubicación
separada. Preparación y cuenta usan las mismas píldoras suaves del componente
`Status`, sin borde exterior. Dos productos con el mismo estado mantienen el
mismo color; no se colorean por producto. `teal-100` / `teal-700` son tokens compartidos,
también accesibles en Tailwind.

## Especificaciones por pantalla

### Comanda de Salón, Pedidos y pedido manual

La tarea a pantalla completa conserva la cabecera índigo homologada, con icono,
título y X. El catálogo presenta buscador, categorías del API y acceso explícito
a «Menús y combos». Las categorías usan una sola fila deslizable con el dedo,
sin barra visible; se conserva el desplazamiento nativo y el foco de teclado.
«Todo» y «Menús y combos» preceden a las categorías del API. El
botón persistente «Categorías» abre la lista completa (incluidos menús y combos),
sin depender de descubrir el gesto de deslizar. Elegir una opción cierra esa
lista, limpia la búsqueda y muestra la categoría seleccionada en la tira.
En móvil los controles tienen áreas táctiles de 44 px; la lista conserva espacio
de lectura también en altura reducida, donde se omite su encabezado auxiliar.

Bajo la cabecera, el contexto de la comanda se presenta como datos con icono de 14 px,
sin separadores «·» ni píldoras (para no confundirse con los chips de categoría):
`mapPin` zona, `users` personas y `user` cliente en Salón; `phone`, `mapPin` dirección y
`note` referencia en delivery. Texto `ink-700` seminegrita; los datos vacíos se omiten.
En móvil (≤600 px) la línea ocupa todo el ancho: el primer dato (zona) queda a la
izquierda y los siguientes (personas, cliente) se alinean a la derecha.

Productos y combos muestran foto propia de 64 px si existe e icono del sistema como
alternativa, sin imágenes inventadas. Nombre y precio tienen jerarquía clara;
el nombre usa seminegrita y el precio negrita oscura, juntos junto a la foto.
Las filas usan separadores suaves, sin tarjetas ni fondos de selección.
La cantidad se acota a las existencias informadas por la carta (`remaining`: porciones
del día o inventario físico) más lo ya reservado por el pedido guardado que se edita;
los productos sin control no tienen tope. Al intentar superar ese tope con el + de la
carta o de la línea, la cantidad no cambia y se abre el aviso global de validación
(`FeedbackProvider`, tono `info`, que no cierra la comanda): «Cantidad no disponible ·
La cantidad excede las existencias: solo quedan N unidades de {producto}». Al llegar al
tope, la línea muestra «Máximo disponible: N» en `warning-800`. El backend vuelve a
validar al guardar y responde con el mismo criterio («La cantidad excede las existencias:
solo quedan N porciones/unidades de {producto}»; «{producto} está agotado por hoy» si
se agotó manualmente).
El + agrega una unidad; al seleccionar un producto aparece `NumberStepper appearance="soft"`
con disminuir, cantidad y aumentar. En una unidad, disminuir pasa a Quitar.
Un producto agotado impide agregar pero permite reducir lo ya seleccionado.
Los combos conservan su configuración de opciones, sin restar elecciones
distintas como si fueran un producto simple. Los estados de disponibilidad no cambian.
La composición conserva blanco (`surface`) para lectura, cápsulas gris frío
(`cloud-100`) para cantidades e índigo tenue (`primary-50`) en el pie y la barra
de subtotal. El importe y el + usan índigo; los iconos alternativos usan
`primary-100`/`primary-700` para productos y `digital-100`/`digital-700` para combos.
Las categorías tienen cuatro acentos de navegación de la paleta: índigo
(`primary-100`/`primary-700`), celeste (`sky-100`/`sky-800`), violeta
(`digital-100`/`digital-700`) y petróleo (`teal-100`/`teal-700`). Se asignan
establemente por ID, no por nombre ni orden del API; las familias pueden repetirse.
«Todo» usa índigo y «Menús y combos» violeta. La opción seleccionada no invierte a
relleno sólido: toma el tinte un paso más marcado de su familia (`primary-200`,
`sky-200`, `digital-200`, `teal-200`) con texto oscuro de la familia (`primary-800`,
`sky-800`, `digital-700`, `teal-700`) y el check blanco dentro de una insignia circular
sólida de 18 px en ese mismo color oscuro (`cmd-chip-check`), además de `aria-pressed`;
la lista completa repite el acento mediante un punto y la selección con check.
Estos colores identifican navegación, no estados del negocio. Todos los pares
de texto/fondo cumplen contraste AA de 4.5:1. Las cantidades
usan cápsulas gris tenue sin borde ni separadores interiores y mantienen 44 px
por botón. No se cambia la matriz de colores semánticos de disponibilidad.
Carga, error recuperable y vacío permanecen explícitos.
Escritorio conserva catálogo y resumen lateral; móvil mantiene el catálogo
completo, barra inferior con importe y «Ver comanda», y resumen en hoja.
«Ver comanda» usa `radius-pill` en móvil, icono lineal `shoppingBag` y altura mínima de 48 px,
conservando importe y contador visibles; no cambia su acción de abrir el resumen.
La hoja móvil se pliega mediante el chevron hacia abajo de 24 px dentro de un
control circular de 44 px, fondo `primary-100`, sin borde y color `primary-700`,
sin sombra
ni cambio de color al hover. Su nombre accesible es «Volver a la carta»;
no guarda, descarta ni cierra la comanda y conserva productos y notas.
La cabecera índigo queda por encima del scrim móvil: conserva título, icono y X
blancos, sin oscurecerse al abrir la hoja. El scrim atenúa únicamente el fondo
operativo y mantiene el cierre de la hoja al tocar fuera de ella.
El resumen conserva la misma composición: foto, nombre/precio juntos y cantidad
a la derecha; en el panel lateral estrecho la cantidad baja sin comprimir el dato.
Las imágenes del borrador son solo presentación y no se envían al API. Las
líneas históricas sin imagen muestran el icono de su tipo; no se consultan fotos por línea.
Carta y resumen se desplazan sin barras visibles. Esta excepción local no oculta
las barras de otros módulos ni de la lista completa de categorías.
La revisión precede al guardado/envío desde la hoja; la barra no envía directamente.
«Guardar comanda» crea un pedido en estado Nuevo sin activar la preparación;
«Enviar a cocina» lo confirma y activa su atención. Ambos persisten
el pedido y cierran la comanda al completarse; guardar no es almacenamiento
local: también reserva las cantidades de inventario según el flujo vigente.
Guardar usa el portapapeles `orders`, fondo `primary-200`, texto `primary-700` y borde
`primary-600`, distinguible del footer `primary-50` sin sombras; enviar conserva
el índigo sólido y el avión de envío `send`. En móvil, ambas acciones usan
`radius-pill`, con altura táctil mínima de 44 px; escritorio conserva su geometría.
Al guardar una comanda solo ese botón
muestra «Guardando…», sin presentar «Enviando…» en la otra acción. Ambas acciones
se bloquean durante la petición. Los avisos dicen «Comanda guardada» (sin enviar
a cocina) o «Comanda enviada», según la acción, sin cambiar los estados del API.
En móvil los botones del resumen ocupan filas completas cuando sus etiquetas no
caben juntas; no se recortan los verbos ni se oculta el envío detrás del contenido.
No se abren resumen ni teclado al agregar un producto. No se cambian precios,
permisos ni el contrato de envío a cocina.

### Salón (tablero de mesas)

Las zonas se filtran con la misma tira de chips en píldora de las categorías de la
Comanda (`cmd-chips`/`cmd-chip`): una fila deslizable sin barra visible, «Todas»
primero, chips de 44 px y la misma selección tenue (`primary-200`, texto
`primary-800` e insignia de check `cmd-chip-check`, con `aria-pressed`); las demás conservan
`primary-100`/`primary-700`. Todas
las zonas usan índigo: los acentos celeste, violeta y petróleo de las categorías ya
significan estados en esta pantalla. En móvil (≤600 px) la búsqueda se desplaza con la
página y la tira de zonas queda fija bajo la barra superior (`top` igual a su altura),
con fondo `cloud-50` a ancho completo y `scroll-padding` para que el primer chip
conserve el margen lateral; escritorio conserva búsqueda y zonas en la misma tarjeta.

Las tarjetas de mesa tienen la misma altura en toda la grilla (filas `1fr`); el
importe o «Abrir mesa» queda anclado abajo. Estado y tiempo de ocupación pasan a una
segunda línea cuando no caben juntos. Debajo de 480 px el resumen Mesas/Libres/Ocupadas
oculta sus iconos para que las tres cifras quepan.

### Ticket de pago para el comensal

Tras confirmar el cobro y cerrar el aviso global se abre el PDF real del ticket.
Formato de rollo de 80 mm, área útil de 70 mm, negro sobre blanco, nombre del
restaurante, datos del local, fecha del cobro, mesa/cliente, consumo con cantidad,
precio e importe, total y medios de pago reales. Los pagos parciales muestran
saldo y nunca se presentan como cuenta pagada. Se identifica como no fiscal;
no inventa serie, impuesto, QR tributario ni aprobación SUNAT. La impresión es
explícita desde «Imprimir ticket»; el documento aislado comparte las líneas
medidas del PDF y no incluye la interfaz. La vista previa conserva descarga,
reintento, skeleton, X y controles de 44 px en móvil. El cierre de caja A4 no cambia.

### Resumen de cierre e informe de caja

El cierre presenta identidad del turno y tres importes destacados, seguido de
secciones desplegables para productos, cobros, egresos, otros ingresos y arqueo.
Cada línea conserva fecha, cantidad, precio y responsable; no se resume perdiendo
filas. Se reutilizan Modal, botones, tabla y tokens globales. A 390 px los datos
de identidad/importes se apilan y las tablas tienen scroll horizontal controlado;
las acciones mantienen 44 px. El cierre ciego conserva el conteo sin revelar
importes y presenta el informe después de confirmar.

El modal de cierre limita su altura al viewport dinámico, incluidos los márgenes
del fondo. Cabecera y pie con «Cerrar turno» siempre permanecen visibles;
solo el contenido se desplaza, con barra visible cuando hay desborde. No envía
al pulsar Enter desde los campos de conteo; la observación conserva saltos de
línea y la acción requiere activar explícitamente su botón.

El informe PDF usa el logo propio de Fudia, los colores del sistema y A4
horizontal con márgenes, texto seleccionable, cabeceras repetidas y numeración.
Los nombres largos se envuelven sin recortar filas.
La vista previa permite descargar o ver el detalle HTML si el navegador móvil
no integra un visor PDF.

### Layout general

- Sidebar: Control, Operación, Carta y producción, Abastecimiento y Configuración.
- El catálogo general se denomina «Carta y productos» y usa el icono de
  cubiertos. «Menús y combos» identifica únicamente productos compuestos;
  «Almuerzo» no se usa como módulo porque excluiría desayuno, cena y otros
  momentos de venta. Los iconos representan el dominio de cada opción y el
  icono de tres líneas se reserva para abrir o cerrar la navegación.
- Topbar compacta; empresa, local y usuario aparecen una sola vez.
- La empresa activa se muestra junto al icono de navegación, sin depender del
  rol. El botón de contexto muestra únicamente el local activo, también para
  Plataforma; el rol permanece debajo del usuario. El administrador de
  plataforma conserva la selección de empresa dentro del popover.
- El selector de empresa/local usa un popover compacto con cabecera y cierre X neutro y redondo,
  campos `FormField`/`Select` y acción «Aplicar» azul `primary-600`; no emplea
  un botón verde propio. Etiquetas en caja normal, espaciado de 16 px y altura
  única de control. Su carga, error, vacío y adaptación móvil se especifican
  en `03_LAYOUTS.md`.
- Dashboard separa el análisis del periodo (Hoy / 7 / 30 días) de la operación en vivo y las alertas.
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
- La imagen abre el formulario de producto junto a la identidad comercial: un
  cuadro 1:1 de 176 px a la izquierda y, a su derecha, Nombre, Precio,
  Categoría y Destino de atención. En el DOM los campos van primero (el nombre
  conserva el foco inicial); el cuadro se ubica a la izquierda por grid.
  El cuadro siempre conserva el borde punteado (`--line-strong`) para indicar
  que acepta arrastrar un archivo. Sin imagen, fondo `cloud-25`, icono `upload`
  en círculo `primary-100`, «Arrastra tu foto aquí», «o haz clic para elegirla»
  y «JPEG, PNG o WebP · 5 MB»; en táctil (`hover: none`) dice «Toca para elegir
  una foto». Con imagen, fondo `surface`, 6 px de margen interno y la foto con
  `object-fit: cover` dentro del marco punteado. Debajo van «Cambiar» (secondary, ancho flexible)
  y un IconButton «Quitar imagen» de 44 px; no se superpone una X sobre la foto.
  Todo el cuadro abre el selector y también acepta arrastrar y soltar un
  archivo (con la misma validación). Mientras se arrastra, el borde pasa a
  `primary-600` punteado sobre `primary-50`. En móvil (≤600 px) el cuadro de 176 px se
  centra arriba del formulario y los campos ocupan el ancho completo debajo.
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

- «Disponibilidad» pertenece a «Carta y producción» e incluye platos, bebidas,
  productos, menús y combos del local activo. La disponibilidad cotidiana no
  se modifica dentro del wizard. Su composición operativa usa cinco columnas
  compartidas por cabecera y filas: Producto, Disponibilidad, Cupo de hoy,
  Unidades de hoy y Acciones. Categoría y control pertenecen a la identidad
  del producto, no a columnas separadas. El nombre usa 16 px/700, categoría
  13 px/400 y control 12 px/400; la foto propia o el icono alternativo ocupa
  48 px. La cabecera índigo `primary-600` usa etiquetas de 14 px/600 en caja
  normal e iconos semánticos. Esta escala es propia del listado operativo,
  no redefine las tablas de gestión. `dish`, `cookingPot`, `stock`, `gauge`,
  `calendar` y `clock` proceden del set compartido.
  Unidades de hoy reúne Vendidas (14 px/400) y Restantes (16 px/700) con
  etiquetas de 12 px y cifras tabulares. Inventario conserva hasta tres
  decimales y su unidad; un saldo nulo muestra «Sin dato», nunca cero.
  El origen de inventario se explica en el lugar del cupo: «Desde Inventario»
  y «Se actualiza con las entradas y ventas». Sin control ocupa el espacio
  conjunto de cupo y balance con «Sin límite de cantidad» y su explicación;
  la disponibilidad derivada indica su cálculo automático sin inventar
  cantidades ni repetir guiones. `Status` conserva sus tonos: Disponible
  verde, Pocas unidades ámbar, Agotado y Fuera de horario gris, siempre tenues.
  Las aclaraciones «Agotado manualmente» y «Calculado automáticamente» solo
  aparecen cuando aportan contexto. Los avisos de horario, componentes, cupo
  o stock agotado son compactos junto al estado, con `warning-50`, icono y
  texto `warning`; no ocupan una franja de fila completa ni llevan acento lateral.
  La barra contiene búsqueda y categoría remota; en solo lectura añade el
  indicador con candado. La fecha operativa usa `calendar` en el encabezado.
  Guardar cupo es un `IconButton` primario junto al campo: aparece únicamente
  tras editar, tiene nombre accesible completo y acompaña «Cambio sin guardar».
  La validación en línea identifica el mínimo y bloquea Guardar si el valor
  no es entero o es menor al mayor entre 1 y las porciones ya vendidas.
  «Agotar hoy» y «Reactivar» son alternativas secundarias: la primera usa
  texto y borde `warning`, fondo blanco y hover `warning-50`, sin rojo ni
  relleno ámbar sólido; Reactivar conserva índigo. Historial usa `clock`,
  tooltip y nombre accesible en escritorio; en contenedores estrechos muestra
  también la etiqueta. Guardar es la única acción sólida y solo existe mientras
  hay un cupo editado. Los valores de columnas y controles se centralizan en
  `variables.css`: controles de 44 px en escritorio/tablet y 48 px en móvil.
  Hasta 1020 px de ancho del panel, las filas pasan a tarjetas sin scroll
  horizontal: identidad y estado arriba, cupo y balance debajo, acciones al pie.
  Dos acciones comparten columnas iguales y una sola llena su fila. Hasta
  520 px de contenedor, identidad y estado se apilan y el balance presenta
  parejas etiqueta/valor. Se conservan skeleton con la misma composición,
  error recuperable, vacío explícito y paginación compartida desde el API.
  «Pocas unidades» es calculado, nunca una acción manual. Horario y componentes
  obligatorios bloquean acciones manuales. Un agotado por cantidades sin
  override manual oculta Agotar/Reactivar y dirige al ajuste de cupo o entrada
  de inventario. Si existe un override Agotado, Reactivar retira únicamente
  ese override: el cálculo por cupo o stock sigue vigente. Guardar, Agotar y
  Reactivar mantienen el formulario de motivo
  obligatorio antes de enviar. Guardar modifica únicamente el cupo escrito;
  Agotar/Reactivar modifican únicamente el override manual y no persisten un
  cupo pendiente. `menu.read` sin `menu.manage` conserva consulta e historial,
  con campos bloqueados y sin acciones de escritura. Empresa, local, contrato
  API, invalidaciones y flujo de historial permanecen independientes del layout.
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
  zonas: datos de la receta e insumos. La cabecera sigue el patrón compartido:
  título de 18 px, icono de 18 px en superficie de 44 px y cierre con
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
- El producto solicita el control mediante radios «Sin control» («Se vende sin
  contar unidades») o «Cupo del día» («Platos o bebidas con cantidad limitada,
  sin usar Inventario»). El nombre no habla de porciones porque también sirve
  para una gaseosa o un producto de reventa en planes sin Inventario. El valor
  técnico sigue siendo `portions`. Al crear, «Cupo del día» revela y exige
  «Cantidad disponible hoy», un entero positivo. Producto y cantidad inicial se guardan juntos para
  el local activo y su fecha actual; no es un cupo predeterminado ni se repone
  automáticamente. «Sin control» no muestra ni envía cantidad. La edición
  comercial no modifica porciones: Disponibilidad permite ajustar su cantidad,
  marcar Agotado o Reactivar. Se reutilizan FormField e Input, sin nuevas tarjetas,
  títulos ni pasos; el campo ocupa el ancho del formulario también desde 390 px.
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
- «Reservar» se precarga con la cantidad disponible del producto en el local,
  tanto en la plantilla como al agregar una opción. Permite reducir el valor,
  pero no superar el saldo ni introducir fracciones. Cero indica agotado; un
  producto sin control conserva el campo vacío con «Sin límite», sin simular cero.
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
- El detalle de cobro reutiliza `ModalWorkspace`, `ModalItemList` y
  `modal-summary`: una columna de contenido y resumen inferior fijo, nunca
  tres tarjetas de importes encima de los productos. La cabecera muestra
  «Detalle de cobro», icono y X; el código va fuera de ella. Mesa y canal
  comparten la primera fila de datos; atendido por y fecha, la segunda para
  Salón. En móvil (≤620 px), `modal-record-data` presenta una lista compacta:
  Mesa, Canal, Atendido por y Fecha del pedido, una pareja etiqueta/valor por fila.
  La columna de etiquetas mide 120 px; los valores comparten el mismo inicio,
  con 12 px entre columnas y 10 px entre filas. No anclar Canal al extremo derecho
  ni alternar etiquetas arriba y al lado del valor. Los nombres envuelven sin recortes.
  La etiqueta indica el dato
  real: «Mesa» si hay mesa, «Cliente» si solo hay cliente y «Venta» en venta directa;
  nunca «Cliente / mesa». Las etiquetas usan tamaño auxiliar, seminegrita e
  `ink-700`; los valores, 14 px, peso regular e `ink-950`.
  Los encabezados de sección usan negrita para conservar un nivel superior.
  Encargado y fecha comparten la misma columna de
  valores para evitar saltos de alineación. No se añade un color de estado al canal.
  El importe de cada producto conserva su posición derecha. El modo de pantalla
  completa empieza a 600 px, coincidiendo con el backdrop sin padding; por encima
  el panel usa el ancho disponible y descuenta los márgenes en su altura máxima.
  Nunca combina `100vw` con un backdrop con padding ni deja el resumen fuera
  del viewport, también con zoom. Productos y pagos tienen
  un único título, sin «CONSUMO», subtítulos redundantes ni contadores de líneas.
  Las cantidades conservan su círculo. Sin pagos, un mensaje breve sustituye
  el panel vacío grande. Una línea neutra y espacio antes del título separan
  los productos de los pagos, incluso cuando no hay pagos registrados.
  Total, pagado neto, saldo y estado real quedan juntos
  en el resumen; el ticket permanece en el pie y las devoluciones en su pago,
  respetando permiso, turno y estado. Loading reproduce cuerpo y resumen;
  errores y ausencia de detalle no muestran importes o estados anteriores.
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
- Contraste WCAG AA y foco visible con `primary-600`.
- Alertas de éxito, error e información usan un diálogo global centrado, icono
  semántico, título, explicación breve y acción «Aceptar». El botón «Aceptar»
  usa `--control-height` como todos los demás botones. El comportamiento y
  jerarquía siguen el estándar de Bodegas, reinterpretado con tokens Foods.
- En la respuesta de éxito, el círculo de 76 px es verde restaurante `brand-700`, con
  check blanco `surface` de 36 px centrado. Es una señal de resultado, no una
  variante sólida de los chips de estado, que mantienen sus fondos tenues.
  Toda la composición conserva la paleta Foods: círculo `brand-700`, título
  `ink-950`, superficie `surface` y texto secundario `ink-600`. «Aceptar» es
  cierre secundario neutro, blanco con borde `line-strong`, nunca un CTA menta.
  No lleva otro check redundante y usa `radius-pill` en móvil.
  Las referencias externas aportan movimiento, nunca sus colores ni valores
  hexadecimales locales; no se redefine la paleta para este modal.
  El panel deja 64 px sobre el círculo, debajo de la fila de la X, y 32 px al
  pie. Título de 18 px / 700 y explicación de 14 px: el resultado tiene mayor
  jerarquía que la descripción. El panel solo se funde en 280 ms (`easing-out`);
  no escala junto al icono. La forma verde usa una capa independiente del check:
  comienza un 30% más grande (`scale(1.3)`) y se contrae hasta `scale(1)`
  con un rebote leve de recuperación. Conserva `border-radius: 50%` y la
  misma escala en ambos ejes durante toda la entrada: nunca se aplana,
  alarga, desplaza ni cambia de contorno. Solo se anima transform en esa
  capa de 76 px, con `easing-spring` y `duration-dialog` (520 ms), sin
  dependencias, esperas ni bucles. Cada respuesta reinicia
  la entrada del símbolo. El check blanco de trazo `stroke-3` permanece centrado
  y sin deformarse; se funde en 200 ms desde los 280 ms, sin barrido lateral.
  Los textos permanecen quietos, sin esperas independientes. La secuencia se
  asienta en 520 ms; la salida del panel es un fundido de 160 ms.
  Movimiento reducido conserva solo un fundido breve, sin expansión ni rebote.
  Error, información y confirmaciones conservan sus variantes independientes.
  Referencia de secuencia observada: [Rappi, confirmación alrededor de 4:57](https://www.youtube.com/watch?v=FEit2N2jcYg&t=297s).
  Se adapta el símbolo de confirmación, no la marca, pantalla verde completa
  ni una carga simulada: el feedback aparece solo después de confirmar el API.
- Éxito/error sustituyen los diálogos de la acción: se cierran primero mediante
  `Dialog.onResponseClose` y luego se muestra la respuesta real del API. No hay
  formularios desenfocados detrás del aviso ni confirmación adicional de descarte.
  La validación previa se muestra junto a los campos, sin modal de respuesta; la única
  excepción es superar las existencias en la Comanda, que usa el aviso `info` (ver Comanda).
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
  principal («Guardar comanda» frente a «Enviar a cocina», o «Guardar borrador»
  frente a aprobar una orden de compra), el sufijo distingue la intención.
  Guardar comanda sí persiste un pedido; no activa todavía la preparación.
- Los botones «Guardar» y «Cancelar» de un mismo footer tienen idéntica altura
  (`--control-height`). El ícono de «Guardar» es un check (✓), no un disquete.
- La cabecera de los modales de registro usa el patrón compartido: padding
  horizontal de 24 px en escritorio y 16 px en móvil, título h2 de 18 px e
  icono blanco de 20 px sin recuadro, en un espacio de 44×44 px. La X tiene
  contorno blanco de 32×32 px y área táctil de 44×44 px; no se fijan dimensiones
  locales para reducir la cabecera. Durante el guardado la X conserva su blanco
  y opacidad completa, aunque esté deshabilitada para evitar cierres prematuros.
  Formularios como Producto usan `ModalForm` y `ModalBody` para que solo el cuerpo
  se desplace; no mezclan un formulario sin padding con los márgenes negativos
  del patrón heredado. El pie mantiene 24/16 px horizontales y 10 px entre acciones.
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
usan ancho completo. Si el footer permite salir sin guardar, Cancelar es neutro; lleva una única acción
primaria con el icono `send`: «Enviar a cocina». Durante el envío muestra
«Enviando…» y bloquea edición.

## Tablas y paginación

- Cabecera azul `primary-600`, texto blanco en mayúsculas, filas alternas sutiles y
  acciones iconográficas consistentes; el color semántico se reserva para estados.
- Escala tipográfica única para todas las tablas, definida en
  `src/design-system/styles/data-table.css` y los tokens, nunca sobrescrita por módulo: cabecera y líneas secundarias usan
  `--font-size-label` (12 px); celdas, paginación y controles usan
  `--font-size-control` (13 px en escritorio). La cabecera conserva peso 800.
- La paginación informa el rango visible y total, permite 10, 20 o 50 filas y
  muestra páginas, elipsis, anterior y siguiente con estado activo inequívoco.
- Toda tabla de gestión incluye paginación, incluida la de categorías.
- La paginación es una primitiva única y compartida. No se replica su cálculo,
  marcado ni estilos dentro de los módulos de negocio.
- No conservar paginadores antiguos ni sus reglas (`catalog-pagination`, `pagination-meta`, estilos
  genéricos de `.management footer` o flechas generadas con pseudo-elementos).
  La geometría base, los controles y la adaptación móvil de `Pagination` pertenecen
  exclusivamente a `src/design-system/styles/pagination.css`, compartido por todas
  las tablas. No se corrigen conflictos con excepciones locales ni `!important`:
  Anterior/Siguiente activos conservan fondo primario y texto blanco;
  deshabilitados usan fondo y texto neutros, con radio y altura estándar.
- El botón «Nuevo» va alineado a la derecha de los tabs de producto/categoría,
  no en el encabezado de página. En móvil se apila debajo de los tabs.
- Las tablas informativas conservan filas y columnas en móvil, con desplazamiento
  horizontal accesible por toque y teclado, sin ocultar la barra de desplazamiento.
  `DataTable` ofrece el contenedor con nombre accesible; su CSS reside en
  `src/design-system/styles/data-table.css`, y Tailwind compone el wrapper.
  El CSS compartido se registra una vez desde el layout para mantener las tablas
  heredadas durante la migración. La paginación móvil es compacta.
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
- Campos y controles estructurales conservan radio de 6 px; los botones de
  texto usan 6 px en escritorio y `radius-pill` en móvil. Los 14 px se reservan
  para tarjetas y paneles.
- En escritorio, los campos usan 38 px de altura,
  radio de 6 px, borde neutro y texto de 13 px. En móvil aumentan a 44 px para
  conservar el área táctil. El foco usa anillo azul Foods y el error rojo semántico.
- Los inputs compuestos con prefijo (montos, «S/») declaran el borde en el
  contenedor, no en cada parte; el prefijo y el input interno no duplican ni
  omiten bordes. El foco se aplica al contenedor con `:focus-within`.

## Responsive

Disponibilidad reorganiza sus cinco columnas según el ancho del panel, con
la acción `clock` «Historial» accesible también en solo lectura. Guardar cupo,
Agotar y Reactivar abren un formulario CRUD
compacto con Motivo del cambio obligatorio; ajustar cupo muestra anterior/nuevo
como lectura. No añade confirmaciones sucesivas ni catálogos locales de motivos.
El historial reutiliza tabla, fecha regional y paginación del sistema: Fecha y
hora, Cambio, Cantidad, Motivo y Usuario, con X como único cierre. Los valores históricos
desconocidos se explican sin mostrar ceros ficticios. En móvil, motivo ocupa el
ancho completo y el historial usa scroll horizontal controlado dentro del modal.

Toda pantalla se valida desde 390 px. En móvil se reorganizan filtros, acciones
y detalle; no se limita a apilar columnas de escritorio.

### Presentación del artículo en Compras

El registro muestra «Unidad de inventario», «Presentación de compra» y su contenido, sin filas de
«Otra presentación», radios de «Predeterminada» ni «Nueva combinación».
Presentación y contenido comparten dos columnas en escritorio y se apilan en
móvil; se evita una tarjeta adicional dentro de la sección existente. El +
de Unidad de inventario y el de Presentación de compra reutilizan el mismo `IconButton`, contenedor
y separación del selector; ambas filas comparten ancho y rejilla, sin márgenes
adicionales. El + junto al selector de Presentación abre un diálogo
breve con Nombre únicamente, X, Cancelar y Guardar. Códigos y preferencias
globales no se solicitan al usuario. Las opciones provienen del backend; las
combinaciones habituales ya se inicializan al crear la empresa. La conversión
se completa por artículo y se explica con una equivalencia breve, sin adivinar
cantidades ni modificar el historial de compras. La compra individual muestra
el nombre remoto de la unidad (por ejemplo, Botella), nunca «Unidad base»;
conserva el código técnico `unit` y contenido 1. Los mismos términos se utilizan
en las órdenes y los ajustes de inventario, sin cambios del contrato ni de stock.

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

- Fondo `navigation-900`, opción activa con el azul operativo vigente e iconos contenidos en una caja estable.
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
estados confirmados, no para representar una acción pendiente. Preparación y cuenta se presentan en regiones distintas: `listo` se expresa como «Listo para entregar»; la cuenta usa Cuenta abierta / Por cobrar / Pagada. Los importes se comunican en Pagado y Saldo pendiente. Las tarjetas de Salón reutilizan ese mismo estado y color,
sin combinar «Listo» o «Entregado» con «por cobrar»; el saldo se consulta en el
resumen de cuenta del detalle. Los modales no tienen franja superior decorativa:
la cabecera índigo sólida llega al borde superior y ocupa todo el ancho.
No se agrega una instrucción genérica para liberar la mesa: el estado, el saldo
y las acciones disponibles ya expresan el avance. Quitar ese texto no modifica
los requisitos de cierre de cuenta, entrega y pago. En móvil, las acciones ocupan todo el ancho
y los estados siguen visibles. Esta variante conserva la altura compartida y
el radio de control en escritorio; en móvil usa `radius-pill`.

El pie del detalle de mesa presenta una sola acción sólida azul, al final del
grupo: Cobrar si existe saldo cobrable; en otro caso, la siguiente acción real
del pedido. Editar es secundaria con fondo blanco y borde azul visible,
manteniendo la misma geometría de control. Cancelar muestra el texto breve «Cancelar»
(su nombre accesible es «Cancelar pedido») y conserva su confirmación, que sí usa el
texto completo. Usa fondo `danger-50`, borde `danger-200` y texto e icono `danger`; en
hover el borde pasa a `danger-600` y el fondo a `danger-100`, sin cambiar el color del
texto. Todos los iconos de acción son outline de 18 px: tarjeta para Cobrar,
plato/check para registrar la entrega, gorro de chef para Enviar comanda. La mesa se libera automáticamente cuando
la cuenta está cerrada, la entrega y el pago están completos, sin botón ni confirmación adicional. En
móvil (≤600 px) se conserva el orden de lectura y de teclado, con secundarias que
reparten el ancho de su fila y llenan toda la fila cuando queda una sola;
principal a ancho completo. No reservar una segunda columna vacía. Todas las acciones del pie son píldoras
(`radius-pill`) de 48 px, en una sola línea, como las acciones de la Comanda. Guardar bloquea las acciones y la navegación
a POS, sin cambiar la jerarquía de color.

El pie prioriza el grupo de acciones alineado a la derecha, sin notas explicativas
permanentes. Solo una restricción real, como devolver pagos antes de cancelar,
justifica una nota contextual: se presenta como texto neutro, sin aspecto de
botón, y en móvil va encima de las acciones. Los botones tienen ancho natural y la misma
altura en escritorio; en móvil ocupan el ancho disponible de su rejilla. Toda acción conserva
texto inequívoco, cursor interactivo, respuesta hover, foco y estado bloqueado.
Los datos y notas no reciben el contorno de acción, cursor de mano ni respuesta hover.

El detalle compartido de Pedidos y Salón usa un modal de hasta 760 px, sin altura
mínima artificial. `ModalHeader` contiene únicamente icono, título («Mesa 04»)
y X, alineados en una fila. Responsables primero (Mozo/Cobrado por), luego fecha
y cliente pertenecen al contexto
del cuerpo, antes de Productos. No se cargan estados ni responsables en la cabecera.
`ModalWorkspace` organiza una sola columna: cuerpo desplazable con productos y notas,
seguido de «Resumen de cuenta» inferior persistente a ancho completo.
No se usa una columna lateral de cuenta, ni se declara como estándar: fue rechazada.
El resumen de cuenta usa fondo neutro muy claro `cloud-50` en toda su superficie
y una línea superior neutra de 1 px (`line`),
sin tarjeta anidada ni altura artificial. Las notas usan `cloud-50`,
pero se agrupan en un bloque compacto dentro del cuerpo, no en el resumen.
La franja de acciones del detalle comparte `cloud-50`, cuando existe.
Los importes pagados usan `brand-800` para conservar contraste legible sobre ese fondo.
El estado de cuenta va alineado a la derecha de «Resumen de cuenta» (en todos los
resúmenes `modal-summary`), con icono `receipt` y la
variante `Status emphasis`: Cuenta abierta usa fondo tenue `sky-100`
con texto e icono `sky-800`, sin borde, sombra ni hover. Todos los estados
usan fondos tenues, nunca etiquetas sólidas. El resumen conserva su
fondo neutro `cloud-50`. Una línea neutra separa la identidad de
los importes. Subtotal y Pagado ocupan columnas equilibradas separadas por una
línea vertical discreta, la misma que separa el saldo. A 390 px Subtotal, Pagado
y Saldo pendiente permanecen en una misma fila, con cifras de 18 px. Nunca se
agregan botones a usuarios sin permisos para hacer más visible el pie.

Cantidad, producto e importe permanecen en una rejilla común, también a 390 px.
El nombre admite varias líneas sin mandar el importe a una fila aparte.
`ModalItemList` presenta una lista simple, sin encabezados Cant. / Producto / Importe
ni borde exterior. Las líneas se separan únicamente con `line`. Cantidades en círculos de 32 px, con cifras
tabulares, fondo `primary-100` y texto `primary-700` de la paleta existente;
son lectura, nunca botón ni indicador de estado. No se repite un conteo agregado de
unidades en el encabezado de productos. Responsables se presentan
en el contexto del cuerpo, antes de la fecha. La fecha se alinea al extremo derecho,
y si falta espacio pasa a otra fila con esa misma alineación.
Etiqueta y nombre comparten 14 px e interlineado mediante `modal-inline-metadata`; la
retícula mantiene la primera línea alineada incluso con nombres largos.
Productos y Resumen de cuenta ocupan el mismo ancho. Los datos de contacto y
entrega permanecen desplazables, antes de las líneas y después del encabezado.

Cada producto ocupa dos filas: cantidad, nombre e importe en la primera; en la
segunda, a todo el ancho, «precio c/u · destino» y el estado de preparación
alineado a la derecha, bajo el importe. Todos los productos (simples, menús y
combos) siguen el mismo patrón: cantidad índigo, nombre, nota, «precio c/u ·
destino» y estado. La información secundaria es siempre neutra: fondo `cloud-50`,
borde `line`, iconos de 14 px en `ink-500` y texto 12 px `ink-700`; no hay
colores propios por tipo de producto. El mismo patrón se aplica en los detalles de
Pedidos/Salón, Ventas y POS mediante la línea compartida de `modal.css`
(`modal-item-info`, `ModalItemNote`, `ModalItemComponents`; ver
`04_COMPONENTS.md`). En Ventas y POS la carta incluye también los modificadores
(«TÉRMINO | Medio»).
La nota («Sin miel», «Helada al polo») va bajo el nombre como cápsula ajustada a
su contenido con icono `note`.
En menús y combos, bajo «precio c/u», un bloque `cloud-50` sin borde
(`radius-md`) se lee como una carta: grupo en versalitas 12 px / 600 `ink-500`,
un filete vertical continuo `line` y el plato 14 px `ink-950` (recargo 12 px
`ink-600`), con el estado a la derecha. Sin iconos repetidos por fila. El destino
(Cocina, Barra, Entrega directa) se indica una sola vez tras el precio
(«S/ 15.00 c/u · Cocina») cuando todos los componentes lo comparten; si difieren,
cada fila lo muestra en texto 12 px `ink-600`. Cada plato aparece una sola vez y
las columnas se alinean con subgrid. En móvil (≤600 px) el filete vertical se
reemplaza por separadores horizontales `line`: grupo y destino comparten la primera
fila, el plato ocupa todo el ancho debajo y el estado, si se muestra, baja a una
tercera fila. El orden de los componentes es el que entrega el
API. El área desplazable de productos conserva el foco por teclado, pero sin
contorno azul visible. El estado de preparación por línea se
omite cuando todas las preparaciones repiten el estado del pedido (Nuevo/Sin enviar,
Preparando, Listo, Entregado), porque el estado del pedido ya lo expresa; se muestra
por línea cuando difieren entre sí o del pedido. Confirmado conserva «Pendiente» por
línea: la comanda enviada aún no empieza a prepararse.

Subtotal, Delivery (solo si existe) y Pagado se agrupan en el resumen inferior;
el saldo queda a la derecha en escritorio y usa `primary-700`, 28 px y
cifras tabulares. A 700 px o menos Subtotal, Pagado y saldo comparten una fila, con cifras de 18 px;
las cifras permiten wrap. El estado de la cuenta se muestra únicamente
en el resumen: abierta `sky`, por cobrar `orange`, pagada `green`, finalizada `gray`.
El texto identifica cada estado, no solo el color. Cabecera, cuenta y pie permanecen visibles al recorrer
productos largos. Los valores admiten wrap en cantidades extremas.

«Resumen de cuenta» conserva encabezado accesible y visible en todos los tamaños.
Pagado cero es neutro; pagos positivos y cuenta pagada usan `brand-800`, sin
simular una acción. Skeleton reproduce cabecera breve, contexto, cuerpo, resumen inferior y pie.
Notas generales van fuera de la lista de productos, en una sección propia con
encabezado y fondo suave `cloud-50` (`modal-detail-notes`); conservan
texto completo, sin borde ni aspecto de advertencia. Esta distribución recupera la referencia
seleccionada por el usuario: no volver al formato de tabla. El resto de los modales comparte cabecera,
márgenes de 24/16 px y pie; `ModalWorkspace` es opcional, no una plantilla forzada.

La autoría se presenta como información de lectura, no como botones: «Mozo» y
«Cobrado por», con etiquetas neutras, nombres completos que admiten salto de línea
y un espaciado compartido. Salón y Pedidos usan la misma franja en sus detalles.
Las tarjetas de mesa y las filas de Pedidos conservan estos nombres también en
móvil. No se muestra un cobrador antes de registrar pagos. Las acciones de atención
no aparecen para otro mozo; el acceso de cobro mantiene su permiso independiente.

Pedidos reutiliza esta misma estructura del detalle de Salón: fecha en el contexto
del cuerpo y cantidades circulares por producto, sin conteo agregado de unidades.
«Registrado» muestra siempre la fecha y hora del registro en la zona del local,
nunca un tiempo relativo como «Hace 17 h». La tabla y el detalle reutilizan el
formateador regional compartido. En móvil, la fecha aparece bajo la identidad del
pedido al ocultarse su columna, con salto de línea para no recortar el contenido.
No conserva bloques antiguos de «Registrado», «Consumo» o «Detalle» sin estilos.
Los productos pertenecen al cuerpo desplazable; cabecera, resumen de cuenta y
acciones permanecen visibles. Subtotal y Pagado aparecen siempre en el resumen;
Delivery añade su fila únicamente si aplica. Saldo pendiente o Total pagado se
destaca por separado, según la cuenta. Las acciones tienen ancho natural en
escritorio; en móvil las secundarias se reparten su fila cuando caben y, si
queda una sola, ocupa todo el ancho. La principal conserva su fila completa
dentro de `modal-action-layout`, sin huecos de media columna.
Los textos de interfaz se conservan en UTF-8 y Unicode NFC; no se reparan
tildes mediante reemplazos indiscriminados sobre nombres recibidos del API.

El botón primario usa `primary-600` (`#4654CD`), `--control-height`,
`--font-size-control` y `--radius-control` en escritorio; en móvil usa
`--radius-pill`, igual que los botones secundarios.
Su jerarquía procede del color, no de una altura distinta. En avisos exitosos,
el verde `brand-700` pertenece al símbolo confirmado, no al botón. «Aceptar»
solo cierra el aviso y usa superficie blanca, texto `ink-950` y borde neutro.
No se utiliza `brand-600` como botón ni blanco sobre menta.

## Acceso administrativo

La pantalla conserva una única tarjeta centrada de ancho máximo
`var(--size-360)`, sin panel lateral. El logotipo mide
`var(--size-120)` y permanece arriba, seguido del título visible
«Iniciar sesión» alineado ópticamente con la marca. Los campos de correo y
contraseña conservan `FormField` e `Input`, sus etiquetas persistentes,
iconos, autocompletado y estados de validación.

El formulario utiliza exclusivamente tokens globales de `variables.css`.
El fondo mantiene halos suaves índigo y menta; la cabecera conserva el
degradado blanco/índigo y la franja superior de marca. La acción primaria
«Ingresar» ocupa todo el ancho del formulario, conserva el tono
`primary-600` y usa el icono `lock` del design system (candado),
con trazo discreto y coherente con los controles del formulario. La recuperación de contraseña
permanece como acción secundaria de texto y muestra únicamente el diálogo
«Próximamente»; no debe insinuar que ya envía solicitudes.

La franja informativa de la tarjeta dice «Mesas, pedidos y cocina en un solo
lugar.», con icono de cubiertos. No añadir «Bienvenido», frases genéricas de
seguridad, promesas comerciales o avisos de autorización redundantes.
El copyright está fuera de la tarjeta, con peso secundario y sin altura
vacía obligatoria. En móvil se ajustan los paddings, sin reordenar los
campos ni recortar el botón; las pantallas bajas mantienen el desplazamiento
natural. Respetar contraste, foco de teclado, estados de carga y sesión activa.

## Integridad CSS

- Un selector se declara una sola vez dentro del mismo contexto CSS.
- Un selector tampoco se repite entre hojas importadas por el mismo contexto:
  cada clase tiene un único dueño. La prueba detecta duplicados tanto dentro
  de `globals.css` como entre `globals.css`, `navigation-state.css`,
  `loading.css`, `data-table.css`, `pagination.css` y `table-actions.css`.
- `data-table.css` es el dueño de filas, columnas, cabeceras y desplazamiento
  horizontal de tablas informativas; no duplicar estos estilos en `globals.css`.
- `pagination.css` es el dueño de todos los estilos de paginación.
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

El dashboard usa términos cotidianos del restaurante: «Ventas · Hoy»,
«Pedidos cobrados», «Promedio por pedido», «Pedidos en atención»,
«Saldo por cobrar», «Pendientes por revisar» y «Productos más vendidos». No
muestra «Ventas netas» ni «Ticket promedio», ni referencias a devoluciones. Los
nombres técnicos `salesNet` y `averageTicket` se conservan en el contrato de API.

La pantalla separa el análisis del periodo (columna principal) de la operación en
vivo (columna lateral «Ahora mismo»). Un único selector Hoy / 7 días / 30 días,
sobre todo el contenido, filtra ventas, indicadores, desgloses y productos; la
operación en vivo y las alertas siempre son actuales.

- **Cifra principal.** «Ventas · {periodo}» en 48 px (36 px en móvil) con la
  variación frente a la comparación y el importe anterior. «Hoy» se compara con el
  mismo día de la semana anterior; 7 y 30 días, con la ventana anterior de igual
  duración. La variación lleva signo (+/−) además del color verde o rojo. Al lado,
  la hora pico o el mejor día.
- **Gráfico de ventas.** Una sola serie: las ventas del periodo como onda
  índigo (`primary-600`) con curva suave monótona (sin picos ni valles inventados),
  línea de 3 px con un brillo suave y degradado que se desvanece hacia la base. No
  dibuja el periodo de comparación ni promedios: la comparación vive en la
  variación junto a la cifra principal, para que el gráfico se lea sin
  ambigüedad. El pico lleva su importe como etiqueta directa. Por horas para «Hoy»
  y por días para 7 y 30 días; la onda se dibuja de izquierda a derecha al cargar
  (sin animación con movimiento reducido). El cursor guía muestra importe y
  pedidos con cobro del tramo; las flechas del teclado lo recorren y una tabla
  oculta (dentro de un contenedor `sr-only`, nunca la tabla con esa clase) replica
  los datos para lectores de pantalla.
- **Indicadores.** Pedidos cobrados y Promedio por pedido con minigráfico del
  periodo y su propia variación; Pedidos en atención con los pedidos en cocina.
- **Desgloses.** Ventas por canal, Medios de pago y Ventas por categoría como
  barras horizontales de una sola serie con importe, cantidad de cobros o unidades
  y participación. Las categorías muestran cinco y agrupan el resto en «Otras».
  Canales y medios usan las etiquetas que entrega el API.
- **Ahora mismo.** Saldo por cobrar destacado y bloques de Atención, Cocina,
  Delivery y Caja según los módulos contratados, seguidos de «Pendientes por
  revisar». En escritorio la columna queda fija al desplazarse.

Distingue «Pedidos cobrados» (pago completo con cobro en el periodo) de «Saldo por
cobrar» (pedidos sin pagar o parcialmente pagados, incluso de días anteriores).
El promedio corresponde al importe total de los pedidos completamente cobrados.
Las ventas reflejan los movimientos de cobro del periodo, también pagos parciales.
La fecha procede del backend en la zona del local. La consulta se actualiza cada
30 segundos y su clave incluye empresa, local y periodo; al cambiar de periodo se
conserva el render anterior atenuado hasta recibir los datos, y las etiquetas
siguen al periodo de los datos mostrados. Las secciones respetan los módulos
contratados y los enlaces usan las mismas reglas de menú y permisos del shell.
El efectivo de cajas ciegas no autorizado muestra «Importe reservado», nunca cero.
Los pendientes de disponibilidad/abastecimiento no repiten las colas de cocina.
La carga reproduce la misma estructura con skeleton; el error es recuperable y no
reemplaza datos por ceros. Bajo 1180 px la columna «Ahora mismo» pasa debajo del
análisis; en 390 px todo se apila, el selector ocupa el ancho con botones de 44 px
y el gráfico conserva el eje X sin desbordar.

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
Los datos de identidad usan una rejilla compacta: mesa (o cliente) y canal en la
primera fila; atendido por y fecha del pedido en la segunda cuando es Salón.
Las columnas tienen el mismo ancho y sus etiquetas y valores se alinean;
no se fuerza la fecha a otra fila ni se deja un espacio vacío junto al mozo.
En móvil (≤620 px), `modal-record-data` convierte las parejas en una lista
compacta de cuatro filas: Mesa, Canal, Atendido por y Fecha del pedido.
Cada etiqueta va junto al valor; etiquetas de 120 px, separación horizontal
de 12 px y vertical de 10 px. Todos los valores comienzan en la misma columna,
sin anclar Canal al extremo derecho ni apilar rótulos sobre sus valores.
Los nombres largos envuelven sin recortarse, igual que Detalle de cobro.
El cuerpo comparte los márgenes de la cabecera y el pie, sin añadir paneles
laterales ni trasladar información adicional al header.
Presenta mesa o cliente con su etiqueta específica, canal, fecha,
productos con cantidad circular, precio unitario y total por línea, composición
y notas cuando existen, subtotal, delivery si aplica, total y pagos netos de
devoluciones. Los medios de pago usan nombres recibidos de la API. Si la venta
cambió después de abrir el historial, el detalle muestra su estado y saldo reales.
La carga reproduce esta estructura con skeleton; error ofrece Reintentar y los
productos/pagos vacíos tienen un mensaje explícito. El cuerpo tiene scroll y
la cabecera permanece visible. Desde 390 px conserva todos los datos y acciones.
Reutiliza `ModalWorkspace` y `ModalItemList`: productos y pagos son listas con
separadores, sin tarjetas con borde exterior. Los pagos tienen una separación
visible respecto a los productos; las notas usan `modal-detail-notes`.
El resumen inferior fijo `modal-summary` muestra «Resumen de venta», estado
real con énfasis, subtotal y pagado neto, y Total como importe principal.
Delivery y saldo pendiente se añaden solo cuando aplican. El total no se
duplica dentro del cuerpo desplazable. El pie de acciones solo existe si se
puede «Ver resumen de pago», con icono `eye` y variante `secondary`; no repite
estado ni añade otro cierre. Consulta el documento operativo existente sin
registrar pagos ni simular una boleta o factura.

Los módulos de directorio, como Clientes, usan la tabla estandarizada en escritorio y tarjetas equivalentes en móvil. Toda consulta remota debe incluir skeleton, error recuperable, vacío contextual y paginación compartida. Las acciones de fila son únicamente iconos homologados con tooltip; alta y edición usan las primitivas `Input`, `Select`, `Textarea`, `Button`, `Status` y `ConfirmDialog`. Los formularios extensos se agrupan por secciones visuales, sin añadir texto explicativo que no ayude a completar la tarea.


### Disponibilidad de módulos

- El catálogo distingue **Disponible**, **En desarrollo** y **Planificado**.
- `active` representa únicamente si un módulo **Disponible** está habilitado para una empresa; no representa su estado de desarrollo.
- El Administrador de plataforma ve todos los módulos en navegación y catálogo, incluso si están inactivos o no terminados.
- Los usuarios de empresa solo ven módulos disponibles, incluidos en su plan, activos para su organización y permitidos por su rol. Emprende incluye «Menús y combos», «Inventario» y «Compras», sin nuevas pantallas ni controles específicos del plan.
- Usuarios y roles muestra únicamente accesos y acciones de módulos efectivos del plan. Los contadores y la vista previa cuentan esos mismos elementos; se omiten grupos vacíos. La empresa no recibe el catálogo comercial completo de Plataforma.
- Un módulo **En desarrollo** o **Planificado** nunca puede activarse para una empresa; el backend debe rechazar cualquier intento aunque el frontend falle.

## Onboarding comercial y suscripción

### Empresas de la plataforma

`/settings/companies` («Empresas y planes») es una página independiente del
admin, con el menú lateral, ubicada en Configuración junto a «Módulos». Solo la
ve el administrador de plataforma (`platformAdminOnly`); cualquier otro usuario
recibe «Acceso restringido». No forma parte del panel de Plataforma.
La página lista todas las empresas cliente (nombre comercial y RUC, sin datos
accesorios como razón social o conteos de locales y usuarios) con plan, ciclo, precio
contratado, próximo vencimiento, último pago confirmado y situación de pago.
Cinco indicadores (Empresas, Al día, Por vencer, Vencidas, En prueba) resumen
la cartera sin filtros y funcionan como filtro rápido. La barra permite buscar
por nombre comercial, razón social o RUC y filtrar por plan y situación; ambos
catálogos y sus etiquetas provienen del API. La situación usa insignias con
punto: verde al día, ámbar por vencer, rojo vencida, violeta en prueba y gris
cancelada o sin suscripción. El vencimiento muestra la fecha y los días
restantes o transcurridos. La tabla conserva skeleton, error con reintento,
vacío y sin coincidencias, scroll horizontal controlado y paginación; los
indicadores pasan a tres y dos columnas en anchos menores.
«Ver detalle» (icono de ojo) abre una ficha de solo lectura, con X como único
cierre: resumen (situación, plan y ciclo, próximo vencimiento o fin de prueba,
total pagado), datos generales y fiscales (razón social, RUC, país, moneda,
impuesto, zona horaria, cliente desde, estado), suscripción y uso frente a los
límites del plan, administradores con correo y locales con dirección y teléfono.
Tiene skeleton, error con reintento y estados vacíos por sección; su pie ofrece
«Gestionar suscripción». En móvil el resumen pasa a dos columnas y los datos a
una.
El nombre de la empresa y la acción «Gestionar suscripción» abren su formulario
propio en `/settings/companies/{id}`, sin cambiar la empresa activa de la
sesión. El encabezado muestra nombre comercial y RUC con retorno a
«Empresas y planes»; el cuerpo reutiliza el mismo formulario de
`/platform/subscription` (plan y ciclo, registrar pago e historial) apuntando a
esa empresa. Carga con el skeleton de suscripción, error con reintento y estado
«Sin suscripción». Guardar o registrar un pago actualiza también el listado.

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
Delivery desglosa Subtotal y Envío antes del Total. En pedidos manuales, solo
«Enviar a cocina» persiste el pedido; «Guardar comanda» es una acción de Salón.
El cierre exige descarte cuando hay cambios y se
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

- Salón/Pedidos distinguen preparación y cuenta sin códigos técnicos ni instrucciones redundantes. El destino y estado de cada producto se presentan en una fila compacta y alineada; el estado se omite cuando repite el del pedido (ver detalle compartido).
- El pie muestra un único botón primario «Confirmar entrega» para todo el pedido cuando todos los productos pendientes están Listos. No se muestran botones de entrega individuales: destino y estado de cada producto son datos de lectura. Cocina/Barra preparan; el mozo confirma todo con una acción. Las rondas adicionales incluyen solo productos aún sin entregar y no repiten las entregas anteriores. No se muestran modales exitosos en estas transiciones operativas.
- «Agregar productos» reutiliza la comanda con solo líneas nuevas y aparece únicamente con cuenta Abierta. «Cerrar cuenta» es primaria y aparece solo después de entregar todos los productos de todas las rondas. Después se muestra el enlace a cobro solo con entrega completa, permiso y saldo; no se ofrecen más consumos ni reapertura.
- Cocina y Barra reutilizan el mismo KDS con selector de estación alimentado por backend, permisos independientes y las mismas primitivas visuales.
- En móvil (390 px) las filas permiten wrap sin desbordar; la entrega global y las demás acciones del pie conservan 44 px y ancho disponible. No se duplican clases de los estilos base.
