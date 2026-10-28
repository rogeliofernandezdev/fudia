# Seguridad

## Colaboradores de Caja

Crear, editar, activar o desactivar cajas requiere `cash.registers.manage`
(«Administrar cajas»). `cash.manage` («Operar turnos y cobros») conserva apertura,
operación, equipo y cierre personal del turno, sin configurar el catálogo de
cajas. El rol reservado Administrador de empresa recibe el nuevo permiso por
defecto y por la migración 94 para roles existentes. Cajero, Supervisor y Gerente
no lo reciben por tener `cash.manage`, por su nombre ni por otros permisos.
Un administrador puede delegar explícitamente el permiso mediante roles.

La asignación activa en `cash_shift_users` concede una capacidad temporal de
cobro, nunca `cash.manage` ni un rol. POS permite cobros individuales y divididos
con permiso administrativo o asignación vigente al turno abierto. Solo habilita
su lectura operativa, medios de venta y pagos del turno; no habilita devoluciones,
movimientos manuales, transferencias, cajas, cierres ni gestión del equipo.
Los permisos de lectura permanentes anteriores continúan independientes.
`cashCollection` del contexto permite mostrar POS temporalmente sin alterar
`permissions` ni `menuAccess`; se refresca en cliente cada 30 s y al recuperar foco.
La API valida cada solicitud y el cobro vuelve a comprobar el turno bajo bloqueo.
El endpoint `pos/access` expone solo identidad y responsable, sin saldos financieros.

Asignar exige `cash.manage`. Selector y escritura excluyen administradores
reservados/de plataforma, al actor, inactivos y usuarios sin rol activo en el
local. Los índices existentes impiden duplicados y otro turno activo en el local.
El responsable que abre un turno se incorpora por el flujo de apertura, separado
de la asignación de colaboradores; conserva sus facultades administrativas.
Cerrar exige además que el usuario autenticado coincida con `opened_by`.
No existe excepción para otro cajero o administrador. Se verifica bajo bloqueo
antes de guardar conteos y se repite la condición en el UPDATE de cierre.

Cobrar bloquea primero el turno y después la asignación en modo compartido;
retirar y cerrar bloquean el turno de forma exclusiva. Así pueden cobrar varios
usuarios mientras el bloqueo del pedido protege el saldo, y ningún cobro nuevo
se confirma con una asignación ya retirada o un turno cerrado. Un cobro que ya
tenía los bloqueos termina antes del retiro/cierre. El historial permanece intacto.
No cambia autenticación, cookies, tablas ni migraciones.

El catálogo de permisos y las respuestas de roles para usuarios de empresa
proyectan únicamente accesos/acciones de módulos incluidos, activos y
disponibles en su suscripción. Crear o editar un rol no permite asignar valores
fuera de ese alcance. Editar conserva asignaciones almacenadas pero ocultas de
planes anteriores. El administrador de plataforma mantiene el catálogo total.

El backend deriva usuario, organización, local y permisos de la sesión. Nunca
confía en identificadores de alcance enviados por el cliente. Se auditan
anulaciones, descuentos, ajustes, cierres y comprobantes.

La autenticación resuelve en una sola consulta (`SessionAccess`) la sesión, si
el usuario es administrador de plataforma y los permisos de sus roles activos en
el local de la sesión, con las mismas reglas que antes aplicaban por separado
`IsPlatformAdmin` y `HasPermission`. El middleware guarda ese acceso en el
contexto de la petición (`application.WithAccess`) y `Authorize` lo reutiliza
solo para el mismo usuario, empresa y local; para cualquier otro alcance, o si
no hay acceso en el contexto, vuelve a consultar. El acceso no se cachea entre
peticiones: un permiso retirado deja de valer en la siguiente.

El rol reservado `system_key='administrator'` («Administrador de empresa»)
se aplica a todos los locales activos de la misma empresa, incluidos los creados
después de su asignación. No requiere duplicar filas en `user_roles`. La regla
se usa en login, listado/cambio de local, permisos de sesión, autorización,
perfil y opciones/permisos de Caja. Los demás roles siguen limitados al local
asignado. El nombre del rol, `users.manage` o el plan contratado no convierten
un rol personalizado en administrador. Un rol inactivo o retirado no concede
acceso; tampoco se permite saltar a otra empresa ni seleccionar locales inactivos.
El administrador de empresa no obtiene privilegios de plataforma y sus módulos
siguen sujetos a la suscripción. El cambio no modifica cookies ni autenticación.

Los roles mantienen dos dimensiones independientes:

- `menu_access`: opciones y rutas administrativas que el usuario puede abrir.
- `permissions`: acciones de lectura o modificación autorizadas dentro de esas rutas.

El catálogo presenta `menu.read` como «Ver carta y productos» y `menu.manage`
como «Administrar carta y productos», coherentes con el módulo. El cambio de
etiquetas no modifica códigos, asignaciones ni validaciones de autorización.

Ocultar una opción no sustituye la autorización de sus APIs. Toda operación
continúa validando `permissions` en backend; los roles inactivos no conceden
accesos ni permisos. El Administrador de empresa conserva el catálogo explícito
de permisos y accesos definido para ese rol; `*` identifica el acceso de plataforma.

## Propiedad de la atención en mesa

`orders.manage` no permite operar una mesa asignada a otra persona. Edición y
transiciones de atención validan `orders.waiter_id` bajo bloqueo de fila, junto
al alcance de empresa/local. No existe una excepción silenciosa por permiso
amplio. La lectura autorizada se conserva; Cocina y Caja tienen flujos separados
con sus permisos y no transfieren la asignación al preparar o cobrar.
