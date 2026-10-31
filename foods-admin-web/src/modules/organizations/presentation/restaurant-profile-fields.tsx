"use client";
import {useEffect,useRef,useState} from "react";
import {FormField,FormSection,IconButton,Input,Status} from "@/design-system";
import {FileImage,ImageDropTile,ImageRemoveButton} from "@/design-system/image-drop-tile";
import {readImageMetadata} from "@/shared/api/image-upload";
import type {MediaDraft,RestaurantProfileDraft} from "../infrastructure/restaurant-profile-api";
import "./restaurant-profile.css";

const imageHint="JPEG, PNG o WebP · 5 MB";

/** Valida los archivos elegidos antes de aceptarlos y expone el error del campo. */
function useImageSelection(onBusy?:(busy:boolean)=>void){
 const[error,setError]=useState(""),[reading,setReading]=useState(false),mounted=useRef(true);
 useEffect(()=>{mounted.current=true;return()=>{mounted.current=false;};},[]);
 async function read(files:File[],limit?:{count:number;max:number}){
  setError("");setReading(true);onBusy?.(true);
  try{
   if(limit&&limit.count+files.length>limit.max)throw new Error(`Puedes subir hasta ${limit.max} imágenes de la carta.`);
   for(const file of files)await readImageMetadata(file);
   return true;
  }catch(cause){if(mounted.current)setError(cause instanceof Error?cause.message:"No pudimos leer la imagen.");return false;}
  finally{if(mounted.current)setReading(false);onBusy?.(false);}
 }
 return{error,reading,read};
}

function MediaStatus({item}:{item:MediaDraft}){
 if(item.file)return null;
 if(item.image?.status==="PROCESSING")return <span className="restaurant-media-status"><Status tone="orange">Optimizando</Status></span>;
 if(item.image?.status==="FAILED")return <span className="restaurant-media-status" role="alert"><Status tone="red">No se pudo optimizar</Status></span>;
 return null;
}

export function RestaurantLogoField({value,onChange,disabled=false,onBusy}:{value:MediaDraft|null;onChange:(value:MediaDraft|null)=>void;disabled?:boolean;onBusy?:(busy:boolean)=>void}){
 const{error,reading,read}=useImageSelection(onBusy),busy=disabled||reading;
 async function choose(files:File[]){if(await read(files))onChange({file:files[0]});}
 return <div className="restaurant-logo">
  <ImageDropTile src={value?.image?.url} file={value?.file} alt="Logo del restaurante" fit="contain" disabled={busy} label={value?"Cambiar logo":"Subir logo"} dropText="Subir logo" tapText="Subir logo" hint={imageHint} removeLabel="Quitar logo" onFiles={files=>void choose(files)} onRemove={()=>onChange(null)}>{value&&<MediaStatus item={value}/>}</ImageDropTile>
  {error&&<p className="field-error" role="alert">{error}</p>}
 </div>;
}

function MenuPage({item,index,total,disabled,move,remove}:{item:MediaDraft;index:number;total:number;disabled:boolean;move:(direction:number)=>void;remove:()=>void}){
 const page=index+1,alt=`Carta virtual, página ${page}`,src=item.image?.url;
 // Storage URLs are signed and change; Next image optimization must not cache them.
 // eslint-disable-next-line @next/next/no-img-element
 const preview=item.file?<FileImage file={item.file} alt={alt}/>:src?<img src={src} alt={alt} decoding="async"/>:<span className="restaurant-page-placeholder">Sin vista previa</span>;
 return <li className="restaurant-page">
  <figure>{preview}<figcaption aria-label={`Página ${page}`}>{page}</figcaption><MediaStatus item={item}/></figure>
  {!disabled&&<><ImageRemoveButton label={`Quitar la página ${page}`} onClick={remove}/><div className="restaurant-page-bar"><IconButton icon="chevronLeft" label="Mover antes" accessibleLabel={`Mover la página ${page} antes`} disabled={index===0} onClick={()=>move(-1)}/><IconButton icon="chevron" label="Mover después" accessibleLabel={`Mover la página ${page} después`} disabled={index===total-1} onClick={()=>move(1)}/></div></>}
 </li>;
}

export function VirtualMenuField({pages,onChange,maxPages,disabled=false,onBusy}:{pages:MediaDraft[];onChange:(pages:MediaDraft[])=>void;maxPages:number;disabled?:boolean;onBusy?:(busy:boolean)=>void}){
 const{error,reading,read}=useImageSelection(onBusy),busy=disabled||reading,full=!maxPages||pages.length>=maxPages;
 async function add(files:File[]){if(await read(files,{count:pages.length,max:maxPages}))onChange([...pages,...files.map(file=>({file}))]);}
 function move(index:number,direction:number){const next=[...pages];[next[index],next[index+direction]]=[next[index+direction],next[index]];onChange(next);}
 return <div className="restaurant-menu">
  <ol className="restaurant-menu-pages" aria-label="Páginas de la carta virtual">
   {pages.map((page,index)=><MenuPage key={page.image?.id??`${page.file?.name}-${index}`} item={page} index={index} total={pages.length} disabled={busy} move={direction=>move(index,direction)} remove={()=>onChange(pages.filter((_,i)=>i!==index))}/>)}
   {!disabled&&!full&&<li className="restaurant-page-add"><ImageDropTile multiple disabled={busy} label="Añadir páginas de la carta" dropText="Añadir páginas" tapText="Añadir páginas" hint={imageHint} onFiles={files=>void add(files)}/></li>}
  </ol>
  {disabled&&!pages.length&&<p className="restaurant-menu-copy">Sin páginas.</p>}
  {error&&<p className="field-error" role="alert">{error}</p>}
 </div>;
}

/** Identidad completa para el registro asistido de Plataforma. */
export function RestaurantProfileFields({value,onChange,maxPages,disabled=false,onBusy}:{value:RestaurantProfileDraft;onChange:(value:RestaurantProfileDraft)=>void;maxPages:number;disabled?:boolean;onBusy?:(busy:boolean)=>void}){
 return <div className="restaurant-profile-fields">
  <FormField label="Fecha de fundación" optional><Input type="date" value={value.foundedOn} disabled={disabled} onChange={event=>onChange({...value,foundedOn:event.target.value})}/></FormField>
  <FormSection icon="star" tone="peach" title="Logo" bare><RestaurantLogoField value={value.logo} onChange={logo=>onChange({...value,logo})} disabled={disabled} onBusy={onBusy}/></FormSection>
  <FormSection icon="utensils" tone="digital" title="Carta virtual" bare><VirtualMenuField pages={value.pages} onChange={pages=>onChange({...value,pages})} maxPages={maxPages} disabled={disabled} onBusy={onBusy}/></FormSection>
 </div>;
}
