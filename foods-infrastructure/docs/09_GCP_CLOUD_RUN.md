# Backend y Admin Web en GCP Cloud Run

## Alcance y estado

Preparación de despliegue, no evidencia de un despliegue real. Los workflows
`foods-gcp.yml` coordina processor → API → frontend por push a `feat/ui-standard`,
según los paths de los proyectos/infraestructura/workflows, usando `gcp-dev`.
Cada etapa exige pruebas → deploy exitosos antes de la siguiente. Reutiliza los
tres workflows del mismo commit con `workflow_call`; conservan `workflow_dispatch`
para operación manual, sin triggers push duplicados.
El job `changes` se autentica por OIDC/WIF en `gcp-dev` y solo ejecuta lecturas
Cloud Run: obtiene la revisión completada que sirve el 100 % del tráfico de cada
componente, lee su SHA y compara ese commit con el del push. No utiliza
`github.event.before` como base de despliegue. Así recupera cambios de frontend
bloqueados por un fallo anterior aunque el nuevo push solo corrija backend.
Sin cambios pendientes en los otros componentes, un cambio solo de frontend no
ejecuta backend, processor ni Job SQL. Las diferencias de infraestructura
compartida/coordinador se evalúan desde la base propia de cada componente.
Si falla la consulta/detección, no se despliega a ciegas.
Las condiciones permiten dependencias omitidas, pero bloquean fallos/cancelaciones.

### Identidad de código y despliegues pendientes

- Cada revisión nueva recibe `fudia-commit` (SHA completo) y `fudia-component`.
  Se consulta metadata de la revisión, nunca las etiquetas mutables del servicio.
- El tag `deployed` se asigna a la revisión exacta junto con su promoción al
  100 %, después de smoke en API/frontend. Processor lo hace solo después de
  startup y configuración exitosa de IAM/Eventarc. No certifica una entrega real.
- Candidatas sin tráfico, `latestReadyRevisionName` o un workflow exitoso de otro
  componente no avanzan la base. Una primera revisión auto-live que falla smoke
  tampoco tiene marca completada y permanece pendiente.
- Servicio inexistente explícitamente, o instalación anterior sin tag/SHA:
  seleccionar ese componente para desplegar una vez e inicializar su identidad.
  La primera publicación de esta corrección puede seleccionar los tres servicios.
- IAM, red, JSON/identidad inválidos o revisión ausente son errores, no ausencia
  de cambios. La detección no escribe outputs parciales si falla una consulta.
- El SHA desplegado debe existir en Git y ser ancestro del commit solicitado.
  Una ejecución antigua o historia divergente falla para evitar downgrade ciego.
- Un reparto de tráfico o rollback que no coincida con `deployed` se considera
  pendiente, no una base certificada. No ejecutar releases manuales/canaries a la
  vez que el coordinador; para mantener un rollback, pausar los pushes automáticos.
- El resumen de `changes` indica componente, revisión, SHA y motivo de selección.
  No se necesitan nuevas variables, claves JSON ni escrituras GitHub para guardar
  marcas. La identidad de deploy existente puede describir servicios/revisiones;
  la detección solo invoca `describe`, sin acceder a Secret Manager ni runtime env.

### Alcance de ejecución

No modifican los workflows de AWS, sus triggers ni el environment `dev`.
Primero se ejecuta processor/migración, después backend y finalmente frontend. Fudia Concierge/Redis no se
migran en este alcance: conservar un `FUDIA_CONCIERGE_URL` HTTPS alcanzable si
se utilizan QR/WhatsApp y actualizar en ese servicio su destino de backend a
GCP al hacer el corte. Dejarlo escribiendo en el API AWS crearía pedidos en la
base antigua, no en Cloud SQL. Sin esa reconexión no se certifica ese flujo.

No se crean recursos, se cambian DNS ni se copian datos al agregar estos archivos.
Los workflows sí crean/actualizan servicios y un Job al ejecutarlos con acceso.
La región, proyecto, facturación y la base destino deben decidirse antes.

