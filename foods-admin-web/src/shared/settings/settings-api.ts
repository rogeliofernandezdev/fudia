import {apiFetch} from "@/shared/api/client";

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

export async function loadOrgSettings():Promise<OrgSettings>{
  return apiFetch<OrgSettings>("settings");
}
