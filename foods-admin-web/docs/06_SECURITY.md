# Seguridad

En Caja, el API limita el historial y los movimientos a los turnos que abrió el
usuario. Solo el administrador reservado consulta todos los del local. La
asignación temporal para cobrar no permite leer las finanzas de otro cajero.
Detalle, equipo e informe se validan también al acceder directamente por UUID.
Las cajas ocupadas por otra persona llegan con `detailsVisible=false`: muestran
ocupación y responsable, sin importes, movimientos, equipo ni acciones del turno.
El resumen agrega solo turnos visibles y las consultas incluyen empresa, local y
usuario en sus claves de caché. La autorización definitiva permanece en backend.

`cashCollection` es una capacidad temporal de sesión que permite abrir POS con
una asignación activa a un turno abierto. No añade `cash.manage`, permisos ni
accesos permanentes al rol. POS consulta `pos/access`, cuya proyección no incluye
saldos administrativos; cobro y devolución mantienen condiciones distintas.
El acceso «Cobrar saldo» en los detalles de Mesa y Pedidos también reconoce
`cashCollection`, únicamente para las atenciones de Salón cuyo `waiterId` coincide
con el mozo autenticado. Ese vínculo se toma del flujo existente al enviar la
comanda; no requiere asignar mesas manualmente. POS reutiliza `canCollectOrder`
para sus filas, el formulario y el envío. El cajero con `cash.manage` conserva
su autorización general. Enlaza a `/pos?orderId=…`, que abre el formulario compartido
de cobro para esa cuenta. Sigue exigiendo entrega, cierre de cuenta y saldo;
no concede edición de otras mesas ni registra pagos al abrir. POS vuelve a
consultar el turno y el backend valida la asignación al confirmar cada cobro.
El equipo de turno excluye al actor, asignados y turnos cerrados en cliente;
la lista de colaboradores y su contador excluyen también a `openedById`, aunque
se consulte como administrador. El responsable conserva su vínculo interno para
operar y el backend impide retirarlo como si fuese un colaborador adicional.
La API filtra administradores, inactivos y usuarios ajenos al local y vuelve a
validar todo al asignar. El backend es la autoridad sobre revocación y cierre,
incluso cuando la caché visual aún no se ha refrescado.
El cierre del turno requiere `cash.manage` y coincidir con `openedById`.
La interfaz oculta el cierre a otros usuarios y la API lo rechaza aunque sean
administradores. Los informes históricos conservan sus autores originales.

La UI representa permisos, pero backend autoriza. No guardar tokens en localStorage.
Proteger mutaciones contra CSRF y pedir motivo para acciones sensibles.
La creación, edición y activación de cajas usa `cash.registers.manage`.
`cash.manage` solo habilita operar turnos; no muestra «Registrar caja», edición
ni cambios de estado. Los diálogos y confirmaciones vuelven a comprobar el permiso
al enviar. La API protege las tres mutaciones con el mismo permiso independiente.

Cada rol tiene dos controles independientes. `menuAccess` decide qué opciones
aparecen y qué rutas puede abrir; `permissions` decide qué acciones puede
ejecutar dentro de ellas. El sidebar y la protección de rutas consumen
`menuAccess`; los botones consumen permisos específicos y el backend vuelve a
validar cada petición. Conceder un permiso no concede automáticamente una opción
del menú, ni conceder una opción autoriza sus acciones.

La entrada `/` pertenece al layout autenticado y espera el contexto antes de
redirigir a `firstAccessibleRoute`, igual que login y cambio de local. Si se abre
`/dashboard` sin acceso efectivo (menú, permiso de lectura y módulo activo), el
shell reemplaza la ruta por ese mismo destino permitido sin mostrar el contenido
del Dashboard ni el aviso de acceso denegado durante la transición. No se infiere
el destino por el nombre del rol. Sin opciones válidas se entra a `/no-access`;
los enlaces genéricos de regreso al inicio usan `/`. Puesta en marcha solo tiene
prioridad si también es accesible. Las demás rutas restringidas mantienen su
aviso y la autorización definitiva del API no cambia.

Carta y productos usa las etiquetas «Ver carta y productos» y «Administrar carta
y productos» del catálogo remoto. Sin `menu.manage`, Productos y Categorías
conservan lectura, filtros y paginación, pero no muestran altas, edición,
activación/desactivación ni columna de acciones. Los vacíos tampoco ofrecen
crear. Si el permiso se retira con un formulario abierto, deja de renderizarse.
No se conceden permisos automáticamente al habilitar el acceso del menú.

Esta regla aplica a todos los roles, incluidos los personalizados: se consulta
el permiso efectivo de la sesión, nunca el nombre del rol. Menús y combos
requiere `menu.manage` para altas, edición y estado; conserva detalle y
paginación para lectura. Mesas requiere `tables.manage` y Zonas `menu.manage`,
igual que el API. Los formularios y confirmaciones de esos módulos no se
renderizan si el permiso efectivo se retira.

Los catálogos y la vista de roles de empresa se limitan a módulos contratados,
activos y disponibles. El backend valida ese límite al guardar. Los permisos
almacenados de módulos fuera del plan se conservan sin mostrarse para que un
cambio de plan no los borre. Roles y permisos se consultan con claves de caché
que incluyen empresa y módulos efectivos. Plataforma conserva el catálogo total.

## Sesión en el navegador

Cuando una sesión expira, la interfaz muestra una transición breve y vuelve al
inicio de sesión. Cualquier respuesta `401` de contexto, consultas, mutaciones,
carga de imágenes o administración de plataforma elimina la cookie, descarta el
estado del usuario y comunica el cierre a las demás pestañas. Un `401` es
definitivo: el contexto no se reintenta y la expiración solo borra la cookie en
el BFF (`DELETE /api/session?scope=local`), sin llamar al logout del backend.

«Cerrar sesión» desde el menú es una sola navegación sin recarga: marca el
cierre como manual (`beginSignOut`), así el `401` que llega después no abre la
salida por «Tu sesión finalizó» ni una segunda llamada. Luego llama a
`DELETE /api/session`, avisa a las demás pestañas y navega a `/login`; la caché
de consultas se vacía al desmontar el panel, nunca antes (vaciarla con el panel
montado vuelve a pedir el contexto y provoca ese `401` en cadena). El BFF borra
la cookie siempre y limita la revocación en el backend a 3 s; el cliente no
espera más de 5 s aunque la red falle.
El login lee la cookie en el servidor: sin cookie muestra el formulario de
inmediato; con cookie valida una sola vez («Validando sesión») y, si sigue
vigente, entra al primer acceso permitido. Si la validación
falla temporalmente por red o por una
indisponibilidad del backend, conserva el sistema visual y ofrece acciones para
reintentar o volver al inicio de sesión; nunca presenta contenido sin estilos.

Una cookie `HttpOnly` representa una sola identidad por perfil y origen del
navegador. Un segundo inicio de sesión nunca reemplaza silenciosamente una
sesión válida: la interfaz exige continuar con la identidad actual o cerrar la
sesión antes de cambiar de cuenta. Login y logout se comunican entre pestañas
mediante `BroadcastChannel`, sin transmitir tokens ni información sensible, y
el contexto se vuelve a consultar al recuperar el foco. Si una pestaña detecta
que la identidad cambió, descarta el estado anterior y navega al primer acceso
válido de la identidad nueva; nunca conserva una ruta que pertenecía al usuario
anterior. Las pruebas simultáneas de roles distintos requieren perfiles o
contextos de navegador aislados. Varias ventanas incógnitas del mismo perfil no
son contextos aislados: comparten una sola cookie mientras permanezcan abiertas.
