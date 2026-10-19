import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
import postcss from "postcss";
import test from "node:test";

const require=createRequire(import.meta.url);
const read=file=>readFileSync(new URL(`../src/${file}`,import.meta.url),"utf8");
const sheet=file=>postcss.parse(read(file));
function declaration(ast,selector,prop,context="root"){
 let value;
 ast.walkRules(rule=>{
  const parent=rule.parent.type==="atrule"?rule.parent.params:"root";
  if(rule.selectors.includes(selector)&&parent===context)rule.walkDecls(prop,d=>{value=d.value;});
 });return value;
}
test("los botones móviles comparten radio de píldora sin cambiar la geometría de campos ni cierre",()=>{
 const css=sheet("styles/globals.css");
 assert.equal(declaration(css,".button","border-radius"),"var(--radius-control)");
 assert.equal(declaration(css,".button","border-radius","(width<=600px)"),"var(--radius-pill)");
 const modal=sheet("design-system/styles/modal.css");
 assert.equal(declaration(modal,".modal-close","border-radius"),"var(--radius-control)");
 for(const [file,selector] of [["modules/menu/availability/presentation/product-availability.css",".availability-actions .button"],["modules/setup/presentation/restaurant-setup.css",".restaurant-setup .button"],["modules/setup/presentation/restaurant-setup-tour.css",".setup-tour .button"]])assert.equal(declaration(sheet(file),selector,"border-radius"),undefined,"el módulo hereda la forma del botón compartido");
});
test("franjas de lectura igualan etiqueta y valor y alinean la fecha al extremo derecho",()=>{
 const css=sheet("design-system/styles/modal.css");
 assert.equal(declaration(css,".modal-inline-metadata :is(dt,dd)","font-size"),"var(--font-size-base)");
 assert.equal(declaration(css,".modal-inline-metadata :is(dt,dd)","line-height"),"var(--line-height-normal)");
 assert.equal(declaration(css,".modal-inline-date","margin-inline-start"),"auto");
 assert.equal(declaration(css,".modal-inline-date","text-align"),"right");
 assert.equal(declaration(css,".modal-inline-metadata","flex-wrap"),"wrap");
 const source=read("modules/operations/orders/presentation/order-detail-dialog.tsx");
 assert.match(source,/order-detail-information modal-inline-metadata/);
 assert.match(source,/order-detail-opened modal-inline-date/);
 assert.equal(declaration(sheet("modules/operations/orders/presentation/order-detail.css"),".order-attribution dt","font-size"),undefined);
});

test("acciones móviles aprovechan toda su fila sin huecos en grupos anidados",()=>{
 const css=sheet("design-system/styles/modal.css"),context="(width<=600px)";
 assert.equal(declaration(css,".modal-action-layout","width",context),"100%");
 assert.equal(declaration(css,".modal-action-layout","flex-wrap"),"wrap");
 assert.equal(declaration(css,".modal-action-layout>.modal-action-subgroup","display",context),"contents");
 assert.equal(declaration(css,".modal-footer .modal-action-layout .button","flex",context),"1 1 var(--size-150)");
 assert.equal(declaration(css,".modal-footer .modal-action-layout .modal-action-main","flex-basis",context),"100%");
 const local=read("modules/operations/orders/presentation/order-detail.css");
 assert.doesNotMatch(local,/\.order-detail-buttons\{[^}]*grid-template-columns/);
 assert.match(read("modules/operations/orders/presentation/order-detail-dialog.tsx"),/order-detail-secondary-actions modal-action-subgroup/);
});

