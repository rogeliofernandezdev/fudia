import {z} from "zod";
import {zodResolver} from "@/shared/forms/zod-resolver";
import type {ZoneDraft} from "./types";

export const zoneSchema=z.object({
 name:z.string().trim().min(1,"Ingresa el nombre de la zona.").max(60,"El nombre no puede superar 60 caracteres."),
 sortOrder:z.coerce.number().int("Usa un número entero.").min(0,"El orden no puede ser negativo."),
});
export const zoneResolver=zodResolver<ZoneDraft>(zoneSchema.passthrough());
