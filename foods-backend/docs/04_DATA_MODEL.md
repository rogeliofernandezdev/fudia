# Modelo de datos

Toda entidad persistida usa id uuid PRIMARY KEY DEFAULT gen_random_uuid().
PostgreSQL genera el identificador. Toda entidad de negocio lleva organization_id;
las operaciones de local llevan también location_id.

Fechas se guardan en UTC; importes monetarios usan numeric, nunca flotantes.

Cada organización configura la moneda ISO 4217, posición del símbolo, precisión,
nombre y tasa del impuesto general, y si el precio publicado ya lo incluye.
Los documentos conservan una copia de estos valores al emitirse para evitar que
un cambio de configuración altere el historial.

El catálogo seleccionable se genera desde «List One: Current Currency & Funds»
de SIX, agencia oficial de mantenimiento de ISO 4217. Se deduplica por código y
excluye metales preciosos, unidades de fondos, códigos de prueba y `XXX`; no se
mantienen subconjuntos manuales en los frontends.

## Disponibilidad de productos

`products` es el catálogo único de todo lo que se vende. Platos, bebidas,
mercadería física, menús y opciones de combo se identifican por `ProductId`.
Pedido y detalle de pedido conservan siempre esa referencia; no existe un segundo
catálogo vendible dentro de Inventario.

Producto describe **qué se vende**: nombre, precio, categoría, imagen, estado y
demás datos comerciales. Además separa dos clasificaciones independientes:
`products.product_type` describe si el producto es preparado o mercadería de
reventa, mientras `products.quantity_control` describe de dónde sale su
disponibilidad. La categoría sigue siendo comercial y no determina ninguna de
esas dos dimensiones. Producto no almacena cantidad; la cantidad pertenece al
local.

### Tipo de producto

| Valor | Significado |
| --- | --- |
| `prepared` | Plato, bebida u otro producto preparado por el restaurante. |
| `retail` | Mercadería vendible recibida físicamente, como gaseosas, agua o snacks. |

El tipo no se infiere por el nombre de la categoría. Cada categoría declara
`product_scope`: `prepared`, `retail` o `both`. Así «Bebidas» puede
aceptar una limonada `prepared` y una gaseosa `retail` usando `both`,
mientras «Segundos» puede permanecer solo en `prepared`. Inventario solicita
únicamente categorías activas compatibles con `retail`; no filtra por nombre.
El alta comercial normal usa `prepared` por defecto; **Inventario > Nuevo
producto vendible** fuerza `retail` en backend para que ese atajo nunca cree
un plato preparado.

### Control de cantidad por producto

| Valor | Significado | Fuente de la cantidad |
| --- | --- | --- |
| `none` | La venta no depende de una cantidad administrada. | No aplica |
| `portions` | Producto preparado por porciones, por ejemplo Ají de gallina. | `product_availability.portion_quantity - sold_quantity` por local y fecha |
| `inventory` | Mercadería física, por ejemplo Coca-Cola o agua mineral. | `stock_balances` del local |

Las porciones se cargan desde Disponibilidad de la carta. No existe un cupo
predeterminado en `products`: cada día/local tiene su cantidad real. El cupo
`portion_quantity` nunca puede reducirse por debajo de `sold_quantity`; esta
invariante se valida en backend bajo bloqueo transaccional para no competir con
la creación, edición o reversa de pedidos.

La mercadería física y los insumos de producción se reponen únicamente mediante
documentos de Inventario. `inventory_items` es el catálogo físico: puede
representar un insumo interno como carne, papa o aceite sin `product_id`, o una
mercadería vendible como una gaseosa enlazada 1:1 a `products`. El vínculo
`inventory_items.product_id` es opcional y no constituye otro catálogo comercial.

Cada `inventory_item` define una **unidad base de stock** (por ejemplo botella,
lata, unidad, kg o litro). `inventory_presentations` define formas reutilizables
de recibir esa mercadería: unidad base (factor 1), paquete o caja con un factor
de conversión. Una entrada de 5 cajas x 12 de un producto cuya unidad base es
botella aumenta el saldo y el Kárdex en 60 botellas. El documento de entrada
conserva la cantidad recibida, la presentación y el factor utilizados.

### Flujo de Inventario

El flujo principal de mercadería física comienza en **Inventario > Nueva entrada**:

1. Si el artículo de inventario existe, se selecciona su `InventoryItemId` y
   se registra la nueva entrada.
2. Si la mercadería se vende directamente, **Nuevo producto vendible** crea
   `Product` + `inventory_item`, exige una categoría cuyo `product_scope`
   admita `retail`, exige precio de venta y registra
   `product_type='retail'` + `quantity_control='inventory'` de forma automática.
3. Si es un ingrediente interno, **Nuevo insumo** crea únicamente
   `inventory_item`; no crea `Product` ni exige precio de venta.
4. Se resuelve la presentación de ingreso. La unidad base siempre existe; una
   presentación nueva como paquete x 6 o caja x 12 queda disponible para futuras
   entradas del mismo producto.
5. Artículo de inventario, posible Producto vinculado, presentación, saldo,
   documento de entrada y movimiento de Kárdex se guardan en una sola transacción.
