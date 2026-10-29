"use client";
import {FormField,Input,Select} from "@/design-system";
import type {TipSettings} from "../domain/types";
import "./tips.css";
export function TipSettingsFields({value,onChange,disabled=false,currency,channelOptions}:{value:TipSettings;onChange:(value:TipSettings)=>void;disabled?:boolean;currency:string;channelOptions:{value:string;label:string}[]}){return <div className="tip-settings-form">
   <FormField label="Funcionamiento"><Select disabled={disabled} value={value.mode} onChange={event=>{const mode=event.target.value as TipSettings["mode"];onChange({...value,mode,showSuggestion:mode==="suggest"?true:value.showSuggestion})}}><option value="disabled">Desactivadas</option><option value="suggest">Solo sugerir en la precuenta</option><option value="collect">Cobrar y registrar propinas</option></Select></FormField>
   {value.mode!=="disabled"&&<>
    <FormField label="Porcentaje sugerido (%)"><Input type="number" step="0.01" min="0" max="100" required disabled={disabled} value={value.percentage} onChange={event=>onChange({...value,percentage:Number(event.target.value)})}/></FormField>
    {value.mode==="collect"&&<label className="tip-checkbox"><input type="checkbox" checked={value.showSuggestion} disabled={disabled} onChange={event=>onChange({...value,showSuggestion:event.target.checked})}/><span>Mostrar sugerencia en la precuenta</span></label>}
    {value.mode==="collect"&&<label className="tip-checkbox"><input type="checkbox" checked={value.allowFixed} disabled={disabled} onChange={event=>onChange({...value,allowFixed:event.target.checked})}/><span>Permitir un monto libre elegido por el cliente</span></label>}
    <fieldset className="tip-channels"><legend>Tipos de atención</legend>{channelOptions.map(channel=><label className="tip-checkbox" key={channel.value}><input type="checkbox" checked={value.channels.includes(channel.value)} disabled={disabled} onChange={event=>onChange({...value,channels:event.target.checked?[...value.channels,channel.value]:value.channels.filter(value=>value!==channel.value)})}/><span>{channel.label}</span></label>)}</fieldset>
   </>}
   <p className="tip-help">{value.mode==="disabled"?"No se sugerirán ni registrarán nuevas propinas en este local.":value.mode==="suggest"?"La sugerencia es voluntaria. No se registra ni se cobra automáticamente.":`El cliente debe aceptar la propina. Se registra separada del consumo, en ${currency}.`}</p>
</div>;}
