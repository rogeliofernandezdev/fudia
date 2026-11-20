"use client";
import {createContext,useContext,useEffect,useMemo} from "react";
import {useQuery} from "@tanstack/react-query";
import {loadSessionContext} from "@/shared/session/session-api";
import {expireBrowserSession} from "@/shared/session/expire-session";

export type SessionUser={id:string;name:string;platformAdmin:boolean};
export type SessionContext={user:SessionUser|null;organization:{id:string;name:string}|null;location:{id:string;name:string;country:string;timezone:string}|null;modules:Record<string,boolean>|null;menuAccess:string[];permissions:string[];setupRequired:boolean;cashCollection:boolean;canAccess:(access:string)=>boolean;can:(permission:string)=>boolean;isLoading:boolean;isError:boolean;isUnauthorized:boolean};

const Context=createContext<SessionContext>({user:null,organization:null,location:null,modules:null,menuAccess:[],permissions:[],setupRequired:false,cashCollection:false,canAccess:()=>false,can:()=>false,isLoading:true,isError:false,isUnauthorized:false});

const isUnauthorizedError=(error:unknown)=>String(error).includes("context_401");

export function SessionProvider({children}:{children:React.ReactNode}){
  // Un 401 es definitivo: reintentarlo solo retrasa la salida al login.
  const query=useQuery({queryKey:["session-context"],queryFn:loadSessionContext,staleTime:300000,refetchInterval:30000,refetchOnWindowFocus:"always",retry:(failures,error)=>!isUnauthorizedError(error)&&failures<1});
  const isUnauthorized=query.isError&&isUnauthorizedError(query.error);
  useEffect(()=>{if(isUnauthorized)expireBrowserSession()},[isUnauthorized]);
  const {data,isLoading,isError}=query;
  // Un valor estable evita volver a renderizar a todos los consumidores de
  // useSession() cada vez que el proveedor se renderiza.
  const value=useMemo<SessionContext>(()=>{
    const permissions=data?.permissions??[];
    const menuAccess=data?.menuAccess??[];
    const cashCollection=Boolean(data?.cashCollection);
    return {user:data?.user??null,organization:data?.organization??null,location:data?.location??null,modules:data?.modules??null,menuAccess,permissions,setupRequired:data?.setupRequired??false,cashCollection,canAccess:(access)=>(access==="pos"&&cashCollection)||menuAccess.includes("*")||menuAccess.includes(access),can:(permission)=>permissions.includes("*")||permissions.includes(permission),isLoading,isError,isUnauthorized};
  },[data,isLoading,isError,isUnauthorized]);
  return <Context.Provider value={value}>{children}</Context.Provider>;
}

export function useSession(){return useContext(Context)}
