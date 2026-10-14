# FudIA Image Processor

Servicio independiente de procesamiento de fotografías. No autentica usuarios,
no autoriza uploads y no recibe imágenes del frontend. Solo consume eventos
Cloud Storage finalized mediante Eventarc en un Cloud Run privado.

Genera WebP de hasta 200 y 600 px sin ampliar; valida contenido, dimensiones y
metadatos. Actualiza estados mediante el contrato de persistencia de imágenes.
La configuración usa PRODUCT_IMAGES_BUCKET y DATABASE_URL (usuario exclusivo
con permisos limitados al contrato de imágenes), sin claves JSON.

## Documentación obligatoria

- [Arquitectura y reglas](docs/01_ARCHITECTURE.md): responsabilidades, fronteras,
  flujo, capas, contrato de eventos/SQL, transformación, estados, idempotencia,
  seguridad, límites, limpieza y verificación. Leer antes de cambiar el servicio.
- [Runbook GCP](../foods-infrastructure/docs/10_PRODUCT_IMAGES.md): recursos,
  identidades, grants, variables y despliegue. Validación en GCP pendiente hasta
  que existan esos recursos.

Verificar: go test ./..., go vet ./..., go build ./cmd/processor.
La conversión real requiere vipsthumbnail; ejecutar la prueba dentro del
contenedor construido con este Dockerfile. Configuración e identidades:
[runbook](../foods-infrastructure/docs/10_PRODUCT_IMAGES.md).