**Decisión del usuario (2026-10-12): solo fotografías desde cero.** Crear un
bucket vacío y subir fotos nuevas; no copiar fotografías de AWS. **La base y sus
datos deben conservarse.** Se eligió **copiar PostgreSQL actual a Cloud SQL y
hacer un corte controlado**, no crear una base funcional vacía ni conservar
AWS como dependencia de red permanente. El Job aplica versiones SQL pendientes; no importa datos ni
borra/recrea la base. No ejecutar bootstrap de cuentas: conservar usuarios actuales.
AWS permanece disponible y no se borra ningún recurso allí.

## Arquitectura

```text
GitHub Actions --OIDC/WIF--> Artifact Registry --> Cloud Run
Navegador --> Admin Web/BFF --> API --> Cloud SQL
    |                                     ^
    +-- subida firmada --> Storage --> Eventarc --> fudia-image-processor
                               ^                       |
                               +-- WebP 200/600 -------+
```

Frontend y backend son servicios HTTPS con acceso público a nivel Cloud Run
(`--no-invoker-iam-check`). La autorización de negocio permanece en el backend,
con sesión, organización/local y permisos. El BFF actual no obtiene ID tokens
de IAM; volver privado el API sin implementar ese cliente rompería login y
consultas. El bucket permanece privado. Para el MVP (decisión 2026-10-13),
Cloud SQL tiene IP pública y Cloud Run conecta mediante el Auth Proxy administrado
y socket Unix, con IAM y transporte remoto cifrado; no hay VPC ni conector de pago.
No significa abrir SQL a todas las IP. CORS no sustituye autenticación:
el navegador usa el BFF del mismo origen. `ALLOWED_ORIGINS` aparece en el ejemplo
histórico, pero actualmente no es un middleware efectivo de este backend.

No se necesita cambiar los Dockerfiles de AWS:

- Backend escucha en `HTTP_ADDR=:8080`.
- Next standalone respeta el `PORT=8080` que inyecta Cloud Run, aunque el valor
  por defecto de su Dockerfile sea 5175. No configurar `PORT` como secret/env propio.
- API comprueba PostgreSQL antes de escuchar y recibe SIGTERM. Su `/health`
  confirma proceso/versión; NO comprueba continuamente la base ni el bucket.
- La probe HTTP de arranque y la prueba de la candidata no certifican todos los
  flujos. Validar login, consulta autenticada, foto, guardar y KDS tras desplegar.

## Recursos que hay que crear en GCP

Un conjunto por entorno; preferir proyectos distintos para desarrollo,
staging y producción. Nunca conectar desarrollo a la base productiva.

1. Proyecto con facturación vinculada y presupuesto/alertas.
2. APIs: Cloud Run, Artifact Registry, IAM, IAM Credentials, Security Token
   Service, Secret Manager, Cloud SQL Admin, Cloud Storage, Eventarc y Pub/Sub.
   Compute Engine, Service Networking y Serverless VPC Access solo para la opción privada.
3. Repositorio **Docker** de Artifact Registry en la región elegida.
4. MVP: Cloud SQL PostgreSQL con IP pública, misma región, backups y PITR.
   API, Job y processor montan la instancia con `GCP_CLOUD_SQL_INSTANCE` y usan
   `/cloudsql/PROYECTO:REGION:INSTANCIA`. Requieren `roles/cloudsql.client`.
   PostgreSQL 18 es la referencia de los workflows GCP; AWS conserva su CI.
   Restaurar los datos e historial existentes antes del primer deploy.
   No autorizar rangos públicos de GitHub o Cloud Run ni `0.0.0.0/0`.
