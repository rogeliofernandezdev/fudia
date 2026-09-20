import {apiFetch} from "@/shared/api/client";
import type {ModulesResponse, ToggleModuleInput} from "../domain/types";

export function listModules(): Promise<ModulesResponse> {
  return apiFetch<ModulesResponse>("modules");
}

export function toggleModule(input: ToggleModuleInput): Promise<void> {
  return apiFetch<void>("modules", {
    method: "PATCH",
    body: JSON.stringify(input),
  });
}
