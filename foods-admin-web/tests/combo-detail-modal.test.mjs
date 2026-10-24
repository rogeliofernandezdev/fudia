import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import postcss from "postcss";

const read=file=>readFileSync(new URL(`../src/${file}`,import.meta.url),"utf8");
const page=read("modules/menu/combos/presentation/combos-page.tsx");
const dialog=page.slice(page.indexOf("function ComboDetailDialog("),page.indexOf("function ComboEditor("));
const detailRules=[];
postcss.parse(read("modules/menu/combos/presentation/combo-editor.css")).walkRules(rule=>{if(rule.selector.includes("combo-detail"))detailRules.push(rule)});
const declarations=rule=>Object.fromEntries(rule.nodes.filter(node=>node.type==="decl").map(node=>[node.prop,node.value]));

test("el detalle de menú usa la anatomía compartida de los modales de detalle",()=>{
 assert.match(dialog,/className="crud-modal size-md combo-detail modal-panel-in"/);
 assert.match(dialog,/<ModalHeader title=\{value\?\.name\?\?"Menú o combo"\} titleId="combo-detail-title" icon="combo" onClose=\{close\} closeLabel="Cerrar"\/>/);
 assert.doesNotMatch(dialog,/subtitle=|DETALLE DEL MENÚ|PARTES DEL MENÚ|QUÉ INCLUYE/,"sin franja ni datos redundantes bajo el título");
 assert.match(dialog,/className="modal-body combo-detail-body"/,"un solo cuerpo desplazable del design system");
 assert.match(dialog,/className="combo-detail-hero"/,"estado y venta encabezan el cuerpo, como el pedido");
 assert.match(dialog,/<p className="combo-detail-schedule"><Icon name="calendar"/);
 assert.match(dialog,/<ModalItemList className="combo-detail-parts">/,"las partes son una lista con su icono, como los productos del pedido");
 assert.match(dialog,/<IconTile name=\{style\.icon\} tone=\{PART_TILE\[style\.tone\]\} size="sm"\/>/);
 assert.match(dialog,/<ModalWorkspace summary=\{<section className="modal-summary"/,"resumen fijo al pie");
 assert.match(dialog,/<div className="modal-summary-total"><span>Precio<\/span>/,"el precio es el importe principal");
 assert.match(dialog,/<ModalError title="No pudimos cargar el menú"/);
 assert.match(dialog,/value\.active&&isOutsideDates\(value\.availableFrom,value\.availableUntil\)\?<Status tone="gray">Fuera de fecha<\/Status>:<Status active=\{value\.active\}>/,"estado real junto a la venta, igual que el listado");
 assert.doesNotMatch(dialog,/<ModalFooter/,"detalle de lectura sin pie de acciones");
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

test("la hoja de menú reutiliza el panel compartido, sin borde, franja ni pasos",()=>{
 const editor=page.slice(page.indexOf("function ComboEditor("),page.indexOf("function CompositionStep("));
 assert.match(editor,/<ModalHeader title=\{editing\?"Actualiza tu menú o combo":"Arma tu menú o combo"\} titleId="combo-editor-title" icon="chefHat" onClose=\{requestClose\}/);
 assert.doesNotMatch(editor,/subtitle=|NUEVO MENÚ O COMBO|EDITAR MENÚ O COMBO/);
 assert.match(editor,/className="crud-modal size-lg combo-editor modal-panel-in"/,"size-lg: pantalla completa con pie fijo en móvil");
 assert.match(editor,/className="modal-body combo-editor-body"/,"un solo desplazamiento del design system");
 assert.doesNotMatch(page,/combo-wizard-steps|STEP_LABELS|Paso \d|ReviewStep|action="continue"/,"sin pasos, resumen aparte ni Continuar");
 const panels=[];
 postcss.parse(read("modules/menu/combos/presentation/combo-editor.css")).walkRules(rule=>{if(rule.selector===".combo-editor")panels.push(declarations(rule))});
 for(const panel of panels)assert.equal(panel.border,undefined,"un borde propio deja un filo claro alrededor de la cabecera índigo");
});

test("el pie resume partes y precio, y el precio es el único protagonista índigo",()=>{
 const editor=page.slice(page.indexOf("function ComboEditor("),page.indexOf("function CompositionStep("));
 assert.match(editor,/className="combo-editor-summary" aria-live="polite"/);
 assert.match(editor,/\{busy\?\(editing\?"Guardando…":"Creando…"\):\(editing\?"Guardar cambios":"Crear menú"\)\}/);
 const rules={};
 postcss.parse(read("modules/menu/combos/presentation/combo-editor.css")).walkRules(rule=>{if(rule.parent.type==="root")for(const selector of rule.selectors)rules[selector]=declarations(rule)});
 assert.equal(rules[".combo-editor-summary strong"].color,"var(--primary-700)");
 assert.equal(rules[".combo-window-option.active"].background,"var(--primary-50)","selección en tinte suave, no índigo sólido");
 assert.equal(rules[".day-chip.active"].background,"var(--primary-50)");
});

test("cada parte tiene identidad propia: tono suave e icono del set, texto y controles neutros",()=>{
 const icons=read("design-system/icons.tsx");
 for(const name of ["salad","dessert","drink"])assert.ok(icons.includes(` ${name}:Icon`),`el set define ${name}`);
 assert.match(page,/\{pattern:\/entrada[^}]*tone:"green",icon:"salad"/);
 assert.match(page,/\{pattern:\/segundo[^}]*tone:"indigo",icon:"dish"/);
 assert.doesNotMatch(read("modules/menu/combos/presentation/combo-editor.css"),/tone-amber|warning-/,"las partes no usan tonos de advertencia");
 assert.match(page,/\{pattern:\/postre[^}]*tone:"violet",icon:"dessert"/);
 assert.match(page,/\{pattern:\/bebida[^}]*tone:"teal",icon:"drink"/);
 assert.match(page,/className=\{`combo-part tone-\$\{style\.tone\}`\}/);
 const rules={};
 postcss.parse(read("modules/menu/combos/presentation/combo-editor.css")).walkRules(rule=>{if(rule.parent.type==="root")for(const selector of rule.selectors)rules[selector]=declarations(rule)});
 assert.equal(rules[".combo-part.tone-green"].background,"linear-gradient(160deg, var(--brand-100), var(--surface) 70%)");
 assert.equal(rules[".combo-part.tone-green .combo-option-head"].color,"var(--brand-800)","brand-700 sobre brand-50 no llega a 4.5:1");
 assert.equal(rules[".combo-part-icon"].background,"var(--surface)");
 for(const selector of [".combo-part-remove",".combo-option-remove"]){
  assert.equal(rules[selector].color,"var(--danger-600)",`${selector}: quitar se ve como acción destructiva`);
  assert.equal(rules[selector].background,"transparent","sin baldosa roja en cada línea");
 }
});
