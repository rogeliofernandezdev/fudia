# Secretos

Secretos en gestor central con identidad administrada, rotación y mínimo privilegio.
Nunca en repositorios, imágenes, logs ni variables públicas de Next.js.

En GCP, GitHub se autentica por OIDC/WIF, sin clave JSON permanente. Variables
de los environments `gcp-*` contienen nombres/referencias a versiones numéricas
de Secret Manager, no credenciales. Las cuentas runtime acceden solo a sus
secrets. El token público Mapbox conserva su nombre GitHub, pero se integra al
build y no es confidencial. Matriz: [09_GCP_CLOUD_RUN.md](09_GCP_CLOUD_RUN.md).
