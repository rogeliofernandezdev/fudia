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
 assert.deepEqual([...rules.noticeTargets({kind:"counter_order"})],[routes.pos,routes.orders]);
 assert.deepEqual([...rules.noticeTargets({kind:"delivery_order"})],[routes.pos,routes.orders]);
 assert.deepEqual([...rules.noticeTargets({kind:"bill_requested"})],[routes.pos,routes.diningRoom]);
 // Listo en mesa: el mozo entrega en Salón; sin mesa (mostrador, recojo, delivery) se ve en Pedidos.
 assert.deepEqual([...rules.noticeTargets({kind:"order_ready",tableId:"t1"})],[routes.diningRoom,routes.orders]);
 assert.deepEqual([...rules.noticeTargets({kind:"order_ready"})],[routes.orders,routes.pos]);
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
