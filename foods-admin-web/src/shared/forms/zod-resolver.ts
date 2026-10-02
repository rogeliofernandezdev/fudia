import type {FieldError,FieldErrors,FieldValues,Resolver} from "react-hook-form";
import type {ZodType} from "zod";

/**
 * Resolver de React Hook Form a partir de un esquema Zod.
 * Devuelve los valores originales (no los transformados) para que el
 * formulario conserve strings en inputs numéricos; conserva el primer
 * mensaje por campo y anida los errores de arreglos (`items.0.quantity`)
 * con la misma forma que espera `formState.errors`.
 */
export function zodResolver<T extends FieldValues>(schema:ZodType):Resolver<T>{
  return (values)=>{
    const result=schema.safeParse(values);
    if(result.success)return {values,errors:{}};
    const errors:Record<string,unknown>={};
    for(const issue of result.error.issues){
      if(issue.path.length)assign(errors,issue.path.map(segment=>typeof segment==="symbol"?String(segment):segment),{type:issue.code,message:issue.message});
    }
    return {values:{},errors:errors as FieldErrors<T>};
  };
}

function assign(target:Record<string,unknown>,path:Array<string|number>,error:FieldError){
  let node=target;
  path.forEach((segment,index)=>{
    const key=String(segment);
    const last=index===path.length-1;
    const current=node[key];
    if(last){
      if(current===undefined)node[key]=error;
      else if(typeof current==="object"&&current!==null&&!("message" in current))Object.assign(current,error);
      return;
    }
    if(typeof current!=="object"||current===null)node[key]=typeof path[index+1]==="number"?[]:{};
    node=node[key] as Record<string,unknown>;
  });
}
