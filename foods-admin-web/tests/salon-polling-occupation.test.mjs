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
 assert.match(page,/<Status tone="green" emphasis>Libre<\/Status>/,"Libre usa el estado positivo compartido aprobado");
 assert.doesNotMatch(page,/<Status tone="red">Ocupada<\/Status>/);
 assert.doesNotMatch(page,/className="salon-table-order-status"/);
 assert.match(page,/elapsed\?\.known&&<small className=\{`salon-table-elapsed/,"sin hora de apertura no se muestra «—»");
 assert.match(page,/<small className="salon-table-caption">/,"bajo el plato: mozo, origen QR o capacidad");
 assert.match(page,/Mozo: \{o\.waiterName\?o\.waiterName\.split\(\/\\s\+\/\)\.slice\(0,2\)/,"mozo con nombre y primer apellido");
 assert.match(page,/<span className="salon-table-who">\{o\?\(o\.customerName\|\|t\.zone\|\|seats\):t\.zone\}<\/span>/,"la cabecera dice quién o dónde; el plato ya identifica la mesa");
 assert.match(page,/className=\{`salon-table-plate \$\{plate\.length<=3\?"is-short"/,"el tamaño del nombre en el plato sigue a su largo");
 assert.match(page,/viaQR&&<span className="salon-table-qr"><Icon name="whatsapp"/,"el pedido QR por WhatsApp lleva su distintivo sobre el plato");
 assert.match(page,/new Intl\.Collator\("es",\{numeric:true,sensitivity:"base"\}\)/,"orden natural por número de mesa");
 assert.match(page,/\[\.\.\.floor\.data\?\.items\?\?\[\]\]\.sort\(\(a,b\)=>tableOrder\.compare\(a\.name,b\.name\)\)/);
 const order=new Intl.Collator("es",{numeric:true,sensitivity:"base"});
 assert.deepEqual(["08","09","01","10","2","Mesa 3"].sort(order.compare),["01","2","08","09","10","Mesa 3"]);
 assert.doesNotMatch(page,/collectedByNames/,"«Cobrado por» queda en el detalle de la mesa");
});

test("paleta de salón: contenedores originales y color semántico solo en estados",()=>{
 const css=readFileSync(new URL("../src/modules/operations/styles/salon.css",import.meta.url),"utf8");
 assert.match(css,/\.salon-table-stage\{[^}]*background:var\(--primary-100\)\}/,"escenario índigo tenue en ocupadas");
 assert.match(css,/\.salon-table-plate\{[^}]*background:var\(--primary-600\);color:var\(--surface\)/,"plato lleno en índigo cuando está ocupada");
 assert.match(css,/\.salon-table\.free \.salon-table-plate\{background:transparent;border:var\(--stroke-2\) dashed var\(--line-strong\)/,"plato vacío punteado cuando está libre");
 assert.match(css,/\.salon-table-foot\{margin-top:auto;/,"pie sin banda de color");
 assert.match(css,/\.salon-table\.occupied \.salon-table-go\{background:var\(--primary-100\);color:var\(--primary-700\)\}/,"el plato es el único círculo sólido");
 assert.match(css,/\.salon-grid\{display:grid;grid-template-columns:repeat\(auto-fill,minmax\(max\(var\(--size-212\),calc\(\(100% - var\(--space-4\) \* 3\) \/ 4\)\),1fr\)\)/,"como máximo cuatro mesas por fila");
 assert.match(css,/\.salon-table-go\{[^}]*background:var\(--navigation-900\);color:var\(--surface\)/,"«Abrir mesa» conserva azul marino");
 assert.match(css,/\.salon-table-open\{color:var\(--navigation-900\)/);
 assert.doesNotMatch(css,/danger|brand-/,"los contenedores no se tiñen de verde o rojo");
 assert.doesNotMatch(css,/\.salon-table[^{]*::before/,"sin bandas superiores");
 const page=readFileSync(new URL("../src/modules/operations/salon/presentation/salon-manager.tsx",import.meta.url),"utf8");
 assert.match(page,/<dl className="salon-overview-stats" aria-label="Resumen del salón">/);
 assert.match(page,/<dt>Consumo abierto<\/dt>/);
 assert.match(css,/\.salon-overview-stats \.occupied dt\{color:var\(--primary-700\)\}/);
 assert.match(css,/\.salon-table-elapsed\.is-stale\{color:var\(--warning-800\)/);
});
