import {expireBrowserSession} from "@/shared/session/expire-session";
import {prepareProductImage} from "./image-upload";
import {apiFetch} from "./client";

export async function uploadProductImage(productId:string,file:File){
  const direct=await prepareProductImage(file,productId);
  if(direct)return apiFetch<{imageUrl:string}>(`products/${productId}/image/attach`,{method:"POST",body:JSON.stringify({imageId:direct.imageId})});
  return uploadLegacyProductImage(productId,file);
}
export async function uploadLegacyProductImage(productId:string,file:File){
  const form=new FormData();
  form.append("file",file);
  const response=await fetch(`/api/admin/products/${productId}/image`,{method:"POST",body:form});
  if(response.status===401)expireBrowserSession();
  if(!response.ok){
    const body=await response.json().catch(()=>({}));
    throw new Error(body.message??"No pudimos subir la imagen del producto.");
  }
  return response.json() as Promise<{imageUrl:string}>;
}
