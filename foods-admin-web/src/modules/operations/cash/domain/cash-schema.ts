import {z} from "zod";
import type {FieldError,FieldErrors,Resolver} from "react-hook-form";
import type {CashMovementDraft,CloseCashShiftDraft,OpenCashShiftDraft} from "./types";

function moneyField(message:string,{allowZero=false}:{allowZero?:boolean}={}){
  return z.string().trim().refine(value=>{
    const parsed=Number(value);
    return value!==""&&Number.isFinite(parsed)&&(allowZero?parsed>=0:parsed>0);
  },message);
}

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

const openSchema=z.object({
  openingAmount:moneyField("Ingresa un fondo inicial válido.",{allowZero:true}),
  note:z.string().max(240,"La observación no puede superar 240 caracteres."),
});

const movementSchema=z.object({
  movementType:z.enum(["income","expense"]),
  amount:moneyField("Ingresa un monto mayor que cero."),
  reason:z.string().trim().min(1,"Indica el motivo del movimiento.").max(120,"El motivo no puede superar 120 caracteres."),
  note:z.string().max(240,"La observación no puede superar 240 caracteres."),
});

const closeSchema=z.object({
  countedAmount:moneyField("Ingresa el efectivo contado.",{allowZero:true}),
  note:z.string().max(240,"La observación no puede superar 240 caracteres."),
});

export const openCashShiftResolver=resolverFor<OpenCashShiftDraft>(openSchema);
export const cashMovementResolver=resolverFor<CashMovementDraft>(movementSchema);
export const closeCashShiftResolver=resolverFor<CloseCashShiftDraft>(closeSchema);
