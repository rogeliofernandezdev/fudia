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
 vm.runInNewContext(ts.transpileModule(readFileSync(new URL(path,root),"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve});
 return exports;
}
const resolver=compile("shared/forms/zod-resolver.ts",require);
const schema=compile("modules/supply/inventory/domain/inventory-settings-schema.ts",name=>name==="@/shared/forms/zod-resolver"?resolver:require(name));
const item={inventoryItemId:"water",name:"Agua mineral San Luis",minimumStock:"2.000",reorderPoint:"2.000",optimalStock:"2.000"};
const tick=()=>new Promise(resolve=>setImmediate(resolve));
function mount({busy=false,save=async()=>{},close=()=>{}}={}){
 let control;const submitting={current:false};
 const {InventorySettingsDialog}=compile("modules/supply/inventory/presentation/inventory-settings-dialog.tsx",name=>{
  if(name==="react")return {useRef:()=>submitting};
  if(name==="react-hook-form")return {useForm:options=>{
   control??=createFormControl(options);
   return {...control,formState:control.control._formState};
  }};
  if(name.startsWith("@/design-system"))return {Dialog:"Dialog",Button:"Button",FormField:"FormField",Input:"Input",Icon:"Icon"};
  if(name==="../domain/inventory-settings-schema")return schema;
  return require(name);
 });
 function render(){
  const nodes=[];
  function visit(node){if(Array.isArray(node)){node.forEach(visit);return}if(node&&typeof node==="object"){nodes.push(node);visit(node.props?.children)}}
  visit(InventorySettingsDialog({item,busy,save,close}));
  return {nodes,dialog:nodes.find(node=>node.type==="Dialog"),form:nodes.find(node=>node.type==="form"),button:nodes.find(node=>node.type==="Button"&&node.props.kind!=="ghost")};
 }
 render();return {render,control};
}

test("niveles de inventario usa cuerpo y footer del modal compartido",()=>{
 const {dialog,form,nodes}=mount().render();
 assert.ok(dialog.props.children.includes(form));
 assert.equal(form.props.children[0].props.className,"form-grid");
 assert.equal(form.props.children[1].type,"footer");
 assert.equal(form.props.children[1].props.children.length,2);
 assert.equal(nodes.find(node=>node.type==="FormField"&&node.props.label==="Stock óptimo").props.className,"span-2");
 assert.equal(dialog.props["aria-labelledby"],nodes.find(node=>node.type==="h2").props.id);
 assert.equal(dialog.props.className,"crud-modal compact modal-panel-in");
 assert.ok(nodes.some(node=>node.props.className==="modal-accent"));
});

test("la estructura activa márgenes, acciones separadas y una columna a 390 px sin CSS local",()=>{
 const css=postcss.parse(readFileSync(new URL("styles/globals.css",root),"utf8"));
 const rules=[];css.walkRules(rule=>rules.push(rule));
 const value=(rule,prop)=>rule.nodes.find(node=>node.prop===prop)?.value;
 const desktop=selector=>rules.find(rule=>rule.selector===selector&&rule.parent.type==="root");
 assert.equal(value(desktop(".crud-modal form"),"padding"),"var(--space-20)");
 assert.equal(value(desktop(".crud-modal form>footer"),"justify-content"),"flex-end");
 assert.equal(value(desktop(".crud-modal form>footer"),"gap"),"var(--space-9)");
 const mobile=rules.find(rule=>rule.selector===".form-grid"&&rule.parent.type==="atrule");
 const breakpoint=Number(mobile.parent.params.match(/width\s*<=\s*(\d+)px/)[1]);
 assert.ok(breakpoint>=390);
 assert.equal(value(mobile,"grid-template-columns"),"1fr");
 assert.ok(rules.some(rule=>rule.parent===mobile.parent&&rule.selector===".crud-modal form>footer .button"&&value(rule,"flex")==="1"));
 const source=readFileSync(new URL("modules/supply/inventory/presentation/inventory-settings-dialog.tsx",root),"utf8");
 assert.doesNotMatch(source,/\.css["']/);
});

test("guardar conserva los niveles y su normalización existente",async()=>{
 const saved=[];const ui=mount({save:async draft=>saved.push(draft)});
 assert.equal(ui.control.getValues("minimumStock"),"2.000");
 ui.control.setValue("minimumStock","5");
 ui.control.setValue("reorderPoint","3");
 ui.control.setValue("optimalStock","4");
 await ui.render().button.props.onClick();
 assert.equal(saved.length,1);
 assert.equal(saved[0].minimumStock,"5");
 assert.equal(saved[0].reorderPoint,"5");
 assert.equal(saved[0].optimalStock,"5");
 assert.equal(saved[0].inventoryItemId,item.inventoryItemId);
});

test("campos vacíos o negativos muestran validación local sin enviar al API",async()=>{
 let saved=0;const ui=mount({save:async()=>{saved++}});
 for(const name of ["minimumStock","reorderPoint","optimalStock"]){
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