5. Alternativa privada, solo si se configura `GCP_VPC_CONNECTOR`: VPC propia,
   rango reservado de Private Service Access y conexión a
   `servicenetworking.googleapis.com`. Cloud SQL PostgreSQL con **solo IP privada**,
   versión igual o superior al origen y compatibilidad de restore verificada
   (16 es la referencia de CI, no una medición de la versión actual de AWS),
   en esa VPC/región, backups automáticos y PITR. En producción definir HA y
   probar restauración y **restaurar los datos existentes antes del primer deploy**.
   No autorizar rangos públicos ni el runner de GitHub.
   Conector Serverless VPC Access regional `READY`, en la VPC que alcanza la base,
   con subred/rango dedicado sin solapamientos. API y Job usan el conector y
   `private-ranges-only`; HTTPS a servicios externos conserva salida normal.
   No se configura Direct VPC en esta versión. El conector tiene coste aun sin tráfico.
   Los dos modos son excluyentes: no definir ambas variables.
6. Conservar la base y usuarios existentes. Preferir usuario de migración con DDL
   y usuario de aplicación sin DDL; si se crean roles nuevos, darles permisos sobre
   el esquema/restauración existentes, sin crear datos ni reemplazar contraseñas.
   Habilitar extensiones exigidas por las migraciones, entre ellas `pgcrypto` y
   `pg_trgm`, con una identidad autorizada si el usuario de migración no puede.
   Excepción temporal MVP: los tres componentes SQL pueden usar el mismo usuario
   y la misma referencia de secreto; no afirmar aislamiento de permisos SQL.
   Antes de producción separar usuarios. En el modo separado, dar al usuario de
   aplicación USAGE en schema, DML en tablas, acceso a secuencias
   y default privileges para objetos futuros creados por el usuario de migración.
7. Bucket exclusivo para fotos, misma región, uniform bucket-level access y
   public access prevention. Activar versionado/retención de borrados según política
   y lifecycle para versiones antiguas. No usar bucket público ni URLs con claves.
8. Secret Manager con versiones numéricas habilitadas:
   - `foods-dev-database-url`: conexión del usuario de aplicación.
   - `foods-dev-migration-database-url`: conexión del usuario de migración.
   - `foods-dev-concierge-api-key`: si se conserva Concierge.
   - `foods-dev-mapbox-server-token`: si se usa geocodificación Mapbox.
9. Seis cuentas de servicio dedicadas: GitHub deploy, backend runtime,
   frontend runtime, migration runtime, processor runtime y Eventarc. Sin claves JSON.
10. Workload Identity Pool y proveedor OIDC para GitHub, restringido al repositorio
    y al environment/ref aprobados. Configurar los permisos de la siguiente sección.

Para el traslado, si se exporta un dump, usar **otro bucket privado temporal**
sin montaje en el backend ni permiso de runtime; nunca guardar backups SQL,
documentos privados o `.env` en el bucket de fotos servido por el API.

Los servicios Cloud Run y el Job los crea el pipeline: no hace falta crear
contenedores vacíos manualmente. No se usa Cloud Build: Docker construye en el
runner y publica en Artifact Registry. Si cambia la región, actualizar también
repositorio, Cloud SQL y bucket (y conector si existe); la ruta de Registry usa `GCP_REGION`.

La conexión secreta usa el nombre existente `DATABASE_URL`. En modo sin VPC:

```text
postgres://USUARIO:PASSWORD_URL_ENCODED@localhost/NOMBRE_BASE?host=/cloudsql/PROYECTO:REGION:INSTANCIA&sslmode=disable
```

`host` en el query reemplaza `localhost` por el socket. No es una conexión TCP
sin TLS: `sslmode=disable` solo aplica al socket local y Auth Proxy cifra la red.
No se cambia la autenticación de FudIA ni se exige login SQL mediante IAM.
Crear una nueva versión del secreto existente y actualizar las tres referencias
numéricas en GitHub. No leer/publicar el valor del secreto desde el pipeline.

Para la alternativa TCP privada:

```text
postgres://USUARIO:PASSWORD_URL_ENCODED@IP_PRIVADA:5432/foods?sslmode=require
```

