const UI_LANGUAGE="es";

export type RegionalFormatContext={country?:string|null;timeZone?:string|null};

function localeForCountry(country?:string|null){
  const region=country?.trim().toUpperCase();
  return region&&/^[A-Z]{2}$/.test(region)?`${UI_LANGUAGE}-${region}`:UI_LANGUAGE;
}

export function formatRegionalDateTime(
  value:string,
  context:RegionalFormatContext={},
  options:Intl.DateTimeFormatOptions={dateStyle:"short",timeStyle:"short"},
){
  const date=new Date(value);
  if(Number.isNaN(date.getTime()))return "—";
  try{
    return new Intl.DateTimeFormat(localeForCountry(context.country),{
      ...options,
      timeZone:context.timeZone||options.timeZone,
    }).format(date);
  }catch{
    return "—";
  }
}

export function formatRegionalCalendarDate(
  value:string,
  country?:string|null,
  options:Intl.DateTimeFormatOptions={dateStyle:"short"},
){
  const match=/^(\d{4})-(\d{2})-(\d{2})/.exec(value);
  if(!match)return "—";
  const date=new Date(Date.UTC(Number(match[1]),Number(match[2])-1,Number(match[3])));
  try{
    return new Intl.DateTimeFormat(localeForCountry(country),{
      ...options,
      timeZone:"UTC",
    }).format(date);
  }catch{
    return "—";
  }
}

export function formatRegionalNumber(value:number,country?:string|null,options:Intl.NumberFormatOptions={}){
  if(!Number.isFinite(value))return "—";
  try{return new Intl.NumberFormat(localeForCountry(country),options).format(value)}catch{return String(value)}
}
