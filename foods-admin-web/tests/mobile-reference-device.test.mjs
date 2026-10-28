import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import postcss from "postcss";

// Viewport móvil de referencia: ≈ 427 × 933 (mínimo 390 px).
// Cada caso corresponde a un defecto observado en ese viewport.
const read=file=>readFileSync(new URL(`../src/${file}`,import.meta.url),"utf8");
const declarations=rule=>Object.fromEntries(rule.nodes.filter(node=>node.type==="decl").map(node=>[node.prop,node.value]));
function rulesFor(file,selector){
 const found=[];
 postcss.parse(read(file)).walkRules(rule=>{
  if(rule.selector===selector||rule.selector.split(",").map(part=>part.trim()).includes(selector)){
   const media=rule.parent?.type==="atrule"?rule.parent.params.replace(/\s+/g,""):"";
   found.push({media,values:declarations(rule)});
  }
 });
 return found;
}
const mobile=media=>/width<=600px|max-width:600px/.test(media);
const touch=media=>mobile(media)&&/any-pointer:coarse/.test(media);

test("la receta en pantalla completa fija su pie abajo y respeta 16 px",()=>{
 const page=read("modules/menu/recipes/presentation/recipes-page.tsx");
 assert.match(page,/<ModalForm className="recipe-form" onSubmit=\{submit\} noValidate>/,"un form heredado desplazaba el pie 16 px fuera del modal");
 assert.doesNotMatch(page,/<form className="recipe-form"/);
 const body=rulesFor("modules/menu/recipes/presentation/recipes.css",".recipe-form-body");
 assert.equal(body.find(rule=>!rule.media)?.values.flex,"1 1 auto","sin crecer, el pie queda a media pantalla a 427 × 933");
 assert.equal(body.find(rule=>mobile(rule.media))?.values.padding,"var(--space-3) var(--space-4)");
});

test("los controles compactos conservan 44 px en táctil y móvil",()=>{
 const qr=rulesFor("modules/operations/styles/table-qr.css",".qr-cell-btn");
 assert.equal(qr.find(rule=>!rule.media)?.values.height,"var(--size-32)","escritorio conserva su geometría");
 assert.equal(qr.find(rule=>touch(rule.media))?.values.height,"var(--control-height)");
 const editor=rulesFor("design-system/styles/rich-text-editor.css",".rte-toolbar button");
 assert.deepEqual([editor.find(rule=>touch(rule.media))?.values.width,editor.find(rule=>touch(rule.media))?.values.height],["var(--size-44)","var(--size-44)"]);
 const drawerClose=rulesFor("shell/styles/shell.css",".sidebar-close").find(rule=>rule.media.includes("width<=820px"));
 assert.deepEqual([drawerClose?.values.width,drawerClose?.values.height],["var(--size-44)","var(--size-44)"]);
 const back=rulesFor("styles/globals.css",".page-header-back").find(rule=>mobile(rule.media));
 assert.equal(back?.values["min-height"],"var(--size-44)");
});

test("Próximamente usa el retorno compartido mediante pageRoutes",()=>{
 const page=read("shared/management/coming-soon-page.tsx");
 assert.match(page,/back=\{\{href:pageRoutes\.home,label:"Volver al inicio"\}\}/);
 assert.match(page,/<EmptyState icon="settings" title="Próximamente"/);
 assert.doesNotMatch(page,/href="\/dashboard"|settings-back/,"el enlace suelto no tenía estilo: icono y texto en dos filas de 36 px");
});

