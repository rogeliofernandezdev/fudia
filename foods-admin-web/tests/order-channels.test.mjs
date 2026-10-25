import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import vm from "node:vm";
import ts from "typescript";

const root=new URL("../src/modules/operations/orders/presentation/",import.meta.url);
const read=path=>readFileSync(new URL(path,root),"utf8");
function compile(path){const exports={};vm.runInNewContext(ts.transpileModule(read(path),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,{exports,require:name=>name==="@/shared/i18n/regional-format"?{formatRegionalDateTime:()=>""}:{}});return exports}

test("pedidos: las pestañas son tipos de atención y WhatsApp QR se agrupa en Salón",()=>{
 const meta=compile("order-meta.ts");
 assert.equal(meta.serviceChannel("whatsapp"),"salon");
 assert.equal(meta.serviceChannel("mostrador"),"recojo");
 for(const channel of ["salon","recojo","delivery"])assert.equal(meta.serviceChannel(channel),channel);
 const manager=read("orders-manager.tsx");
 assert.ok(manager.includes("channelLabel(service)"),"la columna Canal muestra el tipo de atención");
 assert.ok(manager.includes('o.channel==="whatsapp"&&<small className="order-channel-origin">'),"el origen WhatsApp queda como dato secundario");
 assert.ok(manager.includes('service==="salon"&&<small className="order-staff">'),"los pedidos QR de mesa también muestran el mozo");
});
