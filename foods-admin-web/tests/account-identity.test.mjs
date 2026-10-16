import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url);
const read=file=>readFileSync(new URL(`../${file}`,import.meta.url),"utf8");
function compile(file,imports={}){
  const exports={};
  vm.runInNewContext(ts.transpileModule(read(file),{compilerOptions:{target:ts.ScriptTarget.ES2020,module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:name=>imports[name]??require(name)});
  return exports;
}
const {AccountRole}=compile("src/shell/account-role.tsx");

test("la cuenta muestra los roles reales, nunca el nombre de empresa como rol",()=>{
  for(const [roles,label] of [
    [["Mesero"],"Mesero"],
    [["Administrador de empresa"],"Administrador"],
    [["Cocinero"],"Cocinero"],
    [["Cajero","Mesero"],"Cajero · Mesero"],
    [["Administrador de empresa","Administrador de plataforma"],"Administrador"],
    [["Encargado de turno"],"Encargado de turno"],
  ]){
    const node=AccountRole({roleNames:roles,platformAdmin:false,loading:false});
    assert.equal(node.type,"small");
    assert.equal(node.props.children,label);
    assert.equal(node.props.title,label);
  }
  assert.equal(AccountRole({roleNames:[],platformAdmin:false,loading:false}).props.children,"Sin rol asignado");
  assert.equal(AccountRole({platformAdmin:true,loading:true}).props.children,"Administrador");
});

test("el rol remoto presenta skeleton y error explícito sin fingir una asignación",()=>{
  const loading=AccountRole({platformAdmin:false,loading:true});
  assert.equal(loading.props["aria-busy"],"true");
  assert.equal(loading.props["aria-label"],"Cargando rol");
  const failed=AccountRole({roleNames:["Rol anterior"],platformAdmin:false,loading:false,error:"Sin conexión"});
  assert.equal(failed.props.children,"Rol no disponible");
  assert.equal(failed.props.title,"Sin conexión");
});

test("cabecera y perfil comparten consulta aislada por usuario, empresa y local",()=>{
  let session={user:{id:"user-a"},organization:{id:"org-a"},location:{id:"local-a"}};
  const getMyProfile=()=>{};
  const {useCurrentProfile}=compile("src/modules/identity/application/use-current-profile.ts",{
    "@tanstack/react-query":{useQuery:options=>options},
    "@/providers/session-context":{useSession:()=>session},
    "../infrastructure/identity-api":{getMyProfile},
  });
  const first=useCurrentProfile();
  assert.deepEqual(Array.from(first.queryKey),["my-profile","user-a","org-a","local-a"]);
  assert.equal(first.queryFn,getMyProfile);
  assert.equal(first.enabled,true);
  assert.equal(first.refetchOnWindowFocus,"always");
  session={...session,location:{id:"local-b"}};
  assert.notDeepEqual(Array.from(useCurrentProfile().queryKey),Array.from(first.queryKey));
  session={user:null,organization:null,location:null};
  assert.equal(useCurrentProfile().enabled,false);
  assert.match(read("src/modules/identity/presentation/profile-page.tsx"),/const profile=useCurrentProfile\(\)/);
});

test("el rol está debajo del usuario en cabecera y submenú, con reintento recuperable",()=>{
  const shell=read("src/shell/admin-shell.tsx");
  assert.equal((shell.match(/<AccountRole \{\.\.\.roleProps\}\/>/g)??[]).length,2);
  assert.doesNotMatch(shell,/platformAdmin\?"Administrador de plataforma":organization\?\.name/);
  assert.match(shell,/profile\.refetch\(\)/);
  assert.match(shell,/\{organization&&<div className="topbar-company"/);
  assert.doesNotMatch(shell,/!isPlatformAdmin&&organization&&<div className="topbar-company"/);
  const identity=read("src/modules/identity/presentation/users-roles-manager.tsx");
  assert.equal((identity.match(/invalidateQueries\(\{queryKey:\["my-profile"\]\}\)/g)??[]).length,4);
});

test("empresa: identidad junto al icono del menú, separada del buscador y del local",()=>{
  const shell=read("src/shell/admin-shell.tsx");
  assert.match(shell,/<header className="topbar"><div className="topbar-identity"><button className="mobile-nav"/);
  assert.match(shell,/\{organization\.name\}<\/div>\}<\/div><NavigationSearch/);
  const css=read("src/shell/styles/shell.css");
  assert.match(css,/\.topbar-identity\{[^}]*display:flex[^}]*align-items:center[^}]*gap:var\(--space-12\)/);
  assert.match(css,/\.topbar-company\{[^}]*font-size:var\(--font-size-16\)[^}]*text-overflow:ellipsis/);
  assert.doesNotMatch(css,/\.topbar-company\{[^}]*flex:1/);
  assert.match(css,/@media \(width<=600px\)[\s\S]*\.topbar-identity\{flex:1;max-width:none;gap:var\(--space-8\)/);
});

test("el aviso inicial comparte la alineación global del título y listado",()=>{
  const shell=read("src/shell/admin-shell.tsx");
  assert.match(shell,/<div className="content">\{setupRequired[\s\S]*className="setup-reminder"/);
  const css=read("src/shell/styles/shell.css");
  assert.equal((css.match(/\.setup-reminder\{/g)??[]).length,1);
  assert.match(css,/\.setup-reminder\{[^}]*margin:0 0 var\(--space-21\)/);
  assert.match(css,/\.setup-reminder>span\{[^}]*min-width:0/);
  assert.match(css,/\.setup-reminder>svg\{flex-shrink:0\}/);
});
