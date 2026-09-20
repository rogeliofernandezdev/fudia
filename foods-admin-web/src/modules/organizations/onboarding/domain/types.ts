import type {Country,Currency} from "../../domain/types";
export type OnboardingCatalogs={countryOptions:Country[];currencyOptions:Currency[]};
export type ConfigurationOnboardingDraft={
 legalName:string;tradeName:string;taxId:string;timezone:string;country:string;currency:string;
 currencyPosition:"before"|"after";taxName:string;taxRate:string;taxIncluded:boolean;
 locationName:string;locationCode:string;address:string;adminName:string;adminEmail:string;adminPassword:string;
};
