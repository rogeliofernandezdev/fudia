import {z} from "zod";
import {zodResolver} from "@/shared/forms/zod-resolver";

export type LoginDraft={email:string;password:string};
export const loginSchema=z.object({
  email:z.string().trim().min(1,"Ingresa tu correo electrónico.").email("Ingresa un correo válido."),
  password:z.string().min(1,"Ingresa tu contraseña."),
});
export const loginResolver=zodResolver<LoginDraft>(loginSchema);
