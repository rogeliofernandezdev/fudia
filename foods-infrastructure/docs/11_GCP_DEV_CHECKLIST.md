# gcp-dev: checklist para ejecutar los pipelines

Configuración acordada con el operador el 2026-10-13. No es evidencia de un
despliegue ni de una auditoría remota de IAM. AWS no cambia. Actualización:
`foods-gcp` coordina el deploy por push a `feat/ui-standard`, solo en
`gcp-dev`: processor → backend → frontend, cada uno con pruebas → deploy.

## Datos confirmados por el operador

- Proyecto: `project-fb804674-13ae-4b1f-b63`, número `64178523363`.
- Región: `us-central1`.
- Registry Docker: `artifact-fudia-dev`.
- Bucket privado: `bucket-dev-fudia`.
- Instancia SQL: `postgres-fudia-dev`, PostgreSQL 18, IP pública, sin VPC.
- Base restaurada: `postgres`; el historial mostrado llega a `000079_cash_shift_reports`.
  El Job ejecutará las migraciones pendientes, incluida `000080_product_images`;
  no hace bootstrap ni borra los datos restaurados.
- Environment GitHub: `gcp-dev`, repositorio `rogeliofernandezdev/fudia`.
- Usuario SQL/secreto compartido para el MVP; IAM de cada servicio separado.

## 1. Actualizar la conexión de Secret Manager

En GCP, Secret Manager → `database-url-fudia-dev` → Versiones → Añadir nueva
versión. Guardar solamente esta conexión, sustituyendo el marcador de contraseña:

```text
postgres://postgres:PASSWORD_URL_ENCODED@localhost/postgres?host=/cloudsql/project-fb804674-13ae-4b1f-b63:us-central1:postgres-fudia-dev&sslmode=disable
```

La contraseña es la actual del usuario PostgreSQL, codificada para URI. No se
guarda en GitHub ni en el repositorio. Rotar cualquier contraseña divulgada antes
de subir esta versión. No copiar una IP remota con `sslmode=disable`: aquí `host`
selecciona el socket Unix local y el Auth Proxy de Cloud Run cifra la red hacia
Cloud SQL. DBeaver directo mantiene su conexión por IP pública con TLS.

Si es la segunda versión del secreto, las referencias son:

| Variable GitHub (Environment variables) | Valor |
| --- | --- |
| GCP_BACKEND_SECRETS | `DATABASE_URL=database-url-fudia-dev:2` |
| GCP_MIGRATION_DATABASE_SECRET | `database-url-fudia-dev:2` |
| GCP_IMAGE_PROCESSOR_DATABASE_SECRET | `database-url-fudia-dev:2` |

Si Google asigna otro número, reemplazar `2` por ese número habilitado en las
tres referencias. No usar `latest` ni publicar la conexión como variable pública.

## 2. Completar las variables de GitHub

Valores concretos sin credenciales en
[gcp-dev.variables.example.json](../gcp/gcp-dev.variables.example.json).
Crear/verificar cada clave como variable de Environment `gcp-dev`; no importar
estas variables al environment AWS `dev`. Los nombres de servicio/Job/trigger
propuestos se crearán en el primer pipeline: no son recursos que el operador
haya confirmado ya desplegados.

Obligatorio:

```text
GCP_CLOUD_SQL_INSTANCE = project-fb804674-13ae-4b1f-b63:us-central1:postgres-fudia-dev
```

No definir `GCP_VPC_CONNECTOR` en este entorno. El script rechaza ambos modos
simultáneos; conserva el modo privado opcional para otros environments GCP.

Frontend: omitir `FOODS_API_URL` para descubrir automáticamente `status.url` del
servicio `GCP_BACKEND_SERVICE` en este proyecto/región y entregarlo al BFF en
runtime. Si existe un valor heredado de AWS, retirarlo o sustituirlo por el origen
HTTPS del API GCP. Nunca localhost, IP SQL ni `NEXT_PUBLIC_DATABASE_URL`.
Puede configurarse un override `FOODS_API_URL` HTTPS explícito; los tests cubren
ambas opciones. No es necesario rebuild para cambiar el destino del BFF.

Mapbox es opcional para el deploy inicial. Si se usa, añadir el token público
`NEXT_PUBLIC_MAPBOX_ACCESS_TOKEN` en GitHub Secrets y la referencia privada
`MAPBOX_ACCESS_TOKEN=nombre-secreto:version` en `GCP_FRONTEND_SECRETS`, con acceso
de `fudia-web-dev` solo a ese secreto. No se copia ningún token privado al build.
Sin ambos tokens no se certifican mapas/geocodificación. Concierge requiere su
URL HTTPS y reconectar su backend al hacer el corte; no se despliega aquí.

## 3. Comprobar APIs y permisos

APIs: Cloud Run (`run.googleapis.com`), Artifact Registry, Secret Manager, Cloud
SQL Admin (`sqladmin.googleapis.com`), IAM, IAM Service Account Credentials
(`iamcredentials.googleapis.com`), Security Token Service (`sts.googleapis.com`),
Cloud Storage, Eventarc (`eventarc.googleapis.com`) y Pub/Sub.
No habilitar VPC Access/Service Networking para este modo.

