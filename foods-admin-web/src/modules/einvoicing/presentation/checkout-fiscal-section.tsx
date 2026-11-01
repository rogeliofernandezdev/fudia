"use client";
import {useQuery} from "@tanstack/react-query";
import {FormField,FormSection,Input,Select,Tabs} from "@/design-system";
import {useSession} from "@/providers/session-context";
import type {CheckoutOptions,FiscalDocumentType,FiscalDraft} from "../domain/types";
import {documentTypeIcon,invoiceDocumentType,type FiscalErrors} from "../domain/fiscal";
import {getCheckoutOptions} from "../infrastructure/einvoicing-api";
import "./einvoicing.css";

/** Checkout never waits for invoicing: while loading or on error no section is shown. */
export function useCheckoutOptions(){
 const{organization,location}=useSession();
 return useQuery({queryKey:["einvoicing-checkout",organization?.id,location?.id],queryFn:getCheckoutOptions,staleTime:60_000,retry:false});
}

// «Sin comprobante», «Boleta» or «Factura» switch the form mode, so they use the
// shared Tabs. Only the payment that settles the account shows this section.
export function CheckoutFiscalSection({options,value,onChange,errors,disabled}:{options:CheckoutOptions;value:FiscalDraft;onChange:(value:FiscalDraft)=>void;errors:FiscalErrors;disabled:boolean}){
 const set=(patch:Partial<FiscalDraft>)=>onChange({...value,...patch});
 const invoiceType=options.customerDocumentTypes.find(option=>option.value===invoiceDocumentType(options));
 const items=options.documentTypes.map(option=>({value:option.value as FiscalDocumentType,label:option.label,icon:documentTypeIcon[option.value]??"receipt"}));
 return <FormSection icon="receipt" title="Comprobante" bare>
  <div className="checkout-fiscal">
   <Tabs label="Comprobante" value={value.documentType} onChange={documentType=>set({documentType})} items={items}/>
   {value.documentType==="receipt"&&<div className="pos-payment-fields">
     <FormField label="Documento" optional icon={false}><Select disabled={disabled} value={value.customerDocumentType} onChange={event=>set({customerDocumentType:event.target.value,customerDocumentNumber:event.target.value?value.customerDocumentNumber:""})}>
      <option value="">Sin documento</option>
      {options.customerDocumentTypes.map(option=><option key={option.value} value={option.value}>{option.label}</option>)}
     </Select></FormField>
     <FormField label="Número" error={errors.customerDocumentNumber} icon={false}><Input disabled={disabled||!value.customerDocumentType} inputMode="text" autoComplete="off" maxLength={15} value={value.customerDocumentNumber} onChange={event=>set({customerDocumentNumber:event.target.value})} aria-invalid={Boolean(errors.customerDocumentNumber)}/></FormField>
    {value.customerDocumentType&&<FormField className="span-2" label="Nombre del cliente" error={errors.customerName}><Input disabled={disabled} maxLength={200} autoComplete="off" value={value.customerName} onChange={event=>set({customerName:event.target.value})} aria-invalid={Boolean(errors.customerName)}/></FormField>}
   </div>}
   {value.documentType==="invoice"&&<div className="pos-payment-fields">
    <FormField label={invoiceType?.label??"Identificación fiscal"} error={errors.customerDocumentNumber} icon={false}><Input disabled={disabled} inputMode="numeric" autoComplete="off" maxLength={15} value={value.customerDocumentNumber} onChange={event=>set({customerDocumentNumber:event.target.value})} aria-invalid={Boolean(errors.customerDocumentNumber)}/></FormField>
    <FormField className="span-2" label="Razón social" error={errors.customerName}><Input disabled={disabled} maxLength={200} autoComplete="off" value={value.customerName} onChange={event=>set({customerName:event.target.value})} aria-invalid={Boolean(errors.customerName)}/></FormField>
    <FormField className="span-2" label="Dirección" optional><Input disabled={disabled} maxLength={250} autoComplete="off" value={value.customerAddress} onChange={event=>set({customerAddress:event.target.value})}/></FormField>
   </div>}
  </div>
 </FormSection>;
}
