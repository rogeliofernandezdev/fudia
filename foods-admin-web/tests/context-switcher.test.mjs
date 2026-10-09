import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
import postcss from "postcss";

const require=createRequire(import.meta.url);
const root=new URL("../src/modules/context/presentation/",import.meta.url);
const organizations=[{id:"org-a",name:"Foods Restaurante",active:true,locationCount:2},{id:"org-b",name:"Otra empresa",active:true,locationCount:1}];
const locations=[{id:"local-a",name:"Local principal",code:"LP",active:true},{id:"local-b",name:"Terraza",code:"T",active:true}];
function compile(file,resolve){
 const exports={};
 vm.runInNewContext(ts.transpileModule(readFileSync(new URL(file,root),"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve});
 return exports;
}
function nodes(tree,result=[]){if(Array.isArray(tree)){tree.forEach(node=>nodes(node,result));return result}if(tree&&typeof tree==="object"){result.push(tree);nodes(tree.props?.children,result)}return result}
const {ContextSwitcherPanel}=compile("context-switcher-panel.tsx",name=>name==="@/design-system"?Object.fromEntries(["Button","Select","FormField","Icon","IconButton"].map(key=>[key,key])):require(name));
const props={id:"context-panel",platformAdmin:true,organizationId:"org-a",locationId:"local-b",organizations,locations,loading:false,busy:false,canApply:true,onOrganizationChange:()=>{},onLocationChange:()=>{},onApply:()=>{},onClose:()=>{},onRetry:()=>{}};

test("contexto: campos y acciones usan primitivas homologadas con nombre accesible",()=>{
 const view=nodes(ContextSwitcherPanel(props));
 assert.equal(view.find(node=>node.props.role==="dialog").props["aria-labelledby"],"context-panel-title");
 assert.deepEqual(view.filter(node=>node.type==="FormField").map(node=>node.props.label),["Empresa","Local"]);
 assert.equal(view.filter(node=>node.type==="Select").length,2);
 const button=view.find(node=>node.type==="Button");
 assert.equal(button.props.children,"Aplicar");
 assert.equal(button.props.type,"submit");
 assert.equal(button.props.disabled,false);
 assert.equal(button.props.kind,undefined);
 assert.equal(view.find(node=>node.type==="IconButton").props.label,"Cerrar selector");
});

test("contexto: aplicar y seleccionar están bloqueados durante la petición o sin cambio",()=>{
 let calls=0;
 for(const changes of [{canApply:false},{busy:true},{canApply:false,busy:true}]){
  const view=nodes(ContextSwitcherPanel({...props,...changes,onApply:()=>calls++}));
  let prevented=false;
  view.find(node=>node.type==="form").props.onSubmit({preventDefault:()=>{prevented=true}});
  assert.equal(prevented,true);
  assert.equal(view.find(node=>node.type==="Button").props.disabled,true);
  if(changes.busy){
   assert.equal(view.find(node=>node.type==="IconButton").props.disabled,true);
   assert.equal(view.filter(node=>node.type==="Select").every(node=>node.props.disabled),true);
   assert.equal(view.find(node=>node.type==="Button").props.children,"Cambiando…");
  }
 }
 assert.equal(calls,0);
 nodes(ContextSwitcherPanel({...props,onApply:()=>calls++})).find(node=>node.type==="form").props.onSubmit({preventDefault:()=>{}});
 assert.equal(calls,1);
});

test("contexto: skeleton, error recuperable y vacíos no dejan acciones habilitadas",()=>{
 const loading=nodes(ContextSwitcherPanel({...props,loading:true}));
 assert.ok(loading.some(node=>node.props["aria-label"]==="Cargando opciones"));
 assert.equal(loading.some(node=>node.type==="Select"||node.type==="Button"),false);
 let retries=0;
 const failed=nodes(ContextSwitcherPanel({...props,error:"Sin conexión",onRetry:()=>retries++}));
 assert.ok(failed.some(node=>node.props.role==="alert"));
 failed.find(node=>node.type==="Button").props.onClick();assert.equal(retries,1);
 assert.ok(nodes(ContextSwitcherPanel({...props,organizations:[]})).some(node=>node.props.children==="No hay empresas disponibles."));
 const empty=nodes(ContextSwitcherPanel({...props,locations:[],canApply:false}));
 assert.ok(empty.some(node=>node.props.children==="No hay locales disponibles."));
 assert.equal(empty.find(node=>node.type==="FormField"&&node.props.label==="Local").props.children.props.disabled,true);
 const tenant=nodes(ContextSwitcherPanel({...props,platformAdmin:false}));
 assert.deepEqual(tenant.filter(node=>node.type==="FormField").map(node=>node.props.label),["Local"]);
});

function controller({platformAdmin=true,busy=false,error=null,loading=false}={}){
 const state=[true,"",""];let cursor=0;const calls=[];
 const {ContextSwitcher}=compile("context-switcher.tsx",name=>{
  if(name.endsWith(".css"))return{};
  if(name==="react")return{useState:()=>{const index=cursor++;return[state[index],value=>{state[index]=value}]},useEffect:()=>{},useRef:()=>({current:null}),useId:()=>"context-panel"};
  if(name==="@tanstack/react-query")return{useQueryClient:()=>({}),useMutation:()=>({isPending:busy,mutate:input=>calls.push(input)}),useQuery:({queryKey})=>({data:{items:queryKey[0]==="organizations"?organizations:queryKey[0]==="org-locations"&&queryKey[1]==="org-b"?[{...locations[0],id:"local-c"}]:locations},isLoading:loading,isError:Boolean(error),error,refetch:()=>{}})};
  if(name==="@/providers/session-context")return{useSession:()=>({user:{platformAdmin},organization:{id:"org-a",name:"Foods Restaurante"},location:{id:"local-a",name:"Local principal"}})};
  if(name==="@/providers/feedback-provider")return{useFeedback:()=>({notify:()=>{}})};
  if(name==="./context-switcher-panel")return{ContextSwitcherPanel:"Panel"};
  if(name==="@/design-system/icons")return{Icon:"Icon"};
  if(name.startsWith("@/")||name.startsWith("../"))return{};
  return require(name);
 });
 const render=()=>{cursor=0;return nodes(ContextSwitcher())};
 return{render,calls,state};
}

test("contexto: cambiar empresa limpia el local y solo envía destinos válidos distintos",()=>{
 const host=controller();let panel=host.render().find(node=>node.type==="Panel");
 assert.equal(panel.props.canApply,false);panel.props.onApply();assert.equal(host.calls.length,0);
 panel.props.onOrganizationChange("org-b");panel=host.render().find(node=>node.type==="Panel");
 assert.equal(panel.props.locationId,"");assert.equal(panel.props.canApply,false);
 panel.props.onLocationChange("local-b");panel=host.render().find(node=>node.type==="Panel");
 assert.equal(panel.props.canApply,false);
 panel.props.onLocationChange("local-c");panel=host.render().find(node=>node.type==="Panel");
 assert.equal(panel.props.canApply,true);panel.props.onApply();
 assert.equal(JSON.stringify(host.calls),JSON.stringify([{organizationId:"org-b",locationId:"local-c"}]));
 for(const options of [{busy:true},{error:new Error("Sin conexión")},{loading:true}]){
  const blocked=controller(options);blocked.state[2]="local-b";
  blocked.render().find(node=>node.type==="Panel").props.onApply();assert.equal(blocked.calls.length,0);
 }
 const tenant=controller({platformAdmin:false});tenant.state[2]="local-b";
 tenant.render().find(node=>node.type==="Panel").props.onApply();
 assert.equal(JSON.stringify(tenant.calls),JSON.stringify([{organizationId:"org-a",locationId:"local-b"}]));
});

test("contexto: usuarios de empresa mantienen el local activo como contexto",()=>{
 for(const platformAdmin of [true,false]){
  const host=controller({platformAdmin});
  const view=host.render();
  assert.ok(view.some(node=>node.type==="b"&&node.props.children===(platformAdmin?"Foods Restaurante":"Local principal")));
  assert.ok(view.some(node=>node.type==="small"&&node.props.children===(platformAdmin?"EMPRESA":"LOCAL ACTIVO")));
 }
});

test("contexto: CSS único, panel acotado y movimiento reducido",()=>{
 const css=postcss.parse(readFileSync(new URL("context-switcher.css",root),"utf8"));const seen=new Set();
 css.walkRules(rule=>{if(rule.parent.type==="atrule"&&rule.parent.name==="keyframes")return;for(const selector of rule.selectors){const key=(rule.parent.params??"root")+selector;assert.equal(seen.has(key),false,key);seen.add(key)}});
 const source=css.toString();
 assert.doesNotMatch(source,/background:var\(--brand-/);
 assert.match(source,/\.context-popover\{[^}]*max-height:[^}]*overflow:auto/);
 assert.match(source,/@media\(max-width:600px\)[\s\S]*\.context-popover\{[^}]*position:fixed/);
 assert.match(source,/@media\(prefers-reduced-motion:reduce\)/);
});
