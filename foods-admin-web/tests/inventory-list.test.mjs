import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
import postcss from "postcss";

const require=createRequire(import.meta.url);
const root=new URL("../src/",import.meta.url);
function compile(path,resolve){
 const exports={};
 vm.runInNewContext(ts.transpileModule(readFileSync(new URL(path,root),"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve});
 return exports;
}
const regional=compile("shared/i18n/regional-format.ts",require);
function nodes(tree,result=[]){if(Array.isArray(tree)){tree.forEach(node=>nodes(node,result));return result}if(tree&&typeof tree==="object"){result.push(tree);nodes(tree.props?.children,result)}return result}
const base={inventoryItemId:"water",name:"Agua mineral San Luis",kind:"product",unit:"botella",quantity:"0",minimumStock:"2",reorderPoint:"2",optimalStock:"5",averageUnitCost:"0",stockValue:"0",status:"out"};
function mount(items){
 const {InventoryPage}=compile("modules/supply/inventory/presentation/inventory-page.tsx",name=>{
  if(name==="react")return {useState:value=>[value,()=>{}]};
  if(name==="next/dynamic")return {default:()=>"DynamicDialog"};
  if(name.endsWith(".css")||name.startsWith("../"))return {};
  if(name==="@tanstack/react-query")return {useQueryClient:()=>({}),useMutation:()=>({}),useQuery:options=>({data:{items:options.queryKey[0]==="inventory"?items:[],total:items.length},isLoading:false,isError:false})};
  if(name==="@/providers")return {useFeedback:()=>({notify:()=>{}})};
  if(name==="@/providers/session-context")return {useSession:()=>({can:()=>true,location:{id:"local",country:"PE"}})};
  if(name==="@/shared/hooks/use-debounced-value")return {useDebouncedValue:value=>value};
  if(name==="@/shared/i18n/regional-format")return regional;
  if(name.startsWith("@/design-system"))return Object.fromEntries(["Button","Icon","Input","PageHeader","Pagination","RemoteModalSkeleton","RowActionButton","Status","Dialog"].map(key=>[key,key]));
  return require(name);
 });
 return nodes(InventoryPage());
}

test("inventario muestra mínimo y reorden juntos, con cero y cantidades fraccionarias",()=>{
 const items=[base,{...base,inventoryItemId:"cola",minimumStock:"5",reorderPoint:"5"},{...base,inventoryItemId:"zero",minimumStock:"0",reorderPoint:"0"},{...base,inventoryItemId:"fraction",minimumStock:"1.125",reorderPoint:"2.5"}];
 const view=mount(items);
 const rows=view.filter(node=>node.type==="tr"&&node.key!==null);
 assert.equal(rows.length,items.length);
 rows.forEach((row,index)=>{
  const cell=row.props.children[4];
  assert.equal(cell.type,"td");
  assert.equal(cell.props.children.type,"b");
  assert.equal(nodes(cell).some(node=>node.type==="small"||node.type==="br"),false);
  assert.equal(cell.props.children.props.children.join(""),`${regional.formatRegionalNumber(Number(items[index].minimumStock),"PE",{maximumFractionDigits:3})} / ${regional.formatRegionalNumber(Number(items[index].reorderPoint),"PE",{maximumFractionDigits:3})}`);
  assert.equal(row.props.children.length,9);
 });
 assert.equal(view.filter(node=>node.type==="Pagination").length,1);
});

test("la tabla conserva el nowrap global y desplazamiento horizontal en móvil sin sobrescribir textos secundarios",()=>{
 const globals=postcss.parse(readFileSync(new URL("styles/globals.css",root),"utf8"));
 const inventory=postcss.parse(readFileSync(new URL("modules/supply/inventory/presentation/inventory.css",root),"utf8"));
 const rules=[];globals.walkRules(rule=>rules.push(rule));inventory.walkRules(rule=>rules.push(rule));
 const value=(selector,prop)=>rules.find(rule=>rule.selector===selector&&rule.parent.type==="root")?.nodes.find(node=>node.prop===prop)?.value;
 assert.equal(value("table","white-space"),"nowrap");
 assert.equal(value("td small","display"),"block");
 assert.equal(value(".inventory-table-wrap","overflow-x"),"auto");
});
