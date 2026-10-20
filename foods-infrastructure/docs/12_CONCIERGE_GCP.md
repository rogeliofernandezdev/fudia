# Fudia Concierge en GCP: preparación y despliegue

Implementación de repositorio del 2026-10-20. **No se ha ejecutado el deploy**,
leído secretos reales, modificado IAM ni probado proveedores desde esta tarea.
No cambia AWS, la base restaurada ni la autenticación de usuarios de FudIA.

## Responsabilidades y orden

`foods-gcp.yml` invoca `fudia-concierge-gcp.yml` mediante `workflow_call` por
push a `feat/ui-standard`, en `gcp-dev`, cuando Concierge tiene cambios pendientes.
Usa los archivos del mismo commit: **no requiere copiarlos a master**. El workflow
Concierge conserva `workflow_dispatch`, sin un segundo trigger push ni schedule.
Publicar cambios ejecutables en esa rama puede activar/actualizar HTTP y worker;
confirmar antes las variables, IAM y el costo continuo. No configura Meta.

1. Verifica configuración sin secretos, pruebas de infraestructura, ruff, mypy,
   pytest (incluye integración con Redis temporal), build Python y Docker.
2. Solo en ejecución manual con `deploy_backend`, `activate_concierge` y
   `database_ready` confirmados, invoca `foods-backend-gcp.yml`.
   Conserva sus pruebas, migraciones pendientes, candidata, smoke y promoción.
   No restaura ni reinicializa SQL; exige base verificada y backup. Debe estar
   desplegado el procesador requerido por ese workflow.
   El coordinador pasa `deploy_backend=false`: un cambio exclusivo de Concierge
   no fuerza backend ni migraciones. Si backend también está pendiente, el
   coordinador lo despliega antes por su propia selección.
3. Construye la imagen Concierge antes de obtener credenciales WIF. Publica
   una sola imagen por digest para los tres procesos siguientes.
4. Descubre el origen HTTPS de `GCP_BACKEND_SERVICE` en proyecto/región actuales.
   Verifica que la revisión que realmente sirve 100 % use la misma identidad y
   **referencia numérica** de clave que Concierge. No acepta un backend candidato
   no promovido. No requiere registrar `FUDIA_API_URL` en GitHub.
5. Ejecuta un **Cloud Run Job de preflight**, bajo la identidad del worker,
   con Direct VPC egress y los secretos de runtime. Recibe la CA como PEM en
   memoria, igual que el worker, antes de iniciarlo. Comprueba:
   - AUTH/TLS con CA y validación de identidad del servidor Redis;
   - `noeviction` y Redis sin cluster;
   - salud del API y clave de integración (GET sobre QR intencionalmente inexistente);
   - identificador/token/versión Meta mediante GET, sin enviar mensajes;
   - inferencia sintética en ambos modelos Gemini, sin herramientas ni clientes.
6. Despliega el **HTTP ingress** con `WORKER_ENABLED=false`, cuenta propia,
   mínimo cero, máximo uno por defecto. En servicios existentes prueba una
   candidata sin tráfico normal (`/live`, `/ready`). Primera creación puede
   servir su primera revisión antes del smoke: no configurar Meta aún.
7. Solo tras preflight y smoke inicia/actualiza el **Cloud Run worker pool**,
   `python -m src.worker`, una instancia continua por defecto. No tiene puerto
   ni URL HTTP; no se despliega como servicio HTTP ni como Job infinito.
   En un pool existente se despliega sin promoción y se asigna la revisión
   exacta; no depende de `LATEST`. SIGTERM sale del lifespan y drena hasta 8 s.
8. Promueve la revisión HTTP exacta y publica URLs/digest en el resumen.

Si fallan las pruebas o preflight no arranca un consumidor nuevo. Un fallo de
deploy worker no promueve HTTP existente. El despliegue de HTTP y pool **no es
una transacción**: si falla la promoción final puede haber worker nuevo y HTTP
anterior; mantener compatibilidad entre revisiones y revisar el estado antes de
reintentar. El pipeline no hace rollback de SQL ni borra recursos.

La detección comprueba HTTP con tag `deployed` y 100 % de tráfico, y la revisión
real del worker con 100 % de instancias, ambos Ready y con el mismo SHA/componente.
Una candidata o `latestReadyRevisionName` no completa el release. Un despliegue
parcial queda pendiente para el siguiente push, incluso si solo cambia otro
proyecto. Fallos de permisos/red/historial detienen la detección.

