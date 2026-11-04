# Facturación electrónica por país

Estado: **implementado con motor SUNAT propio** (2026-11-01, migración 000098).
Emite boletas y facturas de Perú desde el cobro: genera el XML UBL 2.1, lo
firma con el certificado de la empresa, lo envía por SOAP a SUNAT y guarda el
XML firmado y el CDR. Probado contra SUNAT beta con CDR aceptado (factura,
boleta con DNI, boleta anónima e IGV reducido MYPE). Falta la nota de crédito, el
resumen diario y la comunicación de baja, la emisión posterior desde Ventas y
la cola de Cloud Tasks. La representación impresa con QR se entrega desde el
cobro (2026-11-04). Este documento es
la referencia del módulo `einvoicing` (contexto EInvoicing; clave comercial
`facturacion`). Antes de escribir código, aplicar además `04_DATA_MODEL.md`,
`05_DATABASE_GUIDELINES.md`, `06_API_GUIDELINES.md` y `07_SECURITY.md`.

## Implementación actual

| Pieza | Dónde |
| --- | --- |
| Tablas | `einvoicing_profile_settings` (proveedor y ambiente por perfil fiscal; la columna `enabled` se eliminó en la migración 000105), `einvoicing_location_settings` (comprobante sugerido al cobrar), `einvoicing_series` (serie y último número por local y tipo), `fiscal_documents` (cabecera, estado y valor resumen de la firma; la fila PENDING es el outbox), `fiscal_document_lines`, `fiscal_document_taxes`, `einvoicing_credentials` (credenciales cifradas por perfil fiscal y proveedor, o de plataforma), `fiscal_document_files` (XML firmado y CDR). |
| Dominio | `domain.Compute`: desglose exacto desde el monto cobrado (precios con impuesto), por línea y por componente del impuesto vigente; absorbe hasta 5 céntimos de redondeo en la última línea y rechaza diferencias mayores (`fiscal_total_mismatch`). |
| Perú | `jurisdictions/pe`: boleta `03` y factura `01`; factura con RUC válido (dígito verificador) y razón social; boleta anónima hasta S/ 700, identificada con DNI, CE, RUC o pasaporte; series `B###` y `F###` sugeridas por posición del local. |
| Proveedor SUNAT | `providers/sunat` (motor propio): XML UBL 2.1 (`ubl.go`), firma XMLDSig enveloped RSA-SHA1 con C14N inclusivo (`sign.go`), ZIP + `sendBill` SOAP con WS-Security y lectura del CDR (`soap.go`), importe en letras (`words.go`). Beta usa el usuario público de SUNAT y un certificado autofirmado en memoria; producción exige usuario SOL secundario, clave SOL y certificado `.pfx` de la empresa. |
| Resultado SUNAT | CDR con código `0` → ACCEPTED; con notas → OBSERVED; fault 2000–3999 o 1033 → REJECTED (definitivo, guarda el XML); cualquier otro fault, HTTP sin XML (beta responde 401 si se envía muy rápido) o red → ERROR con reintento. |
| Proveedor de pruebas | `providers/fake` (solo ambiente de pruebas, sin red): acepta todo; un nombre con «[rechazar]» devuelve rechazo y «[error]» una falla temporal. |
| Credenciales | Cada proveedor declara sus campos (`CredentialFields`, tipos texto, secreto o archivo en base64) y los valida (`ValidateCredentials`, que devuelve solo metadatos visibles: usuario, titular y vencimiento del certificado). Se cifran con AES-256-GCM (`internal/platform/secretbox`) y datos asociados `einvoicing|organización|perfil|proveedor`, con versión de llave para rotar. El API nunca las devuelve. |
| Emisor | `ValidateIssuer` de la jurisdicción (Perú: RUC con dígito verificador y razón social de Mi restaurante) bloquea la emisión mientras sea inválido; la dirección es la del local. |
| Archivos | `GET /v1/admin/fiscal-documents/{id}/files/{xml|cdr}` del local de la sesión; se guardan en PostgreSQL (`bytea`), pocos KB por comprobante. |
| Representación impresa | `Jurisdiction.Printout` arma el contenido legal (Perú: «BOLETA DE VENTA ELECTRÓNICA» / «FACTURA ELECTRÓNICA», «SON: … SOLES», QR `RUC\|tipo\|serie\|número\|IGV\|total\|fecha\|tipo doc\|número doc\|valor resumen\|` y la leyenda «Representación impresa de …»). `GET /v1/admin/fiscal-documents/{id}` lo devuelve en `printout` desde que el documento está firmado, aunque SUNAT aún no responda; es `null` antes de firmar y para rechazados o anulados. En ambiente de pruebas lleva `notice` «sin validez tributaria». El XML firmado y el valor resumen se guardan también cuando el envío falla, así el reintento conserva la misma firma (el XML es determinista). La web solo maqueta el ticket de 80 mm y dibuja el QR. |
| Reimpresión | `contracts.Issuer.OrderDocument` devuelve el comprobante más reciente del pedido en el local de la sesión; `GET /v1/admin/pos/orders/{id}` (detalle de cobro y de Ventas) lo expone en `fiscalDocument`. La web reimprime lo que se entregó: el comprobante si tiene valor fiscal; el ticket de pago si no hubo comprobante o si fue rechazado o anulado, con un aviso. |
| Cobro | `PaymentInput.fiscal` y `PaymentBatchInput.fiscal`; Billing llama a `einvoicing/contracts.Issuer.Request` dentro de su transacción solo en el pago que salda la cuenta. Un error revierte el pago completo. Tras el commit, `Dispatch` envía en segundo plano. |
| Envío | `dispatch.InProcess` en el API; `POST /internal/einvoicing/documents/{id}/process` y `POST /internal/einvoicing/sweep` con `X-Fudia-Worker-Key` (`FUDIA_EINVOICING_WORKER_KEY`) para Cloud Tasks y Cloud Scheduler. Reintentos: 1, 2, 4… hasta 60 minutos; un SENDING abandonado 5 minutos se retoma. |
| Habilitación | `subscriptions.Access.ModuleEntitled`, con la misma regla que la sesión (`domain.ModuleEnabled`): `facturacion` es un módulo disponible (2026-11-01) y exige estar activo y en el plan (Impulso y Escala). La migración 000099 lo agrega a esos planes y lo activa para sus suscriptores; emitir sigue apagado hasta que la empresa activa su perfil fiscal. |
| API | `GET /v1/admin/einvoicing/settings`, `PUT /v1/admin/einvoicing/profiles/{id}`, `PUT /v1/admin/einvoicing/profiles/{id}/credentials`, `PUT /v1/admin/einvoicing/locations/{id}`, `GET /v1/admin/einvoicing/checkout-options`, `GET /v1/admin/fiscal-documents[/{id}]`, `GET /v1/admin/fiscal-documents/{id}/files/{kind}`, `POST /v1/admin/fiscal-documents/{id}/retry`. |
| Web | Módulo `src/modules/einvoicing`: sección «Comprobante» del cobro del POS, Configuración › Facturación (estado derivado «Lista para emitir» / «Falta configurar», credenciales armadas con los campos del proveedor, series) y Comprobantes (detalle con descarga de XML y CDR). |
| Variables | `FUDIA_EINVOICING_MASTER_KEY` (32 bytes en base64, desde Secret Manager), `FUDIA_EINVOICING_KEY_VERSION` (por defecto 1), `FUDIA_EINVOICING_PREVIOUS_KEYS` (`versión:base64,…` para leer lo cifrado con llaves anteriores) y `FUDIA_EINVOICING_WORKER_KEY`. Sin llave maestra no se guardan credenciales ni se envía a producción. |

