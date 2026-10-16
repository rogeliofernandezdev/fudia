import {z} from "zod";
import {zodResolver} from "@/shared/forms/zod-resolver";
import type {LocationDraft,ProfileDraft,RateDraft} from "./types";

export const locationSchema=z.object({
 name:z.string().trim().min(1,"Ingresa el nombre del local.").max(120,"El nombre no puede superar 120 caracteres."),
 address:z.string().trim().min(3,"Ingresa la dirección del local."),
 phone:z.string().trim().max(40,"El teléfono no puede superar 40 caracteres."),
 openingHours:z.string().trim().max(120,"El horario no puede superar 120 caracteres."),
 timezone:z.string().trim().min(3,"Indica la zona horaria."),
 fiscalProfileId:z.string().min(1,"Selecciona el perfil fiscal."),
});
export const locationResolver=zodResolver<LocationDraft>(locationSchema.passthrough());

export const profileSchema=z.object({
 country:z.string().min(2,"Selecciona el país."),
 currency:z.string().min(3,"Selecciona la moneda."),
 currencyPosition:z.enum(["before","after"]),
 taxName:z.string().trim().min(1,"Ingresa el nombre del impuesto.").max(30,"Máximo 30 caracteres."),
 taxPercent:z.string().trim().min(1,"Ingresa el porcentaje.").refine(value=>{const n=Number(value);return Number.isFinite(n)&&n>=0&&n<=100},"Usa un porcentaje entre 0 y 100."),
 taxIncluded:z.boolean(),
 default:z.boolean(),
});
export const profileResolver=zodResolver<ProfileDraft>(profileSchema.passthrough());

export const rateSchema=z.object({
 baseCurrency:z.string().min(3,"Selecciona la moneda origen."),
 quoteCurrency:z.string().min(3,"Selecciona la moneda destino."),
 rate:z.string().trim().regex(/^[0-9]+([.][0-9]+)?$/,"Ingresa una tasa numérica.").refine(value=>Number(value)>0,"La tasa debe ser mayor que cero."),
 effectiveAt:z.string().min(1,"Indica desde cuándo rige la tasa."),
 source:z.enum(["manual","provider"]),
 providerReference:z.string().trim().max(120,"Máximo 120 caracteres."),
}).superRefine((value,ctx)=>{
 if(value.baseCurrency===value.quoteCurrency)ctx.addIssue({code:"custom",path:["quoteCurrency"],message:"La moneda destino debe ser distinta de la origen."});
 if(value.source==="provider"&&!value.providerReference)ctx.addIssue({code:"custom",path:["providerReference"],message:"Indica la referencia del proveedor."});
});
export const rateResolver=zodResolver<RateDraft>(rateSchema);
