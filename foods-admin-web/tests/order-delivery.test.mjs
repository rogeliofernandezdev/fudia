import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
import postcss from "postcss";

const require=createRequire(import.meta.url);
const root=new URL("../src/modules/operations/",import.meta.url);
function compile(path,resolve=require,extra=""){
 const exports={};
 const source=readFileSync(new URL(path,root),"utf8")+extra;
 vm.runInNewContext(ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve});
 return exports;
}
const actions=compile("orders/domain/order-actions.ts");
const serviceFlow=compile("orders/domain/service-flow.ts");
const accountLabel=o=>o.completedAt?"Finalizada":o.accountState==="paid"?"Pagada":o.billClosedAt?"Por cobrar":"Cuenta abierta";
const order={id:"pedido-1",code:"PED-001",channel:"salon",status:"listo",billClosedAt:"2026-10-05T11:00:00Z",paymentStatus:"pending",customerName:"",customerPhone:"",tableName:"Mesa 1",tableId:"mesa-1",total:"25.00",subtotal:"25.00",deliveryFee:"0",paidAmount:"0",remainingAmount:"25.00",createdAt:"2026-10-05T10:00:00Z",updatedAt:"2026-10-05T10:00:00Z",items:[]};

test("la entrega de mesa no depende de haber cobrado",()=>{
 for(const paymentStatus of ["pending","partial","paid"]){
  const action=actions.nextOrderAction({...order,paymentStatus});
  assert.equal(action.status,"entregado");
  assert.equal(action.label,"Confirmar entrega");
 }
});

test("una mesa entregada no exige una segunda acción para liberarse",()=>{
 assert.equal(actions.nextOrderAction({...order,status:"entregado"}),null);
 assert.equal(actions.nextOrderAction({...order,status:"entregado",paymentStatus:"partial"}),null);
 assert.equal(actions.nextOrderAction({...order,status:"entregado",paymentStatus:"paid"}),null);
 assert.equal(actions.nextOrderAction({...order,status:"entregado",paymentStatus:"paid",completedAt:"2026-10-05T12:00:00Z"}),null);
});

test("los detalles abiertos reflejan el cobro de otro usuario y dejan de consultar tras el cierre",()=>{
 for(const path of ["salon/presentation/salon-manager.tsx","orders/presentation/orders-manager.tsx"]){
  const source=readFileSync(new URL(path,root),"utf8");
  assert.match(source,/refetchInterval:query=>query\.state\.data\?\.completedAt\?false:10000/);
 }
});

test("Cocina conserva el control de preparación y delivery conserva su despacho",()=>{
 assert.equal(actions.nextOrderAction({...order,status:"confirmado"}),null);
 assert.equal(actions.nextOrderAction({...order,status:"preparando"}),null);
 assert.equal(actions.nextOrderAction({...order,status:"cancelado"}),null);
 assert.equal(actions.nextOrderAction({...order,channel:"delivery"}).status,"en_camino");
 assert.equal(actions.nextOrderAction({...order,channel:"delivery",status:"en_camino"}).status,"entregado");
});

