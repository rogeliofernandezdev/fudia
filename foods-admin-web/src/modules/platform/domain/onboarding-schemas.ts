import {z} from "zod";
import type {FieldError,FieldErrors,Resolver} from "react-hook-form";
import type {PlatformOnboardingDraft} from "./types";

export const companySchema=z.object({
  legalName:z.string().trim().min(2,"Ingresa la razón social").max(180),
  tradeName:z.string().trim().min(2,"Ingresa el nombre comercial").max(180),
  taxId:z.string().trim().min(6,"La identificación fiscal es obligatoria").max(32),
});

export const planSchema=z.object({
  planId:z.string().min(1,"Selecciona un plan"),
  billingCycle:z.enum(["monthly","annual"]),
  termsAccepted:z.boolean().refine(Boolean,"Debes aceptar las condiciones"),
});

export const fiscalSchema=z.object({
  country:z.string().min(2,"Selecciona el país"),
  currency:z.string().min(3,"Selecciona la moneda"),
  taxName:z.string().trim().min(2,"Ingresa el impuesto"),
  taxRate:z.string().trim().min(1,"Ingresa el porcentaje").refine(value=>Number(value)>=0&&Number(value)<=100,"Porcentaje inválido"),
  currencyPosition:z.enum(["before","after"]),
  taxIncluded:z.boolean(),
});

export const locationSchema=z.object({
  timezone:z.string().trim().min(1,"Selecciona la zona horaria"),
  locationName:z.string().trim().min(2,"Ingresa el nombre del local"),
  address:z.string().trim().min(3,"Ingresa la dirección"),
  locationPhone:z.string().optional(),
  locationHours:z.string().optional(),
});

export const adminSchema=z.object({
  adminName:z.string().trim().min(2,"Ingresa el nombre del administrador"),
  adminEmail:z.string().email("Correo inválido"),
  adminPassword:z.string().min(8,"Mínimo 8 caracteres"),
});

const onboardingSchema=companySchema.extend(planSchema.shape).extend(fiscalSchema.shape).extend(locationSchema.shape).extend(adminSchema.shape);

/** Campos que valida cada paso del wizard, en el mismo orden de los pasos. */
export const onboardingStepFields:Array<Array<keyof PlatformOnboardingDraft>>=[
  Object.keys(companySchema.shape),
  Object.keys(planSchema.shape),
  Object.keys(fiscalSchema.shape),
  Object.keys(locationSchema.shape),
  Object.keys(adminSchema.shape),
] as Array<Array<keyof PlatformOnboardingDraft>>;

export const onboardingResolver:Resolver<PlatformOnboardingDraft>=(values)=>{
  const result=onboardingSchema.safeParse(values);
  if(result.success)return {values,errors:{}};
  const errors:Record<string,FieldError>={};
  for(const issue of result.error.issues){
    const field=issue.path[0];
    if(typeof field==="string"&&!errors[field]){
      errors[field]={type:issue.code,message:issue.message};
    }
  }
  return {values:{},errors:errors as FieldErrors<PlatformOnboardingDraft>};
};
