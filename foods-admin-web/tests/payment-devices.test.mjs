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

test("Fudia Pay: la vinculación usa el código temporal por QR y vigila los dispositivos",()=>{
  const dialog=read("src/modules/payment-devices/presentation/pay-pairing-dialog.tsx");
  assert.match(dialog,/createPayPairing\(\)/,"el diálogo genera el código temporal");
  assert.match(dialog,/toCanvas\(canvas\.current,draft\.code/,"el QR se dibuja con el código, no con credenciales");
  assert.match(dialog,/refetchInterval:query=>query\.state\.data\?\.items\.some\(item=>Date\.parse\(item\.createdAt\)>=issuedAt-1000\)\?false:3000/,"vigila los dispositivos cada 3 s hasta que aparece el nuevo");
  assert.doesNotMatch(dialog,/token/i,"el QR y la pantalla nunca muestran la credencial del dispositivo");
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

test("Fudia Pay: Caja integra el tab «Pagos detectados» con dispositivos y revisión",()=>{
  const cash=read("src/modules/operations/cash/presentation/cash-page.tsx");
  assert.match(cash,/value:"pagos",label:"Pagos detectados",icon:"wallet"/);
  assert.match(cash,/PayDevicesPanel dateTime=\{dateTime\}\/><PayNotificationsPanel dateTime=\{dateTime\}/);
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
