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
 vm.runInNewContext(ts.transpileModule(read(file)+extra,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve,URLSearchParams});
 return exports;
}
const regional=compile("src/shared/i18n/regional-format.ts",require);
const resolve=name=>name==="@/design-system"?{ModalHeader:"ModalHeader",ModalFooter:"ModalFooter",ModalWorkspace:"ModalWorkspace",ModalItemList:"ModalItemList",Icon:"Icon",Button:"Button",Input:"Input",FormField:"FormField",Status:"Status",RowActionButton:"RowActionButton",SearchField:"SearchField",EmptyState:"EmptyState",ErrorState:"ErrorState"}:name==="@/design-system/dialog"?{Dialog:"Dialog"}:name.startsWith("@/")||name.startsWith(".")?{}:require(name);
const {SaleDetailDialog}=compile("src/modules/sales/presentation/sale-detail-dialog.tsx",resolve);
const data={order:{id:"sale-1",code:"PED-TEST",channel:"salon",status:"entregado",customerName:"Familia Pérez",tableName:"Mesa 5",subtotal:"50",deliveryFee:"5",total:"55",createdAt:"2026-10-06T14:30:00+00",notes:"Sin sal",items:[{id:"item-1",name:"Menú criollo",qty:"2",unitPrice:"25",note:"Sin hielo",selections:[{groupName:"Segundo",name:"Lomo saltado"}],modifiers:[{groupName:"Extras",name:"Ensalada"}]}]},paidAmount:"55",remainingAmount:"0",paymentStatus:"paid",payments:[{id:"pay-1",method:"custom",methodName:"Convenio empresa",amount:"65",refundedAmount:"10",netAmount:"55",reference:"REF-1",cashRegisterName:"Caja principal",createdByName:"Ana",createdAt:"2026-10-06T15:00:00+00"}]};
const money=value=>`MON ${Number(value).toFixed(2)}`;
const date=value=>regional.formatRegionalDateTime(value,{country:"PE",timeZone:"America/Lima"});
function nodes(root){const result=[];function visit(node){if(!node||typeof node!=="object")return;if(Array.isArray(node)){node.forEach(visit);return}result.push(node);if(typeof node.type==="function")visit(node.type(node.props));else{visit(node.props?.children);visit(node.props?.summary)}}visit(root);return result}
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

test("Ventas muestra medios reales y pagos divididos en tabla y móvil",()=>{
 const {SalesResults,SalesTableSkeleton}=compile("src/modules/sales/presentation/sales-page.tsx",resolve,"\nexport {SalesResults,SalesTableSkeleton};\n");
 const sale={...data.order,paymentMethods:["Efectivo","Convenio empresa"]};
 const view=nodes(SalesResults({items:[sale],money,date,view:()=>{}}));
 const headers=view.filter(node=>node.type==="th").map(node=>node.props.children);
 assert.ok(headers.includes("MEDIO DE PAGO"));assert.equal(headers.length,8);
 assert.equal(view.filter(node=>node.type==="td").length,8);
 assert.equal(view.filter(node=>node.props.children==="Efectivo · Convenio empresa").length,2);
 assert.ok(view.some(node=>node.type==="dt"&&node.props.children==="Medio de pago"));
 const single=nodes(SalesResults({items:[{...sale,paymentMethods:["Convenio empresa"]}],money,date,view:()=>{}}));
 assert.equal(single.filter(node=>node.props.children==="Convenio empresa").length,2);
 for(const paymentMethods of [[],undefined]){
  const empty=nodes(SalesResults({items:[{...sale,paymentMethods}],money,date,view:()=>{}}));
  assert.equal(empty.filter(node=>node.props.children==="—").length,2);
 }
 const skeleton=nodes(SalesTableSkeleton());
 assert.deepEqual(skeleton.filter(node=>node.type==="th").map(node=>node.props.children),headers);
 assert.equal(skeleton.filter(node=>node.type==="td").length,6*8);
 assert.equal(skeleton.filter(node=>node.props.className==="sales-card-methods").length,3);
});

test("el listado paginado recibe los medios sin consultas adicionales por venta",async()=>{
 const response={items:[{...data.order,paymentMethods:["Convenio empresa"]}],total:1,page:2,pageSize:10};
 const calls=[];
 const api=compile("src/modules/sales/infrastructure/sales-api.ts",()=>({apiFetch:async path=>{calls.push(path);return response}}));
 assert.equal(await api.listSales({q:"Mesa",page:2,pageSize:10}),response);
 assert.equal(calls.length,1);
 const params=new URLSearchParams(calls[0].split("?")[1]);
 assert.equal(params.get("paymentStatus"),"paid");assert.equal(params.get("page"),"2");
 assert.equal(params.get("pageSize"),"10");assert.equal(params.get("q"),"Mesa");
});