Crear el valor en Secret Manager, no en GitHub Variables, terminal compartido,
capturas o archivos versionados. `sslmode=require` cifra; no verifica nombre/CA
como `verify-full`. Un endurecimiento con CA/verificación requiere configuración
adicional y prueba de conexión. No usar `sslmode=disable` con una IP TCP remota.

## Identidades y permisos mínimos operativos

| Identidad | Permisos y alcance |
| --- | --- |
| GitHub deploy | `roles/run.admin` y `roles/eventarc.admin` en proyecto; `roles/artifactregistry.writer` solo en repositorio; `roles/iam.serviceAccountUser` solo sobre las cinco cuentas de servicios/Job/Eventarc. `roles/vpcaccess.user` + `roles/compute.viewer` solo en opción privada |
| Backend runtime | `roles/cloudsql.client` en proyecto SQL; `roles/secretmanager.secretAccessor` solo en DATABASE_URL y Concierge si aplica; `roles/storage.objectCreator` + `roles/storage.objectViewer` solo en bucket; signBlob sobre sí misma. Sin Delete/Update |
| Migration runtime | `roles/cloudsql.client` en proyecto SQL; `roles/secretmanager.secretAccessor` solo en la URL de migración; sin bucket ni tokens de terceros |
| Frontend runtime | `roles/secretmanager.secretAccessor` solo en MAPBOX_ACCESS_TOKEN si aplica; sin SQL ni bucket |
| Processor runtime | `roles/cloudsql.client`, acceso a su secreto SQL y `roles/storage.objectUser` solo en bucket |
| Eventarc | `roles/eventarc.eventReceiver` en proyecto; `roles/run.invoker` solo en processor (el pipeline añade este binding antes del trigger) |
| Agente administrado de Cloud Run | Mantener `roles/run.serviceAgent`; permiso de conector solo en opción privada. Shared VPC/cross-project queda fuera de este runbook |

Cuando Fudia Concierge se despliegue en GCP (hoy fuera de alcance) necesitará
su propia cuenta de servicio, sin claves JSON: `roles/aiplatform.user` en el
proyecto con Vertex AI habilitado (`aiplatform.googleapis.com`), porque invoca
Gemini Flash-Lite/Flash con Application Default Credentials, y
`roles/secretmanager.secretAccessor` solo en sus secretos (Meta, Redis,
`FUDIA_CONCIERGE_API_KEY`). Requisitos y variables en
`fudia-concierge/docs/07_MODEL_PROVIDERS.md`. Roles ingress/worker, Redis
privado con `noeviction` y sin cluster, apagado con drenaje y alertas en
`fudia-concierge/docs/08_RECOVERY_AND_OPERATIONS.md`.

No otorgar Owner/Editor al pipeline ni usar la cuenta Compute predeterminada.
La conexión socket usa credenciales PostgreSQL, sin claves JSON; el proxy
administrado autentica mediante IAM y cifra la red. No requiere VPC. La opción
TCP privada usa credenciales PostgreSQL y TLS; no requiere inyectar
una clave de cuenta de servicio ni montar el socket Cloud SQL Auth Proxy.

### OIDC desde GitHub sin una clave de GCP

Estos comandos son una plantilla para **Cloud Shell**, para ejecutar tras revisar
los identificadores. No se han ejecutado ni autorizan automáticamente recursos.
Un administrador crea pool/proveedor, bindings y APIs; el pipeline no lo hace.

