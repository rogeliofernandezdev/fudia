// Interactive production components, illustrative catalog, no persisted writes.
import {readFileSync,readdirSync,existsSync,mkdirSync,writeFileSync} from "node:fs";
import {createRequire} from "node:module";
import {dirname,resolve} from "node:path";
import {fileURLToPath} from "node:url";
import {createServer} from "node:http";
import {build} from "esbuild";
import assert from "node:assert/strict";

const require=createRequire(import.meta.url),root=resolve(dirname(fileURLToPath(import.meta.url)),"../..");
const output=resolve(process.argv[2]??resolve(root,".next/delivery-coverage-preview"));mkdirSync(output,{recursive:true});
const bundle=await build({absWorkingDir:root,bundle:true,write:false,outfile:"preview.js",format:"iife",jsx:"automatic",define:{"process.env.NODE_ENV":'"production"',"process.env":"{}"},stdin:{resolveDir:root,loader:"tsx",contents:`
 import React,{useState} from 'react';
 import {createRoot} from 'react-dom/client';
 import {QueryClient,QueryClientProvider} from '@tanstack/react-query';
 import {DeliveryCoverageFields} from './src/modules/platform/presentation/delivery-coverage-fields';
 import {Dialog} from './src/design-system/dialog';
 import {ModalHeader,ModalForm,ModalBody,ModalFooter} from './src/design-system/modal';
 import {Button,Input} from './src/design-system/page-header';
 import {FormField} from './src/design-system/form-field';
 function Preview(){
  const[codes,setCodes]=useState(['150101']),[labels,setLabels]=useState(['LIMA · LIMA · LIMA']),[saved,setSaved]=useState(false);
  return <div className="modal-backdrop"><Dialog className="crud-modal platform-partner-modal" role="dialog" aria-modal="true" aria-labelledby="preview-title">
   <ModalHeader title="Editar central" titleId="preview-title" icon="motorbike" onClose={()=>{}}/>
   <ModalForm onSubmit={event=>{event.preventDefault();setSaved(true)}}><ModalBody><div className="form-sections-flow">
    <div className="form-grid platform-partner-fields"><FormField label="Nombre" className="span-2"><Input defaultValue="Central de reparto de Lima Metropolitana" required/></FormField><FormField label="WhatsApp" className="span-2"><Input defaultValue="51 987 654 321" required/></FormField></div>
    <DeliveryCoverageFields codes={codes} labels={labels} disabled={false} onChange={(next,names)=>{setCodes(next);setLabels(names)}}/>
    {saved&&<p role="status">Cobertura guardada en la vista de prueba.</p>}
   </div></ModalBody><ModalFooter><Button action="cancel" onClick={()=>{}}>Cancelar</Button><Button type="submit" disabled={!codes.length}>Guardar</Button></ModalFooter></ModalForm>
  </Dialog></div>;
 }
 createRoot(document.getElementById('root')).render(<QueryClientProvider client={new QueryClient({defaultOptions:{queries:{retry:false}}})}><Preview/></QueryClientProvider>);
 `},plugins:[{name:"production-facades",setup(builder){
 builder.onResolve({filter:/^@\//},args=>{
  if(args.path==="@/design-system")return {path:"design",namespace:"facade"};
  if(args.path==="@/modules/organizations")return {path:"geography",namespace:"facade"};
  const base=resolve(root,"src",args.path.slice(2));const path=[base,`${base}.tsx`,`${base}.ts`,resolve(base,"index.ts")].find(file=>existsSync(file)&&/\.(ts|tsx)$/.test(file));
  assert.ok(path,args.path);return {path};
 });
 builder.onLoad({filter:/.*/,namespace:"facade"},args=>({resolveDir:root,loader:"ts",contents:args.path==="geography"?"export {listGeographicAreas} from './src/modules/organizations/infrastructure/geography-api'":
 "export * from './src/design-system/page-header';export * from './src/design-system/form-field';export * from './src/design-system/form-section';export * from './src/design-system/states';export * from './src/design-system/skeleton'"}));
 }}]});
