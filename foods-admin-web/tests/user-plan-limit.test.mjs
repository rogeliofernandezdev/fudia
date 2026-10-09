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
function nodes(tree,result=[]){if(Array.isArray(tree)){tree.forEach(node=>nodes(node,result));return result}if(tree&&typeof tree==="object"){result.push(tree);nodes(tree.props?.children,result)}return result}
const active={id:"admin",fullName:"Panchita",email:"owner@example.test",active:true,platformAdmin:false,assignments:[]};
const inactive={...active,id:"waiter",active:false,fullName:"Mozo"};
function mount({max=6,usage=6,items=[active,inactive],pending=false,error=false,fetching=false,readPlan=true,manage=true,hasData=true}={}){
 const state=[],mutations=[],queries=[],invalidations=[],written=[];let cursor=0,mutationCursor=0,retries=0;
 const session={organization:{id:"panchita"},user:{id:"owner",platformAdmin:false},modules:{usuarios:true},can:permission=>permission==="users.manage"?manage:readPlan};
 const subscription={data:hasData?{plan:{maxUsers:max},usage:{users:usage}}:undefined,isLoading:pending,isError:error,isFetching:fetching,refetch:()=>{retries++}};
 const capacity=compile("modules/identity/application/use-subscription-capacity.ts",name=>{
  if(name==="@/providers/session-context")return {useSession:()=>session};
  if(name==="./use-organization-subscription")return {useOrganizationSubscription:()=>subscription};
  return require(name);
 });
 const {UsersRolesManager}=compile("modules/identity/presentation/users-roles-manager.tsx",name=>{
  if(name==="react")return {useState:initial=>{const index=cursor++;if(!(index in state))state[index]=initial;return[state[index],value=>{state[index]=value}]} ,useMemo:fn=>fn()};
  if(name==="@tanstack/react-query")return {useQueryClient:()=>({invalidateQueries:async options=>{invalidations.push(options.queryKey)}}),useMutation:options=>{const index=mutationCursor++;mutations[index]=options;return {isPending:false,mutate:value=>written.push({index,value})}},useQuery:options=>{queries.push(options);return {data:{items:options.queryKey[0]==="users"?items:[],total:items.length,menuGroups:[],groups:[]},isLoading:false,isError:false}}};
  if(name==="../application/use-subscription-capacity")return capacity;
  if(name==="@/providers/session-context")return {useSession:()=>session};
  if(name==="@/providers/feedback-provider")return {useFeedback:()=>({notify:()=>{}})};
  if(name==="@/design-system")return Object.fromEntries(["Dialog","ConfirmDialog","RemoteModalSkeleton","Icon","Button","FormField","Input","PageHeader","Pagination","RowActionButton","Select","Status","Textarea"].map(key=>[key,key]));
  if(name==="@/design-system/dialog")return {Dialog:"Dialog"};
  if(name==="react-hook-form"||name.startsWith("../")||name.endsWith(".css"))return {};
  return require(name);
 });
 function render(){cursor=mutationCursor=0;const view=nodes(UsersRolesManager());return {view,header:view.find(node=>node.type==="PageHeader"),button:view.find(node=>node.type==="Button"&&["Nuevo usuario","Nuevo rol"].includes(node.props.children))};}
 return {render,subscription,mutations,queries,invalidations,written,get retries(){return retries}};
}

test("cupo de usuarios alcanzado bloquea alta y reactivación, no edición ni desactivación",()=>{
 const ui=mount(),first=ui.render();
 assert.equal(first.button.props.disabled,true);assert.match(first.header.props.description,/Tu plan incluye 6 usuarios activos/);
 first.button.props.onClick();assert.equal(ui.render().view.some(node=>node.type?.name==="UserDialog"),false);
 const activate=first.view.find(node=>node.type==="RowActionButton"&&node.props.action==="activate");
 assert.equal(activate.props.disabled,true);activate.props.onClick();
 assert.equal(ui.render().view.find(node=>node.type==="ConfirmDialog"&&node.props.title==="Activar usuario").props.open,false);
 first.view.find(node=>node.type==="RowActionButton"&&node.props.action==="edit").props.onClick();
 assert.equal(ui.render().view.find(node=>node.type?.name==="UserDialog").props.value.id,active.id);
 first.view.find(node=>node.type==="RowActionButton"&&node.props.action==="deactivate").props.onClick();
 const confirm=ui.render().view.find(node=>node.type==="ConfirmDialog"&&node.props.title==="Desactivar usuario");
 assert.equal(confirm.props.open,true);assert.equal(confirm.props.confirmDisabled,false);
 confirm.props.onConfirm();assert.equal(ui.written.length,1);
});

