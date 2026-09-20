# Sistema de diseño operativo

## Propósito

Este documento es la autoridad visual de foods-operations-web y no depende de
documentos externos. La interfaz debe ser táctil, rápida y legible en celular,
tablet, caja y KDS. No se copian marcas ni composiciones de las referencias.

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
| digital-500 | #7C3AED | acentos digitales |
| ink-950 | #101828 | texto principal |
| ink-600 | #475467 | texto secundario |
| cloud-50 | #F7F8FC | fondo de aplicación |
| surface | #FFFFFF | tarjetas y paneles |
| warning | #B54708 | espera y atención |
| danger | #C9362B | errores y anulaciones |

Proporción recomendada: 80% superficies neutras, 15% índigo primario y 5% colores
semánticos. El violeta identifica automatización; no es el color general del producto.

### Jerarquía cromática

- Índigo `#4654CD`: confirmar acciones principales, cobrar y avanzar en flujos.
- Verde: resultados exitosos, disponibilidad y estados positivos.
- Azul: navegación, selección, foco y acciones operativas reversibles.
- Violeta: automatización, WhatsApp e integración; no usar como acción general.
- Amarillo: espera o advertencia; reservar rojo para error, atraso y anulación.
- Una superficie no combina más de un color de acento salvo que compare estados.
- Fondos y contenedores son neutros para que el color conserve significado.

## Tipografía y forma

- Manrope para interfaz; Geist Mono para correlativos y montos técnicos.
- Titulares de peso 700 u 800 y cuerpo de 14 a 16 px.
- Radio de 18 px en tarjetas y paneles; 10 a 12 px en controles.
- Bordes #EAECF4 y sombras suaves multi-capa (0 1px 2px + 0 6px 20px).
- Montos, tiempos y cantidades con cifras tabulares.
- Sin gradientes intensos ni decoración que reduzca velocidad de lectura.
- Anti-aliasing habilitado (-webkit-font-smoothing: antialiased).

## Dirección minimalista

- Priorizar superficies neutras, bordes finos y alineaciones constantes.
- El color comunica estado o acción; no se usa para decorar contenedores.
- Evitar carruseles o scroll horizontal cuando una cuadrícula adaptable resuelva
  el mismo contenido. Las colas operativas extensas, como comandas, conservan
  scroll vertical independiente y encabezado de columna visible.
- Una sección presenta un encabezado, una jerarquía principal y una acción clara.
- Sombras solo cuando expliquen elevación; las tarjetas internas permanecen planas.

## Componentes

- Área táctil mínima de 44 por 44 px; todos los inputs y botones usan 44 px.
- Acciones estándar: altura 44 px, texto 13 px/20 px con peso 700, radio 10 px,
  padding horizontal 16 px, icono 20 px y separación de 8 px. El ancho responde
  al texto; las acciones de un mismo grupo se alinean o reparten columnas iguales.
- Botones de solo icono: 44 × 44 px. No reducirlos para que entren en móvil.
- Las tarjetas seleccionables de pago/comprobante no son botones de acción:
  mínimo 80 px para título y descripción. Mantener el borde del mismo grosor
  al seleccionar para evitar saltos de posición.
- Ritmo de contenido: 24 px entre secciones, 16 px entre tarjetas y 8 px entre
  elementos relacionados. Texto secundario operativo de al menos 12 px.
- No desplazar tarjetas ni ampliar fotografías al pasar el cursor. Transiciones
  cortas de color y foco; respetar la preferencia de movimiento reducido.
- Botón primario `#4654CD`; verde queda reservado para éxito y confirmación.
- Estados con texto, icono y color; nunca solo color.
- Skeletons con la geometría final y errores con recuperación concreta.
- Foco visible con ops-500 y contraste WCAG AA.
- Hover sin desplazamiento: variar únicamente borde o fondo.

## Densidad por modo

### POS

Productos y modificadores usan tarjetas táctiles. Pedido, total y cobro permanecen
visibles sin navegación profunda.

El wizard Pedido → Personalizar → Pago → Comprobante usa una sola geometría:
encabezado de contexto, progreso visible, contenido principal y resumen lateral.
Cada paso conserva mesa y correlativo, señala etapa actual/completada y presenta
una única acción primaria.

### Salón y móvil

Una columna principal, navegación inferior y acción primaria contextual. No se
permiten desbordamientos horizontales a 390 px.

### KDS

Lectura a distancia y prioridad para tiempo, mesa o pedido, productos,
observaciones y cambio de estado.

La disponibilidad del menú usa tarjetas operativas compactas con estado textual,
cupo y tres acciones táctiles iguales. En móvil las acciones se apilan sin perder
su etiqueta; el agotado manual debe poder ejecutarse en un solo toque.

### Caja