function mount(section,changes={},canManage=true,busy=false,fullControls=false){
 const calls=[];
 const file=section==="salon"?"salon/presentation/salon-manager.tsx":"orders/presentation/orders-manager.tsx";
 const resolve=name=>{
  if(name==="next/link")return{default:"Link"};
  if(name==="@/design-system")return{Button:"Button",Status:"Status",Icon:"Icon"};
  if(name==="@/design-system/icons")return{Icon:"Icon"};
  if(name==="@/providers/session-context")return{useSession:()=>({user:{id:"waiter"},can:()=>canManage,location:{country:"PE",timezone:"America/Lima"}})};
  if(name==="@/providers")return{useSession:()=>({user:{id:"waiter"},can:()=>canManage}),useFeedback:()=>({notify:()=>{}})};
  if(name==="@tanstack/react-query")return{useQueryClient:()=>({invalidateQueries:()=>{},setQueryData:()=>{}}),useMutation:()=>({isPending:false,mutate:()=>{}})};
  if(name==="@/shared/i18n/regional-format")return{formatRegionalDateTime:()=>"Hace un momento"};
  if(name==="@/shared/routing/page-routes")return{pageRoutes:{pos:"/pos"}};
  if(name.endsWith("/domain/order-actions"))return actions;
  if(name.endsWith("/domain/service-flow"))return serviceFlow;
  if(name.endsWith("order-service-controls"))return fullControls?compile("orders/presentation/order-service-controls.tsx",resolve):{accountLabel,OrderAccountActions:"OrderAccountActions",OrderItemService:"OrderItemService"};
  if(name.startsWith("@/")||name.startsWith("../")||name.startsWith("./"))return{};
  return require(name);
 };
 const {OrderDetail}=compile(file,resolve,"\nexport {OrderDetail};\n");
 const rendered=OrderDetail({loading:false,order:{...order,...changes},currencySymbol:"S/",channels:[{value:"salon",label:"Salón"}],canManage,busy,close:()=>{},advance:status=>calls.push(status),edit:()=>{},cancel:()=>{}});
 const nodes=[];
 function visit(node){if(!node||typeof node!=="object")return;if(Array.isArray(node)){node.forEach(visit);return}nodes.push(node);if(fullControls&&typeof node.type==="function")visit(node.type(node.props));else visit(node.props?.children)}
 visit(rendered);
 return{calls,nodes,buttons:nodes.filter(n=>n.type==="Button"),links:nodes.filter(n=>n.type==="Link")};
}

test("salon: tarjetas y detalle comparten el estado operativo sin mezclar el cobro",()=>{
 const source=readFileSync(new URL("salon/presentation/salon-manager.tsx",root),"utf8");
 const {statusMeta}=compile("salon/presentation/salon-manager.tsx",()=>({}),"\nexport {statusMeta};\n");
 assert.equal(statusMeta.listo.label,"Listo para entregar");
 assert.equal(statusMeta.listo.tone,"green");
 assert.equal(statusMeta.entregado.label,"Entregado");
 assert.doesNotMatch(source,/por cobrar/i);
 assert.match(source,/const meta=o\?statusMeta\[o\.status\]/);
 for(const paymentStatus of ["pending","partial","paid"]){
  const view=mount("salon",{paymentStatus});
  assert.equal(view.nodes.find(n=>n.type==="Status").props.children,statusMeta.listo.label);
 }
});

test("salon: la cabecera distingue la preparación del estado de cuenta",()=>{
 for(const changes of [{},{paymentStatus:"paid",paidAmount:"25.00",remainingAmount:"0"},{status:"entregado"}]){
  const view=mount("salon",changes);
  const statuses=view.nodes.filter(n=>n.type==="Status");
  assert.equal(statuses.length,2);
  assert.equal(statuses[1].props.children,"Por cobrar");
  assert.equal(statuses[0].props.children,changes.status==="entregado"?"Entregado":"Listo para entregar");
  assert.ok(view.nodes.some(n=>n.props.className==="salon-order-detail-subtotal"));
  assert.ok(view.nodes.some(n=>n.props.className==="salon-order-detail-grand"));
 }
 assert.ok(mount("salon",{status:"entregado"}).links.some(n=>n.props.href==="/pos?orderId=pedido-1"));
});

test("salon: la franja decorativa conserva el degradado de la paleta",()=>{
 const css=readFileSync(new URL("styles/salon.css",root),"utf8");
 assert.match(css,/\.salon-order-detail-accent\{[^}]*background:linear-gradient\(90deg,var\(--brand-600\),var\(--primary-600\)\)/);
 const accent=mount("salon").nodes.find(n=>n.props.className==="salon-order-detail-accent");
 assert.equal(accent.props["aria-hidden"],"true");
});

test("salon: entregar precede al cobro; cobrar es primario solo con cuenta entregada y cerrada",()=>{
 const view=mount("salon");
 const delivery=view.buttons.find(n=>n.props.children==="Confirmar entrega");
 assert.equal(delivery.props.kind,undefined); // Button's default is primary
 assert.equal(delivery.props.icon,"availability");
 assert.equal(view.buttons.some(n=>n.props.className==="order-detail-primary"),true);
 assert.equal(view.links.length,0);
 assert.match(mount("salon",{status:"entregado"}).links[0].props.className,/\bprimary\b/);
 assert.ok(view.nodes.some(n=>n.props.role==="group"&&n.props["aria-label"]==="Acciones de la mesa"));
});

