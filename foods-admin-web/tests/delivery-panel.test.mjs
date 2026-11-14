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
 vm.runInNewContext(ts.transpileModule(read(path),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,{exports,require,URLSearchParams,Date,Math});
 return exports;
}
const panel=compile("src/modules/delivery-panel/domain/panel.ts");
const tracking=compile("src/modules/concierge-web/domain/tracking.ts");
const order=(overrides={})=>({token:"t1",stage:"offer",orderCode:"PED-7",orderStatus:"confirmado",expiresInSeconds:180,...overrides});

test("panel de reparto: agrupa ofertas, repartos en curso y entregados",()=>{
 const groups=panel.groupOrders([order(),order({token:"t2",stage:"accepted"}),order({token:"t3",stage:"picked_up"}),order({token:"t4",stage:"delivered"})]);
 assert.deepEqual(groups.offers.map(item=>item.token),["t1"]);
 assert.deepEqual(groups.active.map(item=>item.token),["t2","t3"]);
 assert.deepEqual(groups.delivered.map(item=>item.token),["t4"]);
});

test("panel de reparto: solo una oferta que aún no se mostró dispara el aviso",()=>{
 const orders=[order(),order({token:"t2"}),order({token:"t3",stage:"accepted"})];
 assert.deepEqual(panel.newOffers(new Set(["t1"]),orders).map(item=>item.token),["t2"]);
 assert.deepEqual(panel.newOffers(new Set(["t1","t2"]),orders),[]);
});

test("panel de reparto: el reloj de la oferta descuenta el tiempo entre consultas y no baja de cero",()=>{
 assert.equal(panel.remainingSeconds(order(),1_000,1_000),180);
 assert.equal(panel.remainingSeconds(order(),1_000,61_000),120);
 assert.equal(panel.remainingSeconds(order(),1_000,999_000),0);
 assert.equal(panel.formatCountdown(125),"2:05");
 assert.equal(panel.formatCountdown(-3),"0:00");
});

test("panel de reparto: recoger exige el pedido listo; entregar, el pedido recogido",()=>{
 assert.equal(panel.nextStep(order()),null,"la oferta se acepta o se rechaza");
 const waiting=panel.nextStep(order({stage:"accepted",orderStatus:"preparando"}));
 assert.equal(waiting.action,"pickup");
 assert.equal(waiting.enabled,false);
 assert.match(waiting.hint,/preparando/);
 const ready=panel.nextStep(order({stage:"accepted",orderStatus:"listo"}));
 assert.equal(ready.enabled,true);
 assert.equal(ready.label,"Recogí el pedido");
 const onTheWay=panel.nextStep(order({stage:"picked_up",orderStatus:"en_camino"}));
 assert.deepEqual([onTheWay.action,onTheWay.label,onTheWay.enabled],["deliver","Entregado",true]);
 assert.equal(panel.nextStep(order({stage:"delivered",orderStatus:"entregado"})),null);
});

test("panel de reparto: cada resultado del backend tiene una respuesta clara y sin rojo para lo esperado",()=>{
 for(const outcome of ["accepted","already_accepted","rejected","picked_up","delivered"])assert.equal(panel.outcomeFeedback(outcome,"PED-7").tone,"success",outcome);
 for(const outcome of ["not_ready","not_picked_up","expired","taken","cancelled"])assert.equal(panel.outcomeFeedback(outcome).tone,"warning",outcome);
 for(const outcome of ["payment_conflict","not_found","algo_nuevo"])assert.equal(panel.outcomeFeedback(outcome).tone,"danger",outcome);
 assert.match(panel.outcomeFeedback("accepted","PED-7").message,/PED-7/);
 assert.match(panel.outcomeFeedback("payment_conflict").message,/No lo recojas ni cobres/);
});

