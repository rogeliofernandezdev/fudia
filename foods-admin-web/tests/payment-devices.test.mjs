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
  assert.match(panel,/\{pairing&&<PayPairingDialog locationId=\{locationId\} locationName=\{locationName\} onClose=\{closePairing\} onLinked=\{linked\}\/>\}/,"el diálogo se monta solo al abrirlo, ligado al local, y cierra con la X, Cancelar o al vincular");
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
  assert.match(layer,/refetchInterval:10000/,"sondea cada 10 s con sesión y sigue pese a fallos pasajeros del API");
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
  assert.doesNotMatch(detail,/<Select/,"el logo ya identifica el medio de pago: sin selector bloqueado");
  assert.match(detail,/className="pay-order-option" aria-label=\{`Revisar el pedido \$\{code\}`\}/,"cada fila completa es el área táctil para revisar el pedido");
  assert.match(detail,/Monto exacto/,"marca los pedidos cuyo monto coincide con el pago");
  const css=read("src/modules/payment-devices/presentation/payment-devices.css");
  assert.match(css,/\.pay-notice-layer\{position:fixed/);
  assert.match(css,/\.pay-provider-tag\.plin\.is-small \.pay-provider-logo\{width:var\(--size-24\)/,"Plin llena su lienzo: se reduce para igualar el ancho visible de Yape");
  assert.match(css,/\.pay-provider-tag\.yape\.is-small \.pay-provider-logo\{width:var\(--size-48\)/,"Yape se amplía para compensar sus márgenes blancos");
  assert.match(css,/z-index:var\(--z-tooltip\)/,"la capa queda por encima de los modales sin cerrarlos");
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
  assert.match(view,/refetchInterval:query=>query\.state\.data\?\.items\.some\(item=>item\.status==="detected"\)\?10000:30000/,"los pagos por revisar se consultan con más frecuencia");
  assert.match(view,/openPayment\(item,"detail"\)/,"el ojo abre la consulta del pago");
  assert.match(view,/openPayment\(item,"associate"\)/,"el enlace abre directamente la tarea de asociación");
  assert.match(view,/mode=\{detailMode\}/,"el modal recibe la intención elegida");
  const detail=read("src/modules/payment-devices/presentation/pay-notification-detail.tsx");
  assert.match(detail,/La asociación registra el cobro real/);
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
  assert.match(payments,/listPayNotificationPage\(\{status,provider,page,size\}\)/);
  assert.match(payments,/<Pagination page=\{page\} size=\{size\} total=\{notifications\.data\?\.total\?\?0\}/);
  assert.match(payments,/setStatus\(event\.target\.value\);setPage\(1\)/,"cambiar un filtro vuelve a la primera página");
  assert.match(read("src/modules/payment-devices/infrastructure/pay-devices-api.ts"),/page:String\(filter\.page\),size:String\(filter\.size\)/);
  const css=read("src/modules/payment-devices/presentation/payment-devices.css");
  assert.doesNotMatch(css,/\.pay-devices-table th|\.pay-devices-table td\{/,"las filas y cabeceras las pone data-table.css");
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
  assert.match(read("src/modules/payment-devices/infrastructure/pay-devices-api.ts"),/status:filter\.status,page:String\(filter\.page\),size:String\(filter\.size\)/);
});
