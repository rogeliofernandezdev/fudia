import type {Draft} from "./types";

/**
 * Cuándo se vende un menú: siempre (el menú del día, que solo cambia sus platos) o en días y fechas elegidos
 * (lunes a sábado, fines de semana o un especial con sus fechas). No hay «Solo hoy»: obligaba a crear un menú nuevo
 * cada día.
 */
export type SaleWindow="always"|"days";
type Schedule=Pick<Draft,"availableDays"|"availableFrom"|"availableUntil">;

export function saleWindowOf(schedule:Schedule):SaleWindow{
 return schedule.availableDays.length||schedule.availableFrom||schedule.availableUntil?"days":"always";
}

export function applySaleWindow(schedule:Schedule,window:SaleWindow):Schedule{
 return window==="always"?{availableDays:[],availableFrom:"",availableUntil:""}:schedule;
}

// Fechas de venta: el formulario usa hora local del LOCAL (datetime-local sin zona); el API recibe instantes ISO.
// La conversión usa la zona horaria del local, nunca la del navegador.
const pad=(value:number)=>String(value).padStart(2,"0");
function wallClock(date:Date,timeZone?:string|null){
 const format=(zone?:string)=>new Intl.DateTimeFormat("en-CA",{timeZone:zone,hourCycle:"h23",year:"numeric",month:"2-digit",day:"2-digit",hour:"2-digit",minute:"2-digit"});
 let formatter:Intl.DateTimeFormat;
 try{formatter=format(timeZone||undefined)}catch{formatter=format()}
 const parts=Object.fromEntries(formatter.formatToParts(date).map(part=>[part.type,part.value]));
 return {year:Number(parts.year),month:Number(parts.month),day:Number(parts.day),hour:Number(parts.hour),minute:Number(parts.minute)};
}
const offsetAt=(instant:number,timeZone?:string|null)=>{
 const clock=wallClock(new Date(instant),timeZone);
 return Date.UTC(clock.year,clock.month-1,clock.day,clock.hour,clock.minute)-Math.floor(instant/60000)*60000;
};

/** «2026-10-18T23:59» en la zona del local → instante ISO (UTC). Vacío o inválido → null. */
export function zonedInputToIso(value:string,timeZone?:string|null):string|null{
 const match=/^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})/.exec(value);
 if(!match)return null;
 const [,year,month,day,hour,minute]=match.map(Number);
 const guess=Date.UTC(year,month-1,day,hour,minute);
 // Dos pasadas para cruzar bien un cambio de horario.
 let instant=guess-offsetAt(guess,timeZone);
 instant=guess-offsetAt(instant,timeZone);
 return new Date(instant).toISOString();
}

/** Instante ISO del API → «AAAA-MM-DDTHH:mm» en la zona del local, para el campo datetime-local. */
export function isoToZonedInput(value:string|null|undefined,timeZone?:string|null):string{
 if(!value)return "";
 const date=new Date(value);
 if(!Number.isFinite(date.getTime()))return "";
 const clock=wallClock(date,timeZone);
 return `${clock.year}-${pad(clock.month)}-${pad(clock.day)}T${pad(clock.hour)}:${pad(clock.minute)}`;
}

/** Activo pero fuera de su rango de venta (vencido o aún no empieza). Recibe instantes ISO; el listado usa `outsideDates` del API. */
export function isOutsideDates(from:string|null|undefined,until:string|null|undefined,now:number=Date.now()):boolean{
 const start=from?Date.parse(from):NaN,end=until?Date.parse(until):NaN;
 return (Number.isFinite(end)&&now>end)||(Number.isFinite(start)&&now<start);
}
