# Foods Infrastructure

Infraestructura como código. Antes de implementar, leer AGENTS.md y todos los
documentos de docs/.

## Despliegue GCP sin retirar AWS

Workflow `foods-gcp` por push a `feat/ui-standard`: processor → API → frontend
en `gcp-dev`, ejecutando componentes con cambios pendientes desde su revisión
activa y completada en Cloud Run, no solo desde el último push. Cambios compartidos
se evalúan desde la base propia de cada componente. Pruebas → deploy en cada
etapa; un fallo no hace olvidar los componentes bloqueados.
Reutiliza los tres workflows
GCP, que conservan ejecución manual. OIDC/WIF, Artifact
Registry, Job de migración, Cloud SQL por socket administrado (sin VPC en el MVP), fotos persistentes y secretos
de runtime en Secret Manager. Preparación y checklist:
[GCP Cloud Run](docs/09_GCP_CLOUD_RUN.md).

Configuración concreta del MVP sin VPC, nueva versión de DATABASE_URL y orden
processor → API → frontend: [checklist gcp-dev](docs/11_GCP_DEV_CHECKLIST.md).

Fotografías: [carga directa y procesador](docs/10_PRODUCT_IMAGES.md), proyecto
`../fudia-image-processor` y workflow `foods-image-processor-gcp.yml` independiente.
El API GCP ya no monta el bucket ni recibe los bytes de las nuevas fotos.

Verificación local: `node --check gcp/deploy.mjs` y
`node --test tests/*.test.mjs` (Node 22; comandos cloud simulados).
