import {z} from "zod";
import type {FieldError,FieldErrors,Resolver} from "react-hook-form";
import type {CashMovementDraft,CashOperationDraft,CashRegisterDraft,CloseCashShiftDraft,OpenCashShiftDraft} from "./types";

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

const registerSchema=z.object({
  name:z.string().trim().min(1,"Ingresa el nombre de la caja.").max(80,"El nombre no puede superar 80 caracteres."),
  blindClose:z.boolean(),
});

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

const operationSchema=z.object({
  operationType:z.enum(["cash_pull","deposit","transfer"]),
  targetShiftId:z.string(),
  amount:moneyField("Ingresa un monto mayor que cero."),
  reason:z.string().trim().min(1,"Indica el motivo de la operación.").max(120,"El motivo no puede superar 120 caracteres."),
  note:z.string().max(240,"La observación no puede superar 240 caracteres."),
}).superRefine((value,ctx)=>{
  if(value.operationType==="transfer"&&!value.targetShiftId.trim()){
    ctx.addIssue({code:"custom",path:["targetShiftId"],message:"Selecciona la caja de destino."});
  }
});

const closeSchema=z.object({
  countedAmount:z.string(),
  note:z.string().max(240,"La observación no puede superar 240 caracteres."),
  counts:z.array(z.object({
    denomination:z.string(),
    quantity:z.string(),
  })),
}).superRefine((value,ctx)=>{
  const validCounts=value.counts.filter(line=>Number(line.denomination)>0&&Number(line.quantity)>0);
  const counted=Number(value.countedAmount);
  if(!validCounts.length&&!(value.countedAmount!==""&&Number.isFinite(counted)&&counted>=0)){
    ctx.addIssue({code:"custom",path:["countedAmount"],message:"Ingresa el efectivo contado o usa el conteo por denominaciones."});
  }
});

export const cashRegisterResolver=resolverFor<CashRegisterDraft>(registerSchema);
export const openCashShiftResolver=resolverFor<OpenCashShiftDraft>(openSchema);
export const cashMovementResolver=resolverFor<CashMovementDraft>(movementSchema);
export const closeCashShiftResolver=resolverFor<CloseCashShiftDraft>(closeSchema);
export const cashOperationResolver=resolverFor<CashOperationDraft>(operationSchema);