6. Si cualquier paso falla, la transacción hace rollback y no queda un artículo,
   presentación o saldo parcial.
7. Las reposiciones posteriores usan siempre el mismo `InventoryItemId`.

El selector «Producto existente» muestra únicamente productos activos con
`quantity_control='inventory'`. Los productos con `none` o `portions` no son
elegibles para una entrada y registrar stock nunca cambia implícitamente el modo
de control de un plato o producto preparado. La mercadería física nueva se crea
desde **Inventario > Nueva entrada > Nuevo producto físico**.

La pantalla de Productos permanece dedicada al catálogo comercial: alta de
platos y edición de nombre, precio, categoría, imagen y estado. Para productos
físicos existentes muestra su condición de Inventario físico, pero las entradas,
presentaciones y existencias se administran desde Inventario.

### Venta y concurrencia

Al crear un pedido, el backend calcula uso de cantidades únicamente por
`ProductId`. Para productos directos usa el `product_id` de la línea;
para combos usa los `ProductId` seleccionados en sus opciones.

- `portions`: bloquea el registro diario y aumenta `sold_quantity`.
- `inventory`: bloquea el saldo físico con `FOR UPDATE`, valida que la
  existencia alcance y descuenta el stock.
- editar un pedido aplica solo el delta entre la versión anterior y la nueva;
- cancelar un pedido revierte las cantidades consumidas;
- nunca se confirma una operación que produzca stock negativo.

La actualización de pedido, el descuento/restauración y el movimiento de Kárdex
comparten la misma transacción. Esto evita sobreventa cuando dos pedidos intentan
consumir simultáneamente el último stock.

### Resolución de disponibilidad

Orden de precedencia:

1. Producto inactivo: no vendible.
2. Override manual `sold_out` del local/día: agotado.
3. Horario o vigencia comercial fuera de rango: no disponible.
4. `none`: disponible.
5. `portions`: disponible si quedan porciones.
6. `inventory`: disponible si el saldo físico del local es mayor que cero.
7. Un combo exige suficientes opciones disponibles en cada grupo obligatorio.

Los clientes consumen este resultado; no duplican la fórmula ni calculan stock
por su cuenta.

### Tablas del modelo

| Tabla | Propósito |
| --- | --- |
| `products` | Catálogo comercial único; separa `product_type` de `quantity_control`. |
| `product_availability` | Porciones, vendidos y override manual por local/día. |
| `inventory_items` | Catálogo físico: insumos internos o mercadería vendible; `product_id` es opcional y `unit` define la unidad base. |
| `inventory_presentations` | Presentaciones reutilizables de ingreso y su factor hacia la unidad base. |
| `stock_balances` | Saldo físico actual por local e item interno, siempre expresado en unidad base. |
| `inventory_entries` | Documento auditable: cantidad recibida, presentación, factor y equivalencia en unidad base. |
| `stock_movements` | Kárdex: entradas, ventas, reversas y ajustes, con saldo resultante. |
| `menu_combos` | Identifica productos compuestos vendidos como menú o combo. |
| `menu_combo_groups` | Grupos de elección del combo. |
| `menu_combo_options` | Productos existentes usados como alternativas del combo. |

`stock_balances` se modifica dentro de la misma transacción que registra el
`stock_movements` correspondiente. El saldo tiene una restricción de base de
datos que impide valores negativos.

## Configuración inicial de una empresa

El onboarding de una organización crea en una sola transacción la empresa, su perfil fiscal por defecto, el local principal, el administrador, los roles predefinidos, la suscripción y los módulos permitidos por el plan. Además copia desde plantillas persistidas en base de datos los medios de pago, categorías iniciales de gasto, una zona `Principal` y una `Caja principal` para el primer local.

Estas filas son configuración inicial editable, no datos comerciales de ejemplo. No se crean productos, proveedores, recetas, mesas, clientes ni movimientos ficticios.

## Medios de pago

`payment_methods` es el catálogo de medios de pago por empresa y constituye la única fuente de verdad para Cobros y Gastos. Cada fila define código estable, nombre visible, estado, disponibilidad para ventas o gastos y si el medio representa movimiento físico de efectivo. `payments` y `expenses` referencian el catálogo mediante clave foránea compuesta `(organization_id, code)`.

Los valores iniciales se siembran durante la migración, pero la lógica de aplicación no contiene una lista cerrada de medios de pago.

## Gastos operativos

Los gastos operativos son documentos del local y no reemplazan la contabilidad general.
`expense_categories` pertenece a la empresa y define el catálogo reutilizable; `expenses`
pertenece además al local activo y conserva fecha de negocio, descripción, importe exacto
`numeric`, medio de pago, referencia, notas y usuario creador.

Un gasto confirmado no se edita ni elimina. Una corrección se realiza mediante anulación
auditable, conservando usuario, fecha y motivo. El registro del gasto no modifica
implícitamente un turno de caja: Caja y turnos mantiene su propia trazabilidad de efectivo.

## Empresa, locales y configuración financiera

