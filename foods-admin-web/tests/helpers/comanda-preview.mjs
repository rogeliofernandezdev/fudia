// Comanda real con datos ilustrativos. Sin autenticación, API ni registros.
import {readFileSync,readdirSync,mkdirSync,writeFileSync} from "node:fs";
import {createRequire} from "node:module";
import {resolve,dirname} from "node:path";
import {fileURLToPath} from "node:url";
import {createServer} from "node:http";
import assert from "node:assert/strict";
import React from "react";
import {renderToStaticMarkup} from "react-dom/server";
import {detailComponents} from "./order-detail-fixture.mjs";
const require=createRequire(import.meta.url);
const root=resolve(dirname(fileURLToPath(import.meta.url)),"../..");
const css=readdirSync(resolve(root,".next/static/css")).filter(name=>name.endsWith(".css")).map(name=>readFileSync(resolve(root,".next/static/css",name),"utf8")).join("\n");
const font=css.match(/@font-face\{[^}]*font-family:Manrope;[^}]*unicode-range:u\+00\?\?[^}]*\}/)?.[0].match(/media\/([^/)]+\.woff2)/)?.[1];
assert.ok(font,"Requiere el build con la fuente real");
const out=resolve(process.argv[2]??resolve(root,".next/comanda-preview"));mkdirSync(out,{recursive:true});
function fixture(query){
 const {ComandaView}=detailComponents({includeComanda:true,category:query.has("combos")?"__combos":query.get("category")??"",categoriesOpen:query.has("categories"),comandaOpen:query.has("sheet"),catalogState:query.get("state")??"ready",extraCategories:query.has("many")});
 const initial={channel:"salon",customerName:"",customerPhone:"",address:"",reference:"",tableId:"t1",notes:"",deliveryFee:"0",lines:query.has("selected")?[{lineKey:"p0",itemType:"product",productId:"p0",name:"Caldo de Gallina Tradicional",qty:2,unitPrice:18,note:"",selections:[]}]:[]};
 if(query.has("sheet"))initial.lines=[
  {lineKey:"p0",itemType:"product",productId:"p0",name:"Caldo de Gallina Tradicional",qty:2,unitPrice:18,note:"",selections:[]},
  {lineKey:"p2",itemType:"product",productId:"p2",name:"Jarra de Chicha Morada (1.5 L)",qty:1,unitPrice:12,note:"Sin hielo",selections:[]},
  {lineKey:"combo",itemType:"combo",productId:"combo",name:"Menú criollo",qty:1,unitPrice:25,note:"",selections:[{groupId:"g",groupName:"Segundo",productId:"p1",name:"Lomo saltado",surcharge:0}]}
 ];
 if(query.has("long")){initial.lines=initial.lines.map(line=>({...line,name:"Plato con un nombre especialmente largo para comprobar el espacio de lectura",qty:9999}));initial.notes="Preparar las bebidas sin azúcar y servir después de la entrada.";}
 return renderToStaticMarkup(React.createElement(ComandaView,{initial,mode:"create",allTables:[{id:"t1",name:"04",zone:"Salón principal",seats:4,order:null}],busy:false,close:()=>{},save:()=>{throw new Error("No se registran comandas desde el preview")},notify:()=>{}}));
}
const server=createServer((req,res)=>{
 if(req.url==="/font.woff2"){res.setHeader("content-type","font/woff2");res.end(readFileSync(resolve(root,".next/static/media",font)));return;}
 const url=new URL(req.url,"http://localhost");res.setHeader("content-type","text/html; charset=utf-8");
 res.end(`<!doctype html><html lang="es"><head><meta name="viewport" content="width=device-width, initial-scale=1"><style>${css}@font-face{font-family:Manrope;font-weight:200 800;src:url(/font.woff2)}:root{--font-manrope:Manrope}</style></head><body>${fixture(url.searchParams)}</body></html>`);
});
await new Promise(resolve=>server.listen(0,"127.0.0.1",resolve));
const {chromium}=require(process.env.FOODS_PLAYWRIGHT_PATH??"playwright");let browser;
try{
 browser=await chromium.launch({headless:true,executablePath:process.env.FOODS_BROWSER_PATH});const results=[],defects=[];
 const cases=[["desktop",1440,900,""],["mobile",390,844,""],["selected-mobile",390,844,"?selected"],["selected-desktop",1440,900,"?selected"],["category-mobile",390,844,"?category=cat0"],["category-desktop",1440,900,"?category=cat1"],["combos-mobile",390,844,"?combos"],["short-mobile",390,480,""],["tablet",768,1024,""],["many-categories",390,844,"?many"],["all-categories",390,844,"?categories&many"],["loading",390,844,"?state=loading"],["error",390,844,"?state=error"],["category-error",390,844,"?state=category-error"],["empty",390,844,"?state=empty"],["summary-mobile",390,844,"?sheet"],["summary-short",390,480,"?sheet"],["summary-desktop",1440,900,"?sheet"],["summary-narrow",1024,768,"?sheet"],["summary-long",390,844,"?sheet&long"]];
 for(const [name,width,height,query] of cases){
  const page=await browser.newPage({viewport:{width,height},hasTouch:width<900,reducedMotion:"reduce"});
  await page.goto(`http://127.0.0.1:${server.address().port}/${query}`);await page.evaluate(()=>document.fonts.ready);
  const geometry=await page.evaluate(()=>{
   const rect=el=>{const r=el.getBoundingClientRect();return {x:r.x,y:r.y,right:r.right,bottom:r.bottom,width:r.width,height:r.height};};
   const chips=document.querySelector(".cmd-chips"),combo=document.querySelector(".cmd-chip-combos"),list=document.querySelector(".cmd-list-scroll"),all=document.querySelector(".cmd-category-all");
   const buttons=[...chips.querySelectorAll("button")].map(button=>({name:button.textContent,...rect(button)}));
   return {overflow:document.documentElement.scrollWidth>innerWidth,combo:rect(combo),all:rect(all),chips:rect(chips),buttons,list:rect(list),chipWidth:chips.clientWidth,chipScrollWidth:chips.scrollWidth,bar:rect(document.querySelector(".cmd-bar")),scroll:list.scrollHeight>list.clientHeight,scrollbars:[chips,list,document.querySelector(".cmd-summary-scroll")].map(el=>getComputedStyle(el).scrollbarWidth),prices:[...document.querySelectorAll(".cmd-item-price")].map(el=>({fontWeight:getComputedStyle(el).fontWeight,...rect(el)})),controls:[...document.querySelectorAll(".cmd-item button")].map(el=>rect(el))};
  });
  try{
  const categoryColors=await page.locator(".cmd-chips button").evaluateAll(buttons=>buttons.map(button=>({color:getComputedStyle(button).color,background:getComputedStyle(button).backgroundColor,selected:button.getAttribute("aria-pressed")==="true",mark:!!button.querySelector("svg")})));
  const luminance=value=>value.match(/[\d.]+/g).slice(0,3).map(Number).map(channel=>channel/255).map(channel=>channel<=.04045?channel/12.92:((channel+.055)/1.055)**2.4).reduce((sum,channel,index)=>sum+channel*[.2126,.7152,.0722][index],0);
  for(const chip of categoryColors){const light=[luminance(chip.color),luminance(chip.background)].sort((a,b)=>b-a);assert.ok((light[0]+.05)/(light[1]+.05)>=4.5,`${name}: contraste AA de categoría`);if(chip.selected)assert.equal(chip.mark,true,`${name}: selección no depende solo del color`);}
  assert.equal(geometry.overflow,false,`${name}: sin desborde horizontal`);
  assert.deepEqual(geometry.scrollbars,["none","none",width>900?"thin":"none"],`${name}: carta sin barra; resumen con barra fina solo en desktop`);
  assert.ok(geometry.combo.x>=0&&geometry.combo.right<=width&&geometry.combo.bottom<=height,`${name}: Menús y combos siempre visible`);
  assert.ok(geometry.list.height>=100,`${name}: la carta mantiene espacio para consultar al menos un producto`);
  if(width<900){assert.ok(geometry.bar.bottom<=height,`${name}: barra inferior visible`);assert.ok(geometry.combo.height>=44,`${name}: acceso de 44 px`);geometry.buttons.forEach(button=>assert.ok(button.height>=44,`${name}: categorías táctiles`));const barAction=await page.locator(".cmd-bar-submit").evaluate(el=>({radius:getComputedStyle(el).borderRadius,height:el.getBoundingClientRect().height,clipped:el.scrollWidth>el.clientWidth+1}));assert.ok(parseFloat(barAction.radius)>=barAction.height/2,`${name}: Ver comanda con extremos completamente redondeados`);assert.ok(barAction.height>=48,`${name}: Ver comanda conserva altura de 48 px`);assert.equal(barAction.clipped,false,`${name}: Ver comanda sin texto recortado`);}
  assert.ok(geometry.all.right<=width&&geometry.all.height>=44,`${name}: lista completa siempre accesible`);
  if(geometry.chipScrollWidth>geometry.chipWidth&&!query.includes("loading")){
   await page.locator(".cmd-chips").evaluate(el=>{el.scrollLeft=el.scrollWidth;});
   const last=await page.locator(".cmd-chips button").last().boundingBox();assert.ok(last.x>=geometry.chips.x-1&&last.x+last.width<=geometry.chips.right+1,`${name}: última categoría accesible`);
   const all=await page.locator(".cmd-category-all").boundingBox();assert.equal(all.x,geometry.all.x,`${name}: acceso completo no se desplaza`);
   await page.locator(".cmd-chips").evaluate(el=>{el.scrollLeft=0;});
  }
  for(const control of geometry.controls)assert.ok(control.width>=44&&control.height>=44,`${name}: controles de cantidad táctiles`);
  if(query.includes("categories")){assert.equal(await page.locator(".cmd-category-option").count(),30);await page.locator(".modal-body").evaluate(el=>{el.scrollTop=el.scrollHeight;});const last=await page.locator(".cmd-category-option").last().boundingBox();assert.ok(last.y+last.height<=height,`${name}: última categoría en lista completa`);}
  for(const price of geometry.prices){assert.equal(price.fontWeight,"700");assert.ok(price.right<=width,`${name}: precios legibles`);}
  if(query.includes("sheet")){
   const draftColors=await page.locator(".salon-comanda-draft").evaluate(el=>({color:getComputedStyle(el).color,background:getComputedStyle(el).backgroundColor,border:getComputedStyle(el).borderColor,shadow:getComputedStyle(el).boxShadow}));
   const draftLight=[luminance(draftColors.color),luminance(draftColors.background)].sort((a,b)=>b-a);assert.ok((draftLight[0]+.05)/(draftLight[1]+.05)>=4.5,`${name}: contraste AA de Guardar comanda`);assert.equal(draftColors.background,"rgb(219, 223, 251)",`${name}: fondo índigo de Guardar comanda`);assert.equal(draftColors.shadow,"none",`${name}: acción secundaria sin sombra`);
   const borderLight=[luminance(draftColors.border),luminance(draftColors.background)].sort((a,b)=>b-a);assert.ok((borderLight[0]+.05)/(borderLight[1]+.05)>=3,`${name}: borde del botón distinguible de su fondo`);
   await page.locator(".salon-comanda-draft").hover();const hoverColors=await page.locator(".salon-comanda-draft").evaluate(el=>({color:getComputedStyle(el).color,background:getComputedStyle(el).backgroundColor}));const hoverLight=[luminance(hoverColors.color),luminance(hoverColors.background)].sort((a,b)=>b-a);assert.ok((hoverLight[0]+.05)/(hoverLight[1]+.05)>=4.5,`${name}: texto del botón legible también al hover`);await page.mouse.move(0,0);
   if(width<=900){
    const header=await page.locator(".cmd-shell>.modal-header").evaluate(el=>{const rect=el.getBoundingClientRect(),style=getComputedStyle(el);return {background:style.backgroundColor,title:getComputedStyle(el.querySelector("h2")).color,icon:getComputedStyle(el.querySelector(".modal-title-icon")).color,close:getComputedStyle(el.querySelector(".modal-close")).color,uncovered:!!document.elementFromPoint(rect.x+rect.width/2,rect.y+rect.height/2)?.closest(".modal-header")};});
    assert.equal(header.background,"rgb(57, 70, 184)");assert.equal(header.title,"rgb(255, 255, 255)");assert.equal(header.icon,header.title);assert.equal(header.close,header.title);assert.equal(header.uncovered,true,`${name}: el scrim no oscurece ni bloquea la cabecera`);
    const close=page.getByRole("button",{name:"Volver a la carta",exact:true});
    const control=await close.evaluate(el=>{const style=getComputedStyle(el),icon=el.querySelector("svg").getBoundingClientRect(),rect=el.getBoundingClientRect();return {color:style.color,background:style.backgroundColor,width:rect.width,height:rect.height,iconWidth:icon.width,iconHeight:icon.height};});
    assert.equal(control.width,44);assert.equal(control.height,44);assert.equal(control.iconWidth,24);assert.equal(control.iconHeight,24);
    const light=[luminance(control.color),luminance(control.background)].sort((a,b)=>b-a);assert.ok((light[0]+.05)/(light[1]+.05)>=4.5,`${name}: control para volver con contraste AA`);
    await close.focus();assert.notEqual(await close.evaluate(el=>getComputedStyle(el).outlineStyle),"none",`${name}: foco visible del control para volver`);await close.evaluate(el=>el.blur());
   }
   const summary=await page.locator(".cmd-summary").boundingBox(),footer=await page.locator(".cmd-summary-foot").boundingBox();assert.ok(summary.y>=0&&summary.y+summary.height<=height,`${name}: resumen dentro de pantalla`);assert.ok(footer.y+footer.height<=height,`${name}: envío visible`);
   const data=await page.locator(".cmd-summary").evaluate(el=>({overflow:el.scrollWidth>el.clientWidth,buttons:[...el.querySelectorAll(".ds-stepper-button")].map(button=>({width:button.getBoundingClientRect().width,height:button.getBoundingClientRect().height})),capsules:[...el.querySelectorAll(".ds-stepper")].map(item=>({border:getComputedStyle(item).borderTopWidth,radius:getComputedStyle(item).borderRadius,overflow:getComputedStyle(item).overflow}))}));assert.equal(data.overflow,false,`${name}: resumen sin desborde`);
   data.buttons.forEach(button=>assert.ok(button.width>=44&&button.height>=44,`${name}: cantidades táctiles del resumen`));data.capsules.forEach(item=>{assert.equal(item.border,"0px");assert.equal(item.overflow,"visible");});
   const actions=await page.locator(".cmd-actions .button").evaluateAll(buttons=>buttons.map(button=>({clipped:button.scrollWidth>button.clientWidth+1,radius:getComputedStyle(button).borderRadius,...button.getBoundingClientRect().toJSON()})));actions.forEach(action=>{assert.equal(action.clipped,false,`${name}: texto completo del botón`);assert.ok(action.right<=width,`${name}: acción dentro de pantalla`);if(width<=900){assert.ok(parseFloat(action.radius)>=action.height/2,`${name}: botones móviles con extremos redondeados`);assert.ok(action.height>=44,`${name}: acciones táctiles de 44 px`);}});
  }
  }catch(error){defects.push(error.message);}
  await page.screenshot({path:resolve(out,`${name}.png`),fullPage:true});results.push({name,...geometry});console.log(`${name}: captura y geometría comprobadas`);await page.close();
 }
 writeFileSync(resolve(out,"geometry.json"),JSON.stringify(results,null,2));
 assert.deepEqual(defects,[],"Defectos de geometría");
}finally{await browser?.close();await new Promise(resolve=>server.close(resolve));}
