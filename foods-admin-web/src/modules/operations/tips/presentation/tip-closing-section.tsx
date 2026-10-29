"use client";
import LookupSelect from "react-select";
import {FormField,FormSection,Input,Select} from "@/design-system";
import {createLookupSelectStyles} from "@/design-system/search-select-styles";
import type {TipWorkspace} from "../domain/types";
import "./tips.css";

export type TipClosingDraft={recipientId:string;amount:string;method:"cash"|"transfer";requestKey:string};

export function TipClosingSection({workspace,value,onChange,busy,error,formatMoney}:{workspace:TipWorkspace;value?:TipClosingDraft;onChange:(value:TipClosingDraft|undefined)=>void;busy:boolean;error?:string;formatMoney:(value:number)=>string}){
 if(Number(workspace.available)<=0)return null;
 const options=workspace.recipients.map(person=>({value:person.id,label:person.name}));
 return <FormSection title="Propinas pendientes" icon="percent" bare>
  <div className="tip-settings-form">
   <div className="tip-available"><span>Disponibles para entregar</span><strong>{formatMoney(Number(workspace.available))}</strong></div>
   <label className="tip-checkbox"><input type="checkbox" checked={Boolean(value)} disabled={busy} onChange={event=>onChange(event.target.checked?{recipientId:"",amount:workspace.available,method:"transfer",requestKey:crypto.randomUUID()}:undefined)}/><span>Registrar entrega al personal</span></label>
   {value&&<>
    <FormField as="div" label="Trabajador que recibe"><LookupSelect<{value:string;label:string}> inputId="closing-tip-recipient" aria-label="Trabajador que recibe" options={options} value={options.find(option=>option.value===value.recipientId)??null} onChange={option=>onChange({...value,recipientId:option?.value??""})} isDisabled={busy} isSearchable placeholder="Buscar trabajador…" noOptionsMessage={()=>"Sin trabajadores disponibles"} styles={createLookupSelectStyles<{value:string;label:string}>()}/></FormField>
    <div className="tip-settings-grid">
     <FormField label="Monto a entregar"><Input type="number" min="0" inputMode="decimal" step="any" max={workspace.available} disabled={busy} value={value.amount} onChange={event=>onChange({...value,amount:event.target.value})}/></FormField>
     <FormField label="Medio de entrega"><Select disabled={busy} value={value.method} onChange={event=>onChange({...value,method:event.target.value as TipClosingDraft["method"]})}><option value="transfer">Transferencia</option><option value="cash">Efectivo separado para propinas</option></Select></FormField>
    </div>
   </>}
   <p className="tip-help">{value?"Confirma únicamente dinero ya entregado al trabajador. Se registrará junto con el cierre, separado del efectivo esperado.":"Puedes cerrar sin entregarlas. El saldo seguirá pendiente para un próximo turno."}</p>
   {error&&<p className="field-error" role="alert">{error}</p>}
  </div>
 </FormSection>;
}
