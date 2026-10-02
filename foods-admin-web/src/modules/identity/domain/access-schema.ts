import {z} from "zod";
import {zodResolver} from "@/shared/forms/zod-resolver";
import type {RoleDraft,UserDraft} from "./types";

export const userSchema=z.object({
  id:z.string().optional(),
  fullName:z.string().trim().min(1,"Ingresa el nombre completo.").max(180,"El nombre no puede superar 180 caracteres."),
  email:z.string().trim().min(1,"Ingresa el correo electrónico.").email("Ingresa un correo válido."),
  password:z.string(),
  assignments:z.array(z.object({
    roleId:z.string().min(1,"Selecciona un rol."),
    locationId:z.string().min(1,"Selecciona un local."),
  })).min(1,"Agrega al menos un acceso."),
}).superRefine((value,ctx)=>{
  if(!value.id&&value.password.length<8)ctx.addIssue({code:"custom",path:["password"],message:"La contraseña temporal debe tener al menos 8 caracteres."});
  if(value.id&&value.password&&value.password.length<8)ctx.addIssue({code:"custom",path:["password"],message:"La nueva contraseña debe tener al menos 8 caracteres."});
  const seen=new Set<string>();
  value.assignments.forEach((item,index)=>{
    const key=item.roleId+"@"+item.locationId;
    if(item.roleId&&item.locationId&&seen.has(key))ctx.addIssue({code:"custom",path:["assignments",index,"roleId"],message:"Este rol ya está asignado en ese local."});
    seen.add(key);
  });
});
export const userResolver=zodResolver<UserDraft>(userSchema.passthrough());

export const roleSchema=z.object({
  name:z.string().trim().min(1,"Ingresa el nombre del rol.").max(100,"El nombre no puede superar 100 caracteres."),
  description:z.string().trim().min(1,"Describe las responsabilidades del rol.").max(240,"La descripción no puede superar 240 caracteres."),
});
export const roleResolver=zodResolver<RoleDraft>(roleSchema.passthrough());
