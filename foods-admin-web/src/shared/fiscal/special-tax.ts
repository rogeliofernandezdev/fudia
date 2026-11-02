// Reduced IGV set by law for a country (Peru: MYPE restaurants and hotels). Rates are exact
// fractions ("0.105000"); validTo is an exclusive end date, as the API sends it.
export type SpecialTaxRate={id:string;validFrom:string;validTo:string|null;rate:string;legalBasis:string};
export type SpecialTaxRegime={code:string;country:string;name:string;rates:SpecialTaxRate[]};
export type SpecialTaxState="current"|"expiring"|"expired";
export type SpecialTaxStatus={state:SpecialTaxState;rate:string|null;rateValidTo:string|null;coverageEnd:string|null};
export type RatePeriodState="current"|"scheduled"|"ended";

export const SPECIAL_TAX_WARN_DAYS=30;
// Legal dates read in full: 31/12/2027.
export const fullDate:Intl.DateTimeFormatOptions={day:"2-digit",month:"2-digit",year:"numeric"};

export function shiftDate(iso:string,days:number){
 const [year,month,day]=iso.split("-").map(Number);
 return new Date(Date.UTC(year,month-1,day+days)).toISOString().slice(0,10);
}

// "0.105000" → "10,5 %" with the caller's regional number format.
export function ratePercentLabel(rate:string,format:(value:number)=>string){
 return `${format(Math.round(Number(rate)*1_000_000)/10_000)} %`;
}
export function rateToPercent(rate:string){
 return String(Math.round(Number(rate)*1_000_000)/10_000);
}
// Typed percent → exact fraction with six decimals, without float drift.
export function percentToRate(percent:string){
 const [whole,decimals=""]=percent.trim().split(".");
 const digits=(whole.replace(/^0+(?=\d)/,"")+decimals.padEnd(4,"0").slice(0,4)).padStart(7,"0");
 return `${digits.slice(0,-6)||"0"}.${digits.slice(-6)}`.replace(/^0+(?=\d\.)/,"");
}

function rateOn(regime:SpecialTaxRegime,date:string){
 return regime.rates.find(rate=>rate.validFrom<=date&&(!rate.validTo||date<rate.validTo));
}

// Mirrors the API: follows the rates chained from today to find when the
// special rate stops applying, and warns SPECIAL_TAX_WARN_DAYS before.
export function specialTaxStatusOn(regime:SpecialTaxRegime,today:string):SpecialTaxStatus{
 const current=rateOn(regime,today);
 if(!current)return {state:"expired",rate:null,rateValidTo:null,coverageEnd:null};
 let end=current.validTo;
 for(let next=end?rateOn(regime,end):undefined;end&&next;next=end?rateOn(regime,end):undefined)end=next.validTo;
 const state=end&&end<=shiftDate(today,SPECIAL_TAX_WARN_DAYS)?"expiring":"current";
 return {state,rate:current.rate,rateValidTo:current.validTo,coverageEnd:end};
}

export function ratePeriodState(rate:SpecialTaxRate,today:string):RatePeriodState{
 if(rate.validFrom>today)return "scheduled";
 if(rate.validTo&&rate.validTo<=today)return "ended";
 return "current";
}

// Last day the special rate applies (the exclusive end minus one day).
export function lastDay(validTo:string){return shiftDate(validTo,-1)}

// "2026" when the period is a whole calendar year; otherwise its first and
// last day, or only its start when it has no end.
export function ratePeriodLabel(rate:Pick<SpecialTaxRate,"validFrom"|"validTo">,day:(iso:string)=>string){
 const year=rate.validFrom.slice(0,4);
 if(rate.validFrom===`${year}-01-01`&&rate.validTo===`${Number(year)+1}-01-01`)return year;
 return rate.validTo?`${day(rate.validFrom)} – ${day(lastDay(rate.validTo))}`:`Desde ${day(rate.validFrom)}`;
}

export function regimesForCountry(options:SpecialTaxRegime[],country:string){
 return options.filter(option=>option.country===country);
}
