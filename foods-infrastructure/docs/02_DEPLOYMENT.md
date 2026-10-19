# Despliegue

Backend, operaciones y administración se despliegan por separado con versiones
inmutables, health checks y rollback. Migraciones son un paso controlado.

GCP usa el coordinador `foods-gcp` por push a `feat/ui-standard`, solo para
`gcp-dev`: processor → API → frontend. Cada etapa ejecuta pruebas y después
deploy si hay diferencias frente a la revisión activa completada del componente;
si falla, no se ejecuta una etapa dependiente. El siguiente push recupera las
etapas pendientes aunque solo cambie otro componente. El SHA se lee de la revisión
con tráfico y tag `deployed`, no de la última candidata. Cambios compartidos se
evalúan desde la base propia de cada componente. Se conserva ejecución manual de
los componentes. AWS no cambia.
Las migraciones se ejecutan
en un Cloud Run Job antes del API; servicios existentes prueban una candidata
sin tráfico normal y promueven solo tras smoke. Recursos, variables, secrets,
cutover de datos y límites del rollback: [09_GCP_CLOUD_RUN.md](09_GCP_CLOUD_RUN.md).
