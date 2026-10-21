import {z} from "zod";
import {zodResolver} from "@/shared/forms/zod-resolver";
import type {CustomerDraft} from "./types";

const documentRules:Record<string,{pattern:RegExp;message:string}>={
  DNI:{pattern:/^\d{8}$/,message:"El DNI tiene 8 dígitos."},
  RUC:{pattern:/^\d{11}$/,message:"El RUC tiene 11 dígitos."},
  CE:{pattern:/^[A-Za-z0-9]{9,12}$/,message:"El carné de extranjería tiene entre 9 y 12 caracteres."},
};

export const addressSchema=z.object({
  label:z.string().trim().max(60,"La etiqueta no puede superar 60 caracteres."),
  address:z.string().trim().min(1,"Ingresa la dirección."),
  district:z.string().trim(),
  city:z.string().trim(),
  reference:z.string().trim().max(200,"La referencia no puede superar 200 caracteres."),
});

export const customerSchema=z.object({
  customerType:z.enum(["person","company"]),
  displayName:z.string().trim().min(1,"Ingresa el nombre o razón social.").max(180,"El nombre no puede superar 180 caracteres."),
  documentType:z.string(),
  documentNumber:z.string().trim(),
  phone:z.string().trim().max(40,"El teléfono no puede superar 40 caracteres."),
  email:z.string().trim().refine(value=>value===""||/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value),"Ingresa un correo válido."),
  addresses:z.array(addressSchema),
}).superRefine((value,ctx)=>{
  const rule=documentRules[value.documentType];
  if(value.documentType&&!value.documentNumber)ctx.addIssue({code:"custom",path:["documentNumber"],message:"Ingresa el número del documento."});
  else if(rule&&value.documentNumber&&!rule.pattern.test(value.documentNumber))ctx.addIssue({code:"custom",path:["documentNumber"],message:rule.message});
});

export const customerResolver=zodResolver<CustomerDraft>(customerSchema.passthrough());
