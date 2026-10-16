import {z} from "zod";
import {zodResolver} from "@/shared/forms/zod-resolver";
import type {SubscriptionPlanDraft} from "./types";

const money=/^[0-9]+([.][0-9]{1,2})?$/;
const optionalInt=(label:string)=>z.string().trim().refine(value=>value===""||(/^\d+$/.test(value)&&Number(value)>=1),`${label}: usa un entero mayor que cero o déjalo vacío.`);

export const planSchema=z.object({
 code:z.string().trim().min(2,"El código necesita al menos 2 caracteres.").max(40,"El código no puede superar 40 caracteres.").regex(/^[a-z0-9_-]+$/,"Usa minúsculas, números, guion o guion bajo."),
 name:z.string().trim().min(1,"Ingresa el nombre del plan.").max(120,"El nombre no puede superar 120 caracteres."),
 description:z.string().trim().max(500,"La descripción no puede superar 500 caracteres."),
 currency:z.string().min(3,"Selecciona la moneda."),
 monthlyPrice:z.string().trim().regex(money,"Ingresa un precio mensual válido."),
 annualPrice:z.string().trim().regex(money,"Ingresa un precio anual válido."),
 trialDays:z.coerce.number().int("Usa días enteros.").min(0,"No puede ser negativo.").max(365,"Máximo 365 días."),
 maxLocations:optionalInt("Locales"),
 maxUsers:optionalInt("Usuarios"),
 termsVersion:z.string().trim().min(1,"Ingresa la versión de condiciones.").max(80,"Máximo 80 caracteres."),
 moduleKeys:z.array(z.string()).min(1,"Incluye al menos un módulo."),
 active:z.boolean(),
});
export const planResolver=zodResolver<SubscriptionPlanDraft>(planSchema.passthrough());

export const subscriptionChangeSchema=z.object({
 planId:z.string().min(1,"Selecciona un plan."),
 billingCycle:z.enum(["monthly","annual"]),
 status:z.enum(["trial","active","past_due","cancelled"]),
 autoRenew:z.boolean(),
 termsAccepted:z.boolean(),
});

export const subscriptionPaymentSchema=z.object({
 amount:z.string().trim().regex(money,"Ingresa un monto válido."),
 currency:z.string(),
 status:z.enum(["pending","paid","failed","refunded"]),
 provider:z.string().trim().min(1,"Indica el proveedor o canal del pago."),
 externalReference:z.string().trim().max(120,"Máximo 120 caracteres."),
 paidAt:z.string(),
}).superRefine((value,ctx)=>{
 if(value.status==="paid"&&!value.paidAt)ctx.addIssue({code:"custom",path:["paidAt"],message:"Indica cuándo se realizó el pago."});
});
