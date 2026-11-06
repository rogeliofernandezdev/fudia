// Interactive verification of the real tooltip layer, buttons and Dialog. No API or writes.
// From foods-admin-web: node tests/helpers/tooltip-preview.mjs
import {build} from "esbuild";
import {readFile,mkdir} from "node:fs/promises";
import {resolve} from "node:path";
import {createServer} from "node:http";

const root=process.cwd(),output=resolve(root,".cache/tooltip-preview");
await mkdir(output,{recursive:true});
await build({stdin:{resolveDir:root,loader:"jsx",contents:`
import React,{useState} from 'react';import {createRoot} from 'react-dom/client';
import {IconButton,FieldLabel} from './src/design-system/page-header';
import {TooltipLayer} from './src/design-system/tooltip-layer';
import {Tooltip} from './src/design-system/tooltip';
import {Dialog} from './src/design-system/dialog';
function Preview(){const[open,setOpen]=useState(true),[extra,setExtra]=useState(false);return <>
<h1>Tooltips · comprobación aislada</h1><p>Datos ilustrativos, sin operaciones del restaurante.</p>
<div className="clip-panel"><IconButton icon="eye" label="Ver detalle del registro dentro de un panel que recorta"/><IconButton icon="close" className="danger" label="Quitar la opción junto al borde de la pantalla"/></div>
<button onClick={()=>setOpen(true)}>Abrir modal</button><button onClick={()=>setExtra(!extra)}>Alternar acción dinámica</button>
{extra&&<IconButton icon="edit" label="Editar acción incorporada después de cargar la pantalla"/>}
<Tooltip label="Ver contenido" content="Descripción de varias líneas que también debe quedar delante de las otras superficies."><span>Descripción</span></Tooltip>
{open&&<div className="modal-backdrop"><Dialog className="crud-modal preview-modal"><header><h2>Opciones al pedir</h2><button className="button ghost" onClick={()=>setOpen(false)}>Cerrar prueba</button></header>
<div className="modal-body preview-body"><div className="preview-actions"><FieldLabel hint="Esta ayuda también debe salir del cuerpo desplazable del modal.">Grupo</FieldLabel><IconButton icon="trash" className="danger" label="Quitar el grupo Término de la carne"/></div>
{Array.from({length:8},(_,i)=><div className="preview-actions" key={i}><span>Opción {i+1}</span><IconButton icon="close" className="danger" label={'Quitar la opción Jugoso con una etiqueta suficientemente larga para comprobar el borde derecho '+i}/></div>)}
</div></Dialog></div>}<TooltipLayer/></>};createRoot(document.getElementById('root')).render(<Preview/>);`},bundle:true,outfile:resolve(output,"bundle.js"),platform:"browser",jsx:"automatic",alias:{"@":resolve(root,"src")},define:{"process.env.NODE_ENV":'"development"'},plugins:[{name:"preview-link",setup(builder){builder.onResolve({filter:/^next\/link$/},()=>({path:"link",namespace:"preview"}));builder.onLoad({filter:/.*/,namespace:"preview"},()=>({contents:"export default function Link(){return null}",loader:"js"}));}}],logLevel:"warning"});
const cssFiles=["src/styles/variables.css","src/styles/globals.css","src/design-system/styles/modal.css","src/design-system/styles/table-actions.css","src/design-system/styles/tooltip.css"];
const css=(await Promise.all(cssFiles.map(file=>readFile(resolve(root,file),"utf8")))).join('\n').replace(/^@import[^;]+;/gm,'');
const server=createServer(async(req,res)=>{
 if(req.url==='/bundle.js'){res.setHeader('content-type','text/javascript');res.end(await readFile(resolve(output,'bundle.js')));return;}
 res.setHeader('content-type','text/html; charset=utf-8');res.end(`<!doctype html><html lang="es"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Tooltips · comprobación</title><style>${css}body{padding:24px}.clip-panel{display:flex;justify-content:space-between;max-width:480px;overflow:hidden;padding:8px;border:1px solid var(--line)}.preview-modal{width:min(600px,100%);padding:0}.preview-modal>header{display:flex;justify-content:space-between;align-items:center;padding:16px;background:var(--navigation-900);color:var(--surface)}.preview-modal h2{margin:0;font-size:18px}.preview-body{height:280px;overflow:auto}.preview-actions{display:flex;justify-content:space-between;align-items:center;gap:16px;padding:8px 0}.preview-modal{overflow:hidden}</style></head><body><div id="root"></div><script src="/bundle.js"></script></body></html>`);
});
server.listen(5197,'localhost',()=>console.log('Tooltip QA: http://localhost:5197 (sin API ni escrituras)'));