Pruebas: `domain/document_test.go`, `jurisdictions/pe/pe_test.go`,
`providers/sunat/sunat_test.go` (XML, verificación de la firma, CDR, faults y
errores HTTP con un servidor SOAP local), `platform/secretbox` y
`einvoicing_integration_test.go` (cobro → comprobante → proveedor, rollback,
reintento, rechazo, IGV reducido MYPE, aislamiento por empresa, series en uso y
credenciales cifradas usadas en el envío con descarga de XML y CDR). La prueba
contra SUNAT beta real es opcional: `SUNAT_BETA=1 go test ./internal/modules/einvoicing/providers/sunat -run Beta`.

## Principios

1. **Motor propio detrás de un puerto.** Decisión del 2026-11-01 (reemplaza la
   anterior «no construir motor propio»): Fudia genera, firma y envía el XML a
   SUNAT con el certificado de cada empresa, sin costo por comprobante. Ese motor
   es un proveedor más (`providers/sunat`): un OSE/PSE o un servicio como
   llama.pe puede agregarse después como otro adaptador sin cambiar reglas,
   cobro ni pantallas.
2. **El país lo define el perfil fiscal.** La jurisdicción de un comprobante es el
   `country_code` del `organization_fiscal_profiles` al que pertenece el local,
   igual que moneda, impuesto, plantillas de medios de pago y propinas iniciales.
   No existe una preferencia de país independiente para facturación.
