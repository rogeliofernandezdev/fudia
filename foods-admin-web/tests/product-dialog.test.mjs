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
const validDraft={name:"Lomo saltado",price:"25",costPrice:"",prepMinutes:"",categoryId:"cat-general",description:"",quantityControl:"none",allergens:[],featured:false,active:true,productType:"prepared",imageUrl:null};
const tick=()=>new Promise(resolve=>setImmediate(resolve));
function mount({save=async()=>{},draft={...validDraft},busy=false,categories=[],errors={}}={}){
 let formControl,formOptions;
 const {ProductDialog}=compile("modules/menu/products/presentation/product-dialog.tsx",name=>{
  if(name==="react")return {useRef:value=>({current:value}),useState:value=>[value,()=>{}]};
  if(name==="react-hook-form")return {Controller:"Controller",useForm:options=>{
   formOptions=options;
   formControl=createFormControl(options);
   return {...formControl,watch:()=>draft,formState:{errors}};
  }};
  if(name==="../domain/product-schema")return schema;
  if(name==="./product-image-field")return {ProductImageField:"ProductImageField"};
  if(name.endsWith(".css"))return {};
  if(name==="@/design-system")return {FormField:"FormField",InlineNotice:"InlineNotice",Input:"input",Select:"select",ModalHeader:"ModalHeader",ModalForm:"form",ModalBody:"ModalBody",ModalFooter:"ModalFooter"};
  if(name==="@/design-system/dialog")return {Dialog:"Dialog"};
  if(name==="@/design-system/icons")return {Icon:"Icon"};
  if(name==="@/design-system/rich-text-editor")return {RichTextEditor:"RichTextEditor"};
  if(name==="./allergen-field")return {AllergenField:"AllergenField"};
  return require(name);
 });
 const tree=ProductDialog({draft,categories,destinations:[{value:"kitchen",label:"Cocina"},{value:"bar",label:"Barra"},{value:"direct",label:"Sin preparación (lo entrega el mozo)"}],loadAllergens:async()=>[],currencySymbol:"S/",busy,close:()=>{},save});
 const nodes=[];
 function visit(node){if(!node||typeof node!=="object")return;if(Array.isArray(node)){node.forEach(visit);return}nodes.push(node);visit(node.props?.children)}
 visit(tree);
 return {form:nodes.find(node=>node.type==="form"),button:nodes.find(node=>node.type==="button"&&node.props.className==="button primary"),nodes,control:formControl,options:formOptions};
}

test("producto mantiene cuerpo y pie separados con primitivas compartidas al guardar",()=>{
 for(const busy of [false,true]){
  const {form,nodes}=mount({busy});
  assert.deepEqual(Array.from(form.props.children,node=>node.type),["ModalBody","ModalFooter"]);
  assert.equal(nodes.find(node=>node.type==="ModalHeader").props.busy,busy);
  assert.ok(nodes.find(node=>node.type==="ModalBody").props.className.includes("product-form-body"));
  assert.equal(nodes.find(node=>node.type==="button"&&node.props.children==="Cancelar").props.disabled,busy);
 }
});

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

test("crear con porciones muestra la cantidad a ancho completo y editar no la sobrescribe",()=>{
 const created=mount({draft:{...validDraft,quantityControl:"portions"}});
 const field=created.nodes.find(node=>node.type==="FormField"&&node.props.label==="Cantidad disponible hoy");
 assert.ok(field);
 assert.equal(field.props.className,"span-2");
 const input=created.nodes.find(node=>node.type==="input"&&node.props.name==="initialPortionQuantity");
 assert.equal(input.props.type,"number");
 assert.equal(input.props.min,"1");
 assert.equal(input.props.step,"1");
 for(const draft of [{...validDraft},{...validDraft,id:"existing",quantityControl:"portions"}]){
  assert.equal(mount({draft}).nodes.some(node=>node.props?.label==="Cantidad disponible hoy"),false);
 }
});

test("las porciones requieren cantidad positiva y el mismo guardado incluye producto y cantidad",async()=>{
 const saved=[];
 const {button,control}=mount({draft:{...validDraft,quantityControl:"portions"},save:async value=>{saved.push(value)}});
 for(const invalid of ["","0","-1","1.5","abc","2147483648"]){
  control.setValue("initialPortionQuantity",invalid);
  button.props.onClick();await tick();
  assert.equal(saved.length,0,invalid);
 }
 control.setValue("initialPortionQuantity","15");
 button.props.onClick();await tick();
 assert.equal(saved.length,1);
 assert.equal(saved[0].name,"Lomo saltado");
 assert.equal(saved[0].initialPortionQuantity,"15");
});

test("cambiar a Sin control no exige una cantidad previa inválida; edición conserva el flujo comercial",async()=>{
 let saved=0;
 const {button,control}=mount({draft:{...validDraft,quantityControl:"portions",initialPortionQuantity:""},save:async()=>{saved++}});
 control.setValue("quantityControl","none");
 button.props.onClick();await tick();
 assert.equal(saved,1);
 const edited=mount({draft:{...validDraft,id:"existing",quantityControl:"portions"},save:async()=>{saved++}});
 edited.button.props.onClick();await tick();
 assert.equal(saved,2);
});

