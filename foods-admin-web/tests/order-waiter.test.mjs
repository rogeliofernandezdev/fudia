import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url),root=new URL("../src/",import.meta.url);
function compile(file,resolve=require,extra=""){
 const exports={};
 vm.runInNewContext(ts.transpileModule(readFileSync(new URL(file,root),"utf8")+extra,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve});
 return exports;
}
const actions=compile("modules/operations/orders/domain/order-actions.ts");
const attribution=compile("modules/operations/orders/presentation/order-attribution.tsx",name=>name==="@/design-system/icons"?{Icon:"Icon"}:require(name));
const order={id:"order",tableName:"Mesa 04",channel:"salon",status:"nuevo",waiterId:"ana",waiterName:"Ana",collectedByNames:[],items:[],createdAt:"2026-10-07T14:00:00Z",total:"25",subtotal:"25",deliveryFee:"0"};
let viewer="ana",cash=false;
function resolve(name){
 if(name==="next/dynamic")return{default:()=>"ManualOrderDialog"};
 if(name==="next/link")return{default:"Link"};
 if(name==="react")return{...require(name),useState:value=>[value,()=>{}],useCallback:fn=>fn};
 if(name==="@tanstack/react-query")return{useQueryClient:()=>({invalidateQueries:()=>{}}),useMutation:()=>({}),useQuery:()=>({data:{items:[order],channelCounts:{salon:1},channelOptions:[{value:"salon",label:"Salón"}],statusOptions:[],total:1},isPending:false,isError:false})};
 if(name==="@/design-system")return Object.fromEntries(["Button","Status","ConfirmDialog","PageHeader","Pagination","RowActionButton"].map(name=>[name,name]));
 if(name==="@/design-system/icons")return{Icon:"Icon"};
 if(name==="@/providers/session-context")return{useSession:()=>({user:{id:viewer},location:{country:"PE",timezone:"America/Lima"},can:permission=>permission==="cash.manage"?cash:true})};
 if(name==="@/providers/feedback-provider")return{useFeedback:()=>({notify:()=>{}})};
 if(name==="@/providers/settings-context")return{useSettings:()=>({currencySymbol:"S/"})};
 if(name==="@/shared/hooks/use-debounced-value")return{useDebouncedValue:value=>value};
 if(name==="@/shared/routing/page-routes")return{pageRoutes:{pos:"/pos"}};
 if(name==="@/shared/i18n/regional-format")return{formatRegionalDateTime:value=>value};
 if(name.endsWith("order-actions"))return actions;
 if(name.endsWith("order-attribution"))return attribution;
 if(name.startsWith("@/")||name.startsWith("."))return{};
 return require(name);
}
const orders=compile("modules/operations/orders/presentation/orders-manager.tsx",resolve,"\nexport {OrderDetail};");
const salon=compile("modules/operations/salon/presentation/salon-manager.tsx",resolve,"\nexport {OrderDetail};");
function nodes(tree){const all=[];function visit(node){if(!node||typeof node!=="object")return;if(Array.isArray(node)){node.forEach(visit);return}all.push(node);if(typeof node.type==="function")visit(node.type(node.props));else visit(node.props?.children)}visit(tree);return all}
function text(tree){return nodes(tree).flatMap(node=>typeof node.props?.children==="string"?[node.props.children]:Array.isArray(node.props?.children)?node.props.children.filter(child=>typeof child==="string"):[]).join(" ")}
const props={loading:false,order,currencySymbol:"S/",canManage:true,busy:false,close:()=>{},advance:()=>{},edit:()=>{},cancel:()=>{}};

test("la atención pertenece al mozo de la comanda, no a quien consulta",()=>{
 assert.equal(actions.canManageOrderService(order,"ana"),true);
 assert.equal(actions.canManageOrderService(order,"luis"),false);
 assert.equal(actions.canManageOrderService(order,undefined),false);
 assert.equal(actions.canManageOrderService({...order,channel:"delivery",waiterId:""},"luis"),true);
});
test("otro mozo ve el detalle sin ninguna acción en Salón ni Pedidos",()=>{
 viewer="luis";cash=false;
 for(const component of [salon.OrderDetail,orders.OrderDetail]){
  const tree=component(props),all=nodes(tree);
  assert.match(text(tree),/Ana/);
  assert.equal(all.filter(node=>node.type==="Button"||node.type==="Link").length,0);
  assert.equal(all.filter(node=>node.props?.["aria-label"]==="Cerrar detalle").length,1);
 }
});
test("el mozo asignado conserva sus acciones; el permiso sigue siendo obligatorio",()=>{
 viewer="ana";cash=false;
 for(const component of [salon.OrderDetail,orders.OrderDetail]){
  assert.ok(nodes(component(props)).some(node=>node.type==="Button"&&node.props.children==="Enviar a cocina"));
  assert.equal(nodes(component({...props,canManage:false})).filter(node=>node.type==="Button"||node.type==="Link").length,0);
 }
});
test("el cajero puede cobrar, pero no editar ni confirmar la entrega del mozo",()=>{
 viewer="eva";cash=true;
 const tree=salon.OrderDetail({...props,canManage:false,order:{...order,status:"listo",remainingAmount:"25"}}),all=nodes(tree);
 assert.equal(all.filter(node=>node.type==="Button").length,0);
 assert.equal(all.filter(node=>node.type==="Link"&&node.props.href==="/pos?orderId=order").length,1);
});
test("la lista identifica al mozo y al cobrador sin confundirlos",()=>{
 viewer="luis";cash=false;order.collectedByNames=["Eva"];
 const rendered=text(orders.OrdersManager());
 assert.match(rendered,/Mozo:.*Ana/);
 assert.match(rendered,/Cobrado por:.*Eva/);
 const detail=text(attribution.OrderAttribution({...order,collectedByNames:["Eva","Pablo"]}));
 assert.match(detail,/Mozo.*Ana.*Cobrado por.*Eva · Pablo/);
 const unpaid=text(attribution.OrderAttribution({...order,collectedByNames:[]}));
 assert.doesNotMatch(unpaid,/Cobrado por/);
 order.collectedByNames=[];
});
