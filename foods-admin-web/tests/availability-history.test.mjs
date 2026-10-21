import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
import {createFormControl} from "react-hook-form";
import postcss from "postcss";

const require=createRequire(import.meta.url),root=new URL("../src/",import.meta.url);
function compile(path,resolve=require){
 const exports={};vm.runInNewContext(ts.transpileModule(readFileSync(new URL(path,root),"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve,URLSearchParams});return exports;
}
function nodes(tree){const out=[];function visit(node){if(Array.isArray(node)){node.forEach(visit);return}if(node&&typeof node==="object"){out.push(node);visit(node.props?.children)}}visit(tree);return out;}
function words(tree){if(Array.isArray(tree))return tree.map(words).join("");if(tree&&typeof tree==="object")return words(tree.props?.children);return String(tree??"");}
const ds=Object.fromEntries(["Button","Dialog","FormField","Icon","Input","Textarea","PageHeader","Pagination","RowActionButton","Select","Status"].map(name=>[name,name]));
const resolver=compile("shared/forms/zod-resolver.ts");
const schema=compile("modules/menu/availability/domain/availability-reason-schema.ts",name=>name==="@/shared/forms/zod-resolver"?resolver:require(name));
const regional=compile("shared/i18n/regional-format.ts");
const item={productId:"aji",name:"Ají de gallina",quantityControl:"portions",portionQuantity:15,soldQuantity:5,remaining:10,status:"available",manualStatus:"available",source:"portions",note:"Nota conservada",businessDate:"2026-10-08"};
const change={item,status:"available",portionQuantity:20,kind:"quota"};
const tick=()=>new Promise(resolve=>setImmediate(resolve));

function reasonUI({save=async()=>{},busy=false}={}){
 let control;const guard={current:false};
 const {AvailabilityReasonDialog}=compile("modules/menu/availability/presentation/availability-reason-dialog.tsx",name=>{
  if(name==="react")return {useRef:()=>guard};
  if(name==="react-hook-form")return {useForm:options=>{control??=createFormControl(options);return {...control,formState:control.control._formState}}};
  if(name==="@/design-system")return ds;
  if(name==="../domain/availability-reason-schema")return schema;
  return require(name);
 });
 function render(){const all=nodes(AvailabilityReasonDialog({change,busy,close(){},save}));return {all,form:all.find(n=>n.type==="form"),dialog:all.find(n=>n.type==="Dialog"),save:all.find(n=>n.type==="Button"&&n.props.kind!=="ghost")};}
 render();return {render,control};
}

test("Disponibilidad exige un motivo nuevo, valida en línea y no guarda con Enter",async()=>{
 const saved=[],ui=reasonUI({save:async reason=>saved.push(reason)});
 assert.equal(ui.control.getValues("reason"),"");
 for(const value of ["","   ","a".repeat(241)]){
  ui.control.setValue("reason",value);await ui.render().save.props.onClick();assert.equal(saved.length,0);
  assert.ok(ui.render().all.find(n=>n.type==="FormField"&&n.props.label==="Motivo del cambio").props.error);
 }
 let prevented=0;ui.render().form.props.onSubmit({preventDefault(){prevented++}});assert.equal(prevented,1);assert.equal(saved.length,0);
 ui.control.setValue("reason","  Producción adicional  ");await ui.render().save.props.onClick();assert.deepEqual(saved,["Producción adicional"]);
 const {all,dialog}=ui.render();assert.equal(dialog.props.onResponseClose instanceof Function,true);
 assert.equal(dialog.props.className,"crud-modal compact modal-panel-in");
 assert.equal(all.find(n=>n.type==="FormField"&&n.props.label==="Motivo del cambio").props.className,"span-2");
});

test("el motivo espera una sola respuesta y bloquea campos y cierres",async()=>{
 let count=0,finish;const response=new Promise(resolve=>{finish=resolve});
 const ui=reasonUI({save:async()=>{count++;await response}});ui.control.setValue("reason","Corrección de conteo");
 const first=ui.render().save.props.onClick();ui.render().save.props.onClick();await tick();assert.equal(count,1);
 const {all,dialog}=ui.render();assert.equal(dialog.props["aria-busy"],true);
 assert.ok(all.filter(n=>["Button","Input","Textarea","button"].includes(n.type)).every(n=>n.props.disabled));
 assert.equal(words(ui.render().save),"Guardando…");finish();await first;
});

function managerUI(manage=true){
 const state=[];let cursor=0,mutation,apiCalls=[],feedback=[],invalidations=[];
 const {ProductAvailabilityManager}=compile("modules/menu/availability/presentation/product-availability-manager.tsx",name=>{
  if(name.endsWith(".css"))return {};
  if(name==="react")return {useState(initial){const key=cursor++;if(!(key in state))state[key]=initial;return [state[key],next=>{state[key]=typeof next==="function"?next(state[key]):next}];}};
  if(name==="@/providers/session-context")return {useSession:()=>({organization:{id:"org"},location:{id:"local",country:"PE",timezone:"America/Lima"},can:p=>p==="menu.read"||manage,isLoading:false})};
  if(name==="@/providers")return {useFeedback:()=>({notify:value=>feedback.push(value)})};
  if(name==="@tanstack/react-query")return {
   useQuery:options=>options.queryKey[0]==="product-availability"?{data:{items:[item],total:1,businessDate:item.businessDate}}:{data:{items:[]}},
   useQueryClient:()=>({invalidateQueries:value=>invalidations.push(value.queryKey)}),
   useMutation:options=>{mutation=options;return {isPending:false,mutateAsync:async value=>{try{const result=await options.mutationFn(value);options.onSuccess(result,value);return result}catch(error){options.onError(error);throw error}}};},
  };
  if(name==="@/design-system")return ds;
  if(name==="@/shared/i18n/regional-format")return regional;
  if(name==="../infrastructure/availability-api")return {updateAvailability:async(...args)=>{apiCalls.push(args)},listAvailability(){},listAvailabilityCategories(){}};
  if(name==="./availability-reason-dialog")return {AvailabilityReasonDialog:"Reason"};
  if(name==="./availability-history-dialog")return {AvailabilityHistoryDialog:"History"};
  return require(name);
 });
 function render(){cursor=0;return nodes(ProductAvailabilityManager());}
 return {render,apiCalls,feedback,invalidations,get mutation(){return mutation}};
}

test("Guardar cupo pide motivo antes del API; Agotar conserva el cupo sin guardar",async()=>{
 const ui=managerUI();let all=ui.render();all.find(n=>n.type==="Input"&&n.props.type==="number").props.onChange({target:{value:"20"}});
 all=ui.render();const save=all.find(n=>n.type==="Button"&&n.props["aria-label"]==="Guardar cupo de Ají de gallina");assert.equal(save.props.disabled,false);save.props.onClick();
 assert.equal(ui.apiCalls.length,0);assert.equal(ui.feedback.length,0);
 all=ui.render();const dialog=all.find(n=>n.type==="Reason");assert.equal(dialog.props.change.portionQuantity,20);
 await dialog.props.save("Producción adicional");assert.equal(ui.apiCalls.length,1);
 assert.deepEqual(JSON.parse(JSON.stringify(ui.apiCalls[0])),["aji",{status:"available",portionQuantity:20,note:"Nota conservada",reason:"Producción adicional"}]);
 assert.equal(ui.feedback.length,1);assert.equal(ui.mutation.retry,false);assert.ok(ui.invalidations.some(key=>key[0]==="availability-history"));
 all=ui.render();all.find(n=>n.type==="Input"&&n.props.type==="number").props.onChange({target:{value:"50"}});
 ui.render().find(n=>n.type==="Button"&&n.props.children==="Agotar hoy").props.onClick();
 assert.equal(ui.render().find(n=>n.type==="Reason").props.change.portionQuantity,null);
});

test("solo lectura permite historial pero no editar; su caché y paginación están aisladas",()=>{
 const ui=managerUI(false),all=ui.render();assert.equal(all.find(n=>n.type==="Input"&&n.props.type==="number").props.disabled,true);
 assert.equal(all.some(n=>n.type==="Button"&&n.props.children==="Guardar"),false);
 all.find(n=>n.type==="RowActionButton"&&n.props.label==="Ver historial de Ají de gallina").props.onClick();assert.ok(ui.render().some(n=>n.type==="History"));
 const source=readFileSync(new URL("modules/menu/availability/presentation/availability-history-dialog.tsx",root),"utf8");
 assert.match(source,/\["availability-history",organization\?\.id,location\?\.id,item\.productId,page,size\]/);
 assert.match(source,/<Pagination/);assert.doesNotMatch(source,/>Cerrar<|onKeyDown|Escape/);
});

test("historial muestra skeleton, error con reintento, vacío y valores reales sin inventar históricos",()=>{
 let result={isLoading:true},retried=0;const keys=[];
 const historyModule=compile("modules/menu/availability/presentation/availability-history-dialog.tsx",name=>{
  if(name==="react")return {useState:initial=>[initial,()=>{}]};
  if(name==="@tanstack/react-query")return {useQuery:options=>{keys.push(options.queryKey);return {...result,refetch(){retried++}}}};
  if(name==="@/providers/session-context")return {useSession:()=>({organization:{id:"org"},location:{id:"local",country:"PE",timezone:"America/Lima"},can:()=>true})};
  if(name==="@/design-system")return ds;
  if(name==="@/shared/i18n/regional-format")return regional;
  if(name==="../infrastructure/availability-api")return {listAvailabilityHistory(){}};
  return require(name);
 });
 const render=()=>nodes(historyModule.AvailabilityHistoryDialog({item,close(){}}));
 assert.ok(render().some(n=>n.props["aria-label"]==="Cargando historial"));
 result={isError:true,error:{message:"Error remoto"}};render().find(n=>n.type==="Button").props.onClick();assert.equal(retried,1);
 result={data:{items:[],total:0}};assert.ok(render().some(n=>words(n)==="Aún no hay cambios registrados"));
 const entry={id:"event",createdAt:"2026-10-09T01:00:00Z",businessDate:"2026-10-08",reason:"Nueva producción",userName:"Daniel",previousPortionQuantity:15,portionQuantity:20,soldQuantity:5,previousManualStatus:"available",manualStatus:"available"};
 result={data:{items:[entry],total:1}};const all=render();assert.ok(all.some(n=>n.type==="Pagination"&&n.props.total===1));assert.ok(all.some(n=>n.type==="time"&&/8:00|20:00/.test(words(n))));
 assert.deepEqual(JSON.parse(JSON.stringify(keys[0])),["availability-history","org","local","aji",1,10]);
 assert.match(words(historyModule.HistoryChange({entry,country:"PE"})),/Cupo: 15 → 20/);
 assert.match(words(historyModule.HistoryChange({entry:{...entry,businessDate:null,reason:null},country:"PE"})),/Cambio anterior sin detalle registrado/);
});

test("transporte conserva el motivo y el historial consulta solo una página",async()=>{
 const calls=[];const api=compile("modules/menu/availability/infrastructure/availability-api.ts",name=>name==="@/shared/api/client"?{apiFetch:async(...args)=>calls.push(args)}:require(name));
 await api.updateAvailability("aji",{status:"sold_out",portionQuantity:null,note:"",reason:"Sin producción"});
 assert.equal(JSON.parse(calls[0][1].body).reason,"Sin producción");assert.equal(JSON.parse(calls[0][1].body).portionQuantity,null);
 await api.listAvailabilityHistory("aji",2,10);assert.equal(calls[1][0],"product-availability/aji/history?page=2&pageSize=10");
});

test("a 390 px el motivo usa la rejilla compartida y el historial conserva scroll controlado",()=>{
 const css=postcss.parse(readFileSync(new URL("styles/globals.css",root),"utf8"));const rules=[];css.walkRules(rule=>rules.push(rule));
 assert.ok(rules.some(rule=>rule.selector===".form-grid"&&rule.parent.type==="atrule"&&Number(rule.parent.params.match(/width\s*<=\s*(\d+)px/)?.[1])>=390&&rule.nodes.some(n=>n.prop==="grid-template-columns"&&n.value==="1fr")));
 const history=readFileSync(new URL("modules/menu/availability/presentation/product-availability.css",root),"utf8");assert.match(history,/\.availability-history-modal\{width:min\(var\(--size-960\),100%\)\}/);
 assert.match(history,/\.availability-history-table\{min-width:var\(--size-610\)\}/);
 assert.match(history,/\.availability-history-body \.sk-head,\.availability-history-body \.sk-row\{grid-template-columns:1\.2fr 1\.3fr 1\.6fr 1fr\}/);
});
