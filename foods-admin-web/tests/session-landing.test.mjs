import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync,existsSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url);
const source=file=>readFileSync(new URL(`../src/${file}`,import.meta.url),"utf8");
function compile(file,resolve=require){
 const exports={};
 vm.runInNewContext(ts.transpileModule(source(file),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve});
 return exports;
}
const routes=compile("shared/routing/page-routes.ts");
const navigation=compile("shell/navigation.ts",name=>name==="@/shared/routing/page-routes"?routes:require(name));
const cashier={user:{id:"cashier",platformAdmin:false},organization:{id:"org"},location:{id:"local"},modules:{caja:true,pos:true,reportes:true},menuAccess:["caja"],permissions:["cash.read","cash.manage"],setupRequired:false,cashCollection:false,isLoading:false,isError:false,isUnauthorized:false,can:()=>false};
function renderShell(path,changes={}){
 const effects=[],redirects=[];
 const router={replace:href=>redirects.push(href)};
 const resolve=name=>{
  if(name.endsWith(".css"))return {};
  if(name==="react")return {...require(name),useState:value=>[value,()=>{}],useRef:value=>({current:value}),useEffect:effect=>effects.push(effect)};
  if(name==="next/navigation")return {usePathname:()=>path,useRouter:()=>router};
  if(name==="next/link")return {__esModule:true,default:"Link"};
  if(name==="@tanstack/react-query")return {useQueryClient:()=>({clear:()=>{}})};
  if(name==="@/providers/session-context")return {useSession:()=>({...cashier,...changes})};
  if(name==="@/modules/identity")return {useCurrentProfile:()=>({data:{roleNames:["Cajero"]}})};
  if(name==="@/shared/routing/page-routes")return routes;
  if(name==="./navigation")return navigation;
  if(name==="@/design-system")return {FullScreenLoader:"FullScreenLoader"};
  if(name.startsWith("@/")||name.startsWith("."))return new Proxy({},{get:(_,key)=>String(key)});
  return require(name);
 };
 const {AdminShell}=compile("shell/admin-shell.tsx",resolve);
 const tree=AdminShell({children:"PROTECTED_CONTENT"});
 effects.forEach(effect=>effect());
 return {tree,redirects};
}

test("el shell redirige al cajero desde inicio o Dashboard sin montar contenido restringido",()=>{
 for(const path of ["/","/dashboard"]){
  const {tree,redirects}=renderShell(path);
  assert.deepEqual(redirects,["/cash"]);
  assert.equal(tree.type,"FullScreenLoader");
  assert.equal(tree.props.label,"Abriendo tu espacio");
  assert.equal(JSON.stringify(tree).includes("PROTECTED_CONTENT"),false);
 }
});

test("el shell espera contexto válido antes de redirigir",()=>{
 for(const changes of [{isLoading:true},{isError:true},{isUnauthorized:true},{user:null},{location:null},{modules:null}]){
  assert.deepEqual(renderShell("/dashboard",changes).redirects,[]);
 }
});

test("el shell conserva destinos habilitados y no redirige repetidamente",()=>{
 assert.deepEqual(renderShell("/cash").redirects,[]);
 assert.deepEqual(renderShell("/dashboard",{menuAccess:["dashboard"],permissions:["dashboard.read"]}).redirects,[]);
 const none={menuAccess:[],permissions:[]};
 assert.deepEqual(renderShell("/dashboard",none).redirects,["/no-access"]);
 assert.deepEqual(renderShell("/no-access",none).redirects,[]);
});

test("la ruta raíz se atiende dentro del layout autenticado y no fuerza Dashboard",()=>{
 assert.equal(existsSync(new URL("../src/app/page.tsx",import.meta.url)),false);
 assert.equal(compile("app/(admin)/page.tsx").default(),null);
 assert.match(source("app/(admin)/layout.tsx"),/SessionProvider/);
});
