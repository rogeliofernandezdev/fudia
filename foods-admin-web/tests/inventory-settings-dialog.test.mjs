import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
import {createFormControl} from "react-hook-form";
import postcss from "postcss";

const require=createRequire(import.meta.url);
const root=new URL("../src/",import.meta.url);
function compile(path,resolve){
 const exports={};
 vm.runInNewContext(ts.transpileModule(readFileSync(new URL(path,root),"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve,URLSearchParams});
 return exports;
}
const resolver=compile("shared/forms/zod-resolver.ts",require);
const schema=compile("modules/supply/inventory/domain/inventory-settings-schema.ts",name=>name==="@/shared/forms/zod-resolver"?resolver:require(name));
const item={inventoryItemId:"water",productId:null,productName:null,name:"Agua mineral San Luis",unit:"botella",unitName:"Botella",unitCountable:true,active:true,quantity:"0",minimumStock:"2.000",reorderPoint:"2.000",optimalStock:"2.000",hasHistory:false,recipes:[],otherLocations:[],presentations:[],canDelete:false,canDeactivate:true,deactivateBlockers:[]};
const units=[{id:"1",code:"botella",name:"Botella",countable:true},{id:"2",code:"und",name:"Unidad",countable:true},{id:"3",code:"kg",name:"Kilogramo",countable:false}];
const tick=()=>new Promise(resolve=>setImmediate(resolve));
function mount({busy=false,save=async()=>{},close=()=>{},record=item,preview=null,remove=()=>{},removePresentation=()=>{}}={}){
 let control;const submitting={current:false};const previews=[];
 const {InventorySettingsDialog}=compile("modules/supply/inventory/presentation/inventory-settings-dialog.tsx",name=>{
  if(name==="react")return {useRef:()=>submitting};
  if(name==="react-hook-form")return {useForm:options=>{
   control??=createFormControl(options);
   return {...control,formState:control.control._formState};
  },useWatch:({name:field})=>control.getValues(field)};
  if(name==="@tanstack/react-query")return {useQuery:options=>{previews.push(options);return options.enabled?{isLoading:false,isError:false,data:preview}:{isLoading:false,isError:false}}};
  if(name==="../infrastructure/inventory-api")return {previewInventoryUnitChange:()=>preview};
  if(name.startsWith("@/design-system"))return {Dialog:"Dialog",ModalHeader:"ModalHeader",ModalFooter:"ModalFooter",Button:"Button",FormField:"FormField",Input:"Input",Icon:"Icon",ModalForm:"form",ModalBody:"ModalBody",FormSection:"FormSection",Select:"Select",InlineNotice:"InlineNotice",RowActionButton:"RowActionButton",Tag:"Tag"};
  if(name==="../domain/inventory-settings-schema")return schema;
  return require(name);
 });
 function render(){
  const nodes=[];
  function visit(node){if(Array.isArray(node)){node.forEach(visit);return}if(node&&typeof node==="object"){nodes.push(node);visit(node.props?.children)}}
  visit(InventorySettingsDialog({item:record,units,busy,save,close,remove,removePresentation,removingPresentation:""}));
  return {nodes,dialog:nodes.find(node=>node.type==="Dialog"),form:nodes.find(node=>node.type==="form"),button:nodes.find(node=>node.type==="Button"&&!node.props.kind&&node.props.action!=="cancel")};
 }
 render();return {render,control,previews};
}

test("niveles de inventario usa cuerpo y footer del modal compartido",()=>{
 const {dialog,form,nodes}=mount().render();
 assert.ok(dialog.props.children.includes(form));
 const body=form.props.children[0];
 assert.equal(body.type,"ModalBody");
 assert.equal(body.props.children[0].props.title,"Datos del artículo");
 assert.equal(body.props.children[0].props.children.props.className,"form-grid");
 assert.equal(form.props.children[1].type,"ModalFooter");
 assert.deepEqual(Array.from(form.props.children[1].props.children.filter(Boolean),node=>node.props.children),["Cancelar","Guardar"]);
 assert.deepEqual(nodes.filter(node=>node.type==="FormField").map(node=>node.props.label),["Nombre","Unidad","Stock mínimo","Stock óptimo"]);
 assert.equal(nodes.find(node=>node.type==="ModalHeader").props.title,"Editar artículo");
 assert.equal(nodes.some(node=>node.type==="Input"&&node.props.name==="reorderPoint"),false);
 assert.equal(dialog.props["aria-labelledby"],nodes.find(node=>node.type==="ModalHeader").props.titleId);
 assert.equal(dialog.props.className,"crud-modal compact modal-panel-in");
 assert.ok(!nodes.some(node=>node.props.className==="modal-accent"));
});

test("la estructura activa márgenes, acciones contiguas y una columna a 390 px sin CSS local",()=>{
 const parse=file=>{const rules=[];postcss.parse(readFileSync(new URL(file,root),"utf8")).walkRules(rule=>rules.push(rule));return rules;};
 const rules=parse("styles/globals.css"),modal=parse("design-system/styles/modal.css");
 const value=(rule,prop)=>rule.nodes.find(node=>node.prop===prop)?.value;
 const desktop=(list,selector)=>list.find(rule=>rule.selector===selector&&rule.parent.type==="root");
  assert.equal(value(desktop(modal,".modal-body"),"padding"),"var(--space-2) var(--space-6) var(--space-6)","el cuerpo usa el relleno del modal compartido");
 const footer=desktop(modal,".modal-footer,.crud-modal>footer,.crud-modal>form>footer");
 assert.equal(value(footer,"justify-content"),"flex-end");
 assert.equal(value(footer,"gap"),"var(--space-2-5)");
 const mobile=rules.find(rule=>rule.selector===".form-grid"&&rule.parent.type==="atrule");
 const breakpoint=Number(mobile.parent.params.match(/width\s*<=\s*(\d+)px/)[1]);
 assert.ok(breakpoint>=390);
 assert.equal(value(mobile,"grid-template-columns"),"1fr");
 assert.ok(modal.some(rule=>rule.parent.type==="atrule"&&rule.selector===".modal-footer .button,.crud-modal>footer .button,.crud-modal>form>footer .button"&&value(rule,"flex")==="1"));
 const source=readFileSync(new URL("modules/supply/inventory/presentation/inventory-settings-dialog.tsx",root),"utf8");
 assert.doesNotMatch(source,/\.css["']/);
});

test("guardar conserva los niveles y su normalización existente",async()=>{
 const saved=[];const ui=mount({save:async draft=>saved.push(draft)});
 assert.equal(ui.control.getValues("minimumStock"),"2");
 ui.control.setValue("minimumStock","5");
 ui.control.setValue("optimalStock","4");
 await ui.render().button.props.onClick();
 assert.equal(saved.length,1);
 assert.equal(saved[0].minimumStock,"5");
 assert.equal("reorderPoint" in saved[0],false);
 assert.equal(saved[0].optimalStock,"5");
 assert.equal(saved[0].inventoryItemId,item.inventoryItemId);
 assert.equal(saved[0].name,item.name);
});

test("permite corregir el nombre y rechaza nombres vacíos o demasiado largos",async()=>{
 const saved=[];const ui=mount({save:async draft=>saved.push(draft)});
 for(const name of ["   ","a".repeat(161)]){
  ui.control.setValue("name",name);await ui.render().button.props.onClick();
  assert.equal(saved.length,0);assert.ok(ui.control.getFieldState("name").error);
 }
 ui.control.setValue("name","  Agua mineral  ");await ui.render().button.props.onClick();
 assert.equal(saved.length,1);assert.equal(saved[0].name,"Agua mineral");
 assert.equal(saved[0].minimumStock,"2");
});

test("el conflicto de nombre conserva el editor y permite corregir sin perder niveles",async()=>{
 let calls=0;const ui=mount({save:async()=>{if(++calls===1)throw {code:"inventory_item_name_taken"};}});
 ui.control.setValue("name","Nombre repetido");ui.control.setValue("minimumStock","8");
 await ui.render().button.props.onClick();
 assert.match(ui.control.getFieldState("name").error.message,/Ya existe/);
 assert.equal(ui.control.getValues("minimumStock"),"8");
 ui.control.setValue("name","Nombre corregido");await ui.render().button.props.onClick();
 assert.equal(calls,2);assert.equal(ui.control.getFieldState("name").error,undefined);
});

test("el contrato de edición envía el nombre sin cambiar la identidad del artículo",async()=>{
 const requests=[];
 const api=compile("modules/supply/inventory/infrastructure/inventory-api.ts",name=>name==="@/shared/api/client"?{apiFetch:async(path,options)=>requests.push({path,body:JSON.parse(options.body)})}:require(name));
 await api.updateInventorySettings({inventoryItemId:"water",name:"  Agua mineral  ",minimumStock:"5",optimalStock:"10"});
 assert.equal(requests[0].path,"inventory/items/water/settings");
 assert.deepEqual(requests[0].body,{name:"Agua mineral",minimumStock:5,optimalStock:10});
});

test("guardar omite el reorden heredado del contrato HTTP",async()=>{
 const requests=[];
 const api=compile("modules/supply/inventory/infrastructure/inventory-api.ts",name=>name==="@/shared/api/client"?{apiFetch:async(path,options)=>{requests.push({path,body:JSON.parse(options.body)})}}:require(name));
 await api.updateInventorySettings({inventoryItemId:"water",minimumStock:"5",optimalStock:"10"});
 assert.equal(requests[0].path,"inventory/items/water/settings");
 assert.deepEqual(requests[0].body,{minimumStock:5,optimalStock:10});
});

test("unidades contables muestran enteros y rechazan fracciones en ambos niveles",async()=>{
 const saved=[];const ui=mount({save:async value=>saved.push(value)});
 assert.ok(ui.render().nodes.filter(node=>node.type==="Input"&&node.props.type==="number").every(node=>node.props.step==="1"&&node.props.inputMode==="numeric"));
 for(const field of ["minimumStock","optimalStock"]){
  ui.control.setValue(field,"1.5");
  await ui.render().button.props.onClick();
  assert.equal(saved.length,0);
  assert.equal(ui.render().nodes.find(node=>node.type==="FormField"&&node.props.children.props.name===field).props.error,"Ingresa una cantidad entera para esta unidad.");
  ui.control.setValue(field,"2");
 }
 await ui.render().button.props.onClick();assert.equal(saved.length,1);
});

test("unidades medibles conservan fracciones y eliminan solo ceros de presentación",async()=>{
 const saved=[];const ui=mount({record:{...item,unit:"kg",unitName:"Kilogramo",unitCountable:false,minimumStock:"0.500",reorderPoint:"1.250",optimalStock:"2.000"},save:async value=>saved.push(value)});
 assert.equal(ui.control.getValues("minimumStock"),"0.5");
 assert.equal(ui.control.getValues("optimalStock"),"2");
 assert.ok(ui.render().nodes.filter(node=>node.type==="Input"&&node.props.type==="number").every(node=>node.props.step==="0.001"&&node.props.inputMode==="decimal"));
 await ui.render().button.props.onClick();assert.equal(saved[0].minimumStock,"0.5");
});

test("un nivel fraccionario histórico no se redondea silenciosamente",async()=>{
 let saved=0;const ui=mount({record:{...item,minimumStock:"99.993"},save:async()=>{saved++}});
 assert.equal(ui.control.getValues("minimumStock"),"99.993");
 await ui.render().button.props.onClick();assert.equal(saved,0);
 ui.control.setValue("minimumStock","100");
 await ui.render().button.props.onClick();assert.equal(saved,1);
});

test("campos vacíos o negativos muestran validación local sin enviar al API",async()=>{
 let saved=0;const ui=mount({save:async()=>{saved++}});
 for(const name of ["minimumStock","optimalStock"]){
  for(const invalid of ["","-1","NaN","Infinity"]){
   ui.control.setValue(name,invalid);
   await ui.render().button.props.onClick();
   assert.equal(saved,0);
   assert.equal(ui.render().nodes.find(node=>node.type==="FormField"&&node.props.children.props.name===name).props.error,"Ingresa una cantidad igual o mayor que cero.");
  }
  ui.control.setValue(name,"0");
 }
 await ui.render().button.props.onClick();assert.equal(saved,1);
});

test("una sola petición espera la respuesta y bloquea campos y cierres mientras guarda",async()=>{
 let saved=0,finish;const pending=new Promise(resolve=>{finish=resolve});
 const ui=mount({save:async()=>{saved++;await pending}});
 const {button,form}=ui.render();let prevented=0;
 form.props.onSubmit({preventDefault(){prevented++}});
 assert.equal(prevented,1);assert.equal(saved,0);
 const first=button.props.onClick();
 for(let i=0;i<3;i++)button.props.onClick();
 await tick();assert.equal(saved,1);
 const loading=ui.render();
 assert.equal(loading.dialog.props["aria-busy"],true);
 assert.equal(loading.button.props.children,"Guardando…");
 assert.ok(loading.nodes.filter(node=>["Button","Input","button"].includes(node.type)).every(node=>node.props.disabled===true));
 finish();await first;
 assert.equal(ui.render().button.props.disabled,false);
});

test("ocupado también impide llamadas directas y conserva el cierre global por respuesta",async()=>{
 let saved=0,closed=0;const close=()=>{closed++};
 const {button,dialog}=mount({busy:true,close,save:async()=>{saved++}}).render();
 assert.equal(button.props.disabled,true);
 await button.props.onClick();assert.equal(saved,0);
 assert.equal(dialog.props.onResponseClose,close);
 dialog.props.onResponseClose();assert.equal(closed,1);
});

test("la unidad se elige del catálogo del API y solo viaja cuando cambia",async()=>{
 const saved=[];const ui=mount({save:async draft=>saved.push(draft),preview:{unit:"und",unitName:"Unidad",mode:"free",allowed:true,blockers:[],quantity:"0",fromUnit:"Botella",toUnit:"Unidad"}});
 const select=ui.render().nodes.find(node=>node.type==="Select");
 assert.deepEqual(Array.from(select.props.children,option=>option.props.value),["botella","und","kg"]);
 await ui.render().button.props.onClick();
 assert.equal(saved[0].unit,undefined,"sin cambio no envía la unidad");
 ui.control.setValue("unit","und");
 ui.render();
 assert.ok(ui.previews.some(options=>options.enabled&&options.queryKey.includes("und")),"consulta el efecto del cambio");
 await ui.render().button.props.onClick();
 assert.equal(saved[1].unit,"und");
});

test("una corrección con historial explica que las cantidades conservan su número",()=>{
 const ui=mount({record:{...item,hasHistory:true},preview:{unit:"und",unitName:"Unidad",mode:"correction",allowed:true,blockers:[],quantity:"10",fromUnit:"Kilogramo",toUnit:"Unidad"}});
 ui.control.setValue("unit","und");
 const notice=ui.render().nodes.find(node=>node.type==="InlineNotice");
 assert.equal(notice.props.tone,"warning");
 assert.equal(notice.props.title,"Corrección de unidad");
 assert.match(notice.props.children.join(""),/10 kilogramo pasan a 10 unidad/);
});

test("un cambio de unidad bloqueado lista los motivos y no permite guardar",async()=>{
 let saved=0;const ui=mount({save:async()=>{saved++},preview:{unit:"und",unitName:"Unidad",mode:"correction",allowed:false,blockers:["Lo usa la receta de Pan en otra unidad. Cambia esas recetas primero."],quantity:"0",fromUnit:"Botella",toUnit:"Unidad"}});
 ui.control.setValue("unit","und");
 const view=ui.render();
 const notice=view.nodes.find(node=>node.type==="InlineNotice");
 assert.equal(notice.props.tone,"danger");
 assert.ok(view.nodes.some(node=>node.type==="span"&&String(node.props.children).includes("receta de Pan")));
 assert.equal(view.button.props.disabled,true);
 await view.button.props.onClick();assert.equal(saved,0);
});

test("otro local que usa el artículo se menciona al cambiar nombre o unidad",()=>{
 const ui=mount({record:{...item,otherLocations:[{locationId:"n",name:"Sucursal Norte",active:true,quantity:"0"}]}});
 assert.equal(ui.render().nodes.some(node=>node.type==="InlineNotice"),false);
 ui.control.setValue("name","Agua sin gas");
 const notice=ui.render().nodes.find(node=>node.type==="InlineNotice");
 assert.match(notice.props.children.join(""),/Sucursal Norte/);
});

test("eliminar solo aparece sin historial y abre la confirmación del padre",()=>{
 assert.equal(mount().render().nodes.some(node=>node.type==="Button"&&node.props.kind==="danger-outline"),false);
 let removed=0;const view=mount({record:{...item,canDelete:true},remove:()=>{removed++}}).render();
 const footer=view.form.props.children[1].props.children.filter(Boolean);
 assert.deepEqual(Array.from(footer,node=>node.props.children),["Eliminar","Cancelar","Guardar"],"acciones contiguas a la derecha");
 footer[0].props.onClick();assert.equal(removed,1);
});

test("las presentaciones de compra se quitan salvo la base",()=>{
 const removed=[];
 const record={...item,presentations:[{id:"base",presentationType:"unit",unitsPerPresentation:"1",name:"Unidad",isDefault:true,isBase:true,used:true},{id:"pack",presentationType:"package",unitsPerPresentation:"6",name:"Paquete",isDefault:false,isBase:false,used:false}]};
 const view=mount({record,removePresentation:id=>removed.push(id)}).render();
 const actions=view.nodes.filter(node=>node.type==="RowActionButton");
 assert.equal(actions.length,1);
 assert.equal(actions[0].props.label,"Quitar Paquete x 6");
 actions[0].props.onClick();assert.deepEqual(removed,["pack"]);
 assert.ok(view.nodes.some(node=>node.type==="Tag"&&node.props.children==="Base"));
});

test("el contrato envía unidad, estado por local, eliminación y filtro de estado",async()=>{
 const requests=[];
 const api=compile("modules/supply/inventory/infrastructure/inventory-api.ts",name=>name==="@/shared/api/client"?{apiFetch:async(path,options={})=>{requests.push({path,method:options.method,body:options.body?JSON.parse(options.body):undefined})}}:require(name));
 await api.updateInventorySettings({inventoryItemId:"water",name:"Agua",unit:"und",minimumStock:"1",optimalStock:"2"});
 await api.setInventoryItemActive("water",false);
 await api.deleteInventoryItem("water");
 await api.listInventory({q:"",page:1,pageSize:10});
 await api.listInventory({q:"",page:1,pageSize:10,status:"inactive"});
 await api.removeInventoryPresentation("water","pack");
 assert.deepEqual(requests[0].body,{name:"Agua",unit:"und",minimumStock:1,optimalStock:2});
 assert.deepEqual(requests[1],{path:"inventory/items/water/status",method:"PATCH",body:{active:false}});
 assert.deepEqual(requests[2],{path:"inventory/items/water",method:"DELETE",body:undefined});
 assert.match(requests[3].path,/status=active/);
 assert.match(requests[4].path,/status=inactive/);
 assert.deepEqual(requests[5],{path:"inventory/items/water/presentations/pack",method:"DELETE",body:undefined});
});
