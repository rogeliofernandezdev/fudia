import {z} from "zod";
import type {FieldError,FieldErrors,Resolver} from "react-hook-form";
import type {InventoryAdjustmentDraft} from "./types";

const schema=z.object({
  inventoryItemId:z.string().trim().min(1,"Selecciona un artículo."),
  movementType:z.enum(["entry","exit"]),
  reason:z.enum(["surplus_adjustment","shortage_adjustment","waste","expiration","other_exit"]),
  quantity:z.string().trim().refine(value=>{
    const parsed=Number(value);
    return value!==""&&Number.isFinite(parsed)&&parsed>0;
  },"Ingresa una cantidad mayor que cero."),
  observation:z.string().max(240,"La observación no puede superar 240 caracteres."),
}).superRefine((value,ctx)=>{
  if(value.movementType==="entry"&&value.reason!=="surplus_adjustment"){
    ctx.addIssue({code:"custom",path:["reason"],message:"Selecciona un motivo de entrada válido."});
  }
  if(value.movementType==="exit"&&!["shortage_adjustment","waste","expiration","other_exit"].includes(value.reason)){
    ctx.addIssue({code:"custom",path:["reason"],message:"Selecciona un motivo de salida válido."});
  }
});

export const inventoryAdjustmentResolver:Resolver<InventoryAdjustmentDraft>=(values)=>{
  const result=schema.safeParse(values);
  if(result.success)return {values:result.data,errors:{}};

  const errors:Record<string,FieldError>={};
  for(const issue of result.error.issues){
    const field=issue.path[0];
    if(typeof field==="string"&&!errors[field]){
      errors[field]={type:issue.code,message:issue.message};
    }
  }
  return {values:{},errors:errors as FieldErrors<InventoryAdjustmentDraft>};
};
