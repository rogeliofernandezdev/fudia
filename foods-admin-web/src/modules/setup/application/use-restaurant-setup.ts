"use client";
import {useQuery} from "@tanstack/react-query";
import {useSession} from "@/providers/session-context";
import {canOpenNavigationItem,navigationItemForPath} from "@/shell/navigation";
import {getRestaurantSetup} from "../infrastructure/setup-api";

export function useRestaurantSetup(enabled=true){
  const session=useSession();
  const queryKey=["restaurant-setup",session.organization?.id,session.location?.id];
  const query=useQuery({queryKey,queryFn:getRestaurantSetup,enabled:enabled&&Boolean(session.organization&&session.location),staleTime:0});
  const canVisit=(href:string)=>{
    const item=navigationItemForPath(href);
    return Boolean(item&&session.user&&session.modules&&canOpenNavigationItem(item,{...session,user:session.user,modules:session.modules}));
  };
  return {query,queryKey,session,canVisit};
}
