import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import test from "node:test";

const read=file=>readFileSync(new URL(`../src/${file}`,import.meta.url),"utf8");

// La cabecera de un modal (icono y título) no depende del API: se conoce por la acción o por la fila del listado.
// Solo el cuerpo espera la respuesta; la cabecera en carga queda para el enlace directo sin datos previos.
test("los modales remotos muestran su cabecera real mientras cargan",()=>{
 const skeleton=read("design-system/remote-modal-skeleton.tsx");
 assert.match(skeleton,/\{title\?<ModalHeader title=\{title\} icon=\{icon\} onClose=/,"RemoteModalSkeleton conserva título e icono sin otra franja");
 assert.doesNotMatch(skeleton,/subtitle/);
 const calls=[
  "modules/customers/presentation/customers-manager.tsx",
  "modules/identity/presentation/users-roles-manager.tsx",
  "modules/menu/combos/presentation/combos-page.tsx",
  "modules/menu/products/presentation/catalog-manager.tsx",
  "modules/operations/cash/presentation/cash-closing-dialog.tsx",
  "modules/operations/cash/presentation/cash-page.tsx",
  "modules/organizations/presentation/organization-admin.tsx",
  "modules/supply/inventory/presentation/inventory-page.tsx",
  "modules/supply/purchases/presentation/purchases-page.tsx",
 ].flatMap(file=>read(file).split("<RemoteModalSkeleton ").slice(1).map(call=>[file,call.slice(0,call.indexOf("close="))]));
 assert.ok(calls.length>=12);
 for(const [file,call] of calls)assert.match(call,/ title=/,`${file}: ${call.slice(0,80)}`);
});

test("los detalles de venta, cobro, pedido y empresa usan la fila del listado para su cabecera",()=>{
 assert.match(read("modules/sales/presentation/sales-page.tsx"),/preview=\{sales\.data\?\.items\.find\(sale=>sale\.id===detailId\)\}/);
 assert.match(read("modules/operations/pos/presentation/pos-page.tsx"),/preview=\{items\.find\(item=>item\.id===detailId\)\}/);
 assert.match(read("modules/operations/orders/presentation/orders-manager.tsx"),/<OrderDetailSkeleton variant="order" close=\{closeDetail\} preview=/);
 assert.match(read("modules/operations/salon/presentation/salon-manager.tsx"),/<OrderDetailSkeleton variant="table" close=\{closeDetail\} preview=/);
 assert.match(read("modules/platform/presentation/platform-companies-page.tsx"),/previewName=\{data\.items\.find/);
 assert.match(read("modules/customers/presentation/customers-manager.tsx"),/function CustomerDetailSkeleton[\s\S]*?<ModalHeader title="Detalle del cliente"/,"el título del cliente es fijo y no espera al API");
 for(const file of ["modules/sales/presentation/sale-detail-dialog.tsx","modules/operations/pos/presentation/pos-page.tsx"])
  assert.match(read(file),/className="modal-summary is-band" aria-hidden="true"/,`${file}: el skeleton del resumen dibuja la franja`);
});
