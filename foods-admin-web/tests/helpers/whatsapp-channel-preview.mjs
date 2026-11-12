// Production editor with a local transport fixture; no account or provider writes.
import {readFileSync,readdirSync,existsSync} from "node:fs";
import {createServer} from "node:http";
import {resolve} from "node:path";
import {build} from "esbuild";

const root=process.cwd();
const bundle=await build({bundle:true,jsx:"automatic",write:false,outdir:"out",format:"iife",define:{"process.env":"{}","process.env.NODE_ENV":"\"development\""},stdin:{resolveDir:root,loader:"tsx",contents:`
 import React from 'react';
 import {createRoot} from 'react-dom/client';
 import {QueryClient,QueryClientProvider} from '@tanstack/react-query';
 import {WhatsAppChannelEditor} from './src/modules/platform/presentation/whatsapp-channel-editor';
 const existing={id:'00000000-0000-4000-8000-000000000001',countryCode:'PE',countryName:'Perú',phoneNumber:'+51912345678',displayName:'Fudia Perú',active:false,provider:'META_CLOUD_API',revision:1,connectionState:'unconfigured',needsReconciliation:false};
 createRoot(document.getElementById('root')).render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><WhatsAppChannelEditor initial={existing} existing={existing} countries={[{code:'PE',name:'Perú',defaultCurrency:'PEN',callingCode:'+51'}]} usedCountries={['PE']} providers={[{value:'META_CLOUD_API',label:'Meta Cloud API'},{value:'WHATSMEOW',label:'whatsmeow'}]} pending={false} onClose={()=>{}} onSave={value=>{document.getElementById('result').textContent=JSON.stringify({provider:value.provider,phoneNumber:value.phoneNumber,active:value.active})}}/></QueryClientProvider>);
 `},plugins:[{name:"production-facades",setup(builder){
 builder.onResolve({filter:/^@\//},args=>{
  if(args.path==="@/design-system")return {path:"design",namespace:"facade"};
  if(args.path==="@/providers")return {path:"providers",namespace:"facade"};
  const base=resolve(root,"src",args.path.slice(2));const path=[base,`${base}.tsx`,`${base}.ts`,resolve(base,"index.ts")].find(file=>existsSync(file)&&/\.(ts|tsx)$/.test(file));
  if(!path)throw new Error(`Missing dependency: ${args.path}`);return {path};
 });
 builder.onLoad({filter:/.*/,namespace:"facade"},args=>({resolveDir:root,loader:"ts",contents:args.path==="providers"?"export function useSession(){return {location:{country:'PE',timezone:'America/Lima'}}}":
 "export * from './src/design-system/page-header';export * from './src/design-system/form-field';export * from './src/design-system/modal';export * from './src/design-system/confirm-dialog';export * from './src/design-system/states';export * from './src/design-system/skeleton'"}));
 }}]});
const script=bundle.outputFiles.find(file=>file.path.endsWith(".js")).text;
const css=readdirSync(resolve(root,".next/static/css")).filter(name=>name.endsWith(".css")).map(name=>readFileSync(resolve(root,".next/static/css",name),"utf8")).join("\n")+"\n"+(bundle.outputFiles.find(file=>file.path.endsWith(".css"))?.text??"")+"\n"+readFileSync(resolve(root,"src/modules/platform/presentation/platform-global-settings.css"),"utf8");
const server=createServer(async(req,res)=>{
 const url=new URL(req.url,"http://localhost");
 res.setHeader("Cache-Control","no-store");
 if(url.pathname.startsWith("/api/platform/")){
  res.setHeader("content-type","application/json");
  if(url.pathname.endsWith("/history")){res.end(JSON.stringify({items:[]}));return}
  if(url.pathname.endsWith("/link")){
   let raw="";for await(const chunk of req)raw+=chunk;
   const data=JSON.parse(raw||"{}");
   res.end(JSON.stringify({provider:"WHATSMEOW",countryCode:"PE",sessionRef:"00000000-0000-4000-8000-000000000002",state:url.searchParams.get("connected")==="1"?"connected":"pairing",phoneNumber:data.sessionRef?"+51987654321":"",qrCode:"local-visual-qa-device-code",expiresAt:new Date(Date.now()+60000).toISOString()}));return
  }
 }
 if(url.pathname==="/preview.js"){res.setHeader("content-type","text/javascript");res.end(script);return}
 if(url.pathname.startsWith("/_next/static/media/")){res.end(readFileSync(resolve(root,".next/static/media",url.pathname.split("/").at(-1))));return}
 res.setHeader("content-type","text/html; charset=utf-8");res.end(`<!doctype html><html lang="es"><head><meta name="viewport" content="width=device-width,initial-scale=1"><style>${css}</style><title>WhatsApp · vista de prueba local</title></head><body><div id="root"></div><output id="result"></output><script src="/preview.js"></script></body></html>`);
});
server.listen(5193,"127.0.0.1",()=>console.log("WhatsApp local preview: http://127.0.0.1:5193"));
