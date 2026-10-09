import type {Country,PlatformOnboardingDraft} from "./types";

/** Refreshes must not replace a timezone explicitly chosen for the local. */
export function countryLocationDefaults(country:Country,current:Pick<PlatformOnboardingDraft,"country"|"timezone">){
  const changed=current.country!==country.code;
  return {timezone:changed||!current.timezone?country.defaultTimezone??"":current.timezone};
}
