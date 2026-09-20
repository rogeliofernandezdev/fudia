# AGENTS — Infraestructura

Infraestructura como código de la plataforma. Aplicar el `AGENTS.md` raíz.

Regla obligatoria: leer el `README.md` de este proyecto y todos los documentos
de `docs/` antes de implementar cambios relevantes.

## Reglas

- Entornos separados y reproducibles.
- Secretos solo en un gestor de secretos; nunca en código o estado público.
- PostgreSQL administrado con backups y restauración probada.
- Despliegues independientes para backend, operaciones y administración.
- Logs estructurados, métricas, trazas, alertas y correlación extremo a extremo.
- Redis y mensajería solo con una decisión de arquitectura documentada.
