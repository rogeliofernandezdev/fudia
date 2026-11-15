// QA local de acciones reales en la fila y seguimiento del detalle; datos ilustrativos.
import {readFileSync,readdirSync,existsSync,mkdirSync} from "node:fs";
import {createRequire} from "node:module";
import {dirname,resolve} from "node:path";
import {fileURLToPath} from "node:url";
import {createServer} from "node:http";
import assert from "node:assert/strict";
import {build} from "esbuild";

const root=resolve(dirname(fileURLToPath(import.meta.url)),"../..");
const bundle=await build({absWorkingDir:root,bundle:true,write:false,outfile:"preview.js",format:"iife",jsx:"automatic",define:{"process.env.NODE_ENV":'"production"',"process.env":"{}"},stdin:{resolveDir:root,loader:"tsx",contents:`
 import React from 'react';
 import {createRoot} from 'react-dom/client';
 import {QueryClient,QueryClientProvider,useQuery} from '@tanstack/react-query';
 import {FeedbackProvider} from './src/providers/feedback-provider';
 import {OrderDetailContent} from './src/modules/operations/orders/presentation/order-detail-dialog';
 import {OrderDispatchActions} from './src/modules/delivery/presentation/order-dispatch-actions';
 import {orderStatus} from './src/modules/operations/orders/presentation/order-meta';
 import {DataTable} from './src/design-system/data-table';
 import {RowActionButton,Status} from './src/design-system/page-header';
 import {Dialog} from './src/design-system/dialog';
 function Preview(){
   const [opened,setOpened]=React.useState(false);
   const order=useQuery({queryKey:['orders','qa'],queryFn:()=>fetch('/api/admin/orders/preview-order'+location.search).then(r=>r.json())});
   const value=order.data,meta=value&&orderStatus(value.status);
   return value?<><main className="orders-page-shell"><h1>Pedidos</h1><section className="panel management standardized-management orders-panel"><DataTable label="Listado de pedidos" className="orders-table">
    <thead><tr>{['Pedido','Canal','Estado','Registrado','Total','Acciones'].map(label=><th key={label}>{label}</th>)}</tr></thead>
    <tbody><tr className="orders-row" data-dispatch-status={value.deliveryDispatchStatus} onClick={event=>{if(!event.target.closest('button,a'))setOpened(true)}}><td><b>{value.customerName}</b><small>{value.code}</small></td><td>Delivery</td><td>{meta&&<Status tone={meta.tone} solid>{meta.label}</Status>}</td><td>Hoy</td><td><b className="order-table-total">S/ {Number(value.total).toFixed(2)}</b></td><td><div className="orders-table-actions"><RowActionButton action="view" label="Ver detalle de Rosa Quispe" onClick={()=>setOpened(true)}/><OrderDispatchActions order={value}/></div></td></tr></tbody>
   </DataTable></section></main>
   {opened&&<div className="modal-backdrop"><Dialog className="crud-modal order-detail" onResponseClose={()=>setOpened(false)} aria-labelledby="order-detail-title"><OrderDetailContent variant="order" order={value} canManage={!location.search.includes('readonly')} busy={false} close={()=>setOpened(false)} retry={()=>{}} advance={()=>{}} add={()=>{}} cancel={()=>{}}/></Dialog></div>}</>:null;
 }
 createRoot(document.getElementById('root')).render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><FeedbackProvider><Preview/></FeedbackProvider></QueryClientProvider>);
 `},plugins:[{name:"qa-facades",setup(builder){
 builder.onResolve({filter:/^next\/link$/},()=>({path:"link",namespace:"facade"}));
 builder.onResolve({filter:/^@\//},args=>{
  if(args.path==="@/design-system")return {path:"design",namespace:"facade"};
  if(args.path==="@/providers")return {path:"providers",namespace:"facade"};
  if(args.path==="@/modules/delivery")return {path:resolve(root,"src/modules/delivery/presentation/order-dispatch-block.tsx")};
  const base=resolve(root,"src",args.path.slice(2));
  return {path:[`${base}.tsx`,`${base}.ts`,resolve(base,"index.ts")].find(existsSync)};
 });
 builder.onLoad({filter:/.*/,namespace:"facade"},args=>({resolveDir:root,loader:"tsx",contents:args.path==="link"?"export default function Link({children,...props}){return <a {...props}>{children}</a>}":args.path==="providers"?`
 export {useFeedback} from './src/providers/feedback-provider';
 export const useMoney=()=>value=>'S/ '+Number(value).toFixed(2);
 export const useSession=()=>({user:{id:'qa-user'},location:{country:'PE',timezone:'America/Lima'},cashCollection:false,can:()=>!location.search.includes('readonly')});
 `:["page-header","icons","icon-tile","detail-view","tabs","skeleton","states","modal","confirm-dialog","form-field","form-controls"].map(name=>`export * from './src/design-system/${name}'`).join(';')}));
 }}]});
