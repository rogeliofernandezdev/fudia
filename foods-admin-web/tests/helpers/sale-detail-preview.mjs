// Real sale dialog, read-only fixture: no authentication, API requests or writes.
// Usage after build: node tests/helpers/sale-detail-preview.mjs <artifact-directory>
import {readFileSync,readdirSync,mkdirSync,writeFileSync} from "node:fs";
import {createRequire} from "node:module";
import {resolve,dirname} from "node:path";
import {fileURLToPath} from "node:url";
import {createServer} from "node:http";
import assert from "node:assert/strict";
import React from "react";
import {renderToStaticMarkup} from "react-dom/server";
import {detailComponents,exampleOrder} from "./order-detail-fixture.mjs";

const require=createRequire(import.meta.url);
const root=resolve(dirname(fileURLToPath(import.meta.url)),"../..");
const cssDir=resolve(root,".next/static/css");
const css=readdirSync(cssDir).filter(name=>name.endsWith(".css")).map(name=>readFileSync(resolve(cssDir,name),"utf8")).join("\n");
const font=css.match(/@font-face\{[^}]*font-family:Plus Jakarta Sans;[^}]*unicode-range:u\+00\?\?[^}]*\}/)?.[0].match(/media\/([^/)]+\.woff2)/)?.[1];
assert.ok(font,"Run build first to load the real Plus Jakarta Sans font.");
const out=resolve(process.argv[2]??resolve(root,".next/sale-detail-preview"));mkdirSync(out,{recursive:true});
const pos=process.argv[3]==="pos";
const {SaleDetailDialog,POSDetailDialog}=detailComponents({includePOS:pos});
const noop=()=>{};
function fixture(query){
 const long=query.has("long");
 const data={order:{...exampleOrder,code:"PED-61461B13",tableName:"03",waiterName:long?"Daniel Machaca Rojasa Fernández de los Ríos y del Valle":"Daniel Machaca Rojasa",customerName:long?"Familia Fernández de los Ríos y del Valle — reserva de celebración familiar":"",channel:query.has("delivery")?"delivery":"salon"},paidAmount:"30",remainingAmount:"0",paymentStatus:"paid",payments:[]};
 const money=value=>`S/ ${Number(value).toFixed(2)}`;
 const date=value=>new Intl.DateTimeFormat("es-PE",{dateStyle:"medium",timeStyle:"short",timeZone:"America/Lima"}).format(new Date(value));
 if(pos){
  data.paidAmount=query.has("paid")?"10":"0";data.remainingAmount=query.has("paid")?"20":"30";data.paymentStatus=query.has("paid")?"partial":"pending";
  if(query.has("paid"))data.payments=[{id:"preview-payment",method:"cash",amount:"12",refundedAmount:"2",netAmount:"10",cashRegisterName:"Caja principal",createdByName:"Ana Fernández",createdAt:exampleOrder.createdAt,reference:"Referencia de prueba"}];
  return renderToStaticMarkup(React.createElement(POSDetailDialog,{data,loading:query.has("loading"),formatMoney:money,formatDateTime:date,close:noop,retry:noop,ticket:noop,canManage:!query.has("readonly"),hasShift:true,refund:noop}));
 }
 data.payments=query.has("empty")?[]:[{id:"sale-payment",method:"cash",methodName:"Efectivo",amount:"30",refundedAmount:"0",netAmount:"30",cashRegisterName:long?"Caja principal del salón y terraza — turno de la noche":"Caja principal",createdByName:long?"Yulisa Meneses Fernández de los Ríos y del Valle":"Yulisa Meneses",createdAt:exampleOrder.createdAt,reference:long?"Referencia de pago que conserva toda su información sin recortes":""}];
 if(query.has("refund")){data.paidAmount="20";data.remainingAmount="10";data.paymentStatus="partial";data.payments[0]={...data.payments[0],refundedAmount:"10",netAmount:"20"};}
 if(query.has("many"))data.payments=Array.from({length:20},(_,index)=>({...data.payments[0],id:`sale-payment-${index}`,amount:"1.50",netAmount:"1.50"}));
 return renderToStaticMarkup(React.createElement(SaleDetailDialog,{data,loading:query.has("loading"),money,date,close:noop,retry:noop,ticket:noop}));
}
const server=createServer((req,res)=>{
 if(req.url==="/font.woff2"){res.setHeader("content-type","font/woff2");res.end(readFileSync(resolve(root,".next/static/media",font)));return;}
 const url=new URL(req.url,"http://localhost");res.setHeader("content-type","text/html; charset=utf-8");
 res.end(`<!doctype html><html lang="es"><head><meta name="viewport" content="width=device-width, initial-scale=1"><style>${css}@font-face{font-family:Plus Jakarta Sans;font-weight:200 800;src:url(/font.woff2)}:root{--font-jakarta:"Plus Jakarta Sans"}</style></head><body>${fixture(url.searchParams)}</body></html>`);
});
await new Promise(resolve=>server.listen(0,"127.0.0.1",resolve));
const {chromium}=require(process.env.FOODS_PLAYWRIGHT_PATH??"playwright");let browser;
try{
 browser=await chromium.launch({headless:true,executablePath:process.env.FOODS_BROWSER_PATH});
 const results=[];
 for(const [name,width,height,query] of [["desktop",1440,900,""],["mobile",390,844,""],["desktop-long",1440,900,"?long"],["mobile-short-long",390,480,"?long"],["delivery",1440,900,"?delivery"],["loading",390,844,"?loading"],...(pos?[["partial-payment",1440,900,"?paid"],["partial-mobile",390,844,"?paid"],["read-only",390,844,"?paid&readonly"]]:[["refund-mobile",390,844,"?refund"],["many-payments",390,844,"?long&many"],["empty-payments",390,844,"?empty"]]),...[600,610,620,630].map(width=>[`breakpoint-${width}`,width,700,pos?"?long&paid":"?long"])]){
  const page=await browser.newPage({viewport:{width,height},reducedMotion:"reduce"});
  await page.goto(`http://127.0.0.1:${server.address().port}/${query}`);await page.evaluate(()=>document.fonts.ready);
  const geometry=await page.evaluate(()=>{
   const rect=el=>{const r=el.getBoundingClientRect();return {x:r.x,y:r.y,right:r.right,bottom:r.bottom,width:r.width,height:r.height};};
   const panel=document.querySelector(".sales-detail-modal,.pos-detail-modal"),body=document.querySelector(".sales-detail-body,.pos-detail-body"),context=document.querySelector(".sales-detail-context,.pos-detail-context"),summary=panel.querySelector(".modal-summary");
   if(summary&&rect(summary).y<rect(body).bottom-1)throw new Error("The summary must sit below the scrollable body");
   const metadata=[...context.querySelectorAll("dl>div")].filter(el=>el.querySelector("dt")).map(el=>{
    const label=el.querySelector("dt"),value=el.querySelector("dd"),range=document.createRange();range.selectNodeContents(label);
    return {labelRect:rect(range),valueRect:rect(value),labelSize:parseFloat(getComputedStyle(label).fontSize),valueSize:parseFloat(getComputedStyle(value).fontSize),labelWeight:Number(getComputedStyle(label).fontWeight),valueWeight:Number(getComputedStyle(value).fontWeight),labelColor:getComputedStyle(label).color,valueColor:getComputedStyle(value).color};
   });
   return {overflow:document.documentElement.scrollWidth>innerWidth,panel:rect(panel),context:rect(context),fields:[...context.querySelectorAll("dl>div")].map(rect),metadata,clipped:[...context.querySelectorAll("dd")].some(el=>el.scrollWidth>el.clientWidth+1),headerExtras:!!panel.querySelector("header small,header p,header time,header .status"),footer:panel.querySelector("footer")&&rect(panel.querySelector("footer")),summary:summary&&rect(summary),refunds:[...panel.querySelectorAll("button")].filter(button=>button.textContent.includes("Devolver")).length,scrolls:body.scrollHeight>body.clientHeight};
  });
  assert.equal(geometry.overflow,false,name);assert.equal(geometry.clipped,false,name);assert.equal(geometry.headerExtras,false,name);
  assert.ok(geometry.panel.bottom<=height+1,`${name}: modal fits viewport`);
  assert.ok(geometry.panel.x>=0&&geometry.panel.right<=width+1,`${name}: panel is not clipped horizontally`);
  if(geometry.footer)assert.ok(geometry.footer.bottom<=height+1,`${name}: footer remains visible`);
  if(width<=600&&geometry.footer){
   const actions=await page.locator("footer .button").evaluateAll(buttons=>buttons.map(button=>({radius:parseFloat(getComputedStyle(button).borderRadius),height:button.getBoundingClientRect().height,clipped:button.scrollWidth>button.clientWidth+1,text:button.textContent,secondary:button.classList.contains("secondary")})));
   actions.forEach(action=>{assert.ok(action.radius>=action.height/2,`${name}: mobile actions are pills`);assert.ok(action.height>=44,`${name}: touch target`);assert.equal(action.clipped,false,`${name}: action label is not clipped`);});
   const viewPayment=actions.find(action=>action.text==="Ver resumen de pago");assert.ok(viewPayment?.secondary,`${name}: viewing payment is secondary`);
  }
  assert.ok(geometry.summary.bottom<=height+1,`${name}: summary stays visible`);
  if(pos&&query.includes("readonly"))assert.equal(geometry.refunds,0,`${name}: no unauthorized actions`);
  geometry.metadata.forEach(field=>{assert.equal(field.labelSize,12,`${name}: informational labels stay compact`);assert.equal(field.valueSize,14,`${name}: informational data does not inherit input zoom sizing`);assert.equal(field.labelWeight,600,`${name}: labels are semibold`);assert.equal(field.valueWeight,400,`${name}: values are regular`);assert.notEqual(field.labelColor,field.valueColor,`${name}: hierarchy uses different palette roles`);});
  const sectionSizes=await page.locator('.sales-detail-section h3,.pos-detail-section h3,.modal-summary h3').evaluateAll(headings=>headings.map(heading=>parseFloat(getComputedStyle(heading).fontSize)));
  sectionSizes.forEach(size=>assert.equal(size,16,`${name}: section headings lead the informative data`));
  const noteSizes=await page.locator('.sales-detail-lines p,.sales-detail-payments p').evaluateAll(notes=>notes.map(note=>parseFloat(getComputedStyle(note).fontSize)));
  noteSizes.forEach(size=>assert.equal(size,12,`${name}: product notes stay below names and section headings`));
  if(width>620&&!query.includes("delivery")){
   const [a,b,c,d]=geometry.fields;assert.ok(Math.abs(a.y-b.y)<1&&Math.abs(c.y-d.y)<1,`${name}: two paired rows`);
   assert.ok(Math.abs(a.x-c.x)<1&&Math.abs(b.x-d.x)<1,`${name}: equal aligned columns`);
  }else if(width<=620){
   const [first]=geometry.fields;
   geometry.fields.forEach((field,index)=>{assert.equal(field.x,first.x,`${name}: a single aligned metadata list`);if(index)assert.ok(field.y>=geometry.fields[index-1].bottom+9,`${name}: compact rows are separated`);});
   if(geometry.metadata.length){
    geometry.metadata.forEach(field=>{assert.ok(Math.abs(field.labelRect.x-geometry.metadata[0].labelRect.x)<1,`${name}: label column stays aligned`);assert.ok(Math.abs(field.valueRect.x-geometry.metadata[0].valueRect.x)<1,`${name}: value column stays aligned`);assert.ok(Math.abs(field.valueRect.y-field.labelRect.y)<5,`${name}: labels align with the first value line`);});
    if(!query.includes("long"))assert.ok(geometry.context.height<=140,`${name}: four fields stay compact`);
   }
  }
  if(!pos&&!query.includes("loading")){
   const lists=await page.locator(".sales-detail-lines,.sales-detail-payments").evaluateAll(lists=>lists.map(list=>({border:getComputedStyle(list).borderTopWidth,radius:getComputedStyle(list).borderRadius,rows:list.querySelectorAll('.modal-item-row').length})));
   lists.forEach(list=>{assert.equal(list.border,"0px",`${name}: no card border`);assert.equal(list.radius,"0px",`${name}: no card radius`);assert.ok(list.rows>0);});
   assert.equal(await page.locator(".sales-detail-body .modal-summary").count(),0,`${name}: totals are not inside the body`);
   if(geometry.scrolls){
    await page.locator(".sales-detail-body").evaluate(body=>{body.scrollTop=body.scrollHeight;});
    const summary=await page.locator(".modal-summary").boundingBox();assert.ok(Math.abs(summary.y-geometry.summary.y)<1,`${name}: summary is fixed during body scroll`);
    await page.locator(".sales-detail-body").evaluate(body=>{body.scrollTop=0;});
   }
  }
  await page.screenshot({path:resolve(out,`${name}.png`),fullPage:true});
  if(name==="mobile")await page.locator(pos?".pos-detail-context":".sales-detail-context").screenshot({path:resolve(out,"mobile-context.png")});
  results.push({name,...geometry});console.log(`${name}: aligned, no clipping, viewport safe`);await page.close();
 }
 writeFileSync(resolve(out,"geometry.json"),JSON.stringify(results,null,2));
}finally{await browser?.close();await new Promise(resolve=>server.close(resolve));}
