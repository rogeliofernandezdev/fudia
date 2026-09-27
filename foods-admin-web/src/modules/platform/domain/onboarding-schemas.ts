import {z} from "zod";

export const companySchema=z.object({
  legalName:z.string().trim().min(2,"Ingresa la razón social").max(180),
  tradeName:z.string().trim().min(2,"Ingresa el nombre comercial").max(180),
  taxId:z.string().trim().min(6,"La identificación fiscal es obligatoria").max(32),
  timezone:z.string().trim().min(3,"Selecciona la zona horaria"),
});

export const planSchema=z.object({
  planId:z.string().min(1,"Selecciona un plan"),
  billingCycle:z.enum(["monthly","annual"]),
  termsAccepted:z.boolean().refine(Boolean,"Debes aceptar las condiciones"),
});

export const fiscalSchema=z.object({
  country:z.string().min(2),
  currency:z.string().min(3),
  taxName:z.string().trim().min(2,"Ingresa el impuesto"),
  taxRate:z.string().refine(value=>Number(value)>=0&&Number(value)<=100,"Porcentaje inválido"),
  currencyPosition:z.enum(["before","after"]),
  taxIncluded:z.boolean(),
});

export const locationSchema=z.object({
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