```bash
GCP_PROJECT_ID='TU_PROYECTO'
GCP_PROJECT_NUMBER='NUMERO_DEL_PROYECTO'
GITHUB_REPOSITORY_ID='ID_NUMERICO_DEL_REPOSITORIO'
GITHUB_OWNER_ID='ID_NUMERICO_DEL_PROPIETARIO'
DEPLOY_ACCOUNT="github-deploy@${GCP_PROJECT_ID}.iam.gserviceaccount.com"

gcloud iam workload-identity-pools create github \
  --project="$GCP_PROJECT_ID" --location=global --display-name='GitHub deploy'

# Este ejemplo confía solo en gcp-dev. Crear proveedor/pool/binding separado
# para staging/producción y restringir también assertion.ref a la rama aprobada.
gcloud iam workload-identity-pools providers create-oidc github \
  --project="$GCP_PROJECT_ID" --location=global --workload-identity-pool=github \
  --issuer-uri='https://token.actions.githubusercontent.com' \
  --attribute-mapping='google.subject=assertion.sub,attribute.repository_id=assertion.repository_id,attribute.repository_owner_id=assertion.repository_owner_id' \
  --attribute-condition="assertion.repository_id=='${GITHUB_REPOSITORY_ID}' && assertion.repository_owner_id=='${GITHUB_OWNER_ID}' && assertion.sub.endsWith(':environment:gcp-dev')"

gcloud iam service-accounts add-iam-policy-binding "$DEPLOY_ACCOUNT" \
  --project="$GCP_PROJECT_ID" --role=roles/iam.workloadIdentityUser \
  --member="principalSet://iam.googleapis.com/projects/${GCP_PROJECT_NUMBER}/locations/global/workloadIdentityPools/github/attribute.repository_id/${GITHUB_REPOSITORY_ID}"
```

El número del proyecto, IDs de GitHub y resource name del provider no son secretos.
Usar IDs numéricos evita que renombrar/reutilizar un nombre dé acceso a otro repo.
El `sub` de estos jobs contiene el environment, no la rama: si se requiere
restricción de rama, incluir explícitamente `assertion.ref` en la condición.
En GitHub proteger `gcp-production` con reviewers y ramas permitidas, impedir
autoaprobación y bypass según el plan disponible. Sin reglas de protección, el
nombre `production` NO añade seguridad. Las reglas del provider deben coincidir.

## GitHub: environments, Variables y Secrets

Crear `gcp-dev`, `gcp-staging` y `gcp-production` según los entornos habilitados.
AWS conserva `dev` y todas sus variables; no renombrarlas ni eliminarlas.
Poner los valores GCP en **Environment variables**, no compartirlos por defecto
desde Repository variables con producción. No se versionan archivos `.env`.

### Variables comunes a ambos workflows

| Variable | Ejemplo no secreto |
| --- | --- |
| `GCP_PROJECT_ID` | `tu-proyecto-dev` |
| `GCP_REGION` | región elegida; por ejemplo `southamerica-west1` (confirmar disponibilidad/coste/latencia) |
| `GCP_ARTIFACT_REPOSITORY` | `foods` |
| `GCP_WORKLOAD_IDENTITY_PROVIDER` | `projects/123456789/locations/global/workloadIdentityPools/github/providers/github` |
| `GCP_DEPLOY_SERVICE_ACCOUNT` | `github-deploy@tu-proyecto-dev.iam.gserviceaccount.com` |
| `GCP_BACKEND_SERVICE` | `foods-backend-dev` |

### Variables del backend

| Variable | Ejemplo |
| --- | --- |
| `GCP_BACKEND_SERVICE_ACCOUNT` | `foods-backend@tu-proyecto-dev.iam.gserviceaccount.com` |
| `GCP_MIGRATION_JOB` | `foods-migrate-dev` |
| `GCP_MIGRATION_SERVICE_ACCOUNT` | `foods-migrate@tu-proyecto-dev.iam.gserviceaccount.com` |
| `GCP_CLOUD_SQL_INSTANCE` | `proyecto:region:instancia`, mismo proyecto/región; modo socket sin VPC |
| `GCP_VPC_CONNECTOR` | omitir en el MVP; alternativa TCP privada excluyente con la anterior |
| `GCP_UPLOADS_BUCKET` | nombre global único del bucket, solo letras/números/guiones |
| `GCP_BACKEND_SECRETS` | `DATABASE_URL=foods-dev-database-url:1,FUDIA_CONCIERGE_API_KEY=foods-dev-concierge-api-key:1` |
| `GCP_MIGRATION_DATABASE_SECRET` | `foods-dev-migration-database-url:1` |
| `DATABASE_MAX_CONNS` | opcional; default `12` |
| `GCP_BACKEND_MAX_INSTANCES` | opcional; default `3` |
| `GCP_BACKEND_CONCURRENCY` | opcional; default `40` |