test("salon: acciones y notas tienen una apariencia inequívocamente distinta",()=>{
 const css=readFileSync(new URL("styles/salon.css",root),"utf8");
 const secondary=[...css.matchAll(/\.salon-order-detail-deliver,\s*\.salon-order-detail \.salon-order-detail-actions \.salon-order-detail-edit\{([^}]+)\}/g)].map(match=>match[1]).find(rule=>rule.includes("border:"));
 assert.ok(secondary);
 assert.match(secondary,/border:var\(--stroke-1\) solid var\(--primary-600\)/);
 assert.match(secondary,/background:var\(--surface\)/);
 assert.match(secondary,/color:var\(--primary-700\)/);
 const notice=css.match(/\.salon-order-detail-notice\{([^}]+)\}/)[1];
 assert.doesNotMatch(notice,/(?:background|border|cursor):/);
 assert.match(css,/\.salon-order-detail-buttons\{[^}]*justify-content:flex-end/);
});

test("salon: productos y cuenta comparten una columna sin código técnico ni cantidades duplicadas",()=>{
 const view=mount("salon",{items:[{id:"item-1",name:"Plato",qty:"2",unitPrice:"12.50"}]});
 assert.equal(view.nodes.some(n=>n.props.className==="salon-order-detail-meta"),false);
 assert.equal(view.nodes.some(n=>n.props.children===order.code),false);
 assert.equal(view.nodes.find(n=>n.props.id==="salon-order-detail-title").props.children,"Mesa 1");
 const body=view.nodes.find(n=>n.props.className==="order-detail-body salon-order-detail-body");
 const content=body.props.children;
 assert.equal(content.props.className,"salon-order-detail-content");
 assert.ok(content.props.children.some(n=>n.props.className?.startsWith("salon-order-detail-totals")));
 assert.equal(view.nodes.find(n=>n.props.id==="salon-order-account-title").props.children,"Resumen de cuenta");
 assert.equal(view.nodes.some(n=>n.props.children==="DETALLE"),false);
 const css=readFileSync(new URL("styles/salon.css",root),"utf8");
 assert.match(css,/\.salon-order-detail-content\{[^}]*flex-direction:column/);
 assert.match(css,/\.salon-order-detail-totals\{[^}]*grid-template-columns:repeat\(2,minmax\(0,1fr\)\) minmax\(0,1.25fr\)/);
 assert.match(css,/\.salon-order-detail-grand\{[^}]*align-items:flex-end/);
 assert.equal(view.nodes.find(n=>n.props.id==="salon-order-account-title").props.className,"sr-only");
 assert.match(css,/\.salon-order-detail\{[^}]*max-width:var\(--size-680\)/);
});

test("salon: cuenta agrupada con importes legibles y saldo destacado sin apariencia de botón",()=>{
 const css=readFileSync(new URL("styles/salon.css",root),"utf8");
 const balance=css.match(/\.salon-order-detail-grand\{([^}]+)\}/)[1];
 assert.match(balance,/background:var\(--primary-100\)/);
 assert.doesNotMatch(balance,/(?:cursor|transform|box-shadow):/);
 assert.match(css,/\.salon-order-detail-subtotal b\{[^}]*font-size:var\(--font-size-18\)/);
 assert.match(css,/\.salon-order-detail-grand strong\{[^}]*font-size:var\(--font-size-24\)/);
 assert.match(css,/\.salon-order-detail-totals\{[^}]*background:var\(--cloud-50\)/);
 assert.match(css,/\.salon-order-detail-totals\.is-paid \.salon-order-detail-grand\{[^}]*background:var\(--brand-100\)/);
 const footer=css.match(/\.salon-order-detail \.salon-order-detail-actions\{([^}]+)\}/)[1];
 assert.doesNotMatch(footer,/flex-direction:column/);
 assert.match(footer,/align-items:center/);
 assert.match(css,/@media\(max-width:600px\)\{[\s\S]*\.salon-order-detail-grand\{[^}]*grid-column:1\/-1/);
 const view=mount("salon",{deliveryFee:"5",subtotal:"20"});
 assert.ok(view.nodes.some(n=>n.props.children==="Delivery"));
 assert.equal(view.nodes.filter(n=>n.props.className==="salon-order-detail-grand").length,1);
});

