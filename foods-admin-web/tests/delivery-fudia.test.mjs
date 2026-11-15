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
 assert.deepEqual([...rules.noticeTargets({kind:"delivery_pickup_delayed"})],[routes.orders]);
 assert.equal(rules.noticeTone("delivery_pickup_delayed"),"peach");
 assert.match(read("src/modules/notifications/presentation/notification-bell.tsx"),/delivery_pickup_delayed:"clock"/);
});

test("Delivery Fudia: pantallas conectadas sin catálogos ni códigos en el frontend",()=>{
 const detail=read("src/modules/operations/orders/presentation/order-detail-dialog.tsx");
 assert.ok(detail.includes('<OrderDispatchBlock key={order.id} order={order} money={money}/>'));
 const block=read("src/modules/delivery/presentation/order-dispatch-block.tsx");
 assert.ok(block.includes('getDeliveryDispatch(order.id)'));
 assert.doesNotMatch(block,/retryDeliveryDispatch|switchToOwnDelivery|ConfirmDialog/,"el detalle es solo seguimiento");
 const actions=read("src/modules/delivery/presentation/order-dispatch-actions.tsx");
 assert.ok(actions.includes('retryDeliveryDispatch(order.id)'));
 assert.ok(actions.includes('switchToOwnDelivery(order.id)'));
 assert.ok(actions.includes('can("orders.manage")'));
 assert.ok(actions.includes('createPortal(<ConfirmDialog'));
 assert.ok(actions.includes('event.stopPropagation()'),"las acciones no abren el detalle de la fila");
 assert.doesNotMatch(actions,/useQuery\(|getDeliveryDispatch|getOrder\(/,"sin una lectura extra por fila");
 const manager=read("src/modules/operations/orders/presentation/orders-manager.tsx");
 assert.ok(manager.includes('<OrderDispatchActions order={o}'));
 assert.doesNotMatch(block,/Ver en Delivery|deliveryTabPath/);
 for(const key of ['"order"','"orders"','"pos-order"','"pos-orders"','"staff-notices"'])assert.ok(actions.includes(key));
 const section=read("src/modules/delivery/presentation/location-delivery-section.tsx");
 assert.ok(!/Tabs|PartnerSlots|getLocationDelivery/.test(section),"el local ya no elige modo ni centrales: son globales");
 assert.match(section,/publicConciergePath\(link\.data\.token\)/,"el enlace usa la misma ruta pública que el QR de mesa");
 const platform=read("src/modules/platform/presentation/delivery-partners-admin.tsx");
 assert.match(platform,/EmptyState icon="motorbike" title="Aún no hay centrales"/);
 assert.doesNotMatch(read("src/modules/platform/presentation/platform-shell.tsx"),/Centrales de delivery/,"las centrales salen del menú de Plataforma");
 assert.match(read("src/modules/delivery/presentation/delivery-providers.tsx"),/if\(user\?\.platformAdmin\)return <DeliveryPartnersAdmin\/>/,"Plataforma las administra en Delivery › Proveedores");
 // Orden global de centrales y activación por empresa desde Plataforma.
 assert.match(platform,/reorderDeliveryPartners/);
 assert.match(platform,/label="Subir en el orden"/);
 assert.match(read("src/modules/platform/presentation/platform-company-detail-page.tsx"),/<DeliveryCompanyPanel organizationId=\{organizationId\}\/>/);
 const panel=read("src/modules/platform/presentation/platform-delivery-company-panel.tsx");
 assert.match(panel,/setOrganizationDelivery\(organizationId,enabled\)/);
});

const board=compile("src/modules/delivery/domain/board.ts");
const plain=value=>JSON.parse(JSON.stringify(value));
const boardOrder=(overrides={})=>({status:"preparando",failedDelivery:false,dispatchStatus:"accepted",pickupWaitSeconds:0,pickupAlert:"",...overrides});

test("Delivery: una sola opción del menú con pestañas; las de módulos en desarrollo dependen del módulo",()=>{
 assert.deepEqual(plain(board.visibleDeliveryTabs({delivery:true}).map(tab=>tab.value)),["providers","fees","failed"]);
 assert.deepEqual(plain(board.visibleDeliveryTabs({delivery:true,delivery_apps:true,repartidores:true}).map(tab=>tab.value)),["providers","fees","failed","apps","couriers"]);
 assert.equal(board.deliveryTabFrom("fees",{delivery:true}),"fees");
 assert.equal(board.deliveryTabFrom("apps",{delivery:true}),"providers","una pestaña de un módulo inactivo vuelve a Proveedores");
 assert.equal(board.deliveryTabFrom("nope",null),"providers");
 assert.equal(board.deliveryTabFrom("orders",null),"providers");
 assert.ok(read("src/app/(admin)/delivery/page.tsx").includes('if(params.tab==="orders")redirect(pageRoutes.orders)'));
 assert.doesNotMatch(read("src/modules/delivery/presentation/delivery-page.tsx"),/DeliveryBoard/);
 assert.equal(board.deliveryTabPath("providers"),"/delivery?tab=providers");
 for(const tab of board.deliveryTabs)assert.match(tab.value,/^[a-z]+$/,"la pestaña viaja en inglés en la URL");
 const navigation=read("src/shell/navigation.ts");
 assert.match(navigation,/\{label:"DELIVERY",items:\[\{href:pageRoutes\.delivery,name:"Delivery",icon:"truck",module:"delivery"\}\]\}/);
 assert.doesNotMatch(navigation,/Delivery propio|Apps de delivery|App repartidores/);
 const routes=read("src/shared/routing/page-routes.ts");
 for(const [legacy,tab] of [["/delivery-apps","apps"],["/couriers","couriers"],["/repartidores","couriers"],["/platform/delivery-partners","providers"]])assert.ok(routes.includes(`{legacy:"${legacy}",canonical:"/delivery?tab=${tab}"}`),legacy);
 assert.match(read("src/app/(admin)/delivery/page.tsx"),/<DeliveryPage initialTab=\{params\.tab\}\/>/);
});

test("Delivery › Pedidos: en curso, modo, alertas de recogida e incidencias",()=>{
 const orders=[boardOrder(),boardOrder({status:"entregado"}),boardOrder({status:"en_camino",failedDelivery:true}),boardOrder({status:"cancelado"})];
 assert.equal(board.filterBoard(orders,"active").length,1);
 assert.equal(board.filterBoard(orders,"today").length,4);
 assert.equal(board.deliveryMode({dispatchStatus:""}),"Delivery propio");
 assert.equal(board.deliveryMode({dispatchStatus:"own"}),"Delivery propio");
 assert.equal(board.deliveryMode({dispatchStatus:"picked_up"}),"Delivery Fudia");
 assert.deepEqual(Object.keys(board.pickupAlertLabels),["warning","urgent","critical"]);
 assert.equal(board.pickupAlertLabels.critical.tone,"red");
 assert.equal(board.waitMinutes(545),"9 min");
 assert.equal(board.deliveryIssue({failedDelivery:true,issueReportedAt:"x"}),"Entrega fallida");
 assert.equal(board.deliveryIssue({failedDelivery:false,issueReportedAt:"x"}),"No pudo entregar");
 assert.equal(board.deliveryIssue({failedDelivery:false}),"");
 assert.equal(board.formatPartnerPhone("51987654321"),"+51 987 654 321");
 assert.equal(board.formatPartnerPhone(""),"");
});

test("Delivery: cada pestaña con carga, error y vacío; la central se edita solo desde Plataforma",()=>{
 for(const [file,fragments] of Object.entries({
  "order-dispatch-block.tsx":["Skeleton","ErrorState","getDeliveryDispatch"],
  "delivery-providers.tsx":["TableSkeleton","ErrorState","EmptyState","getDeliveryProviders"],
  "failed-deliveries.tsx":["TableSkeleton","ErrorState","EmptyState","getLocationFailedDeliveries","Con aporte Fudia"],
 })){
  const source=read(`src/modules/delivery/presentation/${file}`);
  for(const fragment of fragments)assert.ok(source.includes(fragment),`${file}: ${fragment}`);
 }
 const api=read("src/modules/delivery/infrastructure/delivery-api.ts");
 for(const path of ['"delivery/orders"','"delivery/partners"','"delivery/failed-deliveries"'])assert.ok(api.includes(path),path);
 assert.doesNotMatch(read("src/modules/delivery/presentation/delivery-providers.tsx"),/saveDeliveryPartner|reorderDeliveryPartners/,"el restaurante solo consulta sus centrales");
});
