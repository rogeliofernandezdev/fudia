import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url);
function compile(file,resolve){
 const exports={};
 const source=readFileSync(new URL("../"+file,import.meta.url),"utf8");
 vm.runInNewContext(ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve});
 return exports;
}
function nodes(tree,result=[]){if(Array.isArray(tree))tree.forEach(node=>nodes(node,result));else if(tree&&typeof tree==="object"){result.push(tree);nodes(tree.props?.children,result)}return result}

test("la sesión separa menú de acciones para cualquier rol y no autoriza por nombre",()=>{
 for(const role of ["Administrador","Cocinero","Mozo","Rol personalizado"]){
  let data={user:{id:"user",name:"Usuario",role,platformAdmin:false},menuAccess:["products"],permissions:[]};
  const {SessionProvider}=compile("src/providers/session-context.tsx",name=>{
   if(name==="react")return{createContext:()=>({Provider:"Provider"}),useEffect(){},useContext(){}};
   if(name==="@tanstack/react-query")return{useQuery:()=>({data,isError:false,isLoading:false})};
   if(name.startsWith("@/"))return{};
   return require(name);
  });
  let value=SessionProvider({children:null}).props.value;
  assert.equal(value.canAccess("products"),true,role);
  assert.equal(value.can("menu.manage"),false,role);
  data={...data,permissions:["menu.read","menu.manage"]};
  value=SessionProvider({children:null}).props.value;
  assert.equal(value.can("menu.manage"),true,role);
  assert.equal(value.can("cash.manage"),false,role);
  assert.equal(value.canAccess("cash"),false,role);
  data={...data,permissions:["*"]};
  value=SessionProvider({children:null}).props.value;
  assert.equal(value.can("cash.manage"),true);
  assert.equal(value.canAccess("cash"),false,"todos los permisos no conceden menú automáticamente");
 }
});

function tablesUI({permissions=[],tab="zones",zones=[{id:"zone",name:"Salón",sortOrder:1,active:true}],states={}}={}){
 let index=0;const calls=[];
 const {TablesManager}=compile("src/modules/operations/tables/presentation/tables-manager.tsx",name=>{
  if(name==="react")return{useState:initial=>{const i=index++;return[i===0?tab:states[i]??initial,()=>{}]}};
  if(name==="@tanstack/react-query")return{useQueryClient:()=>({}),useMutation:()=>({mutate:()=>calls.push("write")}),useQuery:options=>({data:{items:options.queryKey[0]==="tables"?[{id:"table",name:"Mesa 1",seats:2,active:true}]:zones,total:1},isLoading:false,isError:false})};
  if(name==="@/providers/session-context")return{useSession:()=>({can:permission=>permissions.includes(permission)})};
  if(name==="@/providers/feedback-provider")return{useFeedback:()=>({notify(){}})};
  if(name.startsWith("@/design-system"))return Object.fromEntries(["Button","IconButton","PageHeader","Pagination","RowActionButton","Status","Icon","Dialog","ConfirmDialog"].map(key=>[key,key]));
  if(name.endsWith(".css")||name.startsWith("../")||name.startsWith("@/")||["next/image","react-hook-form","qrcode"].includes(name))return{};
  return require(name);
 });
 return{view:nodes(TablesManager()),calls};
}

test("zonas: gestionar mesas no habilita gestionar zonas y lectura conserva datos",()=>{
 for(const permissions of [[],["tables.manage"]]){
  const {view}=tablesUI({permissions});
  assert.equal(view.some(node=>node.type==="Button"&&node.props.icon==="plus"),false);
  assert.equal(view.some(node=>node.type==="RowActionButton"),false);
  assert.equal(view.some(node=>node.type==="th"&&node.props.children==="ACCIONES"),false);
  assert.ok(view.some(node=>node.type==="b"&&node.props.children==="Salón"));
  assert.ok(view.some(node=>node.type==="Pagination"));
  assert.equal(tablesUI({permissions,zones:[]}).view.some(node=>node.type==="Button"&&node.props.icon==="plus"),false);
 }
 const {view}=tablesUI({permissions:["menu.manage"]});
 assert.ok(view.some(node=>node.type==="Button"&&node.props.icon==="plus"));
 assert.deepEqual(view.filter(node=>node.type==="RowActionButton").map(node=>node.props.action),["edit","deactivate"]);
});

test("mesas: editar o agregar no permanece habilitado al perder el permiso",()=>{
 const states={8:true,9:[{name:"Mesa nueva",seats:"2",zone:""}],10:{id:"table",name:"Mesa 1",seats:"2"},11:{id:"zone",name:"Salón"},7:{kind:"zones",id:"zone",name:"Salón"}};
 for(const tab of ["tables","zones"]){
  const {view,calls}=tablesUI({tab,states});
  assert.equal(view.some(node=>node.type==="RowActionButton"||node.type==="IconButton"),false);
  assert.equal(view.some(node=>node.type?.name==="ZoneDialog"),false);
  const confirm=view.find(node=>node.type==="ConfirmDialog");
  assert.equal(confirm.props.open,false);
  confirm.props.onConfirm();assert.equal(calls.length,0);
 }
 assert.deepEqual(tablesUI({tab:"tables",permissions:["tables.manage"]}).view.filter(node=>node.type==="RowActionButton").map(node=>node.props.action),["edit","deactivate"]);
 assert.equal(tablesUI({tab:"tables",permissions:["menu.manage"]}).view.some(node=>node.type==="RowActionButton"),false);
});
