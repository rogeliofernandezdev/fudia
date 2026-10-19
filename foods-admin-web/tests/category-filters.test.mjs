import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import vm from "node:vm";
import ts from "typescript";

function compile(path,resolve){const exports={};vm.runInNewContext(ts.transpileModule(readFileSync(new URL(`../src/${path}`,import.meta.url),"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,{exports,URLSearchParams,require:resolve});return exports}
const {filterCategories}=compile("modules/menu/products/domain/category-filters.ts");
const categories=[{id:"1",name:"Bebídas",productScope:"retail",active:true},{id:"2",name:"Menú especial",productScope:"prepared",active:false},{id:"3",name:"Combos",productScope:"both",active:true}];

test("filtros de categorías: búsqueda sin acentos, estado y combinación; el uso no se filtra",()=>{
 for(const [query,ids] of [[{search:" BEBIDAS ",status:""},["1"]],[{search:"",status:"inactive"},["2"]],[{search:"menu",status:"inactive"},["2"]],[{search:"",status:""},["1","2","3"]]])assert.deepEqual(Array.from(filterCategories(categories,query),item=>item.id),ids);
});

test("categorías completas: recorre todas las páginas del API antes de filtrar",async()=>{
 const requests=[];
 const {listAllCategories}=compile("modules/menu/products/infrastructure/products-api.ts",name=>name.includes("product-image")?{}:{apiFetch:async path=>{requests.push(path);return requests.length===1?{items:categories.slice(0,2),total:3,pageSize:2}:{items:categories.slice(2),total:3,pageSize:2}}});
 const result=await listAllCategories();
 assert.deepEqual(Array.from(result.items,item=>item.id),["1","2","3"]);
 assert.equal(requests.length,2);
 assert.match(requests[1],/page=2&pageSize=2/);
 assert.equal(result.total,3);
});

test("categorías completas: un fallo posterior no presenta un catálogo parcial como completo",async()=>{
 let page=0;
 const {listAllCategories}=compile("modules/menu/products/infrastructure/products-api.ts",name=>name.includes("product-image")?{}:{apiFetch:async()=>++page===1?{items:[categories[0]],total:3,pageSize:1}:{items:[],total:3,pageSize:1}});
 await assert.rejects(listAllCategories(),/No pudimos cargar todas las categorías/);
});
