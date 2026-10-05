import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url);
const root=new URL("../src/modules/operations/",import.meta.url);
function compile(path,resolve=require,extra=""){
 const exports={};
 const source=readFileSync(new URL(path,root),"utf8")+extra;
 vm.runInNewContext(ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve});
 return exports;
}
const actions=compile("orders/domain/order-actions.ts");
const order={id:"pedido-1",code:"PED-001",channel:"salon",status:"listo",paymentStatus:"pending",customerName:"",customerPhone:"",tableName:"Mesa 1",tableId:"mesa-1",total:"25.00",subtotal:"25.00",deliveryFee:"0",paidAmount:"0",remainingAmount:"25.00",createdAt:"2026-10-05T10:00:00Z",updatedAt:"2026-10-05T10:00:00Z",items:[]};

test("la entrega de mesa no depende de haber cobrado",()=>{
 for(const paymentStatus of ["pending","partial","paid"]){
  const action=actions.nextOrderAction({...order,paymentStatus});
  assert.equal(action.status,"entregado");
  assert.equal(action.label,"Marcar como entregado");
 }
});

test("una mesa entregada solo ofrece liberar cuando está pagada y sigue abierta",()=>{
 assert.equal(actions.nextOrderAction({...order,status:"entregado"}),null);
 assert.equal(actions.nextOrderAction({...order,status:"entregado",paymentStatus:"partial"}),null);
 assert.equal(actions.nextOrderAction({...order,status:"entregado",paymentStatus:"paid"}).label,"Liberar mesa");
 assert.equal(actions.nextOrderAction({...order,status:"entregado",paymentStatus:"paid",completedAt:"2026-10-05T12:00:00Z"}),null);
});

test("Cocina conserva el control de preparación y delivery conserva su despacho",()=>{
 assert.equal(actions.nextOrderAction({...order,status:"confirmado"}),null);
 assert.equal(actions.nextOrderAction({...order,status:"preparando"}),null);
 assert.equal(actions.nextOrderAction({...order,status:"cancelado"}),null);
 assert.equal(actions.nextOrderAction({...order,channel:"delivery"}).status,"en_camino");
 assert.equal(actions.nextOrderAction({...order,channel:"delivery",status:"en_camino"}).status,"entregado");
});

function mount(section,changes={},canManage=true,busy=false){
 const calls=[];
 const file=section==="salon"?"salon/presentation/salon-manager.tsx":"orders/presentation/orders-manager.tsx";
 const {OrderDetail}=compile(file,name=>{
  if(name==="next/link")return{default:"Link"};
  if(name==="@/design-system")return{Button:"Button",Status:"Status",Icon:"Icon"};
  if(name==="@/design-system/icons")return{Icon:"Icon"};
  if(name==="@/providers/session-context")return{useSession:()=>({location:{country:"PE",timezone:"America/Lima"}})};
  if(name==="@/shared/i18n/regional-format")return{formatRegionalDateTime:()=>"Hace un momento"};
  if(name.endsWith("/domain/order-actions"))return actions;
  if(name.startsWith("@/")||name.startsWith("../")||name.startsWith("./"))return{};
  return require(name);
 },"\nexport {OrderDetail};\n");
 const rendered=OrderDetail({loading:false,order:{...order,...changes},currencySymbol:"S/",channels:[{value:"salon",label:"Salón"}],canManage,busy,close:()=>{},advance:status=>calls.push(status),edit:()=>{},cancel:()=>{}});
 const nodes=[];
 function visit(node){if(!node||typeof node!=="object")return;if(Array.isArray(node)){node.forEach(visit);return}nodes.push(node);visit(node.props?.children)}
 visit(rendered);
 return{calls,buttons:nodes.filter(n=>n.type==="Button"),links:nodes.filter(n=>n.type==="Link")};
}

for(const section of ["salon","orders"]){
 test(`${section}: el detalle ofrece la acción real de entrega antes de cobrar`,()=>{
  const view=mount(section);
  const button=view.buttons.find(n=>n.props.children==="Marcar como entregado");
  assert.ok(button);
  assert.equal(button.props.disabled,false);
  button.props.onClick();
  assert.deepEqual(view.calls,["entregado"]);
 });
 test(`${section}: la entrega pendiente conserva acceso al cobro y luego a liberar mesa`,()=>{
  const pending=mount(section,{status:"entregado"});
  assert.equal(pending.buttons.some(n=>n.props.children==="Liberar mesa"),false);
  assert.ok(pending.links.some(n=>n.props.href==="/pos?orderId=pedido-1"));
  const paid=mount(section,{status:"entregado",paymentStatus:"paid",paidAmount:"25.00",remainingAmount:"0"});
  assert.ok(paid.buttons.some(n=>n.props.children==="Liberar mesa"));
 });
 test(`${section}: la acción respeta permisos y bloquea el botón mientras guarda`,()=>{
  assert.equal(mount(section,{},false).buttons.some(n=>n.props.children==="Marcar como entregado"),false);
  assert.equal(mount(section,{},true,true).buttons.find(n=>n.props.children==="Marcar como entregado").props.disabled,true);
 });
}

test("POS permite corregir pagos de una mesa entregada abierta y los bloquea tras el cierre",()=>{
 const {POSDetailDialog}=compile("pos/presentation/pos-page.tsx",name=>{
  if(name==="./pos-meta")return compile("pos/presentation/pos-meta.ts");
  if(name==="@/design-system")return{Button:"Button",Status:"Status",Icon:"Icon"};
  if(name==="@/providers/session-context")return{useSession:()=>({location:{country:"PE",timezone:"America/Lima"}})};
  if(name==="@/shared/i18n/regional-format")return{formatRegionalDateTime:()=>"Hace un momento"};
  if(name.startsWith("@/")||name.startsWith("../")||name.startsWith("./"))return{};
  return require(name);
 },"\nexport {POSDetailDialog};\n");
 for(const completedAt of [undefined,"2026-10-05T12:00:00Z"]){
  const rendered=POSDetailDialog({loading:false,data:{order:{...order,status:"entregado",completedAt},paidAmount:"5",remainingAmount:"20",paymentStatus:"partial",payments:[{id:"pago-1",method:"card",amount:"5",netAmount:"5",refundedAmount:"0",createdAt:order.createdAt}]},canManage:true,hasShift:true,formatMoney:value=>String(value),formatDateTime:()=>"Hace un momento",close:()=>{},refund:()=>{}});
  const buttons=[];
  function visit(node){if(!node||typeof node!=="object")return;if(Array.isArray(node)){node.forEach(visit);return}if(node.type==="Button")buttons.push(node);visit(node.props?.children)}
  visit(rendered);
  assert.equal(buttons.some(n=>n.props.children==="Devolver"),!completedAt);
 }
});
