import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import test from "node:test";
import postcss from "postcss";

const root=new URL("../",import.meta.url);
const read=path=>readFileSync(new URL(path,root),"utf8");

test("Ventas muestra una unica tabla con los registros de la pagina",()=>{
  const source=read("src/modules/sales/presentation/sales-page.tsx");
  const css=postcss.parse(read("src/modules/sales/presentation/sales.css"));
  assert.match(source,/<DataTable label="Historial de ventas"/);
  assert.match(source,/items\.map\(\(sale,i\)=>/);
  assert.match(source,/key=\{sale\.id\}/);
  assert.doesNotMatch(source,/management-cards|sales-cards/);
  assert.match(source,/<Pagination page=\{page\} size=\{size\} total=\{sales\.data\?\.total\?\?0\}/);
  let usesMinWidth=false;
  css.walkRules(rule=>{
    if(rule.selector===".sales-table-wrap .sales-table"){
      usesMinWidth=rule.nodes.some(node=>node.prop==="min-width"&&node.value==="var(--size-900)");
    }
  });
  assert.ok(usesMinWidth,"La tabla conserva un ancho mínimo en móvil");
});

test("el componente compartido oculta tarjetas duplicadas y mantiene el scroll",()=>{
  const css=postcss.parse(read("src/design-system/styles/data-table.css"));
  let hidden=false,scroll=false;
  css.walkRules(rule=>{
    if(rule.selector===".management-cards"){
      hidden=rule.nodes.some(node=>node.prop==="display"&&node.value==="none");
    }
    if(rule.selector===".table-wrap"){
      scroll=rule.nodes.some(node=>node.prop==="overflow-x"&&node.value==="auto");
    }
  });
  assert.ok(hidden,"Los registros antiguos no deben duplicarse en tarjetas");
  assert.ok(scroll,"Las columnas desbordadas siguen accesibles mediante scroll");
  let frozenLegacy=false;
  css.walkAtRules("media",atRule=>{
    if(atRule.params!=="(width<=700px)")return;
    atRule.walkRules(rule=>{
      if(rule.selector===".panel .table-wrap:not(.data-table-viewport)>table :is(th,td):first-child"){
        frozenLegacy=rule.nodes.some(node=>node.prop==="position"&&node.value==="sticky");
      }
    });
  });
  assert.ok(frozenLegacy,"Las tablas anteriores mantienen la columna identificadora visible al desplazarse");
});

test("Gastos Compras y Kardex conservan sus tablas en móvil",()=>{
  const expenses=read("src/modules/expenses/presentation/expenses.css");
  const purchases=read("src/modules/supply/purchases/presentation/purchases.css");
  const inventory=read("src/modules/supply/inventory/presentation/inventory.css");
  assert.doesNotMatch(expenses,/\.expenses-table-wrap,\.expenses-category-table\{display:none\}/);
  assert.doesNotMatch(expenses,/\.expenses-cards\{display:grid\}/);
  assert.doesNotMatch(purchases,/\.purchases-table-wrap\{display:none\}/);
  assert.doesNotMatch(purchases,/\.purchases-cards\{display:grid\}/);
  assert.doesNotMatch(inventory,/\.kardex-skeleton-table\{display:none\}/);
  assert.doesNotMatch(inventory,/\.kardex-skeleton-cards\{display:grid\}/);
});

test("Mesas y Zonas comparten DataTable y sus dimensiones responsive tokenizadas",()=>{
  const src=read("src/modules/operations/tables/presentation/tables-manager.tsx");
  const css=read("src/modules/operations/styles/table-qr.css");
  assert.match(src,/<DataTable label="Mesas del local" containerClassName="hover-scroll" className="tables-management-table">/);
  assert.match(src,/<DataTable label="Zonas del local" containerClassName="hover-scroll" className="zones-management-table">/);
  assert.equal((src.match(/<DataTable /g)||[]).length,2);
  assert.equal((src.match(/<\/DataTable>/g)||[]).length,2);
  assert.doesNotMatch(src,/className="table-wrap hover-scroll"/);
  assert.match(css,/\.tables-management-table\{min-width:var\(--size-760\)\}/);
  assert.match(css,/\.zones-management-table\{min-width:var\(--size-540\)\}/);
});