test("salon: cantidad circular y pie sin instrucciones redundantes de entrega o cobro",()=>{
 const view=mount("salon",{items:[{id:"item-1",name:"Plato",qty:"2",unitPrice:"12.50"}]});
 assert.equal(view.nodes.find(n=>n.props.className==="salon-order-detail-qty").props.children.join(""),"2×");
 for(const status of ["listo","entregado"]){
  const pending=mount("salon",{status});
  assert.equal(pending.nodes.some(n=>n.props.className==="salon-order-detail-notice"),false);
 }
 const css=readFileSync(new URL("styles/salon.css",root),"utf8");
 const qty=css.match(/\.salon-order-detail-qty\{([^}]+)\}/)[1];
 assert.match(qty,/width:var\(--size-32\)/);
 assert.match(qty,/height:var\(--size-32\)/);
 assert.match(qty,/border-radius:var\(--radius-999\)/);
 assert.match(qty,/align-self:center/);
 assert.match(css,/\.salon-order-detail-skeleton-qty\{[^}]*height:var\(--size-32\)[^}]*border-radius:var\(--radius-999\)/);
 const restricted=mount("salon",{status:"nuevo",paidAmount:"5",remainingAmount:"20"});
 assert.ok(restricted.nodes.some(n=>n.props.className==="salon-order-detail-notice"));
});

test("salon: la cuenta móvil alinea subtotal, pagado y saldo en filas completas",()=>{
 const css=postcss.parse(readFileSync(new URL("styles/salon.css",root),"utf8"));
 const mobileRules=css.nodes.filter(node=>node.type==="atrule"&&node.name==="media"&&node.params==="(max-width:600px)").flatMap(node=>node.nodes);
 const declarations=selector=>Object.fromEntries(mobileRules.find(node=>node.type==="rule"&&node.selectors.includes(selector)).nodes.filter(node=>node.type==="decl").map(node=>[node.prop,node.value]));
 assert.equal(declarations(".salon-order-detail-totals")["grid-template-columns"],"minmax(0,1fr)");
 const row=declarations(".salon-order-detail-subtotal");
 assert.equal(row["flex-direction"],"row");
 assert.equal(row["align-items"],"center");
 assert.equal(row["justify-content"],"space-between");
 assert.equal(row.padding,"0 var(--space-12)");
 const balance=declarations(".salon-order-detail-grand");
 assert.equal(balance["flex-direction"],"row");
 assert.equal(balance["justify-content"],row["justify-content"]);
 for(const variant of ["primary","secondary"]){
  assert.equal(declarations(`.salon-order-detail-skeleton-actions .action.${variant}`).flex,"0 0 auto");
 }
 const status=declarations(".salon-order-detail-skeleton-status");
 assert.equal(status["grid-row"],"2");
 assert.equal(status["grid-column"],"1/-1");
});

test("salon: título de mesa consistente y reglas del detalle sin selectores duplicados",()=>{
 assert.equal(mount("salon",{tableName:"04"}).nodes.find(n=>n.props.id==="salon-order-detail-title").props.children,"Mesa 04");
 assert.equal(mount("salon",{tableName:"Mesa 04"}).nodes.find(n=>n.props.id==="salon-order-detail-title").props.children,"Mesa 04");
 const seen=new Set();
 postcss.parse(readFileSync(new URL("styles/salon.css",root),"utf8")).walkRules(rule=>{
  for(const selector of rule.selectors.filter(value=>value.includes("salon-order-detail"))){
   const context=rule.parent.type==="atrule"?rule.parent.params:"root";
   const key=context+"|"+selector;
   assert.equal(seen.has(key),false,key);seen.add(key);
  }
 });
});

