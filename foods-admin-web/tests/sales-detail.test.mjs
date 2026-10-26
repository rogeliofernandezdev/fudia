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
const resolve=name=>name==="../domain/sales-periods"?compile("src/modules/sales/domain/sales-periods.ts",require):name==="react"?{...require("react"),useState:value=>[value,()=>{}]}:name==="@/design-system"?{DetailHero:"DetailHero",DetailContact:"DetailContact",DetailFacts:"DetailFacts",DetailTimeline:"DetailTimeline",Tabs:"Tabs",Skeleton:"Skeleton",ModalHeader:"ModalHeader",ModalFooter:"ModalFooter",ModalWorkspace:"ModalWorkspace",ModalItemList:"ModalItemList",Icon:"Icon",Button:"Button",Input:"Input",FormField:"FormField",Status:"Status",RowActionButton:"RowActionButton",SearchField:"SearchField",EmptyState:"EmptyState",ErrorState:"ErrorState",DataTable:"DataTable"}:name==="@/design-system/dialog"?{Dialog:"Dialog"}:name.startsWith("@/")||name.startsWith(".")?{}:require(name);
const {SaleDetailDialog}=compile("src/modules/sales/presentation/sale-detail-dialog.tsx",resolve);
const data={order:{id:"sale-1",code:"PED-TEST",channel:"salon",status:"entregado",customerName:"Familia Pérez",tableName:"Mesa 5",subtotal:"50",deliveryFee:"5",total:"55",createdAt:"2026-10-06T14:30:00+00",notes:"Sin sal",items:[{id:"item-1",name:"Menú criollo",qty:"2",unitPrice:"25",note:"Sin hielo",selections:[{groupName:"Segundo",name:"Lomo saltado"}],modifiers:[{groupName:"Extras",name:"Ensalada"}]}]},paidAmount:"55",remainingAmount:"0",paymentStatus:"paid",payments:[{id:"pay-1",method:"custom",methodName:"Convenio empresa",amount:"65",refundedAmount:"10",netAmount:"55",reference:"REF-1",cashRegisterName:"Caja principal",createdByName:"Ana",createdAt:"2026-10-06T15:00:00+00"}]};
const money=value=>`MON ${Number(value).toFixed(2)}`;
const date=value=>regional.formatRegionalDateTime(value,{country:"PE",timeZone:"America/Lima"});
function nodes(root){const result=[];function visit(node){if(!node||typeof node!=="object")return;if(Array.isArray(node)){node.forEach(visit);return}result.push(node);if(typeof node.type==="function")visit(node.type(node.props));else{visit(node.props?.children);visit(node.props?.summary);visit(node.props?.status);visit(node.props?.steps?.map?.(step=>step.aside))}}visit(root);return result}
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

test("una sola tabla permite abrir cada venta sin duplicar acciones",()=>{
 const {SalesResults}=compile("src/modules/sales/presentation/sales-page.tsx",resolve,"\nexport {SalesResults};\n");
 const selected=[];
 const rendered=nodes(SalesResults({items:[{...data.order,paidAmount:"55"}],money,date,view:id=>selected.push(id)}));
 const actions=rendered.filter(node=>node.type==="RowActionButton");
 assert.equal(actions.length,1);
 for(const action of actions){assert.equal(action.props.action,"view");assert.match(action.props.label,/Ver detalle de la venta/);action.props.onClick()}
 assert.deepEqual(selected,["sale-1"]);
 assert.ok(rendered.some(node=>node.type==="th"&&node.props.children==="Acciones"));
 assert.ok(rendered.some(node=>node.type==="DataTable"));
 assert.equal(rendered.some(node=>node.props.className==="management-cards sales-cards"),false);
});

