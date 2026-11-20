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
 vm.runInNewContext(ts.transpileModule(read(path),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,{exports,require,URLSearchParams,Date,Math,TextEncoder});
 return exports;
}
const panel=compile("src/modules/delivery-panel/domain/panel.ts");
const tracking=compile("src/modules/concierge-web/domain/tracking.ts");
const order=(overrides={})=>({token:"t1",stage:"offer",orderCode:"PED-7",orderStatus:"confirmado",paymentConfirmed:false,expiresInSeconds:180,...overrides});

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

test("motorizado: recoger exige pedido listo y pago completo; en camino, entregar o reportar «No pude entregar»",()=>{
 assert.equal(panel.courierSteps(order()),null,"la oferta la responde la central");
 const waiting=panel.courierSteps(order({stage:"accepted",orderStatus:"preparando"}));
 assert.equal(waiting.primary.action,"pickup");
 assert.equal(waiting.primary.enabled,false);
 assert.match(waiting.primary.hint,/preparando/);
 assert.equal(waiting.secondary,null);
 const unpaid=panel.courierSteps(order({stage:"accepted",orderStatus:"listo"}));
 assert.equal(unpaid.primary.enabled,false);
 assert.match(unpaid.primary.hint,/caja/);
 assert.equal(panel.courierSteps(order({stage:"accepted",orderStatus:"listo",paymentConfirmed:undefined})).primary.enabled,false,"sin confirmación del servidor no habilita la recogida");
 assert.equal(panel.courierSteps(order({stage:"accepted",orderStatus:"preparando",paymentConfirmed:true})).primary.enabled,false,"pagar no sustituye la preparación");
 const ready=panel.courierSteps(order({stage:"accepted",orderStatus:"listo",paymentConfirmed:true}));
 assert.deepEqual([ready.primary.enabled,ready.primary.label],[true,"Recogí el pedido"]);
 assert.match(ready.primary.hint,/pago confirmado por caja/);
 const onTheWay=panel.courierSteps(order({stage:"picked_up",orderStatus:"en_camino"}));
 assert.deepEqual([onTheWay.primary.action,onTheWay.primary.label,onTheWay.primary.enabled],["deliver","Entregado",true]);
 assert.deepEqual([onTheWay.secondary.action,onTheWay.secondary.label],["not_delivered","No pude entregar"]);
 const reported=panel.courierSteps(order({stage:"picked_up",orderStatus:"en_camino",issueReportedAt:"2026-11-14T10:00:00Z"}));
 assert.equal(reported.secondary,null,"se reporta una sola vez");
 assert.equal(reported.primary.action,"deliver","si el cliente aparece, aún puede entregarlo");
 assert.match(reported.primary.hint,/cliente aparece/);
 assert.equal(panel.courierSteps(order({stage:"delivered",orderStatus:"entregado"})),null);
});

test("central: nombre del motorizado, enlace privado y liberar solo antes de recoger",()=>{
 assert.equal(panel.courierNameValue("  Luis   Pérez "),"Luis Pérez");
 assert.equal(panel.courierNameValue("L"),"");
 assert.equal(panel.courierNameValue("x".repeat(61)),"");
 const token="a".repeat(32);
 assert.equal(panel.courierLink("https://app.fudia.pe/",token),`https://app.fudia.pe/delivery-courier/${token}`);
 assert.equal(panel.courierLink("https://app.fudia.pe",""),"");
 assert.equal(panel.courierLink("https://app.fudia.pe","CJ-6E57C7"),"");
 assert.equal(panel.canRelease(order({stage:"accepted"})),true);
 for(const stage of ["offer","picked_up","delivered"])assert.equal(panel.canRelease(order({stage})),false,stage);
});

