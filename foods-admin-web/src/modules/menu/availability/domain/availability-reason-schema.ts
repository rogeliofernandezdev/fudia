import {z} from "zod";
import {zodResolver} from "@/shared/forms/zod-resolver";
import type {AvailabilityReasonDraft} from "./types";

/** Motivo de un toque; el detalle escrito solo se exige con «Otro». */
export const availabilityReasonResolver=zodResolver<AvailabilityReasonDraft>(z.object({
 reasonCode:z.string().min(1,"Selecciona el motivo del cambio."),
 reason:z.string().trim().max(240,"El detalle no puede superar 240 caracteres."),
}).superRefine((value,context)=>{
 if(value.reasonCode==="other"&&!value.reason.trim())context.addIssue({code:"custom",path:["reason"],message:"Detalla el motivo."});
}));
