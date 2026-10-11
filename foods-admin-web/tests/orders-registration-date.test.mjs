import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {detailComponents,renderDetail,exampleOrder} from "./helpers/order-detail-fixture.mjs";
const {registeredAt}=detailComponents();
const expected=value=>new Intl.DateTimeFormat("es-PE",{timeZone:"America/Lima",dateStyle:"medium",timeStyle:"short"}).format(new Date(value));
const read=file=>readFileSync(new URL("../src/"+file,import.meta.url),"utf8");

test("Registrado conserva fecha y hora del local y no usa tiempo relativo",()=>{
 for(const value of ["2026-10-07T14:30:00Z","2020-01-01T05:00:00Z","2099-01-01T04:30:00Z"]){
  const label=registeredAt(value,"PE","America/Lima");
  assert.equal(label,expected(value));
  assert.doesNotMatch(label,/Hace|Justo ahora/);
 }
 assert.equal(registeredAt("invalid","PE","America/Lima"),"—");
});
test("la tabla y el detalle reutilizan la fecha del local",()=>{
 const source=read("modules/operations/orders/presentation/orders-manager.tsx");
 assert.match(source,/registeredAt\(o\.createdAt,location\?\.country,location\?\.timezone\)/);
 const html=renderDetail({order:exampleOrder,variant:"order"});
 assert.match(html,/2026-10-09T18:53:00-05:00/);
 assert.match(html,/oct/);
});
test("la fecha móvil permanece en la vista responsiva",()=>{
 const css=read("modules/operations/styles/orders.css");
 assert.match(css,/\.order-mobile-registered/);
 assert.match(css,/white-space:normal/);
});
