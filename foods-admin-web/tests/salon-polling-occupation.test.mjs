import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import {runInNewContext} from "node:vm";
const require=createRequire(import.meta.url);
const ts=require("typescript");
const source=readFileSync(new URL("../src/modules/operations/salon/domain/occupation-time.ts",import.meta.url),"utf8");
const compiled=ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2022}}).outputText;
const sandboxModule={exports:{}};
runInNewContext(compiled,{module:sandboxModule,exports:sandboxModule.exports,Date,Number,Math});
const {formatTableOccupation}=sandboxModule.exports;

test("el tiempo parte de la apertura activa, no de la última edición del pedido",()=>{
 const opened="2026-10-11T10:00:00Z";
 assert.equal(formatTableOccupation(opened,Date.parse("2026-10-11T11:25:00Z")).label,"1 h 25 min");
 assert.equal(formatTableOccupation(opened,Date.parse("2026-10-11T10:00:00Z")).label,"0 min");
});
test("las mesas abiertas durante días se identifican para revisión sin liberarlas",()=>{
 const result=formatTableOccupation("2026-10-05T12:00:00Z",Date.parse("2026-10-11T10:00:00Z"));
 assert.equal(result.label,"5 d 22 h");
 assert.equal(result.needsReview,true);
 assert.match(result.description,/Revisar/);
});
test("no se inventa una antigüedad sin una hora válida",()=>{
 for(const opened of [undefined,"","invalida","2026-10-12T12:00:00Z"]){
  assert.equal(formatTableOccupation(opened,Date.parse("2026-10-11T10:00:00Z")).label,"—");
  assert.equal(formatTableOccupation(opened,Date.parse("2026-10-11T10:00:00Z")).known,false);
 }
});
test("la pantalla conserva el centrado, el polling parcial y estados semánticos",()=>{
 const page=readFileSync(new URL("../src/modules/operations/salon/presentation/salon-manager.tsx",import.meta.url),"utf8");
 assert.match(page,/refetchInterval:10000/);
 assert.match(page,/refetchIntervalInBackground:false/);
 assert.match(page,/formatTableOccupation\(t\.occupiedSince,clock\)/);
 assert.match(page,/setInterval\(\(\)=>setClock\(Date\.now\(\)\),30000\)/);
 assert.match(page,/floor\.isError&&!floor\.data/);
 assert.match(page,/<Status tone=\{meta\.tone\} emphasis>\{meta\.label\}<\/Status>/,"estado destacado con la variante compartida");
 assert.match(page,/<Status tone="navy" emphasis>Libre<\/Status>/,"Libre no comparte tono con Entregado (teal) ni Listo (green)");
 assert.doesNotMatch(page,/<Status tone="red">Ocupada<\/Status>/);
 assert.doesNotMatch(page,/className="salon-table-order-status"/);
 assert.match(page,/elapsed\?\.known&&<small className=\{`salon-table-elapsed/,"sin hora de apertura no se muestra «—»");
 assert.match(page,/<p className="salon-table-meta">/,"una línea por dato, sin iconos ni etiquetas repetidas");
 assert.match(page,/new Intl\.Collator\("es",\{numeric:true,sensitivity:"base"\}\)/,"orden natural por número de mesa");
 assert.match(page,/\[\.\.\.floor\.data\?\.items\?\?\[\]\]\.sort\(\(a,b\)=>tableOrder\.compare\(a\.name,b\.name\)\)/);
 const order=new Intl.Collator("es",{numeric:true,sensitivity:"base"});
 assert.deepEqual(["08","09","01","10","2","Mesa 3"].sort(order.compare),["01","2","08","09","10","Mesa 3"]);
 assert.doesNotMatch(page,/collectedByNames|customerName\?`\$\{o\.customerName\}/,"cobro y familia quedan en el detalle de la mesa");
});

test("paleta de salon: pie de color índigo/azul marino, sin verde, rojo ni bandas; KPI como la referencia",()=>{
 const css=readFileSync(new URL("../src/modules/operations/styles/salon.css",import.meta.url),"utf8");
 assert.match(css,/\.salon-table-foot\{margin:auto calc\(var\(--space-4\) \* -1\) calc\(var\(--space-4\) \* -1\);[^}]*background:var\(--cloud-100\)\}/,"pie a todo el ancho; libre en gris nube");
 assert.match(css,/\.salon-table\.occupied \.salon-table-foot\{background:var\(--primary-100\)\}/,"ocupada en índigo");
 assert.match(css,/\.salon-table\.occupied \.salon-table-go\{background:var\(--primary-600\)\}/);
 assert.match(css,/\.salon-table-go\{[^}]*background:var\(--navigation-900\);color:var\(--surface\)/,"«Abrir mesa» en azul marino");
 assert.match(css,/\.salon-table-open\{color:var\(--navigation-900\)/);
 assert.match(css,/\.salon-table\{[\s\S]*?min-height:var\(--size-240\)/,"tarjetas altas");
 assert.doesNotMatch(css,/danger|brand-/,"sin rojo ni verde en las tarjetas: el verde queda solo en el Status «Libre»");
 assert.doesNotMatch(css,/\.salon-table[^{]*::before/,"sin bandas superiores");
 const page=readFileSync(new URL("../src/modules/operations/salon/presentation/salon-manager.tsx",import.meta.url),"utf8");
 assert.match(page,/<dl className="salon-overview-stats" aria-label="Resumen del salón">/);
 assert.match(page,/<dt>Consumo abierto<\/dt>/);
 assert.match(css,/\.salon-overview-stats \.occupied dt\{color:var\(--primary-700\)\}/);
 assert.match(css,/\.salon-table-elapsed\.is-stale\{color:var\(--warning-800\)/);
});
