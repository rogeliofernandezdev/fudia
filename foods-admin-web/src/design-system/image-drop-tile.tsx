"use client";
/* eslint-disable @next/next/no-img-element -- Signed URLs and immediate blob previews bypass image optimization. */
import {useEffect,useRef,useState,type ReactNode} from "react";
import {Icon} from "./icons";
import "./styles/image-drop-tile.css";

/** Vista inmediata de un archivo local; libera la URL temporal al desmontarse o cambiar. */
export function FileImage({file,alt}:{file:File;alt:string}){
 const image=useRef<HTMLImageElement>(null);
 useEffect(()=>{const url=URL.createObjectURL(file);if(image.current)image.current.src=url;return()=>URL.revokeObjectURL(url);},[file]);
 return <img ref={image} alt={alt} decoding="async"/>;
}

/**
 * X circular montada sobre la esquina superior derecha de una imagen: única
 * forma de quitarla. Su contenedor debe tener `position: relative` y no recortar
 * el desborde.
 */
export function ImageRemoveButton({label,disabled=false,onClick}:{label:string;disabled?:boolean;onClick:()=>void}){
 return <button type="button" className="image-remove" aria-label={label} title={label} disabled={disabled} onClick={onClick}><Icon name="close" size={14}/></button>;
}

/**
 * Recuadro para elegir o arrastrar imágenes. Sin imagen muestra la invitación
 * («Arrastra…» con ratón, «Toca…» en pantallas táctiles) y la regla de archivos.
 * Con imagen, tocarla la reemplaza y, si hay `onRemove`, una X en la esquina la
 * quita. No se agregan botones «Cambiar» ni papelera aparte.
 */
export function ImageDropTile({src,file,alt="",label,dropText,tapText,hint,fit="cover",accept="image/jpeg,image/png,image/webp",multiple=false,disabled=false,className="",removeLabel="Quitar imagen",children,onFiles,onRemove}:{src?:string|null;/** Archivo local recién elegido; tiene prioridad sobre `src`. */file?:File|null;/** Texto alternativo de la imagen presentada. */alt?:string;/** Nombre accesible del botón. */label:string;dropText:string;tapText:string;hint?:ReactNode;fit?:"cover"|"contain";accept?:string;multiple?:boolean;disabled?:boolean;className?:string;removeLabel?:string;/** Contenido superpuesto, como un estado de procesamiento. */children?:ReactNode;onFiles:(files:File[])=>void;onRemove?:()=>void}){
 const[dragging,setDragging]=useState(false);
 const input=useRef<HTMLInputElement>(null);
 const filled=Boolean(file||src);
 const take=(list:FileList|null|undefined)=>{const files=Array.from(list??[]);if(files.length&&!disabled)onFiles(multiple?files:files.slice(0,1));};
 return <div className={["image-drop",className].filter(Boolean).join(" ")}>
  <input ref={input} type="file" accept={accept} multiple={multiple} hidden disabled={disabled} onChange={event=>{const files=event.target.files;take(files);event.target.value="";}}/>
  <button type="button" className={["image-drop-tile",filled?"has-image":"",dragging?"is-dragging":"",fit==="contain"?"fit-contain":""].filter(Boolean).join(" ")} disabled={disabled} aria-label={label} onClick={()=>input.current?.click()} onDragOver={event=>{if(disabled||!event.dataTransfer.types.includes("Files"))return;event.preventDefault();event.dataTransfer.dropEffect="copy";setDragging(true)}} onDragLeave={event=>{if(!event.currentTarget.contains(event.relatedTarget as Node|null))setDragging(false)}} onDrop={event=>{event.preventDefault();setDragging(false);take(event.dataTransfer.files)}}>
   {file?<FileImage file={file} alt={alt}/>:src?<img src={src} alt={alt} decoding="async"/>:<span className="image-drop-empty"><Icon name="upload" size={20}/><b><span className="image-drop-mouse">{dropText}</span><span className="image-drop-touch">{tapText}</span></b>{hint&&<small>{hint}</small>}</span>}
   {children}
  </button>
  {filled&&onRemove&&!disabled&&<ImageRemoveButton label={removeLabel} onClick={onRemove}/>}
 </div>;
}