test("Ventas envía fechas independientes sin perder búsqueda ni paginación",async()=>{
 const calls=[];
 const api=compile("src/modules/sales/infrastructure/sales-api.ts",()=>({apiFetch:async path=>{calls.push(path);return {items:[]}}}));
 for(const range of [{from:"2026-10-01",to:"2026-10-07"},{from:"2026-10-01"},{to:"2026-10-07"},{}]){
  await api.listSales({q:"Mesa 5",page:2,pageSize:20,...range});
  const params=new URLSearchParams(calls.at(-1).split("?")[1]);
  assert.equal(params.get("from"),range.from??null);assert.equal(params.get("to"),range.to??null);
  assert.equal(params.get("q"),"Mesa 5");assert.equal(params.get("page"),"2");assert.equal(params.get("pageSize"),"20");
 }
});

test("filtros de Ventas usan controles homologados y límites visibles",()=>{
 const {SalesFilters}=compile("src/modules/sales/presentation/sales-page.tsx",resolve,"\nexport {SalesFilters};\n");
 const changes=[];const callbacks={search:value=>changes.push(["q",value]),start:value=>changes.push(["from",value]),end:value=>changes.push(["to",value]),clear:()=>changes.push(["clear"])};
 const view=nodes(SalesFilters({q:"Mesa",from:"2026-10-01",to:"2026-10-07",invalidRange:false,...callbacks}));
 assert.deepEqual(view.filter(node=>node.type==="FormField").map(node=>node.props.label),["Buscar","Fecha de inicio","Fecha de fin"]);
 const inputs=view.filter(node=>node.type==="Input");const searchField=view.find(node=>node.type==="SearchField");
 assert.equal(searchField.props.value,"Mesa");
 assert.equal(inputs[0].props.type,"date");assert.equal(inputs[0].props.max,"2026-10-07");assert.equal(inputs[1].props.min,"2026-10-01");
 searchField.props.onChange({target:{value:"Ana"}});inputs[0].props.onChange({target:{value:"2026-10-02"}});inputs[1].props.onChange({target:{value:"2026-10-06"}});
 view.find(node=>node.type==="Button").props.onClick();
 assert.deepEqual(changes,[["q","Ana"],["from","2026-10-02"],["to","2026-10-06"],["clear"]]);
 const invalid=nodes(SalesFilters({q:"",from:"2026-10-07",to:"2026-10-01",invalidRange:true,...callbacks}));
 assert.match(invalid.find(node=>node.type==="FormField"&&node.props.label==="Fecha de fin").props.error,/no puede ser anterior/);
 const empty=nodes(SalesFilters({q:"Mesa",from:"",to:"",invalidRange:false,...callbacks}));
 assert.equal(empty.some(node=>node.type==="Button"),false);
 const css=read("src/modules/sales/presentation/sales.css");
 assert.equal(css.includes(".sales-search-control"),false,"el buscador usa SearchField homologado");
 assert.match(css,/@media\(max-width:600px\)\{\s*\.sales-filters\{grid-template-columns:minmax\(0,1fr\)/);
});

test("cambiar fechas reinicia la página, limpiar conserva búsqueda y un rango invertido no consulta",async()=>{
 function render(from,to){
  const changes=[],queries=[],requests=[];let index=0;
  const state=["Mesa",3,10,null,from,to];
  const {SalesPage}=compile("src/modules/sales/presentation/sales-page.tsx",name=>{
   if(name==="react")return {...require("react"),useState:()=>{const slot=index++;return [state[slot],value=>changes.push([slot,value])]} };
   if(name==="@tanstack/react-query")return {useQuery:options=>{queries.push(options);return {isLoading:false,isError:false,data:{items:[],total:0}}}};
   if(name==="@/providers")return {useSession:()=>({location:{country:"PE",timezone:"America/Lima"}}),useMoney:()=>money};
   if(name==="@/providers/settings-context")return {useSettings:()=>({})};
   if(name==="@/shared/hooks/use-debounced-value")return {useDebouncedValue:value=>value};
   if(name==="../infrastructure/sales-api")return {listSales:async input=>requests.push(input)};
   return resolve(name);
  });
  const view=nodes(SalesPage());return {view,changes,queries,requests};
 }
 const valid=render("2026-10-01","2026-10-07");
 assert.equal(valid.queries[0].enabled,true);assert.deepEqual(Array.from(valid.queries[0].queryKey),["sales","Mesa","2026-10-01","2026-10-07",3,10]);
 await valid.queries[0].queryFn();assert.equal(valid.requests[0].from,"2026-10-01");assert.equal(valid.requests[0].to,"2026-10-07");
 valid.view.find(node=>node.type==="Input"&&node.props.type==="date").props.onChange({target:{value:"2026-10-02"}});
 assert.deepEqual(valid.changes,[[4,"2026-10-02"],[1,1]]);
 valid.changes.length=0;valid.view.find(node=>node.type==="Button").props.onClick();
 assert.deepEqual(valid.changes,[[4,""],[5,""],[1,1]]);
 const invalid=render("2026-10-07","2026-10-01");assert.equal(invalid.queries[0].enabled,false);assert.equal(invalid.requests.length,0);
 assert.equal(invalid.view.some(node=>node.type==="Pagination"),false);
 assert.ok(valid.view.some(node=>node.type==="EmptyState"&&node.props.title==="Sin coincidencias"));
});

test("detalle de venta muestra productos, composición, cuenta y pagos sin mutaciones",()=>{
 const view=mount();const content=JSON.stringify(view.map(node=>node.props.children));
 for(const text of ["Familia Pérez","Mesa 5","Menú criollo","Lomo saltado","Ensalada","Subtotal","Delivery","Total","Pagado neto","Convenio empresa","REF-1","Caja principal","Ana","Cobrado ","Devuelto "]){assert.ok(content.includes(text),text)}
 assert.ok(view.some(node=>node.type==="strong"&&node.props.children==="MON 50.00"));
 assert.ok(view.some(node=>node.type==="strong"&&node.props.children==="MON 55.00"));
 assert.equal(view.filter(node=>node.type==="ModalHeader").length,1);
 assert.equal(view.filter(node=>node.type==="Button").length,0);
 assert.equal(view.filter(node=>node.type==="ModalFooter").length,0,"sin acción real no se inventa un footer vacío");
 assert.equal(view.find(node=>node.type==="ModalWorkspace").props.summary.props.className,"modal-summary");
 assert.equal(view.find(node=>node.type==="Status").props.children,"Pagada");
 assert.equal(view.find(node=>node.type==="Status").props.emphasis,true);
 assert.equal(view.find(node=>node.type==="Dialog").props["aria-labelledby"],"sales-detail-title");
});

test("detalle de venta mantiene cierre explícito, reintento, skeleton y vacíos",()=>{
 let closed=0,retried=0;
 const view=mount({error:"Sin conexión",close:()=>closed++,retry:()=>retried++});
 view.find(node=>node.type==="ModalHeader").props.onClose();view.find(node=>node.type==="ErrorState").props.onRetry();
 assert.equal(closed,1);assert.equal(retried,1);
 assert.equal(view.some(node=>node.props.className==="modal-body sales-detail-body hover-scroll"),false);
 const loading=mount({loading:true});assert.equal(loading.find(node=>node.type==="Dialog").props["aria-busy"],true);
 assert.ok(loading.some(node=>node.props["aria-label"]==="Cargando detalle de venta"));
 const empty=mount({data:{...data,order:{...data.order,items:[]},payments:[]}});
 assert.ok(empty.some(node=>node.props.children==="No hay productos registrados."));
 assert.ok(empty.some(node=>node.props.children==="No hay pagos registrados."));
 assert.ok(mount({data:{...data,paymentStatus:"partial",remainingAmount:"5"}}).some(node=>node.props.children==="Saldo pendiente"));
});

test("datos de la venta alinean mesa/canal y mozo/fecha sin una tercera fila forzada",()=>{
 const view=mount({data:{...data,order:{...data.order,waiterName:"Daniel Machaca Rojasa"}}});
 const context=view.find(node=>node.props.className==="sales-detail-context");
 assert.deepEqual(nodes(context).filter(node=>node.type==="dt").map(node=>node.props.children),["Mesa","Canal","Atendido por","Fecha del pedido"]);
 assert.ok(nodes(context).some(node=>node.type==="dd"&&node.props.children==="Daniel Machaca Rojasa"));
 const direct=mount({data:{...data,order:{...data.order,channel:"delivery"}}});
 assert.equal(direct.some(node=>node.type==="dt"&&node.props.children==="Atendido por"),false);
 const loading=mount({loading:true}).find(node=>node.props.className==="sales-detail-context");
 assert.equal(nodes(loading).filter(node=>node.type==="i").length,4);
 const css=postcss.parse(read("src/modules/sales/presentation/sales.css"));
 const desktop=css.nodes.find(node=>node.type==="rule"&&node.selector===".sales-detail-context dl");
 assert.ok(desktop.nodes.some(node=>node.prop==="grid-template-columns"&&node.value==="repeat(2,minmax(0,1fr))"));
 assert.equal(css.nodes.some(node=>node.selector===".sales-detail-context dl>div:last-child"),false);
 const mobile=css.nodes.find(node=>node.type==="atrule"&&node.params==="(max-width:620px)");
 const mobileGrid=mobile.nodes.find(node=>node.selector===".sales-detail-context dl");
 assert.equal(mobileGrid.nodes.some(node=>node.prop==="grid-template-columns"),false,"mesa y canal conservan dos columnas");
 assert.ok(mobile.nodes.some(node=>node.selector===".sales-detail-context dl>div:nth-child(2)"&&node.nodes.some(decl=>decl.prop==="text-align"&&decl.value==="right")));
 assert.ok(mobile.nodes.some(node=>node.selector===".sales-detail-context dl>div:nth-child(n+3)"&&node.nodes.some(decl=>decl.prop==="grid-column"&&decl.value==="1/-1")));
});

test("CSS de ventas tiene dueño único por contexto y reorganiza el detalle móvil",()=>{
 const seen=new Set();const css=postcss.parse(read("src/modules/sales/presentation/sales.css"));
 css.walkRules(rule=>{for(const selector of rule.selectors){const key=(rule.parent.type==="atrule"?rule.parent.params:"root")+"|"+selector;assert.equal(seen.has(key),false,key);seen.add(key)}});
 const mobile=css.nodes.find(node=>node.type==="atrule"&&node.params==="(max-width:600px)");
 assert.ok(mobile.nodes.some(node=>node.selector===".crud-modal.sales-detail-modal"&&node.nodes.some(decl=>decl.prop==="height"&&decl.value==="100dvh")));
 assert.ok(mount().some(node=>node.props.className==="modal-body sales-detail-body hover-scroll"),"el cuerpo compartido es el único scroll");
 for(const rule of css.nodes.filter(node=>node.type==="rule"&&[".sales-detail-lines",".sales-detail-payments",".sales-detail-lines,.sales-detail-payments"].includes(node.selector))){assert.equal(rule.nodes.some(decl=>["border","border-radius"].includes(decl.prop)),false,"listas sin tarjeta local");}
});

test("venta usa listas y resumen compartidos, separa pagos y conserva la única acción del ticket",()=>{
 let opened=0;
 const view=mount({ticket:()=>opened++});
 assert.equal(view.filter(node=>node.type==="ModalItemList").length,2);
 assert.equal(view.filter(node=>node.props.role==="listitem").length,2);
 assert.equal(view.filter(node=>node.props.className==="modal-item-quantity").length,1);
 const workspace=view.find(node=>node.type==="ModalWorkspace");
 const body=nodes(workspace.props.children);
 assert.equal(body.some(node=>node.props.className==="modal-summary"||node.props.className==="sales-detail-totals"),false);
 const summary=nodes(workspace.props.summary);
 assert.ok(summary.some(node=>node.props.className==="modal-summary-total"));
 assert.ok(summary.some(node=>node.type==="strong"&&node.props.children==="MON 55.00"));
 const button=view.find(node=>node.type==="Button");assert.equal(view.filter(node=>node.type==="Button").length,1);button.props.onClick();assert.equal(opened,1);
 assert.equal(view.filter(node=>node.type==="Status").length,1,"el estado no se duplica en el footer de acciones");
 assert.ok(view.some(node=>node.props.className==="modal-detail-notes sales-detail-notes"));
 assert.match(read("src/modules/sales/presentation/sales.css"),/\.sales-detail-section\+\.sales-detail-section\{[^}]*border-top:var\(--stroke-1\) solid var\(--line\)/);
 for(const [paymentStatus,label] of [["paid","Pagada"],["partial","Pago parcial"],["pending","Sin pagos"]]){
  const state=mount({data:{...data,paymentStatus}}).find(node=>node.type==="Status");assert.equal(state.props.children,label);
 }
 const unpaid=mount({data:{...data,paidAmount:"0",paymentStatus:"pending",payments:[]},ticket:()=>opened++});
 assert.equal(unpaid.some(node=>node.type==="Button"),false);
 const loading=mount({loading:true});assert.ok(loading.some(node=>node.type==="ModalWorkspace"));assert.equal(loading.some(node=>node.type==="Button"||node.type==="Status"),false);
});

test("detalle de venta identifica mesa, cliente o venta directa sin etiqueta combinada",()=>{
 for(const [order,label] of [
  [{...data.order,tableName:"10",customerName:"Ana"},"Mesa"],
  [{...data.order,channel:"delivery",tableName:"",customerName:"Ana"},"Cliente"],
  [{...data.order,channel:"pickup",tableName:"",customerName:""},"Venta"],
 ]){
  const view=mount({data:{...data,order}});
  assert.equal(view.find(node=>node.type==="dt").props.children,label);
  assert.equal(view.some(node=>node.props?.children==="Cliente / mesa"),false);
 }
});
