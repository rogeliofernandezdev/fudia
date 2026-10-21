import {z} from "zod";
import {zodResolver} from "@/shared/forms/zod-resolver";
import type {InventorySettingsDraft} from "./types";

const stockLevel=z.string().trim().refine(value=>value!==""&&Number.isFinite(Number(value))&&Number(value)>=0,"Ingresa una cantidad igual o mayor que cero.");
const schema=z.object({
  inventoryItemId:z.string().trim().min(1,"Selecciona un artículo."),
  minimumStock:stockLevel,
  reorderPoint:stockLevel,
  optimalStock:stockLevel,
});

export const inventorySettingsResolver=zodResolver<InventorySettingsDraft>(schema);
