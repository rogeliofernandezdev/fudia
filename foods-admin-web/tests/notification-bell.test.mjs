import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
import postcss from "postcss";

const require=createRequire(import.meta.url);
const read=path=>readFileSync(new URL(`../${path}`,import.meta.url),"utf8");
const routes={pos:"/pos",orders:"/orders",diningRoom:"/dining-room"};
function compile(path){
 const exports={};
 vm.runInNewContext(ts.transpileModule(read(path),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,{exports,require:name=>name==="@/shared/routing/page-routes"?{pageRoutes:routes}:require(name)});
 return exports;
}
const rules=compile("src/modules/notifications/domain/notice-rules.ts");

test("avisos: cada tipo lleva a la pantalla donde se atiende, en orden de preferencia",()=>{
 assert.deepEqual([...rules.noticeTargets({kind:"table_order"})],[routes.diningRoom,routes.orders]);
 assert.deepEqual([...rules.noticeTargets({kind:"assistance_requested"})],[routes.diningRoom,routes.orders]);
 assert.deepEqual([...rules.noticeTargets({kind:"counter_order"})],[routes.orders,routes.pos]);
 assert.deepEqual([...rules.noticeTargets({kind:"delivery_order"})],[routes.orders,routes.pos]);
 assert.deepEqual([...rules.noticeTargets({kind:"bill_requested"})],[routes.pos,routes.diningRoom]);
 assert.deepEqual([...rules.noticeTargets({kind:"stale_account"})],[routes.diningRoom,routes.pos]);
 // Listo en mesa: el mozo entrega en Salón; sin mesa (mostrador, recojo, delivery) se ve en Pedidos.
 assert.deepEqual([...rules.noticeTargets({kind:"order_ready",tableId:"t1"})],[routes.diningRoom,routes.orders]);
 assert.deepEqual([...rules.noticeTargets({kind:"order_ready"})],[routes.orders,routes.pos]);
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
 assert.equal(rules.noticeTone("order_ready"),"brand");
 assert.equal(rules.noticeTone("bill_requested"),"peach");
 assert.equal(rules.noticeTone("assistance_requested"),"sky");
 assert.equal(rules.noticeTone("stale_account"),"peach");
 assert.equal(rules.noticeHref(routes.pos,{kind:"stale_account",orderId:"o-9"}),"/pos?orderId=o-9","caja cobra la cuenta anterior directamente");
 assert.equal(rules.noticeTone("table_order"),"primary");
 const items=[{id:"n3",read:false},{id:"n2",read:true},{id:"n1",read:false}];
 assert.deepEqual([...rules.freshNotices(items,new Set(["n1"]))].map(item=>item.id),["n3"],"solo los nuevos sin leer emergen");
 const bell=read("src/modules/notifications/presentation/notification-bell.tsx");
 assert.match(bell,/tone=\{item\.read\?"neutral":noticeTone\(item\.kind\)\}/,"leído neutro, sin leer con su tono");
 assert.match(bell,/createPortal\(<section className="notification-toast"/,"el aviso emergente sale de la barra superior");
 assert.match(bell,/if\(!openRef\.current\)\{/,"no emerge con la campana abierta");
 assert.match(bell,/className="notification-toast-main" onClick=\{\(\)=>choose\(toast\.notice\)\}/,"tocar el aviso abre el detalle como en la lista");
 assert.doesNotMatch(bell,/>Ver<\/Button>/,"sin botón «Ver» redundante");
 assert.match(bell,/navigator\.userActivation\?\.hasBeenActive\)navigator\.vibrate\?\./,"vibra solo tras interacción, sin sonido");
 const css=read("src/modules/notifications/presentation/notification-bell.css");
 assert.match(css,/\.notification-item\[data-read=false\]\{background:var\(--primary-50\)\}/);
 assert.match(css,/\.notification-toast\{position:fixed;[^}]*z-index:var\(--z-toast\)/);
 assert.match(css,/@media \(prefers-reduced-motion:reduce\)\{[^}]*\.notification-toast\{animation:none/);
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
 let danger=0;rules.walkDecls(decl=>{if(decl.value.includes("danger"))danger++});
 assert.equal(danger,1,"el rojo se reserva al contador sin leer");
});
