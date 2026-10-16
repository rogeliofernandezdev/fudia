import {z} from "zod";
import type {FieldError,FieldErrors,Resolver} from "react-hook-form";
import type {ExpenseDraft} from "./expense";
export const expenseSchema=z.object({
 categoryId:z.string().trim().min(1,"Selecciona una categoría."),
 description:z.string().trim().min(1,"Ingresa una descripción.").max(180,"La descripción no puede superar 180 caracteres."),
 amount:z.string().trim().regex(/^[0-9]{1,12}([.][0-9]{1,2})?$/,"Ingresa un importe válido.").refine(value=>Number(value)>0,"El importe debe ser mayor que cero."),
 paymentMethod:z.string().trim().min(1,"Selecciona un medio de pago."),
 businessDate:z.string().trim().min(1,"Selecciona la fecha."),
 reference:z.string().trim().max(120,"La referencia no puede superar 120 caracteres."),
 notes:z.string().trim().max(500,"Las notas no pueden superar 500 caracteres."),
});
export const expenseResolver:Resolver<ExpenseDraft>=values=>{
 const result=expenseSchema.safeParse(values);
 if(result.success)return {values:result.data,errors:{}};
 const errors:Record<string,FieldError>={};
 for(const issue of result.error.issues){const key=issue.path[0];if(typeof key==="string"&&!errors[key])errors[key]={type:issue.code,message:issue.message}}
 return {values:{},errors:errors as FieldErrors<ExpenseDraft>};
};
