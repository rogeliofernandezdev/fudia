# PostgreSQL

Servicio administrado con cifrado, backups, restauración probada, pool y métricas.
Credenciales solo mediante gestor de secretos. En el MVP GCP se permite el endpoint
público de Cloud SQL: Cloud Run conecta por socket del Auth Proxy administrado,
con IAM y transporte cifrado, sin VPC ni autorizar rangos públicos de runners.
Para DBeaver directo solo se autoriza la IP individual del operador con TLS.

El backend recibe una única `DATABASE_URL`: `sslmode=require` para TCP directo;
`sslmode=disable` exclusivamente para el salto local por socket `/cloudsql/`,
ya que el proxy cifra la conexión remota. Los frontends
no conocen host, usuario ni contraseña. Las migraciones viven en
`foods-backend/migrations` y se ejecutan como paso controlado de despliegue.

Decisión temporal del MVP: API, migrador y processor pueden compartir usuario SQL
y secreto. No es aislamiento SQL de mínimo privilegio; antes de producción separar
DDL, DML y grants de imágenes. Las cuentas IAM de runtime siguen separadas.
