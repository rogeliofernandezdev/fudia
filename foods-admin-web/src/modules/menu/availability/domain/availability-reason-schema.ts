import {z} from "zod";
import {zodResolver} from "@/shared/forms/zod-resolver";
import type {AvailabilityReasonDraft} from "./types";

export const availabilityReasonResolver=zodResolver<AvailabilityReasonDraft>(z.object({
 reason:z.string().trim().min(1,"Indica el motivo del cambio.").max(240,"El motivo no puede superar 240 caracteres."),
}));
