import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
import postcss from "postcss";

const require=createRequire(import.meta.url);
const read=file=>readFileSync(new URL(`../${file}`,import.meta.url),"utf8");
const exports={};
vm.runInNewContext(ts.transpileModule(read("src/design-system/form-controls.tsx"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:name=>name.endsWith(".css")?{}:name==="./icons"?{Icon:"Icon"}:require(name)});

test("cantidad suave conserva límites, quitar explícito y bloqueo, sin cambiar la apariencia por defecto",()=>{
 const changes=[];let removed=0;
 const base={value:1,min:1,max:3,label:"Caldo",onChange:value=>changes.push(value),onRemove:()=>removed++};
 assert.equal(exports.NumberStepper(base).props.className,"ds-stepper");
 const soft=exports.NumberStepper({...base,appearance:"soft"});assert.equal(soft.props.className,"ds-stepper ds-stepper-soft");
 const [minus,quantity,plus]=soft.props.children;assert.equal(quantity.props.children,1);assert.equal(minus.props["aria-label"],"Quitar Caldo");minus.props.onClick();assert.equal(removed,1);plus.props.onClick();assert.deepEqual(changes,[2]);
 const maximum=exports.NumberStepper({...base,value:3,appearance:"soft"});assert.equal(maximum.props.children[2].props.disabled,true);maximum.props.children[0].props.onClick();assert.deepEqual(changes,[2,2]);
 const locked=exports.NumberStepper({...base,disabled:true,appearance:"soft"});assert.equal(locked.props.children[0].props.disabled,true);assert.equal(locked.props.children[2].props.disabled,true);
});

test("cápsula suave usa tokens compartidos, 44 px y no recorta el foco",()=>{
 const css=postcss.parse(read("src/design-system/styles/form-controls.css"));
 const declarations=selector=>Object.fromEntries(css.nodes.find(node=>node.selector===selector).nodes.filter(node=>node.type==="decl").map(node=>[node.prop,node.value]));
 const soft=declarations(".ds-stepper-soft");assert.equal(soft.background,"var(--cloud-100)");assert.equal(soft.border,"0");assert.equal(soft.height,"var(--size-44)");assert.equal(soft["border-radius"],"var(--radius-pill)");assert.equal(soft.overflow,"visible");assert.equal(declarations(".ds-stepper-soft>.ds-stepper-value").border,"0");
});
