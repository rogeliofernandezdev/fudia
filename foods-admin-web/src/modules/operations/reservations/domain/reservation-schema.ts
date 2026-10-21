import {z} from "zod";
import type {FieldError,FieldErrors,Resolver} from "react-hook-form";
import type {ReservationDraft} from "./types";

export const reservationSchema=z.object({
  id:z.string().optional(),
  customerName:z.string().trim().min(1,"Ingresa el nombre del cliente.").max(180,"El nombre no puede superar 180 caracteres."),
  customerPhone:z.string().trim().max(40,"El teléfono no puede superar 40 caracteres."),
  startsAt:z.string().min(1,"Selecciona la fecha y hora."),
  guests:z.string().min(1,"Indica el número de personas."),
  durationMinutes:z.string().min(1,"Indica la duración de la reserva."),
  tableId:z.string(),
  notes:z.string().trim().max(500,"Las notas no pueden superar 500 caracteres."),
}).superRefine((value,ctx)=>{
  const startsAt=new Date(value.startsAt);
  if(Number.isNaN(startsAt.getTime())){
    ctx.addIssue({code:"custom",path:["startsAt"],message:"Selecciona una fecha y hora válidas."});
  }else if(startsAt.getTime()<Date.now()-5*60*1000){
    ctx.addIssue({code:"custom",path:["startsAt"],message:"La reserva no puede quedar en el pasado."});
  }

  const guests=Number(value.guests);
  if(!Number.isInteger(guests)||guests<1||guests>100){
    ctx.addIssue({code:"custom",path:["guests"],message:"Ingresa entre 1 y 100 personas."});
  }

  const duration=Number(value.durationMinutes);
  if(!Number.isInteger(duration)||duration<15||duration>360||duration%15!==0){
    ctx.addIssue({code:"custom",path:["durationMinutes"],message:"Usa una duración entre 15 y 360 minutos, en bloques de 15."});
  }
});

export const reservationResolver:Resolver<ReservationDraft>=(values)=>{
  const result=reservationSchema.safeParse(values);
  if(result.success)return {values:result.data,errors:{}};
  const errors:Record<string,FieldError>={};
  for(const issue of result.error.issues){
    const field=issue.path[0];
    if(typeof field==="string"&&!errors[field]){
      errors[field]={type:issue.code,message:issue.message};
    }
  }
  return {values:{},errors:errors as FieldErrors<ReservationDraft>};
};
