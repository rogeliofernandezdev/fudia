# Layouts

Sidebar: Control, Operación, Carta y producción, Abastecimiento y Configuración. Topbar compacta.
Gestión: título, acciones, filtros, listado, paginación y detalle.

## Plantillas de página

Toda ruta usa una de estas plantillas. El contenedor `.content` define el
padding (28/32 px en escritorio, 24/16 px desde 820 px) y el ancho máximo; las
páginas no agregan márgenes laterales propios.

### Gestión (listado)

1. `PageHeader`: título igual al nombre de la navegación, descripción de una
   línea y la acción «Nuevo …». Sin eyebrow.
2. `Tabs` opcionales para vistas (con contador); la acción «Nuevo …» se alinea a
   la derecha de las pestañas cuando cambia según la pestaña.
3. Un único `panel` con barra de filtros: `SearchField` primero, `Select` con
   opciones del API y «Limpiar filtros» (`secondary`) solo con filtros activos.
4. Tabla estándar en escritorio y `management-cards` en ≤700 px; **las dos
   siempre**. Si una pantalla no tiene tarjetas, la tabla conserva scroll
   horizontal controlado; nunca desaparece el contenido.
5. Estados: `TableSkeleton` con las columnas reales, `ErrorState` con
   «Reintentar», `EmptyState` con dos textos: vacío («Aún no hay…» + acción) y
   sin coincidencias («Sin coincidencias» + «Limpiar filtros»).
6. `Pagination` del servidor, oculta durante la carga y el error.
7. Acciones por fila: `table-actions` con `RowActionButton` únicamente.

Aplica a Ventas, Punto de venta, Pedidos, Carta y productos, Mesas, Menús y
combos, Recetas, Inventario, Kárdex, Compras, Clientes, Gastos, Reservas,
Locales, Usuarios y roles, Medios de pago y Empresas.

### Configuración y detalle

`PageHeader` con `back` hacia la ruta padre y `meta` para el estado. El cuerpo se
organiza en paneles por sección con encabezado (`h2`) y un pie con «Cancelar» y
«Guardar». El skeleton reproduce la forma del formulario y el error vive dentro
del panel. El inicio de Configuración se genera desde el grupo CONFIGURACIÓN de
`shell/navigation.ts`, con los mismos permisos.

### Tablero operativo

Salón, Cocina y Caja › Cajas. `PageHeader` sin filtros; una barra de contexto
compacta (turno, alertas) y una barra de tablero con búsqueda, zonas o carriles y
actualización. Grilla o carriles con skeleton de la misma forma, vacío por carril
y error recuperable. Sin paginación. Los filtros nunca se colocan dentro del
encabezado.

### Dashboard

Encabezado estable incluso en error, `Tabs` para el periodo (Hoy, 7 días,
30 días), KPI separados de «Ahora mismo», skeleton con la misma estructura y
error recuperable sin reemplazar datos por ceros.

### Asistente

Solo la navegación de pasos visible: sin «Paso X de Y», sin «N pasos» y sin
repetir el estado de cada paso con texto si ya lo comunica el estilo. Pie con
«Anterior» y «Continuar»; la última acción usa el verbo real («Crear empresa»,
«Guardar menú»).

### Próximamente

Un único componente con `EmptyState`, título de la opción y retorno mediante
`pageRoutes`, nunca un enlace literal al Dashboard.

### Tarea a pantalla completa

La comanda ocupa la pantalla con cabecera índigo `primary-700`. El título es el
dato que el usuario confirma («Mesa 04», «Delivery · Rosa
Quispe»), no el nombre genérico de la pantalla. Un solo control de cierre que
confirma el descarte si hay líneas. Detalle completo en `02_DESIGN_SYSTEM.md`
› Comanda.

El pie informativo del acceso presenta Conexión segura, Acceso por empresa y
Usuario validado en una sola fila, con cada icono junto a su texto sin saltos.
Usa tipografía auxiliar y espaciado compacto desde 390 px, sin recortar mensajes.
El login conserva una sola tarjeta centrada y el orden original en todos los
anchos: logo, formulario y mensajes informativos. La cabecera de marca tiene una
superficie casi blanca; los campos permanecen blancos con iconos contenidos,
acción única «Ingresar» y copyright externo. En viewport bajo conserva scroll
de página y acceso a la acción. No añade panel lateral, títulos decorativos,
selección de empresa/local/rol ni indicadores operativos ficticios.
«Recuperar contraseña» se muestra bajo el campo como acción secundaria de texto
y abre únicamente el aviso «Próximamente», sin recuperación funcional ni solicitudes.

