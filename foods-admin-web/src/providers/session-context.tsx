"use client";
import {createContext,useContext} from "react";
import {useQuery} from "@tanstack/react-query";
import {useEffect} from "react";
import {loadSessionContext} from "@/shared/session/session-api";
import {expireBrowserSession} from "@/shared/session/expire-session";

export type SessionUser={id:string;name:string;platformAdmin:boolean};
export type SessionContext={user:SessionUser|null;organization:{id:string;name:string}|null;location:{id:string;name:string;country:string;timezone:string}|null;modules:Record<string,boolean>|null;menuAccess:string[];permissions:string[];setupRequired:boolean;canAccess:(access:string)=>boolean;can:(permission:string)=>boolean;isLoading:boolean;isError:boolean;isUnauthorized:boolean};

const Context=createContext<SessionContext>({user:null,organization:null,location:null,modules:null,menuAccess:[],permissions:[],setupRequired:false,canAccess:()=>false,can:()=>false,isLoading:true,isError:false,isUnauthorized:false});

export function SessionProvider({children}:{children:React.ReactNode}){
  const query=useQuery({queryKey:["session-context"],queryFn:loadSessionContext,staleTime:300000,refetchOnWindowFocus:"always",retry:1});
  const isUnauthorized=query.isError&&String(query.error).includes("context_401");
  useEffect(()=>{if(isUnauthorized)expireBrowserSession()},[isUnauthorized]);
  const permissions=query.data?.permissions??[];
  const menuAccess=query.data?.menuAccess??[];
  const value:SessionContext={user:query.data?.user??null,organization:query.data?.organization??null,location:query.data?.location??null,modules:query.data?.modules??null,menuAccess,permissions,setupRequired:query.data?.setupRequired??false,canAccess:(access)=>menuAccess.includes("*")||menuAccess.includes(access),can:(permission)=>permissions.includes("*")||permissions.includes(permission),isLoading:query.isLoading,isError:query.isError,isUnauthorized};
  return <Context.Provider value={value}>{children}</Context.Provider>;
}

export function useSession(){return useContext(Context)}
