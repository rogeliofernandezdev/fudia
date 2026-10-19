/**
 * Periodos rápidos de Ventas. Todo se calcula con la fecha calendario del local activo (su zona horaria),
 * nunca con la del navegador: el backend interpreta `from`/`to` como días locales del mismo local.
 */
export type SalesPeriod="today"|"week"|"month"|"previousMonth"|"custom";
export type DateRange={from:string;to:string};

export const salesPeriods:{value:SalesPeriod;label:string;summary:string}[]=[
 {value:"today",label:"Hoy",summary:"Ventas de hoy"},
 {value:"week",label:"Esta semana",summary:"Ventas de esta semana"},
 {value:"month",label:"Este mes",summary:"Ventas de este mes"},
 {value:"previousMonth",label:"Mes anterior",summary:"Ventas del mes anterior"},
 {value:"custom",label:"Personalizado",summary:"Ventas del periodo"},
];

/** Fecha de hoy (AAAA-MM-DD) en la zona horaria del local. */
export function localToday(timeZone?:string|null,now:Date=new Date()):string{
 try{return new Intl.DateTimeFormat("en-CA",{timeZone:timeZone||undefined,year:"numeric",month:"2-digit",day:"2-digit"}).format(now)}
 catch{return new Intl.DateTimeFormat("en-CA",{year:"numeric",month:"2-digit",day:"2-digit"}).format(now)}
}

// Aritmética sobre fechas calendario: se anclan a UTC solo para sumar días, sin convertir zonas.
const parse=(value:string)=>{const[year,month,day]=value.split("-").map(Number);return new Date(Date.UTC(year,month-1,day))};
const format=(value:Date)=>value.toISOString().slice(0,10);
const addDays=(value:string,days:number)=>{const date=parse(value);date.setUTCDate(date.getUTCDate()+days);return format(date)};

/** Rango de un periodo rápido a partir de la fecha de hoy del local. La semana empieza el lunes. */
export function periodRange(period:Exclude<SalesPeriod,"custom">,today:string):DateRange{
 const date=parse(today);
 switch(period){
  case "today":return {from:today,to:today};
  case "week":return {from:addDays(today,-((date.getUTCDay()+6)%7)),to:today};
  case "month":return {from:`${today.slice(0,8)}01`,to:today};
  case "previousMonth":{
   const first=new Date(Date.UTC(date.getUTCFullYear(),date.getUTCMonth()-1,1));
   const last=new Date(Date.UTC(date.getUTCFullYear(),date.getUTCMonth(),0));
   return {from:format(first),to:format(last)};
  }
 }
}
