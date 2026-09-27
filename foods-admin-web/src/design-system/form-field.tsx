"use client";
import "./styles/form-field.css";
import {cloneElement,isValidElement,useId,type ReactElement,type ReactNode} from "react";
import {Input,Select,Textarea} from "./page-header";

type FormFieldProps={
  label:ReactNode;
  /** Texto de ayuda breve bajo el control. */
  help?:string;
  /** Mensaje de validación; reemplaza la ayuda mientras exista. */
  error?:string;
  optional?:boolean;
  className?:string;
  /** `div` cuando el contenido no es un único control etiquetable (mapas, grupos de radios). */
  as?:"label"|"div";
  children:ReactNode;
};

/**
 * Campo de formulario homologado: etiqueta, control, ayuda y error.
 * Si el hijo es un único elemento (Input, Select, Textarea…) recibe
 * `aria-describedby` hacia la ayuda o el error y `aria-invalid` cuando hay error.
 */
export function FormField({label,help,error,optional,className="",as="label",children}:FormFieldProps){
  const id=useId();
  const helpId=`${id}-help`,errorId=`${id}-error`;
  const describedBy=error?errorId:help?helpId:undefined;
  const describable=isValidElement(children)&&(typeof children.type==="string"||children.type===Input||children.type===Select||children.type===Textarea);
  const control=describable
    ?cloneElement(children as ReactElement<Record<string,unknown>>,{
      "aria-describedby":[describedBy,(children.props as Record<string,unknown>)["aria-describedby"]].filter(Boolean).join(" ")||undefined,
      "aria-invalid":error?true:(children.props as Record<string,unknown>)["aria-invalid"],
    })
    :children;
  const Tag=as;
  return <Tag className={["form-field",className,error?"has-error":""].filter(Boolean).join(" ")}>
    <span className="form-field-label">{label}{optional&&<em>Opcional</em>}</span>
    {control}
    {error?<small id={errorId} className="field-error" role="alert">{error}</small>:help?<small id={helpId} className="field-help">{help}</small>:null}
  </Tag>;
}