3. **Núcleo neutro, reglas por país.** El dominio no conoce «boleta», «RUC» ni
   «SUNAT». Lo específico de cada país vive en una jurisdicción registrada por
   código ISO 3166-1 alfa-2.
4. **Reglas y transporte separados.** La jurisdicción decide qué es válido; el
   proveedor solo transporta. Un país puede tener varios proveedores y cambiar de
   proveedor no modifica las reglas.
5. **El cobro nunca espera a la autoridad tributaria.** El pago se confirma en su
   transacción; la emisión ocurre después mediante outbox y worker idempotente
   (ver «Ejecución asíncrona»).
6. **Sin catálogos fiscales en frontend.** Tipos de comprobante, tipos de
   documento del cliente, motivos de nota y campos obligatorios llegan desde el
   API por jurisdicción (regla 7 de `AGENTS.md`).
7. **Opcional e independiente.** La emisión es una capacidad adicional del
   cobro, no una condición para cobrar. Sin el módulo, desactivado o sin
   pedido de comprobante, el cobro funciona exactamente como hoy.

## Activación y opcionalidad

Decisión del 2026-11-01. Tres niveles, de mayor a menor alcance:

| Nivel | Quién decide | Efecto cuando está apagado |
| --- | --- | --- |
| Plan | Plataforma. El módulo `facturacion` se incluye desde **Impulso** (también en Escala por ser acumulativo); Emprende no lo incluye. | El cobro no muestra ninguna opción de comprobante electrónico. |
| Empresa | El administrador activa o desactiva el módulo (`organization_modules.active`). Es la **única** activación (2026-11-03): no existe un interruptor por perfil fiscal. Con el módulo activo, un local emite en cuanto tiene emisor válido, credenciales del proveedor para el ambiente y series de cada tipo; si falta algo, `checkout-options` responde `enabled=false` con `issue` y el cobro lo explica. | Igual que sin plan; se conservan configuración e historial y puede reactivarse. |
| Venta | El cajero, según lo que pida el cliente: **Sin comprobante**, **Boleta** o **Factura** (tipos del catálogo del país). | «Sin comprobante» no crea documento, no consume correlativo ni llama al proveedor; queda el ticket operativo actual. |

- El valor inicial del selector en el cobro se configura por local. El valor
  inicial de producto es **Sin comprobante**, para los restaurantes que primero
  preguntan al cliente.
- Si el cliente lo pide después, se puede emitir desde el detalle de la venta
  en Ventas, para un pedido ya cerrado sin comprobante y dentro del plazo que
  defina la jurisdicción. Un pedido tiene como máximo un comprobante vigente.
- El sistema no obliga a emitir. Cumplir la obligación tributaria es
  responsabilidad del negocio. Los reportes muestran ventas con y sin
  comprobante por separado para que pueda controlarlo.
- Billing no depende del módulo para funcionar. Recibe un puerto opcional
  compuesto en `cmd/api`. Si la empresa no tiene la capacidad habilitada y el
  cobro trae un pedido de comprobante, responde `einvoicing_not_enabled` sin
  registrar el pago.

## Estructura del módulo

```text
internal/modules/einvoicing/
  contracts/        contrato público para billing, orders y platformadmin
  domain/           documento fiscal, serie, estados y tipos neutros
  application/      emitir, anular, reintentar, conciliar; outbox
  ports/
    jurisdiction.go reglas fiscales de un país
    provider.go     transporte hacia el proveedor homologado
  jurisdictions/
    registry.go     resuelve jurisdicción por country_code
    pe/             Perú
  providers/
    registry.go     resuelve proveedor por clave configurada
    sunat/          motor propio: UBL 2.1, firma, SOAP y CDR
    fake/           proveedor determinista para pruebas
    <otro>/         OSE/PSE o servicio externo, si se agrega
  infrastructure/postgres/
  transport/http/
```

Los registros de jurisdicciones y proveedores se componen únicamente en
`cmd/api`, como el resto de adaptadores. Un país sin jurisdicción registrada no
ofrece facturación electrónica: el módulo responde `einvoicing_unsupported_country`
y la venta sigue con el ticket operativo existente.

## Puertos

### Jurisdicción

