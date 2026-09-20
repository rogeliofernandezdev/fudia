"use client";
import {createContext,useContext} from "react";
import {useQuery} from "@tanstack/react-query";
import {useRouter} from "next/navigation";
import {useEffect} from "react";
import {deleteSession,loadSessionContext} from "@/shared/session/session-api";

export type SessionUser={id:string;name:string;platformAdmin:boolean};
export type SessionContext={user:SessionUser|null;organization:{id:string;name:string}|null;location:{id:string;name:string}|null;modules:Record<string,boolean>|null;menuAccess:string[];permissions:string[];canAccess:(access:string)=>boolean;can:(permission:string)=>boolean;isLoading:boolean;isError:boolean};

const Context=createContext<SessionContext>({user:null,organization:null,location:null,modules:null,menuAccess:[],permissions:[],canAccess:()=>false,can:()=>false,isLoading:true,isError:false});

export function SessionProvider({children}:{children:React.ReactNode}){
  const router=useRouter();
  const query=useQuery({queryKey:["session-context"],queryFn:loadSessionContext,staleTime:300000,retry:1});
  useEffect(()=>{if(query.isError){const err=String(query.error);if(err.includes("context_401")){void deleteSession().catch(()=>{}).finally(()=>{router.replace("/login");router.refresh()})}}},[query.isError,query.error,router]);
  const permissions=query.data?.permissions??[];
  const menuAccess=query.data?.menuAccess??[];
  const value:SessionContext={user:query.data?.user??null,organization:query.data?.organization??null,location:query.data?.location??null,modules:query.data?.modules??null,menuAccess,permissions,canAccess:(access)=>menuAccess.includes("*")||menuAccess.includes(access),can:(permission)=>permissions.includes("*")||permissions.includes(permission),isLoading:query.isLoading,isError:query.isError};
  return <Context.Provider value={value}>{children}</Context.Provider>;
}

export function useSession(){return useContext(Context)}
