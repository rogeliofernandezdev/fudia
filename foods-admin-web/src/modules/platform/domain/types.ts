import type {TipSettings} from "@/modules/operations/tips";
export type Country={code:string;name:string;defaultCurrency:string;callingCode:string;defaultTimezone:string};
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

/** Plan visto por Plataforma: agrega el tope mensual de Concierge, que nunca llega a las empresas. */
export type PlatformSubscriptionPlan=SubscriptionPlan&{maxConciergeConversations:number|null};

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
 maxConciergeConversations:string;
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
 organization:{id:string;tradeName:string;legalName:string;taxId:string;active:boolean};
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

export type PlatformOnboardingContext={tipOptions?:{defaults:TipSettings;countryDefaults:Record<string,TipSettings>;channels:{value:string;label:string}[]};countryOptions:Country[];currencyOptions:Currency[];plans:SubscriptionPlan[]};
export type PlatformOnboardingDraft={
 tipSettings?:TipSettings;
 legalName:string;tradeName:string;taxId:string;timezone:string;
 planId:string;billingCycle:"monthly"|"annual";termsAccepted:boolean;
 country:string;currency:string;currencyPosition:"before"|"after";taxName:string;taxRate:string;taxIncluded:boolean;
 locationName:string;locationCode:string;address:string;locationPhone:string;locationHours:string;latitude:string;longitude:string;
 adminName:string;adminEmail:string;adminPassword:string;
};

export type PlatformWhatsAppChannel={id:string;countryCode:string;countryName:string;phoneNumber:string;displayName:string;active:boolean};
export type PlatformWhatsAppChannelDraft={countryCode:string;phoneNumber:string;displayName:string;active:boolean};

export type Option={value:string;label:string};
export type PaymentStanding="up_to_date"|"due_soon"|"overdue"|"trial"|"cancelled"|"none";

export type PlatformOrganization={
 id:string;
 legalName:string;
 tradeName:string;
 taxId:string;
 active:boolean;
 createdAt:string;
 subscription:{
  plan:{id:string;code:string;name:string};
  billingCycle:"monthly"|"annual";
  priceAmount:string;
  currency:string;
  status:OrganizationSubscription["status"];
  renewsAt:string|null;
  trialEndsAt:string|null;
  autoRenew:boolean;
 }|null;
 lastPayment:{amount:string;currency:string;paidAt:string|null}|null;
 paymentStanding:PaymentStanding;
};

export type PlatformOrganizationPage={
 items:PlatformOrganization[];
 total:number;
 page:number;
 pageSize:number;
 summary:Record<"total"|PaymentStanding,number>;
 dueSoonDays:number;
 planOptions:Option[];
 standingOptions:Option[];
};

export type PlatformOrganizationFilters={q:string;planId:string;standing:string;page:number;pageSize:number};

export type PlatformOrganizationDetail={
 id:string;
 legalName:string;
 tradeName:string;
 taxId:string;
 countryCode:string;
 currency:string;
 timezone:string;
 taxName:string;
 taxRate:string;
 taxIncluded:boolean;
 active:boolean;
 createdAt:string;
 paymentStanding:PaymentStanding;
 usage:{locations:number;users:number;maxLocations:number|null;maxUsers:number|null};
 subscription:{
  planName:string;
  billingCycle:"monthly"|"annual";
  priceAmount:string;
  currency:string;
  status:OrganizationSubscription["status"];
  startedAt:string;
  renewsAt:string|null;
  trialEndsAt:string|null;
  autoRenew:boolean;
  termsVersion:string|null;
  termsAcceptedAt:string|null;
  moduleCount:number;
 }|null;
 payments:{paidCount:number;totalPaid:string;lastPaidAt:string|null};
 administrators:Array<{name:string;email:string;active:boolean}>;
 locations:Array<{id:string;name:string;code:string;address:string;phone:string;active:boolean}>;
};

/** Uso de Fudia Concierge: exclusivo del administrador de plataforma. */
export type ConciergeUsageMode="measure"|"warn"|"enforce";
export type ConciergeLimitSource="plan"|"custom"|"unlimited";
export type ConciergeAlert="none"|"warning"|"reached";
export type ConciergeTier="lite"|"flash";

export type ConciergeUsageSettings={mode:ConciergeUsageMode;warnPercent:number;updatedAt:string};
export type ConciergeUsageSettingsInput={mode:ConciergeUsageMode;warnPercent:number};
export type ConciergeUsageSettingsDraft={mode:ConciergeUsageMode;warnPercent:string};

export type ConciergeUsageSummary={
 organizationId:string;
 organizationName:string;
 month:string;
 planName:string|null;
 maxConversations:number|null;
 limitSource:ConciergeLimitSource;
 conversations:number;
 rejectedConversations:number;
 usagePercent:number|null;
 alert:ConciergeAlert;
};

export type ConciergeUsagePage={settings:ConciergeUsageSettings;summary:{warning:number;reached:number};items:ConciergeUsageSummary[]};

export type ConciergeTierTokens={tier:ConciergeTier;inputTokens:number;outputTokens:number};
export type ConciergeUsageMonth={month:string;conversations:number;rejectedConversations:number;inputTokens:number;outputTokens:number};

export type ConciergeUsageDetail=ConciergeUsageSummary&{
 planMaxConversations:number|null;
 settings:ConciergeUsageSettings;
 tokens:ConciergeTierTokens[];
 history:ConciergeUsageMonth[];
};

export type ConciergeLimitOverride={limitSource:ConciergeLimitSource;maxConversations:number|null};
export type ConciergeLimitDraft={limitSource:ConciergeLimitSource;maxConversations:string};
