import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync,readdirSync} from "node:fs";
import {join,relative} from "node:path";
import {fileURLToPath} from "node:url";
import postcss from "postcss";

const src=fileURLToPath(new URL("../src/",import.meta.url));
const cssFiles=dir=>readdirSync(dir,{withFileTypes:true}).flatMap(entry=>entry.isDirectory()?cssFiles(join(dir,entry.name)):entry.name.endsWith(".css")?[join(dir,entry.name)]:[]);
const declarations=rule=>Object.fromEntries(rule.nodes.filter(node=>node.type==="decl").map(node=>[node.prop,node.value]));

// Un input oculto con position:absolute sin ancla se refiere a un ancestro lejano (por ejemplo .crud-modal con
// overflow:hidden). Al marcarlo, el foco desplaza ese ancestro y el modal queda en blanco con el pie arriba.
test("los inputs ocultos con position:absolute se anclan a su etiqueta con position:relative",()=>{
 const hidden=[];
 for(const file of cssFiles(src)){
  const rules=[];
  postcss.parse(readFileSync(file,"utf8")).walkRules(rule=>rules.push(rule));
  for(const rule of rules){
   const values=declarations(rule);
   if(values.position!=="absolute"||values.opacity!=="0")continue;
   for(const selector of rule.selectors){
    const match=selector.match(/^(.+?)\s*>?\s*input(?:\[[^\]]*\])?$/);
    if(!match)continue;
    const parent=match[1].trim();
    const anchored=rules.some(candidate=>candidate.selectors.includes(parent)&&["relative","absolute","fixed","sticky"].includes(declarations(candidate).position));
    hidden.push(selector);
    assert.ok(anchored,`${relative(src,file)}: ${parent} debe declarar position:relative para anclar ${selector}`);
   }
  }
 }
 assert.ok(hidden.includes(".combo-toggle > input"),"el interruptor Obligatorio de menús y combos queda cubierto");
});

test("los tooltips de acciones de fila crecen hacia la izquierda y no desbordan la tabla",()=>{
 const css=readFileSync(new URL("../src/design-system/styles/table-actions.css",import.meta.url),"utf8");
 assert.ok(css.includes(":is(.standard-actions,.table-actions,td:last-child) [data-tooltip]:after{left:auto;right:0;transform:translateY(var(--size-4))}"),"anclado al borde derecho del botón");
 assert.match(css,/@media \(hover:hover\) and \(pointer:fine\)\{[^@]*:is\(\.standard-actions,\.table-actions,td:last-child\) \[data-tooltip\]:hover:after\{transform:translateY\(0\)\}/,"al mostrarse no vuelve a centrarse");
});
