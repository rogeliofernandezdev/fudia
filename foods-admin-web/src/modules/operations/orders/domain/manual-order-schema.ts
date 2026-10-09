import {z} from "zod";
import {zodResolver} from "@/shared/forms/zod-resolver";
import type {LineDraft} from "../../salon/domain/types";

export type ManualOrderFields={channel:string;customerName:string;customerPhone:string;address:string;reference:string;deliveryFee:string;notes:string};
export type ManualOrderDraft=ManualOrderFields&{lines:LineDraft[]};
export const emptyManualOrder:ManualOrderFields={channel:"delivery",customerName:"",customerPhone:"",address:"",reference:"",deliveryFee:"0",notes:""};
export const isManualOrderChannel=(channel:string)=>["delivery","recojo","mostrador"].includes(channel);
const validPhone=(value:string)=>/^\+?[\d\s().-]+$/.test(value)&&value.replace(/\D/g,"").length>=7&&value.replace(/\D/g,"").length<=15;
export const manualOrderSchema=z.object({
 channel:z.string().refine(isManualOrderChannel,"Selecciona un tipo de atención válido."),
 customerName:z.string().trim().max(160,"El nombre no puede superar 160 caracteres."),
 customerPhone:z.string().trim().max(30,"El teléfono no puede superar 30 caracteres."),
 address:z.string().trim().max(240,"La dirección no puede superar 240 caracteres."),
 reference:z.string().trim().max(240,"La referencia no puede superar 240 caracteres."),
 deliveryFee:z.string().trim(),
 notes:z.string().trim().max(500,"Las notas no pueden superar 500 caracteres."),
}).superRefine((value,ctx)=>{
 if(value.channel!=="delivery")return;
 if(value.deliveryFee===""||!Number.isFinite(Number(value.deliveryFee))||Number(value.deliveryFee)<0)ctx.addIssue({code:"custom",path:["deliveryFee"],message:"Ingresa un costo de envío válido, igual o mayor que cero."});
 if(!value.customerName)ctx.addIssue({code:"custom",path:["customerName"],message:"Ingresa el nombre del cliente."});
 if(!validPhone(value.customerPhone))ctx.addIssue({code:"custom",path:["customerPhone"],message:"Ingresa un teléfono válido de 7 a 15 dígitos."});
 if(!value.address)ctx.addIssue({code:"custom",path:["address"],message:"Ingresa la dirección de entrega."});
});
export const manualOrderResolver=zodResolver<ManualOrderFields>(manualOrderSchema);

export function manualOrderPayload(draft:ManualOrderDraft){
 const parsed=manualOrderSchema.parse(draft);
 if(!draft.lines.length)throw new Error("Agrega al menos un producto.");
 return {...parsed,address:parsed.channel==="delivery"?parsed.address:"",reference:parsed.channel==="delivery"?parsed.reference:"",deliveryFee:parsed.channel==="delivery"?Number(parsed.deliveryFee):0,sendToKitchen:true,items:draft.lines.map(line=>({productId:line.productId,qty:line.qty,note:line.note,selections:line.selections.map(selection=>({groupId:selection.groupId,productId:selection.productId}))}))};
}
