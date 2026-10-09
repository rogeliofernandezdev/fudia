# Layouts

Sidebar: Control, Operación, Carta y producción, Abastecimiento y Configuración. Topbar compacta.
Gestión: título, acciones, filtros, listado, paginación y detalle.

El pie informativo del acceso presenta Conexión segura, Acceso por empresa y
Usuario validado en una sola fila, con cada icono junto a su texto sin saltos.
Usa tipografía auxiliar y espaciado compacto desde 390 px, sin recortar mensajes.
El login conserva una sola tarjeta centrada y el orden original en todos los
anchos: logo, formulario y mensajes informativos. La cabecera de marca tiene una
superficie índigo clara; los campos permanecen blancos con iconos contenidos,
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
