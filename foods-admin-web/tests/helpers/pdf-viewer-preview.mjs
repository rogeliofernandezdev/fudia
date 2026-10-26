// Isolated interactive harness using the ACTUAL React component and Mozilla viewer.
// Start: node tests/helpers/pdf-viewer-preview.mjs (no API, login or business writes).
import {build} from "esbuild";
import {readFile,mkdir} from "node:fs/promises";
import {createServer} from "node:http";
import {resolve,extname,sep} from "node:path";

const root=process.cwd(),output=resolve(root,".cache/pdf-viewer-qa");await mkdir(output,{recursive:true});
await build({entryPoints:[resolve(root,"tests/helpers/pdf-viewer-preview.jsx")],bundle:true,outfile:resolve(output,"bundle.js"),platform:"browser",jsx:"automatic",conditions:["style"],define:{"process.env.NODE_ENV":'"development"',"process.env":"{}"},alias:{"@":resolve(root,"src")},plugins:[{name:"test-design-system",setup(builder){builder.onResolve({filter:/^@\/design-system$/},()=>({path:resolve(root,"tests/helpers/pdf-viewer-design-system.ts")}));}}],loader:{".woff2":"file"},logLevel:"warning"});
if(process.argv.includes("--build-only"))process.exit(0);
const types={".js":"text/javascript",".mjs":"text/javascript",".css":"text/css",".html":"text/html",".json":"application/json",".ftl":"text/plain",".svg":"image/svg+xml",".png":"image/png",".wasm":"application/wasm",".bcmap":"application/octet-stream",".pfb":"application/octet-stream",".ttf":"font/ttf"};
const server=createServer(async(req,res)=>{
 try{
  const path=new URL(req.url,"http://localhost").pathname;
  if(path==="/"){res.setHeader("content-type","text/html; charset=utf-8");res.end('<!doctype html><html lang="es" style="--font-jakarta:Arial"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1,viewport-fit=cover"><title>PdfViewer · prueba aislada</title><link rel="stylesheet" href="/bundle.css"></head><body><div id="root"></div><script src="/bundle.js"></script></body></html>');return;}
  const base=path.startsWith("/bundle")?output:resolve(root,"public");
  const file=resolve(base,"."+path);if(!file.startsWith(base+sep))throw new Error("invalid path");
  res.setHeader("content-type",types[extname(file)]??"application/octet-stream");res.end(await readFile(file));
 }catch{res.statusCode=404;res.end("Not found");}
});
server.listen(5194,"localhost",()=>console.log("PdfViewer QA: http://localhost:5194 (datos ficticios, sin API)"));
