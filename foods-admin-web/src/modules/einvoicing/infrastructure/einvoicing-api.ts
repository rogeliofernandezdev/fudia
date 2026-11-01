import {apiFetch} from "@/shared/api/client";
import type {CheckoutOptions,CredentialsInput,EInvoicingSettings,FiscalDocumentDetail,FiscalDocumentsPage,FiscalFileKind,LocationSettingsInput,ProfileSettingsInput} from "../domain/types";

export const getCheckoutOptions=()=>apiFetch<CheckoutOptions>("einvoicing/checkout-options");
export const getEInvoicingSettings=()=>apiFetch<EInvoicingSettings>("einvoicing/settings");
export const saveProfileSettings=(profileId:string,input:ProfileSettingsInput)=>apiFetch<EInvoicingSettings>(`einvoicing/profiles/${encodeURIComponent(profileId)}`,{method:"PUT",body:JSON.stringify(input)});
export const saveCredentials=(profileId:string,input:CredentialsInput)=>apiFetch<EInvoicingSettings>(`einvoicing/profiles/${encodeURIComponent(profileId)}/credentials`,{method:"PUT",body:JSON.stringify(input)});
export const saveLocationSettings=(locationId:string,input:LocationSettingsInput)=>apiFetch<EInvoicingSettings>(`einvoicing/locations/${encodeURIComponent(locationId)}`,{method:"PUT",body:JSON.stringify(input)});
export function listFiscalDocuments(input:{q:string;status:string;page:number;pageSize:number}){
 const params=new URLSearchParams({q:input.q,status:input.status,page:String(input.page),pageSize:String(input.pageSize)});
 return apiFetch<FiscalDocumentsPage>(`fiscal-documents?${params.toString()}`);
}
export const getFiscalDocument=(id:string)=>apiFetch<FiscalDocumentDetail>(`fiscal-documents/${encodeURIComponent(id)}`);
/** Same-origin BFF path; the browser downloads it with the session cookie. */
export const fiscalDocumentFileUrl=(id:string,kind:FiscalFileKind)=>`/api/admin/fiscal-documents/${encodeURIComponent(id)}/files/${kind}`;
export const retryFiscalDocument=(id:string)=>apiFetch<FiscalDocumentDetail>(`fiscal-documents/${encodeURIComponent(id)}/retry`,{method:"POST"});
