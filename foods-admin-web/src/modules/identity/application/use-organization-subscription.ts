"use client";
import {useQuery} from "@tanstack/react-query";
import {useSession} from "@/providers/session-context";
import {getOrganizationSubscription} from "../infrastructure/identity-api";

export function useOrganizationSubscription(){
 const{user,organization,can}=useSession();
 return useQuery({
  queryKey:["organization-subscription",organization?.id,user?.id],
  queryFn:getOrganizationSubscription,
  enabled:Boolean(user&&organization&&can("subscription.read")),
  // El plan cambia muy rara vez y el backend aplica sus límites: basta con revisarlo
 // cada minuto y al volver a la pestaña si el dato tiene más de un minuto.
  staleTime:60000,
  refetchOnWindowFocus:true,
  refetchInterval:60000,
 });
}
