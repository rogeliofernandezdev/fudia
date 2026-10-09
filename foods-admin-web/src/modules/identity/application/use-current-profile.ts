"use client";
import {useQuery} from "@tanstack/react-query";
import {useSession} from "@/providers/session-context";
import {getMyProfile} from "../infrastructure/identity-api";

export function useCurrentProfile(){
  const {user,organization,location}=useSession();
  return useQuery({
    queryKey:["my-profile",user?.id,organization?.id,location?.id],
    queryFn:getMyProfile,
    enabled:Boolean(user&&organization&&location),
    staleTime:300000,
    refetchOnWindowFocus:"always",
  });
}
