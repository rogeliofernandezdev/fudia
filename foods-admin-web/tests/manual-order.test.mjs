import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
import postcss from "postcss";
const require=createRequire(import.meta.url);
const root=new URL("../src/modules/operations/",import.meta.url);
function compile(path,resolve=require,context={}){const exports={};vm.runInNewContext(ts.transpileModule(readFileSync(new URL(path,root),"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve,...context});return exports}
const resolver=compile("../../shared/forms/zod-resolver.ts");
const schema=compile("orders/domain/manual-order-schema.ts",name=>name==="@/shared/forms/zod-resolver"?resolver:require(name));
const fields={...schema.emptyManualOrder,customerName:" Ana ",customerPhone:"+51 (999) 123-456",address:" Av. Perú 100 ",reference:"Puerta azul",deliveryFee:"5"};
const line={lineKey:"line-1",productId:"product-1",name:"Plato",itemType:"product",qty:2,unitPrice:25,note:"Sin cebolla",selections:[]};
const channels=[{value:"delivery",label:"Delivery"},{value:"recojo",label:"Recojo"},{value:"mostrador",label:"Mostrador"},{value:"whatsapp",label:"WhatsApp"},{value:"salon",label:"Salón"}];

test("pedido manual: delivery exige cliente, teléfono válido, dirección y envío no negativo",()=>{
 assert.equal(schema.manualOrderSchema.safeParse(fields).success,true);
 for(const patch of [{customerName:" "},{customerPhone:""},{customerPhone:"abc999123456"},{customerPhone:"123"},{customerPhone:"1234567890123456"},{address:" "},{deliveryFee:""},{deliveryFee:"-1"},{deliveryFee:"NaN"},{channel:"whatsapp"},{channel:"salon"}])assert.equal(schema.manualOrderSchema.safeParse({...fields,...patch}).success,false,JSON.stringify(patch));
 assert.equal(schema.manualOrderSchema.safeParse({...fields,customerPhone:"999123456",deliveryFee:"0"}).success,true);
});
test("pedido manual: payload no envía empresa, local, mesa ni precios controlados por cliente",()=>{
 const payload=schema.manualOrderPayload({...fields,lines:[line,{...line,lineKey:"combo-1",itemType:"combo",productId:"combo-product",selections:[{groupId:"group-1",productId:"option-1",name:"Elección",surcharge:3}]}]});
 assert.equal(payload.sendToKitchen,true);assert.equal(payload.deliveryFee,5);assert.equal(payload.customerName,"Ana");assert.equal(payload.address,"Av. Perú 100");
 for(const key of ["organizationId","locationId","tableId","lines"])assert.equal(key in payload,false);
 assert.equal(payload.items[0].qty,2);assert.equal(payload.items[0].note,"Sin cebolla");assert.equal("unitPrice" in payload.items[0],false);assert.equal("name" in payload.items[0],false);
 assert.deepEqual(JSON.parse(JSON.stringify(payload.items[1].selections)),[{groupId:"group-1",productId:"option-1"}]);
 assert.throws(()=>schema.manualOrderPayload({...fields,lines:[]}),/Agrega al menos/);
 const pickup=schema.manualOrderPayload({...fields,channel:"recojo",deliveryFee:"",lines:[line]});assert.equal(pickup.deliveryFee,0);assert.equal(pickup.address,"");assert.equal(pickup.reference,"");
});
test("pedido manual: integración usa el endpoint existente con POST y respuesta real",async()=>{
 let sent;
 const api=compile("orders/infrastructure/orders-api.ts",name=>name==="@/shared/api/client"?{apiFetch:async(path,options)=>{sent={path,options};return{id:"server-uuid",status:"confirmado"}}}:name==="../domain/manual-order-schema"?schema:require(name));
 const result=await api.createManualOrder({...fields,lines:[line]});assert.equal(result.id,"server-uuid");assert.equal(sent.path,"orders");assert.equal(sent.options.method,"POST");assert.equal(JSON.parse(sent.options.body).sendToKitchen,true);
});
function nodes(tree,result=[]){if(Array.isArray(tree)){tree.forEach(node=>nodes(node,result));return result}if(tree&&typeof tree==="object"){result.push(tree);nodes(tree.props?.children,result)}return result}
const ds=Object.fromEntries(["Button","ConfirmDialog","Dialog","FormField","Icon","IconButton","Input","Select","Textarea"].map(name=>[name,name]));
function hooksMount(file,props,overrides={}){
 const state=[],refs=[];let cursor=0,refCursor=0;
 const compiled=compile(file,name=>{
  if(name.endsWith(".css"))return{};
  if(name==="react")return{...require("react"),useEffect:()=>{},useRef:value=>{const index=refCursor++;return refs[index]??(refs[index]={current:value})},useState:value=>{const index=cursor++;if(state[index]===undefined)state[index]=value;return[state[index],next=>{state[index]=typeof next==="function"?next(state[index]):next}]}};
  if(name==="@/design-system")return ds;
  if(name==="@/providers/feedback-provider")return{useFeedback:()=>({notify:()=>{}})};
  if(name==="react-hook-form")return{useWatch:({name})=>fields[name],useForm:()=>({register:name=>({name}),control:{},formState:{errors:{},isDirty:false,isSubmitting:false},handleSubmit:callback=>async()=>{const result=schema.manualOrderSchema.safeParse(fields);if(result.success)return callback(result.data)}})};
  if(name.endsWith("manual-order-schema"))return schema;
  if(name.endsWith("comanda-catalog"))return{ComandaCatalog:"ComandaCatalog",ComboConfigurator:"ComboConfigurator"};
  if(name==="./comanda-view")return{ComandaView:"ComandaView"};
  return require(name);
 },{crypto:{randomUUID:()=>"temporary-key"},window:{addEventListener:()=>{},removeEventListener:()=>{}}});
 const Component=Object.values(compiled)[0];
 return{state,render:()=>{cursor=0;refCursor=0;return nodes(Component({...props,...overrides}))}};
}
const mount=(overrides={})=>hooksMount("orders/presentation/manual-order-dialog.tsx",{channels,currencySymbol:"S/",busy:false,onSave:async()=>{},onClose:()=>{}},overrides);
const mountComanda=(overrides={})=>hooksMount("orders/presentation/comanda-view.tsx",{initial:{...fields,tableId:"",lines:[]},mode:"create",allTables:[],busy:false,currencySymbol:"S/",channelLabel:"Delivery",close:()=>{},save:async()=>{},notify:()=>{}},overrides);
const goToOrder=async view=>{await view.render().find(node=>node.type==="form").props.onSubmit();return view.render().find(node=>node.type==="ComandaView")};

test("pedido manual: datos primero, misma comanda, volver conserva productos y notas sin guardar",async()=>{
 let calls=0;const view=mount({onSave:async()=>calls++});let rendered=view.render();
 assert.equal(rendered.find(node=>node.type==="Dialog").props["aria-modal"],"true");
 assert.deepEqual(rendered.filter(node=>node.type==="FormField").map(node=>node.props.label),["Tipo de atención","Cliente","Teléfono","Dirección de entrega","Referencia de entrega","Costo de envío"]);
 assert.deepEqual(rendered.filter(node=>node.type==="option").map(node=>node.props.children),["Delivery","Recojo","Mostrador"]);
 assert.equal(rendered.some(node=>node.type==="ComandaCatalog"),false);
 let comanda=await goToOrder(view);assert.equal(calls,0);assert.equal(comanda.props.initial.tableId,"");assert.equal(comanda.props.channelLabel,"Delivery");
 comanda.props.onBack({...comanda.props.initial,lines:[line],notes:"Entregar en recepción"});
 fields.deliveryFee="7";comanda=await goToOrder(view);
 assert.equal(comanda.props.initial.lines[0].note,"Sin cebolla");assert.equal(comanda.props.initial.notes,"Entregar en recepción");assert.equal(comanda.props.initial.deliveryFee,"7");assert.equal(calls,0);
 await comanda.props.save(comanda.props.initial,true);assert.equal(calls,1);fields.deliveryFee="5";
});
test("pedido manual: datos inválidos impiden continuar y un fallo conserva la comanda",async()=>{
 const original=fields.address;fields.address="";const invalid=mount();await goToOrder(invalid);assert.equal(invalid.state[0],"details");fields.address=original;
 const view=mount({onSave:async()=>{throw new Error("Servidor no disponible")}});
 const comanda=await goToOrder(view);await assert.rejects(comanda.props.save({...comanda.props.initial,lines:[line]}),/Servidor no disponible/);
 assert.equal(view.state[0],"order");assert.equal(view.state[1].lines.length,1);
});
test("pedido manual: bloqueo y descarte explícito protegen datos e ítems",async()=>{
 const busy=mount({busy:true});const rendered=busy.render();assert.equal(rendered.find(node=>node.type==="IconButton").props.disabled,true);assert.equal(rendered.find(node=>node.type==="fieldset").props.disabled,true);
 let calls=0,finish;const view=mount({onSave:()=>{calls++;return new Promise(resolve=>{finish=resolve})}});const comanda=await goToOrder(view);
 const pending=comanda.props.save({...comanda.props.initial,lines:[line]});await comanda.props.save({...comanda.props.initial,lines:[line]});assert.equal(calls,1);finish();await pending;
 let closed=0;const dirty=mount({onClose:()=>closed++});const order=await goToOrder(dirty);order.props.close({...order.props.initial,lines:[line]});assert.equal(closed,0);assert.equal(dirty.render().find(node=>node.type==="ConfirmDialog").props.open,true);
 dirty.render().find(node=>node.type==="ConfirmDialog").props.onConfirm();assert.equal(closed,1);
});

test("el cierre por respuesta del API no abre confirmación de descarte en el wizard",async()=>{
 let closed=0;
 const view=mount({onClose:()=>closed++});
 const comanda=await goToOrder(view);
 comanda.props.onResponseClose();
 assert.equal(closed,1);assert.equal(view.state[2],false);
});
test("comanda compartida: delivery no pide mesa, incluye envío y se guarda una sola vez",async()=>{
 let calls=0,finish,saved;const view=mountComanda({save:(draft,send)=>{saved={draft,send};calls++;return new Promise(resolve=>{finish=resolve})}});
 let rendered=view.render();assert.equal(rendered.some(node=>node.type==="Select"),false);assert.equal(rendered.some(node=>node.props.children==="Personas"),false);assert.equal(rendered.some(node=>node.props.children==="Guardar borrador"),false);
 rendered.find(node=>node.type==="ComandaCatalog").props.onPick({id:"product-1",name:"Plato",price:"25"});rendered=view.render();
 assert.ok(rendered.some(node=>node.type==="strong"&&JSON.stringify(node.props.children).includes("30.00")));
 const submit=rendered.find(node=>node.type==="Button"&&node.props.className==="salon-comanda-submit");
 const pending=submit.props.onClick();submit.props.onClick();assert.equal(calls,1);assert.equal(saved.draft.tableId,"");assert.equal(saved.send,true);
 rendered.find(node=>node.type==="ComandaCatalog").props.onPick({id:"product-2",name:"Otro",price:"10"});assert.equal(view.state[0].lines.length,1);
 finish();await pending;
});
test("comanda compartida: Salón mantiene mesa obligatoria y borrador; volver incluye elecciones",async()=>{
 const alerts=[];let saved=0,returned;
 const view=mountComanda({initial:{...fields,channel:"salon",tableId:"",lines:[line]},notify:value=>alerts.push(value),save:()=>saved++});
 let rendered=view.render();assert.ok(rendered.some(node=>node.type==="Select"));assert.ok(rendered.some(node=>node.props.children==="Guardar borrador"));
 await rendered.find(node=>node.type==="Button"&&node.props.className==="salon-comanda-submit").props.onClick();assert.equal(saved,0);assert.equal(alerts.length,0);
 assert.ok(view.render().some(node=>node.props.role==="alert"&&node.props.children==="Selecciona la mesa del pedido."));
 const delivery=mountComanda({onBack:draft=>{returned=draft}});
 rendered=delivery.render();rendered.find(node=>node.type==="ComandaCatalog").props.onConfigureCombo("combo-product");rendered=delivery.render();
 const choices=[{groupId:"group-1",groupName:"Segundo",productId:"option-1",name:"Lomo",surcharge:3}];
 rendered.find(node=>node.type==="ComboConfigurator").props.onConfirm({productId:"combo-product",name:"Menú del día",unitPrice:23,selections:choices});
 rendered=delivery.render();rendered.find(node=>node.props["aria-label"]==="Editar datos del pedido").props.onClick();
 assert.equal(returned.lines[0].itemType,"combo");assert.equal(returned.lines[0].selections,choices);
});
test("pedido manual: CSS único, responsive, estados remotos y actualización del flujo",()=>{
 const source=readFileSync(new URL("orders/presentation/orders-manager.tsx",root),"utf8");assert.match(source,/canManage\?<Button icon="plus"/);assert.ok(source.includes("createManualOrder"));assert.ok(source.includes('"kitchen-tickets"'));assert.ok(source.includes('setChannel(order.channel)'));assert.ok(source.includes('setStatus("abiertos")'));
 const css=postcss.parse(readFileSync(new URL("orders/presentation/manual-order.css",root),"utf8"));const seen=new Set();css.walkRules(rule=>{let context="";for(let parent=rule.parent;parent;parent=parent.parent)if(parent.type==="atrule")context+=parent.name+parent.params;for(const selector of rule.selectors){const key=context+selector;assert.equal(seen.has(key),false,key);seen.add(key)}});
 const text=css.toString();assert.ok(text.includes("100dvh"));assert.ok(text.includes("overflow-y:auto"));assert.ok(text.includes("overflow-x:hidden"));assert.ok(text.includes("@media(max-width:600px)"));
 const catalog=readFileSync(new URL("salon/presentation/comanda-catalog.tsx",root),"utf8");assert.ok(catalog.includes("MenuRows"));assert.ok(catalog.includes("salonCatalog.refetch()"));assert.ok(catalog.includes("comboCatalog.refetch()"));
});
