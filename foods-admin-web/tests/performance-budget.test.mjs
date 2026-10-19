import assert from "node:assert/strict";
import {existsSync,readFileSync,readdirSync,statSync} from "node:fs";
import test from "node:test";

const root=new URL("../",import.meta.url);
const read=path=>readFileSync(new URL(path,root),"utf8");

test("rendimiento: los barrels públicos son re-exportaciones puras y se podan por ruta",()=>{
  const pkg=JSON.parse(read("package.json"));
  assert.deepEqual(pkg.sideEffects,["*.css"],"Solo el CSS tiene efectos secundarios; el resto de módulos se puede podar");
  const config=read("next.config.ts");
  assert.match(config,/optimizePackageImports:\["@\/design-system","@\/providers","@\/shared\/management",\.\.\.moduleBarrels\]/);
  assert.match(config,/`@\/modules\/\$\{entry\.name\}`/,"Cada módulo con API pública se declara automáticamente");
  const barrels=["src/design-system/index.ts","src/providers/index.ts","src/shared/management/index.ts",...readdirSync(new URL("src/modules/",root)).map(name=>`src/modules/${name}/index.ts`).filter(path=>existsSync(new URL(path,root)))];
  for(const barrel of barrels){
    const statements=read(barrel).split(/\r?\n/).map(line=>line.trim()).filter(line=>line&&!line.startsWith("//")&&!line.startsWith("/*")&&!line.startsWith("*"));
    for(const line of statements)assert.match(line,/^export (type )?(\{[^}]*\}|\*) from "[^"]+";?$/,`${barrel} solo re-exporta: ${line}`);
  }
});

test("rendimiento: la navegación del shell responde al instante y reutiliza pantallas visitadas",()=>{
  const loading=read("src/app/(admin)/loading.tsx");
  assert.match(loading,/<RouteSkeleton\/>/,"El grupo admin tiene límite de carga propio dentro del shell");
  const skeleton=read("src/shell/route-skeleton.tsx");
  assert.match(skeleton,/role="status" aria-busy="true" aria-label="Cargando sección"/);
  const css=read("src/shell/styles/route-skeleton.css");
  assert.match(css,/@keyframes route-skeleton-in\{from\{opacity:0\}to\{opacity:1\}\}/,"Solo anima opacidad");
  assert.match(css,/animation:route-skeleton-in var\(--duration-fast\) var\(--easing-out\) both/,"Entra de inmediato con un fundido corto, sin dejar el contenido en blanco");
  assert.match(css,/@media\(prefers-reduced-motion:reduce\)\{\.route-skeleton\{animation:none\}\}/);
  assert.match(read("next.config.ts"),/staleTimes:\{dynamic:30\}/);
});

test("rendimiento: dependencias pesadas de interacción se cargan bajo demanda",()=>{
  const tables=read("src/modules/operations/tables/presentation/tables-manager.tsx");
  assert.equal(/^import QRCode from "qrcode"/m.test(tables),false,"qrcode no viaja con la ruta de Mesas");
  assert.match(tables,/import\("qrcode"\)/);
  const picker=read("src/modules/menu/combos/presentation/combo-option-picker.tsx");
  assert.match(picker,/dynamic\(\(\)=>import\("\.\/combo-option-search"\)/,"react-select se descarga al abrir la composición del combo");
  assert.match(picker,/<Skeleton height="var\(--control-height\)"\/>/,"El reemplazo conserva la altura del control");
  assert.match(read("src/modules/menu/combos/presentation/combo-option-search.tsx"),/export function ComboOptionSearch\(/);
  assert.match(read("src/modules/menu/products/presentation/catalog-manager.tsx"),/ProductDialog=dynamic\(/,"El editor de producto (Tiptap) sigue diferido");
});

test("rendimiento: las imágenes de interfaz usan versiones livianas",()=>{
  for(const [asset,limit] of [["public/assets/images/logo-mark.png",40_000],["public/assets/images/login-brand.png",120_000]])assert.ok(statSync(new URL(asset,root)).size<limit,`${asset} supera ${limit} bytes`);
  assert.match(read("src/design-system/logo.tsx"),/src="\/assets\/images\/logo-mark\.png"/);
  assert.match(read("src/modules/auth/presentation/login-form.tsx"),/src="\/assets\/images\/login-brand\.png" alt="fudIA" width=\{360\} height=\{240\}/);
});
