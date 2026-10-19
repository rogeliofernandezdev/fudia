// Mide el JS y CSS de primera carga (gzip) por ruta contra un build de producción.
// Next 16 ya no imprime «First Load JS» en `next build`, así que se lee el HTML
// servido: etiquetas <script>/<link> más los chunks que el payload RSC inline
// referencia. Los polyfills `nomodule` se excluyen (los navegadores modernos no
// los descargan).
//
// Uso:
//   npm run build
//   npx next start --hostname localhost --port 5181
//   node scripts/measure-first-load.mjs http://localhost:5181 [json]
import {existsSync,readFileSync} from "node:fs";
import {join} from "node:path";
import {gzipSync} from "node:zlib";

const [base="http://localhost:5181",mode]=process.argv.slice(2);
const next=join(process.cwd(),".next");
const routes=["/login","/dashboard","/pos","/dining-room","/kitchen","/orders","/combos","/products","/recipes","/cash","/sales","/purchases","/inventory","/customers","/settings/organization","/settings/users","/tables","/reservations","/platform/onboarding","/table/demo"];
const sizes=new Map();
const gzip=file=>{if(!sizes.has(file)){const path=join(next,file);sizes.set(file,existsSync(path)?gzipSync(readFileSync(path)).length:0)}return sizes.get(file)};
const unique=(html,pattern)=>[...new Set([...html.matchAll(pattern)].map(match=>decodeURIComponent(match[0])))];

const report={};
for(const route of routes){
  const response=await fetch(base+route,{redirect:"manual"});
  const html=await response.text();
  const js=unique(html,/static\/chunks\/[A-Za-z0-9_\-.%()[\]/]+?\.js/g).filter(file=>!file.includes("polyfills"));
  const css=unique(html,/static\/css\/[A-Za-z0-9_\-.]+?\.css/g);
  report[route]={status:response.status,js:js.reduce((total,file)=>total+gzip(file),0),css:css.reduce((total,file)=>total+gzip(file),0)};
}
if(mode==="json")console.log(JSON.stringify(report));
else{const kb=bytes=>(bytes/1024).toFixed(1).padStart(7);console.log("estado    JS kB   CSS kB  ruta");for(const [route,{status,js,css}] of Object.entries(report))console.log(`${status}    ${kb(js)}  ${kb(css)}  ${route}`)}
