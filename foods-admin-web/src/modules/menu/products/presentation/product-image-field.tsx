"use client";
/* eslint-disable @next/next/no-img-element -- Signed URLs and immediate blob previews bypass image optimization. */
import {useEffect,useRef,useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {FormField,Button,IconButton,Status} from "@/design-system";
import {Icon} from "@/design-system/icons";
import {localImagePreview,readImageMetadata,validateImageFile} from "@/shared/api/image-upload";
import {getProduct} from "../infrastructure/products-api";
import type {ProductDraft} from "../domain/types";

export function ProductImageField({draft,name,busy,scopeKey,onSelect,onRemove,onValidation}:{draft:ProductDraft;name:string;busy:boolean;scopeKey:string;onSelect:(file:File|null)=>void;onRemove:()=>void;onValidation:(pending:boolean,error:string)=>void}){
 const input=useRef<HTMLInputElement>(null);
 const sequence=useRef(0);
 const blob=useRef<string|null>(null);
 const[preview,setPreview]=useState<string|null>(null);
 const[changed,setChanged]=useState(false);
 const[error,setError]=useState("");
 const[dragging,setDragging]=useState(false);
 const detail=useQuery({queryKey:["product-image-detail",scopeKey,draft.id],queryFn:()=>getProduct(draft.id!),enabled:Boolean(draft.id&&!changed),retry:1,refetchOnWindowFocus:true,refetchInterval:query=>query.state.data?.imageStatus==="PROCESSING"?2000:30*60*1000});
 const image=detail.data??draft;
 const local=image.imageStatus==="PROCESSING"?localImagePreview(scopeKey,image.imageId):null;
 const source=changed?preview:local??image.imageVariants?.medium??image.imageUrl;
 useEffect(()=>()=>{sequence.current++;if(blob.current)URL.revokeObjectURL(blob.current)},[]);
 async function select(file:File){
  const current=++sequence.current;
  try{validateImageFile(file)}catch(error){const message=error instanceof Error?error.message:"La imagen no es válida.";setError(message);onValidation(false,message);return}
  if(blob.current)URL.revokeObjectURL(blob.current);
  blob.current=URL.createObjectURL(file);setPreview(blob.current);setChanged(true);setError("");onSelect(file);onValidation(true,"");
  try{await readImageMetadata(file);if(current===sequence.current)onValidation(false,"")}
  catch(error){if(current!==sequence.current)return;const message=error instanceof Error?error.message:"La imagen no es válida.";setError(message);onSelect(null);onValidation(false,message)}
 }
 function clear(){sequence.current++;if(blob.current)URL.revokeObjectURL(blob.current);blob.current=null;setPreview(null);setChanged(true);setError("");onSelect(null);onValidation(false,"");onRemove()}
 function pick(){input.current?.click()}
 return <FormField as="div" className="product-image" label="Imagen" optional error={error}>
  <input ref={input} type="file" accept="image/png,image/jpeg,image/webp" hidden disabled={busy} onChange={event=>{const file=event.target.files?.[0];event.target.value="";if(file)void select(file)}}/>
  <button type="button" className={`product-image-tile${source?" has-image":""}${dragging?" is-dragging":""}`} disabled={busy} aria-label={source?"Cambiar imagen del producto":"Seleccionar imagen del producto"} onClick={pick} onDragOver={event=>{if(busy||!event.dataTransfer.types.includes("Files"))return;event.preventDefault();event.dataTransfer.dropEffect="copy";setDragging(true)}} onDragLeave={event=>{if(!event.currentTarget.contains(event.relatedTarget as Node|null))setDragging(false)}} onDrop={event=>{event.preventDefault();setDragging(false);const file=event.dataTransfer.files?.[0];if(!busy&&file)void select(file)}}>{source?<img src={source} alt={name} decoding="async"/>:<span className="product-image-empty"><Icon name="upload" size={20}/><b><span className="product-image-drop-hint">Arrastra tu foto aquí</span><span className="product-image-tap-hint">Toca para elegir una foto</span></b><small><span className="product-image-drop-hint">o haz clic para elegirla<br/></span>JPEG, PNG o WebP · 5 MB</small></span>}</button>
  {source&&<div className="product-image-actions"><Button kind="secondary" icon="refresh" disabled={busy} onClick={pick}>Cambiar</Button><IconButton icon="trash" label="Quitar imagen" disabled={busy} onClick={clear}/></div>}
  {!changed&&image.imageStatus==="PROCESSING"&&<p role="status"><Status tone="orange">Optimizando imagen</Status></p>}
  {!changed&&image.imageStatus==="FAILED"&&<p role="status"><Status tone="red">No se pudo optimizar</Status> Selecciona otra imagen para reintentar.</p>}
  {!changed&&detail.isError&&<p role="alert">No pudimos actualizar la imagen. <Button kind="secondary" onClick={()=>void detail.refetch()}>Reintentar</Button></p>}
 </FormField>
}
