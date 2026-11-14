import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url);
const read=path=>readFileSync(new URL(`../${path}`,import.meta.url),"utf8");
const routes={pos:"/pos",orders:"/orders",diningRoom:"/dining-room"};
function compile(path){
 const exports={};
 vm.runInNewContext(ts.transpileModule(read(path),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,{exports,require:name=>name==="@/shared/routing/page-routes"?{pageRoutes:routes}:require(name)});
 return exports;
}
const status=compile("src/modules/delivery/domain/dispatch-status.ts");
const rules=compile("src/modules/notifications/domain/notice-rules.ts");

test("Delivery Fudia: cada estado del despacho tiene etiqueta y tono suave de la matriz",()=>{
 assert.deepEqual(Object.keys(status.dispatchStatusMeta).sort(),["accepted","cancelled","delivered","own","picked_up","searching","unassigned"]);
 assert.equal(status.dispatchStatusMeta.delivered.label,"Entregado por la central");
 assert.equal(status.dispatchStatusMeta.picked_up.tone,"green","«Recogido» conserva su tono aprobado");
 assert.equal(status.dispatchStatusMeta.unassigned.label,"Sin central");
 assert.equal(status.dispatchStatusMeta.unassigned.tone,"orange");
 assert.equal(status.dispatchStatusMeta.searching.tone,"violet");
 assert.ok(!Object.values(status.dispatchStatusMeta).some(meta=>meta.tone==="red"),"sin rojo: ningún estado del despacho es un error");
 assert.equal(status.isDispatchStatus("accepted"),true);
 assert.equal(status.isDispatchStatus(""),false);
 assert.equal(status.isDispatchStatus(undefined),false);
});

test("Delivery Fudia: caja reintenta sin central o desde delivery propio, y deja de usar centrales antes de recoger",()=>{
 assert.deepEqual(["searching","accepted","unassigned","picked_up","delivered","own","cancelled"].filter(status.canRetryDispatch),["unassigned","own"]);
 assert.deepEqual(["searching","accepted","unassigned","picked_up","delivered","own","cancelled"].filter(status.canUseOwnDelivery),["searching","accepted","unassigned"]);
});

test("Delivery Fudia: «Ninguna central aceptó» abre el pedido en Pedidos con tono de atención",()=>{
 assert.deepEqual([...rules.noticeTargets({kind:"delivery_unassigned"})],[routes.orders]);
 assert.equal(rules.noticeHref(routes.orders,{kind:"delivery_unassigned",orderId:"o-1"}),"/orders?orderId=o-1");
 assert.equal(rules.noticeTone("delivery_unassigned"),"peach");
 assert.match(read("src/modules/notifications/presentation/notification-bell.tsx"),/delivery_unassigned:"motorbike"/);
});

test("Delivery Fudia: pantallas conectadas sin catálogos ni códigos en el frontend",()=>{
 const detail=read("src/modules/operations/orders/presentation/order-detail-dialog.tsx");
 assert.match(detail,/<OrderDispatchBlock order=\{order\}/,"el detalle del pedido muestra el despacho en el bloque de entrega");
 const section=read("src/modules/delivery/presentation/location-delivery-section.tsx");
 assert.ok(!/Tabs|PartnerSlots|getLocationDelivery/.test(section),"el local ya no elige modo ni centrales: son globales");
 assert.match(section,/publicConciergePath\(link\.data\.token\)/,"el enlace usa la misma ruta pública que el QR de mesa");
 const platform=read("src/modules/platform/presentation/platform-delivery-partners-page.tsx");
 assert.match(platform,/EmptyState icon="motorbike" title="Aún no hay centrales"/);
 assert.match(read("src/modules/platform/presentation/platform-shell.tsx"),/pageRoutes\.platformDeliveryPartners/);
 // Orden global de centrales y activación por empresa desde Plataforma.
 assert.match(platform,/reorderDeliveryPartners/);
 assert.match(platform,/label="Subir en el orden"/);
 assert.match(read("src/modules/platform/presentation/platform-company-detail-page.tsx"),/<DeliveryCompanyPanel organizationId=\{organizationId\}\/>/);
 const panel=read("src/modules/platform/presentation/platform-delivery-company-panel.tsx");
 assert.match(panel,/setOrganizationDelivery\(organizationId,enabled\)/);
});
