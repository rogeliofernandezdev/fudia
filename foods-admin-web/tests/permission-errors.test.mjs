import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url),root=new URL("../",import.meta.url);
const read=file=>readFileSync(new URL(file,root),"utf8");
function compile(file,resolve){
  const exports={};
  vm.runInNewContext(ts.transpileModule(read(file),{compilerOptions:{target:ts.ScriptTarget.ES2022,module:ts.ModuleKind.CommonJS}}).outputText,{exports,require:resolve});
  return exports;
}

test("un 403 es un error de permiso; otros errores no",()=>{
  const client=compile("src/shared/api/client.ts",name=>name==="@/shared/session/expire-session"?{expireBrowserSession:()=>{}}:require(name));
  const forbidden=new client.ApiClientError("Tu rol no incluye el permiso «Ver inventario». Pide a un administrador que lo agregue en Usuarios y roles.","forbidden",403);
  assert.equal(client.isPermissionError(forbidden),true);
  assert.equal(client.isPermissionError(new client.ApiClientError("Servicio no disponible","inventory_unavailable",503)),false);
  assert.equal(client.isPermissionError(new Error("forbidden")),false);
  assert.equal(client.isPermissionError(null),false);
  assert.equal(client.missingPermissionLabel(forbidden),"Ver inventario");
  assert.equal(client.missingPermissionLabel(new client.ApiClientError("No tienes permisos para realizar esta acción.","forbidden",403)),"","un backend anterior no nombra el permiso");
});

test("Recetas explica el permiso que falta para los insumos y no ofrece reintentar",()=>{
  const page=read("src/modules/menu/recipes/presentation/recipes-page.tsx");
  assert.ok(page.includes("const ingredientsForbidden=isPermissionError(ingredientDefaults.error);"));
  assert.ok(page.includes('const ingredientsPermission=missingPermissionLabel(ingredientDefaults.error)||"Ver inventario";'),"nombra el permiso aunque el backend responda el texto genérico");
  assert.ok(page.includes("{`Falta el permiso «${ingredientsPermission}»`}"),"el título nombra el permiso que falta");
  assert.ok(page.includes("Pide a un administrador que lo agregue a tu rol en Usuarios y roles."),"dice quién y dónde lo resuelve");
  assert.ok(!page.includes("No tienes permisos para realizar esta acción"),"no repite el texto genérico");
  assert.ok(page.includes("disabled={save.isPending||ingredientsForbidden||"),"«Agregar insumo» no se ofrece si no se pueden listar insumos");
  assert.ok(page.includes('onRetry={isPermissionError(recipes.error)?undefined:'),"el listado tampoco ofrece reintentar un 403");
});
