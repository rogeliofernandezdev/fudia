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
 const source=readFileSync(new URL(path,root),"utf8");
 vm.runInNewContext(ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve});
 return exports;
}
const resolver=compile("shared/forms/zod-resolver.ts",require);
const schema=compile("modules/menu/products/domain/product-schema.ts",name=>name==="@/shared/forms/zod-resolver"?resolver:require(name));
const validDraft={name:"Lomo saltado",price:"25",costPrice:"",prepMinutes:"",categoryId:"",description:"",quantityControl:"none",allergens:[],featured:false,active:true,productType:"prepared",imageUrl:null};
const tick=()=>new Promise(resolve=>setImmediate(resolve));
function mount({save=async()=>{},draft={...validDraft},busy=false}={}){
 let formControl;
 const {ProductDialog}=compile("modules/menu/products/presentation/product-dialog.tsx",name=>{
  if(name==="react")return {useRef:value=>({current:value}),useState:value=>[value,()=>{}]};
  if(name==="react-hook-form")return {Controller:"Controller",useForm:options=>{
   formControl=createFormControl(options);
   return {...formControl,watch:()=>draft,formState:{errors:{}}};
  }};
  if(name==="../domain/product-schema")return schema;
  if(name.endsWith(".css"))return {};
  if(name==="@/design-system")return {FormField:"FormField",Input:"input",Select:"select"};
  if(name==="@/design-system/icons")return {Icon:"Icon"};
  if(name==="@/design-system/rich-text-editor")return {RichTextEditor:"RichTextEditor"};
  if(name==="react-select/async")return {default:"AsyncSelect"};
  return require(name);
 });
 const tree=ProductDialog({draft,categories:[],loadAllergens:async()=>[],currencySymbol:"S/",busy,close:()=>{},save});
 const nodes=[];
 function visit(node){if(!node||typeof node!=="object")return;if(Array.isArray(node)){node.forEach(visit);return}nodes.push(node);visit(node.props?.children)}
 visit(tree);
 return {form:nodes.find(node=>node.type==="form"),button:nodes.find(node=>node.type==="button"&&node.props.className==="button primary"),nodes,control:formControl};
}

test("Enter implícito repetido no guarda el producto",async()=>{
 let saved=0,prevented=0;
 const {form,button,nodes}=mount({save:async()=>{saved++}});
 for(let i=0;i<4;i++)form.props.onSubmit({preventDefault(){prevented++}});
 await tick();
 assert.equal(prevented,4);
 assert.equal(saved,0);
 assert.equal(button.props.children,"Guardar");
 // No native submit button can become the default Enter action.
 assert.ok(nodes.filter(node=>node.type==="button").every(node=>node.props.type==="button"||node.props["aria-label"]==="Cerrar"));
});

test("cuatro activaciones rápidas de Guardar producen una sola petición",async()=>{
 let saved=0,finish;
 const pending=new Promise(resolve=>{finish=resolve});
 const {button}=mount({save:async()=>{saved++;await pending}});
 for(let i=0;i<4;i++)button.props.onClick();
 await tick();
 assert.equal(saved,1);
 button.props.onClick();
 await tick();
 assert.equal(saved,1);
 finish();await tick();
});

test("Guardar mantiene la validación y permite corregir datos inválidos",async()=>{
 let saved=0;
 const {button,control}=mount({draft:{...validDraft,name:""},save:async()=>{saved++}});
 button.props.onClick();await tick();
 assert.equal(saved,0);
 control.setValue("name","Lomo saltado");
 button.props.onClick();await tick();
 assert.equal(saved,1);
});

test("un intento fallido gestionado por la mutación permite reintentar",async()=>{
 let attempts=0;
 const {button}=mount({save:async()=>{
  attempts++;
  // CatalogManager reports failures through onError and settles mutateAsync.
  try{if(attempts===1)throw new Error("Sin conexión")}catch{/* Failure already reported by the mutation. */}
 }});
 button.props.onClick();await tick();
 button.props.onClick();await tick();
 assert.equal(attempts,2);
});

test("el estado ocupado bloquea Guardar también desde el handler",async()=>{
 let saved=0;
 const {button}=mount({busy:true,save:async()=>{saved++}});
 assert.equal(button.props.disabled,true);
 assert.equal(button.props.children,"Guardando…");
 button.props.onClick();await tick();
 assert.equal(saved,0);
});