Archivos exclusivos `fudia-concierge/**`, su workflow y `gcp/concierge-*` solo
seleccionan Concierge. Documentación Markdown, `.env.example` y JSON de ejemplo
no seleccionan runtime; **`fudia-concierge/prompts/**` sí**, aunque sean Markdown.
Cambios del detector/coordinador/tests ejecutan las pruebas de infraestructura
sin forzar despliegues. Helpers de infraestructura compartidos seleccionan los
cuatro componentes desde sus respectivas bases desplegadas.

El Job comprueba dependencias reales solo cuando se ejecuta en GCP. No sustituye
evals del router/tools, prueba de pedido/KDS, recepción HMAC ni estados de Meta.
No certifica throughput ni garantiza cero pérdida de datos: Redis Básico con
RDB puede perder escrituras posteriores a la última instantánea.

## Recursos existentes y permisos pendientes de verificar

El operador mostró `fudia-dev`, subred `fudia-concierge-dev` en `us-central1`,
PSA `fudia-services-dev` y Redis Básico 1 GiB `fudia-redis-dev`, privado, sin
cluster, Redis 7.2, AUTH, TLS, `noeviction`, RDB cada hora. Son datos aportados
por capturas, no una auditoría remota. Redis **no se crea ni modifica** por el
workflow. Mantenerlo separado por entorno y vigilar memoria/errores OOM.

| Principal | Permiso / recurso |
| --- | --- |
| `fudia-github-deploy` | Permisos existentes Cloud Run Admin y Artifact Registry Writer; Service Account User sobre las cuentas HTTP y worker además de backend/migrador existentes. Sin acceso al contenido de secretos |
| `fudia-concierge-worker-dev` | `roles/aiplatform.user` en proyecto; `roles/secretmanager.secretAccessor` solo sobre los seis secretos Concierge/Redis |
| `fudia-concierge-http-dev` | Secret Accessor sobre esos mismos seis secretos; no invoca Gemini y no necesita `roles/aiplatform.user` |
| `fudia-api-dev` | **Agregar Secret Accessor únicamente sobre `concierge-api-key-fudia-dev`**; conservar acceso existente a SQL |
| Agente administrado de Cloud Run | Conservar `roles/run.serviceAgent`; acceso a subred del mismo proyecto para Direct VPC. Shared VPC queda fuera del alcance |

Ambos roles reciben la clave del backend: HTTP la necesita para QR y readiness.
El stack/validador común también requiere los tres secretos Meta en ambos; no
afirmar segregación de secretos por función que el código aún no implementa.

En **Secret Manager → `concierge-api-key-fudia-dev` → Permisos → Otorgar acceso**,
agregar el correo completo de la cuenta `GCP_BACKEND_SERVICE_ACCOUNT`, rol
`roles/secretmanager.secretAccessor`, sin condiciones. No usar el deploy como
runtime ni otorgarle acceso a los valores. El workflow no concede IAM.

APIs: Cloud Run, Artifact Registry, Secret Manager, IAM/STS/WIF existentes,
Compute Engine/Service Networking/Memorystore por Redis privado, y
`aiplatform.googleapis.com` (Agent Platform/Vertex AI). El operador confirmó esta
última habilitada y el rol del worker; todavía falta ejecutar una inferencia.
No crear API keys Gemini, credenciales JSON, GPU, notebooks, agentes ni endpoints.
Confirmar disponibilidad de worker pools en la región y facturación/cuotas antes
de activar. No deshabilitar APIs para intentar detener cobros.

## GitHub Environment: qué registrar exactamente

Usar el mismo environment `gcp-dev`, WIF, proyecto, región y Artifact Registry
que los pipelines actuales. Los nombres de variables no se trasladan al
environment AWS. Reutilizar `GCP_PROJECT_ID`, `GCP_REGION`,
`GCP_ARTIFACT_REPOSITORY`, `GCP_WORKLOAD_IDENTITY_PROVIDER`,
`GCP_DEPLOY_SERVICE_ACCOUNT`, `GCP_BACKEND_SERVICE` y
`GCP_BACKEND_SERVICE_ACCOUNT` y `GCP_BACKEND_SECRETS`. Las demás variables del
backend son necesarias si se selecciona su despliegue. **No duplicar GOOGLE_CLOUD_PROJECT:** el
deploy lo deriva de `GCP_PROJECT_ID` para runtime. Proteger entornos/ramas y no
ejecutar el coordinador y este release simultáneamente.

Crear estas **Environment variables**, no GitHub Secrets con credenciales:

El detector del coordinador también exige `GCP_CONCIERGE_HTTP_SERVICE` y
`GCP_CONCIERGE_WORKER_POOL`, aunque otro componente origine el push. Si faltan,
falla explícitamente antes de desplegar; no interpreta configuración ausente
como servicio sin cambios.

