import {z} from "zod";
import {zodResolver} from "@/shared/forms/zod-resolver";
import type {InventoryTransferDraft} from "./types";

const quantity=z.string().trim().refine(value=>{
  const parsed=Number(value);
  return value!==""&&Number.isFinite(parsed)&&parsed>0;
},"Ingresa una cantidad mayor que cero.");

export const inventoryTransferSchema=z.object({
  idempotencyKey:z.string(),
  toLocationId:z.string().trim().min(1,"Selecciona el local destino."),
  notes:z.string().max(500,"La nota no puede superar 500 caracteres."),
  items:z.array(z.object({
    inventoryItemId:z.string().trim().min(1,"Selecciona un artículo."),
    quantity,
  })).min(1,"Agrega al menos un artículo."),
});

export const inventoryTransferResolver=zodResolver<InventoryTransferDraft>(inventoryTransferSchema);