test("salon: verde solo representa pagos registrados y cuenta completamente pagada",()=>{
 for(const [paidAmount,remainingAmount] of [["0","25"],["10","15"],["25","0"]]){
  const view=mount("salon",{paidAmount,remainingAmount});
  const totals=view.nodes.find(n=>n.props.className?.startsWith("salon-order-detail-totals"));
  const paid=view.nodes.find(n=>n.props.className?.includes("salon-order-detail-paid"));
  assert.equal(totals.props.className.includes("is-paid"),Number(remainingAmount)===0);
  assert.equal(paid.props.className.includes("has-payment"),Number(paidAmount)>0);
 }
});

test("salon: la mesa pagada y entregada está liberada sin otro botón",()=>{
 const paid=mount("salon",{status:"entregado",paymentStatus:"paid",paidAmount:"25.00",remainingAmount:"0",completedAt:"2026-10-05T12:00:00Z"});
 assert.equal(paid.buttons.some(n=>n.props.children==="Liberar mesa"),false);
 assert.ok(paid.nodes.some(n=>JSON.stringify(n.props.children)?.includes("Mesa liberada")));
 const initial=mount("salon",{status:"nuevo"});
 const primary=initial.buttons.filter(n=>n.props.className==="order-detail-primary");
 assert.equal(primary.length,1);
 assert.equal(primary[0].props.icon,"chefHat");
 assert.equal(initial.buttons.find(n=>n.props.children==="Editar comanda").props.kind,"secondary");
 assert.equal(initial.buttons.find(n=>n.props.children==="Cancelar pedido").props.kind,"ghost");
});

test("salon: el cobro queda bloqueado mientras se guarda otra acción",()=>{
 const link=mount("salon",{status:"entregado"},true,true).links[0];
 assert.equal(link.props["aria-disabled"],true);
 assert.equal(link.props["aria-busy"],true);
 assert.equal(link.props.tabIndex,-1);
 let prevented=false;
 link.props.onClick({preventDefault:()=>{prevented=true}});
 assert.equal(prevented,true);
});