Responsabilidad: reglas del país, sin red ni credenciales.

- `Catalog()`: tipos de comprobante (código fiscal y tipo neutro), tipos de
  documento del cliente con su validación, motivos de nota de crédito/débito.
- `ValidateIssue(draft)`: obligatorios y umbrales. Ejemplo PE: factura exige RUC;
  boleta con importe mayor al umbral exige documento de identidad.
- `FormatNumber(series, number)`: número fiscal visible, p. ej. `F001-123`.
- `Printout(source)`: textos legales y contenido del QR de la representación
  impresa; el cliente no conoce títulos, leyendas ni formato del QR del país.
- `TaxBreakdown(lines, components)`: desglose con los componentes de impuesto
  vigentes en la fecha de emisión (ver «Impuestos»).
- `CancellationPolicy()`: cómo se anula (nota de crédito, baja, plazos).
- `ExcludedFromTaxBase()`: conceptos que no integran la base imponible
  (la propina, en Perú).

### Proveedor

Responsabilidad: transporte, sin reglas de negocio.

- `CredentialFields()` y `RequiresCredentials(environment)`: qué pide el
  proveedor y en qué ambiente es obligatorio.
- `ValidateCredentials(environment, values) -> metadata`: valida al guardar y
  al comprobar si el perfil está listo; los metadatos no son secretos.
- `Issue(ctx, document) -> Result`: estado, código y mensaje, valor resumen y
  archivos (XML firmado, CDR) que el núcleo guarda.
- Pendientes: `Status` para respuestas con ticket y `Cancel` para bajas.

Cada llamada viaja con una clave de idempotencia derivada del documento. Un
reintento nunca produce dos comprobantes.

## Configuración

| Nivel | Qué guarda |
| --- | --- |
| Perfil fiscal (país) | Proveedor y ambiente `test` o `production` (sin interruptor propio; `ready` e `issue` se derivan del emisor y las credenciales). El emisor sale de Mi restaurante (RUC y razón social) y del local (dirección). |
| Local | Series habilitadas por tipo neutro de comprobante, su último correlativo y el valor inicial del selector en el cobro (inicial: Sin comprobante). |
| Cliente | Tipo y número de documento ya existentes en Customers; Customers expone un contrato de lectura. |

### Credenciales

Decisión del 2026-11-01: **cifradas en PostgreSQL con una llave maestra
externa**, no un secreto de Secret Manager por empresa (miles de empresas, alta
y rotación desde la pantalla). Solo la llave maestra vive en Secret Manager.

- Hoy cada empresa usa su propio certificado y su usuario SOL secundario
  (`scope = organization`, una fila por perfil fiscal y proveedor).
- El modelo admite credenciales de plataforma (`scope = platform`, una por
  proveedor) para un certificado o una cuenta de Fudia ante un proveedor
  externo como llama.pe, sin migrar datos.
- Un valor vacío al guardar conserva el anterior; el certificado se valida al
  guardarlo (contraseña, cadena y vencimiento) y se audita
  `einvoicing.credentials.updated` sin valores.
- Rotar la llave: nueva versión en `FUDIA_EINVOICING_MASTER_KEY` y
  `FUDIA_EINVOICING_KEY_VERSION`, la anterior en `FUDIA_EINVOICING_PREVIOUS_KEYS`;
  cada fila recuerda su versión.

Las series dejan de ser
responsabilidad de Organizations: pertenecen a EInvoicing porque su formato y su
autorización dependen del país.

## Ciclo de un comprobante

```text
cobro (billing) con campo opcional `fiscal`
  ├─ sin `fiscal` o «Sin comprobante» → flujo actual, sin documento
  └─ con `fiscal` y el pago cierra el pedido
       → einvoicing.Request (misma transacción: documento PENDING + outbox)
       → worker: jurisdicción calcula → proveedor emite
       → ACCEPTED | OBSERVED | REJECTED | ERROR
       → reintento idempotente o nota/anulación según la jurisdicción

POS tras el cobro con comprobante (sin ticket de pago):
  consulta el documento cada segundo hasta 8 s
  ├─ ACCEPTED | OBSERVED → abre el PDF del comprobante
  ├─ sin respuesta o ERROR, ya firmado → abre el PDF con aviso; sigue consultando
  ├─ REJECTED → no imprime; muestra el motivo y ofrece el ticket de pago
  └─ sin firmar (p. ej. credenciales) → muestra el motivo y ofrece el ticket de pago

venta cerrada sin comprobante → «Emitir comprobante» en Ventas → mismo Request
```

