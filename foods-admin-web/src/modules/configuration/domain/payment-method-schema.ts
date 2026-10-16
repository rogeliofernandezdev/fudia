import {z} from "zod";
import {zodResolver} from "@/shared/forms/zod-resolver";
import type {PaymentMethodDraft} from "./payment-method";

export const paymentMethodSchema=z.object({
 code:z.string().trim().min(1,"Ingresa el código interno.").regex(/^[a-z0-9][a-z0-9_-]{0,39}$/,"Empieza con letra o número y usa solo minúsculas, números, guiones o guion bajo."),
 name:z.string().trim().min(1,"Ingresa el nombre del medio.").max(80,"El nombre no puede superar 80 caracteres."),
 description:z.string().trim().max(180,"La descripción no puede superar 180 caracteres."),
 salesEnabled:z.boolean(),
 expensesEnabled:z.boolean(),
 affectsCash:z.boolean(),
 sortOrder:z.coerce.number().int("Usa un número entero.").min(0,"El orden no puede ser negativo.").max(9999,"El orden máximo es 9999."),
}).superRefine((value,ctx)=>{
 if(!value.salesEnabled&&!value.expensesEnabled)ctx.addIssue({code:"custom",path:["salesEnabled"],message:"Habilita el medio al menos para Ventas o para Gastos."});
});
export const paymentMethodResolver=zodResolver<PaymentMethodDraft>(paymentMethodSchema);
