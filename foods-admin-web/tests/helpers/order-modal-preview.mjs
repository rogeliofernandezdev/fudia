// Browser QA of real shared modal + order presentation, using illustrative data only.
import {readFileSync,mkdirSync,readdirSync,writeFileSync} from "node:fs";
import {createRequire} from "node:module";
import {dirname,resolve} from "node:path";
import {fileURLToPath} from "node:url";
import {createServer} from "node:http";
import assert from "node:assert/strict";
import {renderDetail,exampleOrder} from "./order-detail-fixture.mjs";
const require=createRequire(import.meta.url);
const root=resolve(dirname(fileURLToPath(import.meta.url)),"../..");
const sourceCss=["styles/variables.css","styles/globals.css","design-system/styles/modal.css","design-system/styles/skeleton.css","modules/operations/orders/presentation/order-service.css","modules/operations/orders/presentation/order-detail.css"].map(path=>readFileSync(resolve(root,"src",path),"utf8").replace(/^@import[^;]+;/gm,"")).join("\n");
const fontCss=readdirSync(resolve(root,".next/static/css")).filter(name=>name.endsWith(".css")).map(name=>readFileSync(resolve(root,".next/static/css",name),"utf8")).join("\n");
const css=process.env.FOODS_PREVIEW_COMPILED_CSS==="1"?fontCss:sourceCss;
if(process.env.FOODS_PREVIEW_COMPILED_CSS==="1")assert.ok(css.includes(".modal-item-list")&&css.includes(".modal-detail-notes")&&!css.includes(".modal-item-columns"),"El build debe incluir la lista simple y las notas de la referencia elegida.");
const font=fontCss.match(/@font-face\{[^}]*font-family:Plus Jakarta Sans;[^}]*unicode-range:u\+00\?\?[^}]*\}/)?.[0]?.match(/media\/([^/)]+\.woff2)/)?.[1];
assert.ok(font,"Un build previo debe aportar la fuente Plus Jakarta Sans real.");
const output=resolve(process.argv[2]??resolve(root,".next/order-modal-preview"));mkdirSync(output,{recursive:true});
const server=createServer((req,res)=>{
 if(req.url==="/font.woff2"){res.setHeader("content-type","font/woff2");res.end(readFileSync(resolve(root,".next/static/media",font)));return;}
 const url=new URL(req.url,"http://localhost");
 const long=url.searchParams.has("long");
 let order=url.searchParams.has("delivery")?{...exampleOrder,channel:"delivery",tableName:undefined,customerName:"Rosa Quispe",customerPhone:"999 123 456",address:"Av. Los Olivos 124, departamento 302",reference:"Frente al parque, llamar al llegar",deliveryFee:"5",total:"35",remainingAmount:"35"}:long?{...exampleOrder,waiterName:"María José Fernández de la Cruz",items:Array.from({length:20},(_,index)=>({...exampleOrder.items[index%2],id:`item-${index}`,name:index===0?"Caldo de Gallina Tradicional con ingredientes y acompañamientos adicionales":"Producto de prueba para verificar desplazamiento",note:"Nota de cocina que conserva toda su información incluso en una pantalla pequeña"}))}:exampleOrder;
 const stage=url.searchParams.get("stage");
 if(stage)order={...order,status:stage,serviceItems:order.serviceItems.map(item=>({...item,status:stage,statusLabel:{nuevo:"Pendiente",preparando:"Preparando",listo:"Listo",entregado:"Entregado",cancelado:"Cancelado"}[stage]??item.statusLabel}))};
 if(stage==="en_camino")order={...order,channel:"delivery",tableName:undefined,customerName:"Rosa Quispe"};
 if(stage==="cancelado")order={...order,channel:"mostrador",tableName:undefined,customerName:"Rosa Quispe"};
 if(url.searchParams.has("unassigned"))order={...order,waiterId:undefined,waiterName:undefined};
 const account=url.searchParams.get("account");
 if(account==="paid"||account==="completed")order={...order,status:"entregado",accountState:"paid",paidAmount:"30",remainingAmount:"0",serviceItems:order.serviceItems.map(item=>({...item,status:"entregado",statusLabel:"Entregado"}))};
 if(account==="completed")order={...order,completedAt:"2026-10-10T13:00:00-05:00"};
 if(account==="awaiting")order={...order,billClosedAt:"2026-10-10T13:00:00-05:00",accountState:"awaiting_payment"};
 res.setHeader("content-type","text/html; charset=utf-8");
 res.end(`<!doctype html><html lang="es"><head><meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover"><style>@font-face{font-family:Plus Jakarta Sans;font-weight:200 800;src:url(/font.woff2)}:root{--font-jakarta:"Plus Jakarta Sans"}${css}</style></head><body><main style="padding:24px"><h1>Pedidos</h1><p>Vista de comprobación; datos ilustrativos, sin operaciones reales.</p></main>${renderDetail({order,userId:url.searchParams.has("readonly")?"another-waiter":undefined,skeleton:url.searchParams.has("skeleton"),editable:url.searchParams.has("editable"),error:url.searchParams.has("error")?"Error de prueba":undefined})}</body></html>`);
});
await new Promise(resolve=>server.listen(0,"127.0.0.1",resolve));
const {chromium}=require(process.env.FOODS_PLAYWRIGHT_PATH??"playwright");
let browser;
try{
 browser=await chromium.launch({executablePath:process.env.FOODS_BROWSER_PATH,headless:true});
 const results=[];
 for(const [name,width,height,touch,query] of [["desktop",1440,900,false,""],["user-1192",1192,712,false,""],["user-967",967,835,false,""],["mobile",390,844,true,""],["new-unassigned-mobile",390,844,true,"?stage=nuevo&unassigned"],["two-secondary-mobile",390,844,true,"?stage=nuevo&unassigned&editable"],["mobile-short",390,480,true,"?long"],["tablet",768,1024,true,"?long"],["delivery",967,835,false,"?delivery"],["delivery-mobile",390,844,true,"?delivery"],["skeleton",967,835,false,"?skeleton"],["skeleton-mobile",390,844,true,"?skeleton"],["error",390,844,true,"?error"],["readonly",967,835,false,"?readonly"],["readonly-mobile",390,844,true,"?readonly"],["account-paid",390,844,true,"?readonly&account=paid"],["account-completed",390,844,true,"?readonly&account=completed"],["account-awaiting",967,835,false,"?readonly&account=awaiting"],...["nuevo","preparando","listo","en_camino","entregado","cancelado"].map(stage=>[`state-${stage}`,1192,835,false,`?stage=${stage}`])]){
  const context=await browser.newContext({viewport:{width,height},hasTouch:touch,reducedMotion:"reduce"});
  const page=await context.newPage();await page.goto(`http://127.0.0.1:${server.address().port}/${query}`);await page.evaluate(()=>document.fonts.ready);
  await page.evaluate(()=>Promise.all(document.getAnimations().filter(animation=>animation.effect?.getTiming().iterations!==Infinity).map(animation=>animation.finished.catch(()=>{}))));
  const geometry=await page.evaluate(()=>{
   const luminance=color=>{
    const rgb=color.match(/[\d.]+/g).slice(0,3).map(Number).map(value=>value/255).map(value=>value<=.04045?value/12.92:((value+.055)/1.055)**2.4);
    return rgb[0]*.2126+rgb[1]*.7152+rgb[2]*.0722;
   };
   const header=document.querySelector(".crud-modal>header");
   const headerRect=header.getBoundingClientRect(),panelRect=document.querySelector(".crud-modal").getBoundingClientRect();
   if(document.querySelector(".modal-accent")||Math.abs(headerRect.y-panelRect.y)>0.5||Math.abs(headerRect.width-panelRect.width)>0.5)throw new Error("La cabecera debe ocupar el borde superior completo, sin franja");
   const headerBg=luminance(getComputedStyle(header).backgroundColor);
   for(const label of header.querySelectorAll("h2,p,small")){
    const text=luminance(getComputedStyle(label).color);
    if((Math.max(text,headerBg)+.05)/(Math.min(text,headerBg)+.05)<4.5)throw new Error("Cabecera con contraste insuficiente");
   }
   if(header.getBoundingClientRect().height>65)throw new Error("La cabecera breve debe medir 64 px");
   for(const pill of document.querySelectorAll(".status,.modal-item-quantity")){
    const style=getComputedStyle(pill),a=luminance(style.color),b=luminance(style.backgroundColor);
    if((Math.max(a,b)+.05)/(Math.min(a,b)+.05)<4.5)throw new Error(`Contraste insuficiente: ${pill.textContent}`);
   }
   for(const quantity of document.querySelectorAll(".modal-item-quantity")){
    const style=getComputedStyle(quantity),rect=quantity.getBoundingClientRect();
    if(Math.abs(rect.width-rect.height)>0.5||parseFloat(style.borderTopLeftRadius)<rect.width/2)throw new Error("La cantidad debe conservar su círculo");
    if(style.color!==getComputedStyle(header).backgroundColor)throw new Error("La cantidad debe reutilizar primary-700 de la paleta");
   }
   const info=document.querySelector(".order-detail-information");
   if(info?.querySelector("time")&&info.querySelector(".order-attribution")){
    if(!(info.querySelector(".order-attribution").compareDocumentPosition(info.querySelector("time"))&Node.DOCUMENT_POSITION_FOLLOWING))throw new Error("El mozo debe preceder a la fecha");
    const date=info.querySelector(".modal-inline-date").getBoundingClientRect();
    if(Math.abs(date.right-info.getBoundingClientRect().right)>1)throw new Error("La fecha debe estar al extremo derecho");
    for(const row of info.querySelectorAll("dl>div")){
     const label=getComputedStyle(row.querySelector("dt")),value=getComputedStyle(row.querySelector("dd"));
     if(label.fontSize!==value.fontSize||label.fontSize!=="14px"||label.lineHeight!==value.lineHeight)throw new Error("Etiqueta y nombre deben compartir 14 px e interlineado");
    }
   }
   const list=document.querySelector(".modal-item-list"),notes=document.querySelector(".modal-detail-notes"),summary=document.querySelector(".modal-summary");
   if(list){
    if(parseFloat(getComputedStyle(list).borderTopWidth)!==0||document.querySelector(".modal-item-columns"))throw new Error("La lista no debe tener borde exterior ni encabezados de tabla");
    if(notes&&list.contains(notes))throw new Error("Las notas deben tener una sección independiente");
   }
   if(summary&&parseFloat(getComputedStyle(summary).borderTopWidth)<1)throw new Error("Resumen sin separador superior");
   if(summary){
    const background=luminance(getComputedStyle(summary).backgroundColor);
    for(const figure of summary.querySelectorAll(".modal-summary-line span,.modal-summary-line b,.modal-summary-total span,.modal-summary-total strong")){
     const foreground=luminance(getComputedStyle(figure).color);
     if((Math.max(foreground,background)+.05)/(Math.min(foreground,background)+.05)<4.5)throw new Error(`Contraste insuficiente en resumen: ${figure.textContent}`);
    }
   }
   if(summary?.querySelector(".modal-summary-line b")&&innerWidth<=700&&summary.querySelector(".modal-summary-lines")?.children.length===2){
    const figures=[...summary.querySelectorAll(".modal-summary-line b,.modal-summary-total strong")].map(el=>el.getBoundingClientRect());
    if(figures.length!==3||Math.max(...figures.map(r=>r.y))-Math.min(...figures.map(r=>r.y))>1)throw new Error("Subtotal, Pagado y total deben compartir fila en móvil");
   }
   const accountStatus=summary?.querySelector(".status");
   if(accountStatus){
    const heading=summary.querySelector("h3"),badge=accountStatus.getBoundingClientRect(),title=heading.getBoundingClientRect();
    if(!accountStatus.classList.contains("emphasized")||!accountStatus.querySelector("svg")||parseInt(getComputedStyle(accountStatus).fontWeight)<700)throw new Error("Estado de cuenta sin icono ni énfasis");
    if(accountStatus.textContent.trim()==="Cuenta abierta"){
     const style=getComputedStyle(accountStatus);
     if(!accountStatus.classList.contains("sky")||style.backgroundColor!=="rgb(224, 242, 254)"||style.color!=="rgb(7, 89, 133)")throw new Error("Cuenta abierta debe usar el celeste tenue aprobado y texto oscuro");
     const confirmed=document.querySelector(".order-detail-section-head .status.blue");
     if(confirmed&&getComputedStyle(confirmed).backgroundColor===style.backgroundColor)throw new Error("Cuenta abierta y Confirmado deben tener colores distintos");
    }
    const label=accountStatus.textContent.trim();
    if(label==="Pagada"&&!accountStatus.classList.contains("green"))throw new Error("Pagada debe conservar verde");
    if(label==="Finalizada"&&!accountStatus.classList.contains("gray"))throw new Error("Finalizada debe usar gris neutro, no un tono verdoso");
    if(Math.abs(badge.y+badge.height/2-title.y-title.height/2)<1&&Math.abs(badge.right-summary.getBoundingClientRect().right+parseFloat(getComputedStyle(summary).paddingRight))>1)throw new Error("Estado de cuenta fuera del borde derecho del resumen");
   }
   const rect=el=>{if(!el)return null;const r=el.getBoundingClientRect();return {x:r.x,y:r.y,width:r.width,height:r.height,right:r.right,bottom:r.bottom};};
   const body=document.querySelector(".order-detail-body");
   if(/\b\d+ unidades?\b/.test(document.querySelector(".order-detail-section-head")?.textContent??""))throw new Error("No se debe repetir el contador de unidades");
   const textRect=el=>{
    if(!el)return null;
    const walker=document.createTreeWalker(el,NodeFilter.SHOW_TEXT);
    let node;while((node=walker.nextNode()))if(node.textContent.trim()){
     const range=document.createRange();range.selectNodeContents(node);
     return rect({getBoundingClientRect:()=>range.getClientRects()[0]});
    }
    return null;
   };
   const attribution=document.querySelector(".order-attribution>div");
   return {overflow:document.documentElement.scrollWidth>innerWidth,modal:rect(document.querySelector(".crud-modal")),close:rect(document.querySelector(".modal-close")),meta:rect(document.querySelector(".modal-heading-meta")),heading:rect(document.querySelector(".modal-heading")),attribution:attribution&&{label:textRect(attribution.querySelector("dt")),name:textRect(attribution.querySelector("dd"))},productsTitle:rect(document.querySelector(".order-detail-section-head h3")),accountTitle:rect(document.querySelector(".modal-summary h3")),summary:rect(document.querySelector(".modal-summary")),footer:rect(document.querySelector(".order-detail-actions")),body:{...rect(body),scrolls:body.scrollHeight>body.clientHeight},buttons:[...document.querySelectorAll(".order-detail-actions button,.order-detail-actions a")].map(rect)};
  });
  results.push({name,...geometry});
  assert.equal(geometry.overflow,false,name);
  assert.ok(geometry.modal.bottom<=height,`${name}: panel dentro del viewport`);
  if(geometry.summary)assert.ok(geometry.summary.bottom<=height,`${name}: resumen visible`);
  if(geometry.footer)assert.ok(geometry.footer.bottom<=height,`${name}: acciones visibles`);
  if(width<=600&&geometry.buttons.length){
   const rows=new Map();
   for(const button of geometry.buttons){const key=Math.round(button.y);rows.set(key,[...(rows.get(key)??[]),button]);}
   const bounds=await page.locator(".modal-action-layout").boundingBox();
   for(const buttons of rows.values()){
    assert.ok(Math.abs(buttons[0].x-bounds.x)<1&&Math.abs(buttons.at(-1).right-bounds.x-bounds.width)<1,`${name}: toda fila de acciones llena el ancho útil`);
    if(buttons.length===2)assert.ok(Math.abs(buttons[0].width-buttons[1].width)<1,`${name}: dos botones comparten ancho por igual`);
   }
   if(name==="new-unassigned-mobile"){assert.equal(geometry.buttons.length,2);assert.equal(rows.size,2);}
   if(name==="two-secondary-mobile"){assert.equal(geometry.buttons.length,3);assert.equal(rows.size,2);assert.equal([...rows.values()][0].length,2);}
  }
  if(touch)assert.ok(geometry.close.width>=44&&geometry.close.height>=44,`${name}: cierre táctil`);
  assert.equal(geometry.meta,null,`${name}: sin estados junto al cierre`);
  assert.ok(Math.abs(geometry.heading.y+geometry.heading.height/2-geometry.close.y-geometry.close.height/2)<0.5,`${name}: título y cierre alineados`);
  if(geometry.attribution?.label&&geometry.attribution.name)assert.ok(Math.abs(geometry.attribution.label.y-geometry.attribution.name.y)<0.5,`${name}: etiqueta y nombre del mozo alineados`);
  if(geometry.summary){
   assert.ok(Math.abs(geometry.summary.width-geometry.body.width)<0.5,`${name}: una sola columna`);
   assert.ok(geometry.summary.y>=geometry.body.bottom-0.5,`${name}: resumen bajo los productos`);
  }
  if(query.includes("long")){
   assert.ok(geometry.body.scrolls,`${name}: productos desplazables`);
   assert.ok(geometry.body.height>=80,`${name}: espacio útil para leer productos`);
   const before=geometry.summary.y;
   await page.locator(".order-detail-body").evaluate(el=>{el.scrollTop=el.scrollHeight;});
   const after=await page.locator(".modal-summary").evaluate(el=>el.getBoundingClientRect().y);
   assert.equal(after,before,`${name}: resumen persistente`);
  }
  await page.screenshot({path:resolve(output,`${name}.png`),fullPage:true});await context.close();
  console.log(`${name}: sin desborde, cuenta y acciones visibles; alineación comprobada`);
 }
 writeFileSync(resolve(output,"geometry.json"),JSON.stringify(results,null,2));
}finally{await browser?.close();await new Promise(resolve=>server.close(resolve));}
