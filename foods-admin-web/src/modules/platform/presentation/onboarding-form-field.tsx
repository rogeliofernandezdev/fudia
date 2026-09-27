"use client";

import type {ReactNode} from "react";

export function OnboardingFormField({label,description,error,optional,wide,children}:{label:string;description?:string;error?:string;optional?:boolean;wide?:boolean;children:ReactNode}){
  return <label className={"onb-field"+(wide?" wide":"")+(error?" has-error":"")}>
    <span className="onb-field-label">{label}{optional&&<em>Opcional</em>}</span>
    {children}
    {error?<small className="onb-field-error" role="alert">{error}</small>:description?<small className="onb-field-hint">{description}</small>:null}
  </label>;
}
