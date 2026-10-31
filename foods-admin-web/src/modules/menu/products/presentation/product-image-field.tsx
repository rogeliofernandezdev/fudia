"use client";
import {useEffect,useRef,useState} from "react";
import {useQuery} from "@tanstack/react-query";
import {FormField,Button,Status} from "@/design-system";
import {ImageDropTile} from "@/design-system/image-drop-tile";
import {localImagePreview,readImageMetadata,validateImageFile} from "@/shared/api/image-upload";
import {getProduct} from "../infrastructure/products-api";
import type {ProductDraft} from "../domain/types";

export function ProductImageField({draft,name,busy,scopeKey,onSelect,onRemove,onValidation}:{draft:ProductDraft;name:string;busy:boolean;scopeKey:string;onSelect:(file:File|null)=>void;onRemove:()=>void;onValidation:(pending:boolean,error:string)=>void}){
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
 return <FormField as="div" className="product-image" label="Imagen" optional error={error}>
  <ImageDropTile src={source} alt={name} disabled={busy} label={source?"Cambiar imagen del producto":"Seleccionar imagen del producto"} dropText="Arrastra tu foto aquí" tapText="Toca para elegir una foto" hint="JPEG, PNG o WebP · 5 MB" removeLabel="Quitar imagen" onFiles={([file])=>void select(file)} onRemove={clear}/>
  {!changed&&image.imageStatus==="PROCESSING"&&<p role="status"><Status tone="orange">Optimizando imagen</Status></p>}
  {!changed&&image.imageStatus==="FAILED"&&<p role="status"><Status tone="red">No se pudo optimizar</Status> Selecciona otra imagen para reintentar.</p>}
  {!changed&&detail.isError&&<p role="alert">No pudimos actualizar la imagen. <Button kind="secondary" onClick={()=>void detail.refetch()}>Reintentar</Button></p>}
 </FormField>
}
