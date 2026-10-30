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
const channels=[{value:"delivery",label:"Delivery"},{value:"recojo",label:"Recojo en local"},{value:"mostrador",label:"Mostrador"},{value:"whatsapp",label:"WhatsApp"},{value:"salon",label:"Salón"}];

test("pedido manual: delivery exige cliente, teléfono válido, dirección y envío no negativo",()=>{
 assert.equal(schema.manualOrderSchema.safeParse(fields).success,true);
 for(const patch of [{customerName:" "},{customerPhone:""},{customerPhone:"abc999123456"},{customerPhone:"123"},{customerPhone:"1234567890123456"},{address:" "},{deliveryFee:""},{deliveryFee:"-1"},{deliveryFee:"NaN"},{channel:"whatsapp"},{channel:"salon"}])assert.equal(schema.manualOrderSchema.safeParse({...fields,...patch}).success,false,JSON.stringify(patch));
 assert.equal(schema.manualOrderSchema.safeParse({...fields,customerPhone:"999123456",deliveryFee:"0"}).success,true);
});
test("pedido manual: recojo en local reemplaza mostrador sin pedir datos de delivery",()=>{
 assert.equal(schema.isManualOrderChannel("mostrador"),false);
 assert.equal(schema.manualOrderSchema.safeParse({...fields,channel:"mostrador"}).success,false);
 const pickup={...fields,channel:"recojo",customerName:"",customerPhone:"",address:"",deliveryFee:""};
 assert.equal(schema.manualOrderSchema.safeParse(pickup).success,true);
 const payload=schema.manualOrderPayload({...pickup,lines:[line]});
 assert.equal(payload.channel,"recojo");assert.equal(payload.deliveryFee,0);assert.equal(payload.address,"");
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
const ds=Object.fromEntries(["Button","ChoiceGroup","ConfirmDialog","Tabs","Dialog","ModalHeader","ModalFooter","FormField","FormSection","Icon","IconButton","Input","LocationMap","MoneyInput","NumberStepper","Select","Textarea"].map(name=>[name,name]));
function hooksMount(file,props,overrides={}){
 const state=[],refs=[];let cursor=0,refCursor=0;
 const compiled=compile(file,name=>{
  if(name.endsWith(".css"))return{};
  if(name==="react")return{...require("react"),useEffect:()=>{},useRef:value=>{const index=refCursor++;return refs[index]??(refs[index]={current:value})},useState:value=>{const index=cursor++;if(state[index]===undefined)state[index]=value;return[state[index],next=>{state[index]=typeof next==="function"?next(state[index]):next}]}};
  if(name==="@/design-system")return ds;
  if(name==="@/design-system/location-map")return{LocationMap:"LocationMap"};
  if(name==="@/providers")return{useMoney:()=>value=>`S/ ${Number(value).toFixed(2)}`};
  if(name==="@/providers/feedback-provider")return{useFeedback:()=>({notify:()=>{}})};
  if(name==="react-hook-form")return{useWatch:({name})=>fields[name],useForm:()=>({register:name=>({name}),setValue:(name,value)=>{fields[name]=value},control:{},formState:{errors:{},isDirty:false,isSubmitting:false},handleSubmit:callback=>async()=>{const result=schema.manualOrderSchema.safeParse(fields);if(result.success)return callback(result.data)}})};
  if(name.endsWith("manual-order-schema"))return schema;
  if(name.endsWith("comanda-catalog"))return{ComandaCatalog:"ComandaCatalog",ComboConfigurator:"ComboConfigurator"};
  if(name==="./comanda-view")return{ComandaView:"ComandaView"};
  if(name==="./order-meta")return{channelIcons:{delivery:"truck",recojo:"box",mostrador:"store"}};
  return require(name);
 },{crypto:{randomUUID:()=>"temporary-key"},window:{addEventListener:()=>{},removeEventListener:()=>{}}});
 const Component=Object.values(compiled)[0];
 return{state,render:()=>{cursor=0;refCursor=0;return nodes(Component({...props,...overrides}))}};
}
const mount=(overrides={})=>hooksMount("orders/presentation/manual-order-dialog.tsx",{channels,currencySymbol:"S/",busy:false,onSave:async()=>{},onClose:()=>{},quoteDelivery:async()=>({covered:true,distanceKm:"2.35",fee:"6",maxDistanceKm:"5.00",currencySymbol:"S/"})},overrides);
const mountComanda=(overrides={})=>hooksMount("orders/presentation/comanda-view.tsx",{initial:{...fields,tableId:"",lines:[]},mode:"create",allTables:[],busy:false,currencySymbol:"S/",channelLabel:"Delivery",close:()=>{},save:async()=>{},notify:()=>{}},overrides);
const goToOrder=async view=>{await view.render().find(node=>node.type==="form").props.onSubmit();return view.render().find(node=>node.type==="ComandaView")};

test("pedido manual: datos primero, misma comanda, volver conserva productos y notas sin guardar",async()=>{
 let calls=0;const view=mount({onSave:async()=>calls++});let rendered=view.render();
 assert.equal(rendered.find(node=>node.type==="Dialog").props["aria-modal"],"true");
 assert.deepEqual(rendered.filter(node=>node.type==="FormField").map(node=>node.props.label),["Nombre del cliente","Teléfono","Referencia","Costo de envío"]);
 assert.equal(rendered.find(node=>node.type==="LocationMap").props.addressLabel,"Dirección de entrega");
 assert.ok(rendered.some(node=>node.type==="MoneyInput"&&node.props.symbol==="S/"));
 const channelTabs=rendered.find(node=>node.type==="Tabs");assert.equal(channelTabs.props.label,"Tipo de atención");assert.equal(channelTabs.props.value,"delivery");
 assert.deepEqual(channelTabs.props.items.map(item=>item.label),["Delivery","Recojo en local"]);assert.ok(channelTabs.props.items.every(item=>item.icon));
 assert.equal(rendered.some(node=>node.type==="ChoiceGroup"||node.type==="FormSection"),false,"sin radios propios ni títulos de sección");
 assert.equal(rendered.find(node=>node.type==="LocationMap").props.hint,false);
 assert.equal(rendered.some(node=>node.type==="ComandaCatalog"),false);
 let comanda=await goToOrder(view);assert.equal(calls,0);assert.equal(comanda.props.initial.tableId,"");assert.equal(comanda.props.channelLabel,"Delivery");
 comanda.props.onBack({...comanda.props.initial,lines:[line],notes:"Entregar en recepción"});
 fields.deliveryFee="7";comanda=await goToOrder(view);
 assert.equal(comanda.props.initial.lines[0].note,"Sin cebolla");assert.equal(comanda.props.initial.notes,"Entregar en recepción");assert.equal(comanda.props.initial.deliveryFee,"7");assert.equal(calls,0);
 await comanda.props.save(comanda.props.initial,true);assert.equal(calls,1);fields.deliveryFee="5";
});
const flush=()=>new Promise(resolve=>setImmediate(resolve));
test("pedido manual: el punto del mapa sugiere el envío, viaja en el alta y respeta la cobertura",async()=>{
 const sent=[];let saved;const view=mount({quoteDelivery:async point=>{sent.push(point);return{covered:true,distanceKm:"2.35",fee:"6",maxDistanceKm:"5.00",currencySymbol:"S/"}},onSave:async draft=>{saved=draft}});
 await view.render().find(node=>node.type==="LocationMap").props.onChange({lat:-12.1219,lng:-77.0297});await flush();
 assert.deepEqual(JSON.parse(JSON.stringify(sent)),[{latitude:-12.1219,longitude:-77.0297}]);assert.equal(fields.deliveryLatitude,"-12.121900");assert.equal(fields.deliveryFee,"6.00");
 assert.equal(view.render().find(node=>node.type==="FormField"&&node.props.label==="Costo de envío").props.help,"Sugerido para 2.35 km");
 const comanda=await goToOrder(view);await comanda.props.save({...comanda.props.initial,lines:[line]});
 assert.equal(saved.deliveryLatitude,"-12.121900");
 const payload=schema.manualOrderPayload(saved);assert.equal(payload.deliveryLatitude,-12.1219);assert.equal(payload.deliveryLongitude,-77.0297);
 assert.equal("deliveryLatitude" in schema.manualOrderPayload({...saved,channel:"recojo"}),false,"recojo no envía punto");
 const far=mount({quoteDelivery:async()=>({covered:false,distanceKm:"9.10",fee:"0.00",maxDistanceKm:"5.00",currencySymbol:"S/"})});
 await far.render().find(node=>node.type==="LocationMap").props.onChange({lat:-12.2,lng:-77.1});await flush();
 assert.equal(fields.deliveryFee,"6.00","fuera de cobertura no cambia el costo escrito");
 assert.equal(far.render().find(node=>node.type==="FormField"&&node.props.label==="Costo de envío").props.help,"Fuera de cobertura (hasta 5.00 km)");
 const manual=mount({quoteDelivery:async()=>{throw Object.assign(new Error("Sin tarifas"),{code:"delivery_unavailable"})}});
 await manual.render().find(node=>node.type==="LocationMap").props.onChange({lat:-12.2,lng:-77.1});await flush();
 assert.equal(manual.render().find(node=>node.type==="FormField"&&node.props.label==="Costo de envío").props.help,undefined,"sin tarifas el costo sigue manual sin avisos");
 Object.assign(fields,{deliveryLatitude:"",deliveryLongitude:"",deliveryFee:"5"});
});
test("pedido manual: datos inválidos impiden continuar y un fallo conserva la comanda",async()=>{
 const original=fields.address;fields.address="";const invalid=mount();await goToOrder(invalid);assert.equal(invalid.state[0],"details");fields.address=original;
 const view=mount({onSave:async()=>{throw new Error("Servidor no disponible")}});
 const comanda=await goToOrder(view);await assert.rejects(comanda.props.save({...comanda.props.initial,lines:[line]}),/Servidor no disponible/);
 assert.equal(view.state[0],"order");assert.equal(view.state[1].lines.length,1);
});
test("pedido manual: bloqueo y descarte explícito protegen datos e ítems",async()=>{
 const busy=mount({busy:true});const rendered=busy.render();assert.equal(rendered.find(node=>node.type==="ModalHeader").props.busy,true);assert.equal(rendered.find(node=>node.type==="fieldset").props.disabled,true);
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
 let rendered=view.render();assert.equal(rendered.some(node=>node.type==="Select"),false);assert.equal(rendered.some(node=>node.props.children==="Personas"),false);assert.equal(rendered.some(node=>node.props.children==="Guardar comanda"),false);
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
 let rendered=view.render();assert.ok(rendered.some(node=>node.type==="Select"));assert.ok(rendered.some(node=>node.props.children==="Guardar comanda"));
 await rendered.find(node=>node.type==="Button"&&node.props.className==="salon-comanda-submit").props.onClick();assert.equal(saved,0);assert.equal(alerts.length,0);
 assert.ok(view.render().some(node=>node.props.role==="alert"&&node.props.children==="Selecciona la mesa del pedido."));
 const delivery=mountComanda({onBack:draft=>{returned=draft}});
 rendered=delivery.render();rendered.find(node=>node.type==="ComandaCatalog").props.onConfigureCombo("combo-product");rendered=delivery.render();
 const choices=[{groupId:"group-1",groupName:"Segundo",productId:"option-1",name:"Lomo",surcharge:3}];
 rendered.find(node=>node.type==="ComboConfigurator").props.onConfirm({productId:"combo-product",name:"Menú del día",unitPrice:23,selections:choices});
 rendered=delivery.render();nodes(rendered.find(node=>node.type==="ModalHeader").props.details).find(node=>node.type==="Button"&&node.props.children==="Editar datos del pedido").props.onClick();
 assert.equal(returned.lines[0].itemType,"combo");assert.equal(returned.lines[0].selections,choices);
});
test("comanda: disminuir conserva notas y combos; el pie abre revisión sin enviar",()=>{
 let saves=0;const initial={...fields,tableId:"",lines:[{...line,lineKey:"first",productId:"p",itemType:"product",qty:2,note:"Primera nota"},{...line,lineKey:"second",productId:"p",itemType:"product",qty:1,note:"Segunda nota"},{...line,lineKey:"combo",productId:"combo",itemType:"combo",qty:2}]};
 const view=mountComanda({initial,save:()=>saves++});
 view.render().find(node=>node.type==="ComandaCatalog").props.onDecrease("p");assert.equal(view.state[0].lines.length,2);assert.equal(view.state[0].lines[0].note,"Primera nota");assert.equal(view.state[0].lines[1].qty,2);
 view.render().find(node=>node.type==="ComandaCatalog").props.onDecrease("p");assert.equal(view.state[0].lines[0].qty,1);
 const reviewButton=view.render().find(node=>node.type==="Button"&&node.props.className==="cmd-bar-submit");assert.equal(reviewButton.props.icon,"shoppingBag");reviewButton.props.onClick();assert.equal(view.state[1],true);assert.equal(saves,0);
 const busy=mountComanda({initial,busy:true});busy.render().find(node=>node.type==="ComandaCatalog").props.onDecrease("p");assert.equal(busy.state[0].lines.length,3);
});

test("comanda: los clics rápidos del mismo render no sobrepasan las existencias",()=>{
 const view=mountComanda();
 const pick=view.render().find(node=>node.type==="ComandaCatalog").props.onPick;
 const product={id:"p",name:"Plato",price:"10",remaining:3};
 for(let index=0;index<100;index++)pick(product);
 assert.equal(view.state[0].lines[0].qty,3);
 const stepper=view.render().find(node=>node.type==="NumberStepper");
 assert.equal(stepper.props.max,3);
});

test("comanda: dos líneas del mismo producto comparten tope incluso con callbacks anteriores",()=>{
 const initial={...fields,tableId:"",lines:[{...line,lineKey:"one",productId:"p",qty:1},{...line,lineKey:"two",productId:"p",qty:1}]};
 const view=mountComanda({initial});
 view.render().find(node=>node.type==="ComandaCatalog").props.onAvailability([{id:"p",remaining:3}]);
 const steppers=view.render().filter(node=>node.type==="NumberStepper");
 assert.deepEqual(steppers.map(node=>node.props.max),[2,2]);
 steppers[0].props.onChange(2);steppers[1].props.onChange(2);
 assert.equal(view.state[0].lines.reduce((sum,item)=>sum+item.qty,0),3);
 assert.deepEqual(view.render().filter(node=>node.type==="NumberStepper").map(node=>node.props.max),[2,1]);
 view.render().filter(node=>node.type==="NumberStepper")[0].props.onChange(1);
 assert.deepEqual(view.render().filter(node=>node.type==="NumberStepper").map(node=>node.props.max),[2,2]);
});

test("comanda: editar suma las unidades ya reservadas y no redondea existencias hacia arriba",()=>{
 const initial={...fields,tableId:"",lines:[{...line,productId:"p",qty:4,sourceItemId:"saved-item"}]};
 const view=mountComanda({initial,mode:"edit"});
 const catalog=view.render().find(node=>node.type==="ComandaCatalog");
 assert.equal(catalog.props.reservedByProduct.p,4);
 const pick=catalog.props.onPick;
 for(let index=0;index<20;index++)pick({id:"p",name:"Plato",price:"10",remaining:2.8});
 assert.equal(view.state[0].lines[0].qty,6);
 assert.equal(view.render().find(node=>node.type==="NumberStepper").props.max,6);
 const fractional=mountComanda();
 fractional.render().find(node=>node.type==="ComandaCatalog").props.onPick({id:"p",name:"Plato",price:"10",remaining:.5});
 assert.equal(fractional.state[0].lines.length,0);
});

test("comanda: los productos sin control no heredan un tope anterior",()=>{
 const view=mountComanda();
 let catalog=view.render().find(node=>node.type==="ComandaCatalog");
 catalog.props.onAvailability([{id:"p",remaining:2}]);
 catalog=view.render().find(node=>node.type==="ComandaCatalog");
 catalog.props.onAvailability([{id:"p",remaining:null}]);
 const pick=view.render().find(node=>node.type==="ComandaCatalog").props.onPick;
 for(let index=0;index<5;index++)pick({id:"p",name:"Plato",price:"10",remaining:null});
 assert.equal(view.state[0].lines[0].qty,5);
 assert.equal(view.render().find(node=>node.type==="NumberStepper").props.max,undefined);
});
test("comanda: conserva la foto real en el resumen con cantidades suaves y fallback sin imagen",()=>{
 const view=mountComanda();view.render().find(node=>node.type==="ComandaCatalog").props.onPick({id:"p",name:"Plato",price:"18",imageUrl:"/foto-del-plato.jpg"});
 const rendered=view.render();assert.equal(view.state[0].lines[0].imageUrl,"/foto-del-plato.jpg");
 const image=rendered.find(node=>node.type==="img");assert.equal(image.props.src,"/foto-del-plato.jpg");assert.equal(image.props.alt,"");
 const target={hidden:false};image.props.onError({currentTarget:target});assert.equal(target.hidden,true);
 assert.ok(rendered.some(node=>node.type==="NumberStepper"&&node.props.appearance==="soft"));
 const historic=mountComanda({initial:{...fields,tableId:"",lines:[line]}}).render();assert.equal(historic.some(node=>node.type==="img"),false);assert.ok(historic.some(node=>node.props.className==="cmd-line-visual"));
 const payload=schema.manualOrderPayload({...fields,lines:view.state[0].lines});assert.equal("imageUrl" in payload.items[0],false);
});
test("comanda: borrador y envío conservan su intención y el progreso corresponde a la acción elegida",async()=>{
 for(const sendToKitchen of [false,true]){
  const calls=[];let finish;
  const view=mountComanda({initial:{...fields,channel:"salon",tableId:"table-1",lines:[line]},save:(draft,send)=>{calls.push({draft,send});return new Promise(resolve=>{finish=resolve})}});
  const buttonClass=sendToKitchen?"salon-comanda-submit":"salon-comanda-draft";
  const initialButton=view.render().find(node=>node.props.className===buttonClass);
  assert.equal(initialButton.props.icon,sendToKitchen?"send":"orders");assert.equal(initialButton.props.children,sendToKitchen?"Enviar a cocina":"Guardar comanda");
  const pending=initialButton.props.onClick();assert.equal(calls.length,1);assert.equal(calls[0].send,sendToKitchen);
  const rendered=view.render();const draft=rendered.find(node=>node.props.className==="salon-comanda-draft"),send=rendered.find(node=>node.props.className==="salon-comanda-submit");
  assert.equal(draft.props.disabled,true);assert.equal(send.props.disabled,true);
  if(!sendToKitchen){assert.equal(draft.props.children,"Guardando…");assert.equal(draft.props.loading,true);assert.equal(send.props.children,"Enviar a cocina");assert.equal(send.props.loading,false);}
  else{assert.equal(send.props.children,"Enviando…");assert.equal(send.props.loading,true);assert.equal(draft.props.children,"Guardar comanda");assert.equal(draft.props.loading,false);}
  send.props.onClick();draft.props.onClick();assert.equal(calls.length,1);
  finish();await pending;assert.equal(view.render().find(node=>node.props.className===buttonClass).props.disabled,false);
 }
 let finishEdit;const edit=mountComanda({mode:"edit",initial:{...fields,channel:"salon",tableId:"table-1",lines:[line]},save:()=>new Promise(resolve=>{finishEdit=resolve})});
 const editPending=edit.render().find(node=>node.props.className==="salon-comanda-submit").props.onClick();
 const editButton=edit.render().find(node=>node.props.className==="salon-comanda-submit");assert.equal(editButton.props.children,"Guardando…");assert.equal(editButton.props.loading,true);assert.equal(edit.render().some(node=>node.props.className==="salon-comanda-draft"),false);finishEdit();await editPending;
 const source=readFileSync(new URL("salon/presentation/salon-manager.tsx",root),"utf8");assert.match(source,/title:variables.sendToKitchen\?"Comanda enviada":"Comanda guardada"/);assert.ok(source.includes("guardado, sin enviar a cocina"));
});
test("comanda: plegar la hoja vuelve a la carta sin guardar, cerrar ni perder productos y notas",()=>{
 let saves=0,closes=0;const initial={...fields,tableId:"",lines:[line],notes:"Servir juntos"};
 const view=mountComanda({initial,save:()=>saves++,close:()=>closes++});
 view.render().find(node=>node.props.className==="cmd-bar-submit").props.onClick();assert.equal(view.state[1],true);
 const control=view.render().find(node=>node.props.className==="cmd-sheet-close");
 assert.equal(control.props["aria-label"],"Volver a la carta");assert.equal(control.props.title,"Volver a la carta");
 assert.equal(control.props.children.props.name,"chevron");assert.equal(control.props.children.props.size,24);
 control.props.onClick();assert.equal(view.state[1],false);assert.equal(view.state[0],initial);assert.equal(saves,0);assert.equal(closes,0);
 const css=postcss.parse(readFileSync(new URL("styles/salon-comanda.css",root),"utf8"));
 const rule=css.nodes.find(node=>node.type==="atrule"&&node.params==="(width<=900px)").nodes.find(node=>node.selector===".cmd-sheet-close");
 const declarations=Object.fromEntries(rule.nodes.filter(node=>node.type==="decl").map(node=>[node.prop,node.value]));
 assert.equal(declarations.width,"var(--size-44)");assert.equal(declarations.height,"var(--size-44)");assert.equal(declarations.background,"var(--primary-100)");assert.equal(declarations.border,"0");assert.equal(declarations.color,"var(--primary-700)");assert.equal(declarations["border-radius"],"var(--radius-pill)");assert.equal(declarations["touch-action"],"manipulation");
});
test("comanda móvil: el scrim no cubre la cabecera y las acciones tienen radio completo",()=>{
 const css=postcss.parse(readFileSync(new URL("styles/salon-comanda.css",root),"utf8"));
 const mobile=css.nodes.find(node=>node.type==="atrule"&&node.params==="(width<=900px)");
 const declarations=selector=>Object.fromEntries(mobile.nodes.find(node=>node.selector===selector).nodes.filter(node=>node.type==="decl").map(node=>[node.prop,node.value]));
 const header=declarations(".cmd-shell>.modal-header"),scrim=declarations(".cmd-scrim");
 assert.equal(header.position,"relative");assert.equal(header["z-index"],"var(--z-header)");assert.equal(scrim["z-index"],"var(--z-sticky)");
 assert.equal(declarations(".cmd-summary-foot .cmd-actions>.button")["border-radius"],"var(--radius-pill)");
 assert.equal(declarations(".cmd-bar>.cmd-bar-submit.button")["border-radius"],"var(--radius-pill)");assert.equal(declarations(".cmd-bar>.cmd-bar-submit.button")["min-height"],"var(--size-48)");
 assert.equal(declarations(".cmd-sheet-close>svg").transform,"rotate(90deg)");
});
test("pedido manual: CSS único, responsive, estados remotos y actualización del flujo",()=>{
 const source=readFileSync(new URL("orders/presentation/orders-manager.tsx",root),"utf8");assert.match(source,/canManage\?<Button icon="plus"/);assert.ok(source.includes("createManualOrder"));assert.ok(source.includes('"kitchen-tickets"'));assert.ok(source.includes('setChannel(order.channel)'));assert.ok(source.includes('setStatus("abiertos")'));
 const css=postcss.parse(readFileSync(new URL("orders/presentation/manual-order.css",root),"utf8"));const seen=new Set();css.walkRules(rule=>{let context="";for(let parent=rule.parent;parent;parent=parent.parent)if(parent.type==="atrule")context+=parent.name+parent.params;for(const selector of rule.selectors){const key=context+selector;assert.equal(seen.has(key),false,key);seen.add(key)}});
 const text=css.toString();assert.ok(text.includes("100dvh"));assert.ok(text.includes("overflow-y:auto"));assert.ok(text.includes("overflow-x:hidden"));assert.ok(text.includes("@media(max-width:600px)"));
 const catalog=readFileSync(new URL("salon/presentation/comanda-catalog.tsx",root),"utf8");assert.ok(catalog.includes("MenuRows"));assert.ok(catalog.includes("salonCatalog.refetch()"));assert.ok(catalog.includes("comboCatalog.refetch()"));
});
