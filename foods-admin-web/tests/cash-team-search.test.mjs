import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url);
function compile(file,resolve=require){
 const exports={};
 vm.runInNewContext(ts.transpileModule(readFileSync(new URL(`../src/${file}`,import.meta.url),"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve});
 return exports;
}
const styles=compile("design-system/search-select-styles.ts");
const {CashCollaboratorSearch}=compile("modules/operations/cash/presentation/cash-collaborator-search.tsx",name=>name==="@/design-system/search-select-styles"?styles:require(name));
const options=[{id:"first-id",name:"María Quispe",assignedShiftId:null},{id:"second-id",name:"Milagros Morales",assignedShiftId:null}];
const search=props=>CashCollaboratorSearch({options,value:"",onChange:()=>{},disabled:false,loading:false,...props}).props;

test("busca el nombre completo o parcial sin distinguir mayúsculas ni tildes",()=>{
 const props=search();
 function matches(input){return options.filter(option=>props.filterOption({data:option,label:props.getOptionLabel(option),value:props.getOptionValue(option)},input)).map(option=>option.id)}
 assert.deepEqual(matches(" MARIA "),["first-id"]);
 assert.deepEqual(matches("QUI"),["first-id"]);
 assert.deepEqual(matches("mila"),["second-id"]);
 assert.deepEqual(matches("first-id"),[]);
 assert.deepEqual(matches("inexistente"),[]);
 assert.deepEqual(matches(""),["first-id","second-id"]);
});

test("elegir y limpiar entrega el identificador y conserva el nombre seleccionado",()=>{
 const selected=[];
 const props=search({value:"second-id",onChange:id=>selected.push(id)});
 assert.equal(props.value,options[1]);
 props.onChange(options[0]);
 props.onChange(null);
 assert.deepEqual(selected,["first-id",""]);
 assert.equal(search({value:"removed-id"}).value,null);
 assert.equal(props.isSearchable,true);
 assert.equal(props.isClearable,true);
});

test("distingue carga, lista vacía y ausencia de coincidencias",()=>{
 assert.equal(search({loading:true,disabled:true}).placeholder,"Cargando usuarios…");
 assert.equal(search({loading:true,disabled:true}).isDisabled,true);
 assert.equal(search({options:[]}).placeholder,"No hay usuarios disponibles");
 assert.equal(search().noOptionsMessage({inputValue:"Pedro"}),"No hay colaboradores con ese nombre");
 assert.equal(search().inputId,"cash-team-user");
});

function nodes(tree){const out=[];function visit(node){if(!node||typeof node!=="object")return;if(Array.isArray(node)){node.forEach(visit);return}out.push(node);visit(node.props?.children)}visit(tree);return out}
const collaborators=compile("modules/operations/cash/domain/collaborators.ts");
function dialog(props={}){
 const calls=[],removed=[];
 const {CashTeamDialog}=compile("modules/operations/cash/presentation/cash-advanced-dialogs.tsx",name=>{
  if(name==="react")return{useState:()=>["first-id",()=>{}],useMemo:fn=>fn()};
  if(name==="next/dynamic")return{default:()=>"CashCollaboratorSearch"};
  if(name==="@/design-system")return new Proxy({},{get:(_,key)=>String(key)});
  if(name==="@/design-system/dialog")return{Dialog:"Dialog"};
  if(name==="@/providers")return{useSession:()=>({user:{id:"self"}})};
  if(name.endsWith("/collaborators"))return collaborators;
  if(name.endsWith("cash-schema"))return{};
  return require(name);
 });
 const tree=CashTeamDialog({shift:{status:"open",cashRegisterName:"Caja principal"},users:[],options,loading:false,busyUserId:null,assign:id=>calls.push(id),unassign:id=>removed.push(id),close:()=>{},...props});
 return{all:nodes(tree),calls,removed};
}

test("asigna solo la selección disponible y bloquea dobles acciones o datos cargando",()=>{
 const view=dialog();
 const button=view.all.find(node=>node.type==="Button"&&node.props.children==="Asignar");
 assert.equal(button.props.disabled,false);
 button.props.onClick();
 assert.deepEqual(view.calls,["first-id"]);
 for(const props of [{loading:true},{busyUserId:"first-id"},{options:[{...options[0],assignedShiftId:"other-shift"}]}]){
  const blocked=dialog(props);
  const action=blocked.all.find(node=>node.type==="Button"&&node.props.children==="Asignar");
  assert.equal(action.props.disabled,true);
  action.props.onClick();
  assert.deepEqual(blocked.calls,[]);
 }
});

test("un error remoto ofrece reintento y no muestra controles de asignación",()=>{
 let retries=0;
 const view=dialog({error:"No se pudo conectar",retry:()=>retries++});
 const error=view.all.find(node=>node.type==="ErrorState");
 assert.equal(error.props.text,"No se pudo conectar");
 error.props.onRetry();
 assert.equal(retries,1);
 assert.equal(view.all.some(node=>node.type==="CashCollaboratorSearch"),false);
});

test("el responsable se omite de colaboradores y búsqueda incluso al consultar como otro usuario",()=>{
 const shift={status:"open",cashRegisterName:"Caja principal",openedById:"responsible"};
 const view=dialog({shift,users:[{userId:"responsible",name:"Cajera responsable"},{userId:"helper",name:"Mozo colaborador"}],options:[...options,{id:"responsible",name:"Cajera responsable",assignedShiftId:null}]});
 assert.equal(view.all.some(node=>node.type==="b"&&node.props.children==="Cajera responsable"),false);
 assert.ok(view.all.some(node=>node.type==="b"&&node.props.children==="Mozo colaborador"));
 const search=view.all.find(node=>node.type==="CashCollaboratorSearch");
 assert.equal(search.props.options.some(option=>option.id==="responsible"),false);
 const remove=view.all.filter(node=>node.type==="Button"&&node.props.children==="Quitar");
 assert.equal(remove.length,1);
 remove[0].props.onClick();
 assert.deepEqual(view.removed,["helper"]);
 const ownOnly=dialog({shift,users:[{userId:"responsible",name:"Cajera responsable"}]});
 assert.ok(ownOnly.all.some(node=>node.type==="EmptyState"&&node.props.title==="Sin colaboradores"));
 assert.equal(ownOnly.all.some(node=>node.type==="Button"&&node.props.children==="Quitar"),false);
});
