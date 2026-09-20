const UI_LANGUAGE="es";

export type RegionalFormatContext={country?:string|null;timeZone?:string|null};

function localeForCountry(country?:string|null){
  const region=country?.trim().toUpperCase();
  return region&&/^[A-Z]{2}$/.test(region)?`${UI_LANGUAGE}-${region}`:UI_LANGUAGE;
}

export function formatRegionalDateTime(value:string,context:RegionalFormatContext={}){
  const date=new Date(value);
  if(Number.isNaN(date.getTime()))return "—";
  try{
    return new Intl.DateTimeFormat(localeForCountry(context.country),{
      dateStyle:"short",
      timeStyle:"short",
      timeZone:context.timeZone||undefined,
    }).format(date);
  }catch{
    return "—";
  }
}

export function formatRegionalNumber(value:number,country?:string|null,options:Intl.NumberFormatOptions={}){
  if(!Number.isFinite(value))return "—";
  try{return new Intl.NumberFormat(localeForCountry(country),options).format(value)}catch{return String(value)}
}
