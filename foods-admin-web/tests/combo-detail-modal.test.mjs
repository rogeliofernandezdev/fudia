import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import postcss from "postcss";

const read=file=>readFileSync(new URL(`../src/${file}`,import.meta.url),"utf8");
const page=read("modules/menu/combos/presentation/combos-page.tsx");
const dialog=page.slice(page.indexOf("function ComboDetailDialog("),page.indexOf("function ComboWizard("));
const detailRules=[];
postcss.parse(read("modules/menu/combos/presentation/combo-wizard.css")).walkRules(rule=>{if(rule.selector.includes("combo-detail"))detailRules.push(rule)});
const declarations=rule=>Object.fromEntries(rule.nodes.filter(node=>node.type==="decl").map(node=>[node.prop,node.value]));

test("el detalle de menú usa la anatomía compartida de los modales de detalle",()=>{
 assert.match(dialog,/className="crud-modal size-md combo-detail modal-panel-in"/);
 assert.match(dialog,/<ModalHeader title=\{value\?\.name\?\?"Menú o combo"\} titleId="combo-detail-title" icon="combo" onClose=\{close\} closeLabel="Cerrar"\/>/);
 assert.doesNotMatch(dialog,/subtitle=|DETALLE DEL MENÚ|PARTES DEL MENÚ|QUÉ INCLUYE/,"sin franja ni datos redundantes bajo el título");
 assert.match(dialog,/className="modal-body combo-detail-body"/,"un solo cuerpo desplazable del design system");
 assert.match(dialog,/<dl className="modal-record-data">/,"precio y venta usan la franja de datos compartida");
 assert.match(dialog,/<ModalItemComponents label="Platos y opciones del menú"/,"las partes del menú se leen como carta compartida");
 assert.match(dialog,/<ModalError title="No pudimos cargar el menú"/);
 assert.match(dialog,/<ModalFooter status=\{<Status active=\{value\.active\}>/);
 assert.doesNotMatch(dialog,/Button[^>]*>Cerrar</,"la X es el único cierre de un detalle");
});

test("el módulo no redefine cuerpo, tamaño ni tipografía del modal",()=>{
 for(const rule of detailRules){
  const values=declarations(rule);
  for(const prop of ["font-weight","width","max-height","padding","overflow-y","background"]){
   if(prop==="width"&&rule.selector.includes("combo-detail-loading"))continue;
   if(prop==="width"&&rule.selector.includes("combo-detail-skeleton-copy"))continue;
   if(prop==="background"&&rule.selector.includes("combo-detail-loading"))continue;
   assert.equal(values[prop],undefined,`${rule.selector} no debe declarar ${prop}`);
  }
 }
 assert.ok(!detailRules.some(rule=>/combo-detail-(summary|group|availability|state)\b/.test(rule.selector)),"se retiraron las tarjetas propias");
});

test("el detalle no repite reglas de elección propias del asistente",()=>{
 assert.doesNotMatch(dialog,/Obligatorio|Opcional|El cliente elige|side:/);
});

test("el asistente de menú no añade borde ni franja bajo la cabecera",()=>{
 const wizard=page.slice(page.indexOf("function ComboWizard("));
 assert.match(wizard,/<ModalHeader title=\{editing\?"Actualiza tu menú o combo":"Arma tu menú o combo"\} titleId="combo-wizard-title" icon="chefHat" onClose=\{requestClose\}/);
 assert.doesNotMatch(wizard,/subtitle=|NUEVO MENÚ O COMBO|EDITAR MENÚ O COMBO/);
 const panels=[];
 postcss.parse(read("modules/menu/combos/presentation/combo-wizard.css")).walkRules(rule=>{if(rule.selector===".combo-wizard")panels.push(declarations(rule))});
 assert.ok(panels.length>0);
 for(const panel of panels)assert.equal(panel.border,undefined,"un borde propio deja un filo claro alrededor de la cabecera índigo");
});

test("en pantalla completa el asistente llena el alto y fija el pie abajo",()=>{
 const stages=[];
 postcss.parse(read("modules/menu/combos/presentation/combo-wizard.css")).walkRules(rule=>{if(rule.selector===".combo-wizard-stage")stages.push(declarations(rule))});
 assert.equal(stages.length,1);
 assert.equal(stages[0].flex,"1 1 auto","sin crecer, el pie queda a media pantalla a 427 × 933");
});