- Los datos fiscales se validan antes de registrar el pago. Una factura sin RUC
  válido rechaza el cobro completo, no solo la emisión.
- Con pagos parciales, el comprobante se solicita en el pago que cierra el
  pedido: un comprobante por pedido.

- La serie y el correlativo se reservan con bloqueo de fila al crear el
  documento, nunca en el worker.
- La fecha de emisión es la fecha de negocio del cobro en la zona horaria del
  local, no la apertura de la mesa ni la hora del servidor.
- El documento guarda una copia inmutable de líneas, importes, descuentos,
  componentes de impuesto aplicados, datos del cliente y del emisor.
- Una nota de crédito referencia exactamente un comprobante y sus líneas
  deben cuadrar con sus totales; la nota parcial no replica el documento
  completo.
- Cada canal de comunicación se usa una sola vez por documento. El motor
  propio envía boletas y facturas individualmente con `sendBill` (SUNAT beta
  las acepta con CDR); por eso no se incluyen además en un resumen diario.

Estados neutros: `PENDING`, `SENDING`, `ACCEPTED`, `OBSERVED`, `REJECTED`,
`ERROR`, `CANCELLED`. El frontend los muestra con la matriz de estados del
sistema visual; no recolorea por país.

## Ejecución asíncrona

Decisión del 2026-11-01: **outbox en PostgreSQL + Cloud Tasks + Cloud Scheduler
de respaldo**.

```text
1. cobro (una transacción): pago + comprobante PENDING + fila en outbox
2. tras el commit: tarea en Cloud Tasks, nombre = id del comprobante
3. Cloud Tasks → endpoint interno de Cloud Run (OIDC) → proveedor
   ├─ aceptado / rechazado → estado final; cierra la fila del outbox
   ├─ respuesta con ticket → tarea «consultar estado» programada (+2 min)
   └─ falla temporal → reintento de la cola con espera creciente
4. Cloud Scheduler cada 5 min → barrido: reencola los PENDING atascados
```

| Pieza | Responsabilidad |
| --- | --- |
| Outbox | Garantiza que no exista un pago con pedido de comprobante sin su tarea; se escribe en la transacción del cobro. |
| Cloud Tasks | Disparo en segundos, límite de envíos por segundo y simultáneos hacia el proveedor (configuración de la cola), tareas programadas para consultas diferidas, reintentos y rechazo de tareas duplicadas por nombre. |
| Cloud Scheduler | Un solo job de barrido. Recupera las tareas que no se llegaron a crear; no es el camino normal. |
| Idempotencia | El id del comprobante es la clave hacia el proveedor; un reintento nunca emite dos veces. |

- El endpoint del worker no es público: valida el token OIDC de la cuenta de
  servicio de Cloud Tasks y de Cloud Scheduler.
- No hay un proceso permanente: Cloud Run atiende cada tarea como una petición,
  aunque el servicio escale a cero.
- Se descartaron un **Scheduler solo** (hasta 60 s de espera y ejecuciones sin
  trabajo) y **publicar sin outbox** (doble escritura: pagos sin comprobante o
  comprobantes sin pago).
- **Pub/Sub** es equivalente si el proveedor elegido no limita el ritmo ni
  responde con tickets: reemplazaría solo el paso 2. Más adelante puede
  agregarse **además** de Cloud Tasks para avisar a varios consumidores
  (WhatsApp, reportes) que un comprobante fue aceptado.
- Costo estimado con ~600 000 comprobantes al mes: Cloud Tasks USD 0,08–0,56
  (el primer millón de operaciones es gratis, luego USD 0,40 por millón) y
  Cloud Scheduler USD 0–0,10 (3 jobs gratis por cuenta de facturación). Con
  motor propio no hay costo por comprobante de un OSE/PSE.
- SUNAT beta limita el ritmo (HTTP 401 si se envía muy rápido); la cola debe
  limitar envíos por segundo, otra razón para Cloud Tasks frente a Pub/Sub.
- Infraestructura nueva en `foods-infrastructure`: cola de Cloud Tasks, job de
  Cloud Scheduler, cuenta de servicio con permiso de invocación y variables de
  la cola en el API.

## Impuestos

