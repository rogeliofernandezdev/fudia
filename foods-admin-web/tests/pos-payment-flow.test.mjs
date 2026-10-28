import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url);
const root=new URL("../src/modules/operations/pos/presentation/",import.meta.url);
const serviceFlowExports={};
vm.runInNewContext(ts.transpileModule(readFileSync(new URL("../src/modules/operations/orders/domain/service-flow.ts",import.meta.url),"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,{exports:serviceFlowExports});
const detail={order:{billClosedAt:"2026-10-07T13:00:00Z",id:"order-1",code:"PED-001",channel:"salon",status:"entregado",tableName:"Mesa 04",customerName:"",total:"40",createdAt:"2026-10-07T12:00:00Z",items:[]},paidAmount:"10",remainingAmount:"30",paymentStatus:"partial",payments:[]};
const primitives=Object.fromEntries(["Button","EmptyState","ErrorState","FormField","Icon","Input","PageHeader","Pagination","RowActionButton","SearchField","Select","Status","Textarea","ModalHeader","ModalFooter","ModalWorkspace","ModalItemList","DetailHero","DetailContact","DetailFacts","DetailTimeline","Tabs","Skeleton"].map(name=>[name,name]));
function nodes(tree){const result=[];function visit(node){if(!node||typeof node!=="object")return;if(Array.isArray(node)){node.forEach(visit);return}result.push(node);visit(node.props?.children);visit(node.props?.action);visit(node.props?.summary);visit(node.props?.status);visit(node.props?.steps?.map?.(step=>step.aside))}visit(tree);return result}
function compile(file,resolve,extra=""){
 const exports={};
 const source=readFileSync(new URL(file,root),"utf8")+extra;
 vm.runInNewContext(ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve});
 return exports;
}
function mountPage({canManage=true,shift=true,busy=false,initialOrderId="order-1",listed=[]}={}){
 const queries=[],mutations=[],changes=[],sent=[],invalidations=[],notices=[];let stateIndex=0,dynamicIndex=0;
 const resolve=name=>{
  if(name==="next/dynamic")return{default:()=>["PaymentDialog","RefundDialog","PaymentTicketDialog"][dynamicIndex++]};
  if(name==="next/link")return{default:"Link"};
  if(name==="react")return{...require(name),useState:value=>{const index=stateIndex++;return[value,next=>changes.push({index,next})]}};
  if(name==="@tanstack/react-query")return{
   useQueryClient:()=>({invalidateQueries:options=>invalidations.push(options.queryKey)}),
   useQuery:options=>{queries.push(options);return{isLoading:false,isPending:false,isFetching:false,isError:false,refetch:()=>{},data:options.queryKey[0]==="pos-cash-access"?{shift:shift?{cashRegisterName:"Caja principal",code:"CAJ-001",openedByName:"Ana"}:null}:options.queryKey[0]==="pos-orders"?{items:listed,total:listed.length}:options.queryKey[0]==="payment-methods"?{items:catalog}:options.enabled?detail:undefined}},
   useMutation:options=>{mutations.push(options);return{isPending:busy,mutate:input=>sent.push(input)}},
  };
  if(name==="@/design-system")return primitives;
  if(name==="@/design-system/dialog")return{Dialog:"Dialog"};
  if(name==="@/providers")return{useFeedback:()=>({notify:value=>notices.push(value)}),useMoney:()=>value=>`S/ ${value}`,useSession:()=>({can:()=>canManage,location:{country:"PE",timezone:"America/Lima"}})};
  if(name==="@/providers/settings-context")return{useSettings:()=>({currencySymbol:"S/",currencyDecimals:2,currencyPosition:"before"})};
  if(name==="@/shared/i18n/regional-format")return{formatRegionalNumber:value=>String(value),formatRegionalDateTime:()=>"Hoy"};
  if(name==="@/shared/hooks/use-debounced-value")return{useDebouncedValue:value=>value};
  if(name==="@/shared/routing/page-routes")return{pageRoutes:{cash:"/cash"}};
  if(name.endsWith("service-flow"))return serviceFlowExports;
  if(name.endsWith("shift-attribution"))return{cashShiftAttribution:()=>({name:"Ana"})};
  if(name.startsWith("@/")||name.startsWith("../")||name.startsWith("./"))return{};
  return require(name);
 };
 const compiled=compile("pos-page.tsx",resolve,"\nexport {POSPaymentEntry,POSPaymentLoading,POSDetailDialog};\n");
 const tree=compiled.POSPage({initialOrderId});
 const entry=nodes(tree).find(node=>node.type===compiled.POSPaymentEntry);
 return{...compiled,entry,queries,mutations,changes,sent,invalidations,notices,tree};
}
const catalog=[{code:"cash",name:"Efectivo",affectsCash:true},{code:"card",name:"Tarjeta",affectsCash:false},{code:"yape",name:"Yape",affectsCash:false},{code:"plin",name:"Plin",affectsCash:false}];
test("un colaborador asignado cobra sin cash.manage; al perder el turno se bloquea",()=>{
 const assigned=mountPage({canManage:false,shift:true});
 assert.equal(assigned.entry.props.canCollect,true);
 assert.equal(assigned.POSPaymentEntry(assigned.entry.props).type,"PaymentDialog");
 assigned.entry.props.save({method:"cash",amount:"30",reference:""});
 assert.equal(assigned.sent.length,1);
 const revoked=mountPage({canManage:false,shift:false});
 assert.equal(revoked.entry.props.canCollect,false);
 revoked.entry.props.save({method:"cash",amount:"30",reference:""});
 assert.equal(revoked.sent.length,0);
});
function entryProps(changes={}){return{loading:false,data:detail,shiftLoading:false,shiftName:"Caja principal",methodsLoading:false,methods:catalog,retryMethods:()=>{},canCollect:true,busy:false,formatMoney:value=>`S/ ${value}`,close:()=>{},retryOrder:()=>{},retryShift:()=>{},save:()=>{},...changes}}

test("Cobrar saldo abre el formulario real, no el detalle, sin registrar pagos al abrir",()=>{
 const view=mountPage();
 assert.ok(view.entry);
 assert.equal(nodes(view.tree).some(node=>node.type===view.POSDetailDialog),false);
 const query=view.queries.find(options=>options.queryKey[1]==="order-1");
 assert.equal(query.enabled,true);
 assert.equal(query.staleTime,0);
 assert.equal(query.refetchOnMount,"always");
 assert.equal(query.refetchOnWindowFocus,false);
 assert.equal(view.sent.length,0);
 const form=view.POSPaymentEntry(view.entry.props);
 assert.equal(form.type,"PaymentDialog");
 assert.equal(form.props.order.id,"order-1");
 assert.equal(form.props.order.paidAmount,"10");
 assert.equal(form.props.order.remainingAmount,"30");
 const draft={method:"cash",amount:"30",reference:""};
 form.props.save(draft);
 assert.equal(view.sent.length,1);
 assert.equal(view.sent[0].orderId,"order-1");
 assert.equal(view.sent[0].draft,draft);
 view.mutations[0].onSuccess({method:"cash",id:"payment-1"},{orderId:"order-1"});
 assert.ok(view.changes.some(change=>change.next===null));
 for(const key of ["pos-orders","pos-order","orders","salon-floor","cash-shift","sales","dashboard"])assert.ok(view.invalidations.some(queryKey=>queryKey[0]===key));
});

test("el ticket abre después del aviso y de la respuesta de pago, nunca en error ni al enviar",()=>{
 const view=mountPage();assert.equal(view.notices.length,0);assert.equal(view.mutations[0].retry,false);
 view.mutations[0].onSuccess({id:"payment-1"},{orderId:"order-1"});
 assert.equal(view.changes.some(change=>change.next?.paymentId),false);
 view.notices[0].onClose();assert.ok(view.changes.some(change=>change.next?.paymentId==="payment-1"&&change.next.orderId==="order-1"));
 const failed=mountPage();failed.mutations[0].onError(new Error("Rechazado"));assert.equal(failed.notices[0].onClose,undefined);assert.equal(failed.changes.length,0);
});

test("Cobrar y Ver detalle de la tabla abren destinos independientes",()=>{
 const view=mountPage({initialOrderId:"",listed:[{...detail.order,paidAmount:detail.paidAmount,remainingAmount:detail.remainingAmount,paymentStatus:detail.paymentStatus}]});
 assert.equal(view.entry,undefined);
 const actions=nodes(view.tree).filter(node=>node.type==="RowActionButton");
 actions.find(node=>node.props.action==="charge").props.onClick();
 actions.find(node=>node.props.action==="view").props.onClick();
 assert.deepEqual(view.changes,[{index:4,next:"order-1"},{index:5,next:"order-1"}]);
 assert.equal(view.sent.length,0);
});

test("el cobro espera el pedido y el turno con skeleton, sin formulario con saldo cacheado",()=>{
 const view=mountPage();
 for(const changes of [{loading:true},{shiftLoading:true}]){
  const tree=view.POSPaymentEntry(entryProps(changes));
  assert.equal(nodes(tree).some(node=>node.type==="PaymentDialog"),false);
  assert.ok(nodes(tree).some(node=>node.type===view.POSPaymentLoading));
 }
 const skeleton=view.POSPaymentLoading();
 assert.equal(skeleton.props["aria-busy"],"true");
 assert.equal(skeleton.props["aria-label"],"Cargando formulario de cobro");
});

test("errores de pedido o caja tienen reintento y la caja ausente ofrece ir a Caja",()=>{
 const view=mountPage();let retries=0;
 for(const changes of [{error:"Pedido inaccesible",retryOrder:()=>retries++},{shiftError:"Caja inaccesible",retryShift:()=>retries++}]){
  const tree=view.POSPaymentEntry(entryProps(changes));
  const retry=nodes(tree).find(node=>node.type==="ErrorState"&&node.props.onRetry);
  assert.ok(retry);retry.props.onRetry();
  assert.equal(nodes(tree).some(node=>node.type==="PaymentDialog"),false);
 }
 assert.equal(retries,2);
 const missing=nodes(view.POSPaymentEntry(entryProps({shiftName:null})));
 assert.ok(missing.some(node=>node.type==="Link"&&node.props.href==="/cash"));
 assert.equal(missing.some(node=>node.type==="PaymentDialog"),false);
});

test("el formulario no permite cobrar sin permiso, pedidos pagados, cancelados o con cuenta abierta",()=>{
 const view=mountPage();
 for(const changes of [{canCollect:false},{data:{...detail,paymentStatus:"paid",remainingAmount:"0"}},{data:{...detail,order:{...detail.order,completedAt:"2026-10-07T14:00:00Z"}}},{data:{...detail,order:{...detail.order,status:"cancelado"}}},{data:{...detail,order:{...detail.order,billClosedAt:undefined,status:"preparando"}}},{data:{...detail,order:{...detail.order,channel:"delivery",status:"preparando"}}},...["confirmado","preparando","listo"].map(status=>({data:{...detail,order:{...detail.order,status}}})),{data:{...detail,order:{...detail.order,serviceItems:[{status:"listo",destination:"bar"}]}}}]){
  assert.equal(nodes(view.POSPaymentEntry(entryProps(changes))).some(node=>node.type==="PaymentDialog"),false);
 }
 assert.equal(view.POSPaymentEntry(entryProps()).type,"PaymentDialog");
 for(const status of ["listo","en_camino","entregado"])assert.equal(view.POSPaymentEntry(entryProps({data:{...detail,order:{...detail.order,channel:"delivery",status}}})).type,"PaymentDialog");
});

test("el envío se bloquea durante el cobro y conserva separados los detalles de consulta",()=>{
 for(const options of [{busy:true},{canManage:false,shift:false},{shift:false}]){
  const view=mountPage(options);
  view.entry.props.save({method:"cash",amount:"30",reference:""});
  assert.equal(view.sent.length,0);
 }
 const view=mountPage();
 const readonly=view.POSDetailDialog({loading:false,data:detail,canManage:true,hasShift:true,formatMoney:String,formatDateTime:String,close:()=>{},refund:()=>{}});
 assert.equal(nodes(readonly).some(node=>node.type==="PaymentDialog"||node.props.type==="submit"),false);
 const route=readFileSync(new URL("../src/app/(admin)/pos/page.tsx",import.meta.url),"utf8");
 assert.match(route,/key=\{params\.orderId\?\?""\}/);
});

test("el formulario de pago tiene Registrar cobro y envía solo al confirmar",()=>{
 for(const busy of [false,true]){
  let defaults,saved=0;
  const {PaymentDialog}=compile("pos-dialogs.tsx",name=>{
   if(name==="@/design-system")return primitives;
   if(name==="@/design-system/dialog")return{Dialog:"Dialog"};
   if(name==="react-hook-form")return{useForm:options=>{defaults=options.defaultValues;return{register:()=>({}),handleSubmit:save=>()=>save(defaults),formState:{errors:{}}}},useWatch:()=>"cash"};
   if(name.startsWith("../"))return{};
   return require(name);
  });
  const tree=PaymentDialog({order:{...detail.order,paidAmount:detail.paidAmount,remainingAmount:detail.remainingAmount,paymentStatus:detail.paymentStatus},methods:catalog,shiftName:"Caja principal",busy,formatMoney:String,close:()=>{},save:()=>saved++});
  const all=nodes(tree),submit=all.find(node=>node.type==="Button"&&node.props.type==="submit"),form=all.find(node=>node.type==="form");
  assert.ok(submit);
  assert.equal(submit.props.children,busy?"Registrando…":"Registrar cobro");
  assert.equal(submit.props.disabled,busy);
  assert.equal(form.props.inert,busy);
  assert.equal(defaults.amount,"30.00");
  assert.equal(saved,0);
  if(!busy){form.props.onSubmit();assert.equal(saved,1)}
 }
});

test("detalle de cobro sigue el patrón del pedido: cabecera con mesa, identidad, pestañas y resumen fijo",()=>{
 const {POSDetailDialog}=mountPage();
 const view=nodes(POSDetailDialog({loading:false,data:{...detail,order:{...detail.order,waiterName:"Daniel Machaca Rojasa",items:[{id:"line-1",name:"Puca picante con chicharrón",qty:"1",unitPrice:"40",note:"Sin sal"}]}},canManage:false,hasShift:false,formatMoney:String,formatDateTime:()=>"7 oct. 2026",close:()=>{},refund:()=>{}}));
 const header=view.find(node=>node.type==="ModalHeader");assert.equal(header.props.title,"Mesa 04");assert.equal(header.props.icon,"utensils");assert.equal(header.props.subtitle,undefined);
 assert.equal(view.some(node=>node.type==="DetailFacts"||node.type==="DetailHero"),false,"sin pestaña Detalles ni datos repetidos");
 assert.deepEqual(Array.from(view.find(node=>node.type==="Tabs").props.items,item=>item.label),["Productos","Pagos"]);
 const identity=nodes(view.find(node=>node.props.className==="pos-detail-identity"));
 assert.equal(identity.find(node=>node.type==="Status").props.tone,"orange");
 assert.ok(identity.some(node=>node.type==="span"&&node.props.children==="PED-001"),"en POS el código solo aparece en el detalle");
 assert.ok(identity.some(node=>node.type==="time"&&node.props.children==="7 oct. 2026"));
 assert.ok(identity.some(node=>node.type==="span"&&node.props.children==="Salón · Atendido por Daniel Machaca Rojasa"));
 assert.equal(view.some(node=>node.type==="DetailContact"),false,"sin cliente no hay tarjeta");
 const workspace=view.find(node=>node.type==="ModalWorkspace");assert.equal(workspace.props.summary.props.className,"modal-summary pos-detail-totals");
 assert.ok(nodes(workspace.props.summary).some(node=>node.props.children==="Pagado neto"));
 assert.ok(nodes(workspace.props.summary).some(node=>node.props.children==="Saldo pendiente"));
 assert.equal(nodes(workspace.props.summary).some(node=>node.type==="Status"),false,"el estado encabeza el cuerpo y no se repite en el resumen");
 assert.equal(view.filter(node=>node.type==="ModalItemList").length,1);
 assert.ok(view.some(node=>node.props.className==="modal-item-quantity"));
 assert.ok(view.some(node=>node.props.children==="Sin pagos registrados."));
 assert.equal(view.some(node=>node.props.className==="pos-detail-summary"),false);
 const paid=nodes(POSDetailDialog({loading:false,data:{...detail,order:{...detail.order,customerName:"Ana"},paidAmount:"40",remainingAmount:"0",paymentStatus:"paid"},canManage:false,hasShift:false,formatMoney:String,formatDateTime:String,close:()=>{},refund:()=>{}}));
 assert.equal(paid.find(node=>node.type==="ModalWorkspace").props.summary.props.className,"modal-summary pos-detail-totals is-paid");
 assert.ok(nodes(paid.find(node=>node.type==="ModalWorkspace").props.summary).some(node=>node.props.children==="Total pagado"));
 assert.equal(paid.find(node=>node.type==="DetailContact").props.name,"Ana");
 assert.match(readFileSync(new URL("pos-page.tsx",root),"utf8"),/<h3 id="pos-detail-payments-title" className="sr-only">/,"cada panel conserva su título accesible");
});

test("detalle de cobro conserva permisos de devolución, turno y cierre",()=>{
 const {POSDetailDialog}=mountPage();let refunded=0;
 const payment={id:"p1",method:"cash",methodName:"Efectivo",affectsCash:true,amount:"10",netAmount:"8",refundedAmount:"2",createdAt:detail.order.createdAt,cashRegisterName:"Caja principal",createdByName:"Ana"};
 const render=changes=>nodes(POSDetailDialog({loading:false,data:{...detail,payments:[payment]},canManage:true,hasShift:true,formatMoney:String,formatDateTime:String,close:()=>{},refund:()=>refunded++,...changes}));
 const action=render().find(node=>node.type==="Button"&&node.props.children==="Devolver");action.props.onClick();assert.equal(refunded,1);
 assert.equal(render({canManage:false}).some(node=>node.props.children==="Devolver"),false);
 assert.equal(render({hasShift:false}).find(node=>node.props.children==="Devolver").props.disabled,true);
 for(const order of [{...detail.order,completedAt:"2026-10-10T15:00:00Z"},{...detail.order,status:"cancelado"}])assert.equal(render({data:{...detail,order,payments:[payment]}}).some(node=>node.props.children==="Devolver"),false);
});

test("consultar el resumen de pago es secundario y no repite el cobro",()=>{
 const {POSDetailDialog}=mountPage();let opened=0;
 const props={loading:false,data:detail,canManage:false,hasShift:false,formatMoney:String,formatDateTime:String,close:()=>{},refund:()=>{},ticket:()=>opened++};
 const button=nodes(POSDetailDialog(props)).find(node=>node.type==="Button"&&node.props.children==="Previsualizar");
 assert.equal(button.props.kind,"secondary");assert.equal(button.props.icon,"eye");button.props.onClick();assert.equal(opened,1);
 for(const changes of [{loading:true},{error:"Sin conexión"},{data:undefined},{data:{...detail,paidAmount:"0"}}])assert.equal(nodes(POSDetailDialog({...props,...changes})).some(node=>node.props.children==="Previsualizar"),false);
});

test("detalle de cobro no muestra importes anteriores en carga, error o ausencia de datos",()=>{
 const {POSDetailDialog}=mountPage();const props={loading:false,data:detail,canManage:false,hasShift:false,formatMoney:String,formatDateTime:String,close:()=>{},refund:()=>{}};
 for(const changes of [{loading:true},{error:"Sin conexión"},{data:undefined}]){
  const tree=POSDetailDialog({...props,...changes}),view=nodes(tree);
  assert.equal(view.some(node=>node.props["aria-labelledby"]==="pos-detail-summary-title"),false);
  if(changes.loading)assert.equal(view.some(node=>node.type?.name==="POSDetailLoading"),true,"skeleton con la geometría final");
 else assert.equal(view.find(node=>node.type==="ModalHeader").props.details,undefined);
  if(changes.loading)assert.equal(view.find(node=>node.type==="Dialog").props["aria-busy"],true);
  if(changes.error)assert.equal(view.find(node=>node.type==="ErrorState").props.text,"Sin conexión");
 }
});

test("detalle POS usa el tamaño y la hoja móvil del pedido; el cobro conserva pantalla completa",()=>{
 const css=readFileSync(new URL("pos.css",root),"utf8");
 const order=readFileSync(new URL("../../orders/presentation/order-detail.css",root),"utf8");
 const rule=(sheet,selector)=>sheet.match(new RegExp(`(?:^|\\n)${selector.replace(/[.()]/g,"\\$&")}\\{([^}]*)\\}`))?.[1];
 assert.equal(rule(css,".crud-modal.pos-detail-modal"),rule(order,".crud-modal.order-detail"),"mismo ancho, alto y columna que el pedido");
 assert.match(css,/@media\(max-width:600px\)\{\s*\.crud-modal\.pos-payment-modal\{width:100%;height:100dvh;max-height:100dvh/);
 assert.match(css,/\.modal-backdrop:has\(\.pos-detail-modal\)\{padding:0\}/);
 assert.match(css,/ \.crud-modal\.pos-detail-modal\{width:100%;max-height:calc\(100dvh - var\(--space-4\) - var\(--safe-area-top\)\)\}/);
 assert.equal(css.includes("width:100vw"),false);
 assert.doesNotMatch(css,/\.pos-detail-context/);
 assert.doesNotMatch(css,/\.pos-detail-tabs/,"pestañas compartidas a 12 px del contenido");
});

test("detalle de cobro usa la línea compartida del modal y muestra su barra",()=>{
 const css=readFileSync(new URL("pos.css",root),"utf8");
 assert.doesNotMatch(css,/\.pos-order-lines/,"las líneas de producto usan la línea compartida del modal");
 const shared=readFileSync(new URL("../src/design-system/styles/modal.css",import.meta.url),"utf8");
 assert.match(shared,/\.modal-item-name\{[^}]*font-weight:var\(--weight-semibold\)/);
 // Los modales muestran la barra siempre que hay más contenido; hover-scroll (oculta hasta el hover) queda para vistas como el menú.
 assert.match(shared,/\.modal-body\{[^}]*overflow-y:auto[^}]*scrollbar-width:thin/);
 for(const file of ["pos-page.tsx","../../../sales/presentation/sale-detail-dialog.tsx"])assert.doesNotMatch(readFileSync(new URL(file,root),"utf8"),/modal-body[^"]*hover-scroll/,`${file}: el cuerpo del modal no oculta su barra`);
});

test("el detalle identifica mesa, cliente o venta directa una sola vez",()=>{
 const {POSDetailDialog}=mountPage();
 for(const [order,title] of [
  [{...detail.order,tableName:"10",customerName:"Ana"},"Mesa 10"],
  [{...detail.order,channel:"delivery",tableName:"",customerName:"Ana"},"Ana"],
  [{...detail.order,channel:"pickup",tableName:"",customerName:""},"Venta directa"],
 ]){
  const view=nodes(POSDetailDialog({loading:false,data:{...detail,order},canManage:false,hasShift:false,formatMoney:String,formatDateTime:()=>"Hoy",close:()=>{},refund:()=>{}}));
  assert.equal(view.find(node=>node.type==="ModalHeader").props.title,title);
  assert.equal(view.some(node=>node.type==="DetailFacts"||node.props?.children==="Cliente / mesa"),false);
 }
});

test("el cobro usa el catálogo de medios de pago y la nota de caja sigue a Impacta caja",()=>{
 for(const watched of ["cash","yape"]){
  let defaults;
  const {PaymentDialog}=compile("pos-dialogs.tsx",name=>{
   if(name==="@/design-system")return primitives;
   if(name==="@/design-system/dialog")return{Dialog:"Dialog"};
   if(name==="react-hook-form")return{useForm:options=>{defaults=options.defaultValues;return{register:()=>({}),handleSubmit:save=>()=>save(defaults),formState:{errors:{}}}},useWatch:()=>watched};
   if(name.startsWith("../"))return{};
   return require(name);
  });
  const all=nodes(PaymentDialog({order:{...detail.order,paidAmount:detail.paidAmount,remainingAmount:detail.remainingAmount,paymentStatus:detail.paymentStatus},methods:catalog,shiftName:"Caja principal",busy:false,formatMoney:String,close:()=>{},save:()=>{}}));
  const labels=all.filter(node=>node.type==="label"&&String(node.props.className).startsWith("pos-method")).map(node=>node.props.children.at(-1).props.children);
  assert.deepEqual(labels,["Efectivo","Tarjeta","Yape","Plin"],"las opciones vienen del catálogo, en su orden");
  assert.equal(defaults.method,"cash","el primer medio activo queda seleccionado");
  assert.equal(all.some(node=>node.props?.className==="pos-cash-impact"),watched==="cash","solo un medio que impacta caja anuncia el efectivo esperado");
  assert.equal(all.some(node=>node.type==="FormField"&&node.props.label==="Referencia"),watched!=="cash");
 }
 const source=readFileSync(new URL("pos-dialogs.tsx",root),"utf8");
 assert.equal(source.includes('["cash","card","transfer","other"]'),false,"el cobro no fija los medios en el frontend");
});

test("el cobro espera el catálogo y explica errores o ausencia de medios",()=>{
 const view=mountPage();
 const loading=nodes(view.POSPaymentEntry(entryProps({methodsLoading:true,methods:undefined})));
 assert.ok(loading.some(node=>node.type===view.POSPaymentLoading));
 const failed=nodes(view.POSPaymentEntry(entryProps({methodsError:"Sin conexión",methods:undefined}))).find(node=>node.type==="ErrorState");
 assert.equal(failed.props.title,"No pudimos cargar los medios de pago");
 const empty=nodes(view.POSPaymentEntry(entryProps({methods:[]}))).find(node=>node.type==="EmptyState");
 assert.equal(empty.props.title,"Sin medios de pago para ventas");
 assert.ok(view.queries.some(options=>options.queryKey[0]==="payment-methods"&&options.queryKey[1]==="sales"));
});
