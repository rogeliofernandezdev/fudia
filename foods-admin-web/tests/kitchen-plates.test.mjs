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
