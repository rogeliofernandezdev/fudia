# AGENTS — WhatsApp Gateway

Aplicar AGENTS.md raíz. Leer README.md, api/openapi.yaml y todos los documentos
de docs/ antes de cambios relevantes. Este servicio solo transporta WhatsApp;
no agregar IA, pedidos, selección de restaurantes ni configuración operativa por país.
Conservar autenticación privada, referencias originales, sesiones SQL protegidas,
idempotencia y espera aleatoria superior a dos segundos. Ejecutar go test ./...,
go vet ./... y go build ./...; las integraciones SQL usan base temporal exclusiva.
No declarar probado WhatsApp real con dobles de transporte.
