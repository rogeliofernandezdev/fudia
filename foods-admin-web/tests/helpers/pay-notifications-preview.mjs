// Interactive fixture of the production components. No authentication or real API writes.
// Run after build: node tests/helpers/pay-notifications-preview.mjs
import {build} from "esbuild";
import {readFile,readdir,mkdir} from "node:fs/promises";
import {createServer} from "node:http";
import {resolve} from "node:path";

const root=process.cwd(),out=resolve(root,".cache/pay-notifications-preview");
await mkdir(out,{recursive:true});
await build({stdin:{loader:"tsx",resolveDir:root,contents:`
import React,{useEffect,useState} from 'react';import {createRoot} from 'react-dom/client';
import {QueryClient,QueryClientProvider} from '@tanstack/react-query';
import {PayNotificationsPanel} from './src/modules/payment-devices/presentation/pay-notifications-panel';
import {PayNotificationDetail} from './src/modules/payment-devices/presentation/pay-notification-detail';
import {PayNoticeLayer} from './src/modules/payment-devices/presentation/pay-notice-layer';
import {NotificationBell} from './src/modules/notifications/presentation/notification-bell';
import {PayCollectionProvider,PayCollectionRegistration} from './src/modules/payment-devices/presentation/pay-collection-context';
const params=new URLSearchParams(location.search),scenario=params.get('case')||'ready';
const provider=params.get('provider')||'yape';
const operationNumber=scenario==='missing-operation'?undefined:'0012345678';
const notification={id:'fixture-payment',deviceId:'fixture-device',deviceName:scenario==='long'?'Teléfono principal de caja del salón y terraza':'DNP-NX9',provider,eventKey:'fixture',senderName:scenario==='long'?'Rogelio Fernández Aybar de los Ríos y del Valle — celebración familiar':'Rogelio Fernández Aybar',amount:'10.50',currency:'PEN',occurredAt:'2026-11-18T15:50:00-05:00',status:['associated','confirmed','dismissed'].includes(scenario)?scenario:'detected',createdAt:'2026-11-18T15:50:00-05:00'};
notification.operationNumber=operationNumber;
window.payFixture={scenario,notification,calls:[],items:[],notices:[]};
function Preview(){const [shown,setShown]=useState(true),[ready,setReady]=useState(false);useEffect(()=>setReady(true),[]);return <PayCollectionProvider>
{scenario==='suggestion'&&<PayCollectionRegistration order={{id:'fixture-order',code:'PED-0042',pending:'10.50'}}/>}
{params.has('notices')?<><header style={{height:64,display:'flex',justifyContent:'flex-end',padding:'8px 16px'}}><NotificationBell/></header><PayNoticeLayer/></>
:params.has('table')?<main style={{padding:16,maxWidth:1100,margin:'auto'}}><section className='panel'><PayNotificationsPanel dateTime={value=>new Intl.DateTimeFormat('es-PE',{dateStyle:'medium',timeStyle:'short',timeZone:'America/Lima'}).format(new Date(value))}/></section></main>
:ready&&shown&&<PayNotificationDetail notification={notification} mode={params.has('associate')?'associate':'detail'} onClose={()=>setShown(false)} onDone={()=>setShown(false)}/>}
</PayCollectionProvider>}
const client=new QueryClient({defaultOptions:{queries:{retry:false,gcTime:0,refetchOnWindowFocus:false}}});
let sequence=0;
window.payFixture.refresh=()=>Promise.all(['pay-notice-feed','staff-notices'].map(key=>client.invalidateQueries({queryKey:[key]})));
window.payFixture.receive=async(provider='yape')=>{const item={...notification,id:'incoming-'+(++sequence),provider};window.payFixture.items.unshift(item);window.payFixture.notices.unshift({id:'notice-'+item.id,kind:'pay_detected',title:'Pago recibido por '+(provider==='yape'?'Yape':'Plin'),detail:'S/ '+item.amount+' · '+item.senderName,createdAt:new Date().toISOString(),read:false});await window.payFixture.refresh();};
createRoot(document.getElementById('root')).render(<QueryClientProvider client={client}><Preview/></QueryClientProvider>);
`},bundle:true,outfile:resolve(out,"bundle.js"),jsx:"automatic",alias:{"@":resolve(root,"src")},define:{"process.env.NODE_ENV":'"development"'},plugins:[{name:"fixture",setup(b){
 b.onResolve({filter:/^@\/providers(?:\/session-context)?$/},()=>({path:"providers",namespace:"fixture"}));
 b.onResolve({filter:/^@\/design-system$/},()=>({path:"design-system",namespace:"fixture"}));
 b.onResolve({filter:/^@\/modules\/payment-devices$/},()=>({path:"pay-provider",namespace:"fixture"}));
 b.onResolve({filter:/pay-devices-api$/},()=>({path:"api",namespace:"fixture"}));
 b.onResolve({filter:/notices-api$/},()=>({path:"notices-api",namespace:"fixture"}));
 b.onResolve({filter:/^next\/navigation$/},()=>({path:"navigation",namespace:"fixture"}));
 b.onResolve({filter:/^next\/link$/},()=>({path:"link",namespace:"fixture"}));
 b.onLoad({filter:/.*/,namespace:"fixture"},({path})=>({loader:"tsx",resolveDir:root,contents:path==="providers"?`
export const useMoney=()=>value=>'S/ '+Number(value||0).toFixed(2);
export const useSession=()=>({user:{id:'cashier'},organization:{id:'restaurant'},location:{id:'location',country:'PE',timezone:'America/Lima'},can:()=>true});
export const useFeedback=()=>({notify:message=>window.payFixture.calls.push({kind:'feedback',message})});
`:path==="design-system"?`
export {Button,IconButton,Select,Pagination,Status} from './src/design-system/page-header';
export {ConfirmDialog} from './src/design-system/confirm-dialog';
export {EmptyState,ErrorState} from './src/design-system/states';
export {SearchField} from './src/design-system/search-field';
export {Skeleton,TableSkeleton} from './src/design-system/skeleton';
export {Modal,ModalBody,ModalFooter} from './src/design-system/modal';
export {FormDisclosure} from './src/design-system/form-controls';
export {Icon} from './src/design-system/icons';export {IconTile} from './src/design-system/icon-tile';
`:path==="pay-provider"?`export {PayProviderTag} from './src/modules/payment-devices/presentation/pay-status-tag';`
:path==="navigation"?`export const useRouter=()=>({push:route=>{window.payFixture.route=route}});`
:path==="notices-api"?`
export const getNotices=async()=>({items:[...window.payFixture.notices],unread:window.payFixture.notices.filter(item=>!item.read).length});
export const markNoticeRead=async id=>{window.payFixture.calls.push({kind:'read',args:[id]});window.payFixture.notices=window.payFixture.notices.map(item=>item.id===id?{...item,read:true}:item)};
export const markAllNoticesRead=async()=>{window.payFixture.notices=window.payFixture.notices.map(item=>({...item,read:true}))};
`:path==="link"?`import React from 'react';export default function Link({children,...props}){return <a {...props}>{children}</a>}`:`
const record=(kind,args)=>window.payFixture.calls.push({kind,args});
const wait=async kind=>{record(kind,[]);const scenario=window.payFixture.scenario;
 if(scenario===kind+'-loading')await new Promise(resolve=>setTimeout(resolve,60000));
 if(scenario===kind+'-error')throw new Error('No hay conexión con el servicio.');};
export const payProviderOptions=['yape','plin'];
export const payStatusOptions=['detected','dismissed'];
export const listPayNotifications=async()=>{await wait('feed');return {items:[...window.payFixture.items],total:window.payFixture.items.length}};
export const paySalesMethods=async()=>{await wait('methods');const scenario=window.payFixture.scenario;return {items:scenario==='methods-empty'?[]:scenario==='wrong-method'?[{code:'cash',name:'Yape'}]:scenario==='renamed-method'?[{code:'yape',name:'Cobro con billetera'}]:[{code:'cash',name:'Efectivo'},{code:'yape',name:'Yape'},{code:'plin',name:'Plin'}]}};
export const searchPayOrders=async q=>{await wait('orders');return {items:['empty','orders-empty'].includes(window.payFixture.scenario)||q==='sin resultados'?[]:[{id:'fixture-order',code:'PED-0042',channel:'salon',customerName:'Ana Quispe',total:'10.50',paymentStatus:'pending'}]}};
export const payOrderDetail=async id=>{await wait('detail');return {order:{code:'PED-0042',channel:'salon',customerName:'Ana Quispe',status:'confirmado',total:'10.50'},paidAmount:'0.00',remainingAmount:window.payFixture.scenario==='paid'?'0.00':window.payFixture.scenario==='over'?'8.00':'10.50',paymentStatus:'pending',payments:[]}};
export const listPayNotificationPage=async filter=>{await wait('table');const all=[window.payFixture.notification,{...window.payFixture.notification,id:'fixture-plin',provider:'plin',senderName:'María Quispe',amount:'25.00'},{...window.payFixture.notification,id:'fixture-dismissed',status:'dismissed',senderName:'Prueba de diagnóstico'}];const items=window.payFixture.scenario==='table-empty'?[]:all.filter(item=>(!filter.status||item.status===filter.status)&&(!filter.provider||item.provider===filter.provider));return {items,total:items.length}};
export const associatePayNotification=async(...args)=>{record('associate',args);return {...window.payFixture.notification,status:'associated'}};
export const dismissPayNotification=async(...args)=>{record('dismiss',args);return {...window.payFixture.notification,status:'dismissed'}};
`}));
}}],logLevel:"warning"});