El comprobante lleva **un solo impuesto** con su tasa total. En Perú el «IGV
18 %» ya es IGV 16 % + IPM 2 %, y el IGV reducido de MYPE de restaurantes,
hoteles y alojamientos (Leyes 31556, 32219 y 32387; padrón SUNAT, al menos 70 %
de ingresos del rubro) solo cambia ese porcentaje: 10 % en 2025, 10,5 % en 2026
(IGV 8 + IPM 2,5) y 15 % en 2027 (IGV 12 + IPM 3); desde 2028 vuelve la tasa
general. El IPM no se emite como un impuesto aparte.

**Implementado (migración 000100, reemplaza los regímenes por perfil de la
000097):** los administradores de plataforma registran en Plataforma ›
Impuestos la tasa de cada periodo
(`special_tax_rates`, con base legal) y el perfil fiscal solo indica si la
empresa está acogida (`special_tax_regime`). Plataforma y la empresa reciben un
aviso 30 días antes de que terminen las vigencias registradas. Ver «IGV reducido MYPE» en `04_DATA_MODEL.md`.
EInvoicing obtiene el impuesto de la fecha de emisión con
`organizations/contracts.TaxResolver` —un único componente con el nombre del
impuesto del perfil— y guarda una copia en cada comprobante; todas las líneas
usan la misma tasa, como exige SUNAT.

Estar en el padrón es una declaración del administrador de la empresa, no una
inferencia del sistema. SUNAT acepta sin observación un comprobante al 18 % de
una empresa del padrón, por eso la tasa se aplica sola una vez marcada.
Pendiente para notas de crédito y débito: heredar la tasa del comprobante que
modifican, no la vigente.

## Perú (`jurisdictions/pe`)

Primera jurisdicción. Referencias normativas vigentes al 2026-11-01:

- Comprobantes: factura `01`, boleta `03`, nota de crédito `07`, nota de débito
  `08`. Tipos de documento del cliente: DNI `1`, RUC `6`, otros según catálogo
  SUNAT; se exponen por API.
- Factura: RUC de 11 dígitos y razón social obligatorios.
- Boleta: documento de identidad obligatorio sobre el umbral vigente.
- La propina no forma parte de la base del IGV.
- RS 000048-2026/SUNAT, vigente desde el 2027-01-01 por la RS 000143-2026:
  comunicación de baja en 7 días calendario y relación uno a uno entre nota y
  comprobante.
- El número fiscal `serie-correlativo` es obligatorio en la representación
  impresa (excepción explícita de la regla 15 de `AGENTS.md`).

Envío directo a SUNAT (SEE-Del contribuyente) con `providers/sunat`:

| Ambiente | Endpoint | Credenciales |
| --- | --- | --- |
| Pruebas | `https://e-beta.sunat.gob.pe/ol-ti-itcpfegem-beta/billService` | Usuario público de SUNAT y certificado autofirmado; no requiere datos de la empresa. |
| Producción | `https://e-factura.sunat.gob.pe/ol-ti-itcpfegem/billService` | Usuario SOL secundario (se envía como RUC + usuario), clave SOL y certificado digital `.pfx` de la empresa. |

Mapeo UBL: `CustomizationID` 2.0, tipo de operación `0101`, importe en letras
(nota `1000`), forma de pago Contado en facturas, IGV `1000` con el porcentaje
total vigente (IGV reducido MYPE 2026: 10,5), afectación `10`, unidad
`NIU` o `ZZ`, cliente anónimo con tipo `0`, número `-` y «CLIENTES VARIOS».

## Agregar un país

1. Crear `jurisdictions/<cc>` con catálogo, validaciones, formato y política de
   anulación, más pruebas unitarias de reglas.
2. Crear o reutilizar un adaptador en `providers/`.
3. Registrar ambos en `cmd/api`.
4. Documentar los regímenes especiales del país. La tasa general se configura
   en el perfil fiscal; los regímenes no se siembran: los registra el
   administrador.
5. Documentar la jurisdicción en este archivo.

No se crean pantallas, tablas ni servicios por país. El formulario de
configuración y el de datos del cliente se arman con el catálogo de la
jurisdicción.

## Referencia descartada

Se revisó un sistema PHP/Laravel con Greenter 2.x (rama de desarrollo y parches
sobre `vendor`). Sirve solo como lista de casos. Errores que este diseño
previene: boletas enviadas individualmente y otra vez en resumen diario; usuario
SOL de producción sin el RUC antepuesto; descuento global fuera de las líneas;
IGV calculado sobre la propina; nota de crédito parcial con todas las líneas;
fecha de emisión tomada de la apertura del pedido; credenciales y certificados
en la base o en el repositorio.
