import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import test from "node:test";
import vm from "node:vm";
import postcss from "postcss";
import ts from "typescript";

const root=new URL("../",import.meta.url),require=createRequire(import.meta.url);
const read=file=>readFileSync(new URL(file,root),"utf8");
const exports={};
vm.runInNewContext(ts.transpileModule(read("src/design-system/page-header.tsx"),{
  compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX},
}).outputText,{exports,require:name=>name.endsWith(".css")?{}:name==="@/design-system/icons"?{Icon:"Icon"}:require(name)});

function nodes(tree){
  const result=[];
  function visit(node){
    if(Array.isArray(node)){node.forEach(visit);return;}
    if(!node||typeof node!=="object")return;
    result.push(node);visit(node.props?.children);
  }
  visit(tree);return result;
}
function label(node){
  if(Array.isArray(node))return node.map(label).join("");
  if(typeof node==="string"||typeof node==="number")return String(node);
  return node?.props?label(node.props.children):"";
}

test("la paginación global no conserva reglas antiguas ni parches de cascada",()=>{
  const ast=postcss.parse(read("src/styles/globals.css"));
  ast.walkRules(rule=>{
    for(const selector of rule.selectors){
      assert.equal(/\.management\s*>?\s*footer|\.catalog-pagination|\.pagination-meta/.test(selector),false,`regla antigua de paginación: ${selector}`);
    }
    if(rule.selector.includes(".standard-pagination"))rule.walkDecls(declaration=>assert.equal(Boolean(declaration.important),false,rule.selector));
  });
  const specialized=postcss.parse(read("src/design-system/styles/pagination.css"));
  specialized.walkRules(rule=>{
    if(rule.selector.includes("pagination"))rule.walkDecls(declaration=>assert.equal(Boolean(declaration.important),false,declaration.toString()));
  });
  assert.equal(read("src/design-system/page-header.tsx").includes("catalog-pagination"),false);
});

test("Pagination conserva Siguiente, páginas y límites en todas sus variantes",()=>{
  for(const mode of ["full","simple","compact"]){
    const changes=[];
    const render=page=>exports.Pagination({page,size:10,total:12,mode,onPage:value=>changes.push(value),onSize:()=>{}});
    const first=nodes(render(1));
    const nav=first.filter(node=>node.type==="button"&&node.props.className==="pagination-nav");
    assert.deepEqual(nav.map(label),["Anterior","Siguiente"]);
    assert.equal(nav[0].props.disabled,true);
    assert.equal(nav[1].props.disabled,false);
    assert.ok(nodes(nav[1]).some(node=>node.type==="Icon"&&node.props.name==="chevron"));
    nav[1].props.onClick();assert.deepEqual(changes,[2]);
    const last=nodes(render(2)).filter(node=>node.type==="button"&&node.props.className==="pagination-nav");
    assert.equal(last[0].props.disabled,false);
    assert.equal(last[1].props.disabled,true);
    if(mode!=="compact")assert.ok(label(render(2)).includes("Mostrando 11–12 de 12"));
    if(mode!=="simple")assert.equal(first.find(node=>node.props["aria-current"]==="page")?.props.children,1);
  }
});

test("los estados de navegación comparten colores y geometría, sin parches por tabla",()=>{
  const ast=postcss.parse(read("src/design-system/styles/pagination.css"));
  const declarations=selector=>{
    const rule=ast.nodes.find(node=>node.type==="rule"&&node.selector===selector);
    assert.ok(rule,selector);
    return Object.fromEntries(rule.nodes.filter(node=>node.type==="decl").map(node=>[node.prop,node.value]));
  };
  const normal=declarations(".pagination-nav");
  assert.equal(normal.background,"var(--primary-600)");
  assert.equal(normal.color,"var(--surface)");
  assert.equal(normal.height,"var(--control-height)");
  assert.equal(normal["border-radius"],"var(--radius-control)");
  assert.equal(declarations(".pagination-nav:hover:not(:disabled)").background,"var(--primary-700)");
  assert.equal(declarations(".pagination-nav:disabled").color,"var(--ink-400)");
  assert.equal(declarations(".pagination-nav:disabled").background,"var(--palette-blue-95)");
});
