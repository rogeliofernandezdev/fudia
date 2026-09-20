# Seguridad

El backend deriva usuario, organización, local y permisos de la sesión. Nunca
confía en identificadores de alcance enviados por el cliente. Se auditan
anulaciones, descuentos, ajustes, cierres y comprobantes.

Los roles mantienen dos dimensiones independientes:

- `menu_access`: opciones y rutas administrativas que el usuario puede abrir.
- `permissions`: acciones de lectura o modificación autorizadas dentro de esas rutas.

Ocultar una opción no sustituye la autorización de sus APIs. Toda operación
continúa validando `permissions` en backend; los roles inactivos no conceden
accesos ni permisos. El rol Administrador usa `*` en ambas dimensiones.
