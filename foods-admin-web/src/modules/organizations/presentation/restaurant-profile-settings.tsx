"use client";
import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,ErrorState,Skeleton} from "@/design-system";
import {useSession} from "@/providers/session-context";
import {useFeedback} from "@/providers/feedback-provider";
import {getRestaurantProfile,profileDraft,saveRestaurantProfile,type RestaurantProfileDraft} from "../infrastructure/restaurant-profile-api";
import {RestaurantProfileFields} from "./restaurant-profile-fields";

export function RestaurantProfileSettings({canManage}:{canManage:boolean}){
 const{organization}=useSession(),{notify}=useFeedback(),client=useQueryClient();
 const key=["restaurant-profile",organization?.id];
 const query=useQuery({queryKey:key,queryFn:()=>getRestaurantProfile(),refetchInterval:query=>query.state.data?.menuPages.some(page=>page.status==="PROCESSING")||query.state.data?.logo?.status==="PROCESSING"?2000:60000});
 const[draft,setDraft]=useState<RestaurantProfileDraft|null>(null),[reading,setReading]=useState(false);
 const save=useMutation({mutationFn:()=>saveRestaurantProfile(draft!,organization?.id??""),onSuccess:profile=>{client.setQueryData(key,profile);setDraft(null);notify({tone:"success",title:"Identidad actualizada",message:"El logo y la carta virtual ya están guardados. Las imágenes se optimizan en segundo plano."});},onError:error=>notify({tone:"danger",title:"No se pudo guardar",message:error.message})});
 return <section className="panel organization-section"><header className="organization-section-header"><div><h2>Identidad y carta virtual</h2><p>Información compartida por tus locales.</p></div></header><div className="organization-fields single">
 {query.isPending?<div role="status" aria-label="Cargando identidad"><Skeleton height="var(--size-120)"/></div>:query.isError?<ErrorState title="No pudimos cargar la identidad" text={query.error.message} onRetry={()=>void query.refetch()}/>:<>
 <RestaurantProfileFields value={draft??profileDraft(query.data)} onChange={setDraft} maxPages={query.data.maxMenuPages} disabled={!canManage||save.isPending} onBusy={setReading}/>
 {canManage&&<div className="organization-savebar"><Button kind="ghost" disabled={!draft||reading||save.isPending} onClick={()=>setDraft(null)}>Cancelar</Button><Button icon="check" disabled={!draft||reading||save.isPending} loading={save.isPending} onClick={()=>save.mutate()}>Guardar identidad</Button></div>}
 </>}
 </div></section>;
}
