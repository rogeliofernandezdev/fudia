# Estado de módulos y prioridad de implementación

> Documento vivo para retomar el desarrollo de FUDIA sin perder el punto de
> continuidad. Debe actualizarse cuando un módulo cambia de estado, se cierra
> una vertical o cambia la prioridad del roadmap.

## Última revisión

- Fecha: **2026-09-24**
- Rama revisada: `feat/redesign-nueva-comanda`
- Commit de implementación/hardening verificado: `27db62e5d770b9fb7ebf35a95475bb0117a5b112`
- Puerta final de rama:
  - `foods-backend` run **1382**: **success** — migrate, tests, vet y build.
  - `foods-admin-web` run **2016**: **success** — lint, typecheck, tests y build.
  - `fudia-concierge` run **324**: **success** — ruff, mypy, pytest, build Python y build Docker.
  - inspección de logs de los tres jobs: **0 warnings**.
- Trabajo activo: **P0 — Facturación electrónica**
- Fudia Concierge: **✅ 100% cerrado en repositorio/CI para el alcance definido**
- Gate externo de Concierge: validación de go-live con credenciales reales; no es código pendiente.

## Cómo interpretar los estados

| Estado | Significado |
| --- | --- |
| ✅ **100% cerrado** | Alcance actual definido, UI y backend reales, persistencia, permisos, multiempresa/local cuando aplica, contrato actualizado y CI verde. |
| 🟢 **Ready / activable** | El catálogo permite activarlo y existe implementación funcional, pero aún no se ha hecho una auditoría de cierre completa para declararlo 100%. |
| 🟡 **Parcial** | Una parte importante funciona, pero faltan capacidades incluidas en el alcance del módulo. |
| 🔵 **En desarrollo** | Existe base o pantalla, pero no está habilitado como módulo terminado para empresas. |
| ⚪ **Planificado** | Está contemplado en el catálogo/roadmap, pero no debe considerarse implementado. |

**Regla:** no marcar un módulo como **100% cerrado** únicamente porque
`moduleAvailability` sea `ready`. El cierre exige verificar el flujo completo.

---

## Capacidades cerradas y verificadas en esta revisión

Estas capacidades quedaron implementadas y cubiertas por CI/pruebas, aunque
algunas forman parte de un módulo mayor.

### ✅ Onboarding operativo de empresa

Una empresa nueva queda creada en una sola transacción con:

- empresa y perfil fiscal inicial;
- local principal;
- administrador inicial;
- roles y permisos predeterminados;
- suscripción y módulos permitidos por el plan;
- medios de pago copiados desde plantillas de base de datos;
- `Caja principal`;
- zona `Principal`;
- categorías iniciales de gastos;
- posibilidad de abrir un turno de caja inmediatamente.

No se crean datos comerciales ficticios como productos, proveedores, clientes,
recetas, mesas o movimientos.

### ✅ Medios de pago

- Catálogo único por empresa en `payment_methods`.
- Administración desde `/configuracion/medios-pago`.
- Crear, editar, ordenar, activar y desactivar.
- Uso independiente para Ventas y Gastos.
- Propiedad `affects_cash` para determinar impacto físico en Caja.
- Código interno inmutable una vez creado.
- Protección para no dejar Ventas o Gastos sin medios operativos.
- POS y Gastos consumen el catálogo desde backend; no mantienen listas propias.
- Pagos y gastos referencian el catálogo en base de datos.

### ✅ Gastos operativos

La vertical de **Gastos** está cerrada para su alcance actual:

- categorías de gasto;
- alta de gasto por empresa/local;
- fecha, descripción, importe, referencia y notas;
- medio de pago dinámico;
- filtros, búsqueda y paginación;
- detalle;
- anulación auditable sin borrado destructivo;
- permisos `expenses.read`, `expenses.manage` y `expenses.void`;
- integración con el onboarding y el catálogo de medios de pago.

**Importante:** esto no significa que el módulo completo **Costos y gastos**
esté terminado. Falta la parte de costeo y rentabilidad.

---

