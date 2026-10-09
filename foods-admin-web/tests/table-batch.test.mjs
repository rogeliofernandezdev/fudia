import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url);
const read=file=>readFileSync(new URL(`../${file}`,import.meta.url),"utf8");
function compile(file,imports={},globals={}){
 const exports={};
 vm.runInNewContext(ts.transpileModule(read(file),{compilerOptions:{target:ts.ScriptTarget.ES2020,module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,Headers,URL,AbortSignal,TypeError,Error,console:{error(){}},...globals,require:name=>imports[name]??require(name)});
 return exports;
}
const {ApiClientError}=compile("src/shared/api/client.ts",{"@/shared/session/expire-session":{expireBrowserSession(){}}});

test("el lote envía una sola escritura y conserva la confirmación del API",async()=>{
 const calls=[];
 const expected={items:[{id:"table-id",name:"Mesa 1"}]};
 const {createTables}=compile("src/modules/operations/tables/infrastructure/tables-api.ts",{"@/shared/api/client":{ApiClientError,apiFetch:async(path,init)=>{calls.push({path,init});return expected}}});
 assert.equal(await createTables([{name:" Mesa 1 ",seats:"4",zone:" Principal "}]),expected);
 assert.equal(calls.length,1);
 assert.equal(calls[0].path,"tables/batch");
 assert.equal(calls[0].init.method,"POST");
 assert.deepEqual(JSON.parse(calls[0].init.body),{items:[{name:"Mesa 1",seats:4,zone:"Principal"}]});
});

test("un corte de red o servicio no afirma que las mesas fueron rechazadas ni repite el POST",async()=>{
 for(const failure of [new TypeError("fetch failed"),new ApiClientError("Unavailable","api_unavailable",503,"request-1"),new ApiClientError("Bad JSON","invalid_server_response",200)]){
  let calls=0;
  const {createTables}=compile("src/modules/operations/tables/infrastructure/tables-api.ts",{"@/shared/api/client":{ApiClientError,apiFetch:async()=>{calls++;throw failure}}});
  await assert.rejects(createTables([{name:"Mesa 1",seats:"2",zone:""}]),error=>{
   assert.equal(error.code,"table_batch_unconfirmed");
   assert.match(error.message,/Actualiza la lista antes de intentarlo nuevamente/);
   assert.equal(error.correlationId,failure.correlationId);
   return true;
  });
  assert.equal(calls,1);
 }
});

test("validación y permisos conservan el error específico",async()=>{
 for(const code of ["invalid_zone","table_conflict","forbidden"]){
  const failure=new ApiClientError("Error específico",code,code==="table_conflict"?409:400);
  const {createTables}=compile("src/modules/operations/tables/infrastructure/tables-api.ts",{"@/shared/api/client":{ApiClientError,apiFetch:async()=>{throw failure}}});
  await assert.rejects(createTables([{name:"Mesa 1",seats:"2",zone:""}]),error=>error===failure);
 }
});

test("el cliente compartido conserva la correlación del error",async()=>{
 const {apiFetch}=compile("src/shared/api/client.ts",{"@/shared/session/expire-session":{expireBrowserSession(){}}},{fetch:async()=>new Response(JSON.stringify({code:"tables_unavailable",message:"No confirmado",correlationId:"backend-request"}),{status:503})});
 await assert.rejects(apiFetch("tables/batch"),error=>error.correlationId==="backend-request");
});

class ProxyResponse extends Response{
 static json(body,init){return new ProxyResponse(JSON.stringify(body),init)}
}
const proxyImports={"next/server":{NextResponse:ProxyResponse},"node:crypto":{randomUUID:()=>"proxy-request-id"}};
const request=method=>({method,nextUrl:new URL("http://localhost/api/admin/tables/batch?test=1"),headers:new Headers({cookie:"foods_session=private-token"}),text:async()=>'{"items":[{"name":"Mesa 1"}]}'});

test("BFF conserva respuesta y correlación del backend, con espera acotada y sin repetir",async()=>{
 let calls=0;
 const {POST}=compile("src/app/api/admin/[...path]/route.ts",proxyImports,{process:{env:{}},fetch:async(target,init)=>{
  calls++;
  assert.equal(target.pathname,"/v1/admin/tables/batch");
  assert.equal(target.search,"?test=1");
  assert.equal(init.headers.get("cookie"),"foods_session=private-token");
  assert.equal(init.method,"POST");
  assert.equal(init.body,'{"items":[{"name":"Mesa 1"}]}');
  assert.ok(init.signal);
  return new Response('{"items":[]}',{status:201,headers:{"X-Request-ID":"backend-request"}});
 }});
 const response=await POST(request("POST"),{params:Promise.resolve({path:["tables","batch"]})});
 assert.equal(response.status,201);
 assert.equal(response.headers.get("X-Request-ID"),"backend-request");
 assert.equal(calls,1);
});

test("BFF distingue resultado incierto de escritura y fallo de lectura, sin exponer cookies",async()=>{
 for(const method of ["POST","GET"]){
  let calls=0;const logs=[];
  const route=compile("src/app/api/admin/[...path]/route.ts",proxyImports,{process:{env:{}},console:{error:(...values)=>logs.push(values)},fetch:async()=>{calls++;throw new TypeError("fetch failed")}});
  const response=await route[method](request(method),{params:Promise.resolve({path:["tables","batch"]})});
  const body=await response.json();
  assert.equal(response.status,503);
  assert.equal(body.correlationId,"proxy-request-id");
  assert.equal(response.headers.get("X-Request-ID"),body.correlationId);
  assert.equal(calls,1);
  assert.doesNotMatch(JSON.stringify(logs),/private-token/);
  assert.match(body.message,method==="POST"?/No pudimos confirmar la operación/:/El servicio no está disponible/);
 }
});

test("el formulario conserva el borrador, refresca resultado incierto y bloquea cambios durante el envío",()=>{
 const source=read("src/modules/operations/tables/presentation/tables-manager.tsx");
 assert.match(source,/retry:false/);
 assert.match(source,/unconfirmed\?"No pudimos confirmar el registro"/);
 assert.match(source,/if\(unconfirmed\)void client.invalidateQueries\(\{queryKey:\["tables"\]\}\)/);
 assert.match(source,/function saveAll\(\)\{if\(saveBatch.isPending\)return/);
 assert.match(source,/RowActionButton action="remove" disabled=\{saveBatch.isPending\}/);
 assert.match(source,/"Guardando…":"Guardar"/);
 const errorHandler=source.slice(source.indexOf("const saveBatch="),source.indexOf("const saveTable="));
 const onError=errorHandler.slice(errorHandler.indexOf("onError:"));
 assert.doesNotMatch(onError,/setNewRows|setAdding/);
});