test("Plataforma usa márgenes de 16 px, áreas de 44 px y Continuar compartido",()=>{
 const file="modules/platform/presentation/platform-shell.css";
 const narrow=selector=>rulesFor(file,selector).find(rule=>rule.media.includes("max-width:760px"))?.values;
 assert.equal(narrow(".platform-content").padding,"var(--space-6) var(--space-4) var(--space-11)");
 assert.equal(narrow(".platform-topbar").padding,"var(--space-2-5) var(--space-4)");
 assert.equal(narrow(".platform-nav a")["min-width"],"var(--control-height)");
 assert.equal(narrow(".platform-topbar .platform-exit")["min-width"],"var(--control-height)");
 assert.equal(narrow(".platform-brand>a")["min-height"],"var(--size-44)");
 const onboarding=read("modules/platform/presentation/platform-onboarding-page.tsx");
 assert.match(onboarding,/<Button type="submit" action="continue">Siguiente<\/Button>/,"la flecha dentro del texto bajaba a otra fila");
 assert.match(onboarding,/<Button action="back" onClick=\{\(\)=>setStep\(s=>s-1\)\}/);
});

test("Mi perfil no recorta rol, empresa ni local en móvil",()=>{
 const copy=rulesFor("modules/identity/presentation/profile-page.css",".profile-account-copy :is(h2,p)").find(rule=>mobile(rule.media));
 assert.equal(copy?.values["white-space"],"normal");
 assert.equal(copy?.values["overflow-wrap"],"anywhere");
});

test("un form heredado de pantalla completa deja el pie abajo sin hueco",()=>{
 const grow=rulesFor("design-system/styles/modal.css",".crud-modal>form:not(.modal-form)>:has(+footer)");
 assert.equal(grow.length,1);
 assert.equal(grow[0].media,"","aplica a todos los tamaños: sin alto libre el margen automático vale 0");
 assert.equal(grow[0].values["margin-bottom"],"auto","Registrar caja e Iniciar turno dejaban 434–476 px vacíos bajo el pie");
});

test("las hojas móviles ocupan el ancho completo, sin tarjetas flotantes",()=>{
 const sheets=[
  ["modules/operations/orders/presentation/manual-order.css",".manual-order-modal"],
  ["modules/customers/presentation/customers.css",".customer-modal"],
  ["modules/identity/presentation/users-roles.css",".access-modal"],
  ["modules/identity/presentation/role-editor.css",".role-editor-modal"],
  ["modules/identity/presentation/roles-table.css",".role-detail-modal"],
  ["modules/supply/purchases/presentation/purchases.css",".supplier-modal"],
  ["modules/platform/presentation/platform-global-settings.css",".platform-channel-modal"],
 ];
 for(const [file,selector] of sheets){
  const narrow=rulesFor(file,selector).filter(rule=>rule.media&&/max-width:(600|620|760)px|width<=600px/.test(rule.media));
  assert.ok(narrow.length>0,`${selector} conserva su variante móvil`);
  for(const rule of narrow)assert.equal(rule.values.width,"100%",`${selector} no deja márgenes propios en móvil`);
 }
 for(const selector of [".expense-editor",".expense-category-editor"]){
  const base=rulesFor("modules/expenses/presentation/expenses.css",selector).find(rule=>!rule.media);
  assert.match(base.values.width,/^min\(var\(--size-\d+\),100%\)$/);
 }
 for(const file of ["modules/operations/orders/presentation/manual-order.css","modules/customers/presentation/customers.css","modules/identity/presentation/users-roles.css","modules/identity/presentation/role-editor.css","modules/identity/presentation/roles-table.css","modules/supply/purchases/presentation/purchases.css","modules/platform/presentation/platform-global-settings.css","modules/expenses/presentation/expenses.css"]){
  assert.doesNotMatch(read(file),/width:calc\(100vw - var\(--size-(8|16|20|24|28)\)\)/,`${file} usa el ancho de la hoja compartida`);
 }
 assert.doesNotMatch(read("modules/operations/orders/presentation/manual-order.css"),/manual-order-backdrop/);
});

