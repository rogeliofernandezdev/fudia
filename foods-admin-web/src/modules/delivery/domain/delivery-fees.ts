/** Rango de tarifa: hasta `maxDistanceKm` (línea recta desde el local) cuesta `fee`. */
export type DeliveryFeeTier={maxDistanceKm:string;fee:string};
export type DeliveryFees={
 locationName:string;
 hasCoordinates:boolean;
 currencySymbol:string;
 tiers:DeliveryFeeTier[];
};
export type DeliveryFeeRow={key:string;maxDistanceKm:string;fee:string};
export type DeliveryFeeRowErrors={maxDistanceKm?:string;fee?:string};
export type DeliveryFeesPayload={tiers:{maxDistanceKm:number;fee:number}[]};

// Mismos límites que valida foods-backend; el backend sigue siendo la autoridad.
export const deliveryFeeLimits={tiers:10,maxDistanceKm:50,maxFee:999} as const;

const twoDecimals=/^\d+(?:[.,]\d{1,2})?$/;

function amount(value:string){
 const text=value.trim();
 return twoDecimals.test(text)?Number(text.replace(",",".")):Number.NaN;
}

export function deliveryFeeRows(tiers:DeliveryFeeTier[]):DeliveryFeeRow[]{
 return tiers.map((tier,index)=>({key:`tier-${index}`,maxDistanceKm:String(Number(tier.maxDistanceKm)),fee:Number(tier.fee).toFixed(2)}));
}

export function deliveryFeeErrors(rows:DeliveryFeeRow[]):Record<string,DeliveryFeeRowErrors>{
 const errors:Record<string,DeliveryFeeRowErrors>={};
 const seen=new Map<number,string>();
 for(const row of rows){
  const current:DeliveryFeeRowErrors={};
  const km=amount(row.maxDistanceKm),fee=amount(row.fee);
  if(!(km>0&&km<=deliveryFeeLimits.maxDistanceKm))current.maxDistanceKm=`Ingresa de 0.01 a ${deliveryFeeLimits.maxDistanceKm} km.`;
  else if(seen.has(km))current.maxDistanceKm="Este límite ya existe en otro rango.";
  else seen.set(km,row.key);
  if(!(fee>=0&&fee<=deliveryFeeLimits.maxFee))current.fee=`Ingresa un costo de 0 a ${deliveryFeeLimits.maxFee}.`;
  if(current.maxDistanceKm||current.fee)errors[row.key]=current;
 }
 return errors;
}

/** Payload ordenado por distancia; lanza si hay errores para no enviar datos inválidos. */
export function deliveryFeesPayload(rows:DeliveryFeeRow[]):DeliveryFeesPayload{
 if(rows.length>deliveryFeeLimits.tiers)throw new Error(`Configura como máximo ${deliveryFeeLimits.tiers} rangos.`);
 if(Object.keys(deliveryFeeErrors(rows)).length)throw new Error("Revisa los rangos marcados.");
 return {tiers:rows.map(row=>({maxDistanceKm:amount(row.maxDistanceKm),fee:amount(row.fee)})).sort((a,b)=>a.maxDistanceKm-b.maxDistanceKm)};
}
