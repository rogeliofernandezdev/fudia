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
const context={user:{platformAdmin:false},modules:{reportes:true,pos:true,pedidos:true,cocina:true,mesas:true,caja:true,productos:true,combos:true,clientes:true,locales:true,fiscal:true,usuarios:true},menuAccess:["*"],permissions:["*"]};
const keys=ctx=>navigation.visibleNavigation(ctx).flatMap(group=>group.items.map(item=>item.module));

test("Emprende presenta Menús y combos sin opciones de módulos ajenos al plan",()=>{
 const modules=keys(context);
 assert.ok(modules.includes("combos"));
 for(const key of ["recetas","inventario","kardex","compras","reservas","facturacion","whatsapp_bot"])assert.equal(modules.includes(key),false,key);
 assert.ok(navigation.visibleNavigation(context).every(group=>group.items.length>0));
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
