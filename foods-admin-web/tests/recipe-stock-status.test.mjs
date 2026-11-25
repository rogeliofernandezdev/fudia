import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url);
const read=path=>readFileSync(new URL(`../${path}`,import.meta.url),"utf8");
function compile(path){
 const exports={};
 vm.runInNewContext(ts.transpileModule(read(path),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,{exports,require,Date,Math});
 return exports;
}

test("receta: los insumos muestran su estado de stock con la matriz aprobada",()=>{
 const status=compile("src/shared/stock-status.ts");
 assert.deepEqual(Object.values(status.stockStatus(10,2)).slice(0,3),["ok","green","Saludable"]);
 assert.deepEqual(Object.values(status.stockStatus(2,2)).slice(0,3),["low","orange","Stock bajo"]);
 assert.deepEqual(Object.values(status.stockStatus(0,2)).slice(0,3),["out","red","Sin stock"]);
 assert.equal(status.stockStatus(1.5,0).key,"ok","sin mínimo configurado no puede aparecer bajo");
 assert.equal(status.stockStatus(NaN,2).key,"out","dato inválido se trata como sin stock");
 const page=read("src/modules/menu/recipes/presentation/recipes-page.tsx");
 assert.match(page,/stockStatus\(/,"el editor de recetas usa el mapeo compartido");
 assert.match(page,/formatOptionLabel=\{\(option,meta\)/,"el desplegable de insumos muestra el estado por opción");
 assert.match(page,/recipe-ingredient-stock/,"la fila del insumo elegido muestra su estado");
 assert.match(read("src/shared/stock-status.ts"),/Reponlo para poder vender/,"sin stock avisa que debe reponer antes de vender");
 assert.match(read("src/shared/stock-status.ts"),/repone pronto para no bloquear pedidos/,"stock bajo avisa el riesgo de bloquear pedidos");
 const api=read("src/modules/menu/recipes/infrastructure/recipes-api.ts");
 assert.match(api,/minimumStock/,"el catálogo de insumos trae el mínimo para calcular el estado");
});
