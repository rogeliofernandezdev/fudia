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
const primitives=Object.fromEntries(["Button","FormField","Icon","Input","PageHeader","Pagination","RowActionButton","Select","Status","Textarea"].map(name=>[name,name]));
function nodes(tree){const result=[];function visit(node){if(!node||typeof node!=="object")return;if(Array.isArray(node)){node.forEach(visit);return}result.push(node);visit(node.props?.children)}visit(tree);return result}
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
   useQuery:options=>{queries.push(options);return{isLoading:false,isPending:false,isFetching:false,isError:false,refetch:()=>{},data:options.queryKey[0]==="cash-shift"?{shift:shift?{cashRegisterName:"Caja principal",code:"CAJ-001",openedByName:"Ana"}:null}:options.queryKey[0]==="pos-orders"?{items:listed,total:listed.length}:options.enabled?detail:undefined}},
   useMutation:options=>{mutations.push(options);return{isPending:busy,mutate:input=>sent.push(input)}},
  };
  if(name==="@/design-system")return primitives;
  if(name==="@/design-system/dialog")return{Dialog:"Dialog"};
  if(name==="@/providers")return{useFeedback:()=>({notify:value=>notices.push(value)}),useSession:()=>({can:()=>canManage,location:{country:"PE",timezone:"America/Lima"}})};
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
function entryProps(changes={}){return{loading:false,data:detail,shiftLoading:false,shiftName:"Caja principal",canManage:true,busy:false,formatMoney:value=>`S/ ${value}`,close:()=>{},retryOrder:()=>{},retryShift:()=>{},save:()=>{},...changes}}

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
  const retry=nodes(tree).find(node=>node.type==="Button"&&node.props.children==="Reintentar");
  assert.ok(retry);retry.props.onClick();
  assert.equal(nodes(tree).some(node=>node.type==="PaymentDialog"),false);
 }
 assert.equal(retries,2);
 const missing=nodes(view.POSPaymentEntry(entryProps({shiftName:null})));
 assert.ok(missing.some(node=>node.type==="Link"&&node.props.href==="/cash"));
 assert.equal(missing.some(node=>node.type==="PaymentDialog"),false);
});

test("el formulario no permite cobrar sin permiso, pedidos pagados, cancelados o con cuenta abierta",()=>{
 const view=mountPage();
 for(const changes of [{canManage:false},{data:{...detail,paymentStatus:"paid",remainingAmount:"0"}},{data:{...detail,order:{...detail.order,completedAt:"2026-10-07T14:00:00Z"}}},{data:{...detail,order:{...detail.order,status:"cancelado"}}},{data:{...detail,order:{...detail.order,billClosedAt:undefined,status:"preparando"}}},{data:{...detail,order:{...detail.order,channel:"delivery",status:"preparando"}}},...["confirmado","preparando","listo"].map(status=>({data:{...detail,order:{...detail.order,status}}})),{data:{...detail,order:{...detail.order,serviceItems:[{status:"listo",destination:"bar"}]}}}]){
  assert.equal(nodes(view.POSPaymentEntry(entryProps(changes))).some(node=>node.type==="PaymentDialog"),false);
 }
 assert.equal(view.POSPaymentEntry(entryProps()).type,"PaymentDialog");
 for(const status of ["listo","en_camino","entregado"])assert.equal(view.POSPaymentEntry(entryProps({data:{...detail,order:{...detail.order,channel:"delivery",status}}})).type,"PaymentDialog");
});

test("el envío se bloquea durante el cobro y conserva separados los detalles de consulta",()=>{
 for(const options of [{busy:true},{canManage:false},{shift:false}]){
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
   if(name==="./pos-meta")return{paymentMethodMeta:Object.fromEntries(["cash","card","transfer","other"].map(value=>[value,{icon:"cash",label:value}]))};
   if(name.startsWith("../"))return{};
   return require(name);
  });
  const tree=PaymentDialog({order:{...detail.order,paidAmount:detail.paidAmount,remainingAmount:detail.remainingAmount,paymentStatus:detail.paymentStatus},shiftName:"Caja principal",busy,formatMoney:String,close:()=>{},save:()=>saved++});
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
