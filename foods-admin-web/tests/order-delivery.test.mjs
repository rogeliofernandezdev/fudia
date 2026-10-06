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
const order={id:"pedido-1",code:"PED-001",channel:"salon",status:"listo",paymentStatus:"pending",customerName:"",customerPhone:"",tableName:"Mesa 1",tableId:"mesa-1",total:"25.00",subtotal:"25.00",deliveryFee:"0",paidAmount:"0",remainingAmount:"25.00",createdAt:"2026-10-05T10:00:00Z",updatedAt:"2026-10-05T10:00:00Z",items:[]};

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
 return{calls,nodes,buttons:nodes.filter(n=>n.type==="Button"),links:nodes.filter(n=>n.type==="Link")};
}

test("salon: la cabecera muestra un solo estado del pedido sin duplicar el pago",()=>{
 for(const changes of [{},{paymentStatus:"paid",paidAmount:"25.00",remainingAmount:"0"},{status:"entregado"}]){
  const view=mount("salon",changes);
  const statuses=view.nodes.filter(n=>n.type==="Status");
  assert.equal(statuses.length,1);
  assert.equal(statuses[0].props.children,changes.status==="entregado"?"Entregado":"Listo para entregar");
  assert.ok(view.nodes.some(n=>n.props.className==="salon-order-detail-subtotal"));
  assert.ok(view.nodes.some(n=>n.props.className==="salon-order-detail-grand"));
 }
 assert.ok(mount("salon").links.some(n=>n.props.href==="/pos?orderId=pedido-1"));
});

test("salon: la franja decorativa conserva el degradado de la paleta",()=>{
 const css=readFileSync(new URL("styles/salon.css",root),"utf8");
 assert.match(css,/\.salon-order-detail-accent\{[^}]*background:linear-gradient\(90deg,var\(--brand-600\),var\(--primary-600\)\)/);
 const accent=mount("salon").nodes.find(n=>n.props.className==="salon-order-detail-accent");
 assert.equal(accent.props["aria-hidden"],"true");
});

test("salon: cobrar es la única acción primaria cuando hay saldo",()=>{
 const view=mount("salon");
 const delivery=view.buttons.find(n=>n.props.children==="Confirmar entrega");
 assert.equal(delivery.props.kind,"secondary");
 assert.equal(delivery.props.icon,"availability");
 assert.equal(view.buttons.some(n=>n.props.className==="order-detail-primary"),false);
 assert.match(view.links[0].props.className,/\bprimary\b/);
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
 assert.ok(content.props.children.some(n=>n.props.className.startsWith("salon-order-detail-totals")));
 assert.equal(view.nodes.find(n=>n.props.id==="salon-order-account-title").props.children,"Resumen de cuenta");
 assert.equal(view.nodes.some(n=>n.props.children==="DETALLE"),false);
 const css=readFileSync(new URL("styles/salon.css",root),"utf8");
 assert.match(css,/\.salon-order-detail-content\{[^}]*flex-direction:column/);
 assert.match(css,/\.salon-order-detail-totals\{[^}]*grid-template-columns:repeat\(2,minmax\(0,1fr\)\) minmax\(0,1.25fr\)/);
 assert.match(css,/\.salon-order-detail-grand\{[^}]*align-items:flex-end/);
 assert.equal(view.nodes.find(n=>n.props.id==="salon-order-account-title").props.className,"sr-only");
 assert.match(css,/\.salon-order-detail\{[^}]*max-width:var\(--size-680\)/);
});

test("salon: cuenta compacta sin tarjetas y pie alineado con acciones",()=>{
 const css=readFileSync(new URL("styles/salon.css",root),"utf8");
 const balance=css.match(/\.salon-order-detail-grand\{([^}]+)\}/)[1];
 assert.doesNotMatch(balance,/(?:background|border-radius):/);
 assert.match(balance,/border-left:var\(--stroke-1\) solid var\(--line\)/);
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

test("salon: el cobro queda bloqueado mientras se registra la entrega",()=>{
 const link=mount("salon",{},true,true).links[0];
 assert.equal(link.props["aria-disabled"],true);
 assert.equal(link.props["aria-busy"],true);
 assert.equal(link.props.tabIndex,-1);
 let prevented=false;
 link.props.onClick({preventDefault:()=>{prevented=true}});
 assert.equal(prevented,true);
});

for(const section of ["salon","orders"]){
 test(`${section}: el detalle ofrece la acción real de entrega antes de cobrar`,()=>{
  const view=mount(section);
  const button=view.buttons.find(n=>n.props.children==="Confirmar entrega");
  assert.ok(button);
  assert.equal(button.props.disabled,false);
  button.props.onClick();
  assert.deepEqual(view.calls,["entregado"]);
 });
 test(`${section}: la entrega pendiente conserva el cobro y el cierre automático no ofrece acciones`,()=>{
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
