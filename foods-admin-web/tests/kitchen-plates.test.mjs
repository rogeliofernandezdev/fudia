import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import vm from "node:vm";
import ts from "typescript";
import {resolveDesignScales} from "./helpers/design-tokens.mjs";

const root=new URL("../",import.meta.url);
const read=file=>readFileSync(new URL(file,root),"utf8").replaceAll("\r\n","\n");
function compile(file){
  const exports={};
  vm.runInNewContext(ts.transpileModule(read(file),{compilerOptions:{target:ts.ScriptTarget.ES2022,module:ts.ModuleKind.CommonJS}}).outputText,{exports,require:()=>({})});
  return exports;
}
const plates=compile("src/modules/operations/kitchen/domain/plates.ts");
const plate=(id,status)=>({id,productId:id,name:id,qty:"1.00",unitPrice:"0.00",note:"",status});
const page=()=>({
  items:[
    {id:"r1",orderId:"o1",status:"preparando",items:[plate("lomo","listo"),plate("ceviche","preparando"),plate("arroz","preparando"),plate("chicha","preparando")]},
    {id:"r2",orderId:"o2",status:"preparando",items:[plate("tallarin","preparando")]},
    {id:"legacy",orderId:"o3",status:"preparando",items:[{id:"x",productId:"x",name:"Sopa",qty:"1",unitPrice:"0",note:""}]},
  ],
  counts:{confirmado:0,preparando:3,listo:0},
  channelOptions:[],serverTime:"",
});

test("Cocina cuenta el progreso por línea de la comanda",()=>{
  const data=page();
  assert.deepEqual({...plates.plateProgress(data.items[0])},{ready:1,total:4});
  assert.equal(plates.plateProgress(data.items[2]),null,"las comandas históricas no se marcan por plato");
});

test("marcar un plato deriva la comanda como el backend y el último la pasa a Por entregar",()=>{
  let data=plates.applyPlateStatus(page(),"r2","tallarin","listo");
  assert.equal(data.items[1].status,"listo");
  assert.deepEqual({...data.counts},{confirmado:0,preparando:2,listo:1});
  data=plates.applyPlateStatus(data,"r1","ceviche","listo");
  assert.equal(data.items[0].status,"preparando","con platos pendientes la comanda sigue en preparación");
  assert.equal(data.items[0].items[1].status,"listo");
  // Reversión del optimista: solo ese plato vuelve a su estado anterior.
  const reverted=plates.applyPlateStatus(plates.applyPlateStatus(page(),"r2","tallarin","listo"),"r2","tallarin","preparando");
  assert.equal(reverted.items[1].status,"preparando");
  assert.deepEqual({...reverted.counts},{confirmado:0,preparando:3,listo:0});
  assert.equal(page().items[1].items[0].status,"preparando","la cola original no se modifica");
});

test("cada plato empieza solo y avanza paso a paso, sin saltar estados",()=>{
  assert.equal(plates.nextPlateStatus(plate("lomo","confirmado")),"preparando","por preparar solo puede empezar");
  assert.equal(plates.nextPlateStatus(plate("lomo","preparando")),"listo");
  assert.equal(plates.nextPlateStatus(plate("lomo","listo")),"preparando","listo vuelve a preparación");
  assert.equal(plates.plateActionLabel(plate("lomo","confirmado")),"Empezar plato");
  // El lomo empieza solo: la comanda pasa a En preparación y el ceviche sigue esperando.
  const waiting={items:[{id:"r9",orderId:"o9",status:"confirmado",items:[plate("lomo","confirmado"),plate("ceviche","confirmado")]}],counts:{confirmado:1,preparando:0,listo:0},channelOptions:[],serverTime:""};
  const started=plates.applyPlateStatus(waiting,"r9","lomo","preparando","2026-11-26T17:00:00.000Z");
  assert.equal(started.items[0].status,"preparando");
  assert.equal(started.items[0].items[0].preparingSince,"2026-11-26T17:00:00.000Z","la cuenta regresiva aparece al instante");
  assert.equal(started.items[0].items[1].status,"confirmado");
  assert.equal(started.items[0].items[1].preparingSince,undefined);
  assert.deepEqual({...started.counts},{confirmado:0,preparando:1,listo:0});
  // La comanda ofrece iniciar lo que espera; «Marcar listo» solo con todo empezado.
  assert.equal(plates.ticketNextStep(started.items[0]),"preparando");
  assert.equal(plates.ticketNextStep({...started.items[0],items:[plate("lomo","preparando"),plate("ceviche","listo")]}),"listo");
  assert.equal(plates.ticketNextStep({...started.items[0],items:[plate("lomo","listo")]}),null);
  const board=read("src/modules/operations/kitchen/presentation/kitchen-board.tsx");
  assert.ok(board.includes('"Iniciar el resto"'),"con un plato en marcha, «Iniciar» empieza solo lo que falta");
  assert.ok(board.includes("status:step"),"el toque usa el siguiente paso del plato");
  const css=read("src/modules/operations/kitchen/presentation/kitchen.css");
  assert.ok(css.includes(".kitchen-plate[data-state=preparando] .kitchen-plate-check{border-color:var(--kds-cooking);background:var(--kds-cooking-soft)}"));
});

