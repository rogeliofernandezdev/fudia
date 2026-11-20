import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url);
const read=file=>readFileSync(new URL(`../${file}`,import.meta.url),"utf8");
function compile(file,imports={},globals={}){
  const exports={};
  vm.runInNewContext(ts.transpileModule(read(file),{compilerOptions:{target:ts.ScriptTarget.ES2020,module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:name=>imports[name]??require(name),setTimeout,clearTimeout,JSON,Set,Boolean,Math,...globals});
  return exports;
}
const query=(meta,data)=>({meta,state:{data}});

test("en vivo: sin canal consulta con su intervalo; con canal baja al respaldo de 60 s",()=>{
  const live=compile("src/shared/live/live-state.ts");
  const kitchen=query({live:["kitchen","orders"],poll:10000});
  assert.equal(live.liveRefetchInterval(kitchen),10000);
  live.setLiveConnected(true);
  assert.equal(live.liveRefetchInterval(kitchen),live.LIVE_FALLBACK_MS);
  assert.equal(live.liveRefetchInterval(query({poll:10000})),10000,"sin temas no hay aviso que la reemplace");
  assert.equal(live.liveRefetchInterval(query({live:["pay"],poll:90000})),90000,"un intervalo mayor que el respaldo se conserva");
  assert.equal(live.liveRefetchInterval(query(undefined)),false,"las consultas sin meta no se vuelven periódicas");
  const detail=query({live:["orders"],poll:q=>q.state.data?.completedAt?false:10000},{completedAt:"2026-11-20"});
  assert.equal(live.liveRefetchInterval(detail),false,"un pedido completado deja de consultarse");
  live.setLiveConnected(false);
  assert.equal(live.liveRefetchInterval(query({live:["orders"],poll:q=>q.state.data?.completedAt?false:10000},{})),10000);
});

test("en vivo: un aviso refresca solo las consultas de sus temas, también los temas según el estado",()=>{
  const live=compile("src/shared/live/live-state.ts");
  const topics=new Set(["availability"]);
  assert.equal(live.matchesLiveTopics(query({live:["availability","catalog"]}),topics),true);
  assert.equal(live.matchesLiveTopics(query({live:["orders"]}),topics),false);
  assert.equal(live.matchesLiveTopics(query(undefined),topics),false);
  const catalog=meta=>query({live:q=>(q.state.data?.pages?.length??1)>1?["catalog"]:["availability","catalog"]},meta);
  assert.equal(live.matchesLiveTopics(catalog({pages:[1]}),topics),true);
  assert.equal(live.matchesLiveTopics(catalog({pages:[1,2]}),topics),false,"con varias páginas la disponibilidad no recarga toda la carta");
  assert.equal(live.matchesLiveTopics(query({live:["orders"]})),true,"sin temas, cualquier consulta en vivo");
});

test("en vivo: abre el canal del local, invalida por tema y reintenta sin canal",()=>{
  const live=compile("src/shared/live/live-state.ts");
  const sources=[];
  class FakeEventSource{
    static CONNECTING=0;static OPEN=1;static CLOSED=2;
    constructor(url){this.url=url;this.readyState=0;this.listeners={};this.closed=false;sources.push(this)}
    addEventListener(type,handler){this.listeners[type]=handler}
    close(){this.closed=true;this.readyState=2}
    emit(type,data){this.listeners[type]?.({data})}
  }
  const invalidations=[];
  const client={invalidateQueries:(filters,options)=>{invalidations.push({filters,options})}};
  let cleanup;const timers=[];
  const {LiveUpdates}=compile("src/shared/live/live-updates.tsx",{
    react:{useEffect:effect=>{cleanup=effect()}},
    "@tanstack/react-query":{useQueryClient:()=>client},
    "@/providers/session-context":{useSession:()=>({organization:{id:"org"},location:{id:"loc"}})},
    "./live-state":live,
  },{EventSource:FakeEventSource,setTimeout:(fn,ms)=>{timers.push({fn,ms});return timers.length},clearTimeout:()=>{}});
  assert.equal(LiveUpdates(),null);
  assert.equal(sources.length,1);assert.equal(sources[0].url,"/api/admin/events");
  sources[0].emit("ready","{}");
  assert.equal(live.isLiveConnected(),true);
  assert.equal(invalidations.length,1,"al conectar se refrescan las consultas en vivo");
  assert.equal(invalidations[0].options.cancelRefetch,false,"no cancela una lectura en curso");
  sources[0].emit("change",'{"topics":["kitchen"]}');
  const predicate=invalidations[1].filters.predicate;
  assert.equal(predicate(query({live:["kitchen","orders"]})),true);
  assert.equal(predicate(query({live:["cash"]})),false);
  sources[0].emit("change","no-json");
  assert.equal(invalidations.length,2,"un evento ilegible no rompe el canal");
  // 503 sin canal: EventSource queda cerrado y se reintenta con espera creciente.
  sources[0].readyState=2;sources[0].onerror();
  assert.equal(live.isLiveConnected(),false);
  assert.equal(timers.at(-1).ms,5000);
  timers.at(-1).fn();
  assert.equal(sources.length,2);
  sources[1].readyState=2;sources[1].onerror();
  assert.equal(timers.at(-1).ms,10000);
  cleanup();
  assert.equal(sources[1].closed,true);
  assert.equal(live.isLiveConnected(),false);
});

test("en vivo: layout, BFF y QueryClient conectan el canal sin plazo de 40 s",()=>{
  assert.match(read("src/app/(admin)/layout.tsx"),/<LiveUpdates\/>/);
  assert.match(read("src/providers/providers.tsx"),/refetchInterval:liveRefetchInterval/);
  const route=read("src/app/api/admin/events/route.ts");
  assert.ok(route.includes("signal:request.signal"),"cerrar la pestaña corta también la conexión con el API");
  assert.ok(!route.includes("AbortSignal.timeout"),"el flujo no se corta por tiempo");
  assert.ok(route.includes('headers.set("cookie",cookie)'));
  assert.ok(route.includes('"Content-Type":"text/event-stream"'));
  for(const [file,topics] of [
    ["src/modules/operations/kitchen/presentation/kitchen-board.tsx",'["kitchen","orders"]'],
    ["src/modules/operations/pos/presentation/pos-page.tsx",'["orders","payments"]'],
    ["src/modules/operations/cash/presentation/cash-page.tsx",'["cash","payments"]'],
    ["src/modules/notifications/application/use-notices.ts",'["notices"]'],
    ["src/modules/dashboard/presentation/dashboard-view.tsx",'["orders","payments","cash","availability","reservations"]'],
  ]){
    const source=read(file);
    assert.ok(source.includes(`meta:{live:${topics}`),`${file} declara sus temas`);
    assert.doesNotMatch(source,/refetchInterval:\d/,`${file} no fija un intervalo propio`);
  }
});
