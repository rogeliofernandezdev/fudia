# FUDIA: monolito modular extraíble y capacidad

**Decisión:** mantener ahora el backend Go como monolito modular hexagonal. Un módulo
no requiere Docker propio. Un servicio se extrae solo cuando existen motivos
operativos medibles (carga, disponibilidad, seguridad o autonomía de equipo).
Los proyectos foods-backend, foods-admin-web y fudia-concierge sí mantienen
Dockerfiles y workflows propios.

## Condiciones para extraer un módulo a Cloud Run

1. **Contratos publicados y versionados.** `internal/modules/<owner>/contracts`
   es el punto de integración. Los alias actuales de entidades de `domain`
   facilitan la compatibilidad de Go, pero **no garantizan independencia de
   versión**: antes de la extracción, reemplazarlos por DTO independientes,
   con traducción en los adaptadores y pruebas de compatibilidad.
2. **Datos de propiedad exclusiva.** Detectar lecturas SQL, JOIN, escrituras y
   transacciones que crucen tablas de varios dominios. La extracción requiere
   eliminar acceso directo a tablas ajenas mediante API/eventos/proyecciones.
3. **Transacciones.** Dentro del monolito la unidad de trabajo PostgreSQL
   compartida mantiene atomicidad de cobros, pedidos y Kárdex. No sustituirla
   por una llamada HTTP sin diseñar consistencia eventual, outbox/inbox,
   deduplicación, idempotencia y reconciliación.
4. **Adaptadores y composición.** Conservar dominio y casos de uso; sustituir
   puertos de infraestructura (HTTP/gRPC/eventos, repositorio, identidad) en
   la raíz de composición del nuevo servicio. Versionar OpenAPI/contratos.
5. **Cloud Run.** Los contenedores deben ser stateless. Antes del escalado,
   migrar imágenes de productos desde `./uploads/products` a Cloud Storage;
   migraciones PostgreSQL en un proceso o Job independiente, no por réplica.
   Configurar instancias máximas, concurrencia y pool Cloud SQL conjuntamente:
   conexiones potenciales = instancias * DATABASE_MAX_CONNS. Exigir readiness
   real (dependencias críticas) y permitir terminación correcta.
6. **Corte gradual.** Extraer una capacidad pequeña de bajo acoplamiento,
   implementar enrutamiento detrás de la API, probar compatibilidad,
   habilitar tráfico progresivo y rollback. Evitar desmembrar Orders/Billing/
   Inventory mientras sus transacciones sigan siendo fuertemente compartidas.

Estas son condiciones y riesgos de extracción futura: el proyecto no declara
que hoy tenga bases de datos o despliegues independientes por módulo.

## Línea base de carga medida: 2026-10-11

- Ejecución: https://github.com/rogeliofernandezdev/fudia/actions/runs/38171538955
- Prueba: `TestCapacityReadProducts` con PostgreSQL 16 en GitHub Actions.
- Ruta HTTP de catálogo con 15 productos, 1/8/32/64 clientes simultáneos,
  scope de test inyectado en vez de autenticación real.
- Resultados medidos (clientes: peticiones, RPS, p95 ms, p99 ms, errores):
  - 1: 80, 1053.5, 2.11, 2.21, 0
  - 8: 320, 2337.0, 6.66, 13.55, 0
  - 32: 640, 2692.0, 12.88, 13.43, 0
  - 64: 640, 2798.0, 23.86, 24.61, 0

**Interpretación:** demuestra capacidad y ausencia de errores únicamente para
esa lectura, ese conjunto de datos y ese runner. No mide autenticación,
escrituras transaccionales, saturación sostenida, red externa, Cloud SQL
remoto ni Cloud Run. El rendimiento puede variar entre ejecuciones.

El workflow del backend repite el escenario y adjunta
`foods-backend-capacity/capacity-report.json` como artefacto cuando la
ejecución finaliza correctamente.

## Puertas de capacidad antes de Cloud Run productivo

- Medir por separado listado de menú, creación de pedidos, cambios KDS,
  pagos concurrentes, inventario, recepciones y lecturas de dashboard.
- Medir p50/p95/p99, RPS, tasa de errores, CPU/memoria, conexiones PostgreSQL,
  bloqueos, cola Redis y límites del proveedor externo.
- Ejecutar pruebas con cargas escalonadas, pico y sostenidas, datos
  representativos, autenticación real y múltiples empresas/locales.
- Definir SLO según el negocio antes de certificar capacidad; no extrapolar
  una prueba de lectura en runner a usuarios reales.
