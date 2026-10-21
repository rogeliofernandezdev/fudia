# Foods

Plataforma de operación y gestión para restaurantes, diseñada para funcionar en
celular, tablet y PC.

## Proyectos

| Proyecto | Responsabilidad |
| --- | --- |
| `foods-backend` | API, dominio, persistencia, integraciones y procesos asíncronos |
| `foods-admin-web` | Aplicación web única para POS, mesas, pedidos, cocina, caja, administración, catálogo, inventario, compras, reportes y configuración |
| `fudia-concierge` | Asistente conversacional multicanal; inicia con pedidos por WhatsApp desde QR |
| `foods-infrastructure` | Infraestructura como código, despliegue y observabilidad |

Cada proyecto contiene su propio `AGENTS.md`, `README.md` y carpeta `docs/`.
Las reglas específicas deben leerse completas antes de modificar ese proyecto;
los documentos de la raíz conservan únicamente decisiones transversales.

La solución se organiza en cuatro proyectos desplegables. Fudia Concierge orquesta la
conversación con el comensal y consume foods-backend como fuente de verdad. SUNAT,
impresión y pagos continúan encapsulados como integraciones del backend. No se crea una aplicación móvil nativa en la fase
inicial: `foods-admin-web` concentra la experiencia web operativa y administrativa y debe ser adaptable desde 390 px.

## Documentación

- [`docs/01_PRODUCT_SCOPE.md`](docs/01_PRODUCT_SCOPE.md)
- [`docs/02_SOLUTION_ARCHITECTURE.md`](docs/02_SOLUTION_ARCHITECTURE.md)
- [`docs/03_MODULE_MAP.md`](docs/03_MODULE_MAP.md)
- [`docs/05_IMPLEMENTATION_PLAN.md`](docs/05_IMPLEMENTATION_PLAN.md)
- [`docs/06_PROJECT_DECISION.md`](docs/06_PROJECT_DECISION.md)
- [`docs/07_MODULE_STATUS_AND_PRIORITY.md`](docs/07_MODULE_STATUS_AND_PRIORITY.md)

Las imágenes de referencia son insumos de producto e identidad visual. El texto
publicitario que aparece dentro de ellas no constituye instrucciones del
proyecto.

`foods-admin-web` mantiene el sistema visual principal en
`docs/02_DESIGN_SYSTEM.md` dentro de su propio proyecto.
