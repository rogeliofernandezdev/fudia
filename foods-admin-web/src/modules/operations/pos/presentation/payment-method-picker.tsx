"use client";
// El componente importa sus estilos: importados solo desde el índice del módulo, el servidor
// de desarrollo los perdía en otras verticales (Reservas, 2026-11-27).
import "./pos.css";
import {IconTile} from "@/design-system";
import {Icon} from "@/design-system/icons";
import type {PaymentMethodOption} from "../domain/types";
import {paymentMethodIcon,paymentMethodTones} from "./pos-dialogs";

/** Tarjetas de medio de pago de Cobros para elegir uno (Reservas cobra la garantía con ellas). */
export function PaymentMethodPicker({methods,value,name,disabled=false,onChange}:{methods:PaymentMethodOption[];value:string;name:string;disabled?:boolean;onChange:(code:string)=>void}){
  const tones=paymentMethodTones(methods);
  return <fieldset className="pos-methods" disabled={disabled}>
    <legend className="sr-only">Medio de pago</legend>
    {methods.map(option=><label className={"pos-method"+(value===option.code?" active":"")} key={option.code}>
      <input type="radio" name={name} value={option.code} checked={value===option.code} onChange={()=>onChange(option.code)}/>
      <IconTile name={paymentMethodIcon(option.affectsCash)} tone={tones[option.code]}/>
      <span className="pos-method-name">{option.name}</span>
      <i className="pos-method-check" aria-hidden="true"><Icon name="check" size={14}/></i>
    </label>)}
  </fieldset>;
}
