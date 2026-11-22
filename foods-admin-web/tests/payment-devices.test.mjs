import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url);
const read=path=>readFileSync(new URL(`../${path}`,import.meta.url),"utf8");
function compile(path){
  const exports={};
  vm.runInNewContext(ts.transpileModule(read(path),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,{exports,require,Date,Math});
  return exports;
}

test("Fudia Pay: la matriz de estados del pago detectado usa la paleta aprobada sin rojo",()=>{
  const status=compile("src/modules/payment-devices/domain/pay-status.ts");
  assert.deepEqual(Object.values(status.payStatusMeta).map(meta=>meta.tone),["orange","blue","green","gray"]);
  assert.deepEqual(Object.values(status.payStatusMeta).map(meta=>meta.label),["Detectado","Asociado","Confirmado","Descartado"]);
  for(const meta of Object.values(status.payStatusMeta))assert.ok(meta.label.length<=12,meta.label);
  assert.equal(status.payProviderLabel.yape,"Yape");
  assert.equal(status.payProviderLabel.plin,"Plin");
});

test("Fudia Pay: la vinculación es como WhatsApp Web: QR, «Vinculando…» al escanear y cierre automático",()=>{
  const dialog=read("src/modules/payment-devices/presentation/pay-pairing-dialog.tsx");
  assert.match(dialog,/createPayPairing\(locationId\)/,"el diálogo genera el código temporal");
  assert.match(dialog,/toCanvas\(canvas\.current,draft\.code/,"el QR se dibuja con el código, no con credenciales");
  assert.match(dialog,/getPayPairingStatus\(locationId,draft!\.id\)/,"sigue el estado del código mostrado");
  assert.match(dialog,/\["linked","expired"\]\.includes\(query\.state\.data\?\.status\?\?""\)\?false:STATUS_POLL_MS/,"consulta hasta que se vincula o vence");
  assert.match(dialog,/Vinculando\{status\.data\?\.model/,"al escanear el QR cede su lugar al progreso con el modelo");
  assert.match(dialog,/<Spinner size=\{40\}\/>/);
  assert.match(dialog,/onLinked\(status\.data\?\.deviceName/,"al vincular avisa al panel, que cierra el modal");
  assert.match(dialog,/Generar otro código/,"un código vencido se renueva sin cerrar el modal");
  assert.doesNotMatch(dialog,/token/i,"el QR y la pantalla nunca muestran la credencial del dispositivo");
  const panel=read("src/modules/payment-devices/presentation/pay-devices-panel.tsx");
  assert.match(panel,/\{pairing&&createPortal\(<PayPairingDialog locationId=\{locationId\} locationName=\{locationName\} onClose=\{closePairing\} onLinked=\{linked\}\/>,document\.body\)\}/,"el diálogo se monta solo al abrirlo, ligado al local, fuera del modal del local para no recortarse");
  assert.match(panel,/\{revokeTarget&&createPortal\(<ConfirmDialog open tone="danger"/,"la confirmación de revocar tampoco queda recortada por el modal del local");
  assert.match(read("src/modules/payment-devices/presentation/payment-devices.css"),/\.crud-modal\.pay-pairing-modal\{width:min\(var\(--size-640\),100%\)\}/,"el QR tiene su propio ancho");
  assert.match(panel,/title:"Dispositivo vinculado"/);
  assert.match(read("src/modules/payment-devices/infrastructure/pay-devices-api.ts"),/\$\{locationDevices\(locationId\)\}\/pairing\/\$\{encodeURIComponent\(id\)\}/);
  assert.match(dialog,/Android de «\{locationName\}»/,"el QR dice a qué local quedará ligado el teléfono");
});

test("Fudia Pay: los endpoints del dispositivo y de Caja siguen el contrato",()=>{
  const api=read("src/modules/payment-devices/infrastructure/pay-devices-api.ts");
  assert.match(api,/const locationDevices=\(locationId:string\)=>`locations\/\$\{encodeURIComponent\(locationId\)\}\/pay-devices`/,"los dispositivos cuelgan del local elegido en Locales");
  assert.match(api,/apiFetch<PayPairingDraft>\(`\$\{locationDevices\(locationId\)\}\/pairing`,\{method:"POST"\}\)/);
  assert.match(api,/\$\{locationDevices\(locationId\)\}\/\$\{encodeURIComponent\(id\)\}\/revoke/);
  assert.doesNotMatch(api,/renamePayDevice|method:"PATCH"/,"la vinculación es única: no se edita el dispositivo");
  assert.match(api,/associatePayNotification\(id:string,orderId:string,method:string,amount\?:number\)/,"asociar registra el cobro real y admite el registro parcial");
  assert.match(api,/pos\/orders\?\$\{params\.toString\(\)\}/,"la búsqueda de pedidos usa el listado del POS");
  assert.doesNotMatch(api,/confirmPayNotification/,"la asociación cierra el flujo: no existe confirmación separada");
});

test("Fudia Pay: aviso flotante global y modal de detalle con asociación revisada",()=>{
  const layer=read("src/modules/payment-devices/presentation/pay-notice-layer.tsx");
  assert.match(layer,/useSession\(\)/,"la capa vive en el área admin y conoce la sesión");
  assert.match(layer,/meta:\{live:\["pay"\],poll:10000\}/,"llega por el canal en vivo y, sin él, sondea cada 10 s con sesión y sigue pese a fallos pasajeros del API");
  assert.match(layer,/retry:1/,"un 503 transitorio no silencia los avisos hasta recargar");
  assert.match(layer,/if\(!sessionOk\)return null/,"sin sesión o sin permiso no se monta el feed: cero 401 que expulsen al login");
  assert.match(layer,/PayNotificationDetail notification=\{detail\}/,"el clic abre el detalle");
  const detail=read("src/modules/payment-devices/presentation/pay-notification-detail.tsx");
  assert.match(detail,/usePayCollection\(\)/,"usa el pedido en cobro como sugerencia");
  assert.match(detail,/la asociación nunca es automática/,"el pedido abierto es sugerencia, no selección");
  assert.match(detail,/Se registrará solo el saldo/,"un pago superior al saldo se registra parcial");
  assert.match(detail,/Volver a pedidos/,"permite regresar a la lista antes de confirmar");
  assert.match(layer,/PayNotificationDetail notification=\{detail\} mode="associate"/,"el aviso de un pago por revisar abre directo la asociación");
  assert.doesNotMatch(detail,/Asociar a pedido/,"la consulta no repite la acción de asociar: ya existe el botón de enlace");
  assert.doesNotMatch(detail,/setMode\(/,"la intención la decide quien abre el modal");
  assert.doesNotMatch(detail,/<Select[\s>]/,"el logo ya identifica el medio de pago: sin selector bloqueado");
  assert.match(detail,/className="pay-order-option" aria-label=\{`Revisar \$\{title\}/,"cada fila completa es el área táctil para revisar el pedido");
  assert.match(detail,/Monto exacto/,"marca los pedidos cuyo monto coincide con el pago");
  const css=read("src/modules/payment-devices/presentation/payment-devices.css");
  assert.match(css,/\.pay-notice-layer\{position:fixed/);
  assert.match(css,/\.pay-provider-tag\.plin\.is-small \.pay-provider-logo\{width:var\(--size-24\)/,"Plin llena su lienzo: se reduce para igualar el ancho visible de Yape");
  assert.match(css,/\.pay-provider-tag\.yape\.is-small \.pay-provider-logo\{width:var\(--size-48\)/,"Yape se amplía para compensar sus márgenes blancos");
  assert.match(css,/z-index:var\(--z-tooltip\)/,"la capa queda por encima de los modales sin cerrarlos");
});

test("Asociar pago: pedidos por mesa o cliente, saldo pendiente y búsqueda en el servidor sin consultas por letra",()=>{
  const detail=read("src/modules/payment-devices/presentation/pay-notification-detail.tsx");
  assert.match(detail,/const query=useDebouncedValue\(q\.trim\(\)\)/,"espera a que termine de escribir");
  assert.match(detail,/queryKey:\["pay-pending-orders",notification\.id,query\]/);
  const api=read("src/modules/payment-devices/infrastructure/pay-devices-api.ts");
  assert.match(api,/export function searchPayOrders\(q:string,paymentStatus="unpaid"\)/,"solo pedidos por cobrar");
  assert.match(api,/new URLSearchParams\(\{q,paymentStatus,ready:"true",page:"1",pageSize:"12"\}\)/,"solo los que el cobro aceptaría: nada que falle al confirmar");
  assert.match(detail,/const due=\(item:PayOrderRef\)=>Number\(item\.remainingAmount\?\?item\.total\?\?0\)/,"compara con el saldo, no con el total");
  assert.match(detail,/table=channel==="salon"\?item\.tableName:""/,"la mesa se muestra con su nombre («Mesa 2»), no solo «Mesa»");
  assert.match(detail,/<PayProviderTag provider=\{notification\.provider\} appearance="circle"\/>/,"el resumen usa el círculo de billetera");
  assert.match(detail,/<p className="pay-detail-sender"><strong>\{notification\.senderName\|\|"Sin remitente"\}<\/strong><time dateTime=\{notification\.occurredAt\}>/,"al asociar se muestra quién pagó y cuándo llegó el pago");
  assert.doesNotMatch(detail,/Los pedidos con el mismo monto aparecen primero/,"la etiqueta «Monto exacto» ya lo dice");
  const css=read("src/modules/payment-devices/presentation/payment-devices.css");
  assert.match(css,/\.pay-order-options li\+li\{border-top:var\(--stroke-1\) solid var\(--line\)\}/,"lista continua con líneas finas");
});

test("Fudia Pay: Caja revisa los pagos detectados y cada local vincula sus dispositivos",()=>{
  const cash=read("src/modules/operations/cash/presentation/cash-page.tsx");
  assert.match(cash,/value:"pagos",label:"Pagos detectados",icon:"wallet"/);
  assert.match(cash,/PayNotificationsPanel dateTime=\{dateTime\}/);
  const admin=read("src/modules/organizations/presentation/organization-admin.tsx");
  assert.doesNotMatch(admin,/pay-devices-settings/,"Mi restaurante ya no lista dispositivos: son de cada local");
  assert.match(admin,/\{value:"devices",label:"Dispositivos",icon:"phoneApp",count:activeDevices\}/,"pestaña del formulario del local con los activos");
  assert.match(admin,/const showDevices=Boolean\(initial\.id\)&&can\("organizations\.manage"\)/,"solo al editar un local existente y con permiso");
  assert.match(admin,/<PayDevicesPanel locationId=\{initial\.id\} locationName=\{initial\.name\} dateTime=\{dateTime\}\/>/);
  assert.match(admin,/<ModalForm hidden=\{tab!=="data"\}/,"cambiar de pestaña no descarta los cambios del local");
  assert.doesNotMatch(cash,/PayDevicesPanel/,"Caja no administra dispositivos: eso es configuración");
  const routing=read("src/shared/routing/page-routes.ts");
  assert.doesNotMatch(routing,/detectedPayments/,"la página separada ya no existe");
  const nav=read("src/shell/navigation.ts");
  assert.doesNotMatch(nav,/detectedPayments/,"sin casilla de navegación para la página eliminada");
  const view=read("src/modules/payment-devices/presentation/pay-notifications-panel.tsx");
  assert.match(view,/meta:\{live:\["pay"\],poll:query=>.*\.some\(item=>item\.status==="detected"\)\?10000:30000\}/,"los pagos por revisar se consultan con más frecuencia");
  assert.match(view,/openPayment\(item,"detail"\)/,"el ojo abre la consulta del pago");
  assert.match(view,/openPayment\(item,"associate"\)/,"el enlace abre directamente la tarea de asociación");
  assert.match(view,/mode=\{detailMode\}/,"el modal recibe la intención elegida");
  const detail=read("src/modules/payment-devices/presentation/pay-notification-detail.tsx");
  assert.doesNotMatch(detail,/La asociación registra el cobro real/,"la revisión no repite lo que dice el botón");
  assert.match(detail,/<SelectedOrder order=\{\{\.\.\.selected,/,"la revisión identifica el pedido como la lista");
  assert.match(detail,/senderMatchesCustomer\(senderName,order\.customerName\)&&<Status tone="green" icon="userCheck">Coincide con el remitente<\/Status>/);
  assert.match(detail,/<PaymentCoverage provider=\{notification\.provider\} remaining=\{remaining\} applied=\{Math\.min\(amount,remaining\)\}/,"barra de cobertura con lo que aplica el pago");
  assert.match(detail,/<dt>Quedará pendiente<\/dt>/);
  assert.match(detail,/left>0\?<Status tone="orange">Pago parcial<\/Status>:<Status tone="green" icon="circleCheck">Pago completo<\/Status>/,"estado del resultado con la etiqueta de cuenta");
  assert.match(detail,/const covered=left===0\?100:Math\.min\(99,Math\.max\(1,Math\.round\(applied\/remaining\*100\)\)\)/,"un pago parcial nunca dice 0 % ni 100 %");
  assert.doesNotMatch(detail,/label:"Pagos registrados"|label:"Total del pedido"/,"la barra reemplaza los importes sueltos");
  assert.match(detail,/associating\?<section className="pay-detail-summary is-compact"/,"al asociar, el resumen reúne remitente y hora");
  assert.match(detail,/closest\("\.modal-body"\)\?\.scrollTo\(\{top:0\}\)/,"cada paso abre desde arriba");
  const css=read("src/modules/payment-devices/presentation/payment-devices.css");
  assert.match(css,/\.pay-coverage-bar\[data-provider=yape\]>span,\.pay-coverage-dot\[data-provider=yape\]\{background:var\(--wallet-yape\)\}/,"la barra usa el color de la billetera");
  assert.doesNotMatch(css,/\.pay-ledger/,"sin restos del cuadre anterior");
  const match=read("src/modules/payment-devices/domain/sender-match.ts");
  assert.match(match,/normalize\("NFD"\)/,"compara sin tildes");
  assert.match(match,/short\.length>=2&&short\.every\(\(word,index\)=>long\[index\]\?\.startsWith\(word\)\)/,"palabras en orden, admite apellidos recortados");
  assert.match(detail,/Asociar pago a pedido/);
  const pos=read("src/modules/operations/pos/presentation/pos-page.tsx");
  assert.match(pos,/PayCollectionRegistration order=/,"el POS registra el pedido en cobro como sugerencia");
  const providers=read("src/providers/providers.tsx");
  assert.match(providers,/PayCollectionProvider/,"la sugerencia vive en contexto global");
  const layout=read("src/app/(admin)/layout.tsx");
  assert.match(layout,/PayNoticeLayer/,"el aviso flotante vive en el área admin con sesión");
});

test("Fudia Pay: las pestañas usan DataTable del sistema y los pagos paginan en el servidor",()=>{
  for(const file of ["pay-devices-panel.tsx","pay-notifications-panel.tsx"]){
    const view=read(`src/modules/payment-devices/presentation/${file}`);
    assert.match(view,/<DataTable label=/,`${file}: tabla del sistema con cabecera marino`);
    assert.doesNotMatch(view,/<table/,`${file}: sin tablas sueltas`);
    assert.match(view,/<div className="table-actions">/);
  }
  const payments=read("src/modules/payment-devices/presentation/pay-notifications-panel.tsx");
  assert.match(payments,/listPayNotificationPage\(\{status,provider,q,page,size\}\)/);
  assert.match(payments,/<Pagination page=\{page\} size=\{size\} total=\{notifications\.data\?\.total\?\?0\}/);
  assert.match(payments,/setStatus\(event\.target\.value\);setPage\(1\)/,"cambiar un filtro vuelve a la primera página");
  assert.match(read("src/modules/payment-devices/infrastructure/pay-devices-api.ts"),/page:String\(filter\.page\),size:String\(filter\.size\)/);
  const css=read("src/modules/payment-devices/presentation/payment-devices.css");
  assert.doesNotMatch(css,/\.pay-devices-table th|\.pay-devices-table td\{/,"las filas y cabeceras las pone data-table.css");
});

test("Pagos detectados: Asociar en azul sólido, monto en negrita y recepción en dos líneas",()=>{
  const payments=read("src/modules/payment-devices/presentation/pay-notifications-panel.tsx");
  assert.match(payments,/<IconButton icon="link" className="solid" label=/,"Asociar se distingue de Ver con el azul sólido");
  assert.match(payments,/<IconButton icon="eye" label=/,"Ver conserva el botón de fila suave");
  assert.match(payments,/\{dateTime\(item\.occurredAt,"date"\)\}<small>\{dateTime\(item\.occurredAt,"time"\)\}<\/small>/,"fecha arriba y hora debajo");
  const css=read("src/modules/payment-devices/presentation/payment-devices.css");
  assert.match(css,/\.pay-notifications-table td\.pay-amount\{color:var\(--ink-950\);font-weight:var\(--weight-bold\)\}/);
  const actions=read("src/design-system/styles/table-actions.css");
  assert.match(actions,/\.ds-icon-button\.solid\{color:var\(--surface\);background:var\(--primary-600\);border-color:var\(--primary-600\)\}/,"la variante sólida vive en el sistema, no en el módulo");
  assert.doesNotMatch(css,/\.solid/,"el módulo no recolorea el botón");
});

test("Pagos detectados: buscador por remitente o monto en el servidor, mientras escribe",()=>{
  const payments=read("src/modules/payment-devices/presentation/pay-notifications-panel.tsx");
  assert.match(payments,/const q=useDebouncedValue\(search\.trim\(\)\)/,"una consulta cuando deja de escribir, no una por letra");
  assert.match(payments,/queryKey:\["pay-notifications",status,provider,q,page,size\]/);
  assert.match(payments,/placeholder="Buscar remitente o monto"/,"sin número de operación: las billeteras no lo envían");
  assert.match(payments,/setSearch\(event\.target\.value\);setPage\(1\)/,"buscar vuelve a la primera página");
  assert.match(read("src/modules/payment-devices/infrastructure/pay-devices-api.ts"),/q:filter\.q,page:String\(filter\.page\)/,"la búsqueda va al servidor");
  const css=read("src/modules/payment-devices/presentation/payment-devices.css");
  assert.match(css,/\.pay-notifications-search\{grid-column:1\/-1;width:100%\}/,"en móvil el buscador ocupa su propia fila");
});

test("Pagos detectados en móvil: Acciones fija a la derecha y monto visible sin deslizar",()=>{
  const payments=read("src/modules/payment-devices/presentation/pay-notifications-panel.tsx");
  assert.match(payments,/<DataTable label="Pagos detectados" containerClassName="hover-scroll pay-notifications-table" stickyActions>/);
  assert.match(payments,/<span className="pay-col-provider-label">\{column\}<\/span>/,"el título Proveedor sigue para lectores de pantalla");
  const table=read("src/design-system/data-table.tsx");
  assert.match(table,/stickyActions&&"data-table--sticky-actions"/,"la variante vive en DataTable, no en el módulo");
  const ds=read("src/design-system/styles/data-table.css");
  assert.match(ds,/\.data-table--sticky-actions :is\(th,td\):last-child\{position:sticky;inset-inline-end:0/,"Acciones fija a la derecha en ≤700 px");
  assert.match(ds,/\.data-table--sticky-actions tbody td:last-child\{background:var\(--surface\)\}/);
  const css=read("src/modules/payment-devices/presentation/payment-devices.css");
  assert.match(css,/\.pay-notification-sender strong\{display:-webkit-box;-webkit-box-orient:vertical;-webkit-line-clamp:2/,"el remitente usa hasta dos líneas en móvil");
});

test("Fudia Pay: dispositivos del local sin edición, con filtro de estado, paginación y plataforma",()=>{
  const panel=read("src/modules/payment-devices/presentation/pay-devices-panel.tsx");
  assert.doesNotMatch(panel,/Renombrar|RenameDialog|icon="edit"/,"la vinculación es única: solo se revoca");
  assert.match(panel,/useState<PayDeviceStatusFilter>\("active"\)/,"por defecto solo los activos; los revocados son historial");
  assert.match(panel,/\{value:"revoked",label:"Revocados"\}/);
  assert.match(panel,/setStatus\(event\.target\.value as PayDeviceStatusFilter\);setPage\(1\)/,"cambiar el filtro vuelve a la primera página");
  assert.match(panel,/<Pagination mode="simple" page=\{page\} size=\{PAGE_SIZE\} total=\{devices\.data\?\.total\?\?0\}/,"pagina en el servidor");
  assert.match(panel,/ANDROID:\{label:"Android",icon:"android"\},IOS:\{label:"iPhone",icon:"apple"\}/,"Android e iPhone con su ícono");
  assert.match(panel,/device\.active&&<IconButton icon="cancel"/,"un revocado no vuelve a revocarse");
  assert.match(panel,/const showState=status===""/,"la columna Estado solo en «Todos»: con filtro sería redundante");
  assert.match(read("src/modules/payment-devices/infrastructure/pay-devices-api.ts"),/status:filter\.status,page:String\(filter\.page\),size:String\(filter\.size\)/);
});

test("Fudia Pay: Caja despierta los teléfonos congelados por el fabricante con un push",()=>{
  const wake=compile("src/modules/payment-devices/domain/pay-wake.ts");
  assert.equal(wake.nextPayWakeDelay(1,false),30_000,"con Caja abierta el pago aparece en menos de 30 s");
  assert.equal(wake.nextPayWakeDelay(2,true),10_000,"con el modal de cobro abierto, cada 10 s (el límite del backend)");
  assert.equal(wake.nextPayWakeDelay(null,true),10_000,"sin respuesta se conserva el ritmo para no perder pagos");
  assert.equal(wake.nextPayWakeDelay(0,true),300_000,"un local sin teléfonos con push no genera tráfico continuo");
  const hook=read("src/modules/payment-devices/presentation/use-pay-device-wake.ts");
  assert.match(hook,/void tick\(\);/,"avisa al montar y al abrir o cerrar el cobro");
  assert.match(hook,/catch\{/,"un aviso fallido no interrumpe a Caja");
  assert.match(read("src/modules/payment-devices/infrastructure/pay-devices-api.ts"),/apiFetch<\{devices:number;sent:number\}>\("pay-devices\/wake",\{method:"POST"\}\)/);
  assert.match(read("src/modules/payment-devices/presentation/pay-notice-layer.tsx"),/usePayDeviceWake\(Boolean\(usePayCollection\(\)\.open\),leader\)/,"el aviso vive en la capa de pagos del cajero y acelera con el cobro abierto");
});

test("Fudia Pay: Yape y Plin suenan distinto y la pestaña parpadea mientras no tiene el foco",()=>{
  const alert=compile("src/modules/payment-devices/domain/pay-alert.ts");
  const melody=provider=>alert.payAlertTones[provider].map(tone=>tone.frequency).join(",");
  assert.notEqual(melody("yape"),melody("plin"),"cada billetera tiene su propio sonido");
  assert.equal(alert.payAlertTones.yape.length,3,"Yape: tres notas que suben");
  assert.ok(alert.payAlertTones.yape.every((tone,index,all)=>!index||tone.frequency>all[index-1].frequency));
  assert.equal(alert.payAlertTones.plin.length,2,"Plin: dos notas agudas");
  assert.ok(Math.min(...alert.payAlertTones.plin.map(tone=>tone.frequency))>Math.max(...alert.payAlertTones.yape.map(tone=>tone.frequency)),"Plin suena más agudo que Yape");
  assert.deepEqual([...alert.payAlertSequence(["yape","yape","plin","yape"])],["yape","plin"],"pagos juntos de la misma billetera suenan una vez");
  assert.equal(alert.payAlertTitle(1,{provider:"Yape",amount:"S/ 25.00"}),"Yape · S/ 25.00 recibido");
  assert.equal(alert.payAlertTitle(3,{provider:"Plin",amount:"S/ 5.00"}),"(3) Pagos recibidos");
  const hook=read("src/modules/payment-devices/presentation/use-pay-alert.ts");
  assert.match(hook,/addEventListener\("pointerdown",enableSound\)/,"el audio se habilita con el primer toque, como exige el navegador");
  assert.match(hook,/setSoundBlocked\(context\.state!=="running"\)/,"sabe si el navegador bloqueó el sonido");
  assert.match(hook,/if\(document\.hasFocus\(\)\)return;/,"con la pestaña a la vista solo suena: el aviso flotante ya se ve");
  assert.match(hook,/addEventListener\("focus",onFocus\)/,"el parpadeo termina al volver a la pestaña");
  assert.match(read("src/modules/payment-devices/presentation/pay-notice-layer.tsx"),/alert\(fresh\.map/,"solo los pagos nuevos suenan, no el historial inicial");
  assert.match(read("src/modules/payment-devices/presentation/pay-notice-layer.tsx"),/\},\[feed\.data,qc\]\);/,"el aviso no se repite en cada render");
});

test("Fudia Pay: el push sigue a tiempo con la pestaña oculta y el sonido bloqueado se ofrece activar",()=>{
  const timer=read("src/modules/payment-devices/presentation/background-timer.ts");
  assert.match(timer,/new Worker\(url\)/,"la espera corre en un Web Worker: Chrome no lo frena con la pestaña oculta");
  assert.match(timer,/window\.setTimeout/,"sin Worker conserva el temporizador de la página");
  const wake=read("src/modules/payment-devices/presentation/use-pay-device-wake.ts");
  assert.match(wake,/createBackgroundTimer\(\)/);
  assert.match(wake,/timer\.dispose\(\)/,"el worker se cierra al salir");
  const layer=read("src/modules/payment-devices/presentation/pay-notice-layer.tsx");
  assert.match(layer,/const askSound=leader&&soundBlocked&&\(pushDevices\?\?0\)>0;/,"solo se ofrece en locales que usan Fudia Pay");
  assert.match(layer,/<Button kind="secondary" icon="volume" className="pay-sound-gate" onClick=\{enableSound\}>Activar sonido de pagos<\/Button>/);
  assert.match(read("src/modules/payment-devices/presentation/payment-devices.css"),/\.pay-sound-gate\{[^}]*min-height:var\(--size-44\)/,"área táctil de 44 px");
});

test("Fudia Pay: con varias pestañas abiertas solo una avisa, la enfocada la reclama y otra la releva al cerrarse",async()=>{
  const exports={};
  vm.runInNewContext(ts.transpileModule(read("src/modules/payment-devices/presentation/pay-leader.ts"),{compilerOptions:{module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2020}}).outputText,{exports,require,AbortController,Promise});
  // LockManager mínimo con las reglas de Web Locks: exclusivo, en fila, `steal` y `signal`.
  const locks=(()=>{
    let holder=null;const queue=[];
    const grant=entry=>{holder=entry;Promise.resolve(entry.callback()).then(value=>{if(holder===entry){holder=null;entry.resolve(value);const next=queue.shift();if(next)grant(next)}})};
    return {request:(_name,options,callback)=>new Promise((resolve,reject)=>{
      const entry={callback,resolve,reject};
      options.signal?.addEventListener("abort",()=>{const index=queue.indexOf(entry);if(index>=0){queue.splice(index,1);reject(new Error("AbortError"))}});
      if(options.steal){if(holder){const previous=holder;holder=null;previous.reject(new Error("AbortError"))}grant(entry);return}
      if(holder)queue.push(entry);else grant(entry);
    })};
  })();
  const flush=()=>new Promise(resolve=>setTimeout(resolve,0));
  const state={};
  const tab=name=>exports.holdPayLeadership(locks,"fudia-pay-alert:caja",leader=>{state[name]=leader});
  const leaders=()=>Object.entries(state).filter(([,leader])=>leader).map(([name])=>name);
  const a=tab("a"),b=tab("b"),c=tab("c");
  await flush();
  assert.deepEqual(leaders(),["a"],"la primera pestaña avisa y las demás esperan");
  b.claim();await flush();
  assert.deepEqual(leaders(),["b"],"la pestaña que el cajero enfoca toma el aviso");
  b.claim();await flush();
  assert.deepEqual(leaders(),["b"],"reclamar de nuevo no cambia nada");
  b.dispose();state.b=false;await flush();
  assert.equal(leaders().length,1,"al cerrar la líder, otra la releva sola");
  const survivor=leaders()[0];
  ({a,c})[survivor].dispose();state[survivor]=false;await flush();
  assert.deepEqual(leaders(),[survivor==="a"?"c":"a"],"siempre queda una sola mientras haya pestañas");
  let fallback=null;
  exports.holdPayLeadership(undefined,"x",leader=>{fallback=leader});
  assert.equal(fallback,true,"sin Web Locks cada pestaña avisa, como antes");
  const layer=read("src/modules/payment-devices/presentation/pay-notice-layer.tsx");
  assert.match(layer,/usePayDeviceWake\(Boolean\(usePayCollection\(\)\.open\),leader\)/,"solo la líder pide el push");
  assert.match(layer,/if\(announces\)alert\(fresh/,"solo la líder suena y parpadea");
  assert.match(layer,/const askSound=leader&&soundBlocked/);
});
