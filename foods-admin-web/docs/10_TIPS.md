# Propinas en configuración, POS y caja

La vertical `src/modules/operations/tips` concentra contratos, cálculos y las dos
superficies reutilizables. Usa los tokens y componentes existentes, sin crear
otra paleta ni otro patrón de modal.

El wizard Registrar empresa incorpora la opción «Configurar propinas para este
local ahora» dentro de Primer local, sin añadir otro paso. Comparte
`TipSettingsFields` con Locales. Si se omite, no envía `tipSettings`; al marcarla
usa los valores y canales del catálogo de onboarding. La validación pertenece al
paso del local y los valores se conservan al avanzar o volver.

- **Locales → editar o nuevo local:** sección «Propinas» dentro del formulario del local (2026-11-01; reemplaza el botón «Propinas» de cada fila). Empieza con el selector «Funcionamiento» y muestra el resto solo si no están desactivadas. Solo la ve el administrador reservado (`editable` del API); para otros usuarios no aparece. Se envían únicamente si se cambiaron, junto con el local y en la misma transacción: un local nuevo sin cambios sigue heredando la configuración de la empresa. Sin pestañas de alcance ni casilla de herencia.
- **POS → Cobrar (incluye el cobro de una mesa):** «Atendió: <mozo>»
  (`order.waiterName` del detalle del pedido; se omite si la atención no tiene
  mozo, p. ej. mostrador o delivery) está siempre visible en el visor fijo del
  cobro, junto a la caja, y no se repite en la propina. «Propina» es un bloque
  plegable (`FormDisclosure`, cerrado al abrir el cobro; su resumen muestra el
  importe una vez aceptada). La aceptación
  inicia sin marcar. El porcentaje no se escribe: es el configurado para el local
  y se muestra en el texto «10 % sobre el consumo de S/ 17.00…». El selector
  «Tipo de propina» (Porcentaje del local / Monto libre) aparece solo si el local
  permite monto libre; el campo «Monto de la propina» solo con monto libre. No
  existe «Entregada directamente al personal» (2026-11-03): una propina recibida
  en mano no se registra. El efectivo recibido y el vuelto toman en cuenta la
  propina, manteniendo `amount` como consumo. El cobro envía `{kind:"percentage"}`
  o `{kind:"fixed",value}`; el backend aplica el porcentaje de la configuración.
- **POS → Ver detalle → Ver precuenta:** PDF de 80 mm con sugerencia voluntaria,
  antes de cobrar. Previsualizar pago mantiene el ticket de cobros confirmados.
- **Caja → Cerrar turno:** si hay saldo pendiente, muestra una sección opcional de entrega con trabajador autocompletable, importe y efectivo separado/transferencia. La casilla inicia desmarcada. El mismo botón de cierre envía conteo y entrega en una sola petición y transacción; no abre otro modal. Sin entrega, el saldo queda pendiente para un próximo turno. Las propinas no modifican ingresos, egresos ni efectivo esperado de caja.
- **Reporte de caja y su PDF:** sección separada de propinas recibidas y entregadas,
  con destinatario, actor, medio, fecha y saldo pendiente del turno.

`LocationDialog` consulta `getTipSettings("location", id)` (o la de la empresa para un local nuevo) y reutiliza `TipSettingsFields` y `tipDraftFromPolicy`; la sección tiene skeleton y error con reintento, y un 403 la oculta. La consulta y el guardado usan el local elegido; no cambian la sesión ni el local activo. `TipClosingSection` es controlado por el formulario de cierre y usa el espacio de propinas del turno, incluso en cierre ciego sin exponer el efectivo esperado. El reporte detallado de propinas se muestra en la consulta del informe; en el cierre se presenta solo la sección de entrega para evitar duplicados.
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

Los reportes, el ticket y el detalle siguen mostrando «Directa al personal» solo
para propinas históricas registradas así.

Pruebas de frontend: importes separados, propina porcentual de configuración/libre, monedas sin
centavos, vuelto, reglas de sugerencia y contenido/tamaño de PDFs. La revisión
visual usa componentes reales con datos controlados, en escritorio y desde 390 px,
incluyendo pantalla de 640 px de alto. Las pruebas de impresión física siguen las
limitaciones del visor PDF documentadas en `09_PDF_VIEWER.md`.

Se necesita la migración 95 y el backend actualizado. No se requieren variables
públicas, secretos ni recursos GCP adicionales. Las devoluciones de consumo no
devuelven automáticamente propinas; el desglose conserva esa separación.