`GCP_BACKEND_SECRETS` contiene únicamente **referencias**, no valores. Se exige
DATABASE_URL y versiones numéricas, no `latest`, para poder reproducir y revertir
una revisión. Concierge puede omitirse de la lista si no se usa; eso no habilita
sus llamadas. El Job recibe únicamente su propia URL, nunca los tokens del API.

### Variables y secret del frontend

| Configuración | Dónde | Ejemplo / uso |
| --- | --- | --- |
| `GCP_FRONTEND_SERVICE` | Environment Variable | `foods-admin-web-dev` |
| `GCP_FRONTEND_SERVICE_ACCOUNT` | Environment Variable | `foods-frontend@tu-proyecto-dev.iam.gserviceaccount.com` |
| `GCP_FRONTEND_SECRETS` | Environment Variable | `MAPBOX_ACCESS_TOKEN=foods-dev-mapbox-server-token:1`; vacío si no se usa Mapbox |
| `FOODS_API_URL` | Environment Variable, opcional | si se omite, descubre URL del servicio backend en ese proyecto/región; override solo origen HTTPS |
| `FUDIA_CONCIERGE_URL` | Environment Variable, opcional | origen HTTPS del Concierge existente; no `localhost` ni nombre de red Docker de EC2 |
| `NEXT_PUBLIC_MAP_PROVIDER` | Environment Variable | `mapbox` |
| `NEXT_PUBLIC_MAPBOX_ACCESS_TOKEN` | Environment Secret | mismo nombre usado en AWS; token **público**, integrado al build y visible al navegador |
| `GCP_FRONTEND_MAX_INSTANCES` | Environment Variable, opcional | default `3` |
| `GCP_FRONTEND_CONCURRENCY` | Environment Variable, opcional | default `40` |

El token privado `MAPBOX_ACCESS_TOKEN` vive en Secret Manager y solo se inyecta
en runtime/BFF. Si se configura el token público, se exige la referencia privada
para no desplegar el mapa con geocodificación rota. Ajustar las restricciones
de origen del token público a la URL/dominio GCP; no reutilizar restricciones
solo válidas para el dominio AWS. El token público puede seguir en GitHub Secrets
por compatibilidad, pero `NEXT_PUBLIC_` significa público, no confidencial.
El deploy valida que el token cartográfico público empiece por `pk.` y rechaza
tokens `sk.` para no incorporarlos al JavaScript público.

**No crear `GCP_CREDENTIALS` ni copiar AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY.**
La autenticación GCP utiliza OIDC/WIF. El archivo temporal `gha-creds-*.json`
queda ignorado preventivamente; estos workflows usan `create_credentials_file:
false` y no lo generan. En los jobs de deploy Docker se construye **antes** de autenticar; después se
solicita un access token de cuenta de servicio de 3600 s para gcloud, transmitido
como output enmascarado de Actions y no impreso. Se evita consumir los 5 minutos
del token OIDC durante builds largos. No existe una clave permanente de GCP.
Cambiar un secret de Secret Manager exige actualizar su número de versión en
la referencia y desplegar una revisión. Cambiar `NEXT_PUBLIC_*` exige rebuild.

Excepción necesaria del coordinador: `changes` autentica antes de detectar para
leer la versión activa en Cloud Run, sin construir ni publicar imágenes. Debe
usar `environment: gcp-dev` para que coincidan el subject WIF y las variables
protegidas existentes. No ampliar la condición WIF a jobs sin environment.

## Fotografías nuevas y persistencia

