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

test("en móvil el editor de roles es pantalla completa con un solo scroll y pestañas fijas",()=>{
 const mobile=css.slice(css.indexOf("@media(max-width:600px){"));
 assert.match(mobile,/\.role-editor-modal\{width:100%;height:100dvh;max-height:100dvh;border-radius:0\}/,"pantalla completa como la Comanda");
 assert.match(mobile,/\.role-editor-body\{[^}]*overflow:visible/,"sin scroll anidado: desplaza el form");
 assert.match(mobile,/\.role-editor-perms-panel\{[^}]*overflow:clip/,"clip conserva el radio sin romper sticky");
 assert.match(mobile,/\.re-perm-tabs\{position:sticky;top:calc\(var\(--space-4\) \* -1\)/,"pestañas y búsqueda fijas arriba");
 assert.match(mobile,/\.re-perm-tabs \.ds-tabs b\{display:none\}/,"sin contadores en las pestañas móviles");
 assert.match(mobile,/\.re-perm-accordion\{overflow:visible\}/);
 assert.doesNotMatch(css.slice(0,css.indexOf("@media")),/\.re-perm-tabs \.ds-tabs b/,"escritorio conserva los contadores");
});

test("el rol editado sigue visible: su nombre es el título de la cabecera fija",()=>{
 assert.match(manager,/const editorTitle=roleName\|\|"Nuevo rol";/);
 assert.match(manager,/<ModalHeader title=\{editorTitle\} titleId="role-editor-title" icon="lock" subtitle=\{editorSubtitle\}/);
 assert.match(manager,/`\$\{initial\.id\?"EDITAR ":""\}ROL \$\{predefined\?"PREDETERMINADO":"PERSONALIZADO"\}`/);
 assert.match(css,/\.re-perm-tabs\{position:sticky;top:calc\(var\(--space-6\) \* -1\)/,"escritorio: pestañas, filtro y búsqueda fijas bajo la cabecera");
 assert.match(css,/\.re-acc-items\{columns:2;/,"permisos por columnas, no en zigzag");
 assert.ok(manager.includes('placeholder={permTab==="menu"?"Buscar opción…":"Buscar permiso…"}'),"placeholders que caben a 390 px");
});