test("Ventas muestra medios reales y pagos divididos una vez por fila",()=>{
 const {SalesResults,SalesTableSkeleton}=compile("src/modules/sales/presentation/sales-page.tsx",resolve,"\nexport {SalesResults,SalesTableSkeleton};\n");
 const sale={...data.order,paymentMethods:["Efectivo","Convenio empresa"]};
 const view=nodes(SalesResults({items:[sale],money,date,view:()=>{}}));
 const headers=view.filter(node=>node.type==="th").map(node=>node.props.children);
 assert.ok(headers.includes("Medio de pago"));assert.equal(headers.length,8);
 assert.equal(view.filter(node=>node.type==="td").length,8);
 assert.equal(view.filter(node=>node.props.children==="Efectivo · Convenio empresa").length,1);
 assert.equal(view.some(node=>node.type==="dt"&&node.props.children==="Medio de pago"),false);
 const single=nodes(SalesResults({items:[{...sale,paymentMethods:["Convenio empresa"]}],money,date,view:()=>{}}));
 assert.equal(single.filter(node=>node.props.children==="Convenio empresa").length,1);
 for(const paymentMethods of [[],undefined]){
  const empty=nodes(SalesResults({items:[{...sale,paymentMethods}],money,date,view:()=>{}}));
  assert.equal(empty.filter(node=>node.props.children==="—").length,1);
 }
 const skeleton=nodes(SalesTableSkeleton());
 assert.deepEqual(skeleton.filter(node=>node.type==="th").map(node=>node.props.children),headers);
 assert.equal(skeleton.filter(node=>node.type==="td").length,6*8);
 assert.equal(skeleton.filter(node=>node.props.className==="sales-card-methods").length,0);
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
 const {SalesFilters,SalesPeriods}=compile("src/modules/sales/presentation/sales-page.tsx",resolve,"\nexport {SalesFilters,SalesPeriods};\n");
 const changes=[];const callbacks={search:value=>changes.push(["q",value]),start:value=>changes.push(["from",value]),end:value=>changes.push(["to",value]),clear:()=>changes.push(["clear"])};
 const view=nodes(SalesFilters({q:"Mesa",period:"custom",from:"2026-10-01",to:"2026-10-07",invalidRange:false,...callbacks}));
 assert.deepEqual(Array.from(view.filter(node=>node.type==="FormField"),node=>node.props.label),["Buscar","Desde","Hasta"]);
 // Los periodos viven fuera de los filtros para quedar fijos en móvil.
 assert.equal(view.some(node=>node.type==="button"&&node.props.role==="radio"),false);
 const periods=nodes(SalesPeriods({period:"custom",choose:value=>changes.push(["period",value])})).filter(node=>node.type==="button"&&node.props.role==="radio");
 assert.deepEqual(Array.from(periods,node=>node.props.children),["Hoy","Esta semana","Este mes","Mes anterior","Personalizado"]);
 assert.equal(periods.find(node=>node.props["aria-checked"]).props.children,"Personalizado");
 const inputs=view.filter(node=>node.type==="Input");const searchField=view.find(node=>node.type==="SearchField");
 assert.equal(searchField.props.value,"Mesa");
 assert.equal(inputs[0].props.type,"date");assert.equal(inputs[0].props.max,"2026-10-07");assert.equal(inputs[1].props.min,"2026-10-01");
 searchField.props.onChange({target:{value:"Ana"}});inputs[0].props.onChange({target:{value:"2026-10-02"}});inputs[1].props.onChange({target:{value:"2026-10-06"}});
 view.find(node=>node.type==="Button").props.onClick();periods[1].props.onClick();
 assert.deepEqual(changes,[["q","Ana"],["from","2026-10-02"],["to","2026-10-06"],["clear"],["period","week"]]);
 const invalid=nodes(SalesFilters({q:"",period:"custom",from:"2026-10-07",to:"2026-10-01",invalidRange:true,...callbacks}));
 assert.match(invalid.find(node=>node.type==="FormField"&&node.props.label==="Hasta").props.error,/no puede ser anterior/);
 const empty=nodes(SalesFilters({q:"Mesa",period:"custom",from:"2026-10-01",to:"",invalidRange:false,...callbacks}));
 assert.equal(empty.find(node=>node.type==="FormField"&&node.props.label==="Hasta").props.help,"Vacío: hasta hoy");
 const preset=nodes(SalesFilters({q:"",period:"today",from:"2026-10-07",to:"2026-10-07",invalidRange:false,...callbacks}));
 assert.equal(preset.some(node=>node.type==="Button"),false,"«Limpiar fechas» solo en Personalizado");
 assert.deepEqual(Array.from(preset.filter(node=>node.type==="FormField"),node=>node.props.label),["Buscar"],"Desde y Hasta solo aparecen en Personalizado");
 assert.equal(preset.find(node=>node.props?.className?.startsWith?.("sales-filters")).props.className,"sales-filters");
 const css=read("src/modules/sales/presentation/sales.css");
 assert.equal(css.includes(".sales-search-control"),false,"el buscador usa SearchField homologado");
 assert.match(css,/@media\(max-width:600px\)\{[\s\S]*?\.sales-filters\{grid-template-columns:minmax\(0,1fr\)/);
});

test("cambiar fechas reinicia la página, limpiar conserva búsqueda y un rango invertido no consulta",async()=>{
 function render(from,to){
  const changes=[],queries=[],requests=[];let index=0;
  const state=["Mesa",3,10,null,from,to,null,"custom"];
  const {SalesPage}=compile("src/modules/sales/presentation/sales-page.tsx",name=>{
   if(name==="react")return {...require("react"),useState:()=>{const slot=index++;return [state[slot],value=>changes.push([slot,value])]} };
   if(name==="@tanstack/react-query")return {useQuery:options=>{queries.push(options);return {isLoading:false,isError:false,data:{items:[],total:0,amountTotal:"0"}}}};
   if(name==="@/providers")return {useSession:()=>({location:{country:"PE",timezone:"America/Lima"}}),useMoney:()=>money};
   if(name==="@/providers/settings-context")return {useSettings:()=>({})};
   if(name==="@/shared/hooks/use-debounced-value")return {useDebouncedValue:value=>value};
   if(name==="../infrastructure/sales-api")return {listSales:async input=>requests.push(input)};
   if(name==="@/shared/i18n/regional-format")return {formatRegionalDateTime:value=>value,formatRegionalCalendarDate:value=>value};
   return resolve(name);
  });
  const view=nodes(SalesPage());return {view,changes,queries,requests};
 }
 const valid=render("2026-10-01","2026-10-07");
 assert.equal(valid.queries[0].enabled,true);assert.deepEqual(Array.from(valid.queries[0].queryKey),["sales","Mesa","2026-10-01","2026-10-07",3,10]);
 await valid.queries[0].queryFn();assert.equal(valid.requests[0].from,"2026-10-01");assert.equal(valid.requests[0].to,"2026-10-07");
 valid.view.find(node=>node.type==="Input"&&node.props.type==="date").props.onChange({target:{value:"2026-10-02"}});
 assert.deepEqual(valid.changes,[[4,"2026-10-02"],[5,"2026-10-07"],[7,"custom"],[1,1]]);
 valid.changes.length=0;valid.view.find(node=>node.type==="Button").props.onClick();
 assert.deepEqual(valid.changes,[[4,""],[5,""],[7,"custom"],[1,1]]);
 const invalid=render("2026-10-07","2026-10-01");assert.equal(invalid.queries[0].enabled,false);assert.equal(invalid.requests.length,0);
 assert.equal(invalid.view.some(node=>node.type==="Pagination"),false);
 assert.ok(valid.view.some(node=>node.type==="EmptyState"&&node.props.title==="Sin coincidencias"));
});

test("detalle de venta muestra productos, composición, cuenta y pagos sin mutaciones",()=>{
 const view=mount();const content=JSON.stringify(view.map(node=>[node.props.children,node.props.steps,node.props.items,node.props.title,node.props.name,node.props.detail]));
 for(const text of ["Familia Pérez","Mesa 5","Menú criollo","Lomo saltado","Ensalada","Subtotal","Delivery","Total","Pagado neto","Convenio empresa","REF-1","Caja principal","Ana","Cobrado ","Devuelto "]){assert.ok(content.includes(text),text)}
 assert.ok(content.includes("MON 50.00"),"el pago muestra su neto");
 assert.ok(view.some(node=>node.type==="strong"&&node.props.children==="MON 55.00"));
 assert.equal(view.filter(node=>node.type==="ModalHeader").length,1);
 assert.equal(view.filter(node=>node.type==="Button").length,0);
 assert.equal(view.filter(node=>node.type==="ModalFooter").length,0,"sin acción real no se inventa un footer vacío");
 assert.equal(view.find(node=>node.type==="ModalWorkspace").props.summary.props.className,"modal-summary sales-detail-totals is-paid");
 assert.equal(view.find(node=>node.type==="Status").props.children,"Pagada");
 assert.equal(view.find(node=>node.type==="Status").props.emphasis,undefined,"el estado acompaña la identidad sin énfasis");
 assert.deepEqual(Array.from(view.find(node=>node.type==="Tabs").props.items,item=>item.label),["Productos","Pagos"],"sin pestaña de Detalles: dos datos no justifican una pestaña");
 assert.equal(view.find(node=>node.type==="Dialog").props["aria-labelledby"],"sales-detail-title");
});

test("detalle de venta mantiene cierre explícito, reintento, skeleton y vacíos",()=>{
 let closed=0,retried=0;
 const view=mount({error:"Sin conexión",close:()=>closed++,retry:()=>retried++});
 view.find(node=>node.type==="ModalHeader").props.onClose();view.find(node=>node.type==="ErrorState").props.onRetry();
 assert.equal(closed,1);assert.equal(retried,1);
 assert.equal(view.some(node=>node.type==="ModalWorkspace"),false,"el error no muestra resumen ni pestañas");
 const loading=mount({loading:true});assert.equal(loading.find(node=>node.type==="Dialog").props["aria-busy"],true);
 assert.ok(loading.some(node=>node.props["aria-label"]==="Cargando detalle de venta"));
 const empty=mount({data:{...data,order:{...data.order,items:[]},payments:[]}});
 assert.ok(empty.some(node=>node.props.children==="No hay productos registrados."));
 assert.ok(empty.some(node=>node.props.children==="No hay pagos registrados."));
 assert.ok(mount({data:{...data,paymentStatus:"partial",remainingAmount:"5"}}).some(node=>node.props.children==="Saldo pendiente"));
});

test("canal y mozo acompañan la fecha en la identidad, sin pestaña ni datos repetidos",()=>{
 const meta=view=>JSON.stringify(nodes(view.find(node=>node.props.className==="sales-detail-identity")).filter(node=>node.type==="span").map(node=>node.props.children));
 const view=mount({data:{...data,order:{...data.order,waiterName:"Daniel Machaca Rojasa"}}});
 assert.ok(meta(view).includes("Salón · Atendido por Daniel Machaca Rojasa"));
 assert.ok(meta(mount()).includes("Salón · Atendido por sin asignar"));
 const direct=mount({data:{...data,order:{...data.order,channel:"delivery"}}});
 assert.ok(meta(direct).includes("\"Delivery\""));assert.equal(meta(direct).includes("Atendido por"),false);
 assert.equal(view.some(node=>node.type==="DetailFacts"),false);
 assert.equal(view.some(node=>node.props["aria-label"]==="Detalles de la venta"),false);
 const code=JSON.stringify(view.map(node=>[node.props.children,node.props.items,node.props.title]));
 assert.equal(code.includes("PED-TEST"),false,"como en el pedido, el código no se repite en el detalle");
});

test("CSS de ventas tiene dueño único por contexto y el modal sigue la hoja móvil del pedido",()=>{
 const seen=new Set();const css=postcss.parse(read("src/modules/sales/presentation/sales.css"));
 css.walkRules(rule=>{for(const selector of rule.selectors){const key=(rule.parent.type==="atrule"?rule.parent.params:"root")+"|"+selector;assert.equal(seen.has(key),false,key);seen.add(key)}});
 const order=postcss.parse(read("src/modules/operations/orders/presentation/order-detail.css"));
 const decls=(sheet,selector,media)=>Object.fromEntries((media?sheet.nodes.find(node=>node.type==="atrule"&&node.params===media).nodes:sheet.nodes).find(node=>node.selector===selector).nodes.map(decl=>[decl.prop,decl.value]));
 assert.deepEqual(decls(css,".crud-modal.sales-detail-modal"),decls(order,".crud-modal.order-detail"),"mismo ancho, alto y columna que el pedido");
 assert.deepEqual(decls(css,".crud-modal.sales-detail-modal","(max-width:600px)"),decls(order,".crud-modal.order-detail","(max-width:600px)"),"misma hoja inferior en móvil");
 assert.deepEqual(decls(css,".modal-backdrop:has(.sales-detail-modal)","(max-width:600px)"),decls(order,".modal-backdrop:has(.order-detail)","(max-width:600px)"));
 assert.deepEqual(decls(css,".sales-detail-panel","(max-width:600px)"),{"padding-inline":"var(--space-3)"});
 assert.equal(decls(css,".sales-detail-panel")["padding-inline"],"var(--space-4)");
 assert.ok(mount().some(node=>node.props.className==="modal-body sales-detail-body"),"el cuerpo compartido es el único scroll, con barra siempre visible");
 for(const rule of css.nodes.filter(node=>node.type==="rule"&&node.selector.includes("sales-detail-lines"))){assert.equal(rule.nodes.some(decl=>["border","border-radius","background"].includes(decl.prop)),false,"listas sin tarjeta ni color local");}
});

test("venta usa listas y resumen compartidos, separa pagos y conserva la única acción del ticket",()=>{
 let opened=0;
 const view=mount({ticket:()=>opened++});
 assert.equal(view.filter(node=>node.type==="ModalItemList").length,1);
 assert.equal(view.filter(node=>node.props.role==="listitem").length,1);
 assert.equal(view.find(node=>node.type==="DetailTimeline").props.steps.length,1,"los pagos se presentan como progreso");
 assert.equal(view.filter(node=>node.props.className==="modal-item-quantity").length,1);
 const workspace=view.find(node=>node.type==="ModalWorkspace");
 const body=nodes(workspace.props.children);
 assert.equal(body.some(node=>node.props.className==="modal-summary"||node.props.className==="sales-detail-totals"),false);
 const summary=nodes(workspace.props.summary);
 assert.ok(summary.some(node=>node.props.className==="modal-summary-total"));
 assert.ok(summary.some(node=>node.type==="strong"&&node.props.children==="MON 55.00"));
 const button=view.find(node=>node.type==="Button");assert.equal(view.filter(node=>node.type==="Button").length,1);assert.equal(button.props.children,"Previsualizar");assert.equal(button.props.kind,"secondary");assert.equal(button.props.icon,"eye");button.props.onClick();assert.equal(opened,1);
 assert.equal(view.filter(node=>node.type==="Status").length,1,"el estado no se duplica en el footer de acciones");
 assert.ok(view.some(node=>node.props.className==="modal-detail-notes sales-detail-notes"));
 assert.doesNotMatch(read("src/modules/sales/presentation/sales.css"),/\.sales-detail-section\+\.sales-detail-section\{[^}]*border-top/,"la división no repite la línea de los productos");
 assert.match(read("src/modules/sales/presentation/sale-detail-dialog.tsx"),/<h3 id="sales-detail-payments" className="sr-only">/,"cada panel conserva su título accesible");
 for(const [paymentStatus,label] of [["paid","Pagada"],["partial","Pago parcial"],["pending","Sin pagos"]]){
  const state=mount({data:{...data,paymentStatus}}).find(node=>node.type==="Status");assert.equal(state.props.children,label);
 }
 const unpaid=mount({data:{...data,paidAmount:"0",paymentStatus:"pending",payments:[]},ticket:()=>opened++});
 assert.equal(unpaid.some(node=>node.type==="Button"),false);
 const loading=mount({loading:true});assert.ok(loading.some(node=>node.type==="ModalWorkspace"));assert.equal(loading.some(node=>node.type==="Button"||node.type==="Status"),false);
});

test("el resumen de pago permanece fuera del scroll y de las tres pestañas",()=>{
 let opened=0;
 for(const tab of ["products","payments"]){
  const {SaleDetailDialog:DialogForTab}=compile("src/modules/sales/presentation/sale-detail-dialog.tsx",name=>name==="react"?{...require("react"),useState:()=>[tab,()=>{}]}:resolve(name));
  const view=nodes(DialogForTab({loading:false,data,money,date,close:()=>{},retry:()=>{},ticket:()=>opened++}));
  const workspace=view.find(node=>node.type==="ModalWorkspace");
  assert.equal(nodes(workspace).some(node=>node.type==="ModalFooter"),false,"el botón no debe desplazarse con el contenido");
  const footer=view.find(node=>node.type==="ModalFooter");
  assert.ok(footer,`pie disponible en ${tab}`);
  const button=nodes(footer).find(node=>node.type==="Button");
  assert.equal(button.props.children,"Previsualizar");
  button.props.onClick();
 }
 assert.equal(opened,2);
});

test("detalle de venta identifica mesa, cliente o venta directa una sola vez",()=>{
 for(const [order,title] of [
  [{...data.order,tableName:"10",customerName:"Ana"},"Mesa 10"],
  [{...data.order,channel:"delivery",tableName:"",customerName:"Ana"},"Ana"],
  [{...data.order,channel:"pickup",tableName:"",customerName:""},"Venta directa"],
 ]){
  const view=mount({data:{...data,order}});
  assert.equal(view.find(node=>node.type==="ModalHeader").props.title,title);
  assert.equal(view.some(node=>node.type==="dt"||node.props?.children==="Cliente / mesa"),false);
 }
});

test("venta sigue el patrón del pedido: cabecera con mesa o cliente, identidad, tarjeta y pestañas",()=>{
 const view=mount({data:{...data,order:{...data.order,tableName:"03",waiterName:"Daniel Machaca Rojasa"}}});
 const header=view.find(node=>node.type==="ModalHeader");
 assert.equal(header.props.title,"Mesa 03");assert.equal(header.props.icon,"utensils");
 assert.equal(mount().find(node=>node.type==="ModalHeader").props.title,"Mesa 5","no duplica Mesa si ya forma parte del nombre");
 const delivery=mount({data:{...data,order:{...data.order,channel:"delivery",tableName:""}}}).find(node=>node.type==="ModalHeader");
 assert.equal(delivery.props.title,"Familia Pérez");assert.equal(delivery.props.icon,"truck");
 assert.equal(mount({data:{...data,order:{...data.order,channel:"mostrador",tableName:"",customerName:""}}}).find(node=>node.type==="ModalHeader").props.title,"Venta directa");
 const hero=view.find(node=>node.props.className==="sales-detail-hero");
 const identity=nodes(hero).find(node=>node.props.className==="sales-detail-identity");
 assert.ok(nodes(identity).some(node=>node.type==="Status"),"estado junto a la fecha, como el pedido");
 assert.ok(nodes(identity).some(node=>node.type==="Icon"&&node.props.name==="clock"));
 assert.ok(nodes(identity).some(node=>node.type==="time"&&node.props.children===date(data.order.createdAt)));
 assert.equal(nodes(hero).find(node=>node.type==="DetailContact").props.name,"Familia Pérez");
 assert.equal(mount({data:{...data,order:{...data.order,customerName:""}}}).some(node=>node.type==="DetailContact"),false,"sin cliente no hay tarjeta");
 const content=view.find(node=>node.props.className==="sales-detail-content");
 assert.ok(nodes(content).some(node=>node.type==="Tabs"&&node.props.className==="sales-detail-tabs"));
 assert.equal(nodes(content).filter(node=>node.props.role==="tabpanel").length,2);
 const products=nodes(content).find(node=>node.props.role==="tabpanel"&&node.props["aria-labelledby"]==="sales-detail-products");
 assert.ok(nodes(products).some(node=>node.props.className==="modal-detail-notes sales-detail-notes"),"las notas acompañan a los productos");
 const css=read("src/modules/sales/presentation/sales.css");
 assert.doesNotMatch(css,/\.ds-tabs\.sales-detail-tabs/,"pestañas compartidas, sin pista local");
 assert.doesNotMatch(css,/\.sales-detail-content\{[^}]*background/,"sin superficie gris envolvente");
 assert.doesNotMatch(css,/modal-item-quantity/,"cantidad con el estilo compartido de la línea");
 assert.doesNotMatch(css,/\.sales-detail-tabs\{/,"pestañas a 12 px del contenido: solo el gap compartido, sin margen extra");
 assert.match(css,/\.sales-detail-hero\{[^}]*padding-bottom:var\(--space-1\)/);
});

test("periodos rápidos de Ventas usan la fecha y zona horaria del local, con semana desde el lunes",()=>{
 const periods=compile("src/modules/sales/domain/sales-periods.ts",resolve);
 // 17 oct 2026 02:00 UTC: en Lima (UTC-5) todavía es viernes 16; en Madrid ya es sábado 17.
 const instant=new Date("2026-10-17T02:00:00Z");
 assert.equal(periods.localToday("America/Lima",instant),"2026-10-16");
 assert.equal(periods.localToday("Europe/Madrid",instant),"2026-10-17");
 const today="2026-10-16";
 const range=period=>JSON.parse(JSON.stringify(periods.periodRange(period,today)));
 assert.deepEqual(range("today"),{from:"2026-10-16",to:"2026-10-16"});
 assert.deepEqual(range("week"),{from:"2026-10-12",to:"2026-10-16"},"la semana empieza el lunes");
 assert.deepEqual(range("month"),{from:"2026-10-01",to:"2026-10-16"});
 assert.deepEqual(range("previousMonth"),{from:"2026-09-01",to:"2026-09-30"});
 assert.deepEqual(JSON.parse(JSON.stringify(periods.periodRange("previousMonth","2026-03-05"))),{from:"2026-02-01",to:"2026-02-28"});
 assert.deepEqual(JSON.parse(JSON.stringify(periods.periodRange("week","2026-10-18"))),{from:"2026-10-12",to:"2026-10-18"},"domingo cierra la semana");
 assert.deepEqual(JSON.parse(JSON.stringify(periods.periodRange("previousMonth","2026-01-10"))),{from:"2025-12-01",to:"2025-12-31"});
});

test("Ventas abre en Hoy con la fecha del local y muestra el total de todo el filtro",async()=>{
 const changes=[],queries=[],requests=[];let index=0;
 const state=["",1,10,null,"","",null,"today"];
 const {SalesPage}=compile("src/modules/sales/presentation/sales-page.tsx",name=>{
  if(name==="react")return {...require("react"),useState:()=>{const slot=index++;return [state[slot],value=>changes.push([slot,value])]} };
  if(name==="@tanstack/react-query")return {useQuery:options=>{queries.push(options);return {isLoading:false,isError:false,data:{items:[],total:4,amountTotal:"150.00"}}}};
  if(name==="@/providers")return {useSession:()=>({location:{country:"PE",timezone:"Pacific/Kiritimati"}}),useMoney:()=>money};
  if(name==="@/providers/settings-context")return {useSettings:()=>({})};
  if(name==="@/shared/hooks/use-debounced-value")return {useDebouncedValue:value=>value};
  if(name==="../infrastructure/sales-api")return {listSales:async input=>requests.push(input)};
  return resolve(name);
 });
 const view=nodes(SalesPage());
 const today=new Intl.DateTimeFormat("en-CA",{timeZone:"Pacific/Kiritimati",year:"numeric",month:"2-digit",day:"2-digit"}).format(new Date());
 assert.deepEqual(Array.from(queries[0].queryKey).slice(2,4),[today,today],"Hoy usa la fecha del local, no la del navegador");
 const summary=view.find(node=>node.props?.className==="sales-period-summary");
 assert.ok(summary,"franja de total del periodo");
 const text=JSON.stringify(summary.props.children);
 assert.ok(text.includes("Ventas de hoy"));assert.ok(text.includes("MON 150.00"));assert.ok(text.includes("4 ventas"));assert.ok(text.includes("MON 37.50"),"ticket promedio = total / ventas");
});
