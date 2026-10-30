"use client";
import {useEffect,useId,useRef,useState} from "react";
import {Button,FormField,Input} from "@/design-system";
import {readImageMetadata} from "@/shared/api/image-upload";
import type {MediaDraft,RestaurantProfileDraft} from "../infrastructure/restaurant-profile-api";
import "./restaurant-profile.css";

function MediaPreview({item,label}:{item:MediaDraft;label:string}){
 const preview=useRef<HTMLImageElement>(null);
 useEffect(()=>{if(!item.file)return;const url=URL.createObjectURL(item.file);if(preview.current)preview.current.src=url;return()=>URL.revokeObjectURL(url);},[item.file]);
 const src=item.image?.url;
 // Storage URLs are signed and change; Next image optimization must not cache them.
 // eslint-disable-next-line @next/next/no-img-element
 return <div className="restaurant-media-preview">{src||item.file?<img ref={preview} src={item.file?undefined:src??undefined} alt={label}/>:<span>Sin vista previa</span>}{item.image?.status==="PROCESSING"&&<small>Optimizando imagen…</small>}{item.image?.status==="FAILED"&&<small role="alert">No se pudo optimizar. Reemplaza esta imagen.</small>}</div>;
}
export function RestaurantProfileFields({value,onChange,maxPages,disabled=false,onBusy}:{value:RestaurantProfileDraft;onChange:(value:RestaurantProfileDraft)=>void;maxPages:number;disabled?:boolean;onBusy?:(busy:boolean)=>void}){
 const id=useId(),[error,setError]=useState(""),[reading,setReading]=useState(false),mounted=useRef(true);
 useEffect(()=>{mounted.current=true;return()=>{mounted.current=false;};},[]);
 const busy=disabled||reading;
 async function select(files:File[],logo=false){
  setError("");setReading(true);onBusy?.(true);
  try{
   if(!logo&&value.pages.length+files.length>maxPages)throw new Error(`Puedes subir hasta ${maxPages} imágenes de la carta.`);
   for(const file of files)await readImageMetadata(file);
   if(mounted.current)onChange(logo?{...value,logo:{file:files[0]}}:{...value,pages:[...value.pages,...files.map(file=>({file}))]});
  }catch(cause){if(mounted.current)setError(cause instanceof Error?cause.message:"No pudimos leer la imagen.");}
  finally{if(mounted.current)setReading(false);onBusy?.(false);}
 }
 function move(index:number,direction:number){const pages=[...value.pages];[pages[index],pages[index+direction]]=[pages[index+direction],pages[index]];onChange({...value,pages});}
 return <div className="restaurant-profile-fields">
  <FormField label="Fecha de fundación" optional help="Se utiliza como referencia para el aniversario del restaurante."><Input type="date" value={value.foundedOn} disabled={busy} onChange={event=>onChange({...value,foundedOn:event.target.value})}/></FormField>
  <section className="restaurant-media-section" aria-labelledby={`${id}-logo`}><h3 id={`${id}-logo`}>Logo del restaurante</h3><p>Se mostrará en tus reportes, tickets y la bienvenida del chatbot.</p>
   <div className="restaurant-logo-editor">{value.logo&&<MediaPreview item={value.logo} label="Logo del restaurante"/>}<div><FormField label={value.logo?"Reemplazar logo":"Subir logo"}><Input type="file" accept="image/jpeg,image/png,image/webp" disabled={busy} onChange={event=>{const files=Array.from(event.target.files??[]);event.target.value="";if(files.length)void select(files,true);}}/></FormField>{value.logo&&!disabled&&<Button kind="ghost" disabled={busy} onClick={()=>onChange({...value,logo:null})}>Quitar logo</Button>}</div></div>
  </section>
  <section className="restaurant-media-section" aria-labelledby={`${id}-menu`}><h3 id={`${id}-menu`}>Carta virtual</h3><p>Sube las páginas de tu carta en el orden en que las verá el cliente en el chatbot. Los precios y la disponibilidad del pedido se validan con tus productos.</p>
   {!value.pages.length&&<p className="restaurant-media-empty">Aún no has añadido imágenes de tu carta.</p>}
   <ol className="restaurant-menu-pages">{value.pages.map((page,index)=><li key={page.image?.id??`${page.file?.name}-${index}`}><MediaPreview item={page} label={`Carta virtual, página ${index+1}`}/><div className="restaurant-page-actions"><b>Página {index+1}</b>{!disabled&&<div><Button kind="ghost" disabled={busy||index===0} aria-label={`Subir página ${index+1}`} onClick={()=>move(index,-1)}>Subir</Button><Button kind="ghost" disabled={busy||index===value.pages.length-1} aria-label={`Bajar página ${index+1}`} onClick={()=>move(index,1)}>Bajar</Button><Button kind="ghost" disabled={busy} aria-label={`Quitar página ${index+1}`} onClick={()=>onChange({...value,pages:value.pages.filter((_,i)=>i!==index)})}>Quitar</Button></div>}</div></li>)}</ol>
   {!disabled&&<FormField label="Añadir imágenes de la carta" help={`JPEG, PNG o WebP · máximo 5 MB y 4000 × 4000 px por imagen${maxPages?` · hasta ${maxPages} páginas`:""}.`}><Input type="file" multiple accept="image/jpeg,image/png,image/webp" disabled={busy||!maxPages||value.pages.length>=maxPages} onChange={event=>{const files=Array.from(event.target.files??[]);event.target.value="";if(files.length)void select(files);}}/></FormField>}
  </section>{reading&&<p role="status">Validando imágenes…</p>}{error&&<p className="field-error" role="alert">{error}</p>}
 </div>;
}
