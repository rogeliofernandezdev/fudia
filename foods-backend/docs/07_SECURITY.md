# Seguridad

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