for(const section of ["salon","orders"]){
 test(`${section}: entrega global usa una petición de estado, refresca vistas y no abre éxito`,async()=>{
  const mutations=[],sent=[],invalidations=[],feedback=[];
  let finishRefresh;
  const refresh=new Promise(resolve=>{finishRefresh=resolve});
  const delivered={...order,status:"entregado",billClosedAt:undefined};
  const update=(id,status)=>{sent.push({id,status});return delivered};
  const session=()=>({user:{id:"waiter"},can:()=>true,location:{country:"PE",timezone:"America/Lima"}});
  const managerExports=compile(`${section}/presentation/${section==="salon"?"salon":"orders"}-manager.tsx`,name=>{
   if(name==="next/dynamic")return{default:()=>"ManualOrderDialog"};
   if(name==="next/link")return{default:"Link"};
   if(name==="react")return{...require(name),useState:value=>[value,()=>{}],useCallback:fn=>fn};
   if(name==="@tanstack/react-query")return{useQueryClient:()=>({invalidateQueries:q=>{invalidations.push(q.queryKey);return refresh}}),useMutation:config=>{mutations.push(config);return{}},useQuery:()=>({data:{items:[],channelCounts:{},channelOptions:[],statusOptions:[],total:0},isLoading:false,isError:false})};
   if(name==="@/providers/session-context")return{useSession:session};
   if(name==="@/providers/feedback-provider")return{useFeedback:()=>({notify:value=>feedback.push(value)})};
   if(name==="@/providers/settings-context")return{useSettings:()=>({currencySymbol:"S/"})};
   if(name==="@/shared/hooks/use-debounced-value")return{useDebouncedValue:value=>value};
   if(name.endsWith("salon-api"))return{updateSalonOrderStatus:update};
   if(name.endsWith("orders-api"))return{updateOrderStatus:update};
   if(name.endsWith("order-actions"))return actions;
   if(name.startsWith("@/")||name.startsWith("."))return{};
   return require(name);
  });
  managerExports[section==="salon"?"SalonManager":"OrdersManager"]();
  const config=mutations[section==="salon"?0:1];
  const response=await config.mutationFn({id:"pedido-1",status:"entregado"});
  const refreshed=config.onSuccess(response);
  assert.equal(typeof refreshed.then,"function"); // Query keeps the action pending until refresh finishes
  let completed=false;refreshed.then(()=>{completed=true});
  await Promise.resolve();assert.equal(completed,false);
  finishRefresh();await refreshed;assert.equal(completed,true);
  assert.deepEqual(sent,[{id:"pedido-1",status:"entregado"}]);
  assert.equal(feedback.length,0);
  for(const key of ["order","orders","salon-floor","pos-orders","pos-order","kitchen-tickets","dashboard","sales"])assert.ok(invalidations.some(q=>q[0]===key),key);
  config.onError(new Error("Todavía hay productos en preparación."));
  assert.equal(feedback.at(-1).tone,"danger");
  assert.equal(feedback.at(-1).message,"Todavía hay productos en preparación.");
 });
 test(`${section}: una sola entrega para Cocina, Barra y directos; sin botones por producto`,()=>{
  const serviceItems=["kitchen","bar","direct"].map((destination,i)=>({id:`service-${i}`,orderItemId:"item",destination,destinationLabel:destination,status:"listo",statusLabel:"Listo",name:"Producto"}));
  const ready={waiterId:"waiter",billClosedAt:undefined,items:[{id:"item",name:"Pedido",qty:"3",unitPrice:"10"}],serviceItems};
  const view=mount(section,ready,true,false,true);
  const delivery=view.buttons.filter(n=>n.props.children==="Confirmar entrega");
  assert.equal(delivery.length,1);
  assert.equal(delivery[0].props.className,"order-detail-primary");
  assert.equal(view.buttons.some(n=>n.props.children==="Entregar"||n.props.children==="Cerrar cuenta"),false);
  delivery[0].props.onClick();assert.deepEqual(view.calls,["entregado"]);
  for(const destination of ["kitchen","bar","direct"])for(const status of ["nuevo","confirmado","preparando"]){
   const pending={...ready,serviceItems:[...serviceItems,{id:"pending",orderItemId:"item",destination,status}]};
   assert.equal(mount(section,pending,true,false,true).buttons.some(n=>n.props.children==="Confirmar entrega"),false);
  }
  for(const changes of [{waiterId:"other"},{completedAt:"date"}])assert.equal(mount(section,{...ready,...changes},true,false,true).buttons.some(n=>n.props.children==="Confirmar entrega"),false);
  assert.equal(mount(section,ready,false,false,true).buttons.some(n=>n.props.children==="Confirmar entrega"),false);
  assert.ok(mount(section,ready,true,true,true).buttons.every(n=>n.props.disabled));
  const served={...ready,status:"entregado",serviceItems:serviceItems.map(item=>({...item,status:"entregado"}))};
  const after=mount(section,served,true,false,true);
  assert.equal(after.buttons.filter(n=>n.props.children==="Cerrar cuenta").length,1);
  assert.equal(after.buttons.some(n=>n.props.children==="Confirmar entrega"||n.props.children==="Entregar"),false);
  const additional={...served,status:"listo",serviceItems:[...served.serviceItems,{...serviceItems[0],id:"new"}]};
  assert.equal(mount(section,additional,true,false,true).buttons.filter(n=>n.props.children==="Confirmar entrega").length,1);
 });
 test(`${section}: permite cancelar entregas directas sin servir y bloquea preparación, entrega o pagos`,()=>{
  const direct={destination:"direct",status:"listo"};
  const eligible={status:"listo",waiterId:"waiter",serviceItems:[direct]};
  const hasCancel=changes=>mount(section,changes).buttons.some(n=>n.props.children==="Cancelar pedido");
  assert.equal(hasCancel(eligible),true);
  assert.equal(hasCancel({...eligible,status:"confirmado",serviceItems:[direct,{destination:"kitchen",status:"confirmado"}]}),true);
  for(const patch of [
   {paidAmount:"5"},{completedAt:"date"},{status:"en_camino"},{status:"entregado"},{status:"cancelado"},
   {waiterId:"other"},{serviceItems:[{...direct,status:"entregado"}]},
   {serviceItems:[direct,{destination:"kitchen",status:"preparando"}]},
   {serviceItems:[direct,{destination:"bar",status:"listo"}]},
   {serviceItems:[direct,{destination:"kitchen",status:"entregado"}]},
  ])assert.equal(hasCancel({...eligible,...patch}),false,JSON.stringify(patch));
  assert.equal(mount(section,eligible,false).buttons.some(n=>n.props.children==="Cancelar pedido"),false);
  assert.equal(mount(section,eligible,true,true).buttons.find(n=>n.props.children==="Cancelar pedido").props.disabled,true);
  assert.equal(hasCancel({status:"listo",serviceItems:[]}),false);
 });
 test(`${section}: el detalle ofrece la acción real de entrega antes de cobrar`,()=>{
  const view=mount(section);
  const button=view.buttons.find(n=>n.props.children==="Confirmar entrega");
  assert.ok(button);
  assert.equal(button.props.disabled,false);
  button.props.onClick();
  assert.deepEqual(view.calls,["entregado"]);
 });
 test(`${section}: el cobro requiere entrega y cuenta cerrada; al finalizar no ofrece acciones`,()=>{
  assert.equal(mount(section).links.some(n=>n.props.href==="/pos?orderId=pedido-1"),false);
  const pending=mount(section,{status:"entregado"});
  assert.equal(pending.buttons.some(n=>n.props.children==="Liberar mesa"),false);
  assert.ok(pending.links.some(n=>n.props.href==="/pos?orderId=pedido-1"));
  const paid=mount(section,{status:"entregado",paymentStatus:"paid",paidAmount:"25.00",remainingAmount:"0",completedAt:"2026-10-05T12:00:00Z"});
  assert.equal(paid.buttons.some(n=>n.props.children==="Liberar mesa"),false);
  assert.equal(paid.links.some(n=>n.props.href==="/pos?orderId=pedido-1"),false);
 });
 test(`${section}: la acción respeta permisos y bloquea el botón mientras guarda`,()=>{
  assert.equal(mount(section,{},false).buttons.some(n=>n.props.children==="Confirmar entrega"),false);
  assert.equal(mount(section,{},true,true).buttons.find(n=>n.props.children==="Confirmar entrega").props.disabled,true);
 });
}

