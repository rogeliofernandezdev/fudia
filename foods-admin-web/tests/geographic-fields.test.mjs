import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url);
const catalog={"": [{code:"15",name:"LIMA",label:"LIMA"},{code:"16",name:"LORETO",label:"LORETO"}],"15":[{code:"1501",name:"LIMA",label:"LIMA · LIMA"}],"1501":[{code:"150101",name:"LIMA",label:"LIMA · LIMA · LIMA"}]};
function nodes(value,result=[]){if(Array.isArray(value))value.forEach(child=>nodes(child,result));else if(value&&typeof value==="object"){result.push(value);nodes(value.props?.children,result)}return result}
function mount({allowBroad=false,initial="",failed="none",loading=false,empty=false}={}){
 let cursor=0,value=initial,retries=0;const state=[],selected=[],keys=[];
 const exports={};
 const source=readFileSync(new URL("../src/modules/organizations/presentation/geographic-fields.tsx",import.meta.url),"utf8");
 vm.runInNewContext(ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,target:ts.ScriptTarget.ES2022}}).outputText,{exports,require:name=>{
  if(name==="react")return {useState:initial=>{const i=cursor++;if(!(i in state))state[i]=initial;return [state[i],next=>{state[i]=next}]}};
  if(name==="@tanstack/react-query")return {useQuery:options=>{const parent=options.queryKey[1],enabled=options.enabled!==false;keys.push(parent);return {data:enabled?(empty?[]:catalog[parent]??[]):undefined,isError:enabled&&failed===parent,error:failed===parent?new Error("Sin conexión"):null,isPending:enabled&&loading,isFetching:false,refetch:()=>{retries++}}}};
  if(name==="@/design-system")return Object.fromEntries(["EmptyState","ErrorState","FormField","Select","Skeleton"].map(key=>[key,key]));
  if(name.endsWith(".css")||name.includes("geography-api"))return {};
  return require(name);
 }});
 return {render(){cursor=0;return nodes(exports.GeographicFields({value,allowBroad,onChange:(code,area)=>{value=code;selected.push(area)}}))},get value(){return value},get retries(){return retries},selected,keys};
}

test("local: departamento y provincia limpian descendientes y solo persisten un distrito",()=>{
 const ui=mount();let fields=ui.render().filter(node=>node.type==="Select");
 fields[0].props.onChange({target:{value:"15"}});assert.equal(ui.value,"");
 fields=ui.render().filter(node=>node.type==="Select");assert.equal(fields[1].props.disabled,false);
 fields[1].props.onChange({target:{value:"1501"}});assert.equal(ui.value,"");
 fields=ui.render().filter(node=>node.type==="Select");fields[2].props.onChange({target:{value:"150101"}});assert.equal(ui.value,"150101");
 fields=ui.render().filter(node=>node.type==="Select");fields[0].props.onChange({target:{value:"16"}});
 fields=ui.render().filter(node=>node.type==="Select");assert.equal(ui.value,"");assert.equal(fields[1].props.value,"");assert.equal(fields[2].props.disabled,true);
});


test("catálogo: carga, error recuperable y vacío explícito",()=>{
 assert.ok(mount({loading:true}).render().some(node=>node.type==="Skeleton"));
 const failed=mount({failed:""}),error=failed.render().find(node=>node.type==="ErrorState");
 assert.equal(error.props.text,"Sin conexión");error.props.onRetry();assert.equal(failed.retries,1);
 assert.ok(mount({empty:true}).render().some(node=>node.type==="EmptyState"));
});

