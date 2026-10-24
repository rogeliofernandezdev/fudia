import type {ReactNode,Ref,ChangeEventHandler,FocusEventHandler} from "react";
import {Icon,type IconName} from "./icons";
import {IconTile,type IconTileTone} from "./icon-tile";

/**
 * Bloque de formulario con título e icono tintado y una retícula de dos
 * columnas (una en móvil). Agrupa los campos por tema, como «Datos del gasto»
 * o «Medio de pago». `span-2` en un campo ocupa todo el ancho.
 */
export function FormSection({icon,title,tone="primary",className="",bare=false,children}:{icon:IconName;title:string;tone?:IconTileTone;className?:string;/** Conserva la retícula propia del formulario (no envuelve los campos). */bare?:boolean;children:ReactNode}){
  return <section className={["form-section",className].filter(Boolean).join(" ")}>
    <h3 className="form-section-title"><IconTile name={icon} tone={tone} size="sm"/>{title}</h3>
    {bare?children:<div className="form-section-grid">{children}</div>}
  </section>;
}

export type ChoiceOption={value:string;label:string;icon?:IconName};

/**
 * Selección única de pocas opciones (hasta 5) como botones: se ven todas a la
 * vez y se eligen con un toque. Es un grupo de radios nativo, compatible con
 * `register` de React Hook Form (`<ChoiceGroup {...register("campo")}/>`).
 */
export function ChoiceGroup({options,label,name,value,defaultValue,onChange,onBlur,ref,disabled,initialFocus=false}:{options:ChoiceOption[];label:string;name:string;value?:string;defaultValue?:string;onChange?:ChangeEventHandler<HTMLInputElement>;onBlur?:FocusEventHandler<HTMLInputElement>;ref?:Ref<HTMLInputElement>;disabled?:boolean;/** Primer foco del modal en la primera opción. */initialFocus?:boolean}){
  return <div className="choice-group" role="radiogroup" aria-label={label}>
    {options.map((option,index)=><label className="choice-option" key={option.value}>
      <input type="radio" name={name} value={option.value} ref={ref} onChange={onChange} onBlur={onBlur} disabled={disabled} data-dialog-initial-focus={initialFocus&&index===0||undefined} {...(value!==undefined?{checked:value===option.value}:{defaultChecked:defaultValue===option.value})}/>
      {option.icon&&<Icon name={option.icon} size={16}/>}
      <span>{option.label}</span>
    </label>)}
  </div>;
}
