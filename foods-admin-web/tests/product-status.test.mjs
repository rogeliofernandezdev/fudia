import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
import {QueryClient} from "@tanstack/react-query";

const require=createRequire(import.meta.url);
const root=new URL("../src/",import.meta.url);
const product={id:"agua",name:"Agua mineral",description:"",categoryId:null,categoryName:null,price:"4.50",active:false,productType:"retail",quantityControl:"inventory",imageUrl:null,prepMinutes:null,allergens:[],featured:false,costPrice:"2.10"};

function compile(path,resolve){
 const exports={};
 const source=readFileSync(new URL(path,root),"utf8");
 vm.runInNewContext(ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve});
 return exports;
}

function mount({active=false,canManage=true,pending=false,activate=async()=>{}}={}){
 const client=new QueryClient({defaultOptions:{queries:{staleTime:60000,retry:false}}});
 const notifications=[];
 const keys=["products","product-availability","order-catalog","purchase-item-picker","inventory-products"];
 for(const key of keys)client.setQueryData([key],[]);
 const {CatalogManager}=compile("modules/menu/products/presentation/catalog-manager.tsx",name=>{
  if(name==="next/dynamic")return {default:()=>"ProductDialog"};
  if(name==="react")return {useState:value=>[value,()=>{}],useEffect:()=>{}};
  if(name==="@tanstack/react-query")return {
   useQueryClient:()=>client,
   useQuery:options=>({data:{items:options.queryKey[0]==="products"?[{...product,active}]:[],total:1}}),
   useMutation:options=>({isPending:pending,variables:product.id,mutate:async variables=>{
    try{const result=await options.mutationFn(variables);options.onSuccess?.(result,variables)}
    catch(error){options.onError?.(error)}
   }}),
  };
  if(name==="@/providers/feedback-provider")return {useFeedback:()=>({notify:message=>notifications.push(message)})};
  if(name==="@/providers/session-context")return {useSession:()=>({can:()=>canManage})};
  if(name==="@/providers/settings-context")return {useSettings:()=>({currencyDecimals:2,currencySymbol:"S/",currencyPosition:"before"})};
  if(name==="@/shared/hooks/use-debounced-value")return {useDebouncedValue:value=>value};
  if(name==="../infrastructure/products-api")return {setProductActive:activate};
  if(name==="@/design-system")return {RowActionButton:"RowActionButton"};
  if(name.startsWith("@/")||name.startsWith("../"))return {};
  return require(name);
 });
 const nodes=[];
 function visit(node){if(!node||typeof node!=="object")return;if(Array.isArray(node)){node.forEach(visit);return}nodes.push(node);visit(node.props?.children)}
 visit(CatalogManager());
 return {client,notifications,keys,actions:nodes.filter(node=>node.type==="RowActionButton")};
}

test("un producto inactivo ofrece Activar y un producto activo ofrece Desactivar",()=>{
 const inactive=mount();
 assert.deepEqual(inactive.actions.map(node=>node.props.action),["edit","activate"]);
 assert.equal(inactive.actions.find(node=>node.props.action==="activate").props.label,"Activar producto");
 inactive.client.clear();
 const active=mount({active:true});
 assert.deepEqual(active.actions.map(node=>node.props.action),["edit","deactivate"]);
 active.client.clear();
});

test("activar envía el estado y refresca los catálogos después de la confirmación",async()=>{
 const calls=[];
 const view=mount({activate:async(id,active)=>{calls.push([id,active])}});
 await view.actions.find(node=>node.props.action==="activate").props.onClick();
 assert.deepEqual(calls,[[product.id,true]]);
 assert.equal(view.notifications[0].title,"Producto activado");
 for(const key of view.keys)assert.equal(view.client.getQueryState([key]).isInvalidated,true,key);
 view.client.clear();
});

test("un error al activar conserva la caché y comunica el error del servidor",async()=>{
 const view=mount({activate:async()=>{throw new Error("No tienes permisos para realizar esta acción.")}});
 await view.actions.find(node=>node.props.action==="activate").props.onClick();
 assert.equal(view.notifications[0].tone,"danger");
 assert.equal(view.notifications[0].message,"No tienes permisos para realizar esta acción.");
 for(const key of view.keys)assert.equal(view.client.getQueryState([key]).isInvalidated,false,key);
 view.client.clear();
});

test("la acción de activar requiere permiso y comunica su estado ocupado",()=>{
 const readonly=mount({canManage:false});
 assert.equal(readonly.actions.some(node=>node.props.action==="activate"),false);
 readonly.client.clear();
 const busy=mount({pending:true});
 const action=busy.actions.find(node=>node.props.action==="activate");
 assert.equal(action.props.disabled,true);
 assert.equal(action.props["aria-busy"],true);
 assert.equal(action.props.label,"Activando producto");
 busy.client.clear();
});

test("el API de estado no reenvía ni sobrescribe la ficha comercial",async()=>{
 const requests=[];
 const {setProductActive}=compile("modules/menu/products/infrastructure/products-api.ts",name=>{
  if(name==="@/shared/api/client")return {apiFetch:async(path,init)=>{requests.push({path,init})}};
  if(name==="@/shared/api/product-image")return {};
  return require(name);
 });
 await setProductActive(product.id,true);
 assert.equal(requests[0].path,"products/agua/status");
 assert.equal(requests[0].init.method,"PATCH");
 assert.deepEqual(JSON.parse(requests[0].init.body),{active:true});
});
