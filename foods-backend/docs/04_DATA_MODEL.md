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
demás datos comerciales. Producto no almacena cantidad. La cantidad pertenece al
local y se resuelve según `products.quantity_control`.

### Control de cantidad por producto

| Valor | Significado | Fuente de la cantidad |
| --- | --- | --- |
| `none` | La venta no depende de una cantidad administrada. | No aplica |
| `portions` | Producto preparado por porciones, por ejemplo Ají de gallina. | `product_availability.portion_quantity - sold_quantity` por local y fecha |
| `inventory` | Mercadería física, por ejemplo Coca-Cola o agua mineral. | `stock_balances` del local |

Las porciones se cargan desde Disponibilidad de la carta. No existe un cupo
predeterminado en `products`: cada día/local tiene su cantidad real.

La mercadería física se repone únicamente mediante documentos de Inventario.
`inventory_items.product_id` es un vínculo interno 1:1 para conectar un
Producto vendible con la infraestructura de existencias; no constituye otro
catálogo comercial.

### Flujo de Inventario

El flujo principal de mercadería física comienza en **Inventario > Nueva entrada**:

1. Si el Producto existe, se selecciona ese mismo `ProductId` y se registra
   la nueva entrada.
2. Si no existe, el usuario crea **Nuevo producto** dentro de la entrada.
3. Producto, vínculo interno de inventario, saldo, documento de entrada y
   movimiento de Kárdex se crean en una sola transacción.
4. Si cualquier paso falla, la transacción hace rollback y no queda un Producto
   huérfano ni un saldo parcial.
5. Las reposiciones posteriores usan siempre el mismo Producto.

La pantalla de Productos permanece dedicada al catálogo comercial: alta de
platos, nombre, precio, categoría, imagen, estado y clasificación de control de
cantidad. No registra entradas ni modifica stock físico.

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
| `products` | Catálogo comercial único y clasificación `quantity_control`. |
| `product_availability` | Porciones, vendidos y override manual por local/día. |
| `inventory_items` | Registro interno de inventario; `product_id` vincula 1:1 mercadería vendible. |
| `stock_balances` | Saldo físico actual por local e item interno. |
| `inventory_entries` | Documento auditable de cada entrada física. |
| `stock_movements` | Kárdex: entradas, ventas, reversas y ajustes, con saldo resultante. |
| `menu_combos` | Identifica productos compuestos vendidos como menú o combo. |
| `menu_combo_groups` | Grupos de elección del combo. |
| `menu_combo_options` | Productos existentes usados como alternativas del combo. |

`stock_balances` se modifica dentro de la misma transacción que registra el
`stock_movements` correspondiente. El saldo tiene una restricción de base de
datos que impide valores negativos.

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