test("con cupo libre, primer usuario y plan ilimitado se permite registrar",()=>{
 for(const options of [{usage:5},{usage:0,items:[]},{max:null,usage:80}]){
  const ui=mount(options),first=ui.render();assert.equal(first.button.props.disabled,false);
  first.button.props.onClick();assert.equal(ui.render().view.find(node=>node.type?.name==="UserDialog").props.value.id,undefined);
 }
 assert.equal(mount({max:2,usage:2}).render().button.props.disabled,true);
 assert.equal(mount({max:2,usage:1}).render().button.props.disabled,false);
});

test("el cupo usa el uso activo remoto y no los resultados de búsqueda/paginación",()=>{
 assert.equal(mount({usage:6,items:[inactive]}).render().button.props.disabled,true);
 assert.equal(mount({usage:5,items:Array.from({length:20},()=>active)}).render().button.props.disabled,false);
 assert.equal(mount({usage:6,items:[]}).render().button.props.disabled,true);
});

test("carga, error, refresco y falta de acceso bloquean el alta; el error ofrece reintentar",()=>{
 for(const options of [{pending:true,hasData:false},{fetching:true,usage:1},{error:true,usage:1},{readPlan:false},{hasData:false}]){
  const ui=mount(options),first=ui.render();assert.equal(first.button.props.disabled,true);
  first.button.props.onClick();assert.equal(ui.render().view.some(node=>node.type?.name==="UserDialog"),false);
 }
 const ui=mount({error:true}),first=ui.render();assert.match(first.header.props.description,/No pudimos comprobar el límite de usuarios/);
 first.view.find(node=>node.type==="Button"&&node.props.children==="Reintentar").props.onClick();assert.equal(ui.retries,1);
 assert.equal(mount({manage:false}).render().button,undefined);
 assert.ok(mount({pending:true,hasData:false}).render().view.some(node=>node.type?.name==="Skeleton"));
});

test("Nuevo rol permanece habilitado al llegar al límite de usuarios",()=>{
 const ui=mount();ui.render().view.find(node=>node.type==="button"&&node.props.children[0]==="Roles ").props.onClick();
 const roles=ui.render();assert.equal(roles.button.props.children,"Nuevo rol");assert.equal(roles.button.props.disabled,false);
 roles.button.props.onClick();assert.ok(ui.render().view.some(node=>node.type?.name==="RoleDialog"));
});

test("confirmación abierta bloquea activar si otro registro consume el último cupo",()=>{
 const ui=mount({usage:5}),first=ui.render();
 first.view.find(node=>node.type==="RowActionButton"&&node.props.action==="activate").props.onClick();
 assert.equal(ui.render().view.find(node=>node.type==="ConfirmDialog"&&node.props.title==="Activar usuario").props.open,true);
 ui.subscription.data.usage.users=6;
 const confirm=ui.render().view.find(node=>node.type==="ConfirmDialog"&&node.props.title==="Activar usuario");
 assert.equal(confirm.props.confirmDisabled,true);assert.match(confirm.props.description,/Ya alcanzaste ese límite/);
 confirm.props.onConfirm();assert.equal(ui.written.length,0);
 confirm.props.onCancel();assert.equal(ui.render().view.find(node=>node.type==="ConfirmDialog"&&node.props.title==="Activar usuario").props.open,false);
});

test("las mutaciones de usuario refrescan el cupo; consultas aisladas por empresa",async()=>{
 const ui=mount();ui.render();
 await ui.mutations[0].onSuccess();await ui.mutations[2].onSuccess();await ui.mutations[0].onError(new Error("Límite alcanzado"));await ui.mutations[2].onError(new Error("Límite alcanzado"));
 assert.equal(ui.invalidations.filter(key=>key[0]==="organization-subscription").length,4);
 assert.ok(ui.queries.some(query=>query.queryKey[0]==="users"&&query.queryKey[1]==="panchita"));
 assert.ok(ui.queries.some(query=>query.queryKey[0]==="locations"&&query.queryKey[2]==="panchita"));
});

test("confirmación compartida deshabilita solo confirmar si el cupo cambia, sin impedir Cancelar",()=>{
 let cancelled=0;
 const {ConfirmDialog}=compile("design-system/confirm-dialog.tsx",name=>{
  if(name==="react")return {...require(name),useId:()=>"confirm-quota"};
  if(name.endsWith(".css"))return {};
  if(name==="./dialog")return {Dialog:"Dialog"};
  if(name==="@/design-system/icons")return {Icon:"Icon"};
  return require(name);
 });
 const view=nodes(ConfirmDialog({open:true,title:"Activar usuario",confirmLabel:"Activar",confirmDisabled:true,onCancel:()=>{cancelled++},onConfirm:()=>{}}));
 const cancel=view.find(node=>node.type==="button"&&node.props.children==="Cancelar");
 const confirm=view.find(node=>node.type==="button"&&node.props.children==="Activar");
 assert.equal(cancel.props.disabled,false);assert.equal(confirm.props.disabled,true);
 cancel.props.onClick();assert.equal(cancelled,1);
});