La jerarquía de autoridad es `organization` → `organization_fiscal_profiles` →
`locations`. La empresa es el tenant y propietario de la configuración; cada
perfil representa una jurisdicción fiscal de esa empresa, y cada local referencia
exactamente un perfil. Así una empresa peruana empieza con PE/PEN/IGV, pero puede
abrir después un local en otra jurisdicción sin duplicar la empresa ni alterar el
histórico de sus locales peruanos.

### Perfiles fiscales por empresa y país

| Campo | Tipo | Propósito |
| --- | --- | --- |
| `organization_id` | `uuid` | Empresa propietaria; forma parte de toda consulta y restricción. |
| `country_code` | `char(2)` | País según ISO 3166-1 alfa-2; es único dentro de la empresa. |
| `currency` | `char(3)` | Moneda funcional ISO 4217 del perfil. |
| `currency_symbol` | `text` | Símbolo de presentación. |
| `currency_position` | `text` | `before` o `after`. |
| `currency_decimals` | `smallint` | Precisión de presentación. |
| `tax_name` | `text` | Nombre del impuesto general de la jurisdicción. |
| `tax_rate` | `numeric(7,6)` | Tasa exacta; `0.180000` representa 18%. |
| `tax_included` | `boolean` | Indica si el precio publicado incluye el impuesto. |
| `is_default` | `boolean` | Solo puede existir un perfil predeterminado por empresa. |

Una organización nueva recibe automáticamente el perfil predeterminado Perú
(`PE`), sol peruano (`PEN`) e IGV (`18%`). Un local nuevo hereda ese perfil, pero
puede asignarse explícitamente a otro perfil de la misma empresa. La llave foránea
compuesta impide asociar un local con la configuración de otro tenant.

### Alta de una empresa

El alta SaaS se ejecuta mediante `POST /v1/platform/organizations` y exige un
usuario con `users.platform_admin=true`. La operación crea dentro de una sola
transacción la empresa, su perfil fiscal predeterminado, el primer local, el
administrador del tenant, el rol y su asignación. Un administrador común de una
empresa nunca puede crear ni consultar otros tenants.

### Tipos de cambio

`organization_exchange_rates` conserva tasas por empresa, par de monedas y fecha
efectiva. Una tasa nunca se sobrescribe: una actualización crea una nueva fila.
Registra origen manual o proveedor, referencia externa y usuario creador. Ventas,
pagos y documentos que conviertan moneda guardan una copia de la tasa aplicada y
su identificador para mantener trazabilidad.

### Reglas

- El código `currency` sigue ISO 4217; el símbolo `currency_symbol` es libre
  porque algunos países usan abreviaturas distintas al símbolo oficial.
- La configuración pertenece a la empresa y se segmenta por país. El local solo
  selecciona uno de los perfiles de su propia empresa; no redefine campos sueltos.
- El país predeterminado es Perú (`PE`) y su moneda inicial es PEN; cambiar el
  perfil predeterminado sugiere la moneda vigente sin sobrescribir automáticamente
  la tasa fiscal.
- `tax_rate` se guarda como `numeric(7,6)` para soportar tasas como 0.180000 (18%)
  o 0.080000 (8%) con precisión exacta. Nunca se usa `float`.
- Cuando `tax_included` es `true`, el precio del producto ya tiene el impuesto
  incluido y el sistema lo desglosa al facturar. Cuando es `false`, el impuesto
  se calcula y suma al subtotal.
- El frontend obtiene esta configuración al iniciar sesión y la expone a
  todas las pantallas mediante un contexto/hook. Ningún componente hardcodea
  el símbolo `S/` ni el porcentaje `18%`.
- Los montos se siguen guardando en `numeric` sin redondear; el redondeo
  con `currency_decimals` es solo de presentación.
- Los catálogos de países y monedas son datos de referencia del backend. Los tipos
  de cambio son datos temporales de negocio y siempre están acotados por empresa.

## Planes y suscripciones SaaS

`subscription_plans` es el catálogo comercial administrado por Plataforma.
Cada plan define precio mensual/anual, moneda, prueba gratuita, límites de
locales y usuarios, módulos habilitados y versión de condiciones. Los precios
del catálogo no sustituyen el precio ya contratado por una empresa.

`organization_subscriptions` conserva una instantánea contractual por empresa:
plan, ciclo, precio, moneda, estado, periodo vigente, renovación automática,
prueba, cancelación y versión/fecha de aceptación de condiciones. Un cambio de
plan o ciclo toma el precio vigente del plan; cambios administrativos que no
alteran plan ni ciclo conservan el precio y las fechas del periodo existente.

`subscription_payments` registra cobros SaaS independientemente del proveedor
de pago. Un pago confirmado reactiva la suscripción, avanza el periodo desde el
fin del periodo vigente cuando corresponde y vuelve a aplicar los módulos del
plan.

Los límites `max_locations` y `max_users` se validan en backend dentro de la
misma transacción que crea el recurso. Un downgrade se rechaza si la empresa ya
supera los límites del plan destino. Los módulos activos de
`organization_modules` se sincronizan desde `subscription_plans.module_keys`;
solo módulos marcados como disponibles por la plataforma pueden activarse.
