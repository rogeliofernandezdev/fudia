"use client";
import {apiFetch,ApiClientError} from "./client";

export type ProductImageState="PROCESSING"|"READY"|"FAILED";
export type ProductImageResult={imageId?:string;imageKey?:string;imageStatus?:ProductImageState;imageUrl:string|null;imageExpiresAt?:string;imageVariants?:{thumbnail?:string;medium?:string}};
type ImageMetadata={contentType:string;sizeBytes:number;width:number;height:number};
type UploadAuthorization={imageId:string;objectKey:string;uploadUrl:string;method:"POST";fields:Record<string,string>;expiresAt:string};
const maxBytes=5*1024*1024;
const prepared=new WeakMap<File,Map<string,ProductImageResult>>();
const uploaded=new WeakMap<File,Map<string,string>>();
const previews=new Map<string,{url:string;expires:number}>();

export function validateImageFile(file:File){
 if(!["image/jpeg","image/png","image/webp"].includes(file.type))throw new Error("Selecciona una imagen JPEG, PNG o WebP.");
 if(file.size<1||file.size>maxBytes)throw new Error("La imagen debe pesar como máximo 5 MB.");
}
export async function readImageMetadata(file:File):Promise<ImageMetadata>{
 validateImageFile(file);
 const url=URL.createObjectURL(file);
 try{
  const dimensions=await new Promise<{width:number;height:number}>((resolve,reject)=>{
   const image=new Image();
   const timer=setTimeout(()=>{image.src="";reject(new Error("No pudimos leer la imagen. Selecciona otro archivo."))},10000);
   image.onload=()=>{clearTimeout(timer);resolve({width:image.naturalWidth,height:image.naturalHeight})};
   image.onerror=()=>{clearTimeout(timer);reject(new Error("El archivo no es una imagen válida."))};
   image.src=url;
  });
  if(dimensions.width<1||dimensions.height<1||dimensions.width>4000||dimensions.height>4000)throw new Error("La resolución máxima es 4000 × 4000 píxeles.");
  return {contentType:file.type,sizeBytes:file.size,...dimensions};
 }finally{URL.revokeObjectURL(url)}
}
function rememberPreview(scopeKey:string,imageId:string,file:File){
 if(!scopeKey)return;
 const now=Date.now();
 for(const[key,value]of previews){if(value.expires<=now){URL.revokeObjectURL(value.url);previews.delete(key)}}
 while(previews.size>=8){const first=previews.entries().next().value;if(!first)break;URL.revokeObjectURL(first[1].url);previews.delete(first[0])}
 const key=`${scopeKey}:${imageId}`;const old=previews.get(key);if(old)URL.revokeObjectURL(old.url);
 previews.set(key,{url:URL.createObjectURL(file),expires:now+5*60*1000});
}
export function localImagePreview(scopeKey:string,imageId?:string){
 if(!scopeKey||!imageId)return null;
 const key=`${scopeKey}:${imageId}`,value=previews.get(key);
 if(value&&value.expires>Date.now())return value.url;
 if(value){URL.revokeObjectURL(value.url);previews.delete(key)}
 return null;
}
export async function prepareProductImage(file:File,productId?:string,scopeKey=""):Promise<ProductImageResult|null>{
 const cacheKey=`${scopeKey}:${productId??"new"}`;
 const cached=scopeKey?prepared.get(file)?.get(cacheKey):undefined;
 if(cached)return cached; // A failed product save must not upload the same photo again.
 const pending=scopeKey?uploaded.get(file)?.get(cacheKey):undefined;
 if(pending){const result=await apiFetch<ProductImageResult>(`product-images/${pending}/complete`,{method:"POST",body:"{}"});let map=prepared.get(file);if(!map){map=new Map();prepared.set(file,map)}map.set(cacheKey,result);return result}
 const metadata=await readImageMetadata(file);
 let authorization:UploadAuthorization;
 try{authorization=await apiFetch<UploadAuthorization>("product-images/uploads",{method:"POST",body:JSON.stringify({...metadata,...(productId?{productId}:{})})})}
 catch(error){
  // Preserve the existing local/AWS adapter. GCS mode never returns this capability error.
  if(error instanceof ApiClientError&&error.code==="direct_upload_unavailable")return null;
  throw error;
 }
 const target=new URL(authorization.uploadUrl);
 if(target.protocol!=="https:"||target.hostname!=="storage.googleapis.com"||authorization.method!=="POST")throw new Error("La autorización de subida no es válida.");
 const form=new FormData();
 for(const[key,value]of Object.entries(authorization.fields))form.append(key,value);
 form.append("file",file); // Storage requires the file to be the last field.
 let response:Response;
 try{response=await fetch(target,{method:"POST",body:form,credentials:"omit",redirect:"error",signal:AbortSignal.timeout(90000)})}
 catch{throw new Error("No pudimos subir la imagen a Storage. Revisa tu conexión y la configuración CORS.")}
 if(!response.ok)throw new Error(response.status===400||response.status===413?"Storage rechazó la imagen. Revisa su formato y tamaño.":"La autorización de subida expiró o fue rechazada. Intenta nuevamente.");
 if(scopeKey){let map=uploaded.get(file);if(!map){map=new Map();uploaded.set(file,map)}map.set(cacheKey,authorization.imageId)}
 rememberPreview(scopeKey,authorization.imageId,file);
 const result=await apiFetch<ProductImageResult>(`product-images/${authorization.imageId}/complete`,{method:"POST",body:"{}"});
 if(scopeKey){let map=prepared.get(file);if(!map){map=new Map();prepared.set(file,map)}map.set(cacheKey,result)}
 return result;
}
