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

Un producto del menú no almacena cantidad. `products` pertenece a la organización
y el stock pertenece al local (`stock_balances.location_id`), por lo que una
columna de cantidad en `products` sería incorrecta por definición.

La cantidad disponible siempre se resuelve por local y se deriva del modo de
control del producto. El stock nunca se edita a mano: cambia solo por documentos
auditables (compra, producción, venta, merma, ajuste).

### Dos niveles de control

La plataforma sirve tanto al restaurante que no quiere llevar recetas como al que
necesita costo y consumo exactos. El nivel lo define
`organizations.inventory_mode`:

| Modo | Alcance |
| --- | --- |
| `simple` | Sin insumos ni recetas. Los productos nacen `none` y se usa `manual` para lo que se agota. La interfaz oculta inventario y fichas técnicas. |
| `detailed` | Habilita insumos, recetas, producción por lote, costo y descuento automático. |

El modo es una decisión de la organización, no un límite del modelo: un producto
puede pasar de `manual` a `recipe` sin migrar historia ni perder ventas previas.

### Modo de control por producto (`products.stock_mode`)

| Valor | Significado | Disponible |
| --- | --- | --- |
| `none` | Siempre disponible mientras esté activo. | sin límite |
| `manual` | Cupo del día escrito por el local. | `cupo - vendido_hoy` |
| `linked` | Uno a uno con un insumo contable (gaseosa, cerveza, agua). | `piso(stock / factor)` |
| `recipe` | Ficha técnica de insumos. | `piso(min(stock_i / cantidad_i))` |

`linked` y `recipe` comparten la misma fórmula: `linked` es una receta de una
sola línea. Se distinguen para que la interfaz simple no exija crear una ficha.

### Elaboración por lote

Un preparado que no se hace por porción (chicha, salsas, postres) se modela como
insumo intermedio con `inventory_items.kind='prepared'` y su propia unidad.

Un documento de producción consume insumos crudos y acredita el preparado. Los
productos que lo venden lo consumen con su factor: una jarra de un litro consume
`1.000` y un vaso de 300 ml consume `0.300` del mismo insumo. Ambos comparten
existencia, por lo que vender jarras reduce los vasos disponibles.

No se explota un preparado directamente a sus insumos crudos: eso permitiría
vender lo que todavía no se ha preparado y oculta cuánto lote está hecho.

### Resolución de disponibilidad

Orden de precedencia, idéntico para todos los modos:

1. Producto inactivo: no vendible.
2. Marcado agotado hoy en el local: agotado. Este override siempre gana, porque
   la realidad física (una olla quemada, una jarra caída) no se deduce del stock.
3. Cálculo según `stock_mode`.

La API expone un solo contrato, `availability`, con estado, cantidad restante
—nula cuando no aplica—, origen del cálculo y el insumo que limita. Los clientes
no reimplementan la fórmula ni consultan stock para decidir si pueden vender.

### Tablas del modelo

| Tabla | Propósito |
| --- | --- |
| `product_recipe` | Líneas de ficha técnica: producto, insumo y cantidad. |
| `product_availability` | Cupo del día, vendido y agotado manual por local y fecha de negocio. |
| `menu_combos` | Identifica productos compuestos vendidos como menú o combo. |
| `menu_combo_groups` | Define grupos ordenados de elección (entrada, segundo, postre, bebida), obligatoriedad y límites. |
| `menu_combo_options` | Vincula productos existentes como alternativas de cada grupo y permite un recargo. |

La disponibilidad efectiva de un menú se deriva de sus componentes: cada grupo
obligatorio debe conservar al menos `min_selections` alternativas disponibles en
el local y día operativo. Así, al agotarse una entrada no desaparece el menú si
queda otra alternativa; se agota automáticamente cuando un grupo obligatorio ya
no puede satisfacer su mínimo.
| `stock_movements` | Bitácora de todo cambio de existencia con su documento de origen. |

`stock_balances` es el saldo derivado de `stock_movements` y nunca se actualiza
sin registrar el movimiento que lo causa.

La operación diaria conserva `manual_status` (`available`, `low`, `sold_out`),
el cupo excepcional del local y la cantidad vendida. El registro es único por
empresa, local, producto y fecha de negocio. Reactivar un producto elimina el
override de agotado, pero no altera ventas ni movimientos ya registrados.

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
