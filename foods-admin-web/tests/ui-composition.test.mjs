import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync,readdirSync} from "node:fs";
import {join} from "node:path";
import {fileURLToPath} from "node:url";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
import postcss from "postcss";
import {resolveTokens} from "./helpers/design-tokens.mjs";

const require=createRequire(import.meta.url);
const read=file=>readFileSync(new URL(`../src/${file}`,import.meta.url),"utf8");

test("Productos precarga su composición y editor antes de abrir el diálogo diferido",()=>{
 const catalog=read("modules/menu/products/presentation/catalog-manager.tsx");
 assert.match(catalog,/import "\.\/product-wizard\.css"/);
 assert.match(catalog,/import "@\/design-system\/styles\/rich-text-editor\.css"/);
 assert.match(read("app/layout.tsx"),/import "@\/design-system\/styles\/image-drop-tile\.css"/);
 const dialog=read("modules/menu/products/presentation/product-dialog.tsx");
 assert.match(dialog,/<FormField as="div" className="product-description span-2" label="Descripción">/);
 assert.doesNotMatch(dialog,/showDescription/);
});

test("la homologación visual conserva el orden aprobado de los campos",()=>{
 function labels(file,name){
  const source=ts.createSourceFile(file,read(file),ts.ScriptTarget.Latest,true,ts.ScriptKind.TSX);
  const fn=source.statements.find(node=>ts.isFunctionDeclaration(node)&&node.name?.text===name),result=[];
  assert.ok(fn);
  function visit(node){if(ts.isJsxOpeningElement(node)&&node.tagName.getText(source)==="FormField"){
   const label=node.attributes.properties.find(attr=>ts.isJsxAttribute(attr)&&attr.name.getText(source)==="label");
   if(label?.initializer&&ts.isStringLiteral(label.initializer))result.push(label.initializer.text);
  }ts.forEachChild(node,visit);}
  visit(fn);return result;
 }
 assert.deepEqual(labels("modules/expenses/presentation/expenses-page.tsx","ExpenseEditor"),["Categoría","Fecha","Descripción","Importe","Referencia","Notas"]);
 assert.deepEqual(labels("modules/supply/purchases/presentation/purchases-page.tsx","SupplierDialog"),["Nombre o razón social","RUC","Teléfono","Correo electrónico"]);
});

test("los formularios modales tienen un cuerpo propio y el form no añade otra sangría",()=>{
 const files=[];
 function walk(dir){for(const e of readdirSync(dir,{withFileTypes:true})){const file=join(dir,e.name);if(e.isDirectory())walk(file);else if(file.endsWith(".tsx"))files.push(file);}}
 walk(fileURLToPath(new URL("../src/modules/",import.meta.url)));
 const legacy=[];
 for(const file of files){
  const sf=ts.createSourceFile(file,readFileSync(file,"utf8"),ts.ScriptTarget.Latest,true,ts.ScriptKind.TSX);
  function visit(node){
   if(ts.isJsxElement(node)&&node.openingElement.tagName.getText(sf)==="form"){
    let ancestor=node.parent;
    while(ancestor){if(ts.isJsxElement(ancestor)&&/crud-modal|^<Modal\b/.test(ancestor.openingElement.getText(sf))){legacy.push(file);break;}ancestor=ancestor.parent;}
   }
   ts.forEachChild(node,visit);
  }
  visit(sf);
 }
 assert.deepEqual(legacy,[],"los formularios heredados duplicaban padding y desplazaban el pie");
 const css=postcss.parse(read("design-system/styles/modal.css"));
 css.walkRules(rule=>{if(rule.selector===".crud-modal>:is(form,.modal-form)")assert.ok(!rule.nodes.some(d=>d.prop==="padding"||d.prop==="overflow-y"));});
 const sections=postcss.parse(read("design-system/styles/form-section.css"));
 sections.walkRules(rule=>{if(rule.selector===".form-sections")assert.ok(!rule.nodes.some(d=>d.prop?.startsWith("padding")),"FormSection agrupa, ModalBody aporta el margen");});
});

test("las columnas fijas comparten cabecera índigo, sin parches de grafito al desplazarse",()=>{
 const ast=postcss.parse(read("design-system/styles/data-table.css"));
 const headers=[];
 ast.walkRules(rule=>{if(rule.selector.includes("thead"))rule.walkDecls("background",d=>headers.push(d.value));});
 assert.ok(headers.length>=4);
 assert.ok(headers.every(value=>value==="var(--table-header-background)"));
 assert.equal(resolveTokens("var(--table-header-background)"),resolveTokens("var(--primary-700)"));
});

test("la pestaña seleccionada permanece visible al abrir un enlace y al cambiar el ancho",()=>{
 let resize,cleanup,disconnected=false;
 let bounds={left:300,right:420};
 const container={scrollLeft:0,getBoundingClientRect:()=>({left:0,right:350}),querySelector:()=>({getBoundingClientRect:()=>bounds})};
 const exports={};
 vm.runInNewContext(ts.transpileModule(read("design-system/tabs.tsx"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{
  exports,
  ResizeObserver:class{constructor(callback){resize=callback;}observe(){}disconnect(){disconnected=true;}},
  require:name=>name.endsWith(".css")?{}:name==="react"?{useRef:()=>({current:container}),useEffect:effect=>{cleanup=effect();}}:name==="./icons"?{Icon:"Icon"}:require(name),
 });
 exports.Tabs({label:"Caja",items:[{value:"expenses",label:"Gastos",icon:"receipt"}],value:"expenses",onChange(){}});
 assert.equal(container.scrollLeft,70,"abrir Gastos no debe dejar la selección fuera del borde derecho");
 bounds={left:-30,right:50};resize();assert.equal(container.scrollLeft,40,"al volver atrás revela el borde izquierdo");
 bounds={left:80,right:180};resize();assert.equal(container.scrollLeft,40,"una pestaña visible no mueve la página");
 cleanup();assert.ok(disconnected);
});