test("central: la alerta de recogida pide decisión hasta «Mantener» y nunca libera sola",()=>{
 assert.deepEqual(Object.keys(panel.pickupAlertMeta),["warning","urgent","critical"]);
 assert.deepEqual(Object.values(panel.pickupAlertMeta).map(meta=>meta.tone),["warning","warning","danger"]);
 for(const meta of Object.values(panel.pickupAlertMeta))assert.ok(meta.label.length<=10,meta.label);
 assert.match(panel.pickupAlertMeta.critical.message,/restaurante/);
 const waiting=order({stage:"accepted",orderStatus:"listo",pickupAlert:"urgent",pickupAlertAcknowledged:false,pickupWaitSeconds:320});
 assert.equal(panel.needsPickupDecision(waiting),true);
 assert.equal(panel.needsPickupDecision({...waiting,pickupAlertAcknowledged:true}),false);
 assert.equal(panel.needsPickupDecision({...waiting,pickupAlert:""}),false);
 assert.equal(panel.needsPickupDecision({...waiting,stage:"picked_up"}),false);
 assert.equal(panel.waitLabel(320),"5 min");
 assert.equal(panel.waitLabel(-4),"0 min");
 const view=read("src/modules/delivery-panel/presentation/partner-panel.tsx");
 assert.match(view,/onAct\("keep"\)/);
 assert.match(view,/onConfirm\("change"\)/);
 assert.match(view,/onConfirm\("release"\)/);
 assert.doesNotMatch(view,/onAct\("release"\)/,"liberar siempre pasa por confirmación");
 assert.doesNotMatch(view,/setTimeout|onAct\("fail"\)/,"nada se libera ni se registra por tiempo");
});

test("central: entrega fallida solo tras el reporte y el tiempo de espera; suenan críticas y reportes",()=>{
 const reported=order({token:"t9",stage:"picked_up",orderStatus:"en_camino",issueReportedAt:"2026-11-14T10:00:00Z",failAvailableInSeconds:120});
 assert.equal(panel.failCountdown(order({stage:"picked_up"}),0,0),null);
 assert.equal(panel.failCountdown(reported,1_000,61_000),60);
 assert.equal(panel.failCountdown(reported,1_000,999_000),0);
 const critical=order({token:"t8",stage:"accepted",orderStatus:"listo",pickupAlert:"critical",pickupAlertAcknowledged:false});
 assert.deepEqual(Array.from(panel.urgentKeys([critical,reported,order({token:"t7",stage:"accepted",pickupAlert:"warning"})])),["t8:critical","t9:issue:2026-11-14T10:00:00Z"]);
 assert.deepEqual(Array.from(panel.urgentKeys([{...critical,pickupAlertAcknowledged:true}])),[]);
});

