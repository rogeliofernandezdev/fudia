"use client";
import {useCallback} from "react";
import {formatRegionalNumber} from "@/shared/i18n/regional-format";
import type {OrgSettings} from "@/shared/settings/settings-api";
import {useSession} from "./session-context";
import {useSettings} from "./settings-context";

/** Importe con símbolo, posición y decimales de la empresa y separadores del país. Conserva el signo. */
export function formatCurrency(amount:number,settings:Pick<OrgSettings,"currencySymbol"|"currencyPosition"|"currencyDecimals"|"country">,country?:string|null){
  const value=formatRegionalNumber(Math.abs(amount),country??settings.country,{minimumFractionDigits:settings.currencyDecimals,maximumFractionDigits:settings.currencyDecimals});
  const sign=amount<0?"-":"";
  return settings.currencyPosition==="before"?`${sign}${settings.currencySymbol} ${value}`:`${sign}${value} ${settings.currencySymbol}`;
}

/** Símbolo, posición y decimales de la moneda de la empresa, para `MoneyInput` y prefijos de importe. */
export function useCurrency(){
  const settings=useSettings();
  return {symbol:settings.currencySymbol,position:settings.currencyPosition==="before"?"before" as const:"after" as const,decimals:settings.currencyDecimals};
}

/** Formateador de dinero homologado para toda la interfaz (nunca `toFixed` + símbolo a mano). */
export function useMoney(){
  const settings=useSettings();
  const{location}=useSession();
  const country=location?.country;
  return useCallback((value:number|string|null|undefined)=>formatCurrency(Number(value)||0,settings,country),[settings,country]);
}
