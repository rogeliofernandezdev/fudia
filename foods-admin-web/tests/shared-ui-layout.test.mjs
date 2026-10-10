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
test("Productos y Mesas usan Tabs; no conservan implementación ni CSS de pestañas antiguos",()=>{
 for(const file of ["modules/menu/products/presentation/catalog-manager.tsx","modules/operations/tables/presentation/tables-manager.tsx"]){
  const source=read(file);assert.match(source,/<Tabs label=/);assert.doesNotMatch(source,/className="catalog-tabs"/);
 }
 assert.doesNotMatch(read("styles/globals.css"),/\.catalog-tabs(?!-row)/);
});
test("selección suave, contador contrastado y altura compartida sin movimiento al cambiar pestañas",()=>{
 const css=sheet("design-system/styles/tabs.css");
 assert.equal(declaration(css,".ds-tabs button","min-height"),"var(--control-height)");
 assert.equal(declaration(css,".ds-tabs button.active","background"),"var(--primary-100)");
 assert.equal(declaration(css,".ds-tabs button.active","color"),"var(--primary-700)");
 assert.equal(declaration(css,".ds-tabs button.active b","background"),"var(--primary-600)");
 assert.doesNotMatch(declaration(css,".ds-tabs button","transition"),/transform|all/);
 assert.equal(declaration(css,".ds-tabs","overscroll-behavior-x"),"contain");
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
 const view=exports.Tabs({label:"Vista",items:[{value:"a",label:"Productos"},{value:"b",label:"Categorías"},{value:"c",label:"Disponibilidad"}],value:"b",onChange:value=>changed.push(value)});
 assert.deepEqual(view.props.children.map(node=>node.props.tabIndex),[-1,0,-1]);
 for(const [index,key,target] of [[1,"ArrowRight",2],[0,"ArrowLeft",2],[2,"Home",0],[0,"End",2]]){
  let prevented=false;view.props.children[index].props.onKeyDown({key,preventDefault:()=>{prevented=true;}});
  assert.ok(prevented);assert.equal(focused.at(-1),target);assert.equal(changed.at(-1),["a","b","c"][target]);
 }
});
