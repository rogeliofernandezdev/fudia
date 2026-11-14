import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
import React from "react";
import {renderToStaticMarkup} from "react-dom/server";

const require=createRequire(import.meta.url),exports={};
const source=readFileSync("src/modules/concierge-web/presentation/restaurant-chat.tsx","utf8");
vm.runInNewContext(ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,esModuleInterop:true}}).outputText,{exports,require:name=>name.endsWith(".css")||name.endsWith("order-tracking")||name.includes("chat-api")||name.startsWith("@/design-system")?{}:require(name)});
const render=(text,assistant=true)=>renderToStaticMarkup(React.createElement(exports.ChatMessageText,{text,assistant}));

test("el chat destaca la acción con negrita real, sin marcas visibles",()=>{
 assert.equal(render("Revisa tu pedido y toca **Confirmar pedido** para enviarlo."),"Revisa tu pedido y toca <strong>Confirmar pedido</strong> para enviarlo.");
 assert.equal(render("Toca **Carta** o abre **Mi pedido**."),"Toca <strong>Carta</strong> o abre <strong>Mi pedido</strong>.");
 assert.equal(render("Abre «Mi pedido». Responde «sí»."),"Abre <strong>Mi pedido</strong>. Responde sí.");
 assert.equal(render("**sin cerrar"),"sin cerrar");
});

test("el formato no interpreta HTML ni cambia los mensajes del cliente",()=>{
 const markup=render('Toca **<img src=x onerror=alert(1)>**.');
 assert.ok(markup.includes("<strong>&lt;img"));
 assert.ok(!markup.includes("<img"));
 assert.equal(render("Usa «mi nombre» y **mis palabras**",false),"Usa «mi nombre» y **mis palabras**");
});