test("Productos y Mesas usan Tabs; no conservan implementación ni CSS de pestañas antiguos",()=>{
 for(const file of ["modules/menu/products/presentation/catalog-manager.tsx","modules/operations/tables/presentation/tables-manager.tsx"]){
  const source=read(file);assert.match(source,/<Tabs label=/);assert.doesNotMatch(source,/className="catalog-tabs"/);
 }
 assert.doesNotMatch(read("styles/globals.css"),/\.catalog-tabs(?!-row)/);
});
test("pestañas compartidas con pista contrastada, selección blanca y contadores índigo",()=>{
 const css=sheet("design-system/styles/tabs.css");
 assert.equal(declaration(css,".ds-tabs button","min-height"),"var(--control-height)");
 assert.equal(declaration(css,".ds-tabs button.active","background"),"var(--surface)");
 assert.equal(declaration(css,".ds-tabs button.active","color"),"var(--primary-700)");
 assert.equal(declaration(css,".ds-tabs b","background"),"var(--primary-600)");
 assert.equal(declaration(css,".ds-tabs b","color"),"var(--surface)");
 assert.equal(declaration(css,".ds-tabs","background"),"var(--cloud-150)");
 assert.equal(declaration(css,".ds-tabs","padding"),"var(--space-1)");
 assert.equal(declaration(css,".ds-tabs","border-radius"),"var(--radius-pill)");
 assert.equal(declaration(css,".ds-tabs button","border-radius"),"var(--radius-pill)");
 assert.doesNotMatch(declaration(css,".ds-tabs button","transition"),/transform|all/);
 assert.equal(declaration(css,".ds-tabs","overscroll-behavior-x"),"contain");
});
test("la barra compartida alinea acciones en escritorio y les da una fila completa en móvil",()=>{
 const css=sheet("design-system/styles/tabs.css");
 assert.equal(declaration(css,".ds-tabs-bar","display"),"flex");
 assert.equal(declaration(css,".ds-tabs-bar","align-items"),"center");
 assert.equal(declaration(css,".ds-tabs-actions","margin-left"),"auto");
 assert.equal(declaration(css,".ds-tabs-bar","flex-direction","(width<=600px)"),"column");
 assert.equal(declaration(css,".ds-tabs-bar>:first-child","width","(width<=600px)"),"100%");
 assert.equal(declaration(css,".ds-tabs-actions","width","(width<=600px)"),"100%");
 assert.equal(declaration(css,".ds-tabs-actions .button","flex","(width<=600px)"),"1 1 var(--size-150)");
 for(const file of ["operations/orders/presentation/orders-manager.tsx","menu/products/presentation/catalog-manager.tsx","operations/tables/presentation/tables-manager.tsx","operations/cash/presentation/cash-page.tsx","supply/purchases/presentation/purchases-page.tsx","expenses/presentation/expenses-page.tsx","identity/presentation/users-roles-manager.tsx","organizations/presentation/organization-admin.tsx","platform/presentation/platform-plans-page.tsx"]){
  assert.match(read(`modules/${file}`),/<TabsBar (sticky )?action=/);
 }
});
test("con teclado abierto en móvil el modal cede alto al formulario sin perder acciones",()=>{
 const css=sheet("design-system/styles/modal.css");const keyboard="(width<=600px) and (height<=560px)";
 assert.equal(declaration(css,".crud-modal.size-md","max-height",keyboard),"100dvh");
 assert.equal(declaration(css,".modal-header","min-height",keyboard),"var(--size-52)");
 assert.equal(declaration(css,".modal-header h2","font-size",keyboard),"var(--font-size-md)");
 assert.equal(declaration(css,".modal-context","display",keyboard),"none");
 assert.equal(declaration(css,".modal-footer","display",keyboard),undefined,"el pie sigue visible");
 assert.equal(declaration(css,".modal-footer","padding-block",keyboard),"var(--space-2) calc(var(--space-2) + var(--safe-area-bottom))");
});
test("filtros de listas largas quedan fijos en móvil bajo la barra superior",()=>{
 const tabs=sheet("design-system/styles/tabs.css");
 assert.equal(declaration(tabs,".ds-tabs-bar.is-sticky","display","(width<=600px)"),"contents");
 assert.equal(declaration(tabs,".ds-tabs-bar.is-sticky>.ds-tabs-sticky","position","(width<=600px)"),"sticky");
 assert.equal(declaration(tabs,".ds-tabs-bar.is-sticky>.ds-tabs-sticky","top","(width<=600px)"),"var(--size-64)");
 assert.match(read("modules/operations/orders/presentation/orders-manager.tsx"),/<TabsBar sticky action=/);
 for(const file of ["operations/cash/presentation/cash-page.tsx","menu/products/presentation/catalog-manager.tsx"])assert.doesNotMatch(read(`modules/${file}`),/<TabsBar sticky/,"las pestañas de secciones no se fijan");
 const sales=sheet("modules/sales/presentation/sales.css");
 assert.equal(declaration(sales,".sales-periods","position","(max-width:600px)"),"sticky");
 assert.equal(declaration(sales,".sales-periods","top","(max-width:600px)"),"var(--size-64)");
 assert.equal(declaration(sales,".sales-panel","overflow","(max-width:600px)"),"clip");
 assert.equal(declaration(sheet("design-system/styles/data-table.css"),".table-wrap","isolation"),"isolate");
});
test("pestañas móviles muestran pares completos con ajuste por grupo y etiquetas largas sin recorte",()=>{
 const css=sheet("design-system/styles/tabs.css");
 assert.equal(declaration(css,".ds-tabs button","flex","(width<=600px)"),"0 0 calc((100% - var(--space-1)) / 2)");
 assert.equal(declaration(css,".ds-tabs","scroll-snap-type","(width<=600px)"),"x mandatory");
 assert.equal(declaration(css,".ds-tabs button:nth-child(odd)","scroll-snap-align","(width<=600px)"),"start");
 assert.equal(declaration(css,".ds-tabs button>span","overflow-wrap","(width<=600px)"),"break-word");
 assert.equal(declaration(css,".ds-tabs button>span","hyphens","(width<=600px)"),"auto");
 assert.equal(declaration(css,".ds-tabs b","display","(width<=600px)"),"none","sin contadores en pestañas móviles");
 assert.equal(declaration(css,".ds-tabs b","display"),"grid","escritorio conserva los contadores");
 assert.equal(declaration(css,".ds-tabs:has(>button:last-child:nth-child(odd):not(:only-child))::after","flex","(width<=600px)"),"0 0 calc((100% - var(--space-1)) / 2)");
 assert.equal(declaration(css,".ds-tabs button","min-height","(width<=600px)"),"var(--size-44)");
 assert.equal(declaration(css,".ds-tabs button","min-width"),"max-content");
 assert.equal(declaration(css,".ds-tabs","overflow-x"),"auto");
 assert.equal(declaration(css,".ds-tabs","scrollbar-width"),"none");
 assert.equal(declaration(css,".ds-tabs button:focus-visible","outline"),"var(--stroke-2) solid var(--primary-600)");
});
test("cabecera, cuerpo y pie comparten margen y el cierre conserva 44 px en todos los tamaños",()=>{
 const css=sheet("design-system/styles/modal.css");
 assert.equal(declaration(css,".crud-modal>header","padding"),"var(--space-2) var(--space-6)");
 assert.equal(declaration(css,".crud-modal>header","min-height"),"var(--size-64)");
 assert.equal(declaration(css,".crud-modal>header","background"),"var(--primary-700)");
 assert.doesNotMatch(read("design-system/styles/modal.css"),/modal-accent/);
 assert.doesNotMatch(read("design-system/modal.tsx"),/modal-accent/);
 assert.equal(declaration(css,".crud-modal>header h2","color"),"var(--surface)");
 assert.equal(declaration(css,".modal-close","color"),"var(--surface)");
 assert.equal(declaration(css,".modal-close:focus-visible","outline"),"var(--stroke-2) solid var(--surface)");
 assert.equal(declaration(css,".modal-body","padding"),"var(--space-6)");
 assert.equal(declaration(css,".modal-footer","padding"),"var(--space-4) var(--space-6) calc(var(--space-4) + var(--safe-area-bottom))");
 for(const prop of ["width","height"])assert.equal(declaration(css,".modal-close",prop),"var(--size-44)");
 assert.equal(declaration(css,".modal-footer>.modal-footer-note","flex-basis"),"100%");
});
test("icono blanco sin recuadro y X blanca con solo contorno y sin hover de color",()=>{
 const css=sheet("design-system/styles/modal.css");
 assert.equal(declaration(css,".modal-title-icon","background"),"transparent");
 assert.equal(declaration(css,".modal-title-icon","color"),"var(--surface)");
 assert.equal(declaration(css,".modal-close","background"),"transparent");
 assert.equal(declaration(css,".modal-close","border"),"0");
 assert.equal(declaration(css,".modal-close::before","border"),"var(--stroke-1) solid var(--surface)");
 for(const prop of ["width","height"])assert.equal(declaration(css,".modal-close::before",prop),"var(--size-32)");
 assert.equal(declaration(css,".modal-close::before","pointer-events"),"none");
 for(const selector of [".crud-modal>header .modal-title-icon>svg",".crud-modal>header>button>svg"]){
  for(const prop of ["width","height"])assert.equal(declaration(css,selector,prop),"var(--size-20)");
 }
 assert.equal(declaration(css,".modal-close:hover:not(:disabled)","background","(hover:hover) and (pointer:fine)"),undefined);
 assert.doesNotMatch(declaration(css,".modal-close","transition"),/background|color/);
 assert.equal(declaration(css,".modal-close:focus-visible","outline"),"var(--stroke-2) solid var(--surface)");
});