test("acordeones, mapa y sugerencias conservan áreas táctiles",()=>{
 const toggle=rulesFor("design-system/styles/form-controls.css",".form-disclosure-toggle").find(rule=>!rule.media);
 assert.equal(toggle.values["min-height"],"var(--control-height)","Direcciones e Información adicional medían 37 px");
 const zoom=rulesFor("design-system/styles/location-map.css",".location-map-canvas .mapboxgl-ctrl-group button").find(rule=>touch(rule.media));
 assert.deepEqual([zoom?.values.width,zoom?.values.height],["var(--size-44)","var(--size-44)"]);
 const suggestion=rulesFor("design-system/styles/location-map.css",".location-map-suggestion").find(rule=>touch(rule.media));
 assert.equal(suggestion?.values["min-height"],"var(--size-44)");
});

test("Reservas presenta skeleton, error con Reintentar y vacío compartidos",()=>{
 const page=read("modules/operations/reservations/presentation/reservations-page.tsx");
 assert.match(page,/list\.isLoading\?<TableSkeleton columns=\{columns\} label="Cargando reservas"\/>/);
 assert.match(page,/<ErrorState title="No pudimos cargar las reservas" text=\{list\.error\.message\} onRetry=\{\(\)=>void list\.refetch\(\)\}/);
 assert.match(page,/<EmptyState icon="search" title="Sin coincidencias"/);
 assert.match(page,/<EmptyState icon="clock" title="Aún no hay reservas"/);
 assert.doesNotMatch(page,/Cargando reservas…/,"un texto suelto no es skeleton");
});

test("Compras aplica su variante móvil: pantalla completa, una columna y 16 px",()=>{
 const file="modules/supply/purchases/presentation/purchases.css";
 const narrow=selector=>rulesFor(file,selector).find(rule=>rule.media==="(max-width:760px)")?.values;
 assert.equal(narrow(".purchase-detail-body").padding,"var(--space-3) var(--space-4)");
 assert.equal(narrow(".purchase-form-body").padding,"var(--space-3) 0","el form heredado ya aporta 16 px");
 assert.equal(narrow(".purchase-item-body").padding,"var(--space-3) 0");
 assert.equal(narrow(".purchase-order-meta-grid")["grid-template-columns"],"1fr");
});

// Guarda general: una regla base escrita después de su @media la anula en silencio
// (Compras perdía pantalla completa, una columna y padding). `.sidebar` queda pendiente:
// corregirla cambia el ancho del drawer (250 → min(320 px, 86vw)) y requiere aprobación.
const knownCascadeDebt=new Set(["shell/styles/shell.css|.sidebar"]);
test("ninguna variante @media queda anulada por una regla base posterior",async()=>{
 const {readdirSync}=await import("node:fs");
 const {join,relative}=await import("node:path");
 const {fileURLToPath}=await import("node:url");
 const src=fileURLToPath(new URL("../src/",import.meta.url));
 const files=[];
 const walk=dir=>{for(const entry of readdirSync(dir,{withFileTypes:true})){const path=join(dir,entry.name);if(entry.isDirectory())walk(path);else if(entry.name.endsWith(".css"))files.push(path)}};
 walk(src);
 const problems=[];
 for(const path of files){
  const file=relative(src,path).replaceAll("\\","/");
  const nodes=postcss.parse(readFileSync(path,"utf8")).nodes;
  nodes.forEach((node,index)=>{
   if(node.type!=="atrule"||node.name!=="media"||/prefers-reduced-motion/.test(node.params))return;
   node.walkRules(rule=>{
    const selectors=rule.selector.split(",").map(part=>part.trim());
    const props=Object.keys(declarations(rule));
    for(const later of nodes.slice(index+1)){
     if(later.type!=="rule")continue;
     const shared=later.selector.split(",").map(part=>part.trim()).filter(selector=>selectors.includes(selector));
     const clash=props.filter(prop=>prop in declarations(later));
     for(const selector of shared)if(clash.length&&!knownCascadeDebt.has(`${file}|${selector}`))problems.push(`${file}: ${selector} (${clash.join(", ")}) @media ${node.params}`);
    }
   });
  });
 }
 assert.deepEqual(problems,[]);
});
