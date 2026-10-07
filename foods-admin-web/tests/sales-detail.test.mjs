import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
import postcss from "postcss";
const require=createRequire(import.meta.url);
const root=new URL("../",import.meta.url);
const read=file=>readFileSync(new URL(file,root),"utf8");
function compile(file,resolve,extra=""){
 const exports={};
 vm.runInNewContext(ts.transpileModule(read(file)+extra,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve});
 return exports;
}
const regional=compile("src/shared/i18n/regional-format.ts",require);
const resolve=name=>name==="@/design-system"?{Icon:"Icon",Button:"Button",Status:"Status",RowActionButton:"RowActionButton"}:name==="@/design-system/dialog"?{Dialog:"Dialog"}:name.startsWith("@/")||name.startsWith(".")?{}:require(name);
const {SaleDetailDialog}=compile("src/modules/sales/presentation/sale-detail-dialog.tsx",resolve);
const data={order:{id:"sale-1",code:"PED-TEST",channel:"salon",status:"entregado",customerName:"Familia Pérez",tableName:"Mesa 5",subtotal:"50",deliveryFee:"5",total:"55",createdAt:"2026-10-06T14:30:00+00",notes:"Sin sal",items:[{id:"item-1",name:"Menú criollo",qty:"2",unitPrice:"25",note:"Sin hielo",selections:[{groupName:"Segundo",name:"Lomo saltado"}],modifiers:[{groupName:"Extras",name:"Ensalada"}]}]},paidAmount:"55",remainingAmount:"0",paymentStatus:"paid",payments:[{id:"pay-1",method:"custom",methodName:"Convenio empresa",amount:"65",refundedAmount:"10",netAmount:"55",reference:"REF-1",cashRegisterName:"Caja principal",createdByName:"Ana",createdAt:"2026-10-06T15:00:00+00"}]};
const money=value=>`MON ${Number(value).toFixed(2)}`;
const date=value=>regional.formatRegionalDateTime(value,{country:"PE",timeZone:"America/Lima"});
function nodes(root){const result=[];function visit(node){if(!node||typeof node!=="object")return;if(Array.isArray(node)){node.forEach(visit);return}result.push(node);if(typeof node.type==="function")visit(node.type(node.props));else visit(node.props?.children)}visit(root);return result}
function mount(overrides={}){return nodes(SaleDetailDialog({loading:false,data,money,date,close:()=>{},retry:()=>{},...overrides}))}

test("fechas PostgreSQL con offset de horas conservan el instante y la zona del local",()=>{
 for(const value of ["2026-10-06T14:30:00+00","2026-10-06T09:30:00-05","2026-10-06T14:30:00Z","2026-10-06T14:30:00+00:00"]){
  assert.equal(date(value),date("2026-10-06T14:30:00Z"));assert.notEqual(date(value),"—");
 }
 assert.equal(date("2026-10-06T14:30:00.123+00"),date("2026-10-06T14:30:00.123Z"));
 assert.equal(date("invalid"),"—");assert.equal(date(""),"—");
});

test("Ventas consulta el detalle real por id y conserva errores del API",async()=>{
 const calls=[];
 const api=compile("src/modules/sales/infrastructure/sales-api.ts",()=>({apiFetch:async path=>{calls.push(path);return data}}));
 assert.equal(await api.getSaleDetail("sale-1"),data);
 assert.deepEqual(calls,["pos/orders/sale-1"]);
 const failure=new Error("No existe en este local");
 const failing=compile("src/modules/sales/infrastructure/sales-api.ts",()=>({apiFetch:async()=>{throw failure}}));
 await assert.rejects(failing.getSaleDetail("other-id"),error=>error===failure);
});

test("tabla y tarjetas ofrecen el mismo ojo y abren la venta seleccionada",()=>{
 const {SalesResults}=compile("src/modules/sales/presentation/sales-page.tsx",resolve,"\nexport {SalesResults};\n");
 const selected=[];
 const rendered=nodes(SalesResults({items:[{...data.order,paidAmount:"55"}],money,date,view:id=>selected.push(id)}));
 const actions=rendered.filter(node=>node.type==="RowActionButton");
 assert.equal(actions.length,2);
 for(const action of actions){assert.equal(action.props.action,"view");assert.match(action.props.label,/Ver detalle de la venta/);action.props.onClick()}
 assert.deepEqual(selected,["sale-1","sale-1"]);
 assert.ok(rendered.some(node=>node.type==="th"&&node.props.children==="ACCIONES"));
 assert.ok(rendered.some(node=>node.props.className==="management-cards sales-cards"));
});

test("detalle de venta muestra productos, composición, cuenta y pagos sin mutaciones",()=>{
 const view=mount();const content=JSON.stringify(view.map(node=>node.props.children));
 for(const text of ["Familia Pérez","Mesa 5","Menú criollo","Lomo saltado","Ensalada","Subtotal","Delivery","Total","Pagado neto","Convenio empresa","REF-1","Caja principal","Ana","Cobrado ","Devuelto "]){assert.ok(content.includes(text),text)}
 assert.ok(view.some(node=>node.type==="strong"&&node.props.children==="MON 50.00"));
 assert.ok(view.some(node=>node.type==="strong"&&node.props.children==="MON 55.00"));
 assert.equal(view.filter(node=>node.type==="button").length,1);
 assert.equal(view.filter(node=>node.type==="Button").length,0);
 assert.equal(view.some(node=>node.type==="footer"),false);
 assert.equal(view.find(node=>node.type==="Dialog").props["aria-labelledby"],"sales-detail-title");
});

test("detalle de venta mantiene cierre explícito, reintento, skeleton y vacíos",()=>{
 let closed=0,retried=0;
 const view=mount({error:"Sin conexión",close:()=>closed++,retry:()=>retried++});
 view.find(node=>node.type==="button").props.onClick();view.find(node=>node.type==="Button").props.onClick();
 assert.equal(closed,1);assert.equal(retried,1);
 assert.equal(view.some(node=>node.props.className==="sales-detail-body hover-scroll"),false);
 const loading=mount({loading:true});assert.equal(loading.find(node=>node.type==="Dialog").props["aria-busy"],true);
 assert.ok(loading.some(node=>node.props["aria-label"]==="Cargando detalle de venta"));
 const empty=mount({data:{...data,order:{...data.order,items:[]},payments:[]}});
 assert.ok(empty.some(node=>node.props.children==="No hay productos registrados."));
 assert.ok(empty.some(node=>node.props.children==="No hay pagos registrados."));
 assert.ok(mount({data:{...data,paymentStatus:"partial",remainingAmount:"5"}}).some(node=>node.props.children==="Saldo pendiente"));
});

test("CSS de ventas tiene dueño único por contexto y reorganiza el detalle móvil",()=>{
 const seen=new Set();const css=postcss.parse(read("src/modules/sales/presentation/sales.css"));
 css.walkRules(rule=>{for(const selector of rule.selectors){const key=(rule.parent.type==="atrule"?rule.parent.params:"root")+"|"+selector;assert.equal(seen.has(key),false,key);seen.add(key)}});
 const mobile=css.nodes.find(node=>node.type==="atrule"&&node.params==="(max-width:600px)");
 assert.ok(mobile.nodes.some(node=>node.selector===".sales-detail-lines>article,.sales-detail-payments>article"));
 assert.match(read("src/modules/sales/presentation/sales.css"),/\.sales-detail-body\{[^}]*overflow-y:auto/);
});
