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
  assert.match(dialog,/createPayPairing\(\)/,"el diálogo genera el código temporal");
  assert.match(dialog,/toCanvas\(canvas\.current,draft\.code/,"el QR se dibuja con el código, no con credenciales");
  assert.match(dialog,/getPayPairingStatus\(draft!\.id\)/,"sigue el estado del código mostrado");
  assert.match(dialog,/\["linked","expired"\]\.includes\(query\.state\.data\?\.status\?\?""\)\?false:STATUS_POLL_MS/,"consulta hasta que se vincula o vence");
  assert.match(dialog,/Vinculando\{status\.data\?\.model/,"al escanear el QR cede su lugar al progreso con el modelo");
  assert.match(dialog,/<Spinner size=\{40\}\/>/);
  assert.match(dialog,/onLinked\(status\.data\?\.deviceName/,"al vincular avisa al panel, que cierra el modal");
  assert.match(dialog,/Generar otro código/,"un código vencido se renueva sin cerrar el modal");
  assert.doesNotMatch(dialog,/token/i,"el QR y la pantalla nunca muestran la credencial del dispositivo");
  const panel=read("src/modules/payment-devices/presentation/pay-devices-panel.tsx");
  assert.match(panel,/\{pairing&&<PayPairingDialog onClose=\{closePairing\} onLinked=\{linked\}\/>\}/,"el diálogo se monta solo al abrirlo y cierra con la X, Cancelar o al vincular");
  assert.match(panel,/title:"Dispositivo vinculado"/);
  assert.match(read("src/modules/payment-devices/infrastructure/pay-devices-api.ts"),/pay-devices\/pairing\/\$\{encodeURIComponent\(id\)\}/);
});

test("Fudia Pay: los endpoints del dispositivo y de Caja siguen el contrato",()=>{
  const api=read("src/modules/payment-devices/infrastructure/pay-devices-api.ts");
  assert.match(api,/apiFetch<PayPairingDraft>\("pay-devices\/pairing",\{method:"POST"\}\)/);
  assert.match(api,/pay-devices\/\$\{encodeURIComponent\(id\)\}\/revoke/);
  assert.match(api,/associatePayNotification\(id:string,orderId:string,method:string,amount\?:number\)/,"asociar registra el cobro real y admite el registro parcial");
  assert.match(api,/pos\/orders\?\$\{params\.toString\(\)\}/,"la búsqueda de pedidos usa el listado del POS");
  assert.doesNotMatch(api,/confirmPayNotification/,"la asociación cierra el flujo: no existe confirmación separada");
});

test("Fudia Pay: aviso flotante global y modal de detalle con asociación revisada",()=>{
  const layer=read("src/modules/payment-devices/presentation/pay-notice-layer.tsx");
  assert.match(layer,/refetchInterval:10000/,"la capa sondea los pagos detectados");
  assert.match(layer,/PayNotificationDetail notification=\{detail\}/,"el clic abre el detalle");
  assert.match(layer,/position:fixed|pay-notice-layer/,"la capa es flotante");
  const detail=read("src/modules/payment-devices/presentation/pay-notification-detail.tsx");
  assert.match(detail,/usePayCollection\(\)/,"usa el pedido en cobro como sugerencia");
  assert.match(detail,/la asociación nunca es automática/,"el pedido abierto es sugerencia, no selección");
  assert.match(detail,/Se registrará solo el saldo/,"un pago superior al saldo se registra parcial");
  assert.match(detail,/Volver a pedidos/,"permite regresar a la lista antes de confirmar");
  const css=read("src/modules/payment-devices/presentation/payment-devices.css");
  assert.match(css,/\.pay-notice-layer\{position:fixed/);
  assert.match(css,/z-index:var\(--z-tooltip\)/,"la capa queda por encima de los modales sin cerrarlos");
});

test("Fudia Pay: «Pagos detectados» es una página propia con pestañas Pagos y Dispositivos",()=>{
  const cash=read("src/modules/operations/cash/presentation/cash-page.tsx");
  assert.doesNotMatch(cash,/Pagos detectados|payment-devices/,"Caja ya no aloja los pagos detectados");
  const page=read("src/modules/payment-devices/presentation/detected-payments-page.tsx");
  assert.match(page,/<PageHeader title="Pagos detectados"/);
  assert.match(page,/\{value:"payments",label:"Pagos",icon:"wallet"/);
  assert.match(page,/\{value:"devices",label:"Dispositivos",icon:"phoneApp"/);
  assert.match(page,/<TabsBar action=\{<Button icon="plus" onClick=\{\(\)=>\{setTab\("devices"\);setPairing\(true\)\}\}>Vincular dispositivo<\/Button>\}>/);
  assert.match(read("src/app/(admin)/detected-payments/page.tsx"),/<DetectedPaymentsPage\/>/);
  const nav=read("src/shell/navigation.ts");
  assert.match(nav,/\{href:pageRoutes\.detectedPayments,name:"Pagos detectados",icon:"wallet",module:"caja",access:"pagos_detectados",permission:"cash\.read"\}/,"casilla propia en roles, mismo módulo y permiso que Caja");
  assert.match(read("src/shared/routing/page-routes.ts"),/detectedPayments:"\/detected-payments"/);
  const view=read("src/modules/payment-devices/presentation/pay-notifications-panel.tsx");
  assert.match(view,/refetchInterval:query=>query\.state\.data\?\.items\.some\(item=>item\.status==="detected"\)\?10000:30000/,"los pagos por revisar se consultan con más frecuencia");
  assert.match(view,/La asociación registra el cobro real/);
  assert.match(view,/Asociar el pago a un pedido/);
  const pos=read("src/modules/operations/pos/presentation/pos-page.tsx");
  assert.match(pos,/PayCollectionRegistration order=/,"el POS registra el pedido en cobro como sugerencia");
  const providers=read("src/providers/providers.tsx");
  assert.match(providers,/PayCollectionProvider/,"la sugerencia vive en contexto global");
  assert.match(providers,/PayNoticeLayer/,"el aviso flotante está en toda la aplicación");
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