test("el tablero marca platos con check circular, progreso y reversión con error",()=>{
  const board=read("src/modules/operations/kitchen/presentation/kitchen-board.tsx");
  const api=read("src/modules/operations/kitchen/infrastructure/kitchen-api.ts");
  assert.match(api,/kitchen\/tickets\/\$\{encodeURIComponent\(ticketId\)\}\/items\/\$\{encodeURIComponent\(itemId\)\}\/status/);
  assert.ok(board.includes("{progress.ready} de {progress.total} listos"),"progreso explícito «2 de 4 listos»");
  assert.ok(board.includes('role="progressbar"'));
  assert.ok(board.includes("aria-pressed={ready}"),"cada plato es un botón alternable");
  assert.ok(board.includes("if(!canManage)return"),"sin permiso de la estación el plato es de solo lectura");
  const plateMutation=board.slice(board.indexOf("const markPlate"));
  assert.ok(plateMutation.includes("patchPlate(ticket.id,itemId,previous)"),"el error revierte solo ese plato");
  assert.match(plateMutation,/notify\(\{tone:"danger",title:"No se pudo actualizar el plato"/);
  for(const key of ["orders","salon-floor","pos-orders","pos-order","dashboard"])assert.ok(plateMutation.includes(`"${key}"`),key);
  assert.ok(board.includes('"Marcar listo"'),"marcar toda la comanda sigue disponible");
});

test("el check y la barra usan el naranja del carril En preparación, con tokens del sistema",()=>{
  const css=resolveDesignScales(read("src/modules/operations/kitchen/presentation/kitchen.css"));
  assert.match(css,/\.kitchen-ticket-item\.kitchen-plate\{[^}]*min-height:44px/,"fila táctil de 44 px");
  assert.match(css,/\.kitchen-plate-check\{[^}]*width:20px;height:20px;[^}]*border:1\.5px solid var\(--line-strong\);border-radius:50%/);
  assert.ok(css.includes(".kitchen-plate[data-ready=true] .kitchen-plate-check{border-color:var(--kds-cooking);background:var(--kds-cooking);color:var(--surface)}"));
  assert.ok(css.includes("text-decoration:line-through"),"el plato listo se tacha");
  assert.match(css,/\.kitchen-progress-track i\{[^}]*background:var\(--kds-cooking\)/);
  assert.match(css,/\.kitchen-lane\[data-status="preparando"\] \.kitchen-ticket-item>b\{background:var\(--kds-cooking-soft\);color:var\(--kds-cooking-dark\)\}/,"la cantidad conserva el círculo del color de su carril");
  assert.equal(/#[0-9a-f]{3,6}\b|rgba?\(/i.test(css),false);
});

const timing=compile("src/modules/operations/kitchen/domain/timing.ts");
const startedAt="2026-11-26T12:00:00-05";
const at=minutes=>new Date("2026-11-26T12:00:00-05:00").getTime()+minutes*60000;
const cooking=(id,prepMinutes,extra={})=>({...plate(id,"preparando"),prepMinutes,preparingSince:startedAt,...extra});

test("cada plato cuenta su tiempo de preparación desde que la estación lo empezó",()=>{
  assert.deepEqual({...timing.plateTimer(cooking("lomo",15),at(4))},{tone:"fresh",label:"Faltan 11 de 15 min"});
  assert.deepEqual({...timing.plateTimer(cooking("lomo",15),at(14.5))},{tone:"warning",label:"Falta 1 de 15 min"},"último cuarto: por vencer");
  assert.deepEqual({...timing.plateTimer(cooking("lomo",15),at(15.2))},{tone:"late",label:"Tiempo cumplido"});
  assert.deepEqual({...timing.plateTimer(cooking("lomo",15),at(18))},{tone:"late",label:"Demora 3 min"});
  assert.equal(timing.plateTimer(cooking("lomo",undefined),at(4)),null,"sin tiempo de preparación no hay contador");
  assert.equal(timing.plateTimer(cooking("lomo",15,{preparingSince:undefined}),at(4)),null,"sin hora de inicio no hay contador");
  assert.equal(timing.plateTimer(cooking("lomo",15,{status:"listo"}),at(4)),null,"un plato listo ya no cuenta");
});

test("la comanda toma el plato más atrasado y su reloj no se reinicia al marcar platos",()=>{
  const ticket={id:"r1",orderId:"o1",status:"preparando",updatedAt:"2026-11-26T12:09:00-05",targetMinutes:20,items:[cooking("ceviche",10),cooking("lomo",20),cooking("chicha",5,{status:"listo"})]};
  assert.equal(timing.elapsedMinutes(ticket,at(12)),12,"cuenta desde el primer plato empezado, no desde la última marca");
  assert.equal(timing.ticketUrgency(ticket,at(12)),"late","el ceviche venció aunque el objetivo de la comanda sea 20 min");
  assert.equal(timing.ticketUrgency(ticket,at(5)),"fresh");
  const legacy={id:"l",orderId:"o3",status:"preparando",updatedAt:startedAt,targetMinutes:10,items:[{id:"x",productId:"x",name:"Sopa",qty:"1",unitPrice:"0",note:""}]};
  assert.equal(timing.ticketUrgency(legacy,at(8)),"warning","sin contador por plato se usa el objetivo de la comanda");
});

test("cada plato respeta su propio tiempo: sin objetivo común de la comanda",()=>{
  assert.equal(timing.plateEstimate(plate("lomo","confirmado")),null,"sin tiempo definido no se inventa uno");
  assert.equal(timing.plateEstimate({...plate("lomo","confirmado"),prepMinutes:15}),"15 min","por preparar muestra el tiempo propio del plato");
  assert.equal(timing.plateEstimate(cooking("lomo",15)),null,"en marcha lo reemplaza la cuenta regresiva");
  const waiting={id:"w",orderId:"o4",status:"confirmado",updatedAt:startedAt,targetMinutes:20,items:[{...plate("lomo","confirmado"),prepMinutes:20}]};
  assert.equal(timing.ticketUrgency(waiting,at(30)),"neutral","la espera no se compara con el plato más largo");
  // La comanda con el plato que consumió más de su propio tiempo va primero, aunque lleve menos minutos.
  const quick={id:"q",orderId:"o5",status:"preparando",updatedAt:startedAt,targetMinutes:6,items:[cooking("causa",6)]};
  const slow={id:"s",orderId:"o6",status:"preparando",updatedAt:startedAt,targetMinutes:30,items:[cooking("seco",30,{preparingSince:"2026-11-26T11:58:00-05"})]};
  assert.ok(timing.ticketPriority(quick,at(5))>timing.ticketPriority(slow,at(5)),"5 de 6 min va antes que 7 de 30 min");
  const board=read("src/modules/operations/kitchen/presentation/kitchen-board.tsx");
  assert.equal(board.includes("Objetivo"),false,"la cabecera no impone el tiempo del plato más largo");
  assert.ok(board.includes("ticketPriority(right,now)-ticketPriority(left,now)"));
});

test("el tablero muestra el contador del plato con tokens del sistema",()=>{
  const board=read("src/modules/operations/kitchen/presentation/kitchen-board.tsx");
  assert.ok(board.includes("const timer=plateTimer(item,now)"));
  assert.ok(board.includes('className="kitchen-plate-timer" data-tone={timer.tone}'));
  assert.equal(board.includes("function parseIsoDate"),false,"el cálculo de tiempos vive en el dominio");
  const css=read("src/modules/operations/kitchen/presentation/kitchen.css");
  assert.ok(css.includes(".kitchen-ticket-item .kitchen-plate-timer[data-tone=late]{"));
  assert.match(css,/\.kitchen-ticket-item \.kitchen-plate-timer\{[^}]*white-space:nowrap/,"el contador cabe en una línea");
});
