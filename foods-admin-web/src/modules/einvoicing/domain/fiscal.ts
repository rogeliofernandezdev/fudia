import type {IconName} from "@/design-system";
import type {CheckoutOptions,FiscalDocumentRef,FiscalDraft,FiscalRequest,FiscalStatus} from "./types";

/** Central mapping of fiscal document states: tone from the state matrix and a business label. */
export const fiscalStatusView:Record<FiscalStatus,{tone:"sky"|"green"|"orange"|"red"|"gray";label:string}>={
 PENDING:{tone:"sky",label:"Emitiendo"},
 SENDING:{tone:"sky",label:"Emitiendo"},
 ACCEPTED:{tone:"green",label:"Aceptado"},
 OBSERVED:{tone:"orange",label:"Observado"},
 REJECTED:{tone:"red",label:"Rechazado"},
 ERROR:{tone:"orange",label:"Con error"},
 CANCELLED:{tone:"gray",label:"Anulado"},
};
export const documentTypeIcon:Record<string,IconName>={none:"circleMinus",receipt:"receipt",invoice:"invoice"};

export function blankFiscalDraft(options?:CheckoutOptions):FiscalDraft{
 return {documentType:options?.enabled?options.defaultDocument:"none",customerDocumentType:"",customerDocumentNumber:"",customerName:"",customerAddress:""};
}

/** The business tax ID an invoice requires comes from the API (RUC in Peru). */
export function invoiceDocumentType(options:CheckoutOptions){
 return options.invoiceCustomerDocumentType;
}

export type FiscalErrors=Partial<Record<"customerDocumentNumber"|"customerName",string>>;

/** Immediate feedback with the API catalog; the backend validates again (RUC check digit included). */
export function validateFiscalDraft(options:CheckoutOptions|undefined,draft:FiscalDraft,total:number):FiscalErrors{
 if(!options?.enabled||draft.documentType==="none")return {};
 const errors:FiscalErrors={};
 const number=draft.customerDocumentNumber.trim(),name=draft.customerName.trim();
 if(draft.documentType==="invoice"){
  const pattern=options.customerDocumentTypes.find(option=>option.value===invoiceDocumentType(options))?.pattern;
  if(!number||(pattern&&!new RegExp(pattern).test(number)))errors.customerDocumentNumber="Revisa el número de identificación fiscal.";
  if(!name)errors.customerName="Ingresa la razón social.";
  return errors;
 }
 if(!draft.customerDocumentType){
  if(options.identityThreshold&&total>Number(options.identityThreshold))errors.customerDocumentNumber="Para este monto ingresa el documento del cliente.";
  return errors;
 }
 const pattern=options.customerDocumentTypes.find(option=>option.value===draft.customerDocumentType)?.pattern;
 if(!number||(pattern&&!new RegExp(pattern).test(number)))errors.customerDocumentNumber="Revisa el número del documento.";
 if(!name)errors.customerName="Ingresa el nombre del cliente.";
 return errors;
}

export function fiscalRequest(options:CheckoutOptions|undefined,draft:FiscalDraft):FiscalRequest|undefined{
 if(!options?.enabled||draft.documentType==="none")return undefined;
 const documentType=draft.documentType==="invoice"?invoiceDocumentType(options):draft.customerDocumentType;
 return {documentType:draft.documentType,customer:{documentType,documentNumber:documentType?draft.customerDocumentNumber.trim():"",name:draft.customerName.trim(),address:draft.documentType==="invoice"?draft.customerAddress.trim():""}};
}

/**
 * Reprints show what was delivered at checkout: the electronic document when
 * the order has one with fiscal value; otherwise (none, rejected, cancelled)
 * the payment ticket.
 */
export function deliveredDocument(ref?:FiscalDocumentRef|null):FiscalDocumentRef|null{
 return ref&&ref.status!=="REJECTED"&&ref.status!=="CANCELLED"?ref:null;
}

/** Context for a payment ticket printed because the order's document has no fiscal value. */
export function ticketNoticeFor(ref?:FiscalDocumentRef|null){
 if(!ref||deliveredDocument(ref))return undefined;
 return `${ref.label} ${ref.number} · ${fiscalStatusView[ref.status].label}. Este ticket no es un comprobante fiscal.`;
}
