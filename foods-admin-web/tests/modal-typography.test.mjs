import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import postcss from "postcss";

const read=file=>readFileSync(new URL(`../src/${file}`,import.meta.url),"utf8");
const rules=file=>{const result=[];postcss.parse(read(file)).walkRules(rule=>result.push(rule));return result;};
const declarations=rule=>Object.fromEntries(rule.nodes.filter(node=>node.type==="decl").map(node=>[node.prop,node.value]));
const shared=rules("design-system/styles/modal.css");

test("un único dueño define etiquetas 600, valores 400 y secciones 700 en modales",()=>{
 const tokens=rules("styles/variables.css");
 const root=declarations(tokens.find(rule=>rule.selector===":root"));
 const panel=declarations(tokens.find(rule=>rule.selector===".crud-modal"));
 assert.equal(root["--modal-label-weight"],"var(--weight-semibold)");
 assert.equal(root["--modal-value-weight"],"var(--weight-regular)");
 assert.equal(panel["--control-value-weight"],"var(--modal-value-weight)");
 for(const [selector,weight,size] of [
  [".crud-modal :where(dt,.modal-data-label)","var(--modal-label-weight)","var(--font-size-label)"],
  [".crud-modal :where(dd,.modal-data-value)","var(--modal-value-weight)","var(--font-size-base)"],
 ]){
  const matches=shared.filter(rule=>rule.selector===selector);assert.equal(matches.length,1);
  assert.equal(declarations(matches[0])["font-weight"],weight);
  assert.equal(declarations(matches[0])["font-size"],size);
 }
 assert.equal(declarations(shared.find(rule=>rule.selector===".crud-modal h3"))["font-weight"],"var(--weight-bold)");
 assert.equal(declarations(shared.find(rule=>rule.selector===".crud-modal h3"))["font-size"],"var(--font-size-md)");
 for(const [file,selector] of [["modules/operations/orders/presentation/order-detail.css",".order-detail-section-head h3"],["modules/sales/presentation/sales.css",".sales-detail-section h3"],["modules/operations/pos/presentation/pos.css",".pos-detail-section h3"]]){
  assert.equal(declarations(rules(file).find(rule=>rule.selector===selector))["font-size"],"var(--font-size-md)",`${file}: sección por encima de los datos`);
 }
 assert.equal(declarations(rules("modules/sales/presentation/sales.css").find(rule=>rule.selector===".sales-detail-lines p,.sales-detail-payments p"))["font-size"],"var(--font-size-label)");
});

test("los módulos no reescriben los pesos de datos informativos semánticos",()=>{
 for(const [file,selectors] of [
  ["modules/operations/pos/presentation/pos.css",[".pos-detail-context dt",".pos-detail-context dd"]],
  ["modules/sales/presentation/sales.css",[".sales-detail-context dt",".sales-detail-context dd"]],
  ["modules/operations/orders/presentation/order-detail.css",[".order-attribution dt",".order-attribution dd"]],
  ["modules/operations/cash/presentation/cash-report.css",[".cash-report-identity dt",".cash-report-identity dd"]],
  ["modules/platform/presentation/platform-companies.css",[".company-detail-fields dt",".company-detail-fields dd"]],
  ["modules/expenses/presentation/expenses.css",[".expense-detail-grid dt",".expense-detail-grid dd"]],
  ["modules/menu/products/presentation/product-wizard.css",[".product-review dt",".product-review dd"]],
  ["modules/menu/combos/presentation/combo-wizard.css",[".combo-review-section dt",".combo-review-section dd"]],
 ])for(const rule of rules(file).filter(rule=>rule.selectors.some(selector=>selectors.includes(selector)))){
  assert.equal(declarations(rule)["font-weight"],undefined,`${file}: ${rule.selector}`);
 }
});

test("datos heredados sin dl consumen los mismos roles en todos los detalles",()=>{
 for(const [file,label,value] of [
  ["modules/customers/presentation/customers.css",".customer-detail-fields small",".customer-detail-fields b"],
  ["modules/supply/purchases/presentation/purchases.css",".purchase-detail-meta small",".purchase-detail-meta b"],
  ["modules/platform/presentation/platform-companies.css",".company-detail-summary small",".company-detail-summary b"],
  ["modules/menu/combos/presentation/combo-wizard.css",".combo-detail-availability small",".combo-detail-availability b"],
  ["modules/operations/orders/presentation/order-detail.css",".order-detail-context small",".order-detail-context b"],
 ]){
  const local=rules(file);
  assert.equal(declarations(local.find(rule=>rule.selectors.includes(label)))["font-weight"],"var(--modal-label-weight)",`${file}: label`);
  assert.equal(declarations(local.find(rule=>rule.selectors.includes(value)))["font-weight"],"var(--modal-value-weight)",`${file}: value`);
 }
});

test("formulario hereda valores regulares sin cambiar controles fuera del modal",()=>{
 assert.match(read("styles/globals.css"),/font-weight:var\(--control-value-weight,var\(--weight-medium\)\)/);
 assert.match(read("design-system/search-select-styles.ts"),/fontWeight:"var\(--control-value-weight,var\(--weight-semibold\)\)"/);
 assert.match(read("design-system/timezone-select.tsx"),/fontWeight:"var\(--control-value-weight,var\(--weight-semibold\)\)"/);
});

test("totales principales y estados conservan su énfasis; no hay cliente/mesa en detalles",()=>{
 assert.equal(declarations(shared.find(rule=>rule.selector===".modal-summary-total strong"))["font-weight"],"var(--weight-bold)");
 assert.match(read("modules/sales/presentation/sale-detail-dialog.tsx"),/className="modal-summary-total"/);
 assert.equal(declarations(rules("modules/operations/cash/presentation/cash-report.css").find(rule=>rule.selector===".cash-report-amounts dd"))["font-weight"],"var(--weight-bold)");
 for(const file of ["modules/sales/presentation/sale-detail-dialog.tsx","modules/operations/pos/presentation/pos-page.tsx"]){
  assert.doesNotMatch(read(file),/Cliente\s*\/\s*mesa/i);
 }
 assert.doesNotMatch(read("design-system/styles/confirm-dialog.css"),/--modal-(?:label|value)-weight/);
});
