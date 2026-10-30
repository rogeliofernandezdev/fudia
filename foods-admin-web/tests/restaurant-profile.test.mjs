import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import vm from "node:vm";
import ts from "typescript";

function setup({failSave=false,host="storage.googleapis.com"}={}){
 const calls=[];let id=0,failed=false;
 const authenticatedFetch=async(path,init)=>{
  calls.push({path,init});
  if(path.endsWith("/uploads"))return{imageId:String(++id),uploadUrl:`https://${host}/bucket`,method:"POST",fields:{policy:"signed"}};
  if(path.endsWith("/complete"))return{id:path.split("/").at(-2),status:"PROCESSING",url:"https://storage.googleapis.com/original.jpg"};
  if(init?.method==="PATCH"&&failSave&&!failed){failed=true;throw new Error("retry save");}
  return{foundedOn:"2020-04-12",logo:null,menuPages:[],maxMenuPages:20};
 };
 const exports={};
 vm.runInNewContext(ts.transpileModule(readFileSync(new URL("../src/modules/organizations/infrastructure/restaurant-profile-api.ts",import.meta.url),"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS}}).outputText,{exports,URL,FormData,AbortSignal,fetch:async(url,init)=>{calls.push({path:"storage",url,init});return{ok:true};},require:name=>{
  if(name==="@/shared/api/client")return{authenticatedFetch};
  if(name==="@/shared/api/image-upload")return{readImageMetadata:async file=>{if(file.size>5242880)throw new Error("5 MB");return{contentType:file.type,sizeBytes:file.size,width:800,height:400};}};
  throw new Error(name);
 }});
 return{api:exports,calls};
}
const photo=()=>new File(["image"],"carta.jpg",{type:"image/jpeg"});

test("restaurant identity uploads directly, preserves page order and persists IDs while PROCESSING",async()=>{
 const{api,calls}=setup();
 await api.saveRestaurantProfile({foundedOn:"2020-04-12",logo:{file:photo()},pages:[{image:{id:"existing"}},{file:photo()}]},"tenant");
 assert.equal(calls.filter(call=>call.path==="storage").length,2);
 for(const call of calls.filter(call=>call.path==="storage")){assert.equal(call.init.credentials,"omit");assert.equal([...call.init.body.keys()].at(-1),"file");}
 const payload=JSON.parse(calls.find(call=>call.init?.method==="PATCH").init.body);
 assert.deepEqual(payload,{foundedOn:"2020-04-12",logoImageId:"1",menuImageIds:["existing","2"]});
 assert.ok(!JSON.stringify(payload).includes("https://"));
});
test("retry after profile failure reuses uploads within tenant; platform target remains explicit",async()=>{
 const{api,calls}=setup({failSave:true}),draft={foundedOn:"",logo:null,pages:[{file:photo()}]};
 await assert.rejects(api.saveRestaurantProfile(draft,"new-tenant","new-tenant"),/retry/);
 await api.saveRestaurantProfile(draft,"new-tenant","new-tenant");
 assert.equal(calls.filter(call=>call.path==="storage").length,1);
 assert.ok(calls.filter(call=>call.path!=="storage").every(call=>call.path.startsWith("/api/platform/organizations/new-tenant/profile")));
});
test("untrusted Storage host cannot receive private files or publish a profile",async()=>{
 const{api,calls}=setup({host:"attacker.example"});
 await assert.rejects(api.saveRestaurantProfile({foundedOn:"",logo:{file:photo()},pages:[]},"tenant"),/autorización/);
 assert.ok(!calls.some(call=>call.path==="storage"||call.init?.method==="PATCH"));
});
