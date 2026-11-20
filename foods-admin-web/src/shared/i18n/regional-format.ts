const UI_LANGUAGE="es";

export type RegionalFormatContext={country?:string|null;timeZone?:string|null};

function localeForCountry(country?:string|null){
  const region=country?.trim().toUpperCase();
  return region&&/^[A-Z]{2}$/.test(region)?`${UI_LANGUAGE}-${region}`:UI_LANGUAGE;
}

// Crear un Intl.*Format cuesta mucho más que usarlo, y las tablas formatean cada
// celda de dinero y fecha en cada refresco. Se reutiliza uno por locale y opciones.
const formatterLimit=200;
const dateFormatters=new Map<string,Intl.DateTimeFormat>();
const numberFormatters=new Map<string,Intl.NumberFormat>();

function cached<T>(cache:Map<string,T>,locale:string,options:object,create:()=>T){
  const key=`${locale}|${JSON.stringify(options)}`;
  let formatter=cache.get(key);
  if(!formatter){
    formatter=create();
    if(cache.size>=formatterLimit)cache.clear();
    cache.set(key,formatter);
  }
  return formatter;
}

function dateFormatter(locale:string,options:Intl.DateTimeFormatOptions){
  return cached(dateFormatters,locale,options,()=>new Intl.DateTimeFormat(locale,options));
}

function numberFormatter(locale:string,options:Intl.NumberFormatOptions){
  return cached(numberFormatters,locale,options,()=>new Intl.NumberFormat(locale,options));
}

export function formatRegionalDateTime(
  value:string,
  context:RegionalFormatContext={},
  options:Intl.DateTimeFormatOptions={dateStyle:"short",timeStyle:"short"},
){
  // PostgreSQL's OF format can return an hour-only UTC offset (e.g. +00).
  const normalized=value.trim().replace(/(T\d{2}:\d{2}:\d{2}(?:\.\d+)?)([+-]\d{2})$/,"$1$2:00");
  const date=new Date(normalized);
  if(Number.isNaN(date.getTime()))return "—";
  try{
    return dateFormatter(localeForCountry(context.country),{
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
    return dateFormatter(localeForCountry(country),{
      ...options,
      timeZone:"UTC",
    }).format(date);
  }catch{
    return "—";
  }
}

export function formatRegionalNumber(value:number,country?:string|null,options:Intl.NumberFormatOptions={}){
  if(!Number.isFinite(value))return "—";
  try{return numberFormatter(localeForCountry(country),options).format(value)}catch{return String(value)}
}
