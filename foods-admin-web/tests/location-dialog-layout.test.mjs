import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
const read=file=>readFileSync(new URL(`../${file}`,import.meta.url),"utf8");
const source=read("src/modules/organizations/presentation/organization-admin.tsx");
const dialog=source.slice(source.indexOf("function LocationDialog"),source.indexOf("const blankProfile"));
const css=read("src/modules/organizations/presentation/organization-admin.css");

test("el modal del local es amplio y su carga usa el mismo tamaño",()=>{
 assert.match(dialog,/className="crud-modal size-lg location-dialog modal-panel-in"/);
 assert.match(source,/<RemoteModalSkeleton className="size-lg" label="Cargando datos del local"/);
});

test("Datos en 3×2 sin huecos: obligatorios arriba, opcionales abajo, antes de la ubicación",()=>{
 const order=["label=\"Nombre\"","label=\"Perfil fiscal\"","label=\"Zona horaria\"","label=\"Teléfono\"","label=\"Horario de atención\"","label=\"Fecha de apertura\"","title=\"Ubicación\""].map(mark=>dialog.indexOf(mark));
 order.forEach(index=>assert.ok(index>0));
 assert.deepEqual([...order].sort((a,b)=>a-b),order,"el orden del DOM coincide con el visual");
 assert.doesNotMatch(dialog,/location-field-wide/,"ningún campo ancho deja un hueco en la fila");
 assert.doesNotMatch(dialog,/aria-label="Configuración fiscal"/,"no queda una sección suelta para un solo campo");
});

test("una sola grilla de 3 columnas que pasa a 1 columna en móvil",()=>{
 assert.match(css,/\.form-grid\.location-form-grid\{grid-template-columns:repeat\(3,minmax\(0,1fr\)\)\}/);
 assert.match(css,/\.location-dialog \.geographic-fields-grid\{gap:var\(--space-4\)\}/,"los selectores de zona caen sobre las columnas de la grilla");
 assert.match(css,/@media \(width<=600px\)\{[\s\S]*?\.form-grid\.location-form-grid\{grid-template-columns:minmax\(0,1fr\)\}/);
 assert.match(css,/\.location-dialog \.tip-settings-form:not\(:has\(>:nth-child\(3\)\)\)>:first-child\{grid-column:1\/-1\}/,"propinas desactivadas no dejan hueco");
 assert.match(css,/\.location-dialog \.delivery-link-content\{grid-template-columns:repeat\(3,minmax\(0,1fr\)\);align-items:start;/,"el QR y sus acciones caen sobre las columnas de la grilla, alineados arriba");
 assert.match(css,/\.location-dialog \.delivery-link-actions\{grid-template-columns:minmax\(0,1fr\)\}/,"las tres acciones se apilan junto al QR, del ancho de su columna");
 assert.match(css,/@media \(width<=600px\)\{[\s\S]*?\.location-dialog \.delivery-link-content\{grid-template-columns:minmax\(0,1fr\)\}/,"en móvil el QR y las acciones se apilan");
});