## Estado actual del catálogo de módulos

### Control

| Módulo | Clave | Estado | Observación |
| --- | --- | --- | --- |
| Reportes | `reportes` | 🟢 Ready / activable | Existe como módulo activo del MVP. Requiere auditoría de cierre y deberá incorporar resultados de costeo/rentabilidad. |

### Operación

| Módulo | Clave | Estado | Observación |
| --- | --- | --- | --- |
| Punto de venta | `pos` | 🟢 Ready / activable | Flujo real de cobro; medios de pago dinámicos y pago dividido integrados. Auditoría final de cierre pendiente. |
| Pedidos | `pedidos` | 🟢 Ready / activable | Módulo habilitado en MVP. Auditoría de cierre pendiente. |
| Cocina | `cocina` | 🟢 Ready / activable | KDS habilitado en MVP. Auditoría de cierre pendiente. |
| Mesas y zonas | `mesas` | 🟢 Ready / activable | Gestión por local implementada; zona inicial creada por onboarding. Auditoría de cierre pendiente. |
| Caja y turnos | `caja` | 🟢 Ready / activable | Apertura/cierre, movimientos, pagos y devoluciones integrados. Auditoría final de cierre pendiente. |
| Reservas | `reservas` | 🟢 Ready / activable | Disponible en MVP. Auditoría de cierre pendiente. |
| Carta digital QR | `carta_qr` | 🔵 En desarrollo | No declararla terminada. |
| Call center | `call_center` | ⚪ Planificado | Pendiente. |
| Kiosco de autoservicio | `kiosco` | ⚪ Planificado | Pendiente. |

### Carta y producción

| Módulo | Clave | Estado | Observación |
| --- | --- | --- | --- |
| Carta y productos | `productos` | 🟢 Ready / activable | Disponible en MVP. Auditoría de cierre pendiente. |
| Menús y combos | `combos` | 🟢 Ready / activable | Disponible en MVP. Auditoría de cierre pendiente. |
| Recetas | `recetas` | 🟢 Ready / activable | Disponible en MVP; será dependencia del costeo real. Auditoría de cierre pendiente. |

### Abastecimiento

| Módulo | Clave | Estado | Observación |
| --- | --- | --- | --- |
| Inventario | `inventario` | 🟢 Ready / activable | Stock por local disponible. Auditoría de cierre pendiente. |
| Kardex | `kardex` | 🟢 Ready / activable | Disponible en MVP. Auditoría de cierre pendiente. |
| Compras | `compras` | 🟢 Ready / activable | Órdenes, proveedores y recepción disponibles. Auditoría de cierre pendiente. |
| Logística | `logistica` | ⚪ Planificado | Pendiente. |

### Delivery

| Módulo | Clave | Estado | Observación |
| --- | --- | --- | --- |
| Delivery propio | `delivery` | ⚪ Planificado | Pendiente. |
| Apps de delivery | `delivery_apps` | ⚪ Planificado | Pendiente. |
| App repartidores | `repartidores` | ⚪ Planificado | Pendiente. |

### Negocio

| Módulo | Clave | Estado | Observación |
| --- | --- | --- | --- |
| Clientes | `clientes` | 🟢 Ready / activable | Disponible en MVP. Auditoría de cierre pendiente. |
| Locales | `locales` | 🟢 Ready / activable | Multi-local y configuración por local disponibles. Auditoría de cierre pendiente. |
| CRM y fidelización | `crm` | ⚪ Planificado | Pendiente. |
| Plaza puntos | `puntos` | ⚪ Planificado | Pendiente. |
| Ofertas y descuentos | `ofertas` | ⚪ Planificado | Pendiente. |
| Personal y asistencias | `personal` | ⚪ Planificado | Pendiente. |

### Inteligencia

| Módulo | Clave | Estado | Observación |
| --- | --- | --- | --- |
| Costos y gastos | `costos` | 🟡 Parcial | **Gastos está terminado.** Falta costo de recetas/productos, costo teórico/real, food cost, margen y rentabilidad. |
| Restaurant BI | `bi` | ⚪ Planificado | Debe apoyarse en ventas, costos e inventario consolidados. |
| App manager | `app_manager` | ⚪ Planificado | Pendiente. |

