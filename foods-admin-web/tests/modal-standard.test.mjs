import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync,readdirSync} from "node:fs";
import {resolve} from "node:path";
import React from "react";
import {renderToStaticMarkup} from "react-dom/server";
import ts from "typescript";
import postcss from "postcss";
import {detailComponents} from "./helpers/order-detail-fixture.mjs";

const root=resolve("src");
const read=file=>readFileSync(resolve(root,file),"utf8");
const {ModalHeader,ModalFooter}=detailComponents();
const h=React.createElement;

test("cabecera solo con icono, título y cierre; contexto siempre fuera",()=>{
 const html=renderToStaticMarkup(h(ModalHeader,{title:"Nueva orden",icon:"receipt",subtitle:"Orden de compra",details:"Proveedor seleccionado",onClose:()=>{}}));
 const header=html.match(/<header\b[\s\S]*?<\/header>/)[0];
 assert.match(header,/<h2[^>]*>Nueva orden<\/h2>/);
 assert.doesNotMatch(header,/<(?:small|p)\b|Orden de compra|Proveedor seleccionado/);
 assert.match(html,/<\/header><div class="modal-context">/);
 assert.match(html,/aria-label="Cerrar"/);
});

test("pie admite el estado real y no lo inventa para formularios",()=>{
 const simple=renderToStaticMarkup(h(ModalFooter,null,h("button",null,"Cancelar"),h("button",null,"Guardar")));
 assert.doesNotMatch(simple,/modal-footer-status/);
 assert.ok(simple.indexOf("Cancelar")<simple.indexOf("Guardar"));
 const detail=renderToStaticMarkup(h(ModalFooter,{status:h("span",null,"Cuenta abierta"),statusLabel:"Cuenta"}));
 assert.match(detail,/modal-footer-status/);assert.match(detail,/Cuenta abierta/);
});

test("acciones contiguas y pie neutro usan la única regla compartida",()=>{
 const rules=[];postcss.parse(read("design-system/styles/modal.css")).walkRules(rule=>rules.push(rule));
 const footer=rules.find(rule=>rule.parent.type==="root"&&rule.selectors.includes(".modal-footer"));
 const value=prop=>footer.nodes.find(node=>node.prop===prop)?.value;
 assert.equal(value("justify-content"),"flex-end");
 assert.equal(value("gap"),"var(--space-2-5)");
 assert.equal(value("background"),"var(--cloud-50)");
 for(const rule of rules.filter(rule=>rule.selector.includes("footer")&&rule.selector.includes(".button"))){
  assert.ok(!rule.nodes.some(node=>node.prop==="margin-right"&&node.value==="auto"),rule.selector);
 }
});

test("diálogos de contenido no duplican header ni footer de formulario",()=>{
 const violations=[];
 function scan(dir){for(const entry of readdirSync(dir,{withFileTypes:true})){
  const file=resolve(dir,entry.name);if(entry.isDirectory()){scan(file);continue}if(!file.endsWith(".tsx"))continue;
  const source=ts.createSourceFile(file,readFileSync(file,"utf8"),ts.ScriptTarget.Latest,true,ts.ScriptKind.TSX);
  function visit(node,dialog=false,section=false){
   const isJsx=ts.isJsxElement(node),tag=isJsx?node.openingElement.tagName.getText(source):"";
   if(tag==="Dialog"){dialog=true;section=false;}
   if(dialog&&["section","article","aside"].includes(tag))section=true;
   if(dialog&&!section&&["header","footer"].includes(tag))violations.push(`${file}:${source.getLineAndCharacterOfPosition(node.pos).line+1} ${tag}`);
   ts.forEachChild(node,child=>visit(child,dialog,section));
  }visit(source);
 }}scan(resolve(root,"modules"));
 assert.deepEqual(violations,[]);
});
