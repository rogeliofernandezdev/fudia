"use client";
import {useState} from "react";
import {useQuery,useQueryClient} from "@tanstack/react-query";
import {ErrorState,Skeleton} from "@/design-system";
import {useSession} from "@/providers/session-context";
import {getRestaurantProfile,profileDraft,saveRestaurantProfile,type RestaurantProfileDraft} from "../infrastructure/restaurant-profile-api";
import {RestaurantLogoField,VirtualMenuField} from "./restaurant-profile-fields";
import {OrganizationPanel} from "./organization-panel";

/** Borrador de la identidad (fundación, logo y carta) que Mi restaurante guarda junto con la empresa. */
export function useRestaurantProfile(){
 const{organization}=useSession(),client=useQueryClient(),key=["restaurant-profile",organization?.id];
 const query=useQuery({queryKey:key,queryFn:()=>getRestaurantProfile(),refetchInterval:query=>query.state.data?.menuPages.some(page=>page.status==="PROCESSING")||query.state.data?.logo?.status==="PROCESSING"?2000:60000});
 const[draft,setDraft]=useState<RestaurantProfileDraft|null>(null),[reading,setReading]=useState(false);
 const value=draft??(query.data?profileDraft(query.data):null);
 async function persist(){
  if(!draft)return;
  const profile=await saveRestaurantProfile(draft,organization?.id??"");
  client.setQueryData(key,profile);setDraft(null);
 }
 return{query,value,dirty:draft!==null,reading,setReading,change:setDraft,reset:()=>setDraft(null),persist};
}
export type RestaurantProfileState=ReturnType<typeof useRestaurantProfile>;

/** Logo dentro del panel Marca; el error se informa una sola vez en Carta virtual. */
export function ProfileLogoSlot({profile,disabled}:{profile:RestaurantProfileState;disabled:boolean}){
 const{query,value}=profile;
 if(query.isPending)return <div role="status" aria-label="Cargando logo"><Skeleton width="var(--size-120)" height="var(--size-120)" radius="var(--radius-tile)"/></div>;
 if(!value)return null;
 return <RestaurantLogoField value={value.logo} onChange={logo=>profile.change({...value,logo})} disabled={disabled} onBusy={profile.setReading}/>;
}

export function ProfileMenuPanel({profile,disabled}:{profile:RestaurantProfileState;disabled:boolean}){
 const{query,value}=profile,max=query.data?.maxMenuPages??0;
 return <OrganizationPanel icon="utensils" tone="digital" title="Carta virtual" className="organization-menu" aside={value&&max?<span className="organization-panel-count">{value.pages.length} de {max}</span>:undefined}>
  {query.isPending?<div className="organization-menu-loading" role="status" aria-label="Cargando carta virtual">{[0,1,2].map(item=><Skeleton key={item} height="var(--size-210)" radius="var(--radius-tile)"/>)}</div>
  :query.isError||!value?<ErrorState title="No pudimos cargar el logo y la carta" text={query.error?.message??"Intenta nuevamente."} onRetry={()=>void query.refetch()} retrying={query.isFetching}/>
  :<VirtualMenuField pages={value.pages} onChange={pages=>profile.change({...value,pages})} maxPages={max} disabled={disabled} onBusy={profile.setReading}/>}
 </OrganizationPanel>;
}