Importes, medios de pago, vuelto y emisión tienen máxima jerarquía. Las acciones
irreversibles requieren confirmación.

## Responsive

- Menú lateral: marca y salida estables; secciones en una zona flexible con
  `min-height: 0`, scroll vertical independiente y barra visible cuando desborda.
- Menú móvil limitado al viewport dinámico, con scroll, espacio para el área
  segura y cierre con Escape o al navegar.
- Mantener columnas reservadas para icono, nombre, contador y flecha: la ausencia
  de un contador no desplaza los demás elementos.
- Separación icono/texto de 8 px en acciones; iconos sin compresión, controles
  de al menos 44 px y grupos que se redistribuyen sin ocultar acciones.

- Móvil desde 390 px.
- Tablet con catálogo y pedido divididos.
- Escritorio con navegación lateral y mayor densidad.
- KDS optimizado para pantalla horizontal.

Cada breakpoint ajusta jerarquía y navegación; no basta con apilar el escritorio.

## Navegación lateral

Sidebar azul oscuro `#1A2151`, texto e iconos claros y sin gradiente. La
selección utiliza el primario `#4654CD`, texto blanco y una sombra discreta.
Verde queda reservado para conexión y confirmaciones; violeta para contadores o
integraciones. Todas las entradas mantienen nombre visible, la misma columna de
iconos y scroll propio.

El menú usa transiciones de 160 a 240 ms solo para color, apertura y ancho. No
desplaza elementos en hover. El drawer móvil anima entrada vertical breve, cierra
con Escape o al navegar y desactiva movimiento cuando el sistema lo solicita.

## Navegación inferior (móvil)

Cinco destinos en columnas iguales: Inicio, Mesas, Venta, Comandas y Pedidos.
Ningún destino flota ni cambia de altura. Todos muestran icono y nombre; los
demás módulos están disponibles en el menú hamburguesa.

## Biblioteca de componentes

`src/components/ui/controls.tsx` define Button, ActionLink, Input, Textarea,
Select, Label, Field y Table. Sus estilos encapsulados se encuentran en
`controls.module.css`; no recrear estos controles con estilos por pantalla.

- Button/ActionLink: tonos neutral, primary (índigo), success (verde) y danger.
- La acción principal azul `#4654CD` mide 48 px de alto, con 20 px de padding horizontal
  y texto 13/20 px; los controles secundarios permanecen en 44 px.
- Button: layout action (44 px), icon (44 × 44 px) o card (según contenido).
- Select: elemento nativo con teclado y foco, sin simular combobox mediante div.
- Input/Select: configuración visual adoptada de Bodegas: borde gris 200,
  radio de 6 px, texto compacto de peso medio, placeholder gris 400, hover gris
  300 y anillo de foco suave de marca. Foods conserva 44 px de altura por su
  contexto táctil operativo.
- Table: encabezados semánticos, caption y scroll horizontal dentro del panel.
- Toda tabla que incluya acciones usa `TableAction`: icono SVG consistente,
  área accesible, `aria-label` y tooltip descriptivo; no se crean variantes locales.
- Field: label asociado por id; ayudas y errores deben vincularse mediante
  aria-describedby en el control y aria-invalid cuando corresponda.
- Las clases específicas de página definen composición, no nuevos tamaños de
  botones ni variantes de campos. Mantener superficies blancas y bordes ligeros.

## Patrón de interfaz compartido con Bodegas

Foods adopta de Bodegas la disciplina de composición, no su identidad: shell
colapsable en escritorio, drawer en móvil, barra superior blanca, fondo cloud,
tarjetas sobrias, formularios con label/ayuda/error, filtros agrupados y tablas
con encabezado azul profundo, filas alternadas y hover. Todas las pantallas,
incluido login, deben reutilizar esta geometría y los componentes centrales.

La paleta continúa siendo exclusiva de Foods: verde para confirmación, azul para
operación y violeta para integración. Los controles conservan 44 px por tratarse
de una aplicación táctil, aunque el referente administrativo use menor altura.

## Integridad CSS

- Un selector se declara una sola vez dentro del mismo contexto CSS.
- No se permiten bloques posteriores que reescriban una clase para corregir
  estilos anteriores.
- Las diferencias de estado, tamaño o dispositivo usan clases modificadoras,
  atributos o `@media` explícitos.
- La prueba `tests/css-duplicates.test.mjs` protege esta regla.

## Carga remota

- Toda lectura remota muestra un skeleton con la geometría aproximada del
  contenido final; tablas, tarjetas y formularios no usan un spinner aislado.
- Toda mutación muestra estado ocupado en la acción que la originó, evita envíos
  duplicados y conserva visible el contexto de la pantalla.
- Después de cargar se presenta contenido, vacío accionable o error recuperable.
