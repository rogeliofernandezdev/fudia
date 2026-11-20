import {cp,mkdir,readFile,writeFile} from "node:fs/promises";
import {createHash} from "node:crypto";
import {transform} from "esbuild";
import {fileURLToPath} from "node:url";
import {join} from "node:path";
import {createElement} from "react";
import {renderToStaticMarkup} from "react-dom/server";
import {IconPrinter,IconDownload} from "@tabler/icons-react";

const root=fileURLToPath(new URL("../",import.meta.url));
const vendor=join(root,"vendor/pdfjs"),destination=join(root,"public/pdfjs");
const manifest=JSON.parse(await readFile(join(vendor,"manifest.json"),"utf8"));
const library=join(root,"node_modules/pdfjs-dist");
const installed=JSON.parse(await readFile(join(library,"package.json"),"utf8"));
const upstream=await readFile(join(vendor,"web/viewer.mjs"),"utf8");
if(installed.version!==manifest.version||!upstream.includes(`pdfjsVersion = ${manifest.version}`)){
 throw new Error("PDF.js: el visor, pdfjs-dist y el worker deben tener exactamente la misma versión.");
}
await mkdir(destination,{recursive:true});
await cp(join(vendor,"web"),join(destination,"web"),{recursive:true});
await cp(join(library,"legacy/build"),join(destination,"build"),{recursive:true,filter:path=>!path.endsWith(".map")});
for(const resource of ["cmaps","standard_fonts","wasm","iccs"]){
 await cp(join(library,resource),join(destination,"web",resource),{recursive:true});
}
await cp(join(vendor,"LICENSE"),join(destination,"LICENSE"));
await cp(join(root,"src/styles/variables.css"),join(destination,"fudia-tokens.css"));
await cp(join(root,"src/shared/pdf-viewer/runtime/fudia-bridge.mjs"),join(destination,"fudia-bridge.mjs"));
await cp(join(root,"src/design-system/styles/pdfjs-viewer.css"),join(destination,"fudia-viewer.css"));
// Cada script y hoja del visor lleva ?v=<versión o huella>: next.config los sirve
// con caché inmutable de un año y un despliegue nuevo cambia la URL.
const v=`?v=${installed.version}`;
const own={};
for(const file of ["fudia-tokens.css","fudia-viewer.css","fudia-bridge.mjs"]){
 own[file]=`?v=${createHash("sha256").update(await readFile(join(destination,file))).digest("hex").slice(0,12)}`;
}
// El visor carga las versiones minificadas del motor, del worker y del sandbox
// (≈ la mitad de bytes) y su propio viewer.mjs minificado con esbuild.
const replaceOnce=(text,from,to)=>{
 if(!text.includes(from))throw new Error(`PDF.js: no se encontró «${from}»; revisa la actualización del visor.`);
 return text.replace(from,to);
};
let viewer=replaceOnce(upstream,'value: "../build/pdf.worker.mjs"',`value: "../build/pdf.worker.min.mjs${v}"`);
viewer=replaceOnce(viewer,'value: "../build/pdf.sandbox.mjs"',`value: "../build/pdf.sandbox.min.mjs${v}"`);
const minified=await transform(viewer,{loader:"js",format:"esm",minify:true,legalComments:"inline",target:"es2022"});
await writeFile(join(destination,"web/viewer.mjs"),minified.code);
// The isolated viewer uses the same Tabler outline family as FudIA, without a React runtime.
const icons=join(destination,"fudia-icons");
await mkdir(icons,{recursive:true});
for(const [name,Icon] of [["printer",IconPrinter],["download",IconDownload]]){
 await writeFile(join(icons,`${name}.svg`),renderToStaticMarkup(createElement(Icon,{size:24,stroke:1.75,color:"black"})));
}
const html=(await readFile(join(vendor,"web/viewer.html"),"utf8"))
 .replace('initial-scale=1, maximum-scale=1','initial-scale=1, viewport-fit=cover')
 .replace('<html dir="ltr"','<html lang="es" dir="ltr"')
 .replace('<title>PDF.js viewer</title>','<title>Visor PDF · FudIA</title>')
 .replace("connect-src * blob: data:","connect-src 'self' blob: data:")
 .replace('<script src="../build/pdf.mjs" type="module"></script>',`<script src="../build/pdf.min.mjs${v}" type="module"></script>`)
 .replace('<link rel="stylesheet" href="viewer.css" />',`<link rel="stylesheet" href="viewer.css${v}" />`)
 .replace('<script src="viewer.mjs" type="module"></script>',`<link rel="stylesheet" href="../fudia-tokens.css${own["fudia-tokens.css"]}" />\n<link rel="stylesheet" href="../fudia-viewer.css${own["fudia-viewer.css"]}" />\n<script src="../fudia-bridge.mjs${own["fudia-bridge.mjs"]}" type="module"></script>\n<script src="viewer.mjs${v}" type="module"></script>`);
for(const asset of [`../build/pdf.min.mjs${v}`,`viewer.css${v}`,`viewer.mjs${v}`]){
 if(!html.includes(asset))throw new Error(`PDF.js: viewer.html no referencia ${asset}; revisa la actualización del visor.`);
}
await writeFile(join(destination,"web/viewer.html"),html);
await writeFile(join(destination,"version.json"),JSON.stringify({version:installed.version}));
console.log(`PDF.js ${installed.version}: visor completo y worker legacy preparados en public/pdfjs.`);
