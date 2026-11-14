import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
import React from "react";
import {renderToStaticMarkup} from "react-dom/server";

const require=createRequire(import.meta.url);
const source=readFileSync("src/modules/concierge-web/presentation/restaurant-chat.tsx","utf8");
const compiled=ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,esModuleInterop:true}}).outputText;
function render(overrides={}){
 const state={context:{organizationName:"Restaurante de prueba",locationName:"Local principal",name:"Mesa 1"},messages:[],cart:{items:[],total:"0.00"},mode:"table",pending:false,failed:false,active:true,...overrides};
 const queries=[{isSuccess:true},{data:state},{data:{items:[],currencySymbol:"S/"}}];
 const exports={};
 vm.runInNewContext(compiled,{exports,require:name=>{
  if(name.endsWith(".css")||name.endsWith("order-tracking")||name.includes("chat-api"))return {};
  if(name==="@tanstack/react-query")return {useQuery:()=>queries.shift(),useQueryClient:()=>({}),useMutation:()=>({isPending:false,mutate:()=>{}})};
  if(name==="@/design-system"||name==="@/design-system/dialog")return new Proxy({},{get:()=>({children})=>React.createElement("div",null,children)});
  if(name==="next/image")return {default:()=>null};
  return require(name);
 }});
 return renderToStaticMarkup(React.createElement(exports.RestaurantChat,{qr:"a".repeat(32)}));
}

test("Pedir cuenta solo aparece con autorización vigente de entrega del backend",()=>{
 assert.doesNotMatch(render(),/Pedir cuenta/);
 assert.doesNotMatch(render({canRequestBill:false}),/Pedir cuenta/);
 assert.match(render({canRequestBill:true}),/Pedir cuenta/);
 for(const mode of ["counter","delivery"])assert.doesNotMatch(render({canRequestBill:true,mode}),/Pedir cuenta/);
 assert.doesNotMatch(render({canRequestBill:true,active:false}),/Pedir cuenta/);
 assert.doesNotMatch(render({canRequestBill:true,pending:true}),/Pedir cuenta/);
 assert.doesNotMatch(render({canRequestBill:true,cart:{items:[{productId:"p1",name:"Lomo",quantity:1}],total:"25.00"}}),/Pedir cuenta/);
});

test("Seguir pedido aparece solo cuando el chat tiene un pedido registrado para seguir",()=>{
 assert.doesNotMatch(render({mode:"delivery"}),/Seguir pedido/);
 assert.doesNotMatch(render({mode:"delivery",trackable:false}),/Seguir pedido/);
 assert.match(render({mode:"delivery",trackable:true}),/Seguir pedido/);
 assert.match(render({mode:"delivery"}),/Compartir ubicación/);
 assert.doesNotMatch(render({mode:"delivery",trackable:true}),/Compartir ubicación/,"la entrega del pedido registrado ya está fija");
});
