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

test("obligatorios primero, opcionales en una fila y el perfil fiscal antes de la ubicación",()=>{
 const order=["label=\"Nombre\"","label=\"Perfil fiscal\"","label=\"Teléfono\"","label=\"Horario de atención\"","label=\"Fecha de apertura\"","title=\"Ubicación\"","label=\"Zona horaria\""].map(mark=>dialog.indexOf(mark));
 order.forEach(index=>assert.ok(index>0));
 assert.deepEqual([...order].sort((a,b)=>a-b),order,"el orden del DOM coincide con el visual");
 assert.match(dialog,/<FormField className="location-field-wide" label="Nombre"/);
 assert.doesNotMatch(dialog,/aria-label="Configuración fiscal"/,"no queda una sección suelta para un solo campo");
 assert.doesNotMatch(dialog,/className="span-2" label="Zona horaria"/);
});

test("una sola grilla de 3 columnas que pasa a 1 columna en móvil",()=>{
 assert.match(css,/\.form-grid\.location-form-grid\{grid-template-columns:repeat\(3,minmax\(0,1fr\)\)\}/);
 assert.match(css,/\.location-dialog \.geographic-fields-grid\{gap:var\(--space-4\)\}/,"los selectores de zona caen sobre las columnas de la grilla");
 assert.match(css,/@media \(width<=600px\)\{[\s\S]*?\.form-grid\.location-form-grid\{grid-template-columns:minmax\(0,1fr\)\}/);
});
