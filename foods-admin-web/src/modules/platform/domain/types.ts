export type Country={code:string;name:string;defaultCurrency:string};
export type Currency={code:string;name:string;symbol:string;decimals:number};
export type PlatformOnboardingContext={countryOptions:Country[];currencyOptions:Currency[]};
export type PlatformOnboardingDraft={
 legalName:string;tradeName:string;taxId:string;timezone:string;
 country:string;currency:string;currencyPosition:"before"|"after";taxName:string;taxRate:string;taxIncluded:boolean;
 locationName:string;locationCode:string;address:string;locationPhone:string;locationHours:string;latitude:string;longitude:string;
 adminName:string;adminEmail:string;adminPassword:string;
};
