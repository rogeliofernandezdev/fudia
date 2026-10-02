# Seguridad

La UI representa permisos, pero backend autoriza. No guardar tokens en localStorage.
Proteger mutaciones contra CSRF y pedir motivo para acciones sensibles.

Cada rol tiene dos controles independientes. `menuAccess` decide qué opciones
aparecen y qué rutas puede abrir; `permissions` decide qué acciones puede
ejecutar dentro de ellas. El sidebar y la protección de rutas consumen
`menuAccess`; los botones consumen permisos específicos y el backend vuelve a
validar cada petición. Conceder un permiso no concede automáticamente una opción
del menú, ni conceder una opción autoriza sus acciones.

## Sesión en el navegador

Cuando una sesión expira, la interfaz muestra una transición breve y vuelve al
inicio de sesión. Si la validación falla temporalmente por red o por una
indisponibilidad del backend, conserva el sistema visual y ofrece acciones para
reintentar o volver al inicio de sesión; nunca presenta contenido sin estilos.

Una cookie `HttpOnly` representa una sola identidad por perfil y origen del
navegador. Un segundo inicio de sesión nunca reemplaza silenciosamente una
sesión válida: la interfaz exige continuar con la identidad actual o cerrar la
sesión antes de cambiar de cuenta. Login y logout se comunican entre pestañas
mediante `BroadcastChannel`, sin transmitir tokens ni información sensible, y
el contexto se vuelve a consultar al recuperar el foco. Las pruebas simultáneas
de roles distintos requieren perfiles o contextos de navegador aislados.
