import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
import {MutationObserver,QueryClient} from "@tanstack/react-query";

const require=createRequire(import.meta.url);
const tick=()=>new Promise(resolve=>setImmediate(resolve));
function deferred(){let resolve,reject;const promise=new Promise((yes,no)=>{resolve=yes;reject=no});return{promise,resolve,reject}}
function mount({status="draft",api,refresh=Promise.resolve()}={}){
 const state=[],refs=[],mutations=[],events=[],calls=[];
 const client=new QueryClient({defaultOptions:{mutations:{retry:2}}});
 let cursor=0,refCursor=0,mutationCursor=0,responseClose=()=>{};
 const order={id:"purchase-1",number:"OC-01",supplierName:"Proveedor",total:"115",status,itemCount:0,items:[],createdAt:"2026-10-08T12:00:00Z"};
 const exports={};
 const source=readFileSync(new URL("../src/modules/supply/purchases/presentation/purchases-page.tsx",import.meta.url),"utf8");
 const resolve=name=>{
  if(name==="react")return{
   useState:initial=>{const i=cursor++;if(!(i in state))state[i]=initial;return[state[i],value=>{state[i]=typeof value==="function"?value(state[i]):value}]},
   useRef:initial=>{const i=refCursor++;return refs[i]??(refs[i]={current:initial})},
  };
  if(name==="@tanstack/react-query")return{
   useQueryClient:()=>({invalidateQueries:({queryKey})=>{events.push("refrescar:"+queryKey[0]);return refresh}}),
   useQuery:({queryKey,enabled})=>({isLoading:false,isError:false,data:queryKey[0]==="purchase-order"?enabled?order:undefined:{items:queryKey[0]==="purchase-orders"?[order]:[],total:1}}),
   useMutation:options=>{
    const i=mutationCursor++;
    const observer=mutations[i]??(mutations[i]=new MutationObserver(client,options));observer.setOptions(options);
    return{...observer.getCurrentResult(),mutateAsync:vars=>observer.mutate(vars),mutate:vars=>{void observer.mutate(vars).catch(()=>{})}};
   },
  };
  if(name==="@/providers")return{
   useSession:()=>({can:()=>true,location:{country:"PE",timezone:"America/Lima"}}),
   useSettings:()=>({currencySymbol:"S/"}),
   useFeedback:()=>({notify:value=>{responseClose();events.push(value)}}),
  };
  if(name.startsWith("@/design-system"))return Object.fromEntries(["Dialog","Button","ConfirmDialog","Icon","Input","PageHeader","Pagination","RemoteModalSkeleton","RowActionButton","Select","Status","Textarea"].map(key=>[key,key]));
  if(name==="next/dynamic")return{__esModule:true,default:()=>"LazyDialog"};
  if(name==="react-hook-form"||name.includes("purchase-schema")||name.endsWith(".css"))return{};
  if(name==="@/shared/hooks/use-debounced-value")return{useDebouncedValue:value=>value};
  if(name==="@/shared/i18n/regional-format")return{formatRegionalNumber:value=>String(value),formatRegionalDateTime:()=>"8 oct. 2026",formatRegionalCalendarDate:()=>"8 oct. 2026"};
  if(name==="../infrastructure/purchases-api")return{
   setPurchaseOrderStatus:async(id,next)=>{calls.push({id,next});return api()},
   approvePurchaseOrder:async id=>{calls.push({id,next:"approved"});return api()},
  };
  return require(name);
 };
 vm.runInNewContext(ts.transpileModule(source,{compilerOptions:{target:ts.ScriptTarget.ES2022,module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve});
 const nodesOf=tree=>{const nodes=[];function walk(node){if(Array.isArray(node)){node.forEach(walk);return}if(node&&typeof node==="object"){nodes.push(node);walk(node.props?.children)}}walk(tree);return nodes};
 function render(){
  cursor=refCursor=mutationCursor=0;
  const nodes=nodesOf(exports.PurchasesPage());
  const detail=nodes.find(node=>node.type?.name==="PurchaseDetail");
  if(detail){const panel=nodesOf(detail.type(detail.props)).find(node=>node.type==="Dialog");responseClose=panel.props.onResponseClose}
  return{nodes,detail};
 }
 render().nodes.find(node=>node.type==="RowActionButton"&&node.props.action==="review").props.onClick();
 return{render,events,calls,dispose:()=>client.clear()};
}

test("Compras espera al API y bloquea clics repetidos, incluso antes del siguiente render",async()=>{
 const response=deferred(),refresh=deferred();
 const ui=mount({api:()=>response.promise,refresh:refresh.promise});
 try{
  const detail=ui.render().detail;
  for(let i=0;i<4;i++)detail.props.changeStatus("pending_approval");
  await tick();assert.equal(ui.calls.length,1);assert.equal(ui.events.length,0);
  assert.equal(ui.render().detail.props.busy,true);
  response.resolve();await tick();
  assert.equal(ui.render().detail,undefined);
  assert.equal(ui.events.filter(value=>value.tone==="success").length,1);
  detail.props.changeStatus("pending_approval");detail.props.approve();await tick();
  assert.equal(ui.calls.length,1,"la acción sigue bloqueada durante el refresco");
  refresh.resolve();await tick();
 }finally{ui.dispose()}
});

test("error remoto cierra el detalle y muestra el error real, sin reintentar la escritura",async()=>{
 const response=deferred();const ui=mount({api:()=>response.promise});
 try{
  ui.render().detail.props.changeStatus("pending_approval");await tick();
  assert.equal(ui.events.length,0);
  response.reject(new Error("La orden no puede pasar al estado solicitado."));await tick();await tick();
  assert.equal(ui.calls.length,1);assert.equal(ui.render().detail,undefined);
  const notices=ui.events.filter(value=>typeof value==="object");
  assert.equal(notices.length,1);assert.equal(notices[0].tone,"danger");
  assert.equal(notices[0].message,"La orden no puede pasar al estado solicitado.");
  assert.ok(ui.events.includes("refrescar:purchase-orders"));
 }finally{ui.dispose()}
});

test("Aprobar y cambiar estado comparten un bloqueo y la aprobación espera su API",async()=>{
 const response=deferred();const ui=mount({status:"pending_approval",api:()=>response.promise});
 try{
  const detail=ui.render().detail;detail.props.approve();detail.props.approve();detail.props.changeStatus("draft");
  await tick();assert.equal(ui.calls.length,1);assert.equal(ui.calls[0].next,"approved");assert.equal(ui.events.length,0);
  response.resolve();await tick();await tick();
  assert.equal(ui.render().detail,undefined);assert.equal(ui.events.filter(value=>value.tone==="success").length,1);
 }finally{ui.dispose()}
});

test("Cancelar orden espera al API y no deja el detalle ni la confirmación detrás",async()=>{
 const response=deferred();const ui=mount({api:()=>response.promise});
 try{
  ui.render().detail.props.cancel();
  const confirm=ui.render().nodes.find(node=>node.type==="ConfirmDialog"&&node.props.title==="Cancelar orden de compra");
  assert.equal(confirm.props.open,true);
  confirm.props.onConfirm();confirm.props.onConfirm();await tick();
  assert.equal(ui.calls.length,1);assert.equal(ui.events.length,0);
  response.resolve();await tick();await tick();
  const result=ui.render();assert.equal(result.detail,undefined);
  assert.equal(result.nodes.find(node=>node.type==="ConfirmDialog"&&node.props.title==="Cancelar orden de compra").props.open,false);
 }finally{ui.dispose()}
});
