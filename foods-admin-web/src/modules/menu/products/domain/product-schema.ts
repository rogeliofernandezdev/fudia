import {z} from "zod";
import {zodResolver} from "@/shared/forms/zod-resolver";
import type {CategoryDraft,ProductDraft} from "./types";

const money=/^[0-9]+([.][0-9]{1,2})?$/;

export const productSchema=z.object({
  id:z.string().optional(),
  initialPortionQuantity:z.string().optional(),
  name:z.string().trim().min(1,"Ingresa el nombre del producto.").max(160,"El nombre no puede superar 160 caracteres."),
  price:z.string().trim().regex(money,"Ingresa un precio válido, por ejemplo 12.50."),
  costPrice:z.string().trim().refine(value=>value===""||money.test(value),"Ingresa un costo válido o déjalo vacío."),
  prepMinutes:z.string().trim().refine(value=>value===""||(/^\d+$/.test(value)&&Number(value)<=600),"Usa minutos enteros, hasta 600."),
  categoryId:z.string(),
  description:z.string(),
  serviceDestination:z.enum(["kitchen","bar","direct"]).default("kitchen"),
  quantityControl:z.enum(["none","portions","inventory"]),
  allergens:z.array(z.string()),
  featured:z.boolean(),
}).superRefine((value,ctx)=>{
  if(!value.id&&value.quantityControl==="portions"){
    const quantity=value.initialPortionQuantity?.trim()??"";
    if(!/^\d+$/.test(quantity)||Number(quantity)<1||Number(quantity)>2147483647){
      ctx.addIssue({code:"custom",path:["initialPortionQuantity"],message:"Ingresa una cantidad entera mayor que cero, hasta 2147483647."});
    }
  }
});
export const productResolver=zodResolver<ProductDraft>(productSchema.passthrough());

export const categorySchema=z.object({
  name:z.string().trim().min(1,"Ingresa el nombre de la categoría.").max(120,"El nombre no puede superar 120 caracteres."),
  productScope:z.enum(["prepared","retail","both"]),
  sortOrder:z.coerce.number().int("Usa un número entero.").min(0,"El orden no puede ser negativo."),
});
export const categoryResolver=zodResolver<CategoryDraft>(categorySchema.passthrough());
