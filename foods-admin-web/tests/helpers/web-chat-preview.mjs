// Real chat with isolated illustrative API responses. No external requests or orders.
import {build} from "esbuild";
import {readFile,readdir,mkdir} from "node:fs/promises";
import {createServer} from "node:http";
import {resolve} from "node:path";

const root=process.cwd(),out=resolve(root,".cache/web-chat-preview");
await mkdir(out,{recursive:true});
await build({
 stdin:{loader:"tsx",resolveDir:root,contents:`
import React from 'react';import {createRoot} from 'react-dom/client';
import {QueryClient,QueryClientProvider} from '@tanstack/react-query';
import {RestaurantChat} from './src/modules/concierge-web/presentation/restaurant-chat';
const mode=new URLSearchParams(location.search).get('case')||'delivery';
const pending=mode==='unquoted',pickup=mode==='pickup',table=mode==='table';
const dish={productId:'p1',name:'Lomo Saltado Criollo',description:'Con arroz y papas.',price:'20.00',imageUrl:null,categoryName:'Fondos',status:'available',isCombo:false,hasModifiers:false,remaining:null};
const state={context:{organizationName:'Restaurante de prueba',locationName:'Local principal',name:'Mesa 1',serviceType:table?'table':'delivery'},messages:[{id:'hello',role:'assistant',text:'Tu pedido está listo para revisar.',createdAt:1794540000}],cart:{items:[{productId:'p1',name:dish.name,quantity:1,unitPrice:'20.00',lineTotal:'20.00',note:'',modifiers:[],selections:[]}],total:'20.00'},orderSummary:{subtotal:'20.00',deliveryFee:pending?null:pickup||table?'0.00':'8.00',total:pending?null:pickup||table?'20.00':'28.00',currencySymbol:'S/',fulfillment:table?'table':pickup?'pickup':'delivery'},confirmationRef:mode==='draft'||pending?null:'ready-1',mode:table?'table':'delivery',customerName:'Ana Quispe',pending:false,failed:false,active:true};
let prepared=0,confirmed=0;const seen=new Set();
if(mode==='addition'){
 state.registeredItems=structuredClone(state.cart.items);
 state.cart.items[0]={...state.cart.items[0],productId:'p2',name:'Tallarín Saltado de Carne',unitPrice:'14.00',lineTotal:'14.00'};
 state.cart.total='14.00';state.orderSummary={...state.orderSummary,addingToOrder:true,subtotal:'34.00',total:'42.00'};
}
window.fetch=async(url,options)=>{
 const operation=String(url).split('/').pop();
 if(operation==='session')return Response.json({ready:true});
 if(operation==='state')return Response.json(state);
 if(operation==='menu')return Response.json({restaurantLogoUrl:null,currencySymbol:'S/',items:[dish]});
 if(operation==='messages'){
  const input=JSON.parse(options.body);if(seen.has(input.requestId))return Response.json({accepted:true});seen.add(input.requestId);
  const action=input.action;
  if(action?.name==='prepare_confirmation'){prepared++;console.info('QA: prepare '+prepared);if(!pending)state.confirmationRef='ready-'+prepared;}
  else if(action?.name==='confirm_order'){
   if(!state.confirmationRef||action.confirmation_ref!==state.confirmationRef)throw new Error('Stale confirmation');
   confirmed++;console.info('QA: confirm '+confirmed);state.confirmationRef=null;state.cart={items:[],total:'0.00'};state.messages.push({id:'done',role:'assistant',text:'Tu pedido fue registrado.',createdAt:1794540060});
  }else if(action?.name==='remove_item'){state.cart={items:[],total:'0.00'};state.confirmationRef=null;}
  else if(action?.name==='add_item'){const quantity=(state.cart.items[0]?.quantity||0)+1;const total=(quantity*20).toFixed(2);state.cart={items:[{productId:'p1',name:dish.name,quantity,unitPrice:'20.00',lineTotal:total,note:'',modifiers:[],selections:[]}],total};state.orderSummary={...state.orderSummary,subtotal:total,total:(quantity*20+Number(state.orderSummary.deliveryFee||0)).toFixed(2)};state.confirmationRef=null;}
  return Response.json({accepted:true});
 }
 throw new Error('Unexpected fixture request: '+url);
};
createRoot(document.getElementById('root')).render(<QueryClientProvider client={new QueryClient()}><RestaurantChat qr={'a'.repeat(32)}/></QueryClientProvider>);
`},
 bundle:true,outfile:resolve(out,"bundle.js"),jsx:"automatic",define:{"process.env.NODE_ENV":'"development"'},
 plugins:[{name:"preview-adapters",setup(b){
  b.onResolve({filter:/^next\/image$/},()=>({path:"image",namespace:"preview"}));
  b.onResolve({filter:/^next\/link$/},()=>({path:"link",namespace:"preview"}));
  b.onResolve({filter:/^@\/design-system$/},()=>({path:"design-system",namespace:"preview"}));
  b.onLoad({filter:/.*/,namespace:"preview"},args=>({loader:"tsx",resolveDir:root,contents:args.path==="image"?
   "import React from 'react';export default function Image({unoptimized,...props}){return <img {...props}/>}":
   args.path==="link"?"import React from 'react';export default function Link({href,children,...props}){return <a href={href} {...props}>{children}</a>}":
   `export {Button,Input} from './src/design-system/page-header';export {ModalBody,ModalFooter,ModalHeader} from './src/design-system/modal';export {Skeleton} from './src/design-system/skeleton';`}));
 }}],logLevel:"warning",
});
const cssFiles=["src/styles/variables.css","src/styles/globals.css","src/modules/concierge-web/presentation/restaurant-chat.css"];
const css=(await Promise.all(cssFiles.map(path=>readFile(resolve(root,path),"utf8")))).join("\n").replace(/^@import[^;]+;/gm,"");
const builtCss=(await Promise.all((await readdir(resolve(root,".next/static/css"))).filter(name=>name.endsWith(".css")).map(name=>readFile(resolve(root,".next/static/css",name),"utf8")))).join("\n");
const font=builtCss.match(/@font-face\{[^}]*font-family:Plus Jakarta Sans;[^}]*unicode-range:u\+00\?\?[^}]*\}/)?.[0].match(/media\/([^/)]+\.woff2)/)?.[1];
if(!font)throw new Error("Build the frontend first to use its real font.");
const fontBytes=await readFile(resolve(root,".next/static/media",font));
createServer(async(req,res)=>{
 if(req.url==="/bundle.js"){res.setHeader("content-type","text/javascript");res.end(await readFile(resolve(out,"bundle.js")));return;}
 if(req.url==="/font.woff2"){res.setHeader("content-type","font/woff2");res.end(fontBytes);return;}
 res.setHeader("content-type","text/html; charset=utf-8");res.end(`<!doctype html><html lang="es"><head><meta name="viewport" content="width=device-width,initial-scale=1"><title>Resumen del chat · Prueba local</title><style>${css}@font-face{font-family:Plus Jakarta Sans;font-weight:200 800;src:url(/font.woff2)}:root{--font-jakarta:"Plus Jakarta Sans"}body{margin:0}</style></head><body><div id="root"></div><script src="/bundle.js"></script></body></html>`);
}).listen(5199,"127.0.0.1",()=>console.log("Chat QA: http://127.0.0.1:5199 (?case=draft|unquoted|pickup|table)"));
