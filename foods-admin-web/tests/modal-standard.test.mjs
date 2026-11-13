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

test("cabecera solo con título y cierre, sin icono; contexto siempre fuera",()=>{
 const html=renderToStaticMarkup(h(ModalHeader,{title:"Nueva orden",details:"Proveedor seleccionado",onClose:()=>{}}));
 const header=html.match(/<header\b[\s\S]*?<\/header>/)[0];
 assert.match(header,/<h2[^>]*>Nueva orden<\/h2>/);
 assert.match(header,/^<header class="modal-header"><div class="modal-heading">/,"el título es lo primero de la cabecera: ningún icono lo precede");
 assert.equal((header.match(/<svg\b/g)??[]).length,1,"el único icono de la cabecera es la X");
 assert.doesNotMatch(header,/<(?:small|p)\b|Orden de compra|Proveedor seleccionado/);
 assert.match(html,/<\/header><div class="modal-context">/);
 assert.match(html,/aria-label="Cerrar"/);
});

test("ningún modal pasa un icono a la cabecera",()=>{
 const files=dir=>readdirSync(resolve(root,dir),{withFileTypes:true}).flatMap(entry=>entry.isDirectory()?files(`${dir}/${entry.name}`):entry.name.endsWith(".tsx")?[`${dir}/${entry.name}`]:[]);
 const offenders=[];
 for(const file of files(".")){
  const source=read(file);
  for(const match of source.matchAll(/<(ModalHeader|Modal|RemoteModalSkeleton)\s[^>]*?\s(icon|iconSlot)=/g))offenders.push(`${file}: <${match[1]} ${match[2]}=`);
 }
 assert.deepEqual(offenders,[],"La cabecera de un modal solo lleva título y X");
 assert.doesNotMatch(read("design-system/modal.tsx"),/modal-title-icon|iconSlot/,"ModalHeader no dibuja ni acepta un icono de título");
});

test("formularios empiezan bajo la cabecera sin franja ni título repetidos",()=>{
 const html=renderToStaticMarkup(h(ModalHeader,{title:"Nueva reserva",onClose:()=>{}}));
 assert.doesNotMatch(html,/modal-context|modal-context-description/);
 const reservation=read("modules/operations/reservations/presentation/reservations-page.tsx");
 assert.match(reservation,/<ModalHeader title=\{initial.id\?"Editar reserva":"Nueva reserva"\}/);
 assert.doesNotMatch(reservation,/<FormSection|title="Datos de la reserva"/);
 const violations=[];
 function scan(dir){for(const entry of readdirSync(dir,{withFileTypes:true})){
  const file=resolve(dir,entry.name);if(entry.isDirectory()){scan(file);continue}if(!file.endsWith(".tsx"))continue;
  const source=ts.createSourceFile(file,readFileSync(file,"utf8"),ts.ScriptTarget.Latest,true,ts.ScriptKind.TSX);
  function visit(node){if((ts.isJsxOpeningElement(node)||ts.isJsxSelfClosingElement(node))&&["Modal","ModalHeader","RemoteModalSkeleton"].includes(node.tagName.getText(source))&&node.attributes.properties.some(attr=>attr.name?.getText(source)==="subtitle"))violations.push(file);ts.forEachChild(node,visit)}visit(source);
 }}scan(resolve(root,"modules"));assert.deepEqual(violations,[]);
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
