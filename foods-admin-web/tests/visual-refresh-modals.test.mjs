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

test("Pedidos: canal en su color de dominio y total tachado si el pedido se cancela",()=>{
  const css=read("src/modules/operations/styles/orders.css");
  assert.match(css,/\.order-row-icon\.oc-salon\{\s*color:var\(--indigo-700\)/);
  assert.match(css,/\.order-row-icon\.oc-recojo\{\s*color:var\(--warning-600\)/);
  assert.match(css,/\.order-row-icon\.oc-delivery\{\s*color:var\(--brand-700\)/);
  assert.match(css,/\.order-table-total\.is-cancelled\{[^}]*text-decoration:line-through/);
  assert.match(read("src/modules/operations/orders/presentation/orders-manager.tsx"),/o\.status==="cancelado"\?" is-cancelled":""/);
});
