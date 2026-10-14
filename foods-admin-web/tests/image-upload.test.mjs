import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import vm from "node:vm";
import ts from "typescript";

function compile(path,globals,require){const exports={};vm.runInNewContext(ts.transpileModule(readFileSync(new URL(`../src/${path}`,import.meta.url),"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,{exports,require,...globals});return exports}
function environment({width=800,height=400,host="storage.googleapis.com",storageStatus=204,saveError=false}={}){
 const calls=[],blobs=new Map(),revoked=[];
 let next=0;
 class ClientError extends Error{constructor(code){super(code);this.code=code}}
 const mockURL=class extends URL{static createObjectURL(file){const key=`blob:${++next}`;blobs.set(key,file);return key}static revokeObjectURL(key){revoked.push(key);blobs.delete(key)}};
 const globals={URL:mockURL,File,FormData,AbortSignal,setTimeout,clearTimeout,Image:class{naturalWidth=width;naturalHeight=height;set src(value){if(value)queueMicrotask(()=>this.onload?.())}},fetch:async(target,init)=>{calls.push({step:"storage",target:String(target),init});return {ok:storageStatus<300,status:storageStatus}}};
 const apiFetch=async(path,init)=>{calls.push({step:path,init});if(path==="product-images/uploads")return {imageId:"image-id",objectKey:"restaurants/org/products/image-id/original.jpg",uploadUrl:`https://${host}/bucket/`,method:"POST",fields:{key:"restaurants/org/products/image-id/original.jpg",policy:"signed-policy"},expiresAt:"2099-01-01"};if(path.endsWith("/complete"))return {imageId:"image-id",imageStatus:"PROCESSING",imageUrl:"https://storage.googleapis.com/original.jpg?signature=temporary"};if(saveError)throw new Error("Invalid product");return {id:"new-product",imageId:"image-id",imageStatus:"PROCESSING"}};
 const upload=compile("shared/api/image-upload.ts",globals,name=>{if(name==="./client")return {apiFetch,ApiClientError:ClientError};throw new Error(name)});
 const products=compile("modules/menu/products/infrastructure/products-api.ts",globals,name=>{if(name==="@/shared/api/client")return {apiFetch};if(name==="@/shared/api/image-upload")return upload;if(name==="@/shared/api/product-image")return {uploadLegacyProductImage:()=>{throw new Error("unexpected binary API upload")}};throw new Error(name)});
 return {upload,products,calls,revoked};
}
const draft={name:"Causa",price:"12",quantityControl:"none",categoryId:"",prepMinutes:"",costPrice:"",imageUrl:null};
const photo=()=>new File(["photo"],"photo.jpg",{type:"image/jpeg"});
test("validates format, 5 MB and 4000px before requesting authorization",async()=>{
 const {upload,calls}=environment();
 await assert.rejects(upload.prepareProductImage(new File(["svg"],"x.svg",{type:"image/svg+xml"})),/JPEG/);
 await assert.rejects(upload.prepareProductImage(new File([new Uint8Array(5*1024*1024+1)],"x.png",{type:"image/png"})),/5 MB/);
 const huge=environment({width:4001});await assert.rejects(huge.upload.prepareProductImage(photo()),/4000/);
 assert.equal(calls.length,0);assert.equal(huge.calls.length,0);assert.ok(huge.revoked.length);
});
test("uploads only to Storage, confirms then saves without waiting for READY",async()=>{
 const {products,calls,upload}=environment();const result=await products.saveProduct(draft,photo(),"org:location");
 assert.equal(result.imageStatus,"PROCESSING");
 assert.deepEqual(calls.map(call=>call.step),["product-images/uploads","storage","product-images/image-id/complete","products"]);
 const storage=calls[1];assert.equal(storage.init.credentials,"omit");assert.equal(storage.init.redirect,"error");assert.equal(storage.init.body.get("policy"),"signed-policy");assert.equal(Array.from(storage.init.body.keys()).at(-1),"file");
 const payload=JSON.parse(calls.at(-1).init.body);assert.equal(payload.imageId,"image-id");assert.equal(payload.imageUrl,null);assert.ok(upload.localImagePreview("org:location","image-id"));assert.equal(upload.localImagePreview("another-org","image-id"),null);
});
test("failed product save reuses confirmed upload instead of uploading again",async()=>{
 const {products,calls}=environment({saveError:true});const file=photo();
 for(let i=0;i<2;i++)await assert.rejects(products.saveProduct(draft,file,"org:location"),/Invalid product/);
 assert.equal(calls.filter(call=>call.step==="storage").length,1);assert.equal(calls.filter(call=>call.step==="products").length,2);
});
test("Storage rejection or foreign upload host cannot save a product",async()=>{
 for(const options of [{storageStatus:413},{host:"attacker.example"}]){const {products,calls}=environment(options);await assert.rejects(products.saveProduct(draft,photo()));assert.ok(!calls.some(call=>call.step==="products"));if(options.host)assert.ok(!calls.some(call=>call.step==="storage"))}
});
test("editing native photos never persists a signed read URL or readonly image metadata",async()=>{
 const {products,calls}=environment();await products.saveProduct({...draft,id:"existing",imageId:"old-image",imageKey:"object/key",imageStatus:"READY",imageUrl:"https://storage.googleapis.com/read?X-Goog-Signature=x"},null);
 const payload=JSON.parse(calls.at(-1).init.body);assert.equal(payload.imageUrl,null);assert.equal(payload.imageId,undefined);assert.equal(payload.imageKey,undefined);assert.equal(payload.imageStatus,undefined);
});
