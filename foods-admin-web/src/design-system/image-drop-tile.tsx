"use client";
/* eslint-disable @next/next/no-img-element -- Signed URLs and immediate blob previews bypass image optimization. */
import {useEffect,useRef,useState,type ReactNode,type RefObject} from "react";
import {Icon} from "./icons";
import "./styles/image-drop-tile.css";

/** Vista inmediata de un archivo local; libera la URL temporal al desmontarse o cambiar. */
export function FileImage({file,alt}:{file:File;alt:string}){
 const image=useRef<HTMLImageElement>(null);
 useEffect(()=>{const url=URL.createObjectURL(file);if(image.current)image.current.src=url;return()=>URL.revokeObjectURL(url);},[file]);
 return <img ref={image} alt={alt} decoding="async"/>;
}

/**
 * Recuadro para elegir o arrastrar imágenes. Sin imagen muestra la invitación
 * («Arrastra…» con ratón, «Toca…» en pantallas táctiles) y la regla de archivos;
 * con imagen la presenta y un toque permite reemplazarla. Las acciones
 * adicionales (Cambiar, Quitar) viven fuera del recuadro.
 */
export function ImageDropTile({src,file,alt="",label,dropText,tapText,hint,fit="cover",accept="image/jpeg,image/png,image/webp",multiple=false,disabled=false,inputRef,className="",children,onFiles}:{src?:string|null;/** Archivo local recién elegido; tiene prioridad sobre `src`. */file?:File|null;/** Texto alternativo de la imagen presentada. */alt?:string;/** Nombre accesible del botón. */label:string;dropText:string;tapText:string;hint?:ReactNode;fit?:"cover"|"contain";accept?:string;multiple?:boolean;disabled?:boolean;/** Permite abrir el selector desde una acción externa, como «Cambiar». */inputRef?:RefObject<HTMLInputElement|null>;className?:string;/** Contenido superpuesto, como un estado de procesamiento. */children?:ReactNode;onFiles:(files:File[])=>void}){
 const[dragging,setDragging]=useState(false);
 const own=useRef<HTMLInputElement|null>(null),input=inputRef??own;
 const take=(list:FileList|null|undefined)=>{const files=Array.from(list??[]);if(files.length&&!disabled)onFiles(multiple?files:files.slice(0,1));};
 return <>
  <input ref={input} type="file" accept={accept} multiple={multiple} hidden disabled={disabled} onChange={event=>{const files=event.target.files;take(files);event.target.value="";}}/>
  <button type="button" className={["image-drop-tile",file||src?"has-image":"",dragging?"is-dragging":"",fit==="contain"?"fit-contain":"",className].filter(Boolean).join(" ")} disabled={disabled} aria-label={label} onClick={()=>input.current?.click()} onDragOver={event=>{if(disabled||!event.dataTransfer.types.includes("Files"))return;event.preventDefault();event.dataTransfer.dropEffect="copy";setDragging(true)}} onDragLeave={event=>{if(!event.currentTarget.contains(event.relatedTarget as Node|null))setDragging(false)}} onDrop={event=>{event.preventDefault();setDragging(false);take(event.dataTransfer.files)}}>
   {file?<FileImage file={file} alt={alt}/>:src?<img src={src} alt={alt} decoding="async"/>:<span className="image-drop-empty"><Icon name="upload" size={20}/><b><span className="image-drop-mouse">{dropText}</span><span className="image-drop-touch">{tapText}</span></b>{hint&&<small>{hint}</small>}</span>}
   {children}
  </button>
 </>;
}
