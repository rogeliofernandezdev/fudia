import {z} from "zod";
import {zodResolver} from "@/shared/forms/zod-resolver";
import type {InventorySettingsDraft} from "./types";

const stockLevel=z.string().trim().refine(value=>value!==""&&Number.isFinite(Number(value))&&Number(value)>=0,"Ingresa una cantidad igual o mayor que cero.");
const schema=z.object({
  inventoryItemId:z.string().trim().min(1,"Selecciona un artículo."),
  minimumStock:stockLevel,
  optimalStock:stockLevel,
});

const wholeSchema=schema.superRefine((value,context)=>{
  for(const field of ["minimumStock","optimalStock"] as const){
    const amount=Number(value[field]);
    if(Number.isFinite(amount)&&amount>=0&&!Number.isInteger(amount))context.addIssue({code:"custom",path:[field],message:"Ingresa una cantidad entera para esta unidad."});
  }
});
const decimalResolver=zodResolver<InventorySettingsDraft>(schema),wholeResolver=zodResolver<InventorySettingsDraft>(wholeSchema);
export const inventorySettingsResolver=(countable:boolean)=>countable?wholeResolver:decimalResolver;
