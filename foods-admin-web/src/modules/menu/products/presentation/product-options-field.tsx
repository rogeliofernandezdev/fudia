"use client";
import {Button,FormField,IconButton,Input,MoneyInput,Select} from "@/design-system";
import {Icon} from "@/design-system/icons";
import {emptyGroup,emptyOption,type ModifierErrors,type ModifierGroupDraft,type ModifierOptionDraft} from "../domain/modifiers";

/**
 * Opciones que se eligen al tomar el pedido: grupos (término, acompañamiento, adicionales)
 * con su regla de elección y opciones con recargo. Un plato sin grupos se agrega directo.
 */
export function ProductOptionsField({groups,onChange,errors,currencySymbol,disabled=false}:{
 groups:ModifierGroupDraft[];
 onChange:(groups:ModifierGroupDraft[])=>void;
 errors:ModifierErrors;
 currencySymbol:string;
 disabled?:boolean;
}){
 const patchGroup=(key:string,patch:Partial<ModifierGroupDraft>)=>onChange(groups.map(group=>group.key===key?{...group,...patch}:group));
 const patchOption=(group:ModifierGroupDraft,key:string,patch:Partial<ModifierOptionDraft>)=>patchGroup(group.key,{options:group.options.map(option=>option.key===key?{...option,...patch}:option)});
 const removeOption=(group:ModifierGroupDraft,key:string)=>{
  const options=group.options.filter(option=>option.key!==key);
  patchGroup(group.key,{options,maxSelections:Math.max(1,Math.min(group.maxSelections,options.length))});
 };
 return <div className="product-options">
  {!groups.length&&<p className="product-options-empty">Sin opciones: el plato se agrega directo a la comanda. Agrega un grupo para pedir el término, el acompañamiento o adicionales.</p>}
  {groups.map((group,index)=>{
   const choices=Math.max(1,group.options.length);
   return <section className="product-option-group" key={group.key} aria-label={group.name.trim()||`Grupo ${index+1}`}>
    <div className="product-option-group-head">
     <FormField label="Grupo" error={errors[`${group.key}:name`]}><Input value={group.name} maxLength={80} disabled={disabled} placeholder="Ej. Acompañamiento" aria-invalid={Boolean(errors[`${group.key}:name`])} onChange={event=>patchGroup(group.key,{name:event.target.value})}/></FormField>
     <FormField label="Se elige"><Select value={String(group.maxSelections)} disabled={disabled} onChange={event=>patchGroup(group.key,{maxSelections:Number(event.target.value)})}>{Array.from({length:choices},(_,i)=>i+1).map(count=><option key={count} value={count}>{count===1?"Una opción":`Hasta ${count}`}</option>)}</Select></FormField>
     <IconButton icon="trash" label={`Quitar el grupo ${group.name.trim()||index+1}`} className="product-option-remove" disabled={disabled} onClick={()=>onChange(groups.filter(item=>item.key!==group.key))}/>
    </div>
    <label className="switch-row compact"><input type="checkbox" checked={group.required} disabled={disabled} onChange={event=>patchGroup(group.key,{required:event.target.checked})}/><span/><b>Obligatorio</b></label>
    <ul className="product-option-list">
     {group.options.map((option,optionIndex)=>{
      const nameError=errors[`${option.key}:name`],surchargeError=errors[`${option.key}:surcharge`];
      return <li key={option.key}>
       <Input value={option.name} maxLength={80} disabled={disabled} placeholder={optionIndex===0?"Ej. Arroz blanco":"Ej. Arroz chaufa"} aria-label={`Opción ${optionIndex+1}`} aria-invalid={Boolean(nameError)} onChange={event=>patchOption(group,option.key,{name:event.target.value})}/>
       <MoneyInput symbol={`+${currencySymbol}`} value={option.surcharge} disabled={disabled} placeholder="0.00" aria-label={`Recargo de la opción ${optionIndex+1}`} aria-invalid={Boolean(surchargeError)} onChange={event=>patchOption(group,option.key,{surcharge:event.target.value})}/>
       <IconButton icon="close" label={`Quitar la opción ${option.name.trim()||optionIndex+1}`} disabled={disabled||group.options.length<=1} onClick={()=>removeOption(group,option.key)}/>
       {(nameError||surchargeError)&&<small className="product-option-error" role="alert">{nameError??surchargeError}</small>}
      </li>;
     })}
    </ul>
    {errors[`${group.key}:options`]&&<small className="product-option-error" role="alert">{errors[`${group.key}:options`]}</small>}
    <button type="button" className="product-option-add" disabled={disabled} onClick={()=>patchGroup(group.key,{options:[...group.options,emptyOption()]})}><Icon name="plus" size={16}/>Agregar opción</button>
   </section>;
  })}
  <Button kind="secondary" icon="plus" disabled={disabled} onClick={()=>onChange([...groups,emptyGroup()])}>Agregar grupo</Button>
 </div>;
}
