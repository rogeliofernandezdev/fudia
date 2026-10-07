# Layouts

Sidebar: Control, Operación, Carta y producción, Abastecimiento y Configuración. Topbar compacta.
Gestión: título, acciones, filtros, listado, paginación y detalle.

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

El buscador superior busca opciones de navegación autorizadas, no productos ni
ventas. Su placeholder describe ese alcance y sus enlaces usan el mismo catálogo
filtrado del sidebar. No se muestra una campana con datos ficticios: el centro
de notificaciones quedará visible cuando exista una fuente remota real.
