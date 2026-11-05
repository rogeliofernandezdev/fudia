import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import vm from "node:vm";
import ts from "typescript";

const source=path=>readFileSync(new URL(`../src/${path}`,import.meta.url),"utf8");
function compile(path,require=name=>{throw new Error(name)}){const exports={};vm.runInNewContext(ts.transpileModule(source(path),{compilerOptions:{module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2022}}).outputText,{exports,require});return exports}
const plain=value=>JSON.parse(JSON.stringify(value));
const modifiers=compile("modules/menu/products/domain/modifiers.ts");
const option=(name,surcharge="")=>({key:`o-${name||Math.random()}`,name,surcharge});
const group=(patch={})=>({key:"g1",name:"Acompañamiento",required:true,maxSelections:1,options:[option("Arroz blanco"),option("Arroz chaufa","3")],...patch});

test("las opciones válidas no generan errores y el PUT respeta las reglas del API",()=>{
 assert.deepEqual(plain(modifiers.validateModifierGroups([group()])),{});
 const payload=modifiers.modifiersPayload([group({options:[option("Arroz blanco"),option("Arroz chaufa","3.50"),option("")],maxSelections:3}),group({key:"g2",name:"Adicionales",required:false,maxSelections:2,options:[option("Huevo frito","2.5")]})]);
 assert.deepEqual(plain(payload),{groups:[
  {name:"Acompañamiento",required:true,minSelections:1,maxSelections:2,options:[{name:"Arroz blanco",surcharge:"0"},{name:"Arroz chaufa",surcharge:"3.50"}]},
  {name:"Adicionales",required:false,minSelections:0,maxSelections:1,options:[{name:"Huevo frito",surcharge:"2.5"}]},
 ]},"descarta filas vacías, el obligatorio exige 1 y el máximo no supera las opciones");
});

test("la validación marca cada campo con el mismo criterio que el backend",()=>{
 const errors=modifiers.validateModifierGroups([
  group({name:" "}),
  group({key:"g2",name:"acompañamiento"}),
  group({key:"g3",name:"Término",options:[option("Jugoso"),{key:"dup",name:"jugoso",surcharge:""},{key:"bad",name:"A punto",surcharge:"-1"}]}),
  group({key:"g4",name:"Vacío",options:[option("")]}),
 ]);
 assert.equal(errors["g1:name"],"Escribe el nombre del grupo");
 assert.equal(errors["g4:options"],"Agrega al menos una opción");
 assert.equal(errors["dup:name"],"Opción repetida");
 assert.equal(errors["bad:surcharge"],"Importe no válido");
 assert.ok(errors.form);
 const duplicated=modifiers.validateModifierGroups([group(),group({key:"g2"})]);
 assert.equal(duplicated["g2:name"],"Ya hay un grupo con este nombre");
});

test("al editar, la configuración del API vuelve al formulario sin perder recargos",()=>{
 const groups=modifiers.groupsFromConfig({productId:"p",groups:[{id:"g",name:"Término",required:true,minSelections:1,maxSelections:1,options:[{id:"o1",name:"Jugoso",surcharge:"0.00"},{id:"o2",name:"Wagyu",surcharge:"12.00"}]}]});
 assert.deepEqual(plain(groups),[{key:"g",name:"Término",required:true,maxSelections:1,options:[{key:"o1",name:"Jugoso",surcharge:""},{key:"o2",name:"Wagyu",surcharge:"12.00"}]}]);
});

test("la comanda pide las opciones antes de agregar y separa cada combinación en su línea",()=>{
 const view=source("modules/operations/orders/presentation/comanda-view.tsx");
 assert.ok(view.includes("if(p.hasModifiers||current.hasModifiers){setProductEditor({product:p,initialModifiers:[]});return}"),"un plato con opciones abre la hoja en lugar de sumarse");
 assert.ok(view.includes("modifiersKey(line.modifiers)===key"),"misma combinación suma cantidad; otra combinación es otra línea");
 assert.ok(view.includes("<ProductOptionsConfigurator"),"la misma hoja sirve para agregar y para «Cambiar»");
 const catalog=source("modules/operations/salon/presentation/comanda-catalog.tsx");
 assert.ok(catalog.includes("if(pending.length){setInvalid(pending);"),"el botón no se bloquea: marca el grupo pendiente");
 assert.match(source("modules/operations/salon/infrastructure/catalog-api.ts"),/hasModifiers:Boolean\(item\.hasModifiers\)/);
 for(const path of ["modules/operations/salon/infrastructure/salon-api.ts","modules/operations/orders/infrastructure/service-api.ts","modules/operations/orders/domain/manual-order-schema.ts"])
  assert.ok(source(path).includes("modifiers:(line.modifiers??[]).map(modifier=>({groupId:modifier.groupId,optionId:modifier.optionId}))"),`${path} envía las opciones elegidas`);
 assert.ok(source("modules/operations/salon/presentation/salon-manager.tsx").includes("modifiers:(it.modifiers??[]).map(modifier=>({"),"editar un pedido conserva sus opciones");
});
