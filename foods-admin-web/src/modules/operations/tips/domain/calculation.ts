import type {TipPolicy} from "./types";
export function calculateTip(enabled:boolean,kind:string,value:string|number,base:number,decimals=2){
 const input=Number(value);
 if(!enabled||!Number.isFinite(input)||input<=0||!Number.isFinite(base)||base<=0)return 0;
 const amount=kind==="percentage"?base*input/100:input;
 return Math.round(amount*10**decimals)/10**decimals;
}
export function canCollectTip(policy:TipPolicy|undefined,channel:string,registered:boolean){return Boolean(!registered&&policy?.mode==="collect"&&policy.channels.includes(channel==="mostrador"?"recojo":channel))}
