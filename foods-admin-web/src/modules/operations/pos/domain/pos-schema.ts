import {z} from "zod";
import type {FieldError,FieldErrors,Resolver} from "react-hook-form";
import type {PaymentDraft,PaymentMethodOption,RefundDraft} from "./types";
import {calculateTip,tipInputValue} from "../../tips/domain/calculation";
import type {TipPolicy} from "../../tips/domain/types";

function resolverFor<T extends Record<string,unknown>>(schema:z.ZodType<T>):Resolver<T>{
  return values=>{
    const result=schema.safeParse(values);
    if(result.success)return {values:result.data,errors:{}};
    const errors:Record<string,FieldError>={};
    for(const issue of result.error.issues){
      const field=issue.path[0];
      if(typeof field==="string"&&!errors[field])errors[field]={type:issue.code,message:issue.message};
    }
    return {values:{},errors:errors as FieldErrors<T>};
  };
}

const positiveMoney=z.string().trim().refine(value=>{
  const parsed=Number(value);
  return value!==""&&Number.isFinite(parsed)&&parsed>0;
},"Ingresa un monto mayor que cero.");

const paymentSchema=z.object({
  method:z.string().trim().min(1,"Selecciona un medio de pago."),
  amount:positiveMoney,
  reference:z.string().max(120,"La referencia no puede superar 120 caracteres."),
  receivedAmount:z.string().optional(),
  tipEnabled:z.boolean().optional(),
  tipKind:z.enum(["percentage","fixed"]).optional(),
  tipValue:z.string().optional(),
});

const refundSchema=z.object({
  amount:positiveMoney,
  reason:z.string().trim().min(1,"Indica el motivo de la devolución.").max(120,"El motivo no puede superar 120 caracteres."),
  note:z.string().max(240,"La observación no puede superar 240 caracteres."),
});

export const paymentResolver=resolverFor<PaymentDraft>(paymentSchema);
export function paymentResolverFor(methods:PaymentMethodOption[],remaining:number,policy?:TipPolicy,base=remaining){
  return resolverFor<PaymentDraft>(paymentSchema.superRefine((draft,ctx)=>{
    const amount=Number(draft.amount);
    if(amount>remaining)ctx.addIssue({code:"custom",path:["amount"],message:"El monto no puede superar el saldo pendiente."});
    const kind=draft.tipKind??"percentage";
    const value=tipInputValue(kind,policy,draft.tipValue);
    const tip=calculateTip(Boolean(draft.tipEnabled),kind,value,base,policy?.decimals??2);
    const scale=10**(kind==="fixed"?(policy?.decimals??2):2);
    if(draft.tipEnabled&&(policy?.mode!=="collect"||tip<=0||value>1e9||Math.abs(value*scale-Math.round(value*scale))>.00001||(kind==="percentage"&&value>100)||(kind==="fixed"&&!policy?.allowFixed))){ctx.addIssue({code:"custom",path:[kind==="fixed"?"tipValue":"tipKind"],message:kind==="fixed"?"Ingresa una propina válida en la moneda del local.":"El local no tiene un porcentaje de propina válido."})}
    if(!methods.some(method=>method.code===draft.method&&method.affectsCash))return;
    const received=Number(draft.receivedAmount);
    if(!draft.receivedAmount?.trim()||!Number.isFinite(received)||received<=0){
      ctx.addIssue({code:"custom",path:["receivedAmount"],message:"Ingresa el efectivo recibido."});
    }else if(received<amount+(draft.tipEnabled?tip:0)){
      ctx.addIssue({code:"custom",path:["receivedAmount"],message:"El efectivo recibido debe cubrir el monto del cobro."});
    }
  }));
}

export function cashChange(amount:string|undefined,received:string|undefined){
  if(!amount?.trim()||!received?.trim())return null;
  const charged=Number(amount),tendered=Number(received);
  return Number.isFinite(charged)&&charged>0&&Number.isFinite(tendered)&&tendered>=charged?Math.round((tendered-charged)*100)/100:null;
}
/**
 * Montos rápidos de efectivo, como en los POS de mostrador: el importe exacto y los
 * siguientes redondeos de la serie 1-2-5 a partir de su orden de magnitud. Es aritmética
 * genérica, no un catálogo de billetes; sirve para cualquier moneda.
 */
export function cashSuggestions(total:number,decimals=2,limit=4){
  if(!Number.isFinite(total)||total<=0)return [];
  const scale=10**decimals,exact=Math.ceil(total*scale-1e-6)/scale;
  const values=[exact];
  for(let power=10**Math.floor(Math.log10(exact));values.length<limit&&power<=exact*1000;power*=10){
    for(const factor of [1,2,5]){
      const rounded=Math.ceil(exact/(power*factor))*power*factor;
      if(rounded>exact&&!values.includes(rounded)&&values.length<limit)values.push(rounded);
    }
  }
  return values.sort((a,b)=>a-b);
}
export const refundResolver=resolverFor<RefundDraft>(refundSchema);
