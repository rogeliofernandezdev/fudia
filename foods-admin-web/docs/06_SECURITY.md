# Seguridad

La UI representa permisos, pero backend autoriza. No guardar tokens en localStorage.
Proteger mutaciones contra CSRF y pedir motivo para acciones sensibles.

Cada rol tiene dos controles independientes. `menuAccess` decide qué opciones
aparecen y qué rutas puede abrir; `permissions` decide qué acciones puede
ejecutar dentro de ellas. El sidebar y la protección de rutas consumen
`menuAccess`; los botones consumen permisos específicos y el backend vuelve a
validar cada petición. Conceder un permiso no concede automáticamente una opción
del menú, ni conceder una opción autoriza sus acciones.
