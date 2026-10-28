import {z} from "zod";
import type {FieldError,FieldErrors,Resolver} from "react-hook-form";
import type {PaymentDraft,PaymentMethodOption,RefundDraft} from "./types";

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
});

const refundSchema=z.object({
  amount:positiveMoney,
  reason:z.string().trim().min(1,"Indica el motivo de la devolución.").max(120,"El motivo no puede superar 120 caracteres."),
  note:z.string().max(240,"La observación no puede superar 240 caracteres."),
});

export const paymentResolver=resolverFor<PaymentDraft>(paymentSchema);
export function paymentResolverFor(methods:PaymentMethodOption[],remaining:number){
  return resolverFor<PaymentDraft>(paymentSchema.superRefine((draft,ctx)=>{
    const amount=Number(draft.amount);
    if(amount>remaining)ctx.addIssue({code:"custom",path:["amount"],message:"El monto no puede superar el saldo pendiente."});
    if(!methods.some(method=>method.code===draft.method&&method.affectsCash))return;
    const received=Number(draft.receivedAmount);
    if(!draft.receivedAmount?.trim()||!Number.isFinite(received)||received<=0){
      ctx.addIssue({code:"custom",path:["receivedAmount"],message:"Ingresa el efectivo recibido."});
    }else if(received<amount){
      ctx.addIssue({code:"custom",path:["receivedAmount"],message:"El efectivo recibido debe cubrir el monto del cobro."});
    }
  }));
}

export function cashChange(amount:string|undefined,received:string|undefined){
  if(!amount?.trim()||!received?.trim())return null;
  const charged=Number(amount),tendered=Number(received);
  return Number.isFinite(charged)&&charged>0&&Number.isFinite(tendered)&&tendered>=charged?Math.round((tendered-charged)*100)/100:null;
}
export const refundResolver=resolverFor<RefundDraft>(refundSchema);
