# Estado de módulos y prioridad de implementación

> Documento vivo para retomar el desarrollo de FUDIA sin perder el punto de
> continuidad. Debe actualizarse cuando un módulo cambia de estado, se cierra
> una vertical o cambia la prioridad del roadmap.

## Última revisión

- Fecha: **2026-09-24**
- Rama revisada: `feat/redesign-nueva-comanda`
- Commit base funcional auditado: `97835260cec6a971b0305eecc085d8bce7378fc7`
- CI del commit base:
  - `foods-backend`: **success**
  - `foods-admin-web`: **success**
  - `foods-operations-web`: **success**
  - `fudia-concierge`: **success**
- Trabajo activo por decisión de producto: **Fudia Concierge**
- Bloques cerrados de Concierge: **Fundación + contrato Fudia + QR/configuración**
- Siguiente bloque concreto: **conversación avanzada — combos**

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
| Fudia Concierge | `whatsapp_bot` | 🔵 En desarrollo | Fundación, contrato server-to-server, menú/pedido por QR, configuración por local, deeplink WhatsApp, confirmación protegida, idempotencia y entrada a KDS están implementados. Faltan combos/modificadores, comanda existente, handoff y observabilidad. |

---

## Trabajo activo — Fudia Concierge

Por decisión de producto se adelantó la vertical conversacional que originalmente
estaba dentro de P3. Esto no elimina la prioridad estratégica de Facturación;
simplemente registra el frente que se está implementando ahora.

### Cerrado en la primera etapa

- quinto proyecto desplegable `fudia-concierge/`;
- FastAPI, LangGraph, OpenAI Responses API y tools;
- webhook Meta con deduplicación y validación de firma;
- sesión/carrito efímero en Redis;
- confirmación explícita protegida también por código;
- contrato `/v1/integrations/concierge/{token}/...` protegido con credencial server-to-server;
- resolución de empresa/local/mesa exclusivamente desde QR;
- menú operativo consultado desde foods-backend;
- precio, disponibilidad y stock revalidados por foods-backend;
- pedido real con canal `whatsapp`;
- idempotencia por `conversationId`;
- auditoría sin suplantar a un usuario interno;
- llegada del pedido confirmado a Pedidos/KDS;
- prueba integrada QR -> menú -> pedido -> KDS;
- CI propio de Concierge.

### Cerrado en la segunda etapa — QR y configuración

- configuración persistente por empresa/local;
- número público de WhatsApp en formato internacional;
- activación/desactivación administrativa en `/whatsapp-bot`;
- contrato de integración bloqueado cuando el local desactiva Concierge;
- QR público expone Concierge únicamente cuando está activo;
- botón **Pedir por WhatsApp** en `/mesa/[qr]`;
- deeplink con mensaje `FUDIA:<qr_token>`;
- inicio de sesión rechazado si el QR pertenece a un local con Concierge desactivado;
- OpenAPI actualizado y migración reversible;
- CI verde de backend, Admin Web, Operations Web y Concierge.

La prueba automatizada cubre QR -> menú -> pedido -> KDS dentro de Fudia. La validación contra la infraestructura real de Meta/WhatsApp queda como prueba de despliegue, ya que requiere credenciales y webhook externos.

### Pendiente inmediato

- combos conversacionales con grupos/opciones;
- modificadores de productos simples;
- agregar a una comanda existente de la mesa;
- handoff humano;
- observabilidad y endurecimiento operativo.

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

### P3 — Integraciones y WhatsApp

Fudia Concierge se adelantó como trabajo activo. Su fundación y contrato de pedidos ya están implementados. En esta prioridad quedarán las integraciones adicionales, configuración completa, handoff humano y endurecimiento multicanal.

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

> **Fudia Concierge — Bloque 4: combos conversacionales**

Orden inmediato:

1. exponer al servicio Concierge la configuración real de grupos y opciones de un combo;
2. añadir una tool para consultar las opciones requeridas;
3. permitir que GPT seleccione únicamente IDs ofrecidos por Fudia;
4. enviar las selecciones al pedido sin aceptar precios generados por el modelo;
5. dejar que `foods-backend` revalide mínimos, máximos, disponibilidad y recargos;
6. probar conversación de combo -> confirmación -> pedido -> KDS.

Después de completar combos/modificadores y el handoff operativo, reevaluar el retorno a **P0 — Facturación electrónica**, que continúa siendo el hueco principal del circuito fiscal de venta.

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
