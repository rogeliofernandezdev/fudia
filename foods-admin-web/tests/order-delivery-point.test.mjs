import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {renderDetail,exampleOrder} from "./helpers/order-detail-fixture.mjs";

const read=path=>readFileSync(new URL("../src/"+path,import.meta.url),"utf8");
const delivery={...exampleOrder,channel:"delivery",tableId:"",tableName:"",customerName:"Ana Quispe",customerPhone:"999888777",address:"Av. Arequipa 1234, Lince",reference:"Frente al parque",deliveryFee:"5",
 deliveryLatitude:-12.0853,deliveryLongitude:-77.0347,deliveryDistanceKm:2.35};

test("delivery con pin muestra Ver ubicación hacia Google Maps y la distancia",()=>{
 const html=renderDetail({order:delivery});
 assert.match(html,/href="https:\/\/www\.google\.com\/maps\/search\/\?api=1&amp;query=-12\.0853%2C-77\.0347"/);
 assert.match(html,/target="_blank" rel="noopener noreferrer"/);
 assert.match(html,/>Ver ubicación<\/a>/);
 assert.match(html,/class="order-detail-map-distance">2\.4 km</);
 assert.ok(html.indexOf("Frente al parque")<html.indexOf("Ver ubicación"),"el enlace va después de la referencia");
});

test("sin coordenadas o fuera de delivery no se muestra el enlace",()=>{
 for(const order of [
  {...delivery,deliveryLatitude:null,deliveryLongitude:null,deliveryDistanceKm:null},
  {...delivery,deliveryLatitude:undefined,deliveryLongitude:undefined,deliveryDistanceKm:undefined},
  {...delivery,deliveryLatitude:-12.0853,deliveryLongitude:null},
  {...delivery,channel:"recojo"},
 ]){
  const html=renderDetail({order});
  assert.doesNotMatch(html,/Ver ubicación|google\.com\/maps/);
 }
});

test("sin distancia cotizada se muestra solo el enlace",()=>{
 const html=renderDetail({order:{...delivery,deliveryDistanceKm:null}});
 assert.match(html,/>Ver ubicación<\/a>/);
 assert.doesNotMatch(html,/order-detail-map-distance/);
});

test("el pin basta para mostrar el bloque de entrega aunque falte la dirección",()=>{
 const html=renderDetail({order:{...delivery,customerPhone:"",address:"",reference:""}});
 assert.match(html,/Dirección no registrada/);
 assert.match(html,/>Ver ubicación<\/a>/);
});

test("el enlace es tipográfico, con área táctil de 44 px y la URL se construye en dominio",()=>{
 const css=read("modules/operations/orders/presentation/order-detail.css");
 const rule=css.match(/\.order-detail-map-link\{[^}]+\}/)?.[0]??"";
 assert.match(rule,/min-height:var\(--size-44\)/);
 assert.match(rule,/text-decoration:underline/);
 assert.match(rule,/color:var\(--ink-950\)/);
 assert.doesNotMatch(rule,/background|primary-|border:/);
 assert.doesNotMatch(read("modules/operations/orders/presentation/order-detail-dialog.tsx"),/google\.com\/maps/);
 assert.match(read("modules/operations/orders/domain/delivery-point.ts"),/google\.com\/maps\/search\/\?api=1/);
});