| Principal | Recurso | Permiso |
| --- | --- | --- |
| fudia-github-deploy | Proyecto | Cloud Run Admin + Eventarc Admin |
| fudia-github-deploy | artifact-fudia-dev | Artifact Registry Writer |
| fudia-github-deploy | Las cinco cuentas runtime/Eventarc | Service Account User |
| fudia-api-dev, fudia-migrate-dev, fudia-processor-dev | Proyecto SQL | Cloud SQL Client |
| Las mismas tres cuentas | database-url-fudia-dev | Secret Manager Secret Accessor |
| fudia-api-dev | bucket-dev-fudia | Storage Object Creator + Viewer |
| fudia-api-dev | Su propia cuenta de servicio | Service Account Token Creator (signBlob) |
| fudia-processor-dev | bucket-dev-fudia | Storage Object User |
| fudia-eventarc-dev | Proyecto | Eventarc Event Receiver |
| service-64178523363@gs-project-accounts.iam.gserviceaccount.com | Proyecto | Pub/Sub Publisher |

WIF: cuenta de deploy conectada al subject
`repo:rogeliofernandezdev/fudia:environment:gcp-dev`, con Workload Identity User.
No descargar claves JSON. El processor pipeline añade `roles/run.invoker` a
Eventarc solo sobre `fudia-processor-dev`, antes de crear el trigger.

## 4. Ejecutar en orden, solo cuando el código esté publicado

No se ha hecho commit/push ni se han ejecutado estos workflows desde esta tarea.
Publicar todos los archivos necesarios de la implementación GCP/imágenes en
`feat/ui-standard`. Un push a la rama AWS puede activar su workflow existente:
revisar sus triggers; mantener intactos los archivos no evita ese comportamiento.

### Despliegue automático por push

1. Verificar datos restaurados, backup, secretos/variables y permisos anteriores.
2. Hacer commit y push a `feat/ui-standard`. Cambios en los proyectos,
   infraestructura o workflows GCP activan **foods-gcp**.
3. GitHub → Actions → **foods-gcp** → ejecución del commit. Deben terminar
   correctamente processor (verify/deploy), backend (verify/deploy) y frontend
   (verify/deploy), en ese orden. Si una etapa falla, no se ejecuta la siguiente.
   Las migraciones de processor/API conservan exclusión mutua. No se generan
   despliegues duplicados: los tres componentes se invocan mediante workflow_call.
4. Abrir la URL frontend del resumen de su deploy. El BFF recibe
   automáticamente la URL del API GCP, salvo override HTTPS explícito.

No se requiere copiar workflows a `master` para este trigger. El pipeline usa
el código del mismo push y las variables/secrets existentes de `gcp-dev`, sin
variables nuevas. Las migraciones pendientes quedan habilitadas para desarrollo:
esto no restaura datos ni comprueba automáticamente su integridad/backups.
La concurrencia no cancela despliegues/migraciones activos; puede reemplazar una
ejecución pendiente. No usar estos recursos para producción.

### Alternativa manual

Se conservan los triggers manuales de cada componente. Para mostrar el botón
`Run workflow`, sus archivos deben existir en la rama predeterminada de GitHub;
esa condición no aplica al despliegue automático anterior.

1. Actions → **foods-image-processor-gcp** → Run workflow → `gcp-dev`.
   Marcar `database_ready` únicamente tras confirmar restore/datos y backup.
   Verifica migraciones con PostgreSQL 18 temporal, convierte una imagen real
   en el contenedor, aplica pendientes en Cloud SQL y despliega processor/Eventarc.
2. Actions → **foods-backend-gcp** → `gcp-dev`, misma confirmación de datos.
   Migra pendientes sin repetir las registradas, prueba candidata y promueve.
3. Actions → **foods-admin-web-gcp** → `gcp-dev`. Descubre el API y entrega
   `FOODS_API_URL` al frontend. Verifica `/login` antes de promover.

Si falla la verificación o migración, no continuar a la siguiente etapa.
Los tests locales de scripts simulan comandos; no certifican conectividad,
IAM/Eventarc reales ni integridad completa de la restauración.

## 5. Antes de probar fotografías en el navegador

Con la URL real del frontend del job summary, adaptar
[product-images-cors.example.json](../gcp/product-images-cors.example.json)
a ese origen exacto. Aplicar CORS al bucket con la cuenta del operador:

```bash
gcloud storage buckets update gs://bucket-dev-fudia --cors-file=cors.json
gcloud storage buckets update gs://bucket-dev-fudia --lifecycle-file=foods-infrastructure/gcp/product-images-lifecycle.json
```

`cors.json` debe ser el archivo adaptado, no un ejemplo con placeholders o `*`.
Estas operaciones no se ejecutan automáticamente en el pipeline ni dan permisos
de administración de bucket a GitHub. Lifecycle borra solo originales exitosos
marcados por Custom-Time, no todas las fotos por antigüedad.

Después verificar login y datos restaurados, guardar producto, preview inmediato,
upload directo y estado READY/WebP 200/600. No cambiar DNS ni apagar AWS hasta
confirmar estos flujos y el corte de escrituras.
