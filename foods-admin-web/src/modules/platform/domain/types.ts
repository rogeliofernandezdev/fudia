export type Country={code:string;name:string;defaultCurrency:string};
export type Currency={code:string;name:string;symbol:string;decimals:number};

export type PlatformModule={key:string;name:string;description:string;category:string;availability:"ready"|"development"|"planned"};

export type SubscriptionPlan={
 id:string;
 code:string;
 name:string;
 description:string;
 currency:string;
 monthlyPrice:string;
 annualPrice:string;
 trialDays:number;
 maxLocations:number|null;
 maxUsers:number|null;
 moduleKeys:string[];
 termsVersion:string;
 active:boolean;
};

export type SubscriptionPlanDraft={
 id?:string;
 code:string;
 name:string;
 description:string;
 currency:string;
 monthlyPrice:string;
 annualPrice:string;
 trialDays:number;
 maxLocations:string;
 maxUsers:string;
 moduleKeys:string[];
 termsVersion:string;
 active:boolean;
};

export type SubscriptionPayment={
 id:string;
 amount:string;
 currency:string;
 status:"pending"|"paid"|"failed"|"refunded";
 provider:string;
 externalReference:string|null;
 paidAt:string|null;
 createdAt:string;
};

export type OrganizationSubscription={
 id:string;
 plan:SubscriptionPlan;
 billingCycle:"monthly"|"annual";
 priceAmount:string;
 currency:string;
 status:"trial"|"active"|"past_due"|"cancelled";
 trialStartsAt:string|null;
 trialEndsAt:string|null;
 currentPeriodStartsAt:string|null;
 currentPeriodEndsAt:string|null;
 renewsAt:string|null;
 autoRenew:boolean;
 termsVersion:string|null;
 termsAcceptedAt:string|null;
 usage:{locations:number;users:number};
 payments:SubscriptionPayment[];
};

export type PlatformOnboardingContext={countryOptions:Country[];currencyOptions:Currency[];plans:SubscriptionPlan[]};
export type PlatformOnboardingDraft={
 legalName:string;tradeName:string;taxId:string;timezone:string;
 planId:string;billingCycle:"monthly"|"annual";termsAccepted:boolean;
 country:string;currency:string;currencyPosition:"before"|"after";taxName:string;taxRate:string;taxIncluded:boolean;
 locationName:string;locationCode:string;address:string;locationPhone:string;locationHours:string;latitude:string;longitude:string;
 adminName:string;adminEmail:string;adminPassword:string;
};