const css=["styles/variables.css","styles/globals.css","modules/operations/styles/orders.css",...readdirSync(resolve(root,"src/design-system/styles")).filter(name=>name.endsWith(".css")).map(name=>`design-system/styles/${name}`)]
 .map(name=>readFileSync(resolve(root,"src",name),"utf8").replace(/^@import.*$/gm,"")).join("\n")+"\n"+(bundle.outputFiles.find(file=>file.path.endsWith(".css"))?.text??"")+"\nbody{padding:16px}.orders-page-shell{min-width:0}";
const script=bundle.outputFiles.find(file=>file.path.endsWith(".js")).text;
const sessions=new Map();
const calls=[];
const initial=status=>({orderId:"preview-order",status,round:1,offers:[],...(status==="accepted"||status==="picked_up"||status==="delivered"?{partnerName:"Central de reparto San Miguel",partnerPhone:"51987654321",courierName:"Rosa Quispe",acceptedAt:"2026-11-15T12:00:00Z"}:{}),...(status==="picked_up"||status==="delivered"?{pickedUpAt:"2026-11-15T12:10:00Z"}:{}),...(status==="delivered"?{deliveredAt:"2026-11-15T12:30:00Z"}:{})});
const server=createServer(async(req,res)=>{
 const url=new URL(req.url,"http://localhost"),scenario=url.searchParams.get("scenario")||"unassigned";
 const sid=req.headers.cookie?.match(/qa=([^;]+)/)?.[1]||scenario;
 if(url.pathname==="/preview.js"){res.setHeader("content-type","text/javascript");res.end(script);return;}
 if(url.pathname==="/api/admin/orders/preview-order"){
  const status=sessions.get(sid)?.status||(["unassigned","own","searching","picked_up","delivered","cancelled"].includes(scenario)?scenario:scenario==="empty"?"":"accepted");
  res.setHeader("content-type","application/json");
  res.end(JSON.stringify({id:"preview-order",code:"PED-004",channel:"delivery",status:scenario==="closed"?"entregado":"listo",deliveryDispatchStatus:status,customerName:"Rosa Quispe",customerPhone:"",address:"Av. Los Olivos 124",reference:"Frente al parque",notes:"",subtotal:"30",deliveryFee:status==="own"?"5":"0",deliveryPartnerFee:"5",total:status==="own"?"35":"30",remainingAmount:status==="own"?"35":"30",paidAmount:"0",createdAt:"2026-11-15T11:50:00Z",updatedAt:"2026-11-15T12:00:00Z",items:[{id:"one",productId:"p-one",name:"Lomo saltado",qty:"1",unitPrice:"30",note:""}]}));return;
 }
 if(url.pathname.startsWith("/api/admin/orders/preview-order/delivery-dispatch")){
  res.setHeader("content-type","application/json");
  calls.push({sid,method:req.method,path:url.pathname});
  if(sid==="error"||sid==="forbidden"||sid==="empty"){
   res.statusCode=sid==="forbidden"?403:sid==="empty"?404:503;
   res.end(JSON.stringify({code:sid==="empty"?"delivery_dispatch_not_found":"test_error",message:"No pudimos consultar el reparto de prueba."}));return;
  }
  if(sid==="loading"){setTimeout(()=>res.end(JSON.stringify(initial("accepted"))),15000);return;}
  if(!sessions.has(sid))sessions.set(sid,initial(["readonly","closed","failure","alert","long"].includes(sid)?"accepted":sid));
  const data=sessions.get(sid);
  if(sid==="alert")Object.assign(data,{pickupAlert:"critical",pickupWaitSeconds:540});
  if(sid==="long")Object.assign(data,{partnerName:"Central "+"Fernández".repeat(20),courierName:"Rosa "+"Quispe".repeat(20)});
  if(req.method==="POST"){
   if(sid==="failure"){res.statusCode=409;res.end(JSON.stringify({code:"delivery_payment_conflict",message:"Devuelve primero los pagos registrados antes de cambiar a delivery propio."}));return;}
   Object.assign(data,initial(url.pathname.endsWith("/retry")?"searching":"own"));
  }
  res.end(JSON.stringify(data));return;
 }
 res.setHeader("set-cookie",`qa=${scenario}; Path=/; SameSite=Lax`);
 res.setHeader("content-type","text/html; charset=utf-8");
 res.end(`<!doctype html><html lang="es"><head><meta name="viewport" content="width=device-width,initial-scale=1"><title>QA · Reparto del pedido</title><style>${css}</style></head><body><div id="root"></div><script src="/preview.js"></script></body></html>`);
});
await new Promise(done=>server.listen(5197,"127.0.0.1",done));
console.log("QA local: http://localhost:5197/?scenario=unassigned|accepted|own|picked_up|delivered|readonly|closed|error|empty|loading|failure|alert|long");
if(process.argv.includes("--verify")){
 const require=createRequire(import.meta.url);
 const {chromium}=require(process.env.FOODS_PLAYWRIGHT_PATH||"playwright");
 const browser=await chromium.launch({headless:true,executablePath:process.env.FOODS_BROWSER_PATH});
 const output=resolve(root,".cache/order-dispatch-preview");mkdirSync(output,{recursive:true});
 try{
  for(const width of [1440,390]){
   for(const scenario of ["unassigned","accepted","own","picked_up","delivered","readonly","closed","error","forbidden","empty","loading","failure","alert","long"]){
    const callStart=calls.length;
    const context=await browser.newContext({viewport:{width,height:844},reducedMotion:"reduce"});
    const page=await context.newPage();
    await page.goto('http://127.0.0.1:5197/?scenario='+scenario);
    const row=page.locator('.orders-row');await row.waitFor();
    const retry=row.getByRole('button',{name:'Reintentar centrales para PED-004',exact:true});
    const own=row.getByRole('button',{name:'Usar delivery propio en PED-004',exact:true});
    assert.equal(await retry.count(),Number(['unassigned','own'].includes(scenario)),scenario);
    assert.equal(await own.count(),Number(!['own','picked_up','delivered','readonly','closed','empty'].includes(scenario)),scenario);
    assert.equal(await page.locator('.order-detail').count(),0,'la fila no abre el detalle automáticamente');
    assert.equal(calls.slice(callStart).filter(call=>call.sid===scenario&&call.method==='GET').length,0,scenario+': sin consulta adicional por fila');
    await page.locator('.data-table-viewport').evaluate(el=>{el.scrollLeft=el.scrollWidth});
    assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,scenario+': viewport');
    for(const button of await row.locator('button').all()){
     const box=await button.boundingBox();
     if(width===390)assert.ok(box.width>=44&&box.height>=44,scenario+': área táctil');
     assert.ok(await button.evaluate(el=>{const r=el.getBoundingClientRect(),hit=document.elementFromPoint(r.x+r.width/2,r.y+r.height/2);return hit===el||el.contains(hit)}),scenario+': acción accesible tras desplazar la tabla');
    }
    if(['unassigned','accepted','readonly'].includes(scenario))await page.screenshot({path:resolve(output,'row-'+scenario+'-'+width+'.png')});
    if(scenario==='unassigned'){
     await retry.click();
     await page.waitForFunction(()=>document.querySelector('.orders-row')?.dataset.dispatchStatus==='searching');
     assert.equal(await page.locator('.order-detail').count(),0,'reintentar no abre detalle');
     assert.equal(calls.filter(call=>call.sid===scenario&&call.method==='POST'&&call.path.endsWith('/retry')).length,width===1440?1:2);
     assert.equal(calls.filter(call=>call.sid===scenario&&call.method==='GET').length,0,'reintentar desde fila no necesita detalle');
    }
    if(scenario==='accepted'||scenario==='failure'){
     await own.click();
     await page.locator('.confirm-description').first().click();
     assert.equal(await page.locator('.order-detail').count(),0,'la confirmación tampoco abre detalle');
     await page.locator('.confirm-card').getByRole('button',{name:'Cancelar',exact:true}).click();
     assert.equal(calls.filter(call=>call.sid===scenario&&call.method==='POST').length,width===1440?0:1,'cancelar no cambia el reparto');
     await own.click();
     await page.locator('.confirm-card').getByRole('button',{name:'Usar propio',exact:true}).click();
     if(scenario==='failure'){
      await page.getByText('Devuelve primero los pagos registrados antes de cambiar a delivery propio.',{exact:true}).waitFor();
      assert.equal(await row.getAttribute('data-dispatch-status'),'accepted');
     }else{
      await page.waitForFunction(()=>document.querySelector('.orders-row')?.dataset.dispatchStatus==='own');
      await row.getByText('S/ 35.00',{exact:true}).waitFor();
     }
     assert.equal(await page.locator('.order-detail').count(),0,'usar propio no abre detalle');
    }
    if(['error','forbidden','empty','loading','long','alert'].includes(scenario)){
     await row.getByRole('button',{name:'Ver detalle de Rosa Quispe',exact:true}).click();
     const block=page.getByRole('region',{name:'Reparto',exact:true});
     if(scenario==='loading')await block.getByRole('status',{name:'Cargando reparto'}).waitFor();
     else if(['error','forbidden'].includes(scenario))await block.getByRole('alert').waitFor();
     else if(scenario==='empty')await block.getByText('Este pedido aún no tiene despacho a centrales registrado.').waitFor();
     else await block.locator('.status').first().waitFor();
     assert.equal(await block.getByRole('button',{name:/Reintentar centrales|Usar delivery propio/}).count(),0,'el detalle es solo seguimiento');
     if(scenario!=='empty')assert.equal(await page.getByRole('button',{name:'En camino',exact:true}).count(),0,scenario+': el avance sigue siendo del motorizado');
     if(scenario==='forbidden')assert.equal(await block.getByRole('button',{name:'Reintentar',exact:true}).count(),0);
    }
    await context.close();sessions.delete(scenario);
   }
  }
  console.log('PASS: 28 vistas; acciones en la fila sin abrir detalle ni consultas extra, permisos, móvil, confirmación, reintento y totales.');
 }finally{await browser.close();server.closeAllConnections();server.close();}
}
