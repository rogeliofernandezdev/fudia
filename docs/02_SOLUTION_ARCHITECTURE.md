# Arquitectura de solución

## Decisión

La solución tendrá cuatro proyectos desplegables coordinados por contrato. Se
conserva la base técnica de `bodegas`: monolito modular, arquitectura hexagonal,
vertical slices, PostgreSQL, REST/OpenAPI y mensajería solo donde aporte valor.

```text
operations-web (PWA) ─┐
                      ├─ HTTPS/OpenAPI ─ backend ─ PostgreSQL
admin-web ────────────┘                    │
                                           ├─ SUNAT/proveedor OSE
                                           ├─ WhatsApp Business Provider
                                           ├─ impresoras/print bridge
                                           └─ pagos y notificaciones

infrastructure: despliegue, secretos, red, observabilidad y colas
```

## Por qué dos frontends

La operación exige interacción táctil rápida, tolerancia a cortes y despliegues
controlados durante el servicio. La administración prioriza formularios,
tablas, análisis y permisos. Separarlos reduce el riesgo de que una entrega
administrativa interrumpa caja o cocina, sin duplicar el dominio.

## Backend

- Go.
- Monolito modular con puertos y adaptadores.
- PostgreSQL como fuente de verdad.
- REST con OpenAPI versionado.
- Outbox para eventos que deban publicarse con garantía.
- WebSocket o SSE para comandas y cocina en tiempo real.
- Redis únicamente para coordinación efímera demostrable; nunca como verdad.

## Frontends

- Next.js, TypeScript estricto y Tailwind CSS.
- TanStack Query para estado remoto.
- React Hook Form y Zod para formularios.
- BFF y sesión con cookies `HttpOnly` cuando corresponda.
- Componentes propios sobre tokens compartidos, sin copiar identidad de terceros.

## Tenencia y seguridad

Toda entidad operativa pertenece a `organization_id` y, cuando corresponda, a
`location_id`. El backend deriva ambos alcances de la sesión. Auditoría obligatoria
para anulaciones, descuentos, reaperturas, ajustes, cierres y comprobantes.

La empresa es la frontera de tenencia. Sus perfiles fiscales se separan por país;
cada local referencia uno de esos perfiles. Moneda, impuesto y tipo de cambio no
se duplican como preferencias del usuario ni como columnas independientes del
local. Los tipos de cambio son históricos, efectivos por fecha y auditables.
