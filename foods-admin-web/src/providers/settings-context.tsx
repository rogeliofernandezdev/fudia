"use client";
import {createContext,useContext} from "react";
import {useQuery as useReactQuery} from "@tanstack/react-query";
import {loadOrgSettings,type OrgSettings} from "@/shared/settings/settings-api";

const defaults:OrgSettings={country:"PE",currency:"PEN",currencySymbol:"S/",currencyPosition:"before",currencyDecimals:2,taxName:"IGV",taxRate:0.18,taxIncluded:false};
const Context=createContext<OrgSettings>(defaults);

export function SettingsProvider({children}:{children:React.ReactNode}){
  const{data}=useReactQuery({queryKey:["org-settings"],queryFn:loadOrgSettings,staleTime:Infinity,retry:1});
  return <Context.Provider value={data??defaults}>{children}</Context.Provider>;
}

export function useSettings(){return useContext(Context)}

export function formatMoney(amount:number,s:OrgSettings=defaults){
  const fixed=amount.toFixed(s.currencyDecimals);
  return s.currencyPosition==="before"?`${s.currencySymbol} ${fixed}`:`${fixed} ${s.currencySymbol}`;
}
