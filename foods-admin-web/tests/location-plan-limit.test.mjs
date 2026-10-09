import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url),root=new URL("../src/",import.meta.url);
function compile(path,resolve){
 const exports={};
 vm.runInNewContext(ts.transpileModule(readFileSync(new URL(path,root),"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve});
 return exports;
}
function nodes(tree,result=[]){if(Array.isArray(tree)){tree.forEach(node=>nodes(node,result));return result}if(tree&&typeof tree==="object"){result.push(tree);nodes(tree.props?.children,result);nodes(tree.props?.action,result)}return result}
const local={id:"local",name:"La Panchita",country:"PE",currency:"PEN",active:true};
function mount({max=1,usage=1,items=[local],pending=false,error=false,fetching=false,manage=true,readPlan=true,hasData=true}={}){
 const mutations=[],queries=[],invalidations=[];let cursor=0,retries=0;
 const state=[];
 const subscription={data:hasData?{plan:{maxLocations:max},usage:{locations:usage}}:undefined,isLoading:pending,isError:error,isFetching:fetching,refetch:()=>{retries++}};
 const capacity=compile("modules/identity/application/use-subscription-capacity.ts",name=>{
  if(name==="@/providers/session-context")return {useSession:()=>({can:()=>readPlan})};
  if(name==="./use-organization-subscription")return {useOrganizationSubscription:()=>subscription};
  return require(name);
 });
 const {LocationsManager}=compile("modules/organizations/presentation/organization-admin.tsx",name=>{
  if(name==="react")return {useState:initial=>{const index=cursor++;if(!(index in state))state[index]=initial;return[state[index],value=>{state[index]=value}]}};
  if(name==="@tanstack/react-query")return {useQueryClient:()=>({invalidateQueries:async options=>{invalidations.push(options.queryKey)}}),useMutation:options=>{mutations.push(options);return {isPending:false}},useQuery:options=>{queries.push(options);return {data:{items:options.queryKey[0]==="locations"?items:[],total:items.length},isLoading:false,isError:false}}};
  if(name==="@/modules/identity")return capacity;
  if(name==="@/providers/session-context")return {useSession:()=>({organization:{id:"panchita"},can:permission=>permission==="organizations.manage"?manage:readPlan})};
  if(name==="@/providers/feedback-provider")return {useFeedback:()=>({notify:()=>{}})};
  if(name.startsWith("@/design-system"))return Object.fromEntries(["Dialog","ConfirmDialog","RemoteModalSkeleton","Icon","LocationMap","Button","FormField","Input","PageHeader","Pagination","RowActionButton","Select","Status"].map(key=>[key,key]));
  if(name==="react-hook-form"||name.startsWith("../")||name.endsWith(".css")||name==="@/shared/i18n/regional-format")return {};
  return require(name);
 });
 function render(){cursor=0;const view=nodes(LocationsManager());return {view,header:view.find(node=>node.type==="PageHeader"),button:view.find(node=>node.type==="Button"&&node.props.children==="Nuevo local")};}
 return {render,subscription,mutations,queries,invalidations,get retries(){return retries}};
}

test("Emprende con un local bloquea Nuevo local también desde el handler, pero permite editar",()=>{
 const ui=mount(),initial=ui.render();
 assert.equal(initial.button.props.disabled,true);
 assert.match(initial.header.props.description,/Tu plan incluye 1 local activo/);
 initial.button.props.onClick();
 assert.equal(ui.render().view.some(node=>node.type?.name==="LocationDialog"),false);
 initial.view.find(node=>node.type==="RowActionButton"&&node.props.action==="edit").props.onClick();
 assert.equal(ui.render().view.find(node=>node.type?.name==="LocationDialog").props.value.id,local.id);
});

test("cero locales permite el primero; el límite remoto admite más locales y null no limita",()=>{
 for(const options of [{usage:0,items:[]},{max:3,usage:2},{max:null,usage:20}]){
  const ui=mount(options),initial=ui.render();
  assert.equal(initial.button.props.disabled,false);
  initial.button.props.onClick();
  assert.equal(ui.render().view.find(node=>node.type?.name==="LocationDialog").props.value.id,undefined);
 }
 for(const options of [{max:3,usage:3},{max:3,usage:4}])assert.equal(mount(options).render().button.props.disabled,true);
});

test("el límite usa el total activo del API, no filas de la página ni locales inactivos",()=>{
 assert.equal(mount({max:3,usage:3,items:[local]}).render().button.props.disabled,true);
 assert.equal(mount({usage:0,items:[{...local,active:false}]}).render().button.props.disabled,false);
 const ui=mount({usage:1,items:[]}),view=ui.render();
 const empty=view.view.find(node=>node.type?.name==="Empty");
 assert.equal(empty.props.disabled,true);
 assert.equal(nodes(empty.type(empty.props)).find(node=>node.type==="Button").props.disabled,true);
 empty.props.action();assert.equal(ui.render().view.some(node=>node.type?.name==="LocationDialog"),false);
});

test("carga, refresco, error y falta de permiso no habilitan creación sin validar capacidad",()=>{
 for(const options of [{pending:true,hasData:false},{fetching:true,usage:0},{error:true,usage:0},{hasData:false},{readPlan:false}]){
  const ui=mount(options),initial=ui.render();
  assert.equal(initial.button.props.disabled,true);
  initial.button.props.onClick();assert.equal(ui.render().view.some(node=>node.type?.name==="LocationDialog"),false);
 }
 const ui=mount({error:true});const view=ui.render();
 assert.match(view.header.props.description,/No pudimos comprobar/);
 view.view.find(node=>node.type==="Button"&&node.props.children==="Reintentar").props.onClick();assert.equal(ui.retries,1);
 assert.equal(mount({manage:false}).render().button,undefined);
 assert.ok(mount({pending:true,hasData:false}).render().view.some(node=>node.type?.name==="LoadingTable"));
});

test("crear, editar, rechazar por límite y desactivar refrescan el uso del plan",async()=>{
 const ui=mount();ui.render();
 await ui.mutations[0].onSuccess();
 await ui.mutations[1].onSuccess();
 await ui.mutations[0].onError(new Error("Límite alcanzado"));
 assert.equal(ui.invalidations.filter(key=>key[0]==="organization-subscription").length,3);
 assert.ok(ui.queries.filter(query=>query.queryKey[0]==="locations"||query.queryKey[0]==="fiscal-profiles-options").every(query=>query.queryKey[1]==="panchita"));
});

test("la consulta compartida aísla suscripción por empresa/usuario y respeta permisos",()=>{
 function query(organization,user,allowed){
  const {useOrganizationSubscription:subscriptionQuery}=compile("modules/identity/application/use-organization-subscription.ts",name=>{
   if(name==="@tanstack/react-query")return {useQuery:options=>options};
   if(name==="@/providers/session-context")return {useSession:()=>({organization:organization?{id:organization}:null,user:user?{id:user}:null,can:()=>allowed})};
   if(name==="../infrastructure/identity-api")return {getOrganizationSubscription:()=>{}};
   return require(name);
  });
  return subscriptionQuery();
 }
 const a=query("a","u1",true),b=query("b","u1",true),c=query("a","u2",true);
 assert.notDeepEqual(a.queryKey,b.queryKey);assert.notDeepEqual(a.queryKey,c.queryKey);
 assert.equal(a.enabled,true);assert.equal(a.staleTime,0);assert.equal(a.refetchOnWindowFocus,"always");assert.equal(a.refetchInterval,30000);
 assert.equal(query("a","u1",false).enabled,false);assert.equal(query(null,"u1",true).enabled,false);
 assert.ok(readFileSync(new URL("modules/identity/presentation/profile-page.tsx",root),"utf8").includes("const subscription=useOrganizationSubscription()"));
});
