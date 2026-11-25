import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url),root=new URL("../src/",import.meta.url);
function compile(path,resolve){const exports={};vm.runInNewContext(ts.transpileModule(readFileSync(new URL(path,root),"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve,URLSearchParams});return exports}
function nodes(tree,out=[]){if(Array.isArray(tree)){tree.forEach(node=>nodes(node,out));return out}if(tree&&typeof tree==="object"){out.push(tree);nodes(tree.props?.children,out);nodes(tree.props?.action,out)}return out}
const regional=compile("shared/i18n/regional-format.ts",require);
const meta=compile("modules/operations/orders/presentation/order-meta.ts",name=>name==="@/shared/i18n/regional-format"?regional:require(name));
const order={id:"order-1",channel:"delivery",status:"en_camino",operationalStatus:"listo",customerName:"Rogelio",total:"20",createdAt:"2026-11-20T20:00:00Z"};
const data={items:[order],total:12,channelCounts:{salon:5,recojo:3,delivery:4},channelOptions:[{value:"salon",label:"Salón"},{value:"recojo",label:"Recojo en local"},{value:"delivery",label:"Delivery"}],statusOptions:[{value:"nuevo",label:"Nuevo"},{value:"listo",label:"Por entregar"},{value:"entregado",label:"Entregado"}],currencySymbol:"S/"};
function mount({response=data,pending=false,error=false,placeholder=false,canManage=true}={}){
 const state=[],queries=[],mutations=[],requests=[],invalidations=[];let index=0;
 const {OrdersManager}=compile("modules/operations/orders/presentation/orders-manager.tsx",name=>{
  if(name==="react")return{useState:initial=>{const slot=index++;if(!(slot in state))state[slot]=initial;return[state[slot],value=>{state[slot]=typeof value==="function"?value(state[slot]):value}]}};
  if(name==="next/dynamic")return{default:()=>"ManualOrderDialog"};
  if(name==="@tanstack/react-query")return{keepPreviousData:value=>value,useQueryClient:()=>({invalidateQueries:query=>invalidations.push(query)}),useMutation:options=>{mutations.push(options);return{}},useQuery:options=>{queries.push(options);return options.queryKey[0]==="orders"?{data:response,isPending:pending,isError:error,error:{message:"Sin conexión"},isPlaceholderData:placeholder,isFetching:placeholder,refetch:()=>{}}:{}}};
  if(name==="@/providers/session-context")return{useSession:()=>({can:()=>canManage,location:{country:"PE",timezone:"America/Lima"}})};
  if(name==="@/providers/settings-context")return{useSettings:()=>({currencySymbol:"S/"})};
  if(name==="@/providers/feedback-provider")return{useFeedback:()=>({notify:()=>{}})};
  if(name==="@/providers")return{useMoney:()=>value=>`S/ ${value}`};
  if(name==="@/shared/hooks/use-debounced-value")return{useDebouncedValue:value=>value};
  if(name==="@/modules/delivery")return{OrderDispatchActions:"OrderDispatchActions"};
  if(name==="./use-order-deep-link")return{useOrderDeepLink:()=>{}};
  if(name==="./order-meta")return meta;
  if(name==="../infrastructure/orders-api")return{listOrders:(...args)=>requests.push(args)};
  if(name.endsWith(".css"))return{};
  if(name==="@/design-system"||name==="@/design-system/dialog")return Object.fromEntries(["Dialog","Button","ConfirmDialog","DataTable","EmptyState","ErrorState","Icon","PageHeader","Pagination","RowActionButton","SearchField","Select","Skeleton","Status","TableSkeleton","Tabs","TabsBar"].map(key=>[key,key]));
  if(name.startsWith("./")||name.startsWith("../"))return{};
  return require(name);
 });
 const render=()=>{index=0;queries.length=0;mutations.length=0;return nodes(OrdersManager())};
 return{render,state,queries,mutations,requests,invalidations};
}

test("Pedidos inicia en Todos y ofrece el catálogo operativo recibido del API",()=>{
 const view=mount(),tree=view.render(),select=tree.find(node=>node.type==="Select");
 assert.equal(select.props.value,"");
 assert.deepEqual(nodes(select).filter(node=>node.type==="option").map(node=>[node.props.value,node.props.children]),[["","Todos"],["nuevo","Nuevo"],["listo","Por entregar"],["entregado","Entregado"]]);
 assert.equal(tree.some(node=>node.props.children==="Limpiar filtros"),false);
 view.queries[0].queryFn({});assert.equal(view.requests[0][0].status,"");
 assert.equal(tree.find(node=>node.type==="Pagination").props.total,12);
});

test("Pedidos muestra Por entregar durante el trayecto y conserva el estado real para las acciones",()=>{
 const tree=mount().render();
 const status=tree.find(node=>node.type==="Status");assert.equal(status.props.children,"Por entregar");assert.equal(status.props.tone,"purple");assert.notEqual(status.props.solid,true,"estado suave en la tabla");
 assert.equal(tree.find(node=>node.type==="OrderDispatchActions").props.order.status,"en_camino");
 const delivered=mount({response:{...data,items:[{...order,status:"entregado",operationalStatus:"entregado"}]}}).render();
 assert.equal(delivered.find(node=>node.type==="Status").props.children,"Entregado");
});

test("Pedidos mantiene contadores por canal para Todos y estados específicos",()=>{
 const view=mount();let tree=view.render();
 assert.deepEqual(Array.from(tree.find(node=>node.type==="Tabs").props.items,item=>item.count),[12,5,3,4]);
 view.state[3]=4;tree.find(node=>node.type==="Select").props.onChange({target:{value:"entregado"}});tree=view.render();
 assert.equal(view.state[3],1);assert.equal(tree.find(node=>node.type==="Select").props.value,"entregado");
 assert.deepEqual(Array.from(tree.find(node=>node.type==="Tabs").props.items,item=>item.count),[12,5,3,4]);
 tree.find(node=>node.type==="Tabs").props.onChange("salon");view.render();view.queries[0].queryFn({});assert.equal(view.requests.at(-1)[0].channel,"salon");assert.equal(view.requests.at(-1)[0].status,"entregado");
});

test("Pedidos limpia filtros y después de crear vuelve a Todos, sin reintroducir Abiertos",()=>{
 const view=mount();view.render();view.state[0]="cliente";view.state[1]="salon";view.state[2]="entregado";view.state[3]=3;
 view.render().find(node=>node.type==="Button"&&node.props.children==="Limpiar filtros").props.onClick();view.render();assert.deepEqual(view.state.slice(0,4),["","","",1]);
 view.state[2]="cancelado";view.state[3]=4;view.render();view.mutations[0].onSuccess({...order,channel:"recojo"});view.render();assert.deepEqual(view.state.slice(0,4),["","recojo","",1]);assert.ok(view.invalidations.some(query=>query.queryKey[0]==="orders"));
});

test("Pedidos conserva skeleton, error y vacío sin inventar contadores",()=>{
 assert.ok(mount({response:undefined,pending:true}).render().some(node=>node.type==="TableSkeleton"));
 const failed=mount({error:true}).render();assert.ok(failed.some(node=>node.type==="ErrorState"));assert.ok(failed.find(node=>node.type==="Tabs").props.items.every(item=>item.count===undefined));
 const view=mount({response:{...data,items:[],total:0,channelCounts:{}}});assert.equal(view.render().find(node=>node.type==="EmptyState").props.title,"Aún no hay pedidos");
 view.state[2]="listo";assert.equal(view.render().find(node=>node.type==="EmptyState").props.title,"Sin coincidencias");
});

test("Pedidos omite filtros vacíos de la URL y transmite el estado operativo seleccionado",()=>{
 const calls=[];const api=compile("modules/operations/orders/infrastructure/orders-api.ts",name=>name==="@/shared/api/client"?{apiFetch:(...args)=>calls.push(args)}:name==="../domain/manual-order-schema"?{}:require(name));
 const query={q:"",channel:"",status:"",page:1,pageSize:10};api.listOrders(query);let params=new URL(calls[0][0],"https://example.test/").searchParams;assert.equal(params.has("status"),false);assert.equal(params.has("channel"),false);
 const signal={};api.listOrders({...query,channel:"delivery",status:"listo"},signal);params=new URL(calls[1][0],"https://example.test/").searchParams;assert.equal(params.get("status"),"listo");assert.equal(params.get("channel"),"delivery");assert.equal(calls[1][1].signal,signal);
});
