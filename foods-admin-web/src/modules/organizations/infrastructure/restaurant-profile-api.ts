"use client";
import {authenticatedFetch} from "@/shared/api/client";
import {readImageMetadata} from "@/shared/api/image-upload";

export type RestaurantImage={id:string;purpose:"restaurant_logo"|"virtual_menu";url:string|null;status:"PROCESSING"|"READY"|"FAILED";expiresAt?:string};
export type RestaurantProfile={foundedOn:string;logo:RestaurantImage|null;menuPages:RestaurantImage[];maxMenuPages:number};
export type MediaDraft={image?:RestaurantImage;file?:File};
export type RestaurantProfileDraft={foundedOn:string;logo:MediaDraft|null;pages:MediaDraft[]};
export function profileDraft(profile:RestaurantProfile):RestaurantProfileDraft{return{foundedOn:profile.foundedOn,logo:profile.logo?{image:profile.logo}:null,pages:profile.menuPages.map(image=>({image}))};}
const root=(organizationId?:string)=>organizationId?`/api/platform/organizations/${encodeURIComponent(organizationId)}/profile`:"/api/admin/organization/profile";
export const getRestaurantProfile=(organizationId?:string)=>authenticatedFetch<RestaurantProfile>(root(organizationId));
const uploads=new WeakMap<File,Map<string,{id:string;complete?:RestaurantImage}>>();
async function upload(file:File,purpose:RestaurantImage["purpose"],scope:string,organizationId?:string){
 const key=`${scope}:${purpose}`,previous=uploads.get(file)?.get(key);
 if(previous?.complete)return previous.complete;
 let id=previous?.id;
 if(!id){
  const metadata=await readImageMetadata(file);
  const authorization=await authenticatedFetch<{imageId:string;uploadUrl:string;method:string;fields:Record<string,string>}>(`${root(organizationId)}/uploads`,{method:"POST",body:JSON.stringify({...metadata,purpose})});
  const target=new URL(authorization.uploadUrl);
  if(target.protocol!=="https:"||target.hostname!=="storage.googleapis.com"||authorization.method!=="POST")throw new Error("La autorización de subida no es válida.");
  const body=new FormData();Object.entries(authorization.fields).forEach(([name,value])=>body.append(name,value));body.append("file",file);
  const response=await fetch(target,{method:"POST",body,credentials:"omit",redirect:"error",signal:AbortSignal.timeout(90000)});
  if(!response.ok)throw new Error("Storage rechazó la imagen. Revisa el tamaño, la conexión y vuelve a guardar.");
  id=authorization.imageId;
  let cache=uploads.get(file);if(!cache){cache=new Map();uploads.set(file,cache);}cache.set(key,{id});
 }
 const complete=await authenticatedFetch<RestaurantImage>(`${root(organizationId)}/images/${id}/complete`,{method:"POST",body:"{}"});
 uploads.get(file)!.set(key,{id,complete});return complete;
}
export async function saveRestaurantProfile(draft:RestaurantProfileDraft,scope:string,organizationId?:string){
 const logo=draft.logo?.file?await upload(draft.logo.file,"restaurant_logo",scope,organizationId):draft.logo?.image;
 const pages:RestaurantImage[]=[];
 for(const page of draft.pages)pages.push(page.file?await upload(page.file,"virtual_menu",scope,organizationId):page.image!);
 await authenticatedFetch<void>(root(organizationId),{method:"PATCH",body:JSON.stringify({foundedOn:draft.foundedOn,logoImageId:logo?.id??null,menuImageIds:pages.map(page=>page.id)})});
 return getRestaurantProfile(organizationId);
}
