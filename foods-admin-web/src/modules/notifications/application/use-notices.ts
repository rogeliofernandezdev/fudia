"use client";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {useSession} from "@/providers/session-context";
import type {NoticeFeed} from "../domain/types";
import {getNotices,markAllNoticesRead,markNoticeRead} from "../infrastructure/notices-api";

/** Consulta periódica de la campana; el backend decide qué avisos ve cada rol. */
export const NOTICES_REFRESH_MS=15000;

export function useNotices(){
 const{user,organization,location,can}=useSession();
 const enabled=Boolean(user&&organization&&location&&(can("tables.read")||can("cash.manage")));
 const queryKey=["staff-notices",organization?.id,location?.id,user?.id];
 const queryClient=useQueryClient();
  // Al recuperar el foco solo consulta si la última respuesta tiene más de 10 s (antes,
 // siempre): el polling de 15 s y la apertura de la campana ya la mantienen al día.
 const feed=useQuery({queryKey,queryFn:getNotices,enabled,staleTime:10000,refetchInterval:NOTICES_REFRESH_MS,refetchOnWindowFocus:true});
 // La lectura se refleja al instante y se reconcilia con el servidor al terminar.
 const markRead=(update:(feed:NoticeFeed)=>NoticeFeed)=>({
  onMutate:async()=>{await queryClient.cancelQueries({queryKey});const previous=queryClient.getQueryData<NoticeFeed>(queryKey);if(previous)queryClient.setQueryData(queryKey,update(previous));return{previous}},
  onError:(_error:Error,_input:unknown,context?:{previous?:NoticeFeed})=>{if(context?.previous)queryClient.setQueryData(queryKey,context.previous)},
  onSettled:()=>queryClient.invalidateQueries({queryKey}),
 });
 const readOne=useMutation({mutationFn:markNoticeRead,...markRead(previous=>previous)});
 const readAll=useMutation({mutationFn:markAllNoticesRead,...markRead(previous=>({unread:0,items:previous.items.map(item=>({...item,read:true}))}))});
 return{
  enabled,feed,readAll,
  read:(id:string)=>{
   const previous=queryClient.getQueryData<NoticeFeed>(queryKey);
   const target=previous?.items.find(item=>item.id===id);
   if(!previous||!target||target.read)return;
   queryClient.setQueryData<NoticeFeed>(queryKey,{unread:Math.max(0,previous.unread-1),items:previous.items.map(item=>item.id===id?{...item,read:true}:item)});
   readOne.mutate(id);
  },
 };
}
