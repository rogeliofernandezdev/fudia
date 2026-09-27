"use client";

import type {ReactNode} from "react";

export function OnboardingFormField({label,description,error,children}:{label:string;description?:string;error?:string;children:ReactNode}){
  return <label className={`onboarding-field${error?" has-error":""}`}>
    <span>{label}</span>
    {children}
    {error?<small className="field-error">{error}</small>:description?<small className="field-description">{description}</small>:null}
  </label>;
}
