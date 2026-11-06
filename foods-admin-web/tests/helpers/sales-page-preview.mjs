// Interactive QA of the production SalesPage with isolated data. No API or writes.
// Run from foods-admin-web after npm run build: node tests/helpers/sales-page-preview.mjs
import {build} from "esbuild";
import {readFile,readdir,mkdir} from "node:fs/promises";
import {createServer} from "node:http";
import {resolve} from "node:path";

const root=process.cwd(),output=resolve(root,".cache/sales-preview");
await mkdir(output,{recursive:true});
const fixture=`
const mode=new URLSearchParams(location.search).get('mode');
const today=new Intl.DateTimeFormat('en-CA',{timeZone:'America/Lima',year:'numeric',month:'2-digit',day:'2-digit'}).format(new Date());
const names=['Mesa 03','Mesa 08','Ana Torres','Mesa 12','Carlos Rojas','Venta directa','Mesa 05','María García','Mesa 02','Luis Pérez','Mesa 06','Mesa 10'];
const items=names.map((name,i)=>({id:'preview-'+i,code:'PED-'+String(1048-i),channel:['salon','salon','delivery','salon','recojo','recojo'][i%6],status:'entregado',customerName:name.startsWith('Mesa')?'Familia Mendoza':name==='Venta directa'?'':name,tableName:name.startsWith('Mesa')?name:'',total:String([148,86.5,64,120,42,25.5][i%6]),paidAmount:String([148,86.5,64,120,42,25.5][i%6]),remainingAmount:'0',paymentStatus:'paid',paymentMethods:[['Efectivo'],['Yape'],['Tarjeta'],['Efectivo','Yape'],['Convenio empresa'],['Plin']][i%6],createdAt:today+'T'+String(18-i%6).padStart(2,'0')+':24:00Z'}));
export async function listSales({q='',page=1,pageSize=10,from='',to=''}){
 if(mode==='loading')return new Promise(()=>{});
 if(mode==='error')throw new Error('No pudimos conectar con el servidor. Inténtalo de nuevo.');
 const rows=mode==='empty'?[]:items.filter(row=>[row.code,row.tableName,row.customerName].join(' ').toLowerCase().includes(q.toLowerCase())&&(!from||row.createdAt.slice(0,10)>=from)&&(!to||row.createdAt.slice(0,10)<=to));
 const result=mode==='long'?rows.map(row=>({...row,customerName:'María de los Ángeles Fernández del Castillo y familia — reserva de celebración',paymentMethods:['Transferencia bancaria por convenio de atención corporativa','Tarjeta'],total:'123456789.50'})):rows;
 return {items:result.slice((page-1)*pageSize,page*pageSize),total:result.length,page,pageSize,amountTotal:String(result.reduce((sum,row)=>sum+Number(row.total),0))};
}
export async function getSaleDetail(id){const sale=items.find(row=>row.id===id);return {order:{...sale,waiterName:'Daniel Rojas',subtotal:sale.total,deliveryFee:'0',notes:'',items:[{id:'line-1',name:'Lomo saltado',qty:'2',unitPrice:String(Number(sale.total)/2),note:'',selections:[],modifiers:[]}]},paidAmount:sale.total,remainingAmount:'0',paymentStatus:'paid',payments:[{id:'payment-1',method:'cash',methodName:sale.paymentMethods[0],amount:sale.total,refundedAmount:'0',netAmount:sale.total,reference:'',cashRegisterName:'Caja principal',createdByName:'Ana Torres',createdAt:sale.createdAt}]};}
`;
const providers=`export const useSession=()=>({location:{id:'preview',country:'PE',timezone:'America/Lima'},can:()=>true});export const useMoney=()=>value=>'S/ '+Number(value).toLocaleString('es-PE',{minimumFractionDigits:2,maximumFractionDigits:2});export const useSettings=()=>({currencyDecimals:2,currencySymbol:'S/',currencyPosition:'before'});export const useFeedback=()=>({notify(){}});`;
const designSystem=["page-header","data-table","states","form-field","icons","icon-tile","search-field","skeleton","detail-view","tabs","modal"].map(file=>`export * from ${JSON.stringify(resolve(root,`src/design-system/${file}`))};`).join('\n');
await build({stdin:{contents:`import React from 'react';import {createRoot} from 'react-dom/client';import {QueryClient,QueryClientProvider} from '@tanstack/react-query';import {SalesPage} from './src/modules/sales/presentation/sales-page';const client=new QueryClient({defaultOptions:{queries:{retry:false,refetchOnWindowFocus:false}}});createRoot(document.getElementById('root')).render(<QueryClientProvider client={client}><SalesPage/></QueryClientProvider>);`,resolveDir:root,loader:"jsx"},bundle:true,outfile:resolve(output,"bundle.js"),platform:"browser",jsx:"automatic",alias:{"@":resolve(root,"src")},define:{"process.env.NODE_ENV":'"development"'},plugins:[{name:"isolated-sales-data",setup(builder){
 builder.onResolve({filter:/^@\/design-system$/},()=>({path:"design-system",namespace:"sales-fixture"}));
 builder.onResolve({filter:/^@\/providers(?:\/.*)?$/},()=>({path:"providers",namespace:"sales-fixture"}));
 builder.onResolve({filter:/sales-api$/},()=>({path:"sales-api",namespace:"sales-fixture"}));
 builder.onResolve({filter:/^@\/modules\/operations\/orders$/},()=>({path:resolve(root,"src/modules/operations/orders/presentation/order-meta.ts")}));
 builder.onResolve({filter:/^@\/modules\/einvoicing$/},()=>({path:"fiscal",namespace:"sales-fixture"}));
 builder.onResolve({filter:/payment-ticket-dialog$/},()=>({path:"ticket",namespace:"sales-fixture"}));
 builder.onResolve({filter:/^next\/link$/},()=>({path:"link",namespace:"sales-fixture"}));
 builder.onResolve({filter:/^next\/dynamic$/},()=>({path:"dynamic",namespace:"sales-fixture"}));
 builder.onLoad({filter:/.*/,namespace:"sales-fixture"},({path})=>({contents:path==="design-system"?designSystem:path==="providers"?providers:path==="sales-api"?fixture:path==="link"?`import React from 'react';export default function Link({children,...props}){return React.createElement('a',props,children)}`:path==="dynamic"?"export default ()=>()=>null;":path==="fiscal"?"export const FiscalStatus=()=>null;export const FiscalPrintDialog=()=>null;export const deliveredDocument=()=>null;export const ticketNoticeFor=()=>undefined;":"export const PaymentTicketDialog=()=>null;",loader:"js",resolveDir:root}));
}}],logLevel:"warning"});
const files=(await readdir(resolve(root,".next/static/css"))).filter(file=>file.endsWith('.css'));
// Keep the shared compiled styles; replace only the module's rules with its current source.
const sharedCss=(await Promise.all(files.map(file=>readFile(resolve(root,".next/static/css",file),"utf8")))).join('\n').replaceAll('.sales-','.sales-previous-').replaceAll('sales-shimmer','sales-previous-shimmer');
const css=sharedCss+'\n'+await readFile(resolve(root,'src/modules/sales/presentation/sales.css'),'utf8');
const fontFiles=await readdir(resolve(root,".next/static/media"));
const font=fontFiles.find(file=>file.endsWith('.woff2')&&css.includes(file));
const server=createServer(async(req,res)=>{try{
 const path=new URL(req.url,"http://localhost").pathname;
 if(path==='/bundle.js'){res.setHeader('content-type','text/javascript');res.end(await readFile(resolve(output,'bundle.js')));return;}
 if(path==='/style.css'){res.setHeader('content-type','text/css');res.end(css);return;}
 if(path.endsWith('.woff2')){const name=path.split('/').at(-1);if(!fontFiles.includes(name))throw new Error('Unknown font');res.setHeader('content-type','font/woff2');res.end(await readFile(resolve(root,'.next/static/media',name)));return;}
 res.setHeader('content-type','text/html; charset=utf-8');res.end(`<!doctype html><html lang="es"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Ventas · Revisión visual</title><link rel="stylesheet" href="/style.css"><style>@font-face{font-family:Preview Jakarta;src:url(/${font});font-weight:200 800}:root{--font-jakarta:'Preview Jakarta'}body{margin:0}.preview-top{height:64px;position:sticky;top:0;z-index:20;display:flex;align-items:center;justify-content:space-between;padding:0 32px;border-bottom:1px solid var(--line);background:var(--surface);font-size:13px;color:var(--ink-500)}.preview-top b{color:var(--ink-950)}main{max-width:1440px;margin:auto;padding:32px}@media(width<=600px){main{padding:16px}.preview-top{padding:0 16px}}</style></head><body><div class="preview-top"><b>FudIA · Ventas</b><span>Datos de prueba</span></div><main id="root"></main><script src="/bundle.js"></script></body></html>`);
 }catch(error){res.statusCode=404;res.end(String(error));}});
server.listen(5196,'localhost',()=>console.log('Ventas QA: http://localhost:5196 (sin API ni escrituras)'));
