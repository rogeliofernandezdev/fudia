// QA local del panel real. Datos ilustrativos en memoria; nunca llama al backend.
import {readFileSync,readdirSync,existsSync} from "node:fs";
import {dirname,resolve} from "node:path";
import {fileURLToPath} from "node:url";
import {createServer} from "node:http";
import {build} from "esbuild";

const root=resolve(dirname(fileURLToPath(import.meta.url)),"../..");
const bundle=await build({absWorkingDir:root,bundle:true,write:false,outfile:"preview.js",format:"iife",jsx:"automatic",define:{"process.env.NODE_ENV":'"production"',"process.env":"{}"},stdin:{resolveDir:root,loader:"tsx",contents:`
 import React from 'react';
 import {createRoot} from 'react-dom/client';
 import {QueryClient,QueryClientProvider} from '@tanstack/react-query';
 import {FeedbackProvider} from './src/providers/feedback-provider';
 import {DeliveryPartnersAdmin} from './src/modules/platform/presentation/delivery-partners-admin';
 
 createRoot(document.getElementById('root')).render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><FeedbackProvider><main className="partners-qa"><h1>Centrales de reparto</h1><DeliveryPartnersAdmin/></main></FeedbackProvider></QueryClientProvider>);
 `},plugins:[{name:"production-facades",setup(builder){
 builder.onResolve({filter:/^@\//},args=>{
  if(args.path==="@/design-system")return {path:"design",namespace:"facade"};
  if(args.path==="@/providers")return {path:"feedback",namespace:"facade"};
  const base=resolve(root,"src",args.path.slice(2));
  const path=[base,`${base}.tsx`,`${base}.ts`,resolve(base,"index.ts")].find(file=>existsSync(file)&&/\.(ts|tsx)$/.test(file));
  if(!path)throw new Error(`Cannot resolve ${args.path}`);
  return {path};
 });
 builder.onLoad({filter:/.*/,namespace:"facade"},args=>({resolveDir:root,loader:"ts",contents:args.path==="feedback"?"export {useFeedback} from './src/providers/feedback-provider'":[
  "modal","page-header","form-field","form-controls","states","skeleton","icons","icon-tile","tabs","search-field","confirm-dialog","route-map"
 ].map(name=>`export * from './src/design-system/${name}'`).join(";")}));
 }}]});
const script=bundle.outputFiles.find(file=>file.path.endsWith(".js")).text;
// Conserva fuentes/utilidades compiladas y aplica los estilos actuales de los
// componentes y botones, incluso si la última build precede a esta revisión.
const currentGlobals=readFileSync(resolve(root,"src/styles/globals.css"),"utf8").replace(/^@import.*$/gm,"");
const baseStyles=readdirSync(resolve(root,".next/static/css"),{recursive:true})
 .filter(name=>name.endsWith(".css")).map(name=>readFileSync(resolve(root,".next/static/css",name),"utf8"))
 .filter(css=>css.includes("--cloud-25:")||css.includes("@font-face")).join("\n");
// No arrastrar CSS antiguo de la ruta: ocultaría eliminaciones o cambios de scroll.
const sharedStyles=["icon-tile","modal","pagination","table-actions","form-field","form-controls","skeleton","tabs","confirm-dialog"]
 .map(name=>readFileSync(resolve(root,`src/design-system/styles/${name}.css`),"utf8")).join("\n");
const css=baseStyles+"\n"+currentGlobals+"\n"+sharedStyles+"\n"+(bundle.outputFiles.find(file=>file.path.endsWith(".css"))?.text??"");

let counter=2;
let scenario='demo';
const partners=[{id:'00000000-0000-0000-0000-000000000001',name:'Warigoo',whatsappPhone:'51941452582',active:true,priority:1,panelToken:'a'.repeat(32)},{id:'00000000-0000-0000-0000-000000000002',name:'Central Norte',whatsappPhone:'51987654321',active:false,priority:2,panelToken:'b'.repeat(32)}];
const server=createServer(async(req,res)=>{
 const url=new URL(req.url,'http://localhost');
 if(url.pathname.startsWith('/api/platform/delivery-partners')){
  res.setHeader('content-type','application/json');
  if(req.method==='GET'){
   if(scenario==='error'){res.statusCode=503;res.end(JSON.stringify({message:'No hay conexión. Vuelve a intentarlo.'}));return;}
   const answer=()=>res.end(JSON.stringify({items:scenario==='empty'?[]:partners}));
   if(scenario==='loading')setTimeout(answer,15000);else answer();return;
  }
  let body='';for await(const chunk of req)body+=chunk;
  const data=body?JSON.parse(body):{};
  const id=url.pathname.split('/')[4],partner=partners.find(p=>p.id===id);
  if(url.pathname.endsWith('/panel-link')){partner.panelToken='c'.repeat(32);res.end(JSON.stringify({partnerId:partner.id,panelToken:partner.panelToken}));return;}
  if(url.pathname.endsWith('/order')){data.partnerIds.forEach((id,i)=>partners.find(p=>p.id===id).priority=i+1);partners.sort((a,b)=>a.priority-b.priority);res.end(JSON.stringify({items:partners}));return;}
  if(Object.hasOwn(data,'coverageCodes')){res.statusCode=400;res.end(JSON.stringify({message:'No se deben enviar distritos.'}));return;}
  if(partner){Object.assign(partner,data);res.end(JSON.stringify(partner));return;}
  counter++;const created={...data,id:'00000000-0000-0000-0000-'+String(counter).padStart(12,'0'),active:true,priority:counter,panelToken:'d'.repeat(32)};partners.push(created);res.statusCode=201;res.end(JSON.stringify(created));return;
 }
 if(url.pathname==='/preview.js'){res.setHeader('content-type','text/javascript');res.end(script);return;}
 if(url.pathname.startsWith('/_next/static/media/')){res.end(readFileSync(resolve(root,'.next/static/media',url.pathname.split('/').at(-1))));return;}
 if(url.pathname.startsWith('/delivery-panel/')){res.end('Panel abierto: enlace ilustrativo de la central.');return;}
 scenario=url.searchParams.get('scenario')||'demo';
 res.setHeader('content-type','text/html; charset=utf-8');
 res.end('<!doctype html><html lang="es"><head><title>QA local · Centrales</title><meta name="viewport" content="width=device-width,initial-scale=1"><style>'+css+'.partners-qa{max-width:1100px;margin:32px auto;padding:0 16px}.partners-qa h1{font-size:24px;margin:0 0 24px}</style></head><body><div id="root"></div><script src="/preview.js"></script></body></html>');
});
server.listen(5199,'127.0.0.1',()=>console.log('QA local: http://localhost:5199 · ?scenario=empty|error|loading'));
