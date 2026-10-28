import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import test from "node:test";
import vm from "node:vm";
import ts from "typescript";

const read=file=>readFileSync(new URL(`../${file}`,import.meta.url),"utf8");
function compile(file,imports={}){
 const exports={};
 vm.runInNewContext(ts.transpileModule(read(file),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,{exports,require:name=>imports[name]});
 return exports;
}
const routes=compile("src/shared/routing/page-routes.ts");
const navigation=compile("src/shell/navigation.ts",{"@/shared/routing/page-routes":routes});
const context={user:{platformAdmin:false},modules:{reportes:true,pos:true,pedidos:true,cocina:true,mesas:true,caja:true,productos:true,combos:true,clientes:true,locales:true,fiscal:true,usuarios:true,whatsapp_bot:true},menuAccess:["*"],permissions:["*"]};
const impulso={...context,modules:{...context.modules,reservas:true,recetas:true,inventario:true,kardex:true,compras:true,costos:true}};
const keys=ctx=>navigation.visibleNavigation(ctx).flatMap(group=>group.items.map(item=>item.module));

test("inicio y Dashboard no habilitado llevan al primer acceso efectivo del cajero",()=>{
 const cashier={...context,menuAccess:["caja"],permissions:["cash.read","cash.manage"]};
 assert.equal(navigation.firstAccessibleRoute(cashier),"/cash");
 for(const path of ["/","/dashboard"])assert.equal(navigation.accessibleLandingRedirect(path,cashier),"/cash");
 const pos={...cashier,menuAccess:["caja","pos"],permissions:[...cashier.permissions,"orders.read"]};
 assert.equal(navigation.accessibleLandingRedirect("/dashboard",pos),"/pos");
 assert.equal(navigation.accessibleLandingRedirect("/pos",pos),null);
 assert.equal(navigation.accessibleLandingRedirect("/settings/users",pos),null);
});

test("el inicio respeta revocación de permisos, módulos y colaboradores temporales",()=>{
 const cashier={...context,menuAccess:["pos","caja"],permissions:["orders.read","cash.read"]};
 assert.equal(navigation.accessibleLandingRedirect("/",{...cashier,modules:{...context.modules,pos:false}}),"/cash");
 assert.equal(navigation.accessibleLandingRedirect("/",{...cashier,permissions:["cash.read"]}),"/cash");
 assert.equal(navigation.accessibleLandingRedirect("/dashboard",{...cashier,permissions:[]}),"/no-access");
 assert.equal(navigation.accessibleLandingRedirect("/",{...context,menuAccess:[],permissions:[],cashCollection:true}),"/pos");
});

test("Dashboard se conserva para quien tiene acceso y sin opciones no hay bucles",()=>{
 assert.equal(navigation.accessibleLandingRedirect("/",context),"/dashboard");
 assert.equal(navigation.accessibleLandingRedirect("/dashboard",context),null);
 const none={...context,menuAccess:[],permissions:[]};
 assert.equal(navigation.accessibleLandingRedirect("/",none),"/no-access");
 assert.equal(navigation.accessibleLandingRedirect("/no-access",none),null);
 assert.equal(navigation.accessibleLandingRedirect("/dashboard",{...none,user:{platformAdmin:true},modules:{}}),null);
});

test("puesta en marcha tiene prioridad únicamente cuando es accesible",()=>{
 const setup={...context,setupRequired:true};
 assert.equal(navigation.firstAccessibleRoute(setup),"/settings/getting-started");
 const restricted={...setup,menuAccess:["caja"],permissions:["organizations.manage","cash.read"]};
 assert.equal(navigation.firstAccessibleRoute(restricted),"/cash");
 assert.equal(navigation.firstAccessibleRoute({...setup,modules:{...context.modules,locales:false}}),"/dashboard");
});

test("Emprende presenta venta, atención y Concierge sin abastecimiento ni control",()=>{
 const modules=keys(context);
 for(const key of ["productos","combos","whatsapp_bot"])assert.ok(modules.includes(key),key);
 for(const key of ["inventario","compras","kardex","recetas","reservas","costos","facturacion"])assert.equal(modules.includes(key),false,key);
 assert.ok(navigation.visibleNavigation(context).every(group=>group.items.length>0));
});

test("Impulso suma abastecimiento, recetas, reservas y gastos a todo Emprende",()=>{
 const base=keys(context),modules=keys(impulso);
 for(const key of base)assert.ok(modules.includes(key),key);
 for(const key of ["inventario","compras","kardex","recetas","reservas","costos"])assert.ok(modules.includes(key),key);
});

test("Inventario y Compras de Impulso conservan acceso y acciones propios del rol",()=>{
 for(const [module,permission] of [["inventario","inventory.read"],["compras","purchases.read"]]){
  const item=navigation.navigationGroups.flatMap(group=>group.items).find(item=>item.module===module);
  assert.equal(navigation.canOpenNavigationItem(item,{...impulso,menuAccess:[module],permissions:[permission]}),true);
  assert.equal(navigation.canOpenNavigationItem(item,{...impulso,menuAccess:["pedidos"],permissions:[permission]}),false);
  assert.equal(navigation.canOpenNavigationItem(item,{...impulso,menuAccess:[module],permissions:["orders.read"]}),false);
  assert.equal(navigation.canOpenNavigationItem(item,{...impulso,modules:{...impulso.modules,[module]:false}}),false);
  assert.equal(navigation.canOpenNavigationItem(item,context),false);
 }
});

test("el módulo contratado no reemplaza acceso de rol ni permiso de lectura",()=>{
 const combo=navigation.navigationGroups.flatMap(group=>group.items).find(item=>item.module==="combos");
 assert.equal(navigation.canOpenNavigationItem(combo,{...context,menuAccess:["productos"]}),false);
 assert.equal(navigation.canOpenNavigationItem(combo,{...context,permissions:["users.read"]}),false);
 assert.equal(navigation.canOpenNavigationItem(combo,{...context,modules:{...context.modules,combos:false}}),false);
 assert.equal(navigation.canOpenNavigationItem(combo,{...context,menuAccess:["combos"],permissions:["menu.read"]}),true);
});

test("el administrador de plataforma conserva el catálogo completo",()=>{
 assert.ok(keys({...context,user:{platformAdmin:true},modules:{}}).includes("compras"));
});

test("roles y permisos remotos se vuelven a consultar al cambiar empresa o módulos",()=>{
 const source=read("src/modules/identity/presentation/users-roles-manager.tsx");
 assert.ok(source.includes('queryKey:["roles",...planScope]'));
 assert.ok(source.includes('queryKey:["permission-catalog",session.organization?.id,session.user?.platformAdmin,session.modules]'));
 assert.ok(source.includes("loading=loading||catalog.isLoading"));
 assert.ok(source.includes("if(catalog.isError)return"));
 assert.ok(source.includes("catalog.refetch()"));
});
