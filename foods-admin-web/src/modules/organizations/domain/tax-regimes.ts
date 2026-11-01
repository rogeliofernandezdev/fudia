import type {TaxRegime,TaxRegimeDraft} from "./types";

// The API stores exact fractions with six decimals and an exclusive end date;
// the form works with human percentages and an inclusive «hasta» date.
export type TaxRegimeInput={name:string;validFrom:string;validTo:string|null;components:{code:string;name:string;rate:string}[]};
export type TaxRegimeStatus="current"|"scheduled"|"ended";

export function percentToRate(percent:string){
 const [whole,decimals=""]=percent.trim().split(".");
 const digits=(whole.replace(/^0+(?=\d)/,"")+decimals.padEnd(4,"0").slice(0,4)).padStart(7,"0");
 return `${digits.slice(0,-6)||"0"}.${digits.slice(-6)}`.replace(/^0+(?=\d\.)/,"");
}
export function rateToPercent(rate:string){
 const value=Math.round(Number(rate)*1_000_000)/10_000;
 return String(Number(value.toFixed(4)));
}
export function shiftDate(iso:string,days:number){
 const [year,month,day]=iso.split("-").map(Number);
 return new Date(Date.UTC(year,month-1,day+days)).toISOString().slice(0,10);
}
export function componentCode(name:string,used:Set<string>){
 const base=name.normalize("NFD").replace(/[̀-ͯ]/g,"").toLowerCase().replace(/[^a-z0-9]+/g,"_").replace(/^_+|_+$/g,"").slice(0,16);
 const safe=/^[a-z]/.test(base)?base:`t_${base}`.replace(/_+$/,"");
 let code=safe,suffix=2;
 while(used.has(code))code=`${safe}_${suffix++}`;
 used.add(code);
 return code;
}
export function draftFromRegime(regime:TaxRegime):TaxRegimeDraft{
 return {id:regime.id,name:regime.name,validFrom:regime.validFrom,validUntil:regime.validTo?shiftDate(regime.validTo,-1):"",components:regime.components.map(component=>({name:component.name,percent:rateToPercent(component.rate)}))};
}
export function regimeInput(draft:TaxRegimeDraft):TaxRegimeInput{
 const used=new Set<string>();
 return {
  name:draft.name.trim(),validFrom:draft.validFrom,validTo:draft.validUntil?shiftDate(draft.validUntil,1):null,
  components:draft.components.map(component=>({code:componentCode(component.name,used),name:component.name.trim(),rate:percentToRate(component.percent)})),
 };
}
export function totalPercent(components:{percent:string}[]){
 const thousandths=components.reduce((sum,component)=>sum+Math.round((Number(component.percent)||0)*10_000),0);
 return thousandths/10_000;
}
// `today` is the business date returned by the API for the session location.
export function regimeStatus(regime:TaxRegime,today:string):TaxRegimeStatus{
 if(regime.validFrom>today)return "scheduled";
 if(regime.validTo&&regime.validTo<=today)return "ended";
 return "current";
}
