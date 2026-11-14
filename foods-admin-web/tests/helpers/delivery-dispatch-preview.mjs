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
 import {PartnerPanel} from './src/modules/delivery-panel/presentation/partner-panel';
 const scenario=new URLSearchParams(location.search).get('scenario')||'demo';
 createRoot(document.getElementById('root')).render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><FeedbackProvider><PartnerPanel token={scenario}/></FeedbackProvider></QueryClientProvider>);
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
  "page-header","form-field","form-controls","states","skeleton","icons","icon-tile","tabs","search-field","confirm-dialog","route-map"
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
const order=(token,changes={})=>({token,stage:"accepted",orderCode:`PED-${token}`,orderStatus:"preparando",paymentConfirmed:false,organizationName:"La Mesa Peruana",locationName:"San Miguel",locationAddress:"Av. La Marina 2140, San Miguel",locationLatitude:null,locationLongitude:null,customerName:"Rosa Quispe",customerPhone:"",address:"Calle Los Cedros 280, Pueblo Libre",reference:"Portón gris, segundo piso",latitude:null,longitude:null,distanceKm:2.4,notes:"Tocar el timbre al llegar.",items:[{quantity:2,name:"Lomo saltado",note:"Uno sin cebolla"},{quantity:1,name:"Chicha morada 1 L"}],currencySymbol:"S/",subtotal:"76.00",deliveryFee:"7.50",total:"83.50",expiresInSeconds:175,createdAt:new Date().toISOString(),courierName:"",courierToken:"",pickupWaitSeconds:0,pickupAlert:"",pickupAlertAcknowledged:false,failAvailableInSeconds:0,...changes});
const assigned=(name)=>({courierName:name,courierToken:"a".repeat(32)});
const createPanel=scenario=>({partnerName:"Central de reparto · Vista de prueba",active:scenario!=="paused",failedDeliveries:[],orders:scenario==="empty"?[]:[
 order("2481",{stage:"offer",customerName:"Mariana Salazar",locationName:"Magdalena",address:"Jr. Tacna 620, Magdalena del Mar"}),
 order("2480"),
 order("2479",{...assigned("Luis Mendoza"),customerName:"Carlos Torres",orderStatus:"listo",paymentConfirmed:true}),
 order("2478",{...assigned("Ana Ríos"),customerName:"Valeria Chávez",stage:"picked_up",orderStatus:"en_camino",pickedUpAt:new Date().toISOString()}),
 order("2477",{...assigned("Luis Mendoza"),customerName:"Diego Huamán",stage:"delivered",orderStatus:"entregado",deliveredAt:new Date().toISOString()}),
 ...(scenario==="alerts"?[order("2482",{...assigned("Ana Ríos"),pickupAlert:"critical",pickupWaitSeconds:485,orderStatus:"listo",paymentConfirmed:true})]:[]),
 ...(scenario==="long"?[order("2483",{customerName:"María Alejandra Fernández de los Ángeles y del Rosario",locationName:"Restaurante de comida criolla y especialidades regionales · Sede principal de San Miguel",address:"Pasaje".repeat(25),courierName:"José Antonio Fernández de los Ángeles y del Rosario",courierToken:"b".repeat(32)})]:[]),
 ...(scenario==="volume"?Array.from({length:63},(_,index)=>order(String(3000+index),{customerName:`Cliente de prueba ${index+1}`,stage:index<23?"accepted":index<43?"picked_up":"delivered",courierName:index<23?"":"Ana Ríos",courierToken:index<23?"":"a".repeat(32),deliveredAt:index>=43?new Date(Date.now()-index*60000).toISOString():undefined})):[])
]});
const panels=new Map();
const server=createServer(async(req,res)=>{
 const url=new URL(req.url,"http://localhost");
 if(url.pathname.startsWith("/api/public/delivery-panel/")){
  const scenario=url.pathname.split("/")[4];
  res.setHeader("content-type","application/json");
  if(scenario==="error"){res.statusCode=503;res.end(JSON.stringify({message:"El servicio de reparto no está disponible. Vuelve a intentarlo."}));return;}
  if(scenario==="loading"){setTimeout(()=>res.end(JSON.stringify(createPanel("empty"))),15000);return;}
  if(!panels.has(scenario))panels.set(scenario,createPanel(scenario));
  const panel=panels.get(scenario);
  if(req.method==="POST"){
   let body="";for await(const chunk of req)body+=chunk;
   const data=JSON.parse(body),target=panel.orders.find(order=>order.token===data.token);
   if(!target){res.statusCode=404;res.end(JSON.stringify({message:"El pedido ya no está disponible."}));return;}
   let outcome="kept";
   if(url.pathname.endsWith("/courier")){target.courierName=data.courierName;target.courierToken="c".repeat(32);outcome="courier_assigned";}
   else if(data.action==="accept"){target.stage="accepted";outcome="accepted";}
   else if(data.action==="keep"){target.pickupAlertAcknowledged=true;}
   else if(["reject","release","fail"].includes(data.action)){panel.orders=panel.orders.filter(order=>order!==target);outcome=data.action==="reject"?"rejected":data.action==="release"?"released":"delivery_failed";}
   res.end(JSON.stringify({outcome}));return;
  }
  res.end(JSON.stringify(panel));return;
 }
 if(url.pathname==="/preview.js"){res.setHeader("content-type","text/javascript");res.end(script);return;}
 if(url.pathname.startsWith("/_next/static/media/")){res.end(readFileSync(resolve(root,".next/static/media",url.pathname.split("/").at(-1))));return;}
 res.setHeader("content-type","text/html; charset=utf-8");
 res.end(`<!doctype html><html lang="es"><head><title>QA local · Central de reparto</title><meta name="viewport" content="width=device-width,initial-scale=1"><style>${css}</style></head><body><div id="root"></div><script src="/preview.js"></script></body></html>`);
});
server.listen(5198,"127.0.0.1",()=>console.log("QA local: http://localhost:5198 · ?scenario=empty|error|loading|alerts|paused|long|volume"));