| Nombre | Valor para el entorno mostrado |
| --- | --- |
| `GCP_CONCIERGE_HTTP_SERVICE` | `fudia-concierge-dev` |
| `GCP_CONCIERGE_WORKER_POOL` | `fudia-concierge-worker-dev` |
| `GCP_CONCIERGE_PREFLIGHT_JOB` | `fudia-concierge-check-dev` |
| `GCP_CONCIERGE_HTTP_SERVICE_ACCOUNT` | `fudia-concierge-http-dev@project-fb804674-13ae-4b1f-b63.iam.gserviceaccount.com` |
| `GCP_CONCIERGE_WORKER_SERVICE_ACCOUNT` | `fudia-concierge-worker-dev@project-fb804674-13ae-4b1f-b63.iam.gserviceaccount.com` |
| `GCP_CONCIERGE_NETWORK` | `fudia-dev` |
| `GCP_CONCIERGE_SUBNET` | `fudia-concierge-dev` |
| `GCP_CONCIERGE_REDIS_CA_SECRET` | `redis-ca-fudia-dev:1` **si la versión habilitada es 1** |
| `WHATSAPP_PHONE_ID` | ID real del número de Meta, no el teléfono ni WABA ID |
| `WHATSAPP_GRAPH_VERSION` | Versión soportada seleccionada en Meta; el operador eligió `v26.0`. Solo versión, sin URL |

`GCP_CONCIERGE_SECRETS`: una sola variable que contiene **referencias**, no
valores. Sustituir cada `:1` por la versión numérica habilitada que realmente
corresponde; no se ha inspeccionado ni se presupone su contenido:

```text
REDIS_URL=redis-url-fudia-dev:1,FUDIA_CONCIERGE_API_KEY=concierge-api-key-fudia-dev:1,WHATSAPP_TOKEN=whatsapp-token-fudia-dev:1,WHATSAPP_VERIFY_TOKEN=whatsapp-verify-token-fudia-dev:1,WHATSAPP_APP_SECRET=whatsapp-app-secret-fudia-dev:1
```

**Editar `GCP_BACKEND_SECRETS`, conservando su DATABASE_URL y número actuales**,
y añadir la misma referencia de integración. Ejemplo si SQL usa versión 2 y
la clave usa 1:

```text
DATABASE_URL=database-url-fudia-dev:2,FUDIA_CONCIERGE_API_KEY=concierge-api-key-fudia-dev:1
```

Esto no reemplaza la contraseña SQL, no migra usuarios a JWT y no cambia el
login. La clave de integración no procede del login: autentica servicio a
servicio. La comparación es de referencias y la prueba HTTP verifica que el
backend acepte la clave; el pipeline nunca descarga/imprime los valores.
Cambiar esta variable no modifica una revisión ya desplegada. Si backend aún no
monta esa referencia, desplegarlo explícitamente primero, con datos/backup
verificados. El preflight de Concierge rechaza un backend sin la clave esperada.

Opcionales (ya tienen defaults explícitos):

| Variable | Default |
| --- | --- |
| `GOOGLE_CLOUD_LOCATION` | `global` (independiente de región Cloud Run) |
| `GEMINI_MODEL_LITE` | `gemini-3.5-flash-lite` |
| `GEMINI_MODEL_FLASH` | `gemini-3.5-flash` |
| `SESSION_TTL_SECONDS` | `3600` desde la última escritura de sesión, no retención de pedidos ni de recuperación |
| `QUEUE_WORKER_CONCURRENCY` | `8` |
| `GCP_CONCIERGE_WORKER_INSTANCES` | `1`, proceso activo continuo; sin autoscaling de cola implementado |
| `GCP_CONCIERGE_HTTP_MAX_INSTANCES` | `1` |

Modelos conservan los defaults aprobados del proyecto; el preflight valida acceso
a ambos, sin sustituir modelos automáticamente ni tocar la política de tools.
`TABLE_IDLE_SECONDS`, memoria de cliente y retención de procesamiento/cola no
se reducen a una hora. `ENV=production` activa validaciones incluso en gcp-dev.

[Ejemplo de variables nuevas](../gcp/concierge-dev.variables.example.json).
No importar el ejemplo ciegamente: phone/version y las referencias del backend
se completan explícitamente, y las versiones de secretos deben verificarse.

## Credenciales y certificado en runtime

- `redis-url-fudia-dev`: URI real `rediss://:PASSWORD_URL_ENCODED@IP_PRIVADA:PUERTO_TLS/0`.
  La credencial AUTH se codifica para URI si contiene caracteres reservados.
