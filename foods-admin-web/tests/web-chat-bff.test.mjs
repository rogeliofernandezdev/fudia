import test from "node:test";
import assert from "node:assert/strict";
import {buildSync} from "esbuild";
import {createRequire} from "node:module";
import {NextRequest} from "next/server.js";

const require=createRequire(import.meta.url);
const compiled=buildSync({entryPoints:["src/modules/concierge-web/infrastructure/chat-bff.ts"],bundle:true,platform:"node",format:"cjs",write:false,external:["next/server"]}).outputFiles[0].text;
const mod={exports:{}};
new Function("require","module","exports",compiled)(require,mod,mod.exports);
const {proxyChat}=mod.exports;
const qr="a".repeat(32);
const base=`https://pedidos.example.com/api/public/chat/${qr}`;

test("el BFF rechaza origen externo y sesiones ausentes sin consultar Concierge",async()=>{
 const before=process.env.FUDIA_CONCIERGE_API_KEY;
 process.env.FUDIA_CONCIERGE_API_KEY="private-integration-key-at-least-32-chars";
 try {
  const external=new NextRequest(`${base}/session`,{method:"POST",headers:{origin:"https://external.example.com"},body:"{}"});
  assert.equal((await proxyChat(external,qr,"session")).status,403);
  const missing=new NextRequest(`${base}/messages`,{method:"POST",headers:{origin:"https://pedidos.example.com"},body:"{}"});
  assert.equal((await proxyChat(missing,qr,"messages")).status,410);
 } finally {if(before===undefined)delete process.env.FUDIA_CONCIERGE_API_KEY;else process.env.FUDIA_CONCIERGE_API_KEY=before}
});

test("detrás del proxy el BFF compara el origen con el host público, no con el interno",async()=>{
 const before=process.env.FUDIA_CONCIERGE_API_KEY,originalFetch=globalThis.fetch;
 process.env.FUDIA_CONCIERGE_API_KEY="private-integration-key-at-least-32-chars";
 globalThis.fetch=async()=>Response.json({capability:"c".repeat(43),expiresIn:3600});
 const internal=`http://0.0.0.0:8080/api/public/chat/${qr}`;
 const proxied=(operation,origin,extra={})=>new NextRequest(`${internal}/${operation}`,{method:"POST",headers:{host:"pedidos.example.com","x-forwarded-proto":"https",origin,...extra},body:"{}"});
 try {
  const opened=await proxyChat(proxied("session","https://pedidos.example.com"),qr,"session");
  assert.equal(opened.status,200);
  for(const cookie of opened.cookies.getAll()) assert.equal(cookie.secure,true);
  assert.equal((await proxyChat(proxied("messages","https://pedidos.example.com",{"x-forwarded-host":"pedidos.example.com"}),qr,"messages")).status,410);
  assert.equal((await proxyChat(proxied("session","https://external.example.com"),qr,"session")).status,403);
  assert.equal((await proxyChat(proxied("session","http://0.0.0.0:8080"),qr,"session")).status,403);
  assert.equal((await proxyChat(proxied("session","https://pedidos.example.com",{"sec-fetch-site":"cross-site"}),qr,"session")).status,403);
  assert.equal((await proxyChat(proxied("session","null"),qr,"session")).status,403);
 } finally {globalThis.fetch=originalFetch;if(before===undefined)delete process.env.FUDIA_CONCIERGE_API_KEY;else process.env.FUDIA_CONCIERGE_API_KEY=before}
});

test("el BFF fija el QR y la identidad; guarda credenciales solo en cookies HttpOnly",async()=>{
 const before=process.env.FUDIA_CONCIERGE_API_KEY,originalFetch=globalThis.fetch;
 const secret="private-integration-key-at-least-32-chars";
 process.env.FUDIA_CONCIERGE_API_KEY=secret;
 let forwarded;
 globalThis.fetch=async(_url,options)=>{
  forwarded=JSON.parse(options.body);
  assert.equal(options.headers["X-Fudia-Concierge-Key"],secret);
  return Response.json({capability:"c".repeat(43),expiresIn:3600});
 };
 try {
  const request=new NextRequest(`${base}/session`,{method:"POST",headers:{origin:"https://pedidos.example.com"},body:JSON.stringify({qr:"forged",visitor:"forged",capability:"forged"})});
  const response=await proxyChat(request,qr,"session");
  assert.equal(response.status,200);
  assert.deepEqual(await response.json(),{ready:true});
  assert.equal(forwarded.qr,qr);
  assert.match(forwarded.visitor,/^[A-Za-z0-9_-]{43}$/);
  assert.equal(forwarded.capability,"");
  for(const cookie of response.cookies.getAll()){
   assert.equal(cookie.httpOnly,true);assert.equal(cookie.secure,true);assert.equal(cookie.sameSite,"lax");
  }
  assert.equal(response.cookies.getAll().length,2);
 } finally {globalThis.fetch=originalFetch;if(before===undefined)delete process.env.FUDIA_CONCIERGE_API_KEY;else process.env.FUDIA_CONCIERGE_API_KEY=before}
});
