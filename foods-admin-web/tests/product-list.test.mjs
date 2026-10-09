import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url);
const root=new URL("../src/",import.meta.url);
function compile(path,resolve){
 const exports={};
 vm.runInNewContext(ts.transpileModule(readFileSync(new URL(path,root),"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve});
 return exports;
}
const regional=compile("shared/i18n/regional-format.ts",require);
function nodes(tree,result=[]){if(Array.isArray(tree)){tree.forEach(node=>nodes(node,result));return result}if(tree&&typeof tree==="object"){result.push(tree);nodes(tree.props?.children,result)}return result}
const base={id:"p1",sku:"P1",name:"Ají de gallina",description:"Descripción que no debe mostrarse",categoryName:"Segundos",price:"15",active:true,quantityControl:"portions",availableQuantity:12,inventoryUnit:null};
function mount({items=[base],loading=false,locationId="local-a"}={}){
 const queries=[];
 const {CatalogManager}=compile("modules/menu/products/presentation/catalog-manager.tsx",name=>{
  if(name==="react")return{useState:value=>[value,()=>{}]};
  if(name==="next/dynamic")return{default:()=>"ProductDialog"};
  if(name==="./category-dialog")return{CategoryDialog:"CategoryDialog"};
  if(name==="@tanstack/react-query")return{useQueryClient:()=>({}),useMutation:()=>({}),useQuery:options=>{queries.push(options);return{data:{items:options.queryKey[0]==="products"?items:[],total:items.length},isLoading:options.queryKey[0]==="products"&&loading,isError:false}}};
  if(name==="@/providers/session-context")return{useSession:()=>({can:()=>true,location:{id:locationId,country:"PE"}})};
  if(name==="@/providers/settings-context")return{useSettings:()=>({currencyPosition:"before",currencySymbol:"S/",currencyDecimals:2})};
  if(name==="@/providers/feedback-provider")return{useFeedback:()=>({notify:()=>{}})};
  if(name==="@/shared/hooks/use-debounced-value")return{useDebouncedValue:value=>value};
  if(name==="@/shared/i18n/regional-format")return regional;
  if(name.startsWith("@/design-system"))return Object.fromEntries(["Button","FormField","Input","PageHeader","Pagination","RowActionButton","Select","Status","Icon","Dialog","ConfirmDialog","RemoteModalSkeleton"].map(key=>[key,key]));
  if(name.startsWith("../"))return{};
  return require(name);
 });
 return{view:nodes(CatalogManager()),queries};
}

test("productos: nombre sin descripción, disponibles del backend y paginación global",()=>{
 const {view}=mount();
 assert.deepEqual(view.filter(node=>node.type==="th").map(node=>node.props.children),["PRODUCTO","CATEGORÍA","PRECIO","CONTROL DE CANTIDAD","DISPONIBLES","ESTADO","ACCIONES"]);
 assert.ok(view.some(node=>node.type==="b"&&node.props.children==="Ají de gallina"));
 assert.ok(view.some(node=>node.type==="b"&&node.props.children==="12"));
 assert.equal(view.some(node=>node.props.children===base.description||node.props.children==="Sin descripción"),false);
 assert.equal(view.filter(node=>node.type==="Pagination").length,1);
 const edit=view.find(node=>node.type==="RowActionButton"&&node.props.action==="edit");
 assert.ok(edit,"La edición sigue disponible y conserva la descripción en el borrador");
});

test("productos: cero agotado, inventario con unidad y sin control sin cantidad inventada",()=>{
 const {view}=mount({items:[{...base,availableQuantity:0},{...base,id:"p2",quantityControl:"inventory",availableQuantity:2.5,inventoryUnit:"litros"},{...base,id:"p3",quantityControl:"none",availableQuantity:null}]});
 const values=view.filter(node=>node.type==="b").map(node=>node.props.children);
 assert.ok(values.includes("0"));
 assert.ok(values.includes(`${regional.formatRegionalNumber(2.5,"PE",{maximumFractionDigits:3})} litros`));
 assert.ok(values.includes("—"));
});

test("productos: caché por local y refresco del saldo sin cambiar paginación",()=>{
 const a=mount(),b=mount({locationId:"local-b"});
 const query=a.queries.find(query=>query.queryKey[0]==="products");
 assert.equal(query.queryKey[1],"local-a");
 assert.equal(b.queries.find(query=>query.queryKey[0]==="products").queryKey[1],"local-b");
 assert.equal(query.refetchOnWindowFocus,"always");
 assert.equal(query.refetchInterval,10000);
});

test("productos: skeleton alineado a siete columnas sin segunda línea de descripción",()=>{
 const {view}=mount({loading:true});
 const loading=view.find(node=>typeof node.type==="function"&&node.props.columns===7);
 assert.ok(loading);
 const skeleton=nodes(loading.type(loading.props));
 assert.equal(skeleton.find(node=>node.props.className==="sk-head").props.children.length,7);
 assert.equal(skeleton.some(node=>node.type==="small"),false);
});
