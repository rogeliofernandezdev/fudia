# Propinas en configuración, POS y caja

La vertical `src/modules/operations/tips` concentra contratos, cálculos y las dos
superficies reutilizables. Usa los tokens y componentes existentes, sin crear
otra paleta ni otro patrón de modal.

El wizard Registrar empresa incorpora la opción «Configurar propinas para este
local ahora» dentro de Primer local, sin añadir otro paso. Comparte
`TipSettingsFields` con Locales. Si se omite, no envía `tipSettings`; al marcarla
usa los valores y canales del catálogo de onboarding. La validación pertenece al
paso del local y los valores se conservan al avanzar o volver.

- **Locales → Propinas:** acción de cada fila que abre la configuración del local elegido, identificado por su nombre. Sin pestañas de alcance ni casilla de herencia. Guardar crea una configuración propia para ese local.
- **POS → Cobrar:** aceptación sin marcar por defecto; porcentaje o monto libre;
  entrega directa al personal opcional. El efectivo recibido y el vuelto toman
  en cuenta la propina retenida, manteniendo `amount` como consumo.
- **POS → Ver detalle → Ver precuenta:** PDF de 80 mm con sugerencia voluntaria,
  antes de cobrar. Previsualizar pago mantiene el ticket de cobros confirmados.
- **Caja → Propinas:** entrega manual a un trabajador con búsqueda autocompletable,
  importe y efectivo/transferencia. No sustituye la función Egreso operativo.
- **Reporte de caja y su PDF:** sección separada de propinas recibidas y entregadas,
  con destinatario, actor, medio, fecha y saldo pendiente del turno.

`TipSettingsDialog` se abre desde las acciones de Locales y recibe `location` (id y nombre) y `close`. La consulta y el guardado usan el local elegido; no cambian la sesión ni el local activo. `TipPayoutDialog` recibe `shift` y `close`.
El flujo usa `ModalForm → ModalBody + ModalFooter`; solo el cuerpo se desplaza, y
el pie permanece accesible en móvil. El estado remoto contempla skeleton, error
con reintento y ausencia de datos. La consulta está ligada a empresa, local y actor.

`PaymentTicketDialog` admite `mode="prebill"`; por defecto conserva `payment`.
El builder y las filas del ticket admiten el mismo modo. Al cambiar el documento,
pago o modo se descarta la preparación anterior, respetando el visor PDF existente.

La propina se acepta una sola vez por cuenta aunque el consumo se cobre en partes.
Si ya existe, el POS muestra el importe y no permite volver a cobrarla. La API es
quien valida canal, permisos, precisión monetaria y duplicados. El frontend nunca
envía empresa, local, actor ni total de propina calculado como autoridad.

La entrega conserva `requestKey` al reintentar. Un conflicto tras una respuesta
incierta requiere revisar el reporte antes de iniciar otra entrega; no se genera
automáticamente una clave nueva para esconder el conflicto.

Pruebas de frontend: importes separados, propina porcentual/libre, monedas sin
centavos, vuelto, reglas de sugerencia y contenido/tamaño de PDFs. La revisión
visual usa componentes reales con datos controlados, en escritorio y desde 390 px,
incluyendo pantalla de 640 px de alto. Las pruebas de impresión física siguen las
limitaciones del visor PDF documentadas en `09_PDF_VIEWER.md`.

Se necesita la migración 95 y el backend actualizado. No se requieren variables
públicas, secretos ni recursos GCP adicionales. Las devoluciones de consumo no
devuelven automáticamente propinas; el desglose conserva esa separación.