test("panel de reparto: enlaces de llamada e indicaciones solo con datos válidos",()=>{
 assert.equal(panel.dialablePhone("+51 987 654 321"),"+51987654321");
 assert.equal(panel.dialablePhone("999-123-456"),"999123456");
 assert.equal(panel.dialablePhone("javascript:alert(1)"),"");
 assert.equal(panel.dialablePhone(""),"");
 assert.equal(panel.directionsUrl(null,-77),"");
 assert.equal(panel.directionsUrl(-12.1219,-77.0297),"https://www.google.com/maps/dir/?api=1&destination=-12.1219,-77.0297");
});

test("panel de reparto: etapas con etiqueta de una línea y tonos de la matriz, sin rojo",()=>{
 assert.deepEqual(Object.keys(panel.stageMeta).sort(),["accepted","delivered","offer","picked_up"]);
 assert.deepEqual(Object.values(panel.stageMeta).map(meta=>meta.tone),["orange","blue","navy","green"]);
 for(const meta of Object.values(panel.stageMeta))assert.ok(meta.label.length<=12,meta.label);
});

test("panel de reparto: pantalla pública, sin sesión, con avisos, ubicación y confirmación de lo irreversible",()=>{
 const page=read("src/app/reparto/[token]/page.tsx");
 assert.match(page,/referrer:"no-referrer"/,"el enlace privado no viaja como referencia a los mapas");
 assert.match(page,/index:false/);
 assert.match(page,/\^\[a-f0-9\]\{32\}\$/);
 const view=read("src/modules/delivery-panel/presentation/partner-panel.tsx");
 assert.match(view,/refetchIntervalInBackground:true/,"los pedidos nuevos llegan con el panel en segundo plano");
 assert.match(view,/action==="reject"\|\|action==="deliver"\?setConfirming/,"rechazar y entregar piden confirmación");
 assert.match(view,/useCourierLocation\(token,sharing&&groups\.active\.length>0\)/,"la ubicación solo se comparte con un reparto en curso");
 assert.match(view,/<EmptyState icon="motorbike" title="Sin pedidos por ahora"/);
 assert.match(view,/<ErrorState title="No pudimos abrir el panel"/);
 assert.match(view,/<Skeleton /);
 assert.doesNotMatch(view,/whatsapp/i);
 const api=read("src/modules/delivery-panel/infrastructure/panel-api.ts");
 assert.match(api,/\/api\/public\/delivery-panel\//);
 assert.match(read("src/modules/delivery-panel/presentation/use-courier-location.ts"),/wakeLock/);
 const css=read("src/modules/delivery-panel/presentation/partner-panel.css");
 assert.match(css,/\.partner-panel-toggle\{[^}]*min-height:var\(--size-56\)/);
 assert.match(css,/\.partner-panel-actions \.button\{[^}]*min-height:var\(--size-48\)/);
});

test("Plataforma: cada central tiene su enlace de panel, copiable y regenerable con confirmación",()=>{
 const page=read("src/modules/platform/presentation/platform-delivery-partners-page.tsx");
 assert.match(page,/<IconButton icon="link" label=\{`Panel de reparto de \$\{item\.name\}`\}/);
 const link=read("src/modules/platform/presentation/delivery-partner-panel-link.tsx");
 assert.match(link,/publicPartnerPanelPath\(partner\.panelToken\)/);
 assert.match(link,/<ConfirmDialog open=\{confirm\} title="Generar enlace nuevo"/);
 assert.doesNotMatch(link,/ModalFooter/,"modal de solo lectura: la X es el único cierre");
 assert.match(read("src/modules/platform/infrastructure/platform-api.ts"),/delivery-partners\/\$\{encodeURIComponent\(id\)\}\/panel-link/);
});

const chatTracking=(overrides={})=>({available:true,code:"PED-7",status:"preparando",channel:"delivery",delivery:{dispatchStatus:"",partnerName:"",originName:"Local",originAddress:"",origin:{latitude:-12.04,longitude:-77.04},destinationAddress:"Av. Perú 100",destination:{latitude:-12.12,longitude:-77.03},courier:null},...overrides});
// Los arreglos del módulo compilado vienen de otro contexto de vm: se copian a este.
const states=value=>Array.from(tracking.trackingSteps(value),step=>`${step.label}:${step.state}`);

test("seguimiento en el chat: cuatro pasos de una línea para delivery y para recojo",()=>{
 assert.deepEqual(states(chatTracking({status:"nuevo"})),["Recibido:current","Preparando:pending","En camino:pending","Entregado:pending"]);
 assert.deepEqual(states(chatTracking({status:"listo"})),["Recibido:done","Preparando:current","En camino:pending","Entregado:pending"],"listo aún espera al repartidor");
 assert.deepEqual(states(chatTracking({status:"en_camino"})),["Recibido:done","Preparando:done","En camino:current","Entregado:pending"]);
 assert.deepEqual(states(chatTracking({status:"entregado"})),["Recibido:done","Preparando:done","En camino:done","Entregado:done"]);
 assert.deepEqual(states(chatTracking({status:"listo",channel:"recojo",delivery:null})),["Recibido:done","Preparando:done","Listo:current","Entregado:pending"]);
 assert.deepEqual(states(chatTracking({status:"cancelado"})),[]);
});

test("seguimiento en el chat: el mapa acompaña solo al reparto en curso y dice qué se sabe del repartidor",()=>{
 const now=Date.parse("2026-11-14T10:00:30Z");
 assert.equal(tracking.showsMap(chatTracking()),true);
 assert.equal(tracking.showsMap(chatTracking({status:"entregado"})),false);
 assert.equal(tracking.showsMap(chatTracking({channel:"recojo",delivery:null})),false);
 assert.equal(tracking.courierNote(chatTracking(),now),"","delivery propio sin ubicación: sin nota inventada");
 const searching=chatTracking({delivery:{...chatTracking().delivery,dispatchStatus:"searching"}});
 assert.match(tracking.courierNote(searching,now),/buscando un repartidor/);
 const assigned=chatTracking({status:"listo",delivery:{...chatTracking().delivery,dispatchStatus:"accepted",partnerName:"Reparto Norte"}});
 assert.match(tracking.courierNote(assigned,now),/aún no comparte su ubicación/);
 assert.match(tracking.trackingHeadline(assigned),/El repartidor va a recogerlo/);
 const live=chatTracking({status:"en_camino",delivery:{...chatTracking().delivery,dispatchStatus:"picked_up",courier:{latitude:-12.08,longitude:-77.035},courierLocatedAt:"2026-11-14T10:00:00Z"}});
 assert.equal(tracking.courierNote(live,now),"Ubicación del repartidor actualizada hace unos segundos.");
 assert.match(tracking.courierNote(live,now+5*60_000),/hace 6 minutos|hace 5 minutos/);
 assert.equal(tracking.trackingHeadline(live),"Tu pedido va en camino.");
 assert.match(tracking.trackingHeadline(chatTracking({status:"listo",channel:"recojo",delivery:null})),/listo para recoger/);
});

test("seguimiento en el chat: atajo «Seguir pedido» solo con pedido registrado y consulta mientras está abierto",()=>{
 const chat=read("src/modules/concierge-web/presentation/restaurant-chat.tsx");
 assert.match(chat,/\{state\.trackable&&<button type="button" aria-expanded=\{showTracking&&!showCart\}/);
 assert.match(chat,/<IconRoute size=\{16\}\/>Seguir pedido/);
 assert.match(chat,/\{showTracking&&!showCart&&state\.trackable&&<OrderTracking qr=\{qr\}/);
 const view=read("src/modules/concierge-web/presentation/order-tracking.tsx");
 assert.match(view,/refetchInterval:5000/);
 assert.match(view,/<RouteMap origin=/);
 assert.match(read("src/modules/concierge-web/infrastructure/chat-bff.ts"),/"messages", "tracking"\]/);
 const map=read("src/design-system/maps/mapbox-route-provider.tsx");
 assert.match(map,/prefers-reduced-motion: reduce/,"el repartidor no se anima con movimiento reducido");
 assert.match(map,/directions\/v5\/mapbox\/driving/);
});
