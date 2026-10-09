import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
import {createFormControl} from "react-hook-form";
const require=createRequire(import.meta.url);
const root=new URL("../src/",import.meta.url);
function compile(path,resolve,globals={}){
 const exports={};
 vm.runInNewContext(ts.transpileModule(readFileSync(new URL(path,root),"utf8"),{compilerOptions:{target:ts.ScriptTarget.ES2020,module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve,...globals});return exports;
}
function nodes(tree,result=[]){if(Array.isArray(tree)){tree.forEach(node=>nodes(node,result));return result}if(tree&&typeof tree==="object"){result.push(tree);nodes(tree.props?.children,result)}return result}
function mount({items=[],manage=true,loading=false,error=false,pending=false}={}){
 const state=[],changes=[],writes=[],queries=[];let cursor=0,creates=0;
 const {PurchaseCategoryField}=compile("modules/supply/purchases/presentation/purchase-category-field.tsx",name=>{
  if(name==="react")return {useState:initial=>{const i=cursor++;if(!(i in state))state[i]=initial;return[state[i],value=>{state[i]=value}]}};
  if(name==="react-dom")return {createPortal:tree=>tree};
  if(name==="@/providers/session-context")return {useSession:()=>({organization:{id:"company-a"},can:()=>manage})};
  if(name==="@/modules/menu")return {CategoryDialog:"CategoryDialog",saveCategory:()=>{}};
  if(name==="@/design-system")return {Button:"Button",FormField:"FormField",Select:"Select"};
  if(name==="../infrastructure/purchases-api")return {listPurchaseItemCategories:()=>{}};
  if(name==="@tanstack/react-query")return {
   useQuery:options=>{queries.push(options);return {data:items,isLoading:loading,isError:error,error:error?new Error("Sin conexión"):null,isFetching:false,refetch:()=>writes.push("retry")}},
   useQueryClient:()=>({setQueryData:(key,fn)=>writes.push(fn(items)),invalidateQueries:options=>writes.push(options)}),
   useMutation:options=>({isPending:pending,error:null,reset:()=>{},mutateAsync:async draft=>{creates++;const result={...draft,id:"database-uuid",active:true};options.onSuccess(result);return result}}),
  };
  return require(name);
 },{document:{body:{}}});
 const render=()=>{cursor=0;return nodes(PurchaseCategoryField({value:"",onChange:value=>changes.push(value),disabled:false,onBusyChange:value=>writes.push(value)}))};
 return {render,changes,writes,queries,get creates(){return creates}};
}
test("sin categorías compatibles hay una salida visible y la creación conserva permisos",async()=>{
 const ui=mount(),tree=ui.render();
 assert.equal(tree.find(node=>node.type==="Select").props.disabled,true);
 assert.match(tree.find(node=>node.type==="FormField").props.help,/Crea una categoría/);
 tree.find(node=>node.props["aria-label"]==="Nueva categoría").props.onClick();
 const dialog=ui.render().find(node=>node.type==="CategoryDialog");
 assert.equal(dialog.props.retailOnly,true);assert.equal(dialog.props.draft.productScope,"retail");
 await dialog.props.save({name:"Bebidas",productScope:"retail",sortOrder:0,active:true});
 assert.equal(ui.creates,1);assert.deepEqual(ui.changes,["database-uuid"]);
 assert.ok(ui.render().some(node=>node.type==="option"&&node.props.value==="database-uuid"));
 assert.ok(ui.writes.some(value=>Array.isArray(value)&&value[0].id==="database-uuid"));
 assert.equal(ui.queries[0].queryKey[1],"company-a");
 const readOnly=mount({manage:false}).render();
 assert.equal(readOnly.some(node=>node.props["aria-label"]==="Nueva categoría"),false);
 assert.match(readOnly.find(node=>node.type==="FormField").props.help,/administrador/);
});
test("categorías remotas muestran skeleton, error recuperable y selección",()=>{
 assert.ok(mount({loading:true}).render().some(node=>node.props["aria-label"]==="Cargando categorías"));
 const error=mount({error:true}),tree=error.render();
 assert.equal(tree.find(node=>node.type==="FormField").props.error,"Sin conexión");
 tree.find(node=>node.type==="Button"&&node.props.children==="Reintentar").props.onClick();assert.ok(error.writes.includes("retry"));
 const ready=mount({items:[{id:"remote-id",name:"Bebidas"}]}),readyTree=ready.render();
 readyTree.find(node=>node.type==="Select").props.onChange({target:{value:"remote-id"}});
 assert.deepEqual(ready.changes,["remote-id"]);
});
test("el API consulta categorías de mercadería y usa el POST existente con UUID de servidor",async()=>{
 const calls=[];
 const api=compile("modules/supply/purchases/infrastructure/purchases-api.ts",name=>name==="@/shared/api/client"?{apiFetch:async(path,options)=>{calls.push({path,options});return {items:[{id:"remote",name:"Bebidas"}],total:1}}}:name==="@/shared/api/product-image"?{}:require(name),{URLSearchParams});
 assert.equal((await api.listPurchaseItemCategories())[0].id,"remote");
 assert.match(calls[0].path,/productType=retail/);
 const catalog=compile("modules/menu/products/infrastructure/products-api.ts",name=>name==="@/shared/api/client"?{apiFetch:async(path,options)=>{calls.push({path,options});return {id:"db-uuid"}}}:name==="@/shared/api/product-image"?{}:require(name));
 await catalog.saveCategory({name:"Bebidas",productScope:"retail",sortOrder:0,active:true});
 const call=calls.at(-1);assert.equal(call.path,"categories");assert.equal(call.options.method,"POST");
 const body=JSON.parse(call.options.body);assert.equal(body.productScope,"retail");assert.equal(Object.hasOwn(body,"id"),false);
});
test("el modal de categoría es compartido, valida, evita Enter y duplica cero peticiones",async()=>{
 const zod=compile("shared/forms/zod-resolver.ts",require);
 const schema=compile("modules/menu/products/domain/product-schema.ts",name=>name==="@/shared/forms/zod-resolver"?zod:require(name));
 let form,finish,saved=0;const pending=new Promise(resolve=>{finish=resolve});
 const {CategoryDialog}=compile("modules/menu/products/presentation/category-dialog.tsx",name=>{
  if(name==="react")return {useRef:value=>({current:value})};
  if(name==="react-hook-form")return {useForm:options=>{form=createFormControl(options);return {...form,formState:{errors:{},isSubmitting:false}}}};
  if(name==="../domain/product-schema")return schema;
  if(name.startsWith("@/design-system"))return Object.fromEntries(["Button","FormField","Icon","Input","Select","Dialog"].map(key=>[key,key]));
  return require(name);
 });
 const tree=nodes(CategoryDialog({draft:{name:"",productScope:"retail",sortOrder:0,active:true},retailOnly:true,busy:false,close:()=>{},save:async()=>{saved++;await pending}}));
 assert.equal(tree.some(node=>node.type==="option"&&node.props.value==="prepared"),false);
 let prevented=0,stopped=0;tree.find(node=>node.type==="form").props.onSubmit({preventDefault(){prevented++},stopPropagation(){stopped++}});
 assert.equal(prevented,1);assert.equal(stopped,1);assert.equal(saved,0);
 const button=tree.find(node=>node.type==="Button"&&node.props.children==="Guardar");
 button.props.onClick();await new Promise(resolve=>setImmediate(resolve));assert.equal(saved,0);
 form.setValue("name","Bebidas");for(let i=0;i<4;i++)button.props.onClick();
 await new Promise(resolve=>setImmediate(resolve));assert.equal(saved,1);finish();
 const source=readFileSync(new URL("modules/menu/products/presentation/catalog-manager.tsx",root),"utf8");
 assert.ok(source.includes('import {CategoryDialog} from "./category-dialog"'));
 assert.equal(source.includes("function CategoryDialog("),false);
});
