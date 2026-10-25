# Alcance del producto

## Propósito

Reducir errores y tiempos desde la toma del pedido hasta el cobro, mantener el
inventario trazable y emitir comprobantes electrónicos para Perú.

## Usuarios

- Administrador o propietario.
- Encargado de local.
- Cajero.
- Mozo.
- Cocina y bar.
- Almacén y compras.
- Repartidor.

## Canales

- Atención en mesa.
- Recojo en local (pedidos presenciales y encargos previos; incluye el antiguo Mostrador).
- Delivery propio.
- Pedido recibido por WhatsApp mediante integración.

## Fase inicial

- Multiempresa y multilocal.
- Catálogo de platos, modificadores, combos, precios e impuestos.
- Mesas, reservas simples y comandas.
- POS, caja, medios de pago, división de cuenta y propina.
- Pantalla de cocina (KDS) e impresión de comandas.
- Boletas, facturas, notas y envío electrónico a SUNAT mediante proveedor.
- Recetas, insumos, compras, movimientos y stock por local.
- Pedidos por WhatsApp con confirmación humana y trazabilidad.
- Dashboard, cierre de caja y reportes de ventas, costos e inventario.

## Fuera de la primera fase

- Marketplace público de restaurantes.
- Aplicación nativa iOS/Android.
- Flota logística avanzada.
- Contabilidad general completa.
- Motor propio de homologación SUNAT.

## Suscripción SaaS

El alta comercial de una empresa se realiza desde Plataforma y exige un plan
activo. El plan define moneda, precio mensual y anual, días de prueba, límites
de locales y usuarios, versión de condiciones y módulos incluidos. La
suscripción de cada empresa conserva el precio contratado, ciclo, estado
(`trial`, `active`, `past_due` o `cancelled`), periodo vigente, fecha de
renovación y aceptación de condiciones.

Los pagos de suscripción se registran separados de los pagos de pedidos. El
modelo es independiente del proveedor para permitir integrar posteriormente una
pasarela sin acoplar el dominio. Los cambios de plan aplican los módulos del
nuevo entitlement y no permiten bajar a límites inferiores al uso activo.

Plataforma consulta la cartera completa de empresas con su plan y situación de
pago (al día, por vencer, vencida, en prueba, cancelada o sin suscripción) y
desde allí abre la gestión de suscripción y cobros de cada empresa.

### Catálogo comercial inicial

FUDIA inicia con tres planes editables desde Plataforma. No son reglas
hardcodeadas del frontend: viven en `subscription_plans` y pueden cambiar sin
desplegar una nueva versión.

| Plan | Precio mensual | Precio anual | Locales | Usuarios | Enfoque |
| --- | ---: | ---: | ---: | ---: | --- |
| Emprende | PEN 89 | PEN 890 | 1 | 6 | Vender y atender en un local |
| Impulso | PEN 169 | PEN 1,690 | 3 | 20 | Control: abastecimiento, recetas y gastos |
| Escala | PEN 299 | PEN 2,990 | 10 | 60 | Impulso para operación multi-local |

Los tres incluyen 14 días de prueba.

#### Módulos por plan (homologación 2026-10-23)

Los planes son **acumulativos**: cada uno incluye todo el anterior. Solo se
venden módulos **Disponibles** (`ready`); los módulos en desarrollo o
planificados no forman parte de ningún plan hasta su cierre.

| Módulo | Clave | Emprende | Impulso | Escala |
| --- | --- | :---: | :---: | :---: |
| Punto de venta | `pos` | ✓ | ✓ | ✓ |
| Pedidos | `pedidos` | ✓ | ✓ | ✓ |
| Cocina | `cocina` | ✓ | ✓ | ✓ |
| Mesas y zonas | `mesas` | ✓ | ✓ | ✓ |
| Caja y turnos | `caja` | ✓ | ✓ | ✓ |
| Carta y productos | `productos` | ✓ | ✓ | ✓ |
| Menús y combos | `combos` | ✓ | ✓ | ✓ |
| Clientes | `clientes` | ✓ | ✓ | ✓ |
| Reportes | `reportes` | ✓ | ✓ | ✓ |
| Locales | `locales` | ✓ | ✓ | ✓ |
| Fiscal y moneda | `fiscal` | ✓ | ✓ | ✓ |
| Usuarios y roles | `usuarios` | ✓ | ✓ | ✓ |
| Fudia Concierge | `whatsapp_bot` | ✓ | ✓ | ✓ |
| Reservas | `reservas` | — | ✓ | ✓ |
| Recetas | `recetas` | — | ✓ | ✓ |
| Inventario | `inventario` | — | ✓ | ✓ |
| Kardex | `kardex` | — | ✓ | ✓ |
| Compras | `compras` | — | ✓ | ✓ |
| Costos y gastos | `costos` | — | ✓ | ✓ |

