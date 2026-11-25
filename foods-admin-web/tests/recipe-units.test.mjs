import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
const require=createRequire(import.meta.url);
function compile(path,resolve=require){const exports={};vm.runInNewContext(ts.transpileModule(readFileSync(new URL("../src/"+path,import.meta.url),"utf8"),{compilerOptions:{target:ts.ScriptTarget.ES2020,module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve,URLSearchParams});return exports}
const unitsApi=compile("shared/units/measurement-unit.ts");
const units=[
 {code:"kg",name:"Kilogramo",family:"mass",referenceFactor:"1000",inventoryDefault:true,recipeDefault:false},
 {code:"g",name:"Gramo",family:"mass",referenceFactor:"1",recipeDefault:true},
 {code:"l",name:"Litro",family:"volume",referenceFactor:"1000",inventoryDefault:true},
 {code:"ml",name:"Mililitro",family:"volume",referenceFactor:"1",recipeDefault:true},
 {code:"und",name:"Unidad",family:"count",referenceFactor:"1",inventoryDefault:true,recipeDefault:true},
 {code:"botella",name:"Botella",family:"count",referenceFactor:"1"},
 {code:"personalizada",name:"Personalizada",family:"custom",referenceFactor:"1"},
];
const find=code=>units.find(unit=>unit.code===code);
test("receta: unidades compatibles y sugerencia salen del catálogo remoto",()=>{
 for(const [base,expected,suggested]of[["kg",["kg","g"],"g"],["l",["l","ml"],"ml"],["botella",["botella"],"botella"],["personalizada",["personalizada"],"personalizada"]]){
  assert.deepEqual(Array.from(unitsApi.compatibleMeasurementUnits(units,base),unit=>unit.code),expected);
  assert.equal(unitsApi.suggestedRecipeUnit(units,base),suggested);
 }
 assert.deepEqual(Array.from(unitsApi.compatibleMeasurementUnits([],"kg")),[]);
});
test("receta: cambiar gramos por kilos conserva la cantidad exacta y admite medio gramo",()=>{
 for(const [value,from,to,expected]of[["200","g","kg","0.2"],["0.2","kg","g","200"],["0.5","g","kg","0.0005"],["0.0005","kg","g","0.5"],["40","ml","l","0.04"],["0.001","g","kg","0.000001"]]){
  assert.equal(unitsApi.convertMeasurementQuantity(value,find(from),find(to)),expected);
 }
 assert.equal(unitsApi.convertMeasurementQuantity("1",find("kg"),find("l")),"");
 assert.equal(unitsApi.convertMeasurementQuantity("1",find("botella"),find("und")),"");
});
test("receta: guardar transmite cantidad y unidad elegidas sin confiar la conversión al cliente",async()=>{
 const requests=[];const api=compile("modules/menu/recipes/infrastructure/recipes-api.ts",name=>name==="@/shared/api/client"?{apiFetch:(path,options)=>{requests.push({path,body:options?.body?JSON.parse(options.body):null});return Promise.resolve({})}}:require(name));
 await api.listRecipeUnits();
 assert.equal(requests[0].path,"inventory/units");
 await api.saveRecipe({productId:"product",yieldQuantity:"1",notes:"",active:true,items:[{inventoryItemId:"item",quantity:"0.5",consumptionUnit:"g",wastePercent:"10"}]});
 assert.deepEqual(requests[1].body.items,[{inventoryItemId:"item",consumptionUnit:"g",consumptionQuantity:0.5,wastePercent:10}]);
 await api.saveRecipe({productId:"product",yieldQuantity:"1",notes:"",active:true,items:[{inventoryItemId:"item",quantity:"0.0005",wastePercent:"0"}]});
 assert.deepEqual(requests[2].body.items,[{inventoryItemId:"item",quantity:0.0005,wastePercent:0}]);
});

function mount({unitError=false,emptyUnits=false}={}){
 const state=[],saved=[],notices=[];let cursor=0;
 const recipe={id:"recipe",productId:"product",productName:"Plato de prueba",yieldQuantity:"1.000",notes:"",active:true,items:[{inventoryItemId:"meat",name:"Carne",unit:"kg",quantity:"0.200000",consumptionUnit:"g",consumptionQuantity:"200.000000",wastePercent:"0.0000"}]};
 const page=compile("modules/menu/recipes/presentation/recipes-page.tsx",name=>{
  if(name==="react")return{useState:initial=>{const index=cursor++;if(!(index in state))state[index]=initial;return[state[index],value=>{state[index]=typeof value==="function"?value(state[index]):value}]}};
  if(name==="next/dynamic")return{default:()=>"AsyncSelect"};
  if(name==="@/providers")return{useSession:()=>({can:()=>true,organization:{id:"org"}}),useFeedback:()=>({notify:value=>notices.push(value)})};
  if(name.startsWith("@/design-system"))return Object.fromEntries(["Button","ConfirmDialog","EmptyState","ErrorState","FieldLabel","Icon","Input","Select","PageHeader","RowActionButton","Status","Textarea","ModalHeader","ModalFooter","ModalForm","Skeleton","DataTable","Dialog"].map(key=>[key,key]).concat([["createLookupSelectStyles",()=>({})]]));
  if(name==="./recipes.css")return{};
  if(name==="@/shared/hooks/use-client-pagination")return{useClientPagination:items=>({rows:items,pagination:{page:1,size:10,total:items.length,onPage:()=>{},onSize:()=>{}}})};
  if(name==="@/shared/stock-status")return{stockStatus:(quantity,minimum)=>{if(!(quantity>0))return{key:"out",tone:"gray",label:"Sin stock",hint:""};if(minimum>0&&quantity<=minimum)return{key:"low",tone:"orange",label:"Stock bajo",hint:""};return{key:"ok",tone:"green",label:"Saludable",hint:""}}};
  if(name==="@/shared/api/client")return{isPermissionError:()=>false,missingPermissionLabel:()=>""};
  if(name==="@/shared/units/measurement-unit")return unitsApi;
  if(name==="../infrastructure/recipes-api")return{getRecipe:async()=>recipe,saveRecipe:draft=>{saved.push(draft)},listRecipeInventory:()=>{},listRecipes:()=>{},listRecipeUnits:()=>{}};
  if(name==="@tanstack/react-query")return{useQueryClient:()=>({invalidateQueries:()=>{}}),useMutation:options=>({isPending:false,mutate:async draft=>options.mutationFn(draft)}),useQuery:({queryKey})=>queryKey[0]==="recipes"?{data:{items:[recipe]},isLoading:false,isError:false}:queryKey[0]==="recipe-units"?{data:{items:emptyUnits?[]:units},isLoading:false,isError:unitError,error:new Error("Sin conexión"),refetch:()=>{}}:{data:{items:[]},isLoading:false,isError:false}};
  return require(name);
 });
 const nodes=tree=>{const result=[];function visit(node){if(Array.isArray(node)){node.forEach(visit);return}if(node&&typeof node==="object"){result.push(node);visit(node.props?.children);visit(node.props?.action)}}visit(tree);return result};
 const render=()=>{cursor=0;return nodes(page.RecipesPage())};
 return{render,saved,open:async()=>{render().find(node=>node.type==="RowActionButton"&&node.props.action==="edit").props.onClick();await new Promise(resolve=>setImmediate(resolve));assert.equal(notices.length,0,JSON.stringify(notices))}};
}

test("receta: editar muestra 200 g sin relleno y cambiar a kg conserva 0.2",async()=>{
 const ui=mount();await ui.open();
 const quantity=()=>ui.render().find(node=>node.type==="Input"&&node.props["aria-label"]==="Cantidad del insumo 1");
 const unit=()=>ui.render().find(node=>node.type==="Select"&&node.props["aria-label"]==="Unidad de consumo del insumo 1");
 assert.equal(quantity().props.value,"200");assert.equal(unit().props.value,"g");
 unit().props.onChange({target:{value:"kg"}});assert.equal(quantity().props.value,"0.2");
 unit().props.onChange({target:{value:"g"}});assert.equal(quantity().props.value,"200");
 ui.render().find(node=>node.type==="ModalForm").props.onSubmit({preventDefault(){}});
 assert.equal(ui.saved[0].items[0].quantity,"200");assert.equal(ui.saved[0].items[0].consumptionUnit,"g");
});

test("receta: error o catálogo vacío bloquean guardar sin perder lo cargado",async()=>{
 for(const options of [{unitError:true},{emptyUnits:true}]){
  const ui=mount(options);await ui.open();const tree=ui.render();
  assert.equal(tree.find(node=>node.type==="Button"&&node.props.type==="submit").props.disabled,true);
  assert.equal(tree.find(node=>node.type==="Input"&&node.props["aria-label"]==="Cantidad del insumo 1").props.value,"200");
  assert.ok(tree.some(node=>node.type===(options.unitError?"ErrorState":"EmptyState")));
 }
});
