# AGENTS — Image Processor

Aplicar AGENTS.md raíz. Leer README.md y docs/ antes de cambios relevantes.
docs/01_ARCHITECTURE.md es el contrato de responsabilidades e invariantes de
este proyecto; no implementar cambios que lo contradigan sin actualizar la
decisión y sus contratos coordinadamente.
Responsabilidad exclusiva: validar, transformar y publicar fotografías.
No introducir login, endpoints comerciales, autorización de uploads ni frontend.
Eventos autenticados por IAM de Cloud Run; procesamiento dentro de la petición.
PostgreSQL es la autoridad de UUID; el procesador nunca genera identificadores.
Mantener dominio/aplicación independientes de GCS, PostgreSQL y libvips.
No cambiar el esquema: las migraciones pertenecen a foods-backend.
Generar únicamente WebP 200/600 sin ampliar; READY exige ambas variantes.
Preservar generación, lease, fencing, tenencia y protección de fotos reemplazadas.
Usar solo el contrato SQL de imágenes y una identidad/usuario de mínimo privilegio.
No firmar URLs de lectura, recibir uploads del frontend ni cambiar sesiones a JWT.
Limpieza del original solo después de READY durable; no borrar variantes/fallidos
mediante la regla de lifecycle exitoso ni afirmar que existe recuperación/DLQ.
No declarar IAM, CORS o Eventarc verificados sin recursos reales y evidencia.
Ejecutar tests, vet y build; validar conversión real e idempotencia.

