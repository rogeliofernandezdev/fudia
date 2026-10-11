import test from "node:test";
import assert from "node:assert/strict";
import {renderDetail,exampleOrder,detailComponents} from "./helpers/order-detail-fixture.mjs";
const {canManageOrderService}=detailComponents();
const served={...exampleOrder,status:"entregado",billClosedAt:"2026-10-11",serviceItems:exampleOrder.serviceItems.map(i=>({...i,status:"entregado"})),remainingAmount:"30"};
test("la atención corresponde al mozo asignado",()=>{
 assert.equal(canManageOrderService(exampleOrder,"preview-waiter"),true);
 assert.equal(canManageOrderService(exampleOrder,"different"),false);
 assert.equal(canManageOrderService(exampleOrder,undefined),false);
 assert.equal(canManageOrderService({...exampleOrder,channel:"delivery"},"different"),true);
});
test("otro mozo ve responsables pero no realiza acciones",()=>{
 const html=renderDetail({userId:"different"});
 assert.match(html,/Cielo Fernandez/);
 assert.doesNotMatch(html,/Agregar productos|Cerrar cuenta|Enviar comanda|Cancelar pedido/);
 assert.match(html,/Cerrar detalle/);
});
test("el mozo asignado mantiene acciones si tiene permiso",()=>{
 const draft={...exampleOrder,status:"nuevo",serviceItems:[]};
 assert.match(renderDetail({order:draft,userId:"preview-waiter"}),/Enviar comanda/);
 assert.doesNotMatch(renderDetail({order:draft,permissions:[]}),/Enviar comanda/);
});
test("cajero cobra sin poder editar comanda ajena",()=>{
 const html=renderDetail({order:served,userId:"cashier",permissions:["cash.manage"]});
 assert.match(html,/Cobrar saldo/);
 assert.doesNotMatch(html,/Agregar productos|Cerrar cuenta|Editar comanda|Confirmar entrega/);
});
test("atribución distingue mozo y personal que cobró",()=>{
 const html=renderDetail({order:{...exampleOrder,collectedByNames:["Eva","Pablo"]}});
 assert.match(html,/Cielo Fernandez/);
 assert.match(html,/Eva/);
 assert.match(html,/Pablo/);
 assert.doesNotMatch(html,/preview-waiter/);
});