### Configuración

| Módulo | Clave | Estado | Observación |
| --- | --- | --- | --- |
| Fiscal y moneda | `fiscal` | 🟢 Ready / activable | Perfil fiscal y moneda reales. Auditoría de cierre pendiente. |
| Usuarios y roles | `usuarios` | 🟢 Ready / activable | Roles/permisos forman parte del onboarding. Auditoría de cierre pendiente. |
| Facturación | `facturacion` | 🔵 En desarrollo | Pantallas actuales no constituyen facturación electrónica real. **Siguiente prioridad.** |
| Integraciones | `integraciones` | 🔵 En desarrollo | Pendiente completar adaptadores operativos. |
| Fudia Concierge | `whatsapp_bot` | ✅ **100% cerrado** | QR/WhatsApp, menú real, pedidos/KDS, combos, modificadores, rondas, cuenta, handoff, multiagente, cola durable, aislamiento multicanal, resiliencia, observabilidad, evals y CI/Docker están cerrados. La activación comercial sigue dependiendo del plan/entitlement; el go-live requiere credenciales externas. |

---

## Cierre auditado — Fudia Concierge

Concierge completa el alcance funcional y técnico definido para esta etapa.

### Capacidades funcionales cerradas

- QR resuelto por foods-backend y configuración por empresa/local;
- número de WhatsApp administrable y activación/desactivación;
- carta real por nombre, descripción y categoría;
- disponibilidad, precio y stock revalidados en backend;
- carrito temporal, combos y modificadores;
- confirmación explícita protegida por código;
- `conversationId` estable y `requestId` idempotente por ronda;
- múltiples rondas sobre una misma comanda aunque rondas anteriores avancen en KDS;
- tickets de cocina independientes por ronda;
- cuenta acumulada, pagos y saldo leídos desde backend;
- handoff humano persistente y resoluble;
- bandeja operativa real;
- aislamiento por número receptor/tenancy.

### Hardening cerrado

- LangGraph con router real a especialistas `menu`, `order` y `service`;
- schemas de tools separados por especialista y autorización duplicada en `ToolRegistry`;
- Redis Streams durable en lugar de `BackgroundTasks`;
- deduplicación atómica, ACK, retries, recuperación de pendientes y dead-letter;
- lock Redis renovable por conversación;
- sesión identificada por `phone_number_id + cliente`, hasheada en Redis;
- rate limiting y límite de entrada;
- timeouts y retries seguros/idempotentes;
- fail-fast de secretos/configuración en producción;
- endpoints `/live`, `/ready` y `/metrics`;
- correlation tracing y métricas Prometheus de baja cardinalidad;
- eval corpus de scope, intención y prompt injection;
- tests de mínimo privilegio;
- dependencias fijadas/constraints;
- contenedor no-root con healthcheck;
- CI valida lint, tipos, tests, build de paquete y build Docker.

### Evidencia de interfaz y contrato

- Admin Web de Concierge incluye loading, error, reintento, validación E.164, permisos y layout adaptable;
- página QR pública incluye estados loading/error y deeplink WhatsApp;
- backend tiene migraciones reversibles e integration tests para settings, handoff, rondas, KDS y cuenta;
- el contrato server-to-server usa credencial propia y no confía en datos comerciales proporcionados por el modelo.

### Gate externo de activación

Antes de un go-live real se debe validar el webhook y envío con credenciales reales de Meta/OpenAI/Redis/Fudia y ejecutar una prueba de humo QR -> WhatsApp -> múltiples rondas -> KDS -> cuenta.

Ese gate depende del entorno y secretos autorizados. **No es una deuda de implementación del repositorio y no se declara ejecutado sin evidencia externa.**

---

## Orden de prioridad acordado

### P0 — Facturación electrónica

**Siguiente implementación.**

Objetivo: cerrar el circuito principal:

`pedido → preparación → cobro → caja → comprobante fiscal`

