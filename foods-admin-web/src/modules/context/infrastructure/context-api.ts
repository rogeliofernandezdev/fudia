import {apiFetch} from "@/shared/api/client";
import type {OrgSummary, LocationSummary, SwitchContextInput, ContextResponse} from "../domain/types";

export function listOrganizations(): Promise<{ items: OrgSummary[] }> {
  return apiFetch<{ items: OrgSummary[] }>("organizations");
}

export function listOrgLocations(orgId: string): Promise<{ items: LocationSummary[] }> {
  return apiFetch<{ items: LocationSummary[] }>(`organizations/${orgId}/locations`);
}

export function listAvailableLocations(): Promise<{ items: LocationSummary[] }> {
  return apiFetch<{ items: LocationSummary[] }>("locations/available");
}

export function switchContext(input: SwitchContextInput): Promise<ContextResponse> {
  return apiFetch<ContextResponse>("switch-context", {
    method: "POST",
    body: JSON.stringify(input),
  });
}
