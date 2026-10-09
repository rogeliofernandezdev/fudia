import {z} from "zod";
import type {FieldError,FieldErrors,Resolver} from "react-hook-form";
import type {MyProfileDraft} from "./types";

export const profileSchema=z.object({
  fullName:z.string().trim().min(1,"Ingresa tu nombre completo.").max(180,"El nombre no puede superar 180 caracteres."),
  currentPassword:z.string(),
  newPassword:z.string(),
}).superRefine((value,ctx)=>{
  if(value.newPassword&&value.newPassword.length<8){
    ctx.addIssue({code:"custom",path:["newPassword"],message:"La nueva contraseña debe tener al menos 8 caracteres."});
  }
  if(value.newPassword&&!value.currentPassword){
    ctx.addIssue({code:"custom",path:["currentPassword"],message:"Ingresa tu contraseña actual para definir una nueva."});
  }
});

export const profileResolver:Resolver<MyProfileDraft>=(values)=>{
  const result=profileSchema.safeParse(values);
  if(result.success)return {values:result.data,errors:{}};
  const errors:Record<string,FieldError>={};
  for(const issue of result.error.issues){
    const field=issue.path[0];
    if(typeof field==="string"&&!errors[field]){
      errors[field]={type:issue.code,message:issue.message};
    }
  }
  return {values:{},errors:errors as FieldErrors<MyProfileDraft>};
};