Bucket vacío y datos PostgreSQL conservados. Las fotos nuevas suben directamente
desde el navegador con política POST firmada. El API recibe metadatos/referencias,
no el archivo, y ya no monta Storage FUSE. El proyecto independiente
fudia-image-processor genera WebP 200/600 mediante Eventarc en Cloud Run privado.
Sesiones actuales sin migración a JWT.

Recursos adicionales, CORS, lifecycle, grants y orden de despliegue:
[10_PRODUCT_IMAGES.md](10_PRODUCT_IMAGES.md). Desplegar processor/trigger primero;
su workflow aplica las migraciones usando la imagen del backend, no del worker.

No copiar fotos de AWS. No borrar/reemplazar las URLs antiguas en la base:
pueden mostrar fallback/404 hasta reemplazarlas explícitamente. GET/HEAD legacy
/uploads/products permanece para AWS/local. Nuevas imágenes GCP usan objectKey
y URLs temporales generadas en consultas.

Las migraciones no trasladan datos de AWS ni sustituyen backup/restore.
Conservar UUID/schema_migrations, verificar recuentos/login y coordinar corte
sin escrituras. Volver DNS a AWS no reconcilia datos nuevos de Cloud SQL.

## Ejecución, capacidad y rollback

1. Crear recursos/secretos/bindings y environments anteriores, con bucket de
   fotos vacío. Ensayar backup/restore de los datos actuales a Cloud SQL; verificar
   UUID, usuarios, migraciones aplicadas, recuentos y backups antes de deploy.
   No se ha realizado la copia: requiere IDs de recursos y acceso autorizado.
2. Commit/push a `feat/ui-standard` activa `foods-gcp` sin registrar archivos en
   la rama por defecto. Revisar Actions → `foods-gcp`: processor → backend → frontend.
   La secuencia completa se serializa sin cancelar migraciones en curso. Cada
   etapa seleccionada exige `verify` y `deploy` exitosos para habilitar la siguiente.
   Las que no tienen diferencias pendientes aparecen `skipped`. Se encolan hasta 100 pushes pendientes
   (`queue: max`) para no reemplazar un push con cambios de otro componente.
   Los triggers de AWS siguen vigentes: un push en su rama de despliegue puede
   desplegar allí. Sus archivos y variables no se modifican.
3. El push habilita migraciones pendientes solo para `gcp-dev`, tomando como
   precondición la base restaurada por el operador y backups verificados antes
   del primer push. No realiza ni certifica restore/backup; no ejecutarlo contra
   una base vacía o productiva. No se añaden variables ni credenciales nuevas.
   La alternativa manual conserva `database_ready=false` por defecto y exige
   confirmación. Para mostrar `Run workflow`, GitHub exige el archivo en la rama
   por defecto; esa condición no aplica al trigger por push.
4. CI verifica Go y PostgreSQL temporal. Construye Docker linux/amd64, publica
   tag SHA+run+attempt y despliega por **digest**. Crea/actualiza Job y espera su
   terminación. En socket, `jobs deploy` monta Cloud SQL y `jobs update` retira
   red/conector antes de execute (deploy no admite clear-vpc-connector).
   Sin éxito de migración no despliega el servicio.
5. API candidata con probe HTTP. En servicios existentes no recibe tráfico
   normal hasta pasar `/health` con la versión esperada; entonces se promueve
   esa revisión específica al 100 %. En la primera creación Cloud Run sirve
   su primera revisión antes de la prueba post-deploy: aún no cambiar DNS.
6. Después del backend, el workflow frontend ejecuta lint, typecheck,
   suite y build; si falla, no despliega. Descubre API o usa override, verifica
   `/login` candidata y promueve. La URL/revisión/digest quedan en job summary.
7. Validar flujos autenticados y red/bucket, actualizar restricciones Mapbox,
   conectar Concierge si se usa. Solo entonces considerar DNS/dominio propio.

