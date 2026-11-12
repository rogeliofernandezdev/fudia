import {apiFetch} from "@/shared/api/client";

import type {GeographicArea} from "../domain/types";
export async function listGeographicAreas(parentCode=""){
 const result=await apiFetch<{items:GeographicArea[]}>(`geographic-areas?parentCode=${encodeURIComponent(parentCode)}`);
 return result.items;
}