const cssDir=resolve(root,".next/static/css");
const css=(await Promise.all((await readdir(cssDir)).filter(name=>name.endsWith(".css")).map(name=>readFile(resolve(cssDir,name),"utf8")))).join("\n")+"\n"+await readFile(resolve(out,"bundle.css"),"utf8");
if(!css.includes(".pay-provider-logo"))throw new Error("Build first: the production CSS must include the payment styles.");
const font=css.match(/@font-face\{[^}]*font-family:Plus Jakarta Sans;[^}]*unicode-range:u\+00\?\?[^}]*\}/)?.[0].match(/media\/([^/)]+\.woff2)/)?.[1];
if(!font)throw new Error("Build first to load the production font.");
const fontData=await readFile(resolve(root,".next/static/media",font));
createServer(async(req,res)=>{
 const url=new URL(req.url,"http://localhost");
 if(url.pathname==="/bundle.js"){res.setHeader("content-type","text/javascript");res.end(await readFile(resolve(out,"bundle.js")));return;}
 if(url.pathname==="/font.woff2"){res.setHeader("content-type","font/woff2");res.end(fontData);return;}
 if(["/brands/yape.png","/brands/plin.png"].includes(url.pathname)){res.setHeader("content-type","image/png");res.end(await readFile(resolve(root,"public",url.pathname.slice(1))));return;}
 res.setHeader("content-type","text/html; charset=utf-8");
 res.end(`<!doctype html><html lang="es"><head><meta name="viewport" content="width=device-width,initial-scale=1"><title>Pagos detectados · prueba local</title><style>${css}@font-face{font-family:Plus Jakarta Sans;font-weight:200 800;src:url(/font.woff2)}:root{--font-jakarta:"Plus Jakarta Sans"}</style></head><body><div id="root"></div><script src="/bundle.js"></script></body></html>`);
}).listen(5198,"127.0.0.1",()=>console.log("Pagos QA: http://127.0.0.1:5198 (?table&provider=plin&case=long|empty|orders-error|orders-loading|methods-error|detail-error|associated|paid|over)"));
