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
  staleTime:0,
  refetchOnWindowFocus:"always",
  refetchInterval:30000,
 });
}
