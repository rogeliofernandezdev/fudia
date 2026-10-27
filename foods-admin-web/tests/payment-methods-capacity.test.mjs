import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url);
const file=new URL("../src/modules/configuration/presentation/payment-methods-page.tsx",import.meta.url);
const source=ts.transpileModule(readFileSync(file,"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText;
const item={code:"cash",name:"Efectivo",description:"",active:true,salesEnabled:true,expensesEnabled:true,affectsCash:true,sortOrder:10};
function nodes(tree,out=[]){if(Array.isArray(tree)){tree.forEach(x=>nodes(x,out));return out}if(tree&&typeof tree==="object"){out.push(tree);nodes(tree.props?.children,out)}return out}
function mount({catalogTotal=4,catalogLimit=4,total=1,items=[item],loading=false,error=false,allowed=true}={}){
 const state=["","",1,20,null,undefined,null],mutations=[],invalidations=[],queries=[];let cursor=0;
 const exports={};
 vm.runInNewContext(source,{exports,require:name=>{
  if(name.endsWith(".css"))return{};
  if(name==="react")return{useState:()=>{const n=cursor++;return[state[n],value=>{state[n]=value}]}};
  if(name==="@tanstack/react-query")return{useQueryClient:()=>({invalidateQueries:options=>invalidations.push(options.queryKey)}),useQuery:options=>{queries.push(options);return{data:{items,total,catalogTotal,catalogLimit,page:1,pageSize:20},isLoading:loading,isError:error,error:new Error("Sin conexión")}},useMutation:options=>{mutations.push(options);return{isPending:false,mutate:()=>{}}}};
  if(name==="@/providers/session-context")return{useSession:()=>({can:()=>allowed,organization:{id:"restaurant-a"}})};
  if(name==="@/providers/feedback-provider")return{useFeedback:()=>({notify:()=>{}})};
  if(name==="@/design-system")return Object.fromEntries(["Button","PageHeader","Pagination","FormField","Input","Select","SearchField","Textarea","Status","RowActionButton","EmptyState","ErrorState","TableSkeleton","FormSection","ModalHeader","ModalForm","ModalBody","ModalFooter"].map(x=>[x,x]));
  if(name==="@/design-system/data-table")return{DataTable:"DataTable"};
  if(name==="@/design-system/dialog")return{Dialog:"Dialog"};
  if(name==="@/design-system/confirm-dialog")return{ConfirmDialog:"ConfirmDialog"};
  if(name==="@/design-system/icons")return{Icon:"Icon"};
  if(name.startsWith("../"))return{};
  return require(name);
 }});
 return{render:()=>{cursor=0;return nodes(exports.PaymentMethodsPage())},state,mutations,invalidations,queries};
}

test("el catálogo completo oculta Nuevo medio aunque los filtros o la página no muestren filas",()=>{
 for(const options of [{catalogTotal:4},{catalogTotal:4,total:0,items:[]},{catalogTotal:5,total:0,items:[]}]){
  const app=mount(options);app.state[0]="sin coincidencias";app.state[2]=9;
  const view=app.render();
  assert.equal(view.find(x=>x.type==="PageHeader").props.action,undefined);
  assert.equal(view.filter(x=>x.type==="EmptyState").some(x=>x.props.action),false);
 }
});

test("con una plaza disponible permite crear; conserva editar cuando el catálogo está completo",()=>{
 const app=mount({catalogTotal:3});
 const view=app.render();
 const create=view.find(x=>x.type==="PageHeader").props.action;
 assert.equal(create.props.children,"Nuevo medio");create.props.onClick();
 assert.ok(app.render().some(x=>x.type?.name==="PaymentMethodDialog"&&x.props.editing===false));
 const full=mount({catalogTotal:4});
 full.render().find(x=>x.type==="RowActionButton"&&x.props.action==="edit").props.onClick();
 assert.ok(full.render().some(x=>x.type?.name==="PaymentMethodDialog"&&x.props.editing===true));
});

test("la capacidad viene del API y no habilita altas durante carga, error o sin permiso",()=>{
 assert.ok(mount({catalogTotal:4,catalogLimit:5}).render().find(x=>x.type==="PageHeader").props.action);
 for(const options of [{loading:true},{error:true},{allowed:false}])assert.equal(mount({catalogTotal:3,...options}).render().find(x=>x.type==="PageHeader").props.action,undefined);
 const app=mount();app.render();app.mutations[0].onError(new Error("Límite alcanzado por otra sesión"));
 assert.ok(app.invalidations.some(key=>key[0]==="payment-methods-admin"));
 assert.equal(app.queries[0].queryKey[1],"restaurant-a");
});