Debe incluir como mínimo:

1. configuración administrativa de facturación;
2. series/correlativos por local y tipo de comprobante;
3. boleta y factura vinculadas al pedido/pago real;
4. datos tributarios del cliente;
5. adaptador de proveedor homologado;
6. estados de emisión, errores y reintentos idempotentes;
7. historial real en `/ventas/comprobantes`;
8. impresión/representación del comprobante;
9. notas de crédito/anulación según soporte del proveedor;
10. permisos, auditoría, OpenAPI y pruebas;
11. onboarding/configuración inicial necesaria para una empresa nueva.

**No construir un motor SUNAT propio.** La arquitectura define al proveedor
electrónico como adaptador externo.

### P1 — Costeo real y rentabilidad

Completar `Costos y gastos`:

- costo de insumos;
- costo por receta;
- costo por producto/plato;
- costo teórico vs. real;
- costo por porción;
- food cost;
- margen bruto;
- impacto de mermas;
- gastos operativos ya implementados;
- rentabilidad por producto, categoría, local y periodo.

Al cerrar esta prioridad el módulo `costos` podrá evaluarse para **100% cerrado**.

### P2 — Reportes consolidados

Cerrar y auditar `reportes` utilizando datos reales de:

- ventas;
- caja;
- comprobantes;
- costos;
- gastos;
- compras;
- inventario;
- margen/rentabilidad.

### P3 — Integraciones adicionales

Fudia Concierge ya quedó cerrado para su alcance actual. En esta prioridad futura quedan únicamente integraciones distintas de Concierge o ampliaciones de producto que todavía no formen parte del alcance cerrado.

### P4 — Carta digital QR

Completar la vertical que ya figura `development` y conectarla con catálogo,
disponibilidad, mesas y pedidos.

### P5 — Delivery propio y omnicanal

- despacho;
- repartidores;
- zonas/tarifas;
- seguimiento;
- recojo;
- canales externos.

---

## CONTINUAR DESDE AQUÍ

Cuando se retome este proyecto, continuar por:

> **P0 — Facturación electrónica**

Fudia Concierge está **✅ 100% cerrado en repositorio/CI** para el alcance definido.
Su prueba con credenciales reales pertenece al gate de activación del entorno y
no bloquea el inicio de **P0 — Facturación electrónica**.

Punto de continuación de Facturación:

1. definir modelo persistente de series y comprobantes;
2. definir contrato del adaptador de proveedor homologado;
3. implementar configuración administrativa por empresa/local;
4. emitir boleta/factura desde una venta/pago real;
5. implementar historial, estados, reintentos idempotentes y auditoría;
6. integrar notas/anulaciones según soporte del proveedor;
7. cerrar OpenAPI, pruebas y flujo POS -> comprobante fiscal.

---

## Criterio para declarar un módulo 100% cerrado

Antes de mover cualquier módulo a **✅ 100% cerrado**, verificar:

- [ ] alcance funcional actual documentado y sin huecos conocidos;
- [ ] backend y persistencia reales, sin mocks;
- [ ] UI conectada a datos reales;
- [ ] sin catálogos de negocio hardcodeados en frontend;
- [ ] autorización por empresa y local donde corresponda;
- [ ] permisos definidos;
- [ ] estados loading, error y vacío;
- [ ] responsive desde 390 px;
- [ ] contrato OpenAPI actualizado;
- [ ] migraciones reversibles;
- [ ] pruebas unitarias/integración relevantes;
- [ ] lint, typecheck, tests y build verdes;
- [ ] auditoría funcional del flujo completo;
- [ ] documentación de este archivo actualizada.

## Regla de mantenimiento

Cada vez que se cierre una prioridad:

1. actualizar este documento en el mismo PR/commit;
2. mover el módulo al estado correcto;
3. registrar la nueva prioridad en **CONTINUAR DESDE AQUÍ**;
4. actualizar fecha, rama y commit base auditado;
5. no borrar el historial conceptual: resumir qué quedó cerrado y qué dependencia
   habilitó la siguiente fase.
