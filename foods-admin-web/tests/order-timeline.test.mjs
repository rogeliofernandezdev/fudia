import test from "node:test";
import assert from "node:assert/strict";
import {renderDetail,exampleOrder} from "./helpers/order-detail-fixture.mjs";

const steps=html=>[...html.matchAll(/<li class="is-(done|current|pending)">.*?<b>([^<]+)<\/b>/g)].map(match=>`${match[1]}:${match[2]}`);

test("seguimiento de mesa: alta con hora, preparación actual y cierres pendientes",()=>{
 const html=renderDetail({order:{...exampleOrder,status:"preparando",billClosedAt:undefined,completedAt:undefined,paymentStatus:"pending"}});
 assert.match(html,/id="order-timeline-title" class="order-timeline-title">Progreso del pedido</);
 assert.deepEqual(steps(html),["done:Pedido registrado","current:En preparación","pending:Cuenta cerrada","pending:Pago","pending:Mesa liberada"]);
 assert.match(html,/<time dateTime="[^"]+">/,"el alta muestra su hora real");
});

test("seguimiento usa solo horas reales: cierre de cuenta y liberación con hora, pago sin hora inventada",()=>{
 const html=renderDetail({order:{...exampleOrder,status:"entregado",billClosedAt:"2026-10-09T20:10:00-05:00",completedAt:"2026-10-09T20:30:00-05:00",paymentStatus:"paid"}});
 assert.deepEqual(steps(html),["done:Pedido registrado","done:Entregado","done:Cuenta cerrada","done:Pago completo","done:Mesa liberada"]);
 assert.equal((html.match(/<time dateTime=/g)??[]).length>=3,true);
 assert.match(html,/<b>Pago completo<\/b><small>Registrado<\/small>/);
});

test("delivery no muestra cierre de cuenta y un pedido cancelado corta el seguimiento",()=>{
 const delivery=renderDetail({order:{...exampleOrder,channel:"delivery",tableId:"",tableName:"",status:"listo",paymentStatus:"partial"}});
 assert.deepEqual(steps(delivery),["done:Pedido registrado","current:Listo para entregar","current:Pago parcial","pending:Pedido completado"]);
 const cancelled=renderDetail({order:{...exampleOrder,status:"cancelado"}});
 assert.deepEqual(steps(cancelled),["done:Pedido registrado","done:Pedido cancelado"]);
});