const script=bundle.outputFiles.find(file=>file.path.endsWith(".js")).text;
const extraCss=bundle.outputFiles.find(file=>file.path.endsWith(".css"))?.text??"";
const css=readdirSync(resolve(root,".next/static/css")).filter(name=>name.endsWith(".css")).map(name=>readFileSync(resolve(root,".next/static/css",name),"utf8")).join("\n")+"\n"+extraCss+"\n"+readFileSync(resolve(root,"src/modules/platform/presentation/platform-delivery-partners.css"),"utf8");
const catalog={"": [{code:"15",parentCode:"",name:"LIMA",label:"LIMA"}],"15":[{code:"1501",parentCode:"15",name:"LIMA",label:"LIMA · LIMA"}],"1501":[{code:"150101",parentCode:"1501",name:"LIMA",label:"LIMA · LIMA · LIMA"},{code:"150122",parentCode:"1501",name:"MIRAFLORES",label:"MIRAFLORES · LIMA · LIMA"}]};
const server=createServer((req,res)=>{
 const url=new URL(req.url,"http://localhost");
 if(url.pathname.startsWith("/api/admin/geographic-areas")){res.setHeader("content-type","application/json");res.end(JSON.stringify({items:catalog[url.searchParams.get("parentCode")??""]??[]}));return;}
 if(url.pathname==="/preview.js"){res.setHeader("content-type","text/javascript");res.end(script);return;}
 if(url.pathname.startsWith("/_next/static/media/")){res.end(readFileSync(resolve(root,".next/static/media",url.pathname.split("/").at(-1))));return;}
 res.setHeader("content-type","text/html; charset=utf-8");res.end(`<!doctype html><html lang="es"><head><meta name="viewport" content="width=device-width,initial-scale=1"><style>${css}</style></head><body><div id="root"></div><script src="/preview.js"></script></body></html>`);
});
await new Promise(done=>server.listen(0,"127.0.0.1",done));
const {chromium}=require(process.env.FOODS_PLAYWRIGHT_PATH??"playwright");let browser;
try{
 browser=await chromium.launch({executablePath:process.env.FOODS_BROWSER_PATH,headless:true});const results=[];
 for(const [name,width,height] of [["mobile",390,844],["desktop",1440,900],["short",390,480]]){
  const context=await browser.newContext({viewport:{width,height},hasTouch:width===390,reducedMotion:"reduce"});const page=await context.newPage();
  const errors=[];page.on("pageerror",error=>errors.push(error.message));
  page.on("console",message=>{if(message.type()==="error")errors.push(message.text())});
  await page.goto(`http://127.0.0.1:${server.address().port}`);
  try{await page.getByLabel("Departamento",{exact:true}).selectOption("15");}catch(error){await page.screenshot({path:resolve(output,"debug.png")});console.log(JSON.stringify({errors,body:await page.locator("body").innerText()}));throw error;}
  await page.getByLabel("Provincia",{exact:true}).selectOption("1501");
  await page.getByRole("checkbox",{name:"MIRAFLORES",exact:true}).check();
  await page.getByRole("checkbox",{name:"LIMA",exact:true}).uncheck();
  await page.getByText("1 distrito seleccionado",{exact:true}).waitFor();
  await page.getByLabel("Provincia",{exact:true}).selectOption("");
  // Saving with an empty zone picker must work when saved coverage exists.
  await page.getByRole("button",{name:"Guardar",exact:true}).click();
  await page.getByRole("status").filter({hasText:"Cobertura guardada"}).waitFor();
  await page.getByLabel("Provincia",{exact:true}).selectOption("1501");
  assert.equal(await page.getByRole("checkbox",{name:"MIRAFLORES",exact:true}).isChecked(),true);
  const geometry=await page.evaluate(()=>({overflow:document.documentElement.scrollWidth>innerWidth,footer:document.querySelector('.modal-footer').getBoundingClientRect().bottom,controls:[...document.querySelectorAll('button,select,input:not([type="checkbox"]),.partner-district-option')].filter(el=>el.getBoundingClientRect().height>0).map(el=>({name:el.getAttribute('aria-label')||el.textContent,height:(el.closest(".ds-input-shell")??el).getBoundingClientRect().height}))}));
  assert.equal(geometry.overflow,false,name);assert.ok(geometry.footer<=height+1,`${name}: footer visible`);
  if(width===390)assert.ok(geometry.controls.every(control=>control.height>=44),JSON.stringify(geometry.controls));
  assert.deepEqual(errors,[]);await page.screenshot({path:resolve(output,`${name}.png`),fullPage:true});results.push({name,width,height,...geometry});await context.close();
 }
 writeFileSync(resolve(output,"results.json"),JSON.stringify(results,null,2));console.log(JSON.stringify(results));
}finally{await browser?.close();await new Promise(done=>server.close(done));}
