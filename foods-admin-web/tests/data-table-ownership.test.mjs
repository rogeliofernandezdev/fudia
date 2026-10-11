import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import test from "node:test";
import postcss from "postcss";

const root=new URL("../",import.meta.url);
const read=path=>readFileSync(new URL(path,root),"utf8");

test("las tablas informativas tienen estilos propios y no dependen de globals",()=>{
  const source=read("src/design-system/styles/data-table.css");
  const global=read("src/styles/globals.css");
  const sheet=postcss.parse(source);
  const selectors=new Set();
  sheet.walkRules(rule=>rule.selectors.forEach(selector=>selectors.add(selector)));
  for(const selector of ["table","th","td",".table-wrap",".table-wrap table"]){
    assert.ok(selectors.has(selector),selector);
  }
  assert.doesNotMatch(global,/\.table-wrap\{|^table\{|^th\{|^td\{/m);
  assert.match(read("src/app/layout.tsx"),/design-system\/styles\/data-table\.css/);
  assert.match(source,/scrollbar-width:thin/,"el desplazamiento no debe ocultarse en móvil");
});

test("DataTable es una tabla nativa con región desplazable etiquetada y composición Tailwind",()=>{
  const table=read("src/design-system/data-table.tsx");
  assert.match(table,/role="region" aria-label=\{label\} tabIndex=\{0\}/);
  assert.match(table,/className=\{`table-wrap min-w-0 max-w-full /);
  assert.match(table,/<table \{\.\.\.tableProps\}>\{children\}<\/table>/);
  assert.match(read("src/design-system/index.ts"),/export \{DataTable\} from "\.\/data-table"/);
  const management=read("src/shared/management/management-page.tsx");
  assert.match(management,/<DataTable label=/);
  assert.doesNotMatch(management,/management-cards/);
});

test("Pagination es propietaria de los estilos de paginación informativa",()=>{
  const css=read("src/design-system/styles/pagination.css");
  const global=read("src/styles/globals.css");
  assert.match(css,/\.standard-pagination\{display:flex/);
  assert.match(css,/@media \(width<=700px\)/);
  assert.doesNotMatch(global,/\.standard-pagination\{display:flex/);
});
