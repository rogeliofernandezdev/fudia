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
const categoryFilters=compile("modules/menu/products/domain/category-filters.ts",require);
function nodes(tree,result=[]){if(Array.isArray(tree)){tree.forEach(node=>nodes(node,result));return result}if(tree&&typeof tree==="object"){result.push(tree);nodes(tree.props?.children,result);nodes(tree.props?.action,result)}return result}
const base={id:"p1",sku:"P1",name:"Ají de gallina",description:"Descripción que no debe mostrarse",categoryName:"Segundos",price:"15",active:true,quantityControl:"portions",availableQuantity:12,inventoryUnit:null};
function mount({items=[base],loading=false,locationId="local-a",canManage=true,tab="products",categories=[],states={}}={}){
 const queries=[],updates=[];
 let stateIndex=0;
 const {CatalogManager}=compile("modules/menu/products/presentation/catalog-manager.tsx",name=>{
  if(name==="react")return{useState:value=>{const index=stateIndex++;return[index===0?tab:states[index]??value,next=>updates.push([index,next])]}};
  if(name==="next/dynamic")return{default:()=>"ProductDialog"};
  if(name==="./category-dialog")return{CategoryDialog:"CategoryDialog"};
  if(name==="@tanstack/react-query")return{useQueryClient:()=>({}),useMutation:()=>({}),useQuery:options=>{queries.push(options);return{data:{items:options.queryKey[0]==="products"?items:categories,total:items.length},isLoading:options.queryKey[0]==="products"&&loading,isError:false}}};
  if(name==="@/providers/session-context")return{useSession:()=>({can:permission=>permission==="menu.manage"?canManage:true,canAccess:()=>true,location:{id:locationId,country:"PE"}})};
  if(name==="@/providers/settings-context")return{useSettings:()=>({currencyPosition:"before",currencySymbol:"S/",currencyDecimals:2})};
  if(name==="@/providers/feedback-provider")return{useFeedback:()=>({notify:()=>{}})};
  if(name==="@/shared/hooks/use-debounced-value")return{useDebouncedValue:value=>value};
  if(name==="@/shared/i18n/regional-format")return regional;
  if(name==="../domain/category-filters")return categoryFilters;
  if(name.startsWith("@/design-system"))return Object.fromEntries(["Button","FormField","Input","PageHeader","Pagination","RowActionButton","SearchField","Select","Status","Icon","Dialog","ConfirmDialog","RemoteModalSkeleton","Tabs","TabsBar"].map(key=>[key,key]));
  if(name.startsWith("../"))return{};
  return require(name);
 });
 return{view:nodes(CatalogManager()),queries,updates};
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

const category={id:"category",name:"Segundos",sortOrder:1,active:true,productCount:1,productScope:"prepared"};
test("acceso al menú sin permiso de gestión: productos y categorías no ofrecen acciones",()=>{
 for(const tab of ["products","categories"]){
  const {view}=mount({canManage:false,tab,items:[base,{...base,id:"inactive",active:false}],categories:[category,{...category,id:"inactive-category",active:false}]});
  assert.equal(view.filter(node=>node.type==="RowActionButton"||node.type==="Button").length,0,tab);
  assert.equal(view.some(node=>node.type==="th"&&node.props.children==="ACCIONES"),false);
  assert.equal(view.filter(node=>node.type==="th").length,tab==="products"?6:4);
  assert.equal(view.filter(node=>node.type==="Pagination").length,1);
  assert.ok(view.some(node=>node.type==="b"&&node.props.children===(tab==="products"?base.name:category.name)));
 }
});

test("Administrar carta y productos habilita alta, edición y estado en ambos tabs",()=>{
 for(const tab of ["products","categories"]){
  const {view}=mount({tab,items:[base,{...base,id:"inactive",active:false}],categories:[category,{...category,id:"inactive-category",active:false}]});
  assert.equal(view.filter(node=>node.type==="Button"&&node.props.icon==="plus").length,1);
  assert.deepEqual(view.filter(node=>node.type==="RowActionButton").map(node=>node.props.action),["edit","deactivate","edit","activate"]);
  assert.equal(view.find(node=>node.type==="PageHeader").props.title,"Carta y productos");
 }
});

test("solo lectura: estados vacíos y skeleton no ofrecen crear ni columna de acciones",()=>{
 for(const tab of ["products","categories"]){
  const {view}=mount({canManage:false,tab,items:[]});
  const empty=view.find(node=>typeof node.type==="function"&&node.props.title);
  assert.ok(empty);assert.equal(empty.props.action,undefined);
  assert.equal(nodes(empty.type(empty.props)).some(node=>node.type==="button"),false);
 }
 const {view}=mount({canManage:false,loading:true});
 assert.ok(view.find(node=>typeof node.type==="function"&&node.props.columns===6));
});

test("al perder gestión no permanecen formularios ni confirmaciones editables",()=>{
 const {view}=mount({canManage:false,states:{8:base,9:category,10:{kind:"products",id:base.id,name:base.name}}});
 assert.equal(view.some(node=>node.type==="ProductDialog"||node.type==="CategoryDialog"),false);
 assert.equal(view.find(node=>node.type==="ConfirmDialog").props.open,false);
});

test("categorías: buscador y estado sustituyen el título redundante y reinician la página",()=>{
 const {view,updates}=mount({tab:"categories",categories:[category]});
 assert.equal(view.some(node=>node.props.className==="toolbar-title"),false);
 const search=view.find(node=>node.type==="SearchField");
 const selects=view.filter(node=>node.type==="Select");
 assert.equal(search.props.label,"Buscar categorías por nombre");
 assert.deepEqual(selects.map(node=>node.props["aria-label"]),["Filtrar categorías por estado"]);
 for(const field of [search,...selects])field.props.onChange({target:{value:"active"}});
 assert.equal(updates.filter(([index,value])=>index===6&&value===1).length,2);
});

test("la categoría no expone su uso: sin columna ni filtro «Uso» en ningún plan",()=>{
 const {view}=mount({tab:"categories",categories:[category]});
 assert.equal(view.some(node=>node.type==="th"&&node.props.children==="USO"),false);
 assert.equal(view.some(node=>node.type==="Select"&&node.props["aria-label"]==="Filtrar categorías por uso"),false);
});

test("categorías: los filtros combinados se aplican antes de paginar y conservan el catálogo del selector",()=>{
 const categories=Array.from({length:25},(_,index)=>({...category,id:String(index),name:index===24?"Bebídas especiales":`Categoría ${index}`,productScope:index===24?"retail":"prepared",active:index===24}));
 const {view,queries}=mount({tab:"categories",categories,states:{11:" bebidas ",12:"active"}});
 assert.deepEqual(view.filter(node=>node.type==="tr"&&node.props.className!==undefined).map(node=>node.key),["24"]);
 assert.equal(view.find(node=>node.type==="Pagination").props.total,1);
 assert.deepEqual(Array.from(queries.find(query=>query.queryKey[0]==="categories").queryKey),["categories","catalog"]);
});

test("categorías: sin coincidencias ofrece ajustar filtros, no afirma que el catálogo esté vacío",()=>{
 const {view}=mount({tab:"categories",categories:[category],states:{11:"no existe"}});
 const empty=view.find(node=>typeof node.type==="function"&&node.props.title);
 assert.equal(empty.props.title,"No encontramos categorías");
 assert.equal(empty.props.text,"Ajusta los filtros para ver otros resultados.");
});
