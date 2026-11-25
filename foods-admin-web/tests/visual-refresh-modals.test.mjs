import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";

const read=path=>readFileSync(new URL(`../${path}`,import.meta.url),"utf8");

test("el interruptor de estado es un switch dibujado: verde encendido y gris apagado, nunca rojo",()=>{
  const source=read("src/design-system/page-header.tsx");
  assert.match(source,/role="switch" aria-checked=\{action==="deactivate"\}/);
  assert.match(source,/<span className="status-switch" aria-hidden="true"><span\/><\/span>/);
  const css=read("src/design-system/styles/table-actions.css");
  assert.match(css,/\.status-switch\{[^}]*background:var\(--ink-300\)/);
  assert.match(css,/\.status-toggle\[aria-checked="true"\] \.status-switch\{background:var\(--brand-600\)\}/);
  assert.doesNotMatch(css,/status-toggle\{[^}]*danger/,"apagado no usa rojo: desactivar no es un error");
});

test("la confirmación muestra el registro directo en la frase, sin contenedor gris",()=>{
  const css=read("src/design-system/styles/confirm-dialog.css");
  assert.match(css,/\.confirm-card\{[^}]*width:min\(var\(--size-440\),100%\)/);
  assert.match(css,/\.confirm-icon\{[^}]*width:var\(--size-48\)/);
  assert.match(css,/\.confirm-actions\{[^}]*border-top:var\(--stroke-1\) solid var\(--line\)/);
  assert.doesNotMatch(css,/\.confirm-description\{[^}]*background/);
});

test("FormSection vivid usa icono sólido, título de 16 px y línea fina",()=>{
  const source=read("src/design-system/form-section.tsx");
  assert.match(source,/appearance==="vivid"\?"is-vivid"/);
  assert.match(source,/<IconTile name=\{icon\} tone=\{tone\} appearance=\{appearance\}/);
  const css=read("src/design-system/styles/form-section.css");
  assert.match(css,/\.form-section\.is-vivid>\.form-section-title\{[^}]*border-bottom:var\(--stroke-1\) solid var\(--line\)[^}]*font-size:var\(--font-size-md\)/);
});

test("Cerrar turno: esperado en franja azul sólida y diferencia con píldora por resultado",()=>{
  const css=read("src/modules/operations/cash/presentation/cash.css");
  assert.match(css,/\.cash-close-summary \.expected\{[^}]*background:var\(--primary-600\)/);
  assert.match(css,/\.cash-close-difference\.balanced em\{background:var\(--brand-600\)\}/);
  assert.match(css,/\.cash-close-difference\.positive em\{background:var\(--primary-600\)\}/);
  assert.match(css,/\.cash-close-difference\.negative em\{background:var\(--danger-500\)\}/);
  assert.match(css,/\.cash-close-warning\{[^}]*background:var\(--warning-100\)/);
  const source=read("src/modules/operations/cash/presentation/cash-dialogs.tsx");
  assert.match(source,/appearance="vivid" title="Conteo de efectivo"/);
});

test("Pedidos: canal con icono en su color (moto para Delivery) y total tachado si el pedido se cancela",()=>{
  const css=read("src/modules/operations/styles/orders.css");
  assert.match(css,/\.order-channel-name\.tone-indigo>svg\{color:var\(--indigo-600\)\}/);
  assert.match(css,/\.order-channel-name\.tone-orange>svg\{color:var\(--warning-500\)\}/);
  assert.match(css,/\.order-channel-name\.tone-green>svg\{color:var\(--brand-600\)\}/);
  assert.match(css,/\.order-table-total\.is-cancelled\{[^}]*text-decoration:line-through/);
  const source=read("src/modules/operations/orders/presentation/orders-manager.tsx");
  assert.match(source,/className=\{`order-channel-name tone-\$\{channelTone\(service\)\}`\}/);
  assert.match(source,/o\.status==="cancelado"\?" is-cancelled":""/);
  assert.match(read("src/modules/operations/orders/presentation/order-meta.ts"),/delivery:"motorbike"/);
});

test("Tablas: más contraste e importes a la derecha con su cabecera",()=>{
  const css=read("src/design-system/styles/data-table.css");
  assert.match(css,/(^|\n)td\{[^}]*color:var\(--ink-700\)/);
  assert.match(css,/td small\{[^}]*color:var\(--ink-500\)/);
  assert.match(css,/\.data-table tbody td:is\(\.table-amount,:has\(>\.table-amount\)\)\{text-align:right\}/);
  assert.match(css,/thead th:nth-child\(5\),[\s\S]*?thead th:nth-child\(8\)\{text-align:right\}/);
  const expenses=read("src/modules/expenses/presentation/expenses-page.tsx");
  assert.match(expenses,/item\.status==="void"\?" is-void":""/);
  assert.match(expenses,/registeredSameDay\(item\.createdAt,item\.businessDate,location\?\.timezone\)\?\{timeStyle:"short"\}/);
});

test("Cerrar turno: el cierre ciego oculta los importes y el normal siempre muestra el resumen, sin repetirlo en el informe",()=>{
  const source=read("src/modules/operations/cash/presentation/cash-dialogs.tsx");
  assert.match(source,/\{blind\s*\?<div className="cash-blind-close-note">[\s\S]*?:<section className="cash-close-summary"/);
  assert.doesNotMatch(source,/!report&&<section className="cash-close-summary"/,"con informe también se ve el resumen");
  assert.match(source,/<CashReportContent report=\{report\} showTips=\{false\} showCashAmounts=\{false\}\/>/);
  const report=read("src/modules/operations/cash/presentation/cash-report-content.tsx");
  assert.match(report,/\{showCashAmounts&&<><div><dt>Egresos de efectivo<\/dt>/);
});
