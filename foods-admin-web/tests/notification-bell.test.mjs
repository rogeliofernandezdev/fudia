import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
import postcss from "postcss";

const require=createRequire(import.meta.url);
const read=path=>readFileSync(new URL(`../${path}`,import.meta.url),"utf8");
const routes={pos:"/pos",orders:"/orders",diningRoom:"/dining-room",cash:"/cash"};
function compile(path){
 const exports={};
 vm.runInNewContext(ts.transpileModule(read(path),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,{exports,require:name=>name==="@/shared/routing/page-routes"?{pageRoutes:routes}:require(name)});
 return exports;
}
const rules=compile("src/modules/notifications/domain/notice-rules.ts");

test("campana: identifica el logo de Yape o Plin solo en los avisos de pago correspondientes",()=>{
 assert.equal(rules.noticePaymentProvider({kind:"pay_detected",title:"Pago recibido por Yape"}),"yape");
 assert.equal(rules.noticePaymentProvider({kind:"pay_detected",title:"Pago recibido por Plin"}),"plin");
 assert.equal(rules.noticePaymentProvider({kind:"order_ready",title:"Pago recibido por Yape"}),null);
 assert.equal(rules.noticePaymentProvider({kind:"pay_detected",title:"Pago recibido por otra billetera"}),null);
});

test("avisos: cada tipo lleva a la pantalla donde se atiende, en orden de preferencia",()=>{
 assert.deepEqual([...rules.noticeTargets({kind:"table_order"})],[routes.diningRoom,routes.orders]);
 assert.deepEqual([...rules.noticeTargets({kind:"assistance_requested"})],[routes.diningRoom,routes.orders]);
 assert.deepEqual([...rules.noticeTargets({kind:"counter_order"})],[routes.orders,routes.pos]);
 assert.deepEqual([...rules.noticeTargets({kind:"delivery_order"})],[routes.orders,routes.pos]);
 assert.deepEqual([...rules.noticeTargets({kind:"bill_requested"})],[routes.pos,routes.diningRoom]);
 assert.deepEqual([...rules.noticeTargets({kind:"stale_account"})],[routes.diningRoom,routes.pos]);
 assert.deepEqual([...rules.noticeTargets({kind:"pay_detected"})],[routes.cash]);
 assert.equal(rules.noticeHref(routes.cash,{kind:"pay_detected"}),"/cash?tab=payments");
 // Listo en mesa: el mozo entrega en Salón; sin mesa (mostrador, recojo, delivery) se ve en Pedidos.
 assert.deepEqual([...rules.noticeTargets({kind:"order_ready",tableId:"t1"})],[routes.diningRoom,routes.orders]);
 assert.deepEqual([...rules.noticeTargets({kind:"order_ready"})],[routes.orders,routes.pos]);
});

test("avisos de pago: abren «Asociar pago a pedido» en el lugar con Caja permitida",()=>{
 const id="0f6d6c1e-3c43-4a59-9f5d-5c2f3f1a7b20";
 assert.equal(rules.noticePaymentId(routes.cash,{kind:"pay_detected",payNotificationId:id}),id);
 assert.equal(rules.noticePaymentId(undefined,{kind:"pay_detected",payNotificationId:id}),null,"sin permiso de Caja no abre el pago");
 assert.equal(rules.noticePaymentId(routes.cash,{kind:"pay_detected"}),null,"sin id sigue llevando a Caja › Pagos detectados");
 assert.equal(rules.noticePaymentId(routes.orders,{kind:"order_ready",payNotificationId:id}),null);
 const bell=read("src/modules/notifications/presentation/notification-bell.tsx");
 assert.match(bell,/const payment=noticePaymentId\(target,notice\);\s+if\(payment\)\{setPaymentId\(payment\);return\}/);
 assert.match(bell,/createPortal\(<PayNotificationOpener id=\{paymentId\}/);
 const opener=read("src/modules/payment-devices/presentation/pay-notification-opener.tsx");
 assert.match(opener,/mode="associate"/,"un pago detectado va directo a asociarlo");
 assert.match(opener,/payment\.isError\s*\?<ErrorState/,"error explícito al leer el pago");
 assert.match(opener,/role="status" aria-label="Cargando el pago"/,"skeleton mientras carga");
 assert.match(read("src/modules/payment-devices/infrastructure/pay-devices-api.ts"),/apiFetch<PayNotification>\(`pay-notifications\/\$\{encodeURIComponent\(id\)\}`\)/);
 const openapi=readFileSync(new URL("../../foods-backend/api/openapi.yaml",import.meta.url),"utf8");
 assert.match(openapi,/^ {2}\/v1\/admin\/pay-notifications\/\{id\}:\r?\n {4}parameters:[^\n]*\n {4}get:/m,"GET del pago en el contrato");
 assert.ok(openapi.includes("payNotificationId: { type: string, format: uuid"),"el aviso expone su pago");
});

test("avisos: Salón y Pedidos abren el detalle del pedido; POS solo cobra la cuenta pedida",()=>{
 const ready={kind:"order_ready",orderId:"o-1",tableId:"t1"};
 assert.equal(rules.noticeHref(routes.diningRoom,ready),"/dining-room?orderId=o-1");
 assert.equal(rules.noticeHref(routes.orders,{kind:"order_ready",orderId:"o 2"}),"/orders?orderId=o%202");
 assert.equal(rules.noticeHref(routes.pos,{kind:"bill_requested",orderId:"o-3"}),"/pos?orderId=o-3");
 assert.equal(rules.noticeHref(routes.pos,{kind:"counter_order",orderId:"o-4"}),"/pos","un pedido nuevo no abre el cobro");
 assert.equal(rules.noticeHref(routes.diningRoom,{kind:"assistance_requested"}),"/dining-room","sin pedido llega a la pantalla");
 const bell=read("src/modules/notifications/presentation/notification-bell.tsx");
 assert.match(bell,/router\.push\(noticeHref\(target,notice\)\)/);
 for(const page of ["src/app/(admin)/orders/page.tsx","src/app/(admin)/dining-room/page.tsx"])assert.match(read(page),/initialOrderId=\{params\.orderId\}/,page);
 for(const manager of ["src/modules/operations/orders/presentation/orders-manager.tsx","src/modules/operations/salon/presentation/salon-manager.tsx"])assert.match(read(manager),/useOrderDeepLink\(initialOrderId,setDetailId\)/,manager);
 assert.match(read("src/modules/operations/orders/presentation/use-order-deep-link.ts"),/router\.replace\(pathname,\{scroll:false\}\)/,"el parámetro se consume para que el aviso pueda reabrir el detalle");
});

test("avisos: tono por tipo, jerarquía sin leer y aviso emergente",()=>{
 assert.equal(rules.noticeTone("order_ready"),"brand","pedido por entregar usa la familia éxito de su estado");
 assert.equal(rules.noticeTone("bill_requested"),"peach");
 assert.equal(rules.noticeTone("assistance_requested"),"sky");
 assert.equal(rules.noticeTone("stale_account"),"peach");
 assert.equal(rules.noticeHref(routes.pos,{kind:"stale_account",orderId:"o-9"}),"/pos?orderId=o-9","caja cobra la cuenta anterior directamente");
 assert.equal(rules.noticeTone("table_order"),"primary");
 const items=[{id:"n3",read:false},{id:"n2",read:true},{id:"n1",read:false}];
 assert.deepEqual([...rules.freshNotices(items,new Set(["n1"]))].map(item=>item.id),["n3"],"solo los nuevos sin leer emergen");
 const bell=read("src/modules/notifications/presentation/notification-bell.tsx");
 assert.match(bell,/tone=\{item\.read\?"neutral":noticeTone\(item\.kind\)\}/,"leído neutro, sin leer con su tono");
 assert.match(bell,/createPortal\(<section key=\{toast\.notice\.id\} className="notification-toast"/,"el aviso emergente sale de la barra superior");
 assert.match(bell,/if\(!openRef\.current&&popup\.length\)\{/,"no emerge con la campana abierta ni duplica los avisos propios de pagos");
 assert.match(bell,/className="notification-toast-main" onClick=\{\(\)=>choose\(toast\.notice\)\}/,"tocar el aviso abre el detalle como en la lista");
 assert.doesNotMatch(bell,/>Ver<\/Button>/,"sin botón «Ver» redundante");
 assert.match(bell,/navigator\.userActivation\?\.hasBeenActive\)navigator\.vibrate\?\./,"vibra solo tras interacción, sin sonido");
 const css=read("src/modules/notifications/presentation/notification-bell.css");
 assert.match(css,/\.notification-item\[data-read=false\]\{background:var\(--primary-50\)\}/);
 assert.match(css,/\.notification-toast\{position:fixed;[^}]*z-index:var\(--z-toast\)/);
 assert.match(css,/\.notification-toast\{position:fixed;top:calc\(var\(--size-64\) \+ var\(--space-3\)\);right:var\(--space-6\)/,"el aviso emerge arriba a la derecha, bajo la barra superior");
 assert.doesNotMatch(css,/\.notification-toast\{[^}]*bottom:/,"el aviso ya no se ancla abajo");
 assert.match(css,/\.notification-toast\[data-leaving=true\]\{pointer-events:none;animation:notification-toast-out var\(--duration-fast\)/,"sale hacia arriba y más rápido de lo que entra");
 assert.match(css,/\.notification-toast\[data-paused=true\] \.notification-toast-timer\{animation-play-state:paused\}/,"la barra de tiempo se detiene con la pausa");
 assert.match(bell,/document\.addEventListener\("visibilitychange",sync\)/,"se pausa con la pestaña oculta");
 assert.match(bell,/key=\{toast\.notice\.id\}/,"un aviso nuevo reinicia la entrada y el tiempo");
 assert.match(css,/@media \(prefers-reduced-motion:reduce\)\{[^}]*\.notification-toast\{animation:none/);
});

test("avisos: lista continua, cabecera blanca y pago como movimiento",()=>{
 assert.deepEqual({...rules.noticePaymentSummary({kind:"pay_detected",title:"Pago recibido por Yape",detail:"S/ 1.00 · Rosa Quispe"})},{provider:"yape",wallet:"Yape",amount:"S/ 1.00",title:"Rosa Quispe"},"quien pagó es el título y el monto va aparte");
 assert.deepEqual({...rules.noticePaymentSummary({kind:"pay_detected",title:"Pago recibido por Plin",detail:"S/ 5.50"})},{provider:"plin",wallet:"Plin",amount:"S/ 5.50",title:"Pago recibido"},"sin remitente el título es «Pago recibido»");
 assert.equal(rules.noticePaymentSummary({kind:"order_ready",title:"Mesa 4 · listo para entregar",detail:"Cocina"}),null);
 const bell=read("src/modules/notifications/presentation/notification-bell.tsx");
 assert.doesNotMatch(bell,/notification-head-icon/,"la cabecera no lleva icono antes del título");
 assert.match(bell,/<span className="notification-new">\{unread===1\?"1 nuevo":`\$\{unreadLabel\(unread\)\} nuevos`\}<\/span>/,"la cabecera dice cuántos avisos son nuevos");
 assert.match(bell,/<strong className="notification-amount">\{payment\.amount\}<\/strong>/,"el monto del pago va a la derecha");
 assert.match(bell,/\{origin&&`\$\{origin\} · `\}<time dateTime=\{item\.createdAt\}>/,"«Yape · hace 5 min» bajo el título");
 const css=read("src/modules/notifications/presentation/notification-bell.css");
 assert.match(css,/\.notification-head\{[^}]*background:var\(--surface\)\}/,"cabecera blanca como los modales (FUDIA DS 1.0)");
 assert.match(css,/\.notification-item\[data-read=false\] \.notification-amount\{color:var\(--primary-600\)\}/,"monto azul mientras no se lee");
 assert.match(css,/\.notification-list li\+li\{border-top:var\(--stroke-1\) solid var\(--line\)\}/,"avisos separados por línea fina, no tarjetas sueltas");
 assert.match(css,/\.notification-list\{[^}]*scrollbar-width:thin/,"barra de desplazamiento fina");
 assert.match(bell,/<PayProviderTag provider=\{payment\.provider\} appearance="circle" decorative\/>/,"Avisos usa el círculo de billetera, no el logotipo");
 const pay=read("src/modules/payment-devices/presentation/payment-devices.css");
 assert.match(pay,/\.pay-wallet-mark\{[^}]*border-radius:var\(--radius-pill\)[^}]*color:var\(--surface\)/,"billetera como círculo lleno con su nombre en blanco");
 assert.match(pay,/\.pay-wallet-mark\[data-provider=yape\]\{background:var\(--wallet-yape\)\}/);
 assert.match(pay,/\.pay-wallet-mark\[data-provider=plin\]\{background:var\(--wallet-plin\)\}/);
 assert.match(read("src/modules/payment-devices/presentation/pay-notifications-panel.tsx"),/<td><PayProviderTag provider=\{item\.provider\} appearance="circle"\/><\/td>/,"la tabla de Pagos detectados usa el mismo círculo");
});

test("avisos: antigüedad corta y contador acotado",()=>{
 const now=Date.parse("2026-11-02T15:00:00Z");
 assert.equal(rules.noticeAge("2026-11-02T14:59:40Z",now),"Ahora");
 assert.equal(rules.noticeAge("2026-11-02T14:55:00Z",now),"Hace 5 min");
 assert.equal(rules.noticeAge("2026-11-02T12:10:00Z",now),"Hace 2 h");
 assert.equal(rules.noticeAge("no-es-fecha",now),"Ahora");
 assert.equal(rules.unreadLabel(7),"7");
 assert.equal(rules.unreadLabel(140),"99+");
});

test("campana: componente independiente montado por el shell sin lógica propia",()=>{
 const shell=read("src/shell/admin-shell.tsx");
 assert.match(shell,/import \{NotificationBell\} from "@\/modules\/notifications";/);
 assert.match(shell,/<NotificationBell canOpen=\{canOpenRoute\}\/><ContextSwitcher\/>/);
 assert.doesNotMatch(shell,/notices|getNotices|staff-notices/,"el shell no consulta avisos");
 const bell=read("src/modules/notifications/presentation/notification-bell.tsx");
 assert.match(bell,/useNotices\(\)/);
 assert.doesNotMatch(bell,/@\/shell/,"la campana no depende del shell");
 assert.equal(read("src/modules/notifications/index.ts").split("\n").filter(line=>line&&!line.startsWith("export ")).length,0,"el barrel solo re-exporta");
});

test("campana: estados remotos explícitos y nombre accesible",()=>{
 const bell=read("src/modules/notifications/presentation/notification-bell.tsx");
 assert.match(bell,/feed\.isLoading\?<div className="notification-loading" role="status"/);
 assert.match(bell,/<ErrorState className="notification-state" title="No pudimos cargar los avisos"/);
 assert.match(bell,/<EmptyState className="notification-state" icon="bell" title="Sin avisos"/);
 assert.match(bell,/aria-label=\{label\}/);
 assert.match(bell,/aria-live="polite"/);
 assert.doesNotMatch(bell,/Paso \d|CJ-|CAJ-|CLI-/);
});

test("campana: consulta periódica y endpoints del contrato",()=>{
 const hook=read("src/modules/notifications/application/use-notices.ts");
 assert.match(hook,/NOTICES_REFRESH_MS=15000/);
 assert.match(hook,/can\("tables\.read"\)\|\|can\("cash\.manage"\)/);
 for(const file of ["src/modules/operations/pos/presentation/pos-page.tsx","src/modules/operations/orders/presentation/orders-manager.tsx","src/modules/operations/salon/presentation/salon-manager.tsx","src/modules/operations/kitchen/presentation/kitchen-board.tsx"])assert.match(read(file),/"staff-notices"/,file);
 const api=read("src/modules/notifications/infrastructure/notices-api.ts");
 assert.match(api,/apiFetch<NoticeFeed>\("notices"\)/);
 assert.match(api,/`notices\/\$\{encodeURIComponent\(id\)\}\/read`,\{method:"POST"\}/);
 assert.match(api,/"notices\/read",\{method:"POST"\}/);
 const openapi=readFileSync(new URL("../../foods-backend/api/openapi.yaml",import.meta.url),"utf8");
 for(const path of ["/v1/admin/notices:","/v1/admin/notices/read:","/v1/admin/notices/{id}/read:"])assert.ok(openapi.includes(path),path);
});

test("campana: área táctil de 44 px y contador rojo solo en la campana",()=>{
 const rules=postcss.parse(read("src/modules/notifications/presentation/notification-bell.css"));
 const decls=selector=>{const out={};rules.walkRules(rule=>{if(rule.selector===selector)rule.walkDecls(decl=>{out[decl.prop]=decl.value})});return out};
 assert.equal(decls(".notification-trigger").width,"var(--size-44)");
 assert.equal(decls(".notification-trigger").height,"var(--size-44)");
 assert.equal(decls(".notification-count").background,"var(--danger-600)");
 assert.equal(decls(".notification-count").color,"var(--surface)");
 assert.equal(decls(".notification-new").background,"var(--danger-600)","«N nuevos» repite el rojo del contador (aprobado 2026-11-21)");
 const danger=[];rules.walkDecls(decl=>{if(decl.value.includes("danger"))danger.push(decl.parent.selector)});
 assert.deepEqual(danger,[".notification-count",".notification-new"],"el rojo se reserva a los avisos sin leer");
});

test("avisos: un solo sonido general, la alerta de delivery suena doble y los pagos conservan el suyo",()=>{
 const sound=compile("src/modules/notifications/domain/notice-sound.ts");
 for(const kind of ["table_order","delivery_order","pickup_order","order_ready","bill_requested","assistance_requested"])assert.equal(sound.noticeSound(kind),"general",kind);
 for(const kind of ["delivery_unassigned","delivery_pickup_delayed"])assert.equal(sound.noticeSound(kind),"alert",kind);
 for(const kind of ["counter_order","stale_account","pay_detected"])assert.equal(sound.noticeSound(kind),null,`${kind} no suena en la campana`);
 assert.equal(sound.noticeSoundFor([{kind:"table_order"},{kind:"delivery_unassigned"}]),"alert","juntos suenan una vez y gana la alerta");
 assert.equal(sound.noticeSoundFor([{kind:"counter_order"},{kind:"pay_detected"}]),null);
 const general=sound.noticeSoundSchedule("general"),alert=sound.noticeSoundSchedule("alert");
 assert.equal(alert.length,general.length*2,"la alerta repite el mismo sonido");
 assert.deepEqual(alert.slice(general.length).map(tone=>tone.frequency),general.map(tone=>tone.frequency));
 assert.ok(alert[general.length].offset>=general.at(-1).offset+general.at(-1).duration,"la segunda vez empieza al terminar la primera");
 const pay=compile("src/modules/payment-devices/domain/pay-alert.ts").payAlertTones;
 const melody=tones=>tones.map(tone=>tone.frequency).join(",");
 for(const provider of ["yape","plin"])assert.notEqual(melody(general),melody(pay[provider]),`distinto de ${provider}`);
 assert.ok(general[1].frequency<general[0].frequency,"baja (ding-dong), al revés de la escalera de Yape");
 const bell=read("src/modules/notifications/presentation/notification-bell.tsx");
 assert.match(bell,/usePayLeader\(`fudia-notice-sound:/,"suena en una sola pestaña");
 assert.match(bell,/useStoredFlag\(`fudia-notice-sound-muted:/,"cada usuario puede silenciarlo");
 assert.match(bell,/className="notification-sound" aria-pressed=\{!muted\}/);
 assert.match(bell,/if\(tone&&sound\.current\.soundLeader&&!sound\.current\.muted\)/);
});