El aviso de configuración inicial comparte los bordes laterales del título y
del listado: el padding lo define únicamente el contenedor global `.content`.
El aviso no agrega margen lateral ni superior y conserva 21 px de separación
inferior. En móvil el texto puede envolver sin comprimir los iconos.

El selector de contexto se ancla al control de empresa/local en la barra
superior. Tiene cabecera compacta con icono, título y X; usa `FormField`,
`Select`, `IconButton` y `Button` del design system, sin controles locales.
Empresa precede a Local; cambiar de empresa limpia el local seleccionado.
«Aplicar» usa azul primario, se habilita únicamente para un destino válido y
distinto del actual y muestra «Cambiando…» durante la petición. Los campos y
el cierre quedan bloqueados mientras se aplica. No se cambia el contexto al
seleccionar una opción: la aplicación requiere una acción explícita.
La carga conserva la geometría de campos y acción mediante skeleton; un error
ofrece Reintentar y los catálogos vacíos se indican explícitamente. En móvil
el panel ocupa el ancho útil bajo la barra, con altura acotada, scroll cuando
excede el espacio y controles táctiles de 44 px. Las animaciones respetan
movimiento reducido. El cambio conserva la limpieza de caché y la navegación
al primer acceso permitido por el nuevo contexto.

La cuenta del usuario vive en la barra superior. Las iniciales, nombre y rol
abren un submenú con perfil, preferencias, seguridad y cierre de sesión. La
salida no se ejecuta al tocar las iniciales ni se duplica en el sidebar.
El rol real del local activo aparece debajo del nombre, también en el submenú.
La etiqueta es breve: «Administrador», «Cocinero», «Cajero», etc. Los sufijos
«de empresa» y «de plataforma» del administrador se omiten solo en esa etiqueta;
no se cambian los roles almacenados ni sus permisos. Roles personalizados
conservan su nombre y las asignaciones múltiples no repiten etiquetas iguales.
El selector conserva el local activo para todos los usuarios. El nombre de
empresa permanece visible por separado en la barra superior, también en móvil;
no reemplaza al rol ni al local. Plataforma conserva el cambio de empresa dentro
del popover, pero el botón de contexto muestra únicamente el local activo.
El nombre se alinea inmediatamente junto al icono del menú en un bloque izquierdo
de identidad, con separación fija y mayor peso visual. El buscador y los controles
de local/usuario permanecen fuera de ese bloque. En móvil el nombre ocupa el ancho
restante y se trunca sin ocultarse ni desbordar.
Mi perfil y la cabecera comparten una consulta
con caché por usuario, empresa y local, skeleton y error con reintento.

El buscador superior busca opciones de navegación autorizadas, no productos ni
ventas. Su placeholder describe ese alcance y sus enlaces usan el mismo catálogo
filtrado del sidebar. No se muestra una campana con datos ficticios: el centro
de notificaciones quedará visible cuando exista una fuente remota real.

## Plantilla compartida de modal

Autoridad visual: `02_DESIGN_SYSTEM.md`; contrato: `04_COMPONENTS.md`.
Cabecera breve (icono, título, X), cuerpo desplazable, resumen inferior opcional
y pie fijo con acciones autorizadas. Fecha y responsables van en el contexto del
cuerpo; los estados acompañan los datos que describen. No se usa columna lateral
de cuenta. Esto no impide rejillas de campos dentro de un formulario.

Usar `ModalHeader` y `ModalFooter` en modales de contenido. Cabecera solo con
icono, título y X, sin información debajo del título. Botones contiguos a la
derecha, Cancelar antes de Guardar, separación de 10 px. En móvil comparten
fila cuando caben. Estados reales en resumen o pie, sin estados inventados.
`ConfirmDialog` y respuestas de `FeedbackProvider` quedan exceptuados.
Etiquetas en seminegrita, valores regulares y títulos de sección en negrita;
la jerarquía pertenece al sistema compartido, no a cada pantalla. Los detalles
usan «Mesa», «Cliente» o «Venta» según el dato, sin rótulos combinados.
