import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import vm from "node:vm";
import ts from "typescript";

const read=path=>readFileSync(new URL(`../${path}`,import.meta.url),"utf8");
const exports={};
vm.runInNewContext(ts.transpileModule(read("src/modules/identity/domain/role-requirements.ts"),{compilerOptions:{module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2022}}).outputText,{exports,Map,Set});
const {requiredPermissions,effectivePermissions}=exports;
// Los arreglos nacen en el contexto del vm: se normalizan para compararlos.
const plain=value=>JSON.parse(JSON.stringify(value));

const requirements={
 menus:{recetas:["menu.read","inventory.read"],usuarios:["users.read","organizations.read"],pedidos:["orders.read","menu.read"]},
 permissions:{"users.manage":["users.read","organizations.read"],"users.read":["organizations.read"],"recipes.manage":["menu.read","inventory.read"]},
};
const labels={"menu:recetas":"Recetas","menu:pedidos":"Pedidos","users.manage":"Administrar usuarios y roles","recipes.manage":"Administrar recetas"};

test("roles: las pantallas y acciones marcadas arrastran las acciones que el API exige",()=>{
 const required=requiredPermissions(["recetas"],["recipes.manage"],requirements,labels);
 assert.deepEqual(plain(required.get("inventory.read")),["Recetas","Administrar recetas"],"Recetas necesita ver inventario para listar insumos");
 assert.deepEqual(plain(required.get("menu.read")),["Recetas","Administrar recetas"]);
 // Transitivo y con motivo: administrar usuarios -> ver usuarios -> ver locales.
 const users=requiredPermissions([],["users.manage"],requirements,labels);
 assert.deepEqual(plain([...users.keys()].sort()),["organizations.read","users.read"]);
 assert.deepEqual(plain(users.get("organizations.read")),["Administrar usuarios y roles"]);
 // Varios motivos se acumulan sin repetir.
 assert.deepEqual(plain(requiredPermissions(["recetas","pedidos"],[],requirements,labels).get("menu.read")),["Recetas","Pedidos"]);
 assert.deepEqual(plain(effectivePermissions(["recipes.manage"],required)),["recipes.manage","menu.read","inventory.read"]);
 assert.equal(requiredPermissions(["recetas"],[],undefined).size,0,"sin catálogo no inventa requisitos");
});

test("editor de roles: requeridas marcadas, bloqueadas, con motivo y guardadas",()=>{
 const editor=read("src/modules/identity/presentation/users-roles-manager.tsx");
 assert.match(editor,/requiredPermissions\(v\.menuAccess,v\.permissions,catalog\.data\?\.requirements,labels\)/);
 assert.match(editor,/const selected=permTab==="menu"\?v\.menuAccess:permissions;/,"las requeridas se ven marcadas");
 const tree=read("src/modules/identity/presentation/role-action-tree.tsx");
 assert.match(tree,/disabled=\{Boolean\(reasons\)\}/,"no se pueden quitar");
 assert.match(tree,/Necesario para \{reasons\.join\(", "\)\}/,"explica el motivo");
 assert.match(editor,/save\(\{\.\.\.v,permissions,/,"se guardan junto a las marcadas");
 assert.match(read("src/modules/identity/infrastructure/identity-api.ts"),/requirements\?:RoleRequirements/);
});
