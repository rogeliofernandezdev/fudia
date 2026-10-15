import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";

const read=file=>readFileSync(new URL("../"+file,import.meta.url),"utf8");
const css=read("src/modules/identity/presentation/role-editor.css");
const manager=read("src/modules/identity/presentation/users-roles-manager.tsx");
const rule=selector=>{
 const match=css.match(new RegExp(`(?:^|\\n|\\})${selector.replace(/[.*+?^${}()|[\]\\]/g,"\\$&")}\\{([^}]*)\\}`));
 assert.ok(match,`falta la regla ${selector}`);
 return match[1];
};

test("los checks del editor de roles no desplazan ni vacían el modal al marcarse",()=>{
 assert.match(manager,/<label key=\{item\.value\} className=\{`re-acc-item[^>]*><input type="checkbox"/,"el checkbox nativo vive dentro de la etiqueta re-acc-item");
 assert.match(rule(".re-acc-item"),/(?:^|;)position:relative(?:;|$)/,"la etiqueta es el bloque contenedor del checkbox oculto, no .crud-modal");
 const input=rule(".re-acc-item input");
 assert.match(input,/position:absolute/);
 assert.match(input,/opacity:0/);
 assert.match(input,/width:var\(--size-1\)/,"el checkbox oculto no desborda la etiqueta");
 assert.match(input,/height:var\(--size-1\)/);
 assert.match(input,/margin:0/);
});
