import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url);
const source=readFileSync(new URL("../src/modules/menu/products/presentation/allergen-field.tsx",import.meta.url),"utf8");
function load(){
 const exports={};
 const resolve=name=>{
  if(name==="react")return {useState:value=>[value,()=>{}]};
  if(name==="react-select/async-creatable")return {default:"CreatableSelect"};
  if(name==="@/design-system")return {FormField:"FormField"};
  return require(name);
 };
 vm.runInNewContext(ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve});
 return exports;
}
const {AllergenField,allergenKey}=load();
const select=props=>AllergenField(props).props.children.props;
const options=[{label:"Sésamo",value:"Sésamo"},{label:"Gluten",value:"Gluten"}];

test("la clave de alérgeno ignora mayúsculas, tildes y espacios, pero distingue la ñ",()=>{
 assert.equal(allergenKey("  Ají   AMARILLO "),"aji amarillo");
 assert.equal(allergenKey("Sésamo"),allergenKey("sesamo"));
 assert.notEqual(allergenKey("Piña"),allergenKey("Pina"));
});

test("Agregar solo aparece si el nombre no existe entre opciones ni seleccionados",()=>{
 const props=select({value:["Ají amarillo"],onChange:()=>{},loadAllergens:async()=>[],createAllergen:async name=>name});
 assert.equal(props.isValidNewOption("sesamo",[],options),false,"equivalente a una opción existente");
 assert.equal(props.isValidNewOption("AJI amarillo",[],[]),false,"ya seleccionado");
 assert.equal(props.isValidNewOption("k",[],[]),false,"muy corto");
 assert.equal(props.isValidNewOption("Kiwi",[],options),true);
 assert.equal(props.formatCreateLabel("  Kiwi  verde "),"Agregar «Kiwi verde»");
 assert.equal(props.noOptionsMessage({inputValue:"aji amarillo"}),"Ya está seleccionado");
});

test("sin permiso de creación no se ofrece Agregar",()=>{
 const props=select({value:[],onChange:()=>{},loadAllergens:async()=>[]});
 assert.equal(props.isValidNewOption("Kiwi",[],[]),false);
});

test("crear agrega el nombre devuelto por el backend a la selección",async()=>{
 const changes=[];
 const props=select({value:["Gluten"],onChange:next=>changes.push(next),loadAllergens:async()=>[],createAllergen:async name=>`${name} (org)`});
 props.onCreateOption("  Kiwi ");
 await new Promise(resolve=>setImmediate(resolve));
 assert.equal(JSON.stringify(changes),JSON.stringify([["Gluten","Kiwi (org)"]]));
});