test("orders: fecha en cabecera, unidades únicas y total dentro del cuerpo desplazable",()=>{
 const view=mount("orders",{items:[{id:"item-1",name:"Ají de gallina",qty:"2",unitPrice:"12.50"}]});
 assert.equal(view.nodes.some(n=>n.props.className?.includes("salon-order-detail-meta")),false);
 assert.equal(view.nodes.some(n=>["DETALLE","REGISTRADO","CONSUMO"].includes(n.props.children)),false);
 assert.ok(view.nodes.some(n=>n.props.className==="salon-order-detail-opened"));
 const body=view.nodes.find(n=>n.props.className==="order-detail-body salon-order-detail-body");
 assert.equal(body.props.children.props.className,"salon-order-detail-content");
 assert.ok(body.props.children.props.children.some(n=>n?.props?.className==="salon-order-detail-totals"));
 assert.equal(view.nodes.find(n=>n.props.className==="salon-order-detail-section-head").props.children[1].props.children.join(""),"2 unidades");
 assert.ok(view.nodes.some(n=>n.type==="small"&&n.props.children?.join?.("")==="S/ 12.50 c/u"));
 assert.ok(view.nodes.some(n=>n.props.role==="group"&&n.props["aria-label"]==="Acciones del pedido"));
 assert.equal(view.buttons.find(n=>n.props.children==="Confirmar entrega").props.icon,"availability");
 const css=readFileSync(new URL("styles/salon.css",root),"utf8");
 assert.match(css,/\.salon-order-detail-grand:only-child\{[^}]*grid-column:1\/-1[^}]*justify-content:space-between/);
});

test("orders: textos españoles en UTF-8 estricto y NFC, sin tildes corruptas",()=>{
 const bytes=readFileSync(new URL("orders/presentation/orders-manager.tsx",root));
 const source=new TextDecoder("utf-8",{fatal:true}).decode(bytes);
 assert.equal(source,source.normalize("NFC"));
 assert.doesNotMatch(source,/Ã|Â|\uFFFD/);
 for(const text of ["Dirección no registrada","Sin ítems cargados aún."])assert.ok(source.includes(text));
 assert.ok(mount("orders",{customerName:"José Muñoz",tableName:"Mesa 05"}).nodes.some(n=>n.props.children==="José Muñoz"));
});

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
