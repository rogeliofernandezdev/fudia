# PostgreSQL

Servicio administrado con cifrado, backups, restauración probada, pool y métricas.
No se expone públicamente. Credenciales solo mediante gestor de secretos.

El backend recibe una única `DATABASE_URL` con `sslmode=require`; los frontends
no conocen host, usuario ni contraseña. Las migraciones viven en
`foods-backend/migrations` y se ejecutan como paso controlado de despliegue.