- `redis-ca-fudia-dev`: PEM descargado de Memorystore, con todas las CA válidas.
  HTTP conserva el montaje de solo lectura en `/var/run/secrets/redis/server-ca.pem`.
  Worker y Job de preflight reciben la misma referencia numérica desde Secret
  Manager como `REDIS_CA_CERT_PEM`, con `REDIS_CA_CERTFILE` vacío. El cliente carga
  el PEM directamente en memoria, sin archivos temporales ni volúmenes de secretos.
  Esto evita el rechazo de la CLI al montar secretos en worker pools observado
  durante el primer despliegue. No exige un secreto, variable GitHub ni rol nuevos.
  La aplicación aplica la misma CA a sesiones, locks, rate limit, cola y DLQ.
  No configurar simultáneamente archivo y PEM; no se imprime el certificado.
- TLS exige certificado válido y hostname/IP coincidente; no se permite URI
  con `ssl_cert_reqs=none`, `ssl_check_hostname=false` ni overrides `ssl_*`.
  Ante una CA incorrecta, faltante o rotada, el despliegue falla: renovar bundle,
  crear nueva versión del secreto y actualizar referencia, nunca desactivar TLS.
- `private-ranges-only`: Redis por VPC; Meta/Gemini/API HTTPS salen por ruta
  normal. No requiere Cloud NAT ni conector Serverless VPC Access para este caso.
- WIF autentica GitHub; ADC usa la cuenta runtime para Gemini. Son identidades
  diferentes y no se descargan claves JSON de ninguna.

## Ejecución y cierre (no realizados)

1. Confirmar IAM, versiones de secretos, datos/backup SQL, variables y cuotas.
2. Si backend todavía no monta la clave compartida, preparar su despliegue
   explícito antes del bot; conservar DATABASE_URL y no reinicializar SQL.
3. Publicar código revisado a `feat/ui-standard`. En Actions abrir `foods-gcp`
   del mismo commit y su job `concierge`. No se hizo commit/push desde esta tarea.
   Los demás componentes se omiten si no tienen cambios pendientes; una falla
   anterior o falta de marcas de release sí puede seleccionarlos.
4. Alternativa manual: Actions → `fudia-concierge-gcp` → Run workflow → entorno.
   El botón exige el archivo en la rama predeterminada; el trigger automático
   anterior no. `activate_concierge=false` solo verifica. Para desplegar el bot
   sin backend: `activate_concierge=true`, `deploy_backend=false`.
   Para desplegar también backend: las tres casillas deben estar confirmadas.
   No ejecutar la alternativa manual en paralelo al coordinador.
5. Revisar gates de pruebas → preflight → HTTP → worker → promoción, y backend
   antes si fue seleccionado. El resumen
   entrega origen HTTP y callback `https://ORIGEN/webhook`.
6. En Meta configurar callback y el valor privado de WHATSAPP_VERIFY_TOKEN,
   suscripción a mensajes/estados y validar desafío/HMAC. No poner ese token
   en una URL pública, captura, GitHub variable ni log.
7. Registrar origen HTTP como `FUDIA_CONCIERGE_URL` en el mismo Environment y
   ejecutar el frontend GCP existente para conectar sus enlaces QR. El workflow
   Concierge no cambia ni despliega frontend; el coordinador solo lo selecciona
   si tiene cambios pendientes. Cambiar una variable GitHub no dispara un push.
8. Probar QR de empresa/local de prueba con `whatsapp_bot` habilitado, carta,
   confirmación explícita, pedido persistido, KDS, cuenta y estados Meta.
   No cambiar DNS ni apagar AWS hasta validar el corte de escrituras.

Redis provisionado y el pool continuo generan costos incluso sin conversaciones.
El preflight añade un Job corto e inferencias sintéticas al ejecutar el workflow.
Backups, logs, red, almacenamiento de imágenes y el backend conservan sus costos.
No hay pausa/borrado automático ni compromiso de costo mensual fijo.

## Verificación local

`node --check foods-infrastructure/gcp/concierge-deploy.mjs` y
`node --test foods-infrastructure/tests/*.test.mjs`; en Concierge:
`ruff check .`, `mypy src config`, `pytest -q`, `python -m build`, Docker build.
Mocks cubren configuración, referencias, orden y barreras; no prueban IAM,
Direct VPC ni disponibilidad/cuotas de modelos reales. Los tests no llaman
`src.preflight` contra el entorno real.

Fuentes: [worker pools](https://docs.cloud.google.com/run/docs/deploy-worker-pools),
[CLI worker deploy](https://docs.cloud.google.com/sdk/gcloud/reference/run/worker-pools/deploy),
[Direct VPC](https://docs.cloud.google.com/run/docs/configuring/vpc-direct-vpc),
[Secret Manager](https://docs.cloud.google.com/run/docs/configuring/services/secrets).
