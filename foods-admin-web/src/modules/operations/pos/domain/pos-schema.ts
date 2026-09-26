import {z} from "zod";
import type {FieldError,FieldErrors,Resolver} from "react-hook-form";
import type {PaymentDraft,RefundDraft} from "./types";

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
  method:z.enum(["cash","card","transfer","other"]),
  amount:positiveMoney,
  reference:z.string().max(120,"La referencia no puede superar 120 caracteres."),
});

const refundSchema=z.object({
  amount:positiveMoney,
  reason:z.string().trim().min(1,"Indica el motivo de la devolución.").max(120,"El motivo no puede superar 120 caracteres."),
  note:z.string().max(240,"La observación no puede superar 240 caracteres."),
});

export const paymentResolver=resolverFor<PaymentDraft>(paymentSchema);
export const refundResolver=resolverFor<RefundDraft>(refundSchema);
