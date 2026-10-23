import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
import {createFormControl} from "react-hook-form";

const require=createRequire(import.meta.url);
const root=new URL("../src/",import.meta.url);
function compile(path,resolve){
 const exports={};
 vm.runInNewContext(ts.transpileModule(readFileSync(new URL(path,root),"utf8"),{compilerOptions:{target:ts.ScriptTarget.ES2020,module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve});
 return exports;
}
const schema=compile("modules/supply/purchases/domain/purchase-schema.ts",require);
const tick=()=>new Promise(resolve=>setImmediate(resolve));
const valid={name:"Agua mineral",categoryId:"drinks",price:"3.50",unit:"botella",presentationType:"package",unitsPerPresentation:"10"};
function mount({save=async()=>{},busy=false,modules={},linkable=[]}={}){
 const state=[],refs=[];let cursor=0,refCursor=0,formControl;
 const {PurchaseItemDialog}=compile("modules/supply/purchases/presentation/purchase-item-dialog.tsx",name=>{
  if(name==="react")return {useState:initial=>{const i=cursor++;if(!(i in state))state[i]=initial;return[state[i],value=>{state[i]=value}]},useRef:initial=>{const i=refCursor++;return refs[i]??(refs[i]={current:initial})},useDeferredValue:value=>value};
  if(name==="react-hook-form")return {useForm:options=>{
   formControl??=createFormControl(options);
   return {...formControl,formState:{errors:formControl.control._formState.errors,isSubmitted:formControl.control._formState.isSubmitted,isSubmitting:formControl.control._formState.isSubmitting}};
  },useWatch:({name})=>formControl.getValues(name)};
  if(name==="@tanstack/react-query")return {useQuery:({queryKey})=>({data:{items:queryKey[0]==="purchase-linkable-products"?linkable:[]},isLoading:false,isError:false})};
  if(name==="@/providers/session-context")return {useSession:()=>({location:{country:"PE"},modules})};
  if(name.startsWith("@/design-system"))return Object.fromEntries(["Button","Icon","Input","Select","Textarea","Dialog","EmptyState","ErrorState","Skeleton","Status","ModalHeader","ModalFooter"].map(key=>[key,key]));
  if(name==="../domain/linkable-product")return {linkableControlLabel:()=>"Usa cupo del día"};
  if(name==="../domain/purchase-schema")return schema;
  if(name==="./purchase-unit-field")return {PurchaseUnitField:"PurchaseUnitField"};
  if(name==="./purchase-category-field")return {PurchaseCategoryField:"PurchaseCategoryField"};
  if(name==="./purchase-presentations-field")return {PurchasePresentationsField:"PurchasePresentationsField"};
  if(name==="../infrastructure/purchases-api"||name==="@/shared/i18n/regional-format")return {};
  return require(name);
 });
 function render(){
  cursor=refCursor=0;const nodes=[];
  function visit(node){if(Array.isArray(node)){node.forEach(visit);return}if(node&&typeof node==="object"){nodes.push(node);visit(node.props?.children);visit(node.props?.action)}}
  visit(PurchaseItemDialog({currencySymbol:"S/",busy,close:()=>{},choose:()=>{},save}));
  return {nodes,form:nodes.find(node=>node.type==="form"),button:nodes.find(node=>node.type==="Button"&&["Crear y agregar","Vincular y agregar","Guardando…"].includes(node.props.children))};
 }
 render();
 // Crear parte de la búsqueda; el interruptor «Se vende a clientes» decide el discriminador.
 const chooseMode=mode=>{
  render().nodes.find(node=>node.type==="Button"&&node.props.children==="Crear artículo nuevo").props.onClick();
  render().nodes.find(node=>node.type==="input"&&node.props.type==="checkbox").props.onChange({target:{checked:mode==="new_product"}});
 };
 const fill=values=>Object.entries(values).forEach(([key,value])=>formControl.setValue(key,value));
 return {render,chooseMode,fill,get control(){return formControl}};
}

test("Crear y agregar guarda productos e insumos con unidad y presentación seleccionadas",async()=>{
 for(const mode of ["new_product","new_ingredient"]){
  const saved=[];const ui=mount({save:async draft=>saved.push(draft)});
  ui.chooseMode(mode);ui.fill(valid);
  const {button}=ui.render();
  await button.props.onClick();await tick();
  assert.equal(saved.length,1);
  assert.equal(saved[0].mode,mode);
  assert.equal(saved[0].unit,"botella");
  assert.equal(saved[0].unitsPerPresentation,"10");
 }
});

test("sin unidad el botón permite validar en lugar de quedar inerte",async()=>{
 const ui=mount();ui.chooseMode("new_ingredient");ui.fill({name:"Azúcar"});
 const {button}=ui.render();assert.equal(button.props.disabled,false);
 button.props.onClick();await tick();
 assert.equal(ui.control.control._formState.errors.unit.message,"Selecciona una unidad de inventario.");
 assert.ok(ui.render().nodes.some(node=>node.props.role==="alert"));
});

test("la selección remota corrige la unidad y el mismo botón crea y agrega",async()=>{
 const saved=[];const ui=mount({save:async draft=>saved.push(draft)});
 ui.chooseMode("new_ingredient");ui.fill({name:"Agua mineral"});
 ui.render().button.props.onClick();await tick();
 ui.render().nodes.find(node=>node.type==="PurchaseUnitField").props.onChange("botella");
 ui.render().nodes.find(node=>node.type==="PurchasePresentationsField").props.onChange([{presentationType:"package",unitsPerPresentation:"10",isDefault:true}]);
 ui.render().button.props.onClick();await tick();
 assert.equal(saved.length,1);assert.equal(saved[0].unit,"botella");
});

test("categoría, precio y factor inválidos no envían y explican la corrección",async()=>{
 const saved=[];const ui=mount({save:async draft=>saved.push(draft)});
 ui.chooseMode("new_product");ui.fill(valid);
 for(const [field,value,message] of [["categoryId","","Selecciona una categoría."],["price","abc","Ingresa un precio válido."],["unitsPerPresentation","0","Ingresa un contenido mayor que cero, con hasta tres decimales. La compra en la misma unidad de inventario equivale a 1."]]){
  ui.fill({...valid,[field]:value});ui.render().button.props.onClick();await tick();
  assert.equal(saved.length,0);
  const alert=ui.render().nodes.find(node=>node.props.role==="alert");
  assert.equal(alert.props.children[1],message);
 }
 ui.fill(valid);ui.render().button.props.onClick();await tick();assert.equal(saved.length,1);
});

test("Enter no crea; cuatro activaciones rápidas esperan una única petición",async()=>{
 let saved=0,finish,prevented=0;const pending=new Promise(resolve=>{finish=resolve});
 const ui=mount({save:async()=>{saved++;await pending}});ui.chooseMode("new_ingredient");ui.fill(valid);
 const {button,form}=ui.render();
 form.props.onSubmit({preventDefault(){prevented++}});await tick();
 assert.equal(prevented,1);assert.equal(saved,0);assert.equal(button.props.type,"button");
 for(let i=0;i<4;i++)button.props.onClick();await tick();assert.equal(saved,1);
 assert.equal(ui.render().button.props.disabled,true);
 button.props.onClick();await tick();assert.equal(saved,1);
 finish();await tick();
});

test("API fallido conserva el borrador y permite reintentar sin bloqueo permanente",async()=>{
 let attempts=0;const ui=mount({save:async()=>{
  attempts++;
  try{if(attempts===1)throw new Error("Sin conexión")}catch{/* The parent mutation reports the failure. */}
 }});ui.chooseMode("new_ingredient");ui.fill(valid);
 ui.render().button.props.onClick();await tick();
 assert.equal(ui.control.getValues("name"),valid.name);
 ui.render().button.props.onClick();await tick();assert.equal(attempts,2);
 const busy=mount({busy:true,save:async()=>{attempts++}});busy.chooseMode("new_ingredient");busy.fill(valid);
 busy.render().button.props.onClick();await tick();assert.equal(attempts,2);
 assert.equal(busy.render().form.props.children[0].props.inert,true);
});

test("buscar es la vista inicial y no ofrece crear hasta elegirlo",()=>{
 const ui=mount();const {nodes,button}=ui.render();
 assert.equal(button,undefined);
 assert.ok(nodes.some(node=>node.type==="Button"&&node.props.children==="Crear artículo nuevo"));
 assert.ok(nodes.some(node=>node.type==="Button"&&node.props.children==="Cancelar"));
});

test("con Recetas el artículo nuevo nace como insumo y sin Recetas como producto vendible",()=>{
 for(const [modules,mode] of [[{recetas:true},"new_ingredient"],[{},"new_product"]]){
  const ui=mount({modules});
  ui.render().nodes.find(node=>node.type==="Button"&&node.props.children==="Crear artículo nuevo").props.onClick();
  assert.equal(ui.control.getValues("mode"),mode);
 }
});

test("un producto de la carta sin inventario se vincula sin crear otro",async()=>{
 const saved=[];const product={productId:"product-1",sku:"PROD-0001",name:"Inca Kola 500 ml",categoryName:"Bebidas",quantityControl:"portions"};
 const ui=mount({save:async draft=>saved.push(draft),linkable:[product]});
 ui.render().nodes.find(node=>node.type==="Input"&&node.props.placeholder==="Escribe el nombre del producto o insumo").props.onChange({target:{value:"Inca"}});
 const list=ui.render().nodes.find(node=>node.props?.["aria-label"]==="Productos de la carta sin inventario");
 list.props.children[0].props.onClick();
 ui.fill({unit:"botella",presentationType:"unit",unitsPerPresentation:"1"});
 const {button}=ui.render();
 assert.equal(button.props.children,"Vincular y agregar");
 await button.props.onClick();await tick();
 assert.equal(saved.length,1);
 assert.equal(saved[0].mode,"link_product");
 assert.equal(saved[0].productId,"product-1");
});
