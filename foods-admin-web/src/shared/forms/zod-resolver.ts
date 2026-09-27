import type {FieldError,FieldErrors,FieldValues,Resolver} from "react-hook-form";
import type {ZodType} from "zod";

/**
 * Resolver de React Hook Form a partir de un esquema Zod.
 * Devuelve los valores originales (no los transformados) para que el
 * formulario conserve strings en inputs numéricos; conserva el primer
 * mensaje por campo.
 */
export function zodResolver<T extends FieldValues>(schema:ZodType):Resolver<T>{
  return (values)=>{
    const result=schema.safeParse(values);
    if(result.success)return {values,errors:{}};
    const errors:Record<string,FieldError>={};
    for(const issue of result.error.issues){
      const field=issue.path.map(String).join(".");
      if(field&&!errors[field])errors[field]={type:issue.code,message:issue.message};
    }
    return {values:{} as T,errors:errors as FieldErrors<T>};
  };
}
