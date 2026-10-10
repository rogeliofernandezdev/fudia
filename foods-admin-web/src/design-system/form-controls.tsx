"use client";
import "./styles/form-controls.css";
import {forwardRef,type InputHTMLAttributes,type ReactNode} from "react";
import {Icon,type IconName} from "./icons";

type MoneyInputProps=Omit<InputHTMLAttributes<HTMLInputElement>,"type"|"inputMode">&{
  /** Símbolo de la moneda de la empresa (useCurrency().symbol). Nunca se escribe a mano. */
  symbol:string;
  position?:"before"|"after";
};

/**
 * Importe homologado: un solo borde en el contenedor, símbolo como prefijo o sufijo, teclado decimal
 * y cifras tabulares. `aria-invalid` y `aria-describedby` llegan al input real (compatible con FormField).
 */
export const MoneyInput=forwardRef<HTMLInputElement,MoneyInputProps>(function MoneyInput({symbol,position="before",className="",...props},ref){
  return <span className={`money-input ${position==="after"?"suffix":""} ${props["aria-invalid"]?"invalid":""} ${props.disabled?"disabled":""}`.replace(/\s+/g," ").trim()}>
    {position==="before"&&<span className="money-input-symbol" aria-hidden="true">{symbol}</span>}
    <input ref={ref} type="text" inputMode="decimal" autoComplete="off" className={`ds-input ${className}`.trim()} {...props}/>
    {position==="after"&&<span className="money-input-symbol" aria-hidden="true">{symbol}</span>}
  </span>;
});

type NumberStepperProps={
  value:number;
  onChange:(value:number)=>void;
  /** Nombre del dato para los nombres accesibles: «Disminuir {label}». */
  label:string;
  min?:number;
  max?:number;
  step?:number;
  disabled?:boolean;
  /** En el mínimo, el botón izquierdo pasa a «Quitar» con papelera en vez de restar. */
  onRemove?:()=>void;
};

/** Cantidad con botones de 44 px en táctil, valor tabular y quitar explícito en el mínimo. */
export function NumberStepper({value,onChange,label,min=0,max,step=1,disabled=false,onRemove}:NumberStepperProps){
  const atMin=value<=min,atMax=max!==undefined&&value>=max;
  const removing=atMin&&Boolean(onRemove);
  return <div className="ds-stepper" role="group" aria-label={label}>
    <button type="button" className={`ds-stepper-button${removing?" danger":""}`} aria-label={removing?`Quitar ${label}`:`Disminuir ${label}`} disabled={disabled||(atMin&&!onRemove)} onClick={()=>removing?onRemove?.():onChange(Math.max(min,value-step))}><Icon name={removing?"trash":"minus"} size={16}/></button>
    <output className="ds-stepper-value" aria-live="polite">{value}</output>
    <button type="button" className="ds-stepper-button" aria-label={`Aumentar ${label}`} disabled={disabled||atMax} onClick={()=>onChange(max===undefined?value+step:Math.min(max,value+step))}><Icon name="plus" size={16}/></button>
  </div>;
}

type NoticeTone="info"|"warning"|"danger"|"success";
const noticeIcon:Record<NoticeTone,IconName>={info:"note",warning:"alert",danger:"alert",success:"circleCheck"};

/**
 * Aviso contextual dentro de una pantalla o modal (restricciones, advertencias operativas). No es un botón
 * ni un toast: texto e icono semántico sobre una superficie tintada, sin barra lateral de color.
 */
export function InlineNotice({tone="info",title,children,action,className=""}:{tone?:NoticeTone;title?:ReactNode;children?:ReactNode;action?:ReactNode;className?:string}){
  return <div className={`ds-notice ${tone} ${className}`.trim()} role={tone==="danger"?"alert":"status"}>
    <Icon name={noticeIcon[tone]} size={18}/>
    <div>{title&&<b>{title}</b>}{children&&<p>{children}</p>}</div>
    {action}
  </div>;
}
