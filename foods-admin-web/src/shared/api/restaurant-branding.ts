"use client";
import {apiFetch} from "./client";

// Resolve a fresh signed URL for each document, inside the authenticated tenant.
// Normalize the PROCESSING original too: jsPDF cannot embed every WebP variant.
export async function loadRestaurantReportLogo():Promise<Uint8Array|null>{
 try{
  const branding=await apiFetch<{restaurantLogoUrl?:string|null}>("restaurant-branding");
  if(!branding.restaurantLogoUrl)return null;
  const url=new URL(branding.restaurantLogoUrl,location.origin);
  if(url.protocol!=="https:"&&url.origin!==location.origin)return null;
  const response=await fetch(url,{credentials:"omit",signal:AbortSignal.timeout(10000)});
  if(!response.ok)return null;
  const image=await createImageBitmap(await response.blob());
  try{
   const ratio=Math.min(1,600/Math.max(image.width,image.height));
   const canvas=document.createElement("canvas");canvas.width=Math.max(1,Math.round(image.width*ratio));canvas.height=Math.max(1,Math.round(image.height*ratio));
   canvas.getContext("2d")!.drawImage(image,0,0,canvas.width,canvas.height);
   const blob=await new Promise<Blob|null>(resolve=>canvas.toBlob(resolve,"image/png"));
   return blob?new Uint8Array(await blob.arrayBuffer()):null;
  }finally{image.close();}
 }catch{return null;} // A missing logo never prevents collecting or printing.
}