Ambos tienen concurrencia por environment/servicio y no cancelan migraciones
en curso. **No lanzar manualmente el Job en paralelo**: el migrador actual no
tiene advisory lock global y no se altera en este alcance. No mezclar migraciones
AWS/GCP contra una misma base sin coordinación externa.

Máximos iniciales: 3 instancias, 40 requests concurrentes, CPU 1 y RAM 1 GiB.
No son una certificación de capacidad. El pool default tiene 12 conexiones:
3 × 12 = 36 por revisión, más 2 del Job. Reservar margen para coexistencia de
revisiones durante despliegues (hasta 72 o más si se mantienen candidatas),
administración y otras cargas. Escalado máximo no es una garantía absoluta de
conexiones. Ajustar Cloud SQL/pool/concurrencia en conjunto y medir cargas reales.
Cloud SQL, backups, almacenamiento y builds generan costes; conector solo en la
alternativa privada. Escala a
cero de Cloud Run no elimina esos cargos. Presupuesto no es un tope de gasto.

Para volver a una revisión compatible, un operador autorizado revisa el nombre
y ejecuta (plantilla, no automatismo):

```bash
gcloud run services update-traffic SERVICIO --project=PROYECTO --region=REGION \
  --to-revisions=REVISION_ANTERIOR=100
```

No deshacer SQL automáticamente ni eliminar revisiones/objetos/backups. Mantener
migraciones backward-compatible; restaurar datos requiere un plan separado.
Una candidata que falla no se promueve, pero sus migraciones ya aplicadas
permanecen. Para rollback histórico conservar digests y versiones de secrets.

## Verificación local

`node --check foods-infrastructure/gcp/deploy.mjs` y
`node --test foods-infrastructure/tests/*.test.mjs` comprueban configuración,
secrets numéricos, nombres/HTTPS, identidades separadas, argumentos de Job/servicio,
orden migrate → candidata → smoke → promoción y bloqueo tras fallos. Las pruebas
usan comandos simulados: no crean recursos ni ejecutan gcloud/Docker.
`foods-admin-web/tests/uploads-route.test.mjs` prueba bytes/HEAD, errores,
filtrado de nombres/MIME y ausencia de cookies/proxy abierto.
`pending-components.test.mjs` crea un repositorio Git temporal y reproduce
cambios de frontend bloqueados → corrección solo backend → recuperación del
frontend pendiente. Cubre distintas bases por componente, primera instalación,
legado sin etiquetas, candidata sin tráfico, IAM/red, historial y estado inválido.
Las pruebas de promoción verifican que smoke, migración o Eventarc fallidos no
actualicen la marca completada. No hacen llamadas a recursos GCP reales.
Además ejecutar las puertas generales de los proyectos. No omitir pruebas
fallidas para habilitar un despliegue; el pipeline se mantiene bloqueado.

## Fuentes oficiales consultadas

- [Autenticación GitHub/GCP por WIF](https://github.com/google-github-actions/auth).
- [CLI para servicios](https://docs.cloud.google.com/sdk/gcloud/reference/run/deploy) y
  [Jobs](https://docs.cloud.google.com/sdk/gcloud/reference/run/jobs/deploy).
- [Conexión Cloud SQL desde Cloud Run](https://docs.cloud.google.com/sql/docs/postgres/connect-run).
- [Importar/exportar PostgreSQL](https://docs.cloud.google.com/sql/docs/postgres/import-export/import-export-sql).
- [Montaje Cloud Storage y limitaciones FUSE](https://docs.cloud.google.com/run/docs/configuring/services/cloud-storage-volume-mounts).
- [Acceso público de Cloud Run](https://docs.cloud.google.com/run/docs/authenticating/public).
- [Workload Identity con pipelines y condiciones](https://docs.cloud.google.com/iam/docs/workload-identity-federation-with-deployment-pipelines).
- [Etiquetas de revisiones](https://docs.cloud.google.com/run/docs/configuring/services/labels) y
  [tráfico/tags](https://docs.cloud.google.com/sdk/gcloud/reference/run/services/update-traffic).