test("el API guarda las porciones en un único POST, nunca con un PATCH adicional de disponibilidad",async()=>{
 const calls=[];
 const {saveProduct}=compile("modules/menu/products/infrastructure/products-api.ts",name=>{
  if(name==="@/shared/api/client")return {apiFetch:async(path,init)=>{calls.push({path,init});return {id:"created"}}};
  if(name==="@/shared/api/product-image")return {};
  if(name==="@/shared/api/image-upload")return {};
  return require(name);
 });
 await saveProduct({...validDraft,quantityControl:"portions",initialPortionQuantity:"15"},null);
 assert.equal(calls.length,1);
 assert.equal(calls[0].path,"products");
 assert.equal(calls[0].init.method,"POST");
 assert.equal(JSON.parse(calls[0].init.body).initialPortionQuantity,15);
 for(const draft of [{...validDraft,initialPortionQuantity:"15"},{...validDraft,id:"existing",quantityControl:"portions",initialPortionQuantity:"15"}]){
  await saveProduct(draft,null);
  assert.equal(Object.hasOwn(JSON.parse(calls.at(-1).init.body),"initialPortionQuantity"),false);
 }
});

test("Destino de atención: sin texto de ayuda y con aviso si «Sin preparación» se aplica a algo que se prepara",()=>{
 const field=nodes=>nodes.find(node=>node.type==="FormField"&&node.props.label==="Destino de atención");
 const kitchen=mount({draft:{...validDraft,serviceDestination:"kitchen"}}).nodes;
 assert.equal(field(kitchen).props.help,undefined,"la etiqueta ya explica el destino");
 assert.equal(kitchen.some(node=>node.type==="InlineNotice"),false);
 const soda=mount({draft:{...validDraft,serviceDestination:"direct"}}).nodes;
 assert.equal(field(soda).props.help,undefined);
 assert.equal(soda.some(node=>node.type==="InlineNotice"),false,"una gaseosa sin receta ni tiempo no muestra aviso");
 for(const draft of [{...validDraft,serviceDestination:"direct",hasRecipe:true},{...validDraft,serviceDestination:"direct",prepMinutes:"15"}]){
  const notice=mount({draft}).nodes.find(node=>node.type==="InlineNotice");
  assert.equal(notice?.props.tone,"warning","aviso suave, no bloqueo");
 }
});

const categoryField=nodes=>nodes.find(node=>node.type==="FormField"&&node.props.label==="Categoría");
const optionLabels=nodes=>{
 const select=nodes.find(node=>node.type==="select"&&node.props.name==="categoryId");
 return [select.props.children].flat(2).map(option=>option.props.children);
};

test("la categoría es obligatoria: sin «Sin categoría» y con el error visible en el campo",async()=>{
 const categories=[{id:"cat-general",name:"General",active:true,productScope:"both"},{id:"cat-old",name:"Antigua",active:false,productScope:"both"},{id:"cat-drinks",name:"Bebidas",active:true,productScope:"both"}];
 const {nodes}=mount({draft:{...validDraft,categoryId:""},categories});
 assert.deepEqual(optionLabels(nodes),["Selecciona una categoría","General","Bebidas"]);
 assert.equal(nodes.some(node=>node.type==="option"&&node.props.children==="Sin categoría"),false);
 const invalid=mount({draft:{...validDraft,categoryId:""},categories,errors:{categoryId:{message:"Selecciona una categoría."}}}).nodes;
 assert.equal(categoryField(invalid).props.error,"Selecciona una categoría.");
 assert.equal(invalid.find(node=>node.type==="select"&&node.props.name==="categoryId").props["aria-invalid"],true);
 let saved=0;
 const {button,control}=mount({draft:{...validDraft,categoryId:""},categories,save:async()=>{saved++}});
 button.props.onClick();await tick();
 assert.equal(saved,0,"no guarda sin categoría");
 assert.equal(control.getFieldState("categoryId").error?.message,"Selecciona una categoría.");
 control.setValue("categoryId","cat-drinks");
 button.props.onClick();await tick();
 assert.equal(saved,1);
});

test("la categoría inactiva solo se conserva para el producto que ya la usa y una única activa se preselecciona",()=>{
 const categories=[{id:"cat-general",name:"General",active:true,productScope:"both"},{id:"cat-old",name:"Antigua",active:false,productScope:"both"}];
 assert.deepEqual(optionLabels(mount({draft:{...validDraft,id:"existing",categoryId:"cat-old"},categories}).nodes),["Selecciona una categoría","General","Antigua"]);
 assert.equal(mount({draft:{...validDraft,categoryId:""},categories}).options.defaultValues.categoryId,"cat-general","empresa nueva: «General» queda elegida");
 assert.equal(mount({draft:{...validDraft,categoryId:""},categories:[...categories,{id:"cat-drinks",name:"Bebidas",active:true,productScope:"both"}]}).options.defaultValues.categoryId,"","con varias categorías se elige explícitamente");
});

test("el esquema rechaza la categoría vacía o con espacios",()=>{
 for(const categoryId of [""," "]){
  const result=schema.productSchema.safeParse({...validDraft,categoryId});
  assert.equal(result.success,false);
  assert.equal(result.error.issues.find(issue=>issue.path[0]==="categoryId")?.message,"Selecciona una categoría.");
 }
 assert.equal(schema.productSchema.safeParse(validDraft).success,true);
});