test("panel de reparto: cada resultado del backend tiene una respuesta clara y sin rojo para lo esperado",()=>{
 for(const outcome of ["accepted","already_accepted","rejected","picked_up","delivered","courier_assigned","released","kept","issue_reported","delivery_failed"])assert.equal(panel.outcomeFeedback(outcome,"PED-7").tone,"success",outcome);
 for(const outcome of ["not_ready","not_picked_up","expired","taken","cancelled","payment_pending","already_picked_up","not_reported","too_soon"])assert.equal(panel.outcomeFeedback(outcome).tone,"warning",outcome);
 for(const outcome of ["payment_conflict","not_found","algo_nuevo"])assert.equal(panel.outcomeFeedback(outcome).tone,"danger",outcome);
 assert.match(panel.outcomeFeedback("accepted","PED-7").message,/PED-7/);
 assert.match(panel.outcomeFeedback("payment_conflict").message,/No lo recojas ni cobres/);
 assert.match(panel.outcomeFeedback("payment_pending").message,/caja confirme el pago completo/);
 assert.match(panel.outcomeFeedback("payment_conflict","PED-7","release").message,/antes de liberarlo/);
 assert.match(panel.outcomeFeedback("released","PED-7").message,/otras centrales/);
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

test("panel de la central con sesión; enlace del motorizado público; lo irreversible pide confirmación",()=>{
 const page=read("src/app/delivery-panel/page.tsx");
 assert.match(page,/referrer:"no-referrer"/,"el panel no viaja como referencia a los mapas");
 assert.match(page,/index:false/);
 assert.match(page,/fudia_partner_session[\s\S]*redirect\("\/delivery-panel\/login"\)/,"sin cookie de sesión va al acceso");
 assert.match(read("src/app/delivery-panel/[token]/page.tsx"),/redirect\("\/delivery-panel\/login"\)/,"los enlaces privados antiguos llevan al acceso");
 assert.match(read("src/app/delivery-panel/login/page.tsx"),/<PartnerLogin hasSession=\{hasSession\}\/>/);
 const view=read("src/modules/delivery-panel/presentation/partner-panel.tsx");
 assert.match(view,/export function PartnerPanel\(\)\{/,"el panel ya no recibe un enlace privado");
 assert.match(view,/needsLogin\(panel\.error\)\|\|needsPasswordChange\(panel\.error\)/,"sesión vencida o contraseña temporal vuelven al acceso");
 assert.match(view,/router\.replace\("\/delivery-panel\/login"\)/);
 assert.match(view,/logoutPartner\(\)/,"el operador cierra su sesión");
 assert.match(view,/refetchIntervalInBackground:true/,"los pedidos nuevos llegan con el panel en segundo plano");
 assert.match(view,/onConfirm\("reject"\)/,"rechazar pide confirmación");
 assert.doesNotMatch(view,/useCourierLocation/,"la central ya no comparte ubicación: lo hace el motorizado");
 assert.doesNotMatch(view,/"pickup"|"deliver"/,"recoger y entregar se registran en el enlace del motorizado");
 assert.match(view,/title="Registrar entrega fallida"/);
 assert.match(view,/100 % de lo que pagó por la comida/);
 assert.match(view,/50 % Fudia y 50 % tu central/);
 assert.match(read("src/modules/delivery-panel/presentation/dispatch-workspace.tsx"),/<EmptyState icon="motorbike" title="Sin pedidos por ahora"/);
 assert.match(view,/<ErrorState title="No pudimos abrir el panel"/);
 assert.match(view,/<Skeleton /);
 assert.doesNotMatch(view,/whatsapp/i);
 const courierPage=read("src/app/delivery-courier/[token]/page.tsx");
 assert.match(courierPage,/referrer:"no-referrer"/);
 assert.match(courierPage,/index:false/);
 assert.match(courierPage,/\^\[a-f0-9\]\{32\}\$/);
 const courier=read("src/modules/delivery-panel/presentation/courier-panel.tsx");
 assert.match(courier,/useCourierLocation\(token,sharing&&inProgress\)/,"la ubicación solo se comparte con el reparto en curso");
 assert.match(courier,/setConfirming\("not_delivered"\)/,"«No pude entregar» pide confirmación");
 assert.match(courier,/llama al cliente al número del pedido, espera unos minutos y vuelve a intentarlo/);
 assert.match(courier,/<ErrorState title="No pudimos abrir el reparto"/);
 assert.match(courier,/<EmptyState icon="circleCheck" title="Entrega registrada"/);
 assert.match(courier,/<Skeleton /);
 const api=read("src/modules/delivery-panel/infrastructure/panel-api.ts");
 assert.doesNotMatch(api,/\/api\/public\/delivery-panel/,"el panel de la central ya no es público");
 assert.match(api,/const partnerBase="\/api\/delivery-partner"/);
 assert.match(api,/\$\{partnerBase\}\/panel\/courier/);
 assert.match(api,/\/api\/public\/delivery-courier\//,"el enlace del motorizado sigue público");
 const proxy=read("src/app/api/delivery-partner/[...path]/route.ts");
 assert.match(proxy,/\/v1\/delivery-partner\//);
 assert.match(proxy,/const SESSION_COOKIE="fudia_partner_session"/);
 assert.match(proxy,/next\.headers\.set\("set-cookie",cookie\)/,"el BFF devuelve la cookie HttpOnly del backend");
 assert.match(proxy,/if\(logout&&session\)failed\.headers\.set\("set-cookie",CLEAR_COOKIE\)/,"cerrar sesión borra la cookie aunque el backend no responda");
 assert.match(read("src/modules/delivery-panel/presentation/dispatch-workspace.css"),/@media\(max-width:600px\)\{\.dispatch-app-logout\{[^}]*width:var\(--size-44\)[^}]*\}\.dispatch-app-logout>span\{display:none\}/,"en móvil «Salir» es solo icono para no partir «Avisos con sonido»");
 assert.match(read("src/modules/delivery-panel/presentation/use-courier-location.ts"),/wakeLock/);
 const css=read("src/modules/delivery-panel/presentation/partner-panel.css");
 assert.match(css,/\.partner-panel-toggle\{[^}]*min-height:var\(--size-56\)/);
 assert.match(css,/\.partner-panel-actions \.button\{[^}]*min-height:var\(--size-48\)/);
});

test("acceso de la central: correo y contraseña, y la temporal se reemplaza antes de entrar",()=>{
 assert.equal(panel.passwordChangeProblem("","clave-nueva","clave-nueva"),"Ingresa la contraseña temporal.");
 assert.match(panel.passwordChangeProblem("Temp12345678","corta","corta"),/al menos 8/);
 assert.match(panel.passwordChangeProblem("Temp12345678","Temp12345678","Temp12345678"),/distinta/);
 assert.match(panel.passwordChangeProblem("Temp12345678","clave-nueva","clave-nuevo"),/no coinciden/);
 assert.match(panel.passwordChangeProblem("Temp12345678","x".repeat(73),"x".repeat(73)),/demasiado larga/);
 assert.equal(panel.passwordChangeProblem("Temp12345678","clave-nueva","clave-nueva"),"");
 const login=read("src/modules/delivery-panel/presentation/partner-login.tsx");
 assert.match(login,/autoComplete="username"/);
 assert.match(login,/autoComplete="new-password"/);
 assert.match(login,/<h1 id="partner-login-title">Crea tu contraseña<\/h1>/);
 assert.match(login,/session\.operator\.mustChangePassword/);
 assert.doesNotMatch(login,/localStorage|sessionStorage/,"la sesión vive solo en la cookie HttpOnly");
 assert.match(login,/queryClient\.setQueryData\(\["delivery-partner-session"\],session\)/,"el panel reutiliza la sesión del acceso sin pedirla otra vez");
 assert.match(login,/enter\(await changePartnerPassword\(current,next\)\)/);
});

test("Plataforma: la central se edita con pestañas Datos y Operadores; la contraseña temporal se ve una vez",()=>{
 const page=read("src/modules/platform/presentation/delivery-partners-admin.tsx");
 assert.match(page,/\{value:"data",label:"Datos",icon:"motorbike"\},\{value:"operators",label:"Operadores",icon:"users"\}/);
 assert.match(page,/<DeliveryPartnerOperators partner=\{editing\}\/>/);
 assert.doesNotMatch(page,/panelToken|panel-link|icon="link"/,"ya no hay enlace privado de la central");
 const operators=read("src/modules/platform/presentation/delivery-partner-operators.tsx");
 assert.match(operators,/title="Contraseña temporal"/);
 assert.match(operators,/Se muestra una sola vez/);
 assert.match(operators,/<ConfirmDialog open=\{Boolean\(resetTarget\)\} tone="danger" title="Nueva contraseña temporal"/);
 assert.match(operators,/action=\{operator\.active\?"deactivate":"activate"\}/);
 assert.match(operators,/<EmptyState icon="users" title="Sin operadores"/);
 assert.match(operators,/<ErrorState title="No pudimos cargar los operadores"/);
 assert.match(operators,/<Skeleton /);
 const api=read("src/modules/platform/infrastructure/platform-api.ts");
 assert.match(api,/\/operators`/);
 assert.match(api,/\/password-reset`/);
 assert.doesNotMatch(api,/panel-link/);
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

test("panel de reparto: cada contador tiene su propio reloj y el panel no se re-renderiza cada segundo",()=>{
 const panel=read("src/modules/delivery-panel/presentation/partner-panel.tsx");
 const workspace=read("src/modules/delivery-panel/presentation/dispatch-workspace.tsx");
 assert.doesNotMatch(panel,/setInterval/,"el panel ya no tiene un reloj global");
 assert.match(panel,/const clock=useTickingClock\(order\.stage==="offer"\|\|order\.pickupWaitSeconds>0/,"la tarjeta solo avanza mientras tiene un contador");
 assert.match(workspace,/<OfferCountdown order=\{order\} receivedAt=\{receivedAt\}\/>/,"la fila de la lista cuenta su oferta");
 assert.match(read("src/modules/delivery-panel/presentation/use-ticking-clock.ts"),/return Math\.max\(now,since\)/);
});
