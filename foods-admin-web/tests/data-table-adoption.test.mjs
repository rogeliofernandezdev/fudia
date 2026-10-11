import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import test from "node:test";

const root=new URL("../",import.meta.url);
const read=path=>readFileSync(new URL(path,root),"utf8");

const listings={
  "src/modules/operations/tables/presentation/tables-manager.tsx":2,
  "src/modules/operations/orders/presentation/orders-manager.tsx":1,
  "src/modules/sales/presentation/sales-page.tsx":1,
  "src/modules/customers/presentation/customers-manager.tsx":1,
  "src/modules/menu/combos/presentation/combos-page.tsx":1,
  "src/modules/menu/products/presentation/catalog-manager.tsx":2,
  "src/modules/operations/reservations/presentation/reservations-page.tsx":1,
  "src/modules/supply/inventory/presentation/inventory-page.tsx":1,
  "src/modules/expenses/presentation/expenses-page.tsx":2,
  "src/modules/identity/presentation/users-roles-manager.tsx":2,
  "src/modules/operations/cash/presentation/cash-page.tsx":1,
  "src/modules/organizations/presentation/organization-admin.tsx":3,
  "src/modules/platform/presentation/platform-companies-page.tsx":1,
  "src/modules/menu/recipes/presentation/recipes-page.tsx":1,
  "src/modules/supply/inventory/presentation/kardex-page.tsx":1,
  "src/modules/operations/pos/presentation/pos-page.tsx":1,
  "src/modules/supply/purchases/presentation/purchases-page.tsx":6,
  "src/modules/operations/cash/presentation/cash-report-content.tsx":1,
  "src/modules/platform/presentation/platform-global-settings-page.tsx":1,
  "src/modules/platform/presentation/platform-subscription-page.tsx":1,
  "src/modules/configuration/presentation/payment-methods-page.tsx":1,
  "src/shared/management/management-page.tsx":1
};

test("todas las tablas informativas principales usan un solo DataTable reutilizable",()=>{
  for(const [file,expected] of Object.entries(listings)){
    const source=read(file);
    const openings=(source.match(/<DataTable\s/g)||[]).length;
    const closings=(source.match(/<\/DataTable>/g)||[]).length;
    assert.equal(openings,expected,`${file}: deben existir ${expected} tablas homologadas`);
    assert.equal(closings,expected,`${file}: cierres DataTable`);
    assert.doesNotMatch(source,/<div className="table-wrap hover-scroll[^"]*">\s*<table/,`${file}: tabla suelta sin DataTable`);
  }
});

test("Mesas y Zonas mantienen nombres accesibles y css por componente",()=>{
  const src=read("src/modules/operations/tables/presentation/tables-manager.tsx");
  const css=read("src/modules/operations/styles/table-qr.css");
  assert.match(src,/label="Mesas del local"/);
  assert.match(src,/label="Zonas del local"/);
  assert.match(css,/\.tables-management-table\{min-width:var\(--size-760\)\}/);
  assert.match(css,/\.zones-management-table\{min-width:var\(--size-540\)\}/);
});

test("el scroll móvil es común y usa tokens sin cambiar los CSS de cada módulo",()=>{
  const css=read("src/design-system/styles/data-table.css");
  assert.match(css,/\.data-table-viewport\{[^}]*overflow-x:auto/);
  assert.match(css,/\.panel \.table-wrap:not\(\.data-table-viewport\)>table\{width:max-content;min-width:100%\}/);
  assert.match(css,/\.data-table :is\(th,td\):first-child\{position:sticky/);
  assert.match(css,/\.data-table-hint\{display:flex/);
});
