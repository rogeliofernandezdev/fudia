"use client";
/* eslint-disable @next/next/no-img-element -- Signed URLs and immediate blob previews bypass image optimization. */
import {useEffect,useRef,useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {FormField,Button,Status} from "@/design-system";
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
 return <FormField as="div" className="product-image span-2" label="Imagen del producto" optional help="JPEG, PNG o WebP · máximo 5 MB · hasta 4000 × 4000 px" error={error}>
  <input ref={input} type="file" accept="image/png,image/jpeg,image/webp" hidden disabled={busy} onChange={event=>{const file=event.target.files?.[0];event.target.value="";if(file)void select(file)}}/>
  <div className="image-drop">
   <button type="button" className="product-image-choice" disabled={busy} aria-label="Seleccionar imagen del producto" onClick={()=>input.current?.click()}>{source?<img src={source} alt={name} decoding="async"/>:<div className="image-drop-empty"><Icon name="box" size={24}/><span>Seleccionar imagen</span></div>}</button>
   {source&&<button type="button" className="image-clear" disabled={busy} onClick={clear} aria-label="Quitar imagen"><Icon name="close" size={16}/></button>}
  </div>
  {!changed&&image.imageStatus==="PROCESSING"&&<p role="status"><Status tone="orange">Optimizando imagen</Status></p>}
  {!changed&&image.imageStatus==="FAILED"&&<p role="status"><Status tone="red">No se pudo optimizar</Status> Selecciona otra imagen para reintentar.</p>}
  {!changed&&detail.isError&&<p role="alert">No pudimos actualizar la imagen. <Button kind="secondary" onClick={()=>void detail.refetch()}>Reintentar</Button></p>}
 </FormField>
}