- **Emprende** vende y atiende en un solo local. Los egresos del turno se
  registran en Caja; no incluye abastecimiento ni control de costos.
- **Impulso** agrega el control del negocio: lo que entra (Compras), lo que hay
  (Inventario y Kardex), lo que cuesta (Recetas y Costos y gastos) y Reservas.
  El costeo y la rentabilidad futuros de `costos` dependen de Recetas e
  Inventario, por eso viven en el mismo plan.
- **Escala** tiene los mismos módulos que Impulso con límites de 10 locales y
  60 usuarios.
- Concierge está en todos los planes, con un límite mensual de conversaciones
  por plan, como los límites de locales y usuarios.

#### Límite mensual de Concierge (acordado e implementado 2026-10-23, migración 000091)

Una conversación es una atención completa: desde que el cliente inicia por el
QR hasta que la atención termina o la sesión expira; los mensajes y rondas
dentro de ella no suman aparte. Cuando la atención en mesa termina y el mismo
chat toma un pedido de delivery o recojo, esa nueva atención cuenta como otra
conversación. Volver a escanear el QR abre una conversación nueva. Se cuenta
por empresa y mes calendario en la zona horaria de la empresa.

| Plan | Conversaciones al mes | Promedio diario |
| --- | ---: | ---: |
| Emprende | 500 | ~17 |
| Impulso | 1,500 | ~50 |
| Escala | 4,000 | ~130 |

- Los topes son una base: el costo estimado (S/0.03–0.10 por conversación) no
  está medido todavía. Criterio: Concierge no debe superar un tercio del precio
  del plan.
- Primero se mide durante un mes (conversaciones y tokens por empresa) sin
  bloquear; con el costo real se confirman o ajustan los topes.
- Al 80 % del tope se avisa al administrador de plataforma; al 100 % no se
  abren conversaciones nuevas y nunca se corta una conversación en curso.
- Topes, modo (solo medir, avisar o bloquear), porcentaje de aviso, excepciones
  por empresa y uso mensual son configurables desde Plataforma sin desplegar, y
  solo los ve y edita el administrador de plataforma (validado en backend). Las
  empresas no ven su tope ni su consumo.

Dónde se configura cada valor (Plataforma):

| Valor | Dónde | Inicial |
| --- | --- | --- |
| Tope por plan (`max_concierge_conversations`; vacío = ilimitado) | Planes | 500 / 1,500 / 4,000; `emprende-abastecimiento` = Emprende; `legacy` ilimitado |
| Modo `measure` / `warn` / `enforce` | Configuración global › Fudia Concierge | `measure` |
| Porcentaje de aviso | Configuración global › Fudia Concierge | 80 % |
| Excepción de la empresa (tope del plan, tope propio o ilimitado) | Empresas › detalle | Tope del plan |

El modo y el porcentaje son **globales** y no por plan: la fase de medición y
el paso a bloquear son una decisión operativa de la plataforma para todas las
empresas a la vez, y lo que diferencia a los planes ya es su tope. Las
excepciones cubren los casos puntuales.

- **Solo medir** (`measure`): cuenta conversaciones y tokens; no avisa ni
  bloquea.
- **Avisar** (`warn`): al llegar al porcentaje de aviso la cartera de empresas
  y su detalle marcan «Cerca del tope» y, al 100 %, «Tope alcanzado». Solo lo
  ve el administrador de plataforma; nunca se avisa a la empresa.
- **Bloquear** (`enforce`): además del aviso, al 100 % foods-backend rechaza
  abrir conversaciones nuevas (`409 concierge_conversation_limit_reached`) y
  Concierge responde en texto plano que en este momento no puede tomar el
  pedido por WhatsApp e invita a pedir al personal en la mesa (o en el local,
  si era delivery), sin mencionar topes ni planes. Una conversación en curso
  nunca se corta. Los intentos rechazados se cuentan aparte.
- Los tokens se guardan por empresa, mes y nivel de modelo (Lite y Flash) a
  partir de los reportes de cada turno de Concierge; Prometheus conserva solo
  totales por nivel, sin etiquetas por empresa.

Las empresas suscritas a Emprende antes de la homologación pasan al plan no
vendible **Emprende con abastecimiento** (`emprende-abastecimiento`), con el
mismo precio y límites, para conservar Inventario y Compras. Al cambiar de plan
reciben el alcance del plan destino. El plan `legacy` no cambia.

Plataforma puede editar los planes, pero cualquier cambio de composición debe
actualizar esta tabla. La migración 000090 es la referencia técnica.
