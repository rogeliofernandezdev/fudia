"use client";
import {createContext,useContext} from "react";
import {useQuery as useReactQuery} from "@tanstack/react-query";

export type OrgSettings={
  country:string;
  currency:string;
  currencySymbol:string;
  currencyPosition:"before"|"after";
  currencyDecimals:number;
  taxName:string;
  taxRate:number;
  taxIncluded:boolean;
  currencyOptions?:Array<{code:string;name:string;symbol:string;decimals:number}>;
  countryOptions?:Array<{code:string;name:string;defaultCurrency:string}>;
};

const defaults:OrgSettings={country:"PE",currency:"PEN",currencySymbol:"S/",currencyPosition:"before",currencyDecimals:2,taxName:"IGV",taxRate:0.18,taxIncluded:false};
const Context=createContext<OrgSettings>(defaults);

async function fetchSettings():Promise<OrgSettings>{const r=await fetch("/api/admin/settings");if(!r.ok)throw new Error();return r.json()}

export function SettingsProvider({children}:{children:React.ReactNode}){
  const{data}=useReactQuery({queryKey:["org-settings"],queryFn:fetchSettings,staleTime:Infinity,retry:1});
  return <Context.Provider value={data??defaults}>{children}</Context.Provider>;
}

export function useSettings(){return useContext(Context)}

export function formatMoney(amount:number,s:OrgSettings=defaults){
  const fixed=amount.toFixed(s.currencyDecimals);
  return s.currencyPosition==="before"?`${s.currencySymbol} ${fixed}`:`${fixed} ${s.currencySymbol}`;
}
