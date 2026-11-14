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
 assert.match(page,/<Status tone=\{meta\.tone\} emphasis solid>\{meta\.label\}<\/Status>/,"estado operativo destacado y sólido con la variante compartida");
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

test("paleta de salón: acento Cobalto aprobado y color semántico solo en estados",()=>{
 const css=readFileSync(new URL("../src/modules/operations/styles/salon.css",import.meta.url),"utf8");
 assert.match(css,/\.salon-table-stage\{[^}]*background:color-mix\(in srgb, var\(--primary-100\) 50%, var\(--surface\)\)/,"casetón apenas teñido en ocupadas: su único color pleno es el plato");
 assert.match(css,/\.salon-table\.free \.salon-table-stage\{background:color-mix\(in srgb, var\(--primary-100\) 50%, var\(--cloud-50\)\)/,"casetón hundido en nube, teñido igual, en libres");
 assert.match(css,/\.salon-table-plate\{[^}]*background:var\(--primary-600\);color:var\(--surface\)/,"plato lleno en azul operativo cuando está ocupada");
 assert.match(css,/\.salon-table-plate\{[^}]*box-shadow:var\(--elevation-primary-soft\)/,"el plato proyecta su sombra azul suave desde el token de elevación tintada");
 assert.match(css,/\.salon-table\.free \.salon-table-plate\{background:transparent;box-shadow:none;border:var\(--stroke-2\) dashed color-mix\(in srgb, var\(--primary-600\) 50%, var\(--line-strong\)\)/,"plato vacío punteado en azul cuando está libre");
 assert.match(css,/\.salon-table-foot\{margin-top:auto;/,"pie sin banda de color");
 assert.match(css,/\.salon-table-total\{color:var\(--ink-950\)/,"el importe es un dato, no un acento");
 assert.match(css,/\.salon-table\{[^}]*border-radius:var\(--radius-card\)/,"las tarjetas usan el radio de tarjeta, no el de hoja");
 assert.match(css,/\.salon-table-plate\{[^}]*width:var\(--size-88\)/,"el plato llena el escenario (88 px)");
 assert.match(css,/\.salon-table:not\(:disabled\):hover\{border-color:color-mix\(in srgb, var\(--primary-600\) 27\.5%, var\(--line\)\)\}/,"el borde se aviva al pasar el cursor");
 assert.doesNotMatch(css,/\.salon-table[^{]*::before/,"sin bandas ni filetes superiores en las tarjetas");
 assert.doesNotMatch(css,/\.salon-overview-stats[^{]*::before/,"la tira de resumen tampoco lleva filete superior");
 assert.match(css,/\.salon-table-go\{[^}]*background:var\(--cloud-100\);color:var\(--navigation-900\)/,"el disco del pie es neutro en la ocupada: su único círculo vivo es el plato");
 assert.match(css,/\.salon-table\.free \.salon-table-go\{background:color-mix\(in srgb, var\(--primary-700\) 50%, var\(--navigation-900\)\);color:var\(--surface\)\}/,"«Abrir mesa» lleva disco macizo azul profundo desde el reposo");
 assert.match(css,/\.salon-table\.free:hover \.salon-table-go\{background:var\(--primary-700\)\}/,"el disco libre se aviva en primary-700 al hover");
 assert.match(css,/\.salon-table-open\{color:color-mix\(in srgb, var\(--primary-700\) 50%, var\(--navigation-900\)\)/,"«Abrir mesa» en azul profundo");
 assert.match(css,/\.salon-grid\{display:grid;grid-template-columns:repeat\(auto-fill,minmax\(max\(var\(--size-212\),calc\(\(100% - var\(--space-4\) \* 3\) \/ 4\)\),1fr\)\)/,"como máximo cuatro mesas por fila");
 assert.match(css,/\.salon-overview-stats\{[^}]*grid-template-columns:repeat\(4,minmax\(0,1fr\)\)[^}]*gap:var\(--space-3\)/,"el resumen es un tablero de cuatro tarjetas iguales");
 assert.match(css,/\.salon-overview-stats>div\{[^}]*border:var\(--stroke-1\) solid var\(--line\);[^}]*border-radius:var\(--radius-card\);[^}]*box-shadow:var\(--elevation-1\)/,"cada indicador es una tarjeta con el mismo borde, radio y elevación");
 assert.match(css,/\.salon-overview-stats>div\{[^}]*align-items:flex-start/,"icono, cifra y rótulo comparten el eje izquierdo de la tarjeta");
 assert.match(css,/\.salon-overview-stats dd\{[^}]*display:flex;[^}]*align-items:center;[^}]*gap:var\(--space-2-5\)/,"icono y cifra comparten la misma fila, centrados");
 assert.match(css,/\.salon-overview-stats dd\{[^}]*order:1;[^}]*white-space:nowrap/,"la cifra es la protagonista y nunca se parte");
 assert.match(css,/\.salon-overview-stats dt\{[^}]*order:2;/,"el rótulo cierra la tarjeta, debajo de la cifra");
 assert.match(css,/\.salon-overview-stats dd>b\{font-weight:inherit;animation:salon-stat-pop var\(--duration-slow\) var\(--easing-out\)\}/,"la cifra remonta suave al refrescarse");
 assert.match(css,/\.salon-overview-stats dd>b\{animation:none\}/,"el remonte se apaga con prefers-reduced-motion");
 assert.doesNotMatch(css,/danger|brand-/,"el CSS local no tiñe contenedores de rojo o verde: los IconTile viven en el design system");
 const page=readFileSync(new URL("../src/modules/operations/salon/presentation/salon-manager.tsx",import.meta.url),"utf8");
 assert.match(page,/<dl className="salon-overview-stats" aria-label="Resumen del salón">/);
 assert.match(page,/<dd><IconTile name="grid" tone="navy" appearance="vivid"\/>/,"Mesas: navy sobre la cifra");
 assert.match(page,/<dd><IconTile name="check" tone="brand" appearance="vivid"\/>/,"Libres: verde de Libre");
 assert.match(page,/<dd><IconTile name="utensils" tone="primary" appearance="vivid"\/>/,"Ocupadas: azul del plato");
 assert.match(page,/<dd><IconTile name="wallet" tone="sky" appearance="vivid"\/>/,"Consumo abierto: celeste de cuenta");
 assert.match(page,/<b key=\{freeCount\}>\{freeCount\}<\/b>/,"la cifra lleva key para remontar al cambiar");
 assert.match(css,/\.salon-overview-stats \.occupied dt\{color:var\(--primary-700\)\}/);
 assert.match(css,/\.salon-table-elapsed\.is-stale\{color:var\(--warning-800\)/);
});
