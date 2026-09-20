import test from "node:test";
import assert from "node:assert/strict";
import {existsSync,readFileSync,readdirSync,statSync} from "node:fs";
import {join} from "node:path";

const root=process.cwd();
const read=p=>readFileSync(join(root,p),"utf8");
const walk=dir=>readdirSync(join(root,dir)).flatMap(name=>{
  const p=join(dir,name);return statSync(join(root,p)).isDirectory()?walk(p):[p];
});

test("la arquitectura no usa un contenedor generico de features",()=>{
  assert.equal(existsSync(join(root,"src/components")),false);
  for(const moduleName of ["auth","configuration","context","customers","dashboard","identity","menu","modules","operations","organizations","platform","public-menu","sales","supply"]){
    assert.equal(existsSync(join(root,`src/modules/${moduleName}/index.ts`)),true,`falta index publico: ${moduleName}`);
  }
});

test("app no contiene CSS de negocio",()=>{
  const css=walk("src/app").filter(p=>p.endsWith(".css"));
  assert.deepEqual(css,[]);
});

test("todo CSS tiene owner explicito",()=>{
  const css=walk("src").filter(p=>p.endsWith(".css"));
  const invalid=css.filter(p=>![
    "src/styles/","src/design-system/styles/","src/shell/styles/","src/providers/styles/","src/modules/"
  ].some(prefix=>p.startsWith(prefix)));
  assert.deepEqual(invalid,[]);
});

test("no quedan imports a components ni CSS antiguo de app",()=>{
  const source=walk("src").filter(p=>/\.(ts|tsx)$/.test(p));
  const bad=[];
  for(const p of source){
    const c=read(p);
    if(c.includes("@/components/")||/from ["'][.]{1,2}\/app\//.test(c)||/import ["'][^"']*\/app\/[^"']*\.css["']/.test(c))bad.push(p);
  }
  assert.deepEqual(bad,[]);
});

test("presentation no construye transporte HTTP",()=>{
  const presentation=walk("src/modules").filter(p=>p.includes("/presentation/")&&/\.(ts|tsx)$/.test(p));
  const bad=[];
  for(const p of presentation){
    const source=read(p);
    if(/\bapiFetch\b/.test(source)||/\bfetch\s*\(/.test(source))bad.push(p);
  }
  assert.deepEqual(bad,[]);
});

test("providers shell y pages no construyen transporte HTTP",()=>{
  const source=[
    ...walk("src/providers").filter(p=>/\.(ts|tsx)$/.test(p)),
    ...walk("src/shell").filter(p=>/\.(ts|tsx)$/.test(p)),
    ...walk("src/app").filter(p=>p.endsWith("/page.tsx")),
  ];
  const bad=[];
  for(const p of source){
    const content=read(p);
    if(/\bapiFetch\b/.test(content)||/\bfetch\s*\(/.test(content))bad.push(p);
  }
  assert.deepEqual(bad,[]);
});

test("los estilos especializados se cargan desde su owner",()=>{
  const owners={
    "src/shell/admin-shell.tsx":"./styles/shell.css",
    "src/providers/feedback-provider.tsx":"./styles/feedback.css",
    "src/design-system/confirm-dialog.tsx":"./styles/confirm-dialog.css",
    "src/design-system/maps/mapbox-location-provider.tsx":"../styles/location-map.css",
    "src/modules/dashboard/presentation/dashboard-view.tsx":"./dashboard.css",
    "src/modules/configuration/presentation/configuration-home-page.tsx":"./configuration.css",
    "src/modules/modules/presentation/modules-view.tsx":"./modules.css",
    "src/modules/platform/presentation/platform-onboarding-page.tsx":"./platform-onboarding.css",
    "src/modules/platform/presentation/platform-shell.tsx":"./platform-shell.css",
  };
  for(const [p,css] of Object.entries(owners))assert.ok(read(p).includes(css),`${p} no carga ${css}`);
});

test("el root layout carga solo la base global",()=>{
  const c=read("src/app/layout.tsx");
  const cssImports=[...c.matchAll(/import ["']([^"']+\.css)["']/g)].map(m=>m[1]);
  assert.deepEqual(cssImports,["@/styles/globals.css"]);
});

test("las rutas principales componen modulos",()=>{
  const expected={
    "src/app/(admin)/pedidos/page.tsx":"@/modules/operations",
    "src/app/(admin)/salon/page.tsx":"@/modules/operations",
    "src/app/(admin)/mesas/page.tsx":"@/modules/operations",
    "src/app/(admin)/productos/page.tsx":"@/modules/menu",
    "src/app/(admin)/combos/page.tsx":"@/modules/menu",
    "src/app/(admin)/clientes/page.tsx":"@/modules/customers",
    "src/app/(admin)/locales/page.tsx":"@/modules/organizations",
    "src/app/(admin)/configuracion/usuarios/page.tsx":"@/modules/identity"
  };
  for(const [p,dependency] of Object.entries(expected))assert.ok(read(p).includes(dependency),p);
});

test("no se versionan secretos locales ni artefactos temporales en la raiz admin",()=>{
  assert.equal(existsSync(join(root,".env.local")),false);
  const rootEntries=readdirSync(root);
  assert.deepEqual(rootEntries.filter(name=>/^__(diag|shot|sheet|comanda)/.test(name)),[]);
});

test("se preservan contratos visuales base",()=>{
  const css=read("src/styles/globals.css").replace(/\s+/g,"");
  for(const token of ["--brand-700","--ops-700","--digital-700","--primary-600","--control-height"])assert.ok(css.includes(token),token);
  const shell=read("src/shell/admin-shell.tsx");
  for(const label of ["Reportes","Punto de venta","Carta y productos","Inventario","Compras","CONFIGURACIÓN"])assert.ok(shell.includes(label),label);
});
