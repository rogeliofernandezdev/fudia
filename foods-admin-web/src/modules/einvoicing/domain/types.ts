export type Option={value:string;label:string};
export type CustomerDocumentOption=Option&{pattern:string};
export type FiscalDocumentType="none"|"receipt"|"invoice";
export type FiscalStatus="PENDING"|"SENDING"|"ACCEPTED"|"OBSERVED"|"REJECTED"|"ERROR"|"CANCELLED";

export type CheckoutOptions={
 enabled:boolean;
 defaultDocument:FiscalDocumentType;
 documentTypes:Option[];
 customerDocumentTypes:CustomerDocumentOption[];
 identityThreshold:string;
 invoiceCustomerDocumentType:string;
 currency:string;
};
/** What the cashier types; sent as `fiscal` on the payment that settles the account. */
export type FiscalDraft={documentType:FiscalDocumentType;customerDocumentType:string;customerDocumentNumber:string;customerName:string;customerAddress:string};
export type FiscalRequest={documentType:FiscalDocumentType;customer:{documentType:string;documentNumber:string;name:string;address:string}};
export type FiscalDocumentRef={id:string;documentType:string;label:string;number:string;status:FiscalStatus};

export type FiscalDocumentSummary={
 id:string;documentType:string;label:string;number:string;issueDate:string;customerName:string;customerDocument:string;
 currency:string;total:string;status:FiscalStatus;statusMessage:string;orderId:string;orderCode:string;createdAt:string;acceptedAt:string|null;canRetry:boolean;
};
export type FiscalLine={no:number;description:string;quantity:string;unitPrice:string;netAmount:string;taxAmount:string;total:string};
export type FiscalTax={code:string;name:string;rate:string;amount:string};
export type FiscalDocumentDetail=FiscalDocumentSummary&{
 customer:{documentType:string;documentNumber:string;name:string;address:string};
 breakdown:{lines:FiscalLine[];taxes:FiscalTax[];netTotal:string;taxTotal:string;total:string};
 providerLabel:string;environment:"test"|"production";providerReference:string;pdfUrl:string;xmlUrl:string;cdrUrl:string;attempts:number;nextAttemptAt:string|null;
 digestValue:string;providerCode:string;files:FiscalFileKind[];
};
export type FiscalFileKind="xml"|"cdr";
export type FiscalDocumentsPage={items:FiscalDocumentSummary[];total:number;page:number;pageSize:number;statusOptions:Option[]};

export type SeriesSetting={documentType:string;label:string;series:string;suggested:string;lastNumber:number};
/** Declared by the provider: a file travels as base64; secrets never come back. */
export type CredentialField={key:string;label:string;type:"text"|"secret"|"file";accept?:string;required:boolean};
export type CredentialStatus={configured:boolean;required:boolean;updatedAt:string;metadata:Record<string,string>|null;fields:CredentialField[]|null};
export type ProfileSetting={fiscalProfileId:string;country:string;countryName:string;currency:string;default:boolean;supported:boolean;enabled:boolean;provider:string;environment:"test"|"production";issuerIssue:string;credentials:CredentialStatus};
export type LocationSetting={locationId:string;name:string;fiscalProfileId:string;country:string;supported:boolean;defaultDocument:FiscalDocumentType;series:SeriesSetting[]};
export type EInvoicingSettings={entitled:boolean;profiles:ProfileSetting[];locations:LocationSetting[];providerOptions:Option[];environmentLabels:Record<string,string>};
export type ProfileSettingsInput={enabled:boolean;provider:string;environment:"test"|"production"};
export type CredentialsInput={provider:string;values:Record<string,string>};
export type LocationSettingsInput={defaultDocument:FiscalDocumentType;series:Record<string,string>};
