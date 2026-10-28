import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {renderDetail,exampleOrder,detailComponents} from "./helpers/order-detail-fixture.mjs";

const {nextOrderAction,canCancelOrder,canManageOrderService,canChargeAccount,canCloseAccount,canAddConsumption,orderStatus}=detailComponents();
const read=path=>readFileSync(new URL("../src/"+path,import.meta.url),"utf8");
const ready={...exampleOrder,status:"listo",serviceItems:exampleOrder.serviceItems.map(i=>({...i,status:"listo"}))};
const served={...exampleOrder,status:"entregado",serviceItems:exampleOrder.serviceItems.map(i=>({...i,status:"entregado"}))};

test("entrega no depende del pago ni requiere una segunda transición",()=>{
 for(const p of ["pending","partial","paid"]){
  assert.equal(nextOrderAction({...ready,paymentStatus:p}).status,"entregado");
  assert.equal(nextOrderAction({...served,paymentStatus:p}),null);
 }
});
test("cocina prepara y delivery despacha",()=>{
 for(const status of ["confirmado","preparando","cancelado"])assert.equal(nextOrderAction({...ready,status}),null);
 assert.equal(nextOrderAction({...ready,channel:"delivery"}).status,"en_camino");
 assert.equal(nextOrderAction({...ready,channel:"delivery",status:"en_camino"}).status,"entregado");
});
test("cobro requiere entrega y cierre efectivo",()=>{
 assert.equal(canChargeAccount(ready),false);
 assert.equal(canChargeAccount(served),false);
 assert.equal(canChargeAccount({...served,billClosedAt:"2026-10-11"}),true);
 assert.equal(canChargeAccount({...served,billClosedAt:"2026-10-11",completedAt:"2026-10-11"}),false);
});
test("cierre exige todas las preparaciones entregadas",()=>{
 assert.equal(canCloseAccount(ready),false);
 assert.equal(canCloseAccount(served),true);
 assert.equal(canCloseAccount({...served,billClosedAt:"2026-10-11"}),false);
 assert.equal(canCloseAccount({...served,serviceItems:[{status:"listo"}]}),false);
});
test("el cajero tiene un enlace de cobro solo con saldo y permiso",()=>{
 const closed={...served,billClosedAt:"2026-10-11",remainingAmount:"30"};
 assert.match(renderDetail({order:closed,permissions:["cash.manage"]}),/Cobrar saldo/);
 assert.doesNotMatch(renderDetail({order:closed,permissions:[]}),/Cobrar saldo/);
 assert.doesNotMatch(renderDetail({order:{...closed,remainingAmount:"0"},permissions:["cash.manage"]}),/Cobrar saldo/);
});

test("el colaborador cobra desde Mesa y Pedidos usando el mismo modal de POS sin cash.manage",()=>{
 const closed={...served,billClosedAt:"2026-10-11",remainingAmount:"30"};
 for(const variant of ["table","order"]){
  const html=renderDetail({order:closed,variant,permissions:["orders.read"],userId:"other",cashCollection:true});
  assert.match(html,/Cobrar saldo/);
  assert.ok(html.includes(`href="/pos?orderId=${closed.id}"`));
  assert.doesNotMatch(html,/Editar comanda|Cancelar pedido|Agregar productos/);
  assert.doesNotMatch(renderDetail({order:closed,variant,permissions:["orders.read"],cashCollection:false}),/Cobrar saldo/);
 }
});

test("la autorización temporal no permite cobrar cuentas abiertas, pendientes de entrega o pagadas",()=>{
 const closed={...served,billClosedAt:"2026-10-11",remainingAmount:"30"};
 for(const order of [served,ready,{...closed,remainingAmount:"0"},{...closed,completedAt:"2026-10-11"},{...closed,serviceItems:[{status:"listo"}]}]){
  assert.doesNotMatch(renderDetail({order,variant:"table",permissions:[],cashCollection:true}),/Cobrar saldo/);
 }
});
test("mozos ajenos no administran entrega ni cobro",()=>{
 assert.equal(canManageOrderService(ready,"preview-waiter"),true);
 assert.equal(canManageOrderService(ready,"other"),false);
 assert.doesNotMatch(renderDetail({order:ready,userId:"other"}),/Confirmar entrega|Agregar productos|Cobrar saldo/);
});
test("un pago existente impide cancelar",()=>{
 assert.equal(canCancelOrder({...ready,paidAmount:"1"}),false);
 assert.equal(canCancelOrder({...exampleOrder,status:"nuevo",paidAmount:"0",serviceItems:[]}),true);
});
test("los botones se bloquean durante la mutación",()=>{
 const html=renderDetail({order:ready,busy:true});
 assert.match(html,/disabled=""/);
 assert.match(html,/aria-busy="true"/);
});
test("la preparación y el saldo se muestran sin mezclarse",()=>{
 assert.equal(orderStatus("listo").tone,"green");
 assert.equal(orderStatus("entregado").tone,"teal");
 const html=renderDetail({order:ready});
 assert.match(html,/Por entregar/);
 assert.match(html,/Saldo pendiente/);
 assert.match(html,/Confirmar entrega/);
});
test("las cantidades y notas no filtran el identificador interno",()=>{
 const html=renderDetail({order:ready});
 assert.match(html,/Caldo de Gallina/);
 assert.match(html,/Jarra de Chicha/);
 assert.match(html,/order-detail-qty/);
 assert.doesNotMatch(html,/PED-004/);
});
test("Salón y Pedidos reutilizan el mismo detalle",()=>{
 const salon=read("modules/operations/salon/presentation/salon-manager.tsx");
 const orders=read("modules/operations/orders/presentation/orders-manager.tsx");
 assert.match(salon,/<OrderDetailContent/);
 assert.match(orders,/<OrderDetailContent/);
 assert.doesNotMatch(salon,/function OrderDetail\(/);
 assert.doesNotMatch(orders,/function OrderDetail\(/);
});
test("las vistas refrescan estado operativo hasta completar",()=>{
 for(const path of ["modules/operations/salon/presentation/salon-manager.tsx","modules/operations/orders/presentation/orders-manager.tsx"]){
  assert.match(read(path),/refetchInterval:query=>query\.state\.data\?\.completedAt\?false:10000/);
 }
});
test("agregar productos se impide tras cierre o finalización",()=>{
 assert.equal(canAddConsumption({...served,billClosedAt:"2026-10-11"}),false);
 assert.equal(canAddConsumption({...served,completedAt:"2026-10-11"}),false);
 assert.equal(canAddConsumption({...ready,billClosedAt:undefined}),true);
});
test("el resumen, carga, errores y estados vacíos están disponibles",()=>{
 assert.match(renderDetail({order:ready}),/Resumen de cuenta/);
 assert.match(renderDetail({order:served}),/Cerrar cuenta/);
 assert.match(renderDetail({skeleton:true}),/ds-skeleton/);
 assert.match(renderDetail({order:{...ready,items:[]}}),/Sin ítems cargados aún/);
 assert.match(renderDetail({error:"Sin conexión"}),/No pudimos cargar el pedido/);
});
test("cuenta finalizada no puede cobrarse de nuevo",()=>{
 const closed={...served,billClosedAt:"2026-10-11",accountState:"paid",paidAmount:"30",remainingAmount:"0",completedAt:"2026-10-11"};
 const html=renderDetail({order:closed,permissions:["cash.manage"]});
 assert.match(html,/Total pagado/);
 assert.doesNotMatch(html,/Cobrar saldo|Cerrar cuenta|Confirmar entrega/);
});