test("al guardar la X permanece blanca y opaca, y Producto no altera márgenes del modal",()=>{
 const css=sheet("design-system/styles/modal.css");
 assert.equal(declaration(css,".modal-close:disabled","opacity"),"1");
 assert.equal(declaration(css,".modal-close:disabled","cursor"),"not-allowed");
 assert.equal(declaration(css,".modal-close:disabled","color"),undefined);
 const local=read("modules/menu/products/presentation/product-wizard.css");
 assert.doesNotMatch(local,/\.crud-modal\.product-form>form(?:\{|>footer)/);
 assert.doesNotMatch(local,/\.product-form-body\{[^}]*padding/);
});
test("modal completo no agrega espacio fuera del viewport y conserva las áreas seguras",()=>{
 const css=sheet("design-system/styles/modal.css");
 assert.equal(declaration(css,".modal-backdrop","padding","(width<=600px)"),"0");
 assert.equal(declaration(css,".crud-modal.size-lg","height","(width<=600px)"),"100dvh");
 assert.match(declaration(css,".crud-modal.size-lg>header","padding-top","(width<=600px)"),/--safe-area-top/);
 assert.equal(declaration(css,".modal-body","overscroll-behavior"),"contain");
});
test("Tabs conserva flechas, Home, End y un único elemento en el orden de Tab",()=>{
 const focused=[],changed=[];
 const elements=Array.from({length:3},(_,i)=>({focus:()=>focused.push(i)}));
 const exports={};
 vm.runInNewContext(ts.transpileModule(read("design-system/tabs.tsx"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:name=>name.endsWith(".css")?{}:name==="react"?{useRef:()=>({current:{querySelectorAll:()=>elements}})}:name==="./icons"?{Icon:"Icon"}:require(name)});
 const view=exports.Tabs({label:"Vista",items:[{value:"a",label:"Productos",icon:"utensils"},{value:"b",label:"Categorías",icon:"layers"},{value:"c",label:"Disponibilidad",icon:"availability"}],value:"b",onChange:value=>changed.push(value)});
 assert.deepEqual(view.props.children.map(node=>node.props.tabIndex),[-1,0,-1]);
 assert.deepEqual(view.props.children.map(node=>node.props.children[0].props.name),["utensils","layers","availability"]);
 assert.ok(view.props.children.every(node=>node.props.children[0].props.size===16));
 for(const [index,key,target] of [[1,"ArrowRight",2],[0,"ArrowLeft",2],[2,"Home",0],[0,"End",2]]){
  let prevented=false;view.props.children[index].props.onKeyDown({key,preventDefault:()=>{prevented=true;}});
  assert.ok(prevented);assert.equal(focused.at(-1),target);assert.equal(changed.at(-1),["a","b","c"][target]);
 }
});
test("todas las pestañas requieren iconos y Cocina no los oculta en móvil",()=>{
 const source=read("design-system/tabs.tsx");
 assert.match(source,/icon:IconName/);
 assert.doesNotMatch(source,/icon\?:IconName|item\.icon&&/);
 assert.doesNotMatch(read("modules/operations/kitchen/presentation/kitchen.css"),/\.ds-tabs\.kitchen-mobile-tabs button svg\{display:none\}/);
 assert.doesNotMatch(read("modules/operations/kitchen/presentation/kitchen.css"),/\.ds-tabs\.kitchen-mobile-tabs button\{[^}]*flex:/);
});
